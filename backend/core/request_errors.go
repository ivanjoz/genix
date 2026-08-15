package core

import (
	coretypes "app/core/types"
	"context"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// The failures of the request currently being served, collected so prepareResponse can hand them
// to server_utils in one frame.
//
// Only the code line and a preview travel. The full message and its stack are already going to
// CloudWatch under this request's ID, and duplicating them into ScyllaDB would buy nothing except
// a table that grows with traffic instead of with the codebase.

const (
	// Four distinct code lines is where a request stops telling you anything new: past that it is
	// one failure cascading, and the frame stays small enough to never fragment.
	maxRequestErrors = 4
	// Enough to recognise a failure, not enough to reproduce it. CloudWatch has the rest.
	maxErrorTextBytes = 200
)

// RequestError is one distinct failing code line within one request.
type RequestError struct {
	// ID is the hash of Line, which is what user_logs.error_ids stores.
	ID int32
	// Line is "file.go:539" — the identity of the error, not the message.
	Line string
	// Text is the truncated preview.
	Text string
}

var (
	requestErrorsMu sync.Mutex
	requestErrors   []RequestError
)

// CallerCodeLine names the source position of the code that should be blamed. Skip counts frames
// above the caller: 0 is the line calling this function, 1 is its caller, and so on.
func CallerCodeLine(skip int) string {
	_, filePath, lineNumber, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}
	fileName := filePath
	if separator := strings.LastIndexByte(filePath, '/'); separator >= 0 {
		fileName = filePath[separator+1:]
	}
	return fileName + ":" + strconv.Itoa(lineNumber)
}

// RegisterRequestErrorAt records one failure against a code line already resolved by the caller.
//
// Repeats of the same line collapse into the first one seen: a loop that fails ten thousand times
// is one error at one place, and keeping the first text means the preview describes the failure
// that started it rather than the last one before the cap.
func RegisterRequestErrorAt(codeLine, message string) {
	if codeLine == "" {
		return
	}

	requestErrorsMu.Lock()
	defer requestErrorsMu.Unlock()

	if len(requestErrors) >= maxRequestErrors {
		return
	}
	for _, existing := range requestErrors {
		if existing.Line == codeLine {
			return
		}
	}

	requestErrors = append(requestErrors, RequestError{
		ID:   coretypes.MakeRequestErrorID(codeLine),
		Line: codeLine,
		Text: truncateErrorText(message),
	})
}

// ResetRequestErrors clears the accumulator at the start of a request.
func ResetRequestErrors() {
	requestErrorsMu.Lock()
	requestErrors = nil
	requestErrorsMu.Unlock()
}

// TakeRequestErrors drains what this request collected. Draining rather than reading means a
// second call cannot send the same errors twice, which matters because prepareResponse runs once
// per response but a panic can produce a second one.
func TakeRequestErrors() []RequestError {
	requestErrorsMu.Lock()
	drained := requestErrors
	requestErrors = nil
	requestErrorsMu.Unlock()
	return drained
}

// EmitRequestLog hands one finished request to server_utils, and is the last thing that happens to
// it. Everything it needs is already resolved: the identity from HandlerArgs, the failures from
// the accumulator, the elapsed time from the response.
//
// A request that failed nothing leaves no row unless LOG_ALL_REQUESTS says otherwise. user_logs
// answers "what broke, on what route, for which company", and a row per successful request buys
// nothing for that question while costing one write per request forever. Credit-limit rejections
// are not a special case here: MakeCreditRateLimitResponse builds its 429 through MakeErrCode,
// which captures, so a refused request arrives with an entry and passes the gate on its own.
//
// Nothing here can fail in a way the caller should care about. The daemon does not answer, an
// unreachable one is logged once and forgotten, and a request that has already produced its
// response must never be affected by what happened to its log row.
func EmitRequestLog(req *HandlerArgs, elapsedMs int64) {
	captured := TakeRequestErrors()
	entries := make([]RequestLogEntry, 0, len(captured))
	for _, requestError := range captured {
		entries = append(entries, RequestLogEntry{
			ID:   requestError.ID,
			Line: requestError.Line,
			Text: requestError.Text,
		})
	}

	// After the drain and never before it: TakeRequestErrors is what clears the accumulator, so
	// returning any earlier would leave this request's errors to be reported against whichever
	// request drains next.
	if len(entries) == 0 && (Env == nil || !Env.LOG_ALL_REQUESTS) {
		return
	}

	companyID, userID := int32(0), int32(0)
	if req.User != nil {
		companyID, userID = req.User.CompanyID, req.User.ID
	}

	now := time.Now().UTC()
	// Elapsed saturates rather than wrapping: a request slower than 65 seconds is already the
	// outlier being looked for, and a negative number in the column would just look like a bug.
	if elapsedMs > math.MaxInt16 {
		elapsedMs = math.MaxInt16
	} else if elapsedMs < 0 {
		elapsedMs = 0
	}

	record := RequestLogRecord{
		Date:      int16(now.Unix() / 86_400),
		RequestID: req.RequestID,
		RouteID:   req.RouteID,
		Frame:     coretypes.FrameOfDay(now.Unix()),
		CompanyID: companyID,
		UserID:    userID,
		ElapsedMs: int16(elapsedMs),
		Errors:    entries,
	}

	if err := SendRequestLog(context.Background(), record); err != nil {
		// logNoCapture, not Log: this message says "error" and would otherwise register itself as
		// a failure of the *next* request, which is the one whose accumulator is now open.
		logNoCapture("request log not sent::", err)
	}
}

// messageLooksLikeFailure is the heuristic half of error capture: most failures in this backend
// are reported by a plain Log call that says so in words, and waiting for every one of those to be
// migrated to LogError would mean shipping a table with holes in it.
func messageLooksLikeFailure(lowercaseMessage string) bool {
	return strings.Contains(lowercaseMessage, "error") || strings.Contains(lowercaseMessage, "warn")
}

// truncateErrorText cuts to maxErrorTextBytes without splitting a rune, and flattens the newlines
// and tabs that would otherwise break a log line into fragments that no longer parse as one entry.
func truncateErrorText(message string) string {
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\t", " ")
	if len(message) <= maxErrorTextBytes {
		return message
	}
	cut := maxErrorTextBytes
	for cut > 0 && !utf8.RuneStart(message[cut]) {
		cut--
	}
	return message[:cut]
}
