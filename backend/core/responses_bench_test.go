package core_test

// Stage 0 baseline for the full response tail: MakeResponse (minijson) → MakeResponseFinal
// (compress + base64 + Lambda envelope). minijson's benchmarks cover the
// encode step alone; the gap between the two is what stages 3+ of
// PLAN_RESPONSE_SERIALIZATION_MEMORY.md have to work with.
//
// Run: go test ./core/ -bench 'Response' -benchmem -run '^$'

import (
	"app/core"
	"app/tests/fixtures"
	"io"
	"testing"
)

func benchmarkResponsePipeline(b *testing.B, productCount int, encoding string) {
	products := fixtures.MakeProducts(productCount)
	request := &core.HandlerArgs{Route: "products", Encoding: encoding}

	// Size the throughput report by the encoded payload, not the record count.
	sizingResponse := core.MakeResponse(request, &products)
	if sizingResponse.Body == nil {
		b.Fatalf("MakeResponse produced no body: %v", sizingResponse.Error)
	}
	b.SetBytes(int64(len(*sizingResponse.Body)))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		response := core.MakeResponse(request, &products)
		if response.Body == nil {
			b.Fatalf("MakeResponse produced no body: %v", response.Error)
		}
		response.Encoding = encoding

		final := core.MakeResponseFinal(&response)
		if len(final.Body) == 0 {
			b.Fatal("MakeResponseFinal produced an empty body")
		}
	}
}

func BenchmarkResponsePipelineZstd(b *testing.B) {
	benchmarkResponsePipeline(b, 1000, "zstd")
}

func BenchmarkResponsePipelineGzip(b *testing.B) {
	benchmarkResponsePipeline(b, 1000, "gzip")
}

// benchmarkStreamingPipeline mirrors benchmarkResponsePipeline for the RESPONSE_STREAM invoke
// mode, draining the response the way the Lambda runtime does so the base64 step it avoids is
// actually reflected in the numbers.
func benchmarkStreamingPipeline(b *testing.B, productCount int, encoding string) {
	products := fixtures.MakeProducts(productCount)
	request := &core.HandlerArgs{Route: "products", Encoding: encoding}

	sizingResponse := core.MakeResponse(request, &products)
	if sizingResponse.Body == nil {
		b.Fatalf("MakeResponse produced no body: %v", sizingResponse.Error)
	}
	b.SetBytes(int64(len(*sizingResponse.Body)))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		response := core.MakeResponse(request, &products)
		if response.Body == nil {
			b.Fatalf("MakeResponse produced no body: %v", response.Error)
		}
		response.Encoding = encoding

		streaming := core.MakeStreamingResponseFinal(&response)
		written, err := io.Copy(io.Discard, streaming)
		if err != nil {
			b.Fatalf("draining streaming response: %v", err)
		}
		if written == 0 {
			b.Fatal("streaming response was empty")
		}
		if err := streaming.Close(); err != nil {
			b.Fatalf("closing streaming response: %v", err)
		}
	}
}

func BenchmarkStreamingPipelineZstd(b *testing.B) {
	benchmarkStreamingPipeline(b, 1000, "zstd")
}

func BenchmarkStreamingPipelineGzip(b *testing.B) {
	benchmarkStreamingPipeline(b, 1000, "gzip")
}
