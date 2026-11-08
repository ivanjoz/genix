package core

import (
	"net/http"
	"sync/atomic"
)

// CountingResponseWriter counts the body bytes written through it. A stream handler writes to the
// ResponseWriter itself and never hands a body back to the router, so this count is the only size
// a streamed GET has; the router settles its top-up from it once the stream ends (fareward plan D2).
type CountingResponseWriter struct {
	http.ResponseWriter
	written atomic.Int64
}

func NewCountingResponseWriter(writer http.ResponseWriter) *CountingResponseWriter {
	return &CountingResponseWriter{ResponseWriter: writer}
}

func (writer *CountingResponseWriter) Write(body []byte) (int, error) {
	written, err := writer.ResponseWriter.Write(body)
	writer.written.Add(int64(written))
	return written, err
}

// Flush keeps the wrapper an http.Flusher: SSE refuses a writer that is not one.
func (writer *CountingResponseWriter) Flush() {
	if flusher, ok := writer.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap lets http.ResponseController reach the server's own writer.
func (writer *CountingResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

// BytesWritten is how many body bytes went out so far.
func (writer *CountingResponseWriter) BytesWritten() int64 {
	return writer.written.Load()
}

// StreamedBytes is what a stream handler wrote through the request's writer, or 0 when the
// request has no counting writer (Lambda, tests).
func (req *HandlerArgs) StreamedBytes() int {
	if req == nil || req.ResponseWriter == nil {
		return 0
	}
	if counter, ok := (*req.ResponseWriter).(*CountingResponseWriter); ok {
		return int(counter.BytesWritten())
	}
	return 0
}
