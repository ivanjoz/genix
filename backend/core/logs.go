package core

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mileusna/useragent"
)

var LogsSaved = []string{}
var LogCounter = new(int32)
var logsMu sync.Mutex

// The identity the log prefix is built from, and the counter behind a VPS request ID.
//
// These three are package globals for the same reason Env.REQUEST_ID is: the free Log function has
// no request to read from. They are only ever printed under IS_SERVERLESS, where the runtime
// serializes invocations — the concurrent server prints bare lines and never reads them.
var (
	LogCompanyID   int32
	LogUserID      int32
	LogRouteID     int16
	requestCounter uint32
)

// Decimal packing on a SUnix base, the same shape as SUnixTimeUUID: a time in the high digits and
// a tail below it. Bit shifts would work too, but they produce a number nobody can read — the
// timestamp inside one of these is meant to be recoverable by dividing, at a glance.
const (
	// Seven digits of counter under SUnixTime, whose tick is two seconds: ten million requests
	// inside one tick before it would borrow from the next one's range, which no single process
	// will ever reach. SUnixTime is around 3.9e8, so an ID sits near 3.9e15 and stays far inside
	// int64 — the room is there, and a counter that can never realistically wrap is worth it.
	vpsRequestsPerTick = 10_000_000
	// Six digits under SUnixTimeMilli: three of per-process salt, three of counter. Milliseconds
	// rather than SUnixTime's two seconds because concurrency is the whole problem here, and a
	// two-second bucket would put far more environments in the same one.
	serverlessTailSpan    = 1_000_000
	serverlessSaltSpan    = 1_000
	serverlessCounterSpan = 1_000
)

// processSalt separates two execution environments that mint in the same millisecond. Drawn once
// at start: within a process the counter already guarantees uniqueness, so the only thing left to
// distinguish is one process from another.
var processSalt = int64(rand.IntN(serverlessSaltSpan))

// lastVPSRequestID is the last ID handed out, which is also the (tick, counter) pair: they are the
// same number. Keeping one value instead of two removes the race at a tick boundary entirely —
// there is nothing to update in step.
var lastVPSRequestID atomic.Int64

// MakeRequestID mints the identity of one request. Time occupies the high digits in both modes, so
// the value sorts chronologically and the clustering key of user_logs ends up in write order for
// free.
//
// On the VPS a single long-lived process serves every request, so the in-memory counter is the
// whole answer: SUnixTime in the high digits, an autoincrement within that tick below it, and two
// requests cannot collide however close together they arrive.
//
// Under Lambda there is no shared counter — concurrent execution environments each start their own
// and would hand out the same numbers. So the tail is split: a per-process salt drawn at cold
// start, and the counter beneath it. Within one environment the counter still makes collision
// impossible; across environments, two requests collide only if they land in the same millisecond
// AND drew the same 1-in-1000 salt AND are at the same counter. That residual chance is accepted
// deliberately — this is a best-effort log row, and the alternative is coordinating IDs across
// Lambdas to protect a table whose whole purpose is to be cheap.
func MakeRequestID() int64 {
	if Env != nil && Env.IS_SERVERLESS {
		counter := int64(atomic.AddUint32(&requestCounter, 1))
		return SUnixTimeMilli()*serverlessTailSpan +
			processSalt*serverlessCounterSpan +
			counter%serverlessCounterSpan
	}

	// The counter restarts at each SUnixTime tick, so an ID reads as "this instant, request N of
	// it". A tick that somehow exceeds its 10 000 slots keeps counting into the next tick's range
	// rather than colliding: the ID stays unique and monotonic, and only the time decoded back out
	// of it runs slightly ahead.
	base := int64(SUnixTime()) * vpsRequestsPerTick
	for {
		previous := lastVPSRequestID.Load()
		next := previous + 1
		if next < base {
			next = base
		}
		if lastVPSRequestID.CompareAndSwap(previous, next) {
			return next
		}
	}
}

// SetLogRequest opens a request's prefix with what is known before the token is checked, and
// clears the identity of whatever was served before it. Splitting this from SetLogUser is what
// keeps a request rejected at the token check from being logged under the previous caller's
// company: it has a route and an ID, and no user, which is exactly what happened.
//
// Both setters do nothing outside Lambda. These globals exist only to feed a prefix that only the
// serverless branch prints, and writing them from the concurrent server would be a data race for
// the benefit of a value nothing reads. Every request still carries its identity on HandlerArgs,
// which is what reaches user_logs.
func SetLogRequest(requestID int64, routeID int16) {
	if Env == nil || !Env.IS_SERVERLESS {
		return
	}
	Env.REQUEST_ID = requestID
	LogRouteID = routeID
	LogCompanyID = 0
	LogUserID = 0
}

// SetLogUser fills in who the request turned out to belong to, once the session token resolves.
func SetLogUser(companyID, userID int32) {
	if Env == nil || !Env.IS_SERVERLESS {
		return
	}
	LogCompanyID = companyID
	LogUserID = userID
}

func LogDebug(args ...any) {
	if Env.LOGS_DEBUG {
		logLine(2, false, args...)
	}
}

func Log(args ...any) {
	logLine(2, false, args...)
}

// LogError records a failure against the caller's code line whatever its wording, for the ones
// that log and carry on instead of returning a response. MakeErr and its siblings cover the rest.
func LogError(args ...any) {
	logLine(2, true, args...)
}

// logNoCapture prints without registering anything. It exists for core's own error plumbing:
// MakeErrCode already recorded the handler's line, and letting the heuristic fire on the message
// it then logs would record a second error pointing at responses.go — burning one of the four
// slots on this file instead of on the code that failed.
func logNoCapture(args ...any) {
	logLine(0, false, args...)
}

