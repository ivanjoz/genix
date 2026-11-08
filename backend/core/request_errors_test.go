package core

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// withCaptureEnabled gives the test an Env, which capture is gated on so that package-init logging
// records nothing. isServerless decides whether Log/LogError capture at all: only Lambda knows
// which request a bare Log call belongs to.
func withCaptureEnabled(t *testing.T, isServerless bool) {
	t.Helper()
	previousEnv := Env
	Env = &EnvStruct{IS_SERVERLESS: isServerless}
	ResetServerlessLogErrors()
	t.Cleanup(func() {
		ResetServerlessLogErrors()
		Env = previousEnv
	})
}

// The whole point of the frame-counting in MakeErr and logLine is that the recorded line is the
// caller's, not the line inside core that did the recording. Nothing else in the pipeline notices
// when that slips — the row still writes, it just points at responses.go forever.
func TestCaptureBlamesTheCallerNotCore(t *testing.T) {
	withCaptureEnabled(t, true)

	args := &HandlerArgs{Errors: &RequestErrors{}}
	args.MakeErr("no se pudo obtener el registro")
	LogError("fallo al sincronizar")
	Log("Error:: la consulta no devolvió filas")

	captured := append(args.Errors.Take(), serverlessLogErrors.Take()...)
	if len(captured) != 3 {
		t.Fatalf("expected three distinct code lines, got %d: %+v", len(captured), captured)
	}
	for _, requestError := range captured {
		if !strings.HasPrefix(requestError.Line, "request_errors_test.go:") {
			t.Errorf("blamed %q; every one of these should point at this test file", requestError.Line)
		}
		if requestError.ID <= 0 {
			t.Errorf("code line %q got a non-positive ID: %d", requestError.Line, requestError.ID)
		}
	}
}

// A message that says nothing about failing is not a failure, or every request would arrive
// carrying four errors and the error count would stop meaning anything.
func TestPlainLogRecordsNothing(t *testing.T) {
	withCaptureEnabled(t, true)

	Log("Ejecutando Handler:: GET.almacenes")
	Log("Finalizado Handler:: GET.almacenes | Len: 412")

	if captured := serverlessLogErrors.Take(); len(captured) != 0 {
		t.Fatalf("ordinary log lines were recorded as errors: %+v", captured)
	}
}

// On the VPS a bare Log cannot know its request, so it must not write anywhere another request
// could read it back: that shared accumulator is the race this replaced.
func TestVPSLogErrorIsNotCapturedGlobally(t *testing.T) {
	withCaptureEnabled(t, false)

	LogError("fallo al sincronizar")
	Log("Error:: la consulta no devolvió filas")

	if captured := serverlessLogErrors.Take(); len(captured) != 0 {
		t.Fatalf("a VPS log line reached the shared accumulator: %+v", captured)
	}
}

// Two requests failing at the same time each report their own error and only their own.
func TestConcurrentRequestsKeepTheirOwnErrors(t *testing.T) {
	withCaptureEnabled(t, false)

	firstRequest := &HandlerArgs{Route: "a", Errors: &RequestErrors{}}
	secondRequest := &HandlerArgs{Route: "b", Errors: &RequestErrors{}}
	var failing sync.WaitGroup
	for _, request := range []*HandlerArgs{firstRequest, secondRequest} {
		failing.Add(1)
		go func() {
			defer failing.Done()
			for range 100 {
				request.Errors.RegisterAt("route_"+request.Route+".go:1", "error")
			}
		}()
	}
	failing.Wait()

	for _, request := range []*HandlerArgs{firstRequest, secondRequest} {
		captured := request.Errors.Take()
		if len(captured) != 1 || captured[0].Line != "route_"+request.Route+".go:1" {
			t.Fatalf("request %s reported %+v", request.Route, captured)
		}
	}
}

