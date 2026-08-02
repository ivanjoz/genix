package core

// Streaming (RESPONSE_STREAM) response coverage.
//
// The runtime reads the response as one stream: a JSON prelude of status and headers, eight
// NUL bytes, then the body verbatim. These tests read the response the same way the runtime
// does and decode what comes out, because the failure mode of getting this wrong is not a
// compile error — it is every request returning a malformed body in production.

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// readStreamingResponse consumes the response exactly as the Lambda runtime does and splits it
// into the decoded prelude and the raw body bytes.
func readStreamingResponse(t *testing.T, response io.Reader) (map[string]any, []byte) {
	t.Helper()

	raw, err := io.ReadAll(response)
	if err != nil {
		t.Fatalf("reading streaming response: %v", err)
	}

	separator := bytes.Repeat([]byte{0}, 8)
	splitAt := bytes.Index(raw, separator)
	if splitAt < 0 {
		t.Fatalf("streaming response has no 8-NUL prelude separator: %q", raw)
	}

	prelude := map[string]any{}
	if err := json.Unmarshal(raw[:splitAt], &prelude); err != nil {
		t.Fatalf("decoding prelude %q: %v", raw[:splitAt], err)
	}

	return prelude, raw[splitAt+len(separator):]
}

func preludeHeaders(t *testing.T, prelude map[string]any) map[string]string {
	t.Helper()

	headers := map[string]string{}
	rawHeaders, present := prelude["headers"].(map[string]any)
	if !present {
		return headers
	}
	for name, value := range rawHeaders {
		text, isText := value.(string)
		if isText {
			headers[name] = text
		}
	}
	return headers
}

func TestMakeStreamingResponseFinalCompressesAndStreams(t *testing.T) {
	payload := []byte(strings.Repeat(`{"producto":"prueba con acentos ñ á"},`, 500))

	cases := []struct {
		acceptEncoding string
		wantEncoding   string
		decode         func(*testing.T, []byte) []byte
	}{
		{"zstd, gzip", "zstd", decodeZstd},
		{"gzip", "gzip", decodeGzip},
		{"", "", func(_ *testing.T, body []byte) []byte { return body }},
	}

	for _, testCase := range cases {
		body := append([]byte(nil), payload...)
		response := MakeStreamingResponseFinal(&HandlerResponse{
			Body:     &body,
			Encoding: testCase.acceptEncoding,
			Headers:  map[string]string{},
		})

		prelude, streamedBody := readStreamingResponse(t, response)

		if statusCode, _ := prelude["statusCode"].(float64); int(statusCode) != 200 {
			t.Errorf("Accept-Encoding %q: want status 200, got %v", testCase.acceptEncoding, prelude["statusCode"])
		}

		headers := preludeHeaders(t, prelude)
		if got := headers["Content-Encoding"]; got != testCase.wantEncoding {
			t.Errorf("Accept-Encoding %q: want Content-Encoding %q, got %q",
				testCase.acceptEncoding, testCase.wantEncoding, got)
		}

		if decoded := testCase.decode(t, streamedBody); !bytes.Equal(decoded, payload) {
			t.Errorf("Accept-Encoding %q: body did not round-trip (%d bytes streamed, %d decoded, %d expected)",
				testCase.acceptEncoding, len(streamedBody), len(decoded), len(payload))
		}

		// The runtime closes the response once streamed; that is what recycles the buffer.
		if err := response.Close(); err != nil {
			t.Errorf("Accept-Encoding %q: Close: %v", testCase.acceptEncoding, err)
		}
	}
}

// TestStreamingResponseBodyIsNotBase64 is the point of the whole exercise: under RESPONSE_STREAM
// the body must be the compressed bytes themselves, not a base64 string of them.
func TestStreamingResponseBodyIsNotBase64(t *testing.T) {
	body := []byte(strings.Repeat("compress me ", 200))
	response := MakeStreamingResponseFinal(&HandlerResponse{
		Body:     &body,
		Encoding: "zstd",
		Headers:  map[string]string{},
	})
	defer response.Close()

	_, streamedBody := readStreamingResponse(t, response)

	// zstd frames start with the magic number 0x28 0xB5 0x2F 0xFD. A base64 string never does.
	if len(streamedBody) < 4 || !bytes.Equal(streamedBody[:4], []byte{0x28, 0xB5, 0x2F, 0xFD}) {
		t.Fatalf("expected a raw zstd frame, got first bytes %v", streamedBody[:min(8, len(streamedBody))])
	}
}

// TestStreamingResponseCloseIsIdempotent guards the pooled buffer: a double Close must not hand
// the same buffer to the pool twice, which would let two later requests share one buffer.
func TestStreamingResponseCloseIsIdempotent(t *testing.T) {
	body := []byte("small payload")
	response := MakeStreamingResponseFinal(&HandlerResponse{
		Body:     &body,
		Encoding: "zstd",
		Headers:  map[string]string{},
	})

	if _, err := io.ReadAll(response); err != nil {
		t.Fatalf("reading response: %v", err)
	}
	if err := response.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := response.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestMakeErrStreamingFinal(t *testing.T) {
	cases := []struct {
		statusCode int32
		wantStatus int
	}{{400, 400}, {401, 401}, {500, 500}}

	for _, testCase := range cases {
		response := MakeErrStreamingFinal(testCase.statusCode, "algo salió mal")
		prelude, streamedBody := readStreamingResponse(t, response)

		if statusCode, _ := prelude["statusCode"].(float64); int(statusCode) != testCase.wantStatus {
			t.Errorf("status %d: want %d, got %v", testCase.statusCode, testCase.wantStatus, prelude["statusCode"])
		}

		decoded := map[string]string{}
		if err := json.Unmarshal(streamedBody, &decoded); err != nil {
			t.Fatalf("status %d: decoding error body %q: %v", testCase.statusCode, streamedBody, err)
		}
		if decoded["error"] != "algo salió mal" {
			t.Errorf("status %d: want error message preserved, got %q", testCase.statusCode, decoded["error"])
		}
	}
}

// TestStreamingAndBufferedAgreeOnBody pins the two invoke modes to the same bytes: whatever the
// buffered path base64-encodes must be exactly what the streaming path writes raw.
func TestStreamingAndBufferedAgreeOnBody(t *testing.T) {
	payload := []byte(strings.Repeat(`{"a":1,"b":"texto"},`, 300))

	bufferedBody := append([]byte(nil), payload...)
	buffered := MakeResponseFinal(&HandlerResponse{
		Body:     &bufferedBody,
		Encoding: "zstd",
		Headers:  map[string]string{},
	})

	streamingBody := append([]byte(nil), payload...)
	streaming := MakeStreamingResponseFinal(&HandlerResponse{
		Body:     &streamingBody,
		Encoding: "zstd",
		Headers:  map[string]string{},
	})
	defer streaming.Close()

	_, streamedBytes := readStreamingResponse(t, streaming)

	if !bytes.Equal(decodeZstd(t, streamedBytes), decodeZstd(t, decodeBase64(t, buffered.Body))) {
		t.Fatal("buffered and streaming invoke modes produced different payloads")
	}
	if !buffered.IsBase64Encoded {
		t.Error("buffered mode should still base64-encode; only streaming avoids it")
	}
}