// logLine is the single implementation behind every log helper.
//
// captureSkip is how many frames sit between this function and the code a failure should be
// blamed on; zero disables capture. force records regardless of wording, which is what separates
// LogError from a plain Log that happens to mention an error.
func logLine(captureSkip int, force bool, args ...any) {
	if len(args) == 0 {
		return
	}

	logMsg := strings.ToLower(Concats(args...))
	envIsReady := Env != nil
	doLog := true

	// Capture before the doLog gate below: whether a failure is worth printing and whether it is
	// worth recording are different questions, and a suppressed log line is still a real error.
	// Gated on Env so package-init logging, which belongs to no request, records nothing.
	if envIsReady && captureSkip > 0 && (force || messageLooksLikeFailure(logMsg)) {
		RegisterRequestErrorAt(CallerCodeLine(captureSkip), logMsg)
	}

	// Allow startup/package-init logging before Env is configured to avoid nil dereferences.
	if envIsReady {
		doLog = Env.LOGS_FULL || !Env.IS_SERVERLESS
		if Env.IS_SERVERLESS {
			if len(logMsg) > 1 && logMsg[0:1] == "*" {
				doLog = true
				args[0] = fmt.Sprintf("%v", args[0])[1:]
			} else if messageLooksLikeFailure(logMsg) {
				doLog = true
			}
		}
	}

	// LogsSaved is primarily used for Lambda request log persistence.
	// Protect the slice for cases where handlers run goroutines (errgroup).
	if envIsReady && (Env.LOGS_ONLY_SAVE || (Env.IS_SERVERLESS && doLog)) {
		logsMu.Lock()
		LogsSaved = append(LogsSaved, logMsg)
		logsMu.Unlock()
	}
	if !doLog {
		return
	}

	if envIsReady && Env.IS_SERVERLESS {
		args = append([]any{makeLogPrefix()}, args...)
	}
	fmt.Println(args...)
}

// makeLogPrefix builds the tokens CloudWatch indexes this line by.
//
//	#<requestID>#<n>#c7#u7_42#r118#|
//
// The request ID is decimal so a value read out of user_logs pastes straight into a filter. The
// three index tokens are what the log group is actually searched by: c7 is every request of
// company 7, u7_42 narrows it to one user as a single token rather than a pair the query has to
// AND together, and r118 is the route as its generated number.
//
// It replaces a per-path FnvHashString64 of the route, which cost a hash per line and produced a
// token nobody could read back without recomputing it.
func makeLogPrefix() string {
	counter := atomic.AddInt32(LogCounter, 1)
	tokens := []string{
		strconv.FormatInt(Env.REQUEST_ID, 10),
		strconv.Itoa(int(counter)),
		"c" + strconv.Itoa(int(LogCompanyID)),
		"u" + strconv.Itoa(int(LogCompanyID)) + "_" + strconv.Itoa(int(LogUserID)),
		"r" + strconv.Itoa(int(LogRouteID)),
	}
	return "#" + strings.Join(tokens, "#") + "#|"
}

func MakeReqLogParams() ReqLog {
	reqLog := ReqLog{}
	if len(Env.REQ_PARAMS) > 5 {
		params := strings.Split(Env.REQ_PARAMS, "|")
		if strings.Contains(params[0], "/") && len(params) > 4 {
			reqLog.PathName = params[0]
			reqLog.TimeZone = SrtToInt32(params[3]) * -1
			reqLog.Languages = params[4]
			reqLog.HardwareInfo = params[5]
		}
	}
	return reqLog
}

type ReqLog struct {
	SK           string   `json:"sk"`
	UserID       int32    `json:"u"`
	Device       string   `json:"d"`
	IP           string   `json:"i"`
	Accion       int32    `json:"a"`
	TimeElapsed  int      `json:"t"`
	PathName     string   `json:"p"`
	TimeZone     int32    `json:"z"`
	Languages    string   `json:"l"`
	HardwareInfo string   `json:"h"`
	ApiUrl       string   `json:"api"`
	Type         uint8    `json:"x"`
	Created      string   `json:"crd"`
	Logs         []string `json:"logs"`
}

var StartTime int64 = 0
var SessionLogs []string = []string{}

func AddToLogs(msg string) {
	SessionLogs = append(SessionLogs, msg)
}

func MakeReqLog() ReqLog {
	userAgent := Env.REQ_USER_AGENT
	nowTime := (time.Now()).UnixMilli()
	ua := useragent.Parse(userAgent)

	reqLog := ReqLog{
		UserID:      User.ID,
		IP:          Env.REQ_IP,
		Accion:      0,
		Logs:        SessionLogs,
		ApiUrl:      Env.REQ_PATH,
		TimeElapsed: int(nowTime - StartTime),
	}

	if reqLog.Device == "" {
		reqLog.Device = ua.Name + "|" + ua.Version + "|" + ua.OS
	}

	if len(reqLog.IP) > 19 {
		reqLog.IP = reqLog.IP[(len(reqLog.IP) - 19):]
	}

	reqLog.Created = ToBase36(nowTime)
	if reqLog.ApiUrl != "" {
		if strings.Contains(reqLog.ApiUrl, "GET.") {
			reqLog.Type = 1
		} else {
			reqLog.Type = 2
		}
	}

	req_params := Env.REQ_PARAMS
	if len(req_params) > 5 {
		params := strings.Split(req_params, "|")
		if strings.Contains(params[0], "/") && len(params) > 4 {
			reqLog.PathName = params[0]
			reqLog.TimeZone = SrtToInt(params[3]) * -1
			reqLog.Languages = params[4]
			reqLog.HardwareInfo = params[5]
		}
	}

	return reqLog
}