// MakeErr logs a message containing "Error::" through core's own plumbing. If that log call went
// through the heuristic, every handler failure would produce two entries: one at the handler and
// one at responses.go.
func TestMakeErrRecordsOneEntryNotTwo(t *testing.T) {
	withCaptureEnabled(t, true)

	args := &HandlerArgs{Errors: &RequestErrors{}}
	args.MakeErr("algo falló")

	captured := append(args.Errors.Take(), serverlessLogErrors.Take()...)
	if len(captured) != 1 {
		t.Fatalf("expected exactly one entry, got %d: %+v", len(captured), captured)
	}
	if strings.HasPrefix(captured[0].Line, "responses.go:") {
		t.Fatalf("the entry was blamed on core's own plumbing: %q", captured[0].Line)
	}
}

// A recovered panic is reported against the line that panicked, not against the recover.
func TestPanicCodeLineNamesThePanickingLine(t *testing.T) {
	var panicLine string
	func() {
		defer func() {
			recover()
			panicLine = PanicCodeLine()
		}()
		panic("boom") // the line PanicCodeLine must name
	}()
	if !strings.HasPrefix(panicLine, "request_errors_test.go:") {
		t.Fatalf("blamed %q instead of this test file", panicLine)
	}
}

// A loop that fails ten thousand times is one error at one place. Without the collapse it would
// fill all four slots with the same line and hide the three other things that went wrong.
func TestRepeatedLineCollapsesAndKeepsTheFirstText(t *testing.T) {
	accumulator := &RequestErrors{}
	for attempt := 0; attempt < 50; attempt++ {
		accumulator.RegisterAt("stock.go:120", "error en el intento "+string(rune('a'+attempt%26)))
	}

	captured := accumulator.Take()
	if len(captured) != 1 {
		t.Fatalf("expected one collapsed entry, got %d", len(captured))
	}
	if captured[0].Text != "error en el intento a" {
		t.Fatalf("kept %q; the first text is the one that describes what started the failure", captured[0].Text)
	}
}

func TestCapIsFourDistinctLines(t *testing.T) {
	accumulator := &RequestErrors{}
	for line := 100; line < 120; line++ {
		accumulator.RegisterAt("stock.go:"+string(rune('0'+line%10))+"x", "error")
	}

	if captured := accumulator.Take(); len(captured) > maxRequestErrors {
		t.Fatalf("recorded %d entries, more than the cap of %d", len(captured), maxRequestErrors)
	}
}

// Draining matters because prepareResponse runs once per response, and a panic produces a second
// one. Without it the panic's row would carry the errors of the response that already reported them.
func TestTakeDrains(t *testing.T) {
	accumulator := &RequestErrors{}
	accumulator.RegisterAt("stock.go:120", "error")
	if len(accumulator.Take()) != 1 {
		t.Fatal("the first take returned nothing")
	}
	if leftover := accumulator.Take(); len(leftover) != 0 {
		t.Fatalf("the second take returned %d entries; the accumulator did not drain", len(leftover))
	}
}

// captureEmittedLogs swaps the send seam so a test can see what EmitRequestLog decided, and pins
// LOG_ALL_REQUESTS to a known value — the whole point of these tests is which requests reach the
// daemon, and a nil Env or a leftover flag would make that answer meaningless.
func captureEmittedLogs(t *testing.T, logAllRequests bool) *[]RequestLogRecord {
	t.Helper()
	previousEnv, previousSend := Env, SendRequestLog
	Env = &EnvStruct{LOG_ALL_REQUESTS: logAllRequests}
	emitted := []RequestLogRecord{}
	SendRequestLog = func(ctx context.Context, record RequestLogRecord) error {
		emitted = append(emitted, record)
		return nil
	}
	ResetServerlessLogErrors()
	t.Cleanup(func() {
		ResetServerlessLogErrors()
		Env, SendRequestLog = previousEnv, previousSend
	})
	return &emitted
}

