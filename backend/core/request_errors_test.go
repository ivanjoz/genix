package core

import (
	"strings"
	"testing"
)

// Every test here needs Env to exist, because capture is gated on it so that package-init logging
// records nothing.
func withCaptureEnabled(t *testing.T) {
	t.Helper()
	previousEnv := Env
	if Env == nil {
		Env = &EnvStruct{}
	}
	ResetRequestErrors()
	t.Cleanup(func() {
		ResetRequestErrors()
		Env = previousEnv
	})
}

// The whole point of the frame-counting in MakeErr and logLine is that the recorded line is the
// caller's, not the line inside core that did the recording. Nothing else in the pipeline notices
// when that slips — the row still writes, it just points at responses.go forever.
func TestCaptureBlamesTheCallerNotCore(t *testing.T) {
	withCaptureEnabled(t)

	args := &HandlerArgs{}
	args.MakeErr("no se pudo obtener el registro")
	LogError("fallo al sincronizar")
	Log("Error:: la consulta no devolvió filas")

	captured := TakeRequestErrors()
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
	withCaptureEnabled(t)

	Log("Ejecutando Handler:: GET.almacenes")
	Log("Finalizado Handler:: GET.almacenes | Len: 412")

	if captured := TakeRequestErrors(); len(captured) != 0 {
		t.Fatalf("ordinary log lines were recorded as errors: %+v", captured)
	}
}

// MakeErr logs a message containing "Error::" through core's own plumbing. If that log call went
// through the heuristic, every handler failure would produce two entries: one at the handler and
// one at responses.go.
func TestMakeErrRecordsOneEntryNotTwo(t *testing.T) {
	withCaptureEnabled(t)

	args := &HandlerArgs{}
	args.MakeErr("algo falló")

	captured := TakeRequestErrors()
	if len(captured) != 1 {
		t.Fatalf("expected exactly one entry, got %d: %+v", len(captured), captured)
	}
	if strings.HasPrefix(captured[0].Line, "responses.go:") {
		t.Fatalf("the entry was blamed on core's own plumbing: %q", captured[0].Line)
	}
}

// A loop that fails ten thousand times is one error at one place. Without the collapse it would
// fill all four slots with the same line and hide the three other things that went wrong.
func TestRepeatedLineCollapsesAndKeepsTheFirstText(t *testing.T) {
	withCaptureEnabled(t)

	for attempt := 0; attempt < 50; attempt++ {
		RegisterRequestErrorAt("stock.go:120", "error en el intento "+string(rune('a'+attempt%26)))
	}

	captured := TakeRequestErrors()
	if len(captured) != 1 {
		t.Fatalf("expected one collapsed entry, got %d", len(captured))
	}
	if captured[0].Text != "error en el intento a" {
		t.Fatalf("kept %q; the first text is the one that describes what started the failure", captured[0].Text)
	}
}

func TestCapIsFourDistinctLines(t *testing.T) {
	withCaptureEnabled(t)

	for line := 100; line < 120; line++ {
		RegisterRequestErrorAt("stock.go:"+string(rune('0'+line%10))+"x", "error")
	}

	if captured := TakeRequestErrors(); len(captured) > maxRequestErrors {
		t.Fatalf("recorded %d entries, more than the cap of %d", len(captured), maxRequestErrors)
	}
}

// Draining matters because prepareResponse runs once per response, and a panic produces a second
// one. Without it the panic's row would carry the errors of the response that already reported them.
func TestTakeDrains(t *testing.T) {
	withCaptureEnabled(t)

	RegisterRequestErrorAt("stock.go:120", "error")
	if len(TakeRequestErrors()) != 1 {
		t.Fatal("the first take returned nothing")
	}
	if leftover := TakeRequestErrors(); len(leftover) != 0 {
		t.Fatalf("the second take returned %d entries; the accumulator did not drain", len(leftover))
	}
}

// The preview is a fixed-width field on the wire, so a multi-byte rune landing on the boundary
// must not be cut in half — an invalid UTF-8 tail would travel all the way into ScyllaDB.
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
