package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// A stream writes through the wrapper and stays a Flusher, and its size is what the router reads
// back to settle the top-up.
func TestCountingResponseWriterCountsAStream(t *testing.T) {
	recorder := httptest.NewRecorder()
	var writer http.ResponseWriter = NewCountingResponseWriter(recorder)
	args := &HandlerArgs{ResponseWriter: &writer}

	if _, ok := writer.(http.Flusher); !ok {
		t.Fatal("the wrapper is not a Flusher: SSE would refuse it")
	}
	writer.Write([]byte("event: metrics\n"))
	writer.(http.Flusher).Flush()
	writer.Write([]byte("data: {}\n\n"))

	if got := args.StreamedBytes(); got != 25 {
		t.Fatalf("StreamedBytes = %d, want 25", got)
	}
	if !recorder.Flushed || recorder.Body.Len() != 25 {
		t.Fatalf("flushed=%v body=%d", recorder.Flushed, recorder.Body.Len())
	}
	if got := (&HandlerArgs{}).StreamedBytes(); got != 0 {
		t.Fatalf("no writer: StreamedBytes = %d", got)
	}
}