// The default rule: user_logs keeps failures. A row per successful request is a write per request
// forever, and it answers nothing the table exists to answer.
func TestErrorFreeRequestLeavesNoRow(t *testing.T) {
	emitted := captureEmittedLogs(t, false)

	EmitRequestLog(&HandlerArgs{RouteID: 34}, 12)

	if len(*emitted) != 0 {
		t.Fatalf("a request that failed nothing still wrote a row: %+v", *emitted)
	}
}

// The gate must not swallow the rows the table is for. This is the case that would break silently:
// nothing else in the pipeline notices when failures stop being recorded.
func TestFailedRequestStillWritesItsRow(t *testing.T) {
	emitted := captureEmittedLogs(t, false)

	args := &HandlerArgs{RouteID: 34, Errors: &RequestErrors{}}
	args.Errors.RegisterAt("stock.go:120", "error al leer el lote")
	EmitRequestLog(args, 12)

	if len(*emitted) != 1 {
		t.Fatalf("expected one row, got %d", len(*emitted))
	}
	if len((*emitted)[0].Errors) != 1 {
		t.Fatalf("the row carried %d errors, not the one captured", len((*emitted)[0].Errors))
	}
	if leftover := args.Errors.Take(); len(leftover) != 0 {
		t.Fatalf("%d errors survived the emit and would be blamed on the next request", len(leftover))
	}
}

// The switch is what makes the table usable for measuring traffic rather than only failures.
func TestLogAllRequestsWidensToEveryRequest(t *testing.T) {
	emitted := captureEmittedLogs(t, true)

	EmitRequestLog(&HandlerArgs{RouteID: 34}, 12)

	if len(*emitted) != 1 {
		t.Fatalf("expected one row with the switch on, got %d", len(*emitted))
	}
	if len((*emitted)[0].Errors) != 0 {
		t.Fatalf("a successful request carried %d errors", len((*emitted)[0].Errors))
	}
}

// A refused request has to reach user_logs, and it does so without a special case in EmitRequestLog:
// MakeCreditRateLimitResponse answers through MakeErrCode, which captures. That is load-bearing now
// — if it ever stopped capturing, exhausted-quota events would vanish from the table and the only
// symptom would be their absence.
func TestCreditRateLimitRejectionWritesItsRow(t *testing.T) {
	emitted := captureEmittedLogs(t, false)

	args := &HandlerArgs{RouteID: 34, Errors: &RequestErrors{}}
	response := args.MakeCreditRateLimitResponse(&CreditLimitExceeded{Code: 3, Window: "1h", CPU: true})
	if response.StatusCode != 429 {
		t.Fatalf("a credit rejection answered %d, not 429", response.StatusCode)
	}
	EmitRequestLog(args, 12)

	if len(*emitted) != 1 {
		t.Fatalf("the rejection left no row in user_logs")
	}
	if len((*emitted)[0].Errors) == 0 {
		t.Fatal("the rejection's row carried no error entry")
	}
}

// The preview is a fixed-width field on the wire, so a multi-byte rune landing on the boundary
// must not be cut in half — an invalid UTF-8 tail would travel all the way into the database.
func TestTruncateErrorTextKeepsRunesWhole(t *testing.T) {
	longAccented := strings.Repeat("á", 300)
	truncated := truncateErrorText(longAccented)

	if len(truncated) > maxErrorTextBytes {
		t.Fatalf("truncated to %d bytes, over the %d ceiling", len(truncated), maxErrorTextBytes)
	}
	if !strings.HasPrefix(longAccented, truncated) {
		t.Fatal("truncation produced something that is not a prefix of the original")
	}
	for _, character := range truncated {
		if character == '�' {
			t.Fatal("truncation split a rune")
		}
	}
}

// Newlines would break one log entry into fragments that no longer parse as a single record.
func TestTruncateErrorTextFlattensWhitespace(t *testing.T) {
	flattened := truncateErrorText("primera línea\nsegunda\tlínea")
	if strings.ContainsAny(flattened, "\n\t") {
		t.Fatalf("newlines or tabs survived: %q", flattened)
	}
}
