package main

// Layer B: the whole request path under concurrency — LocalHandler → mainHandler → CheckUser →
// fareward charge → handler (ORM scan + colbin) → zstd → write — at a fixed number of requests in
// flight. It reports throughput, p50/p99 latency, peak memory and stdout log bytes per request.
// The net/http server itself is not in the loop: the request goes straight to LocalHandler, so
// per-connection costs (one goroutine, ~8 KB of bufio buffers) come on top of these numbers.
//
// Prerequisites and command lines: tests/bench/README.md.
//
//	GENIX_BENCH=1 go test . -run '^$' -bench Router -benchtime 2000x -cpu 2

import (
	"app/core"
	"app/tests/bench"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// routerConcurrencyEnv overrides the concurrency sweep, e.g. GENIX_BENCH_CONCURRENCY=64,512.
const routerConcurrencyEnv = "GENIX_BENCH_CONCURRENCY"

// The two sweeps differ because the two sync kinds differ by orders of magnitude in memory: an
// incremental sync holds a few KB per request, a first sync of 50k products holds the full colbin
// body and its compressed copy at once.
var defaultIncrementalConcurrency = []int{64, 512, 2048, 5000}
var defaultFirstSyncConcurrency = []int{16, 128, 512}

// noAdmissionEnv turns the server's admission control off, to compare against a run without it.
const noAdmissionEnv = "GENIX_BENCH_NO_ADMISSION"

// routerScenario is the set of companies one sub-benchmark spreads its requests over, round robin.
type routerScenario struct {
	name     string
	datasets []bench.Dataset
}

func BenchmarkRouter(b *testing.B) {
	if os.Getenv(bench.EnabledEnv) == "" {
		b.Skip("set " + bench.EnabledEnv + "=1 with the bench stack running (see tests/bench/README.md)")
	}
	bench.Setup()
	makeAppHandlers()
	// What the VPS server runs with: request slots and GOMEMLIMIT. Excess requests queue, and past
	// the queue's wait they answer 503 — which err-% then shows.
	if os.Getenv(noAdmissionEnv) == "" {
		core.ConfigureServerAdmission()
	}

	for _, syncKind := range []bench.SyncKind{bench.IncrementalSync, bench.FirstSync} {
		scenarios := []routerScenario{}
		// An incremental sync with nothing new reads two empty ranges whatever the catalog size, so
		// one single-company dataset is enough for it; first syncs sweep the three sizes.
		singleCompanyDatasets := bench.Datasets
		if syncKind == bench.IncrementalSync {
			singleCompanyDatasets = []bench.Dataset{bench.Datasets[1]}
		}
		for _, dataset := range singleCompanyDatasets {
			scenarios = append(scenarios, routerScenario{name: dataset.Name, datasets: []bench.Dataset{dataset}})
		}
		// The same load spread over eight tenants: eight partitions instead of one hot one.
		scenarios = append(scenarios, routerScenario{name: "multi8x10k", datasets: bench.MultiCompanyDatasets})

		for _, scenario := range scenarios {
			for _, endpoint := range bench.Endpoints {
				for _, concurrency := range routerConcurrencyLevels(syncKind) {
					benchName := fmt.Sprintf("%s/%s/%s/c%d", syncKind, scenario.name, endpoint.Name, concurrency)
					b.Run(benchName, func(b *testing.B) {
						authorizations := make([]string, 0, len(scenario.datasets))
						for _, dataset := range scenario.datasets {
							if err := bench.EnsureSeeded(dataset); err != nil {
								b.Fatalf("seeding %s: %v", dataset.Name, err)
							}
							authorizations = append(authorizations, bench.MakeSessionToken(dataset.CompanyID, bench.BenchUserID))
						}
						var requestCounter atomic.Uint64
						bench.RunAtConcurrency(b, concurrency, func() (int, error) {
							authorization := authorizations[requestCounter.Add(1)%uint64(len(authorizations))]
							return serveOneRequest(endpoint, syncKind, authorization)
						})
					})
				}
			}
		}
	}
}

// serveOneRequest builds the request the browser sends and runs it through LocalHandler, the
// same entry point the VPS server mounts.
func serveOneRequest(endpoint bench.Endpoint, syncKind bench.SyncKind, authorization string) (int, error) {
	request := httptest.NewRequest(http.MethodGet, bench.RequestURL(endpoint, syncKind), nil)
	request.Header.Set("Authorization", authorization)
	request.Header.Set("Accept", core.ColbinMediaType)
	request.Header.Set("Accept-Encoding", "zstd")

	responseWriter := &discardResponseWriter{header: http.Header{}}
	LocalHandler(responseWriter, request)

	if responseWriter.statusCode != 0 && responseWriter.statusCode != http.StatusOK {
		return responseWriter.writtenBytes, fmt.Errorf("%s answered %d: %s",
			endpoint.Route, responseWriter.statusCode, responseWriter.errorBody)
	}
	return responseWriter.writtenBytes, nil
}

func routerConcurrencyLevels(syncKind bench.SyncKind) []int {
	if configured := os.Getenv(routerConcurrencyEnv); configured != "" {
		levels := []int{}
		for _, levelText := range strings.Split(configured, ",") {
			if level, err := strconv.Atoi(strings.TrimSpace(levelText)); err == nil && level > 0 {
				levels = append(levels, level)
			}
		}
		return levels
	}
	if syncKind == bench.FirstSync {
		return defaultFirstSyncConcurrency
	}
	return defaultIncrementalConcurrency
}

// discardResponseWriter counts the body instead of keeping it: a recorder would hold every
// response in memory and inflate the peak the benchmark is there to measure. Only an error body
// is kept, truncated, so a failure reports why.
type discardResponseWriter struct {
	header       http.Header
	statusCode   int
	writtenBytes int
	errorBody    string
}

func (writer *discardResponseWriter) Header() http.Header { return writer.header }

func (writer *discardResponseWriter) WriteHeader(statusCode int) { writer.statusCode = statusCode }

func (writer *discardResponseWriter) Write(body []byte) (int, error) {
	writer.writtenBytes += len(body)
	if writer.statusCode != 0 && writer.statusCode != http.StatusOK && writer.errorBody == "" {
		writer.errorBody = string(body[:min(len(body), 300)])
	}
	return len(body), nil
}
