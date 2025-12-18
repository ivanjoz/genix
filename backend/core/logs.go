package core

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mileusna/useragent"
)

var LogsSaved = []string{}
var LogCounter = new(int32)
var logsMu sync.Mutex

func LogDebug(args ...any) {
	if Env.LOGS_DEBUG {
		Log(args)
	}
}

func Log(args ...any) {
	if len(args) == 0 {
		return
	}

	logMsg := strings.ToLower(Concats(args...))

	doLog := Env.LOGS_FULL || Env.IS_LOCAL
	if !Env.IS_LOCAL {
		if len(logMsg) > 1 && logMsg[0:1] == "*" {
			doLog = true
			args[0] = fmt.Sprintf("%v", args[0])[1:]
		} else {
			if strings.Contains(logMsg, "error") || strings.Contains(logMsg, "warn") {
				doLog = true
			}
		}
	}

	// LogsSaved is primarily used for Lambda request log persistence.
	// Protect the slice for cases where handlers run goroutines (errgroup).
	if Env.LOGS_ONLY_SAVE || (!Env.IS_LOCAL && doLog) {
		logsMu.Lock()
		LogsSaved = append(LogsSaved, logMsg)
		logsMu.Unlock()
	}
	if !doLog {
		return
	}

	if !Env.IS_LOCAL {
		newCounter := atomic.AddInt32(LogCounter, 1)
		hashs := []string{Env.REQ_ID, fmt.Sprintf("%v", newCounter)}

		if len(REQ_PATHS) > 0 {
			for _, reqPath := range REQ_PATHS {
				hash := FnvHashString64(reqPath, 64, 5)
				hashs = append(hashs, hash)
			}
		}
		hashString := "#" + strings.Join(hashs, "#") + "#|"
		args = append([]any{hashString}, args...)
	}
	fmt.Println(args...)
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
		UserID:      Usuario.ID,
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
