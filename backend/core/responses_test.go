package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendLocalResponseWithoutCompression(t *testing.T) {
	body := []byte("plain response")
	recorder := httptest.NewRecorder()
	var responseWriter http.ResponseWriter = recorder

	// Simulate compression support while the endpoint explicitly disables it.
	SendLocalResponse(HandlerArgs{
		ResponseWriter: &responseWriter,
		Encoding:       "zstd, gzip",
	}, HandlerResponse{
		Body:               &body,
		Headers:            map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		DisableCompression: true,
	})

	if contentEncoding := recorder.Header().Get("Content-Encoding"); contentEncoding != "" {
		t.Fatalf("expected no Content-Encoding header, got %q", contentEncoding)
	}
	if recorder.Body.String() != string(body) {
		t.Fatalf("expected raw body %q, got %q", body, recorder.Body.String())
	}
}

func TestMakeResponseFinalWithoutCompression(t *testing.T) {
	body := []byte("plain response")

	// Verify the Lambda response also preserves the unencoded text.
	response := MakeResponseFinal(&HandlerResponse{
		Body:               &body,
		Encoding:           "br, gzip",
		Headers:            map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		DisableCompression: true,
	})

	if contentEncoding := response.Headers["Content-Encoding"]; contentEncoding != "" {
		t.Fatalf("expected no Content-Encoding header, got %q", contentEncoding)
	}
	if response.IsBase64Encoded {
		t.Fatal("expected raw text response, got base64 encoding")
	}
	if response.Body != string(body) {
		t.Fatalf("expected raw body %q, got %q", body, response.Body)
	}
}
