package core

// Pooled compressors hand callers a slice that points into a recycled buffer, so the failure
// mode is not "wrong output" but "output that was correct until the next request reused the
// buffer". These decompress what the pool produced, and do it concurrently, because a pooling
// bug only shows up when two callers are in flight at once.

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func decodeZstd(t *testing.T, compressed []byte) []byte {
	t.Helper()
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatalf("zstd.NewReader: %v", err)
	}
	defer decoder.Close()

	decoded, err := decoder.DecodeAll(compressed, nil)
	if err != nil {
		t.Fatalf("zstd DecodeAll: %v", err)
	}
	return decoded
}

func decodeGzip(t *testing.T, compressed []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer reader.Close()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("gzip ReadAll: %v", err)
	}
	return decoded
}

func TestPooledCompressorsRoundTrip(t *testing.T) {
	bodies := [][]byte{
		[]byte(""),
		[]byte("short"),
		[]byte(strings.Repeat("repetitive payload ", 5000)),
		[]byte(`[[1,0,"id",1,"nm"]],[[1,[1],1,"árbol <b>ñ</b>"]]`),
	}

	for i, body := range bodies {
		var zstdDecoded, gzipDecoded []byte

		CompressZstdPooled(body, func(compressed []byte) {
			zstdDecoded = decodeZstd(t, compressed)
		})
		if !bytes.Equal(zstdDecoded, body) {
			t.Errorf("body %d: zstd round-trip mismatch (%d bytes in, %d out)", i, len(body), len(zstdDecoded))
		}

		if err := CompressGzipPooled(body, func(compressed []byte) {
			gzipDecoded = decodeGzip(t, compressed)
		}); err != nil {
			t.Fatalf("body %d: CompressGzipPooled: %v", i, err)
		}
		if !bytes.Equal(gzipDecoded, body) {
			t.Errorf("body %d: gzip round-trip mismatch (%d bytes in, %d out)", i, len(body), len(gzipDecoded))
		}
	}
}

// TestPooledCompressorsConcurrent is the real guard: distinct payloads compressed at the same
// time must each decode back to their own content, never to a neighbour's.
func TestPooledCompressorsConcurrent(t *testing.T) {
	const workers = 32

	var waitGroup sync.WaitGroup
	for worker := range workers {
		waitGroup.Add(1)
		go func(worker int) {
			defer waitGroup.Done()

			// Distinct length and content per worker so a swapped buffer cannot go unnoticed.
			body := []byte(strings.Repeat(fmt.Sprintf("worker-%02d ", worker), worker*40+1))

			CompressZstdPooled(body, func(compressed []byte) {
				if decoded := decodeZstd(t, compressed); !bytes.Equal(decoded, body) {
					t.Errorf("worker %d: zstd payload corrupted across pool reuse", worker)
				}
			})

			if err := CompressGzipPooled(body, func(compressed []byte) {
				if decoded := decodeGzip(t, compressed); !bytes.Equal(decoded, body) {
					t.Errorf("worker %d: gzip payload corrupted across pool reuse", worker)
				}
			}); err != nil {
				t.Errorf("worker %d: CompressGzipPooled: %v", worker, err)
			}
		}(worker)
	}
	waitGroup.Wait()
}

// TestMakeResponseFinalNegotiatesZstdFirst pins the codec preference: zstd when the client
// advertises it, gzip otherwise.
func TestMakeResponseFinalNegotiatesZstdFirst(t *testing.T) {
	cases := []struct {
		acceptEncoding string
		wantEncoding   string
	}{
		{"zstd, gzip, deflate", "zstd"},
		{"gzip, deflate", "gzip"},
		{"gzip", "gzip"},
		{"", ""},
	}

	for _, testCase := range cases {
		body := []byte(strings.Repeat("negotiation payload ", 100))
		response := MakeResponseFinal(&HandlerResponse{
			Body:     &body,
			Encoding: testCase.acceptEncoding,
			Headers:  map[string]string{},
		})

		got := response.Headers["Content-Encoding"]
		if got != testCase.wantEncoding {
			t.Errorf("Accept-Encoding %q: want Content-Encoding %q, got %q",
				testCase.acceptEncoding, testCase.wantEncoding, got)
		}
	}
}

// decodeBase64 unwraps the buffered invoke mode's body so it can be compared against what the
// streaming mode writes raw.
func decodeBase64(t *testing.T, encoded string) []byte {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	return decoded
}
