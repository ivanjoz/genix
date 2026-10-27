package core

import (
	"bytes"
	"encoding/base64"
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
		Encoding:           "zstd, gzip",
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

// Bytes that are not valid UTF-8 survive an uncompressed Lambda answer only base64'd.
func TestMakeResponseFinalBinaryWithoutCompression(t *testing.T) {
	body := []byte{0x04, 0xff, 0x80, 0x00, 0xc3}
	req := HandlerArgs{}
	noEncoding := req.MakeBinaryResponse(body)
	// The client accepts gzip but the response opts out of compression: still raw bytes.
	compressionDisabled := req.MakeBinaryResponse(body)
	compressionDisabled.Encoding, compressionDisabled.DisableCompression = "gzip", true
	for _, response := range []HandlerResponse{noEncoding, compressionDisabled} {
		final := MakeResponseFinal(&response)
		decoded, err := base64.StdEncoding.DecodeString(final.Body)
		if !final.IsBase64Encoded || err != nil || !bytes.Equal(decoded, body) {
			t.Fatalf("binary body not base64'd intact: %+v (%v)", final, err)
		}
		if final.Headers["Content-Type"] != "application/octet-stream" || final.Headers["Content-Encoding"] != "" {
			t.Fatalf("headers: %v", final.Headers)
		}
	}
}
