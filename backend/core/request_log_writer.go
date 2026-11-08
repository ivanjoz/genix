package core

import (
	"app/core/types"
	"app/db"
	"sync"
	"time"
)

// The request log is written here, into user_logs and request_errors. It used to travel to fareward
// in a fire-and-forget frame and be written by the daemon; now fareward holds no database and this
// process, which already has the pool, writes the rows itself (fareward plan D7).
//
// It fails open, like before: a row that cannot be written is dropped with a warning, never turned
// into a failed request. On the VPS the rows are queued and written in batches by one goroutine, so a
// request never waits on them. On Lambda the process may freeze once the response is out, so the row
// is written before the handler returns; only requests that failed write one (LOG_ALL_REQUESTS
// aside), so the cost is paid by the requests that already went wrong.
//
// user_logs is a MyRocks log table (types.UserLog): compressed, written without the write-ahead log,
// expired by the engine after 30 days.

const (
	// requestLogQueueCapacity bounds what a stalled database can make this process hold. Past it,
	// rows are dropped and counted: logging must never slow the requests down.
	requestLogQueueCapacity = 8_192
	// requestLogMaxBatch rows are written in one statement, or whatever arrived within one interval.
	requestLogMaxBatch      = 512
	requestLogFlushInterval = time.Second
	// requestErrorFreshness: a failing code line usually fails thousands of times an hour, and its
	// request_errors row is identical every time. It is rewritten at most once per this interval.
	requestErrorFreshness = 10 * time.Minute
	// requestErrorGateEntries bounds the gate: the distinct failing code lines of the codebase.
	requestErrorGateEntries = 20_000
)

// requestLogEntry is one finished request: its user_logs row and the request_errors rows it may
// refresh.
type requestLogEntry struct {
	row    types.UserLog
	errors []types.RequestError
}

type requestLogWriter struct {
	queue   chan requestLogEntry
	dropped sync.Mutex
	// droppedRows counts the rows a full queue refused since the last warning.
	droppedRows int
	gate        requestErrorGate
}

var (
	requestLogOnce     sync.Once
	sharedRequestLog   *requestLogWriter
	requestLogGateLock sync.Mutex
	// serverlessErrorGate is the Lambda path's gate: one per warm container.
	serverlessErrorGate = newRequestErrorGate(requestErrorFreshness, requestErrorGateEntries)
)

// writeRequestLog hands one finished request to the writer. A variable so the tests can watch what
// EmitRequestLog decides without a database.
var writeRequestLog = enqueueRequestLog

func enqueueRequestLog(entry requestLogEntry) {
	if Env != nil && Env.IS_SERVERLESS {
		requestLogGateLock.Lock()
		errorRows := serverlessErrorGate.admit(entry.errors, time.Now())
		requestLogGateLock.Unlock()
		writeRequestLogBatch([]types.UserLog{entry.row}, errorRows)
		return
	}
	requestLogOnce.Do(func() {
		sharedRequestLog = &requestLogWriter{
			queue: make(chan requestLogEntry, requestLogQueueCapacity),
			gate:  newRequestErrorGate(requestErrorFreshness, requestErrorGateEntries),
		}
		go sharedRequestLog.run()
	})
	select {
	case sharedRequestLog.queue <- entry:
	default:
		sharedRequestLog.dropped.Lock()
		sharedRequestLog.droppedRows++
		sharedRequestLog.dropped.Unlock()
	}
}

// run drains the queue: a batch is written when it is full or when the interval ticks.
func (writer *requestLogWriter) run() {
	ticker := time.NewTicker(requestLogFlushInterval)
	defer ticker.Stop()
	rows := make([]types.UserLog, 0, requestLogMaxBatch)
	errorRows := []types.RequestError{}
	flush := func() {
		writer.dropped.Lock()
		dropped := writer.droppedRows
		writer.droppedRows = 0
		writer.dropped.Unlock()
		if dropped > 0 {
			logNoCapture("request log: la cola está llena, filas descartadas::", dropped)
		}
		if len(rows) > 0 || len(errorRows) > 0 {
			writeRequestLogBatch(rows, errorRows)
		}
		rows, errorRows = rows[:0], errorRows[:0]
	}
	for {
		select {
		case entry := <-writer.queue:
			rows = append(rows, entry.row)
			errorRows = append(errorRows, writer.gate.admit(entry.errors, time.Now())...)
			if len(rows) >= requestLogMaxBatch {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// writeRequestLogBatch writes the rows, dropping what fails: there is no retry, a retried batch
// could only fail the same way or duplicate what half-landed.
func writeRequestLogBatch(rows []types.UserLog, errorRows []types.RequestError) {
	if len(rows) > 0 {
		batch := rows
		if err := db.Insert(&batch); err != nil {
			// logNoCapture: this line says "error" and must not register as a failure of a request.
			logNoCapture("request log: no se pudieron escribir las filas de user_logs::", len(rows), "::", err)
		}
	}
	if len(errorRows) > 0 {
		batch := errorRows
		if err := db.Upsert(&batch); err != nil {
			logNoCapture("request log: no se pudieron escribir las filas de request_errors::", len(errorRows), "::", err)
		}
	}
}

// requestErrorGate decides which request_errors rows are worth rewriting. Keyed on the pair, not on
// the ID alone: two code lines that collide on the same int32 are two rows, and keying on the hash
// would let one suppress the other for good.
type requestErrorGate struct {
	lastWritten map[requestErrorKey]time.Time
	freshness   time.Duration
	capacity    int
}

type requestErrorKey struct {
	id   int32
	line string
}

func newRequestErrorGate(freshness time.Duration, capacity int) requestErrorGate {
	return requestErrorGate{lastWritten: map[requestErrorKey]time.Time{}, freshness: freshness, capacity: max(capacity, 1)}
}

// admit returns the rows to write now, and records them as written.
func (gate *requestErrorGate) admit(errorRows []types.RequestError, now time.Time) []types.RequestError {
	admitted := []types.RequestError{}
	for _, errorRow := range errorRows {
		key := requestErrorKey{errorRow.ID, errorRow.CodeLine}
		if writtenAt, isKnown := gate.lastWritten[key]; isKnown && now.Sub(writtenAt) < gate.freshness {
			continue
		}
		if _, isKnown := gate.lastWritten[key]; !isKnown && len(gate.lastWritten) >= gate.capacity {
			gate.evictOldest()
		}
		gate.lastWritten[key] = now
		admitted = append(admitted, errorRow)
	}
	return admitted
}

// evictOldest is linear on purpose: the map holds the failing code lines of the codebase and only
// evicts once it reached the ceiling.
func (gate *requestErrorGate) evictOldest() {
	var oldestKey requestErrorKey
	var oldestTime time.Time
	isFirst := true
	for key, writtenAt := range gate.lastWritten {
		if isFirst || writtenAt.Before(oldestTime) {
			oldestKey, oldestTime, isFirst = key, writtenAt, false
		}
	}
	delete(gate.lastWritten, oldestKey)
}
