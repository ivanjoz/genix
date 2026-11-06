package bench_test

// Layer A: what each stage of one GET costs, serially, with -benchmem. Stages are cumulative —
// scan, then scan + colbin (the handler), then + zstd — so each one's cost is the difference
// with the previous. The concurrent picture is the router benchmark (router_bench_test.go).
//
// Prerequisites and the full command line: README.md in this folder.
//
//	GENIX_BENCH=1 go test ./tests/bench/ -run '^$' -bench . -benchmem

import (
	"app/core"
	"app/logistics"
	"app/production"
	"app/sales"
	"app/tests/bench"
	"context"
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var handlersByEndpoint = core.AppRouterType{
	"products":       production.GetProducts,
	"sale-orders":    sales.GetSaleOrders,
	"products-stock": logistics.GetProductsStock,
}

func requireBenchStack(b *testing.B) {
	if os.Getenv(bench.EnabledEnv) == "" {
		b.Skip("set " + bench.EnabledEnv + "=1 with the bench stack running (see tests/bench/README.md)")
	}
	bench.Setup()
}

func seededOrFatal(b *testing.B, dataset bench.Dataset) {
	if err := bench.EnsureSeeded(dataset); err != nil {
		b.Fatalf("seeding %s: %v", dataset.Name, err)
	}
}

// TestSeedBenchData seeds every dataset up front, so the first benchmark is not the one that pays
// for 50k inserts: GENIX_BENCH=1 go test ./tests/bench/ -run TestSeedBenchData -v
func TestSeedBenchData(t *testing.T) {
	if os.Getenv(bench.EnabledEnv) == "" {
		t.Skip("set " + bench.EnabledEnv + "=1 with the bench stack running")
	}
	for _, dataset := range append(slices.Clone(bench.Datasets), bench.MultiCompanyDatasets...) {
		if err := bench.EnsureSeeded(dataset); err != nil {
			t.Fatalf("seeding %s: %v", dataset.Name, err)
		}
		for endpointName, scan := range bench.ScanOnly {
			rowCount, err := scan(dataset.CompanyID)
			if err != nil {
				t.Fatalf("%s/%s: %v", dataset.Name, endpointName, err)
			}
			t.Logf("%s/%s: first sync returns %d rows", dataset.Name, endpointName, rowCount)
		}
	}
}

func makeHandlerArgs(endpoint bench.Endpoint, companyID int32) *core.HandlerArgs {
	query := map[string]string{}
	for key, values := range endpoint.BaseQuery {
		query[key] = values[0]
	}
	return &core.HandlerArgs{
		Method:   "GET",
		Route:    endpoint.Route,
		Query:    query,
		Accept:   core.ColbinMediaType,
		Encoding: "zstd",
		User:     &core.UsuarioToken{CompanyID: companyID, ID: bench.BenchUserID},
	}
}

func BenchmarkStageScan(b *testing.B) {
	requireBenchStack(b)
	for _, dataset := range bench.Datasets {
		for _, endpoint := range bench.Endpoints {
			b.Run(dataset.Name+"/"+endpoint.Name, func(b *testing.B) {
				seededOrFatal(b, dataset)
				scan := bench.ScanOnly[endpoint.Name]
				b.ReportAllocs()
				for b.Loop() {
					if _, err := scan(dataset.CompanyID); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkStageHandler is scan + colbin: the handler as the router calls it, minus auth and
// compression.
func BenchmarkStageHandler(b *testing.B) {
	requireBenchStack(b)
	for _, dataset := range bench.Datasets {
		for _, endpoint := range bench.Endpoints {
			b.Run(dataset.Name+"/"+endpoint.Name, func(b *testing.B) {
				seededOrFatal(b, dataset)
				handler := handlersByEndpoint[endpoint.Name]
				args := makeHandlerArgs(endpoint, dataset.CompanyID)
				b.ReportAllocs()
				bodyBytes := 0
				for b.Loop() {
					response := handler(args)
					if response.Error != "" || response.Body == nil {
						b.Fatalf("handler failed: %q", response.Error)
					}
					bodyBytes = len(*response.Body)
				}
				b.ReportMetric(float64(bodyBytes), "colbin-B")
			})
		}
	}
}

// BenchmarkStageHandlerZstd adds the compression SendLocalResponse applies before writing.
func BenchmarkStageHandlerZstd(b *testing.B) {
	requireBenchStack(b)
	for _, dataset := range bench.Datasets {
		for _, endpoint := range bench.Endpoints {
			b.Run(dataset.Name+"/"+endpoint.Name, func(b *testing.B) {
				seededOrFatal(b, dataset)
				handler := handlersByEndpoint[endpoint.Name]
				args := makeHandlerArgs(endpoint, dataset.CompanyID)
				b.ReportAllocs()
				compressedBytes := 0
				for b.Loop() {
					response := handler(args)
					if response.Error != "" || response.Body == nil {
						b.Fatalf("handler failed: %q", response.Error)
					}
					core.CompressZstdPooled(*response.Body, func(compressed []byte) {
						compressedBytes = len(compressed)
					})
				}
				b.ReportMetric(float64(compressedBytes), "zstd-B")
			})
		}
	}
}

// BenchmarkStageHandlerIncremental is the handler answering an incremental sync with nothing new
// (RequestURL's cursor): the steady state of every page load, and so the bulk of the traffic. It
// needs no fareward, unlike the router benchmark that also measures it.
func BenchmarkStageHandlerIncremental(b *testing.B) {
	requireBenchStack(b)
	for _, dataset := range bench.Datasets {
		for _, endpoint := range bench.Endpoints {
			b.Run(dataset.Name+"/"+endpoint.Name, func(b *testing.B) {
				seededOrFatal(b, dataset)
				handler := handlersByEndpoint[endpoint.Name]
				b.ReportAllocs()
				for b.Loop() {
					args := makeHandlerArgs(endpoint, dataset.CompanyID)
					readTime := core.SUnixTime()
					args.Query[endpoint.CursorParam] = strconv.Itoa(int(readTime-10)) + "." + strconv.Itoa(int(readTime))
					response := handler(args)
					if response.Error != "" || response.Body == nil {
						b.Fatalf("handler failed: %q", response.Error)
					}
				}
			})
		}
	}
}

// BenchmarkHandlerConcurrent runs the handler alone (scan + colbin, no auth, no fareward, no
// compression) at increasing concurrency, to separate what ScyllaDB and the ORM do under load from
// what the router adds. GENIX_BENCH_CONCURRENCY narrows the sweep, as in the router benchmark.
func BenchmarkHandlerConcurrent(b *testing.B) {
	requireBenchStack(b)
	concurrencyLevels := []int{16, 128}
	if configured := os.Getenv("GENIX_BENCH_CONCURRENCY"); configured != "" {
		concurrencyLevels = nil
		for _, levelText := range strings.Split(configured, ",") {
			if level, err := strconv.Atoi(strings.TrimSpace(levelText)); err == nil && level > 0 {
				concurrencyLevels = append(concurrencyLevels, level)
			}
		}
	}
	for _, dataset := range bench.Datasets {
		for _, endpoint := range bench.Endpoints {
			for _, concurrency := range concurrencyLevels {
				b.Run(dataset.Name+"/"+endpoint.Name+"/c"+strconv.Itoa(concurrency), func(b *testing.B) {
					seededOrFatal(b, dataset)
					handler := handlersByEndpoint[endpoint.Name]
					bench.RunAtConcurrency(b, concurrency, func() (int, error) {
						// Fresh args per call: the router builds one per request, and handlers may
						// write into them.
						response := handler(makeHandlerArgs(endpoint, dataset.CompanyID))
						if response.Error != "" {
							return 0, errors.New(response.Error)
						}
						return len(*response.Body), nil
					})
				})
			}
		}
	}
}

// BenchmarkCheckUser is the session token check every private request pays: base64, colbin
// decode and the keyed hash.
func BenchmarkCheckUser(b *testing.B) {
	requireBenchStack(b)
	args := &core.HandlerArgs{Headers: map[string]string{"Authorization": bench.MakeSessionToken(101, bench.BenchUserID)}}
	b.ReportAllocs()
	for b.Loop() {
		if user := core.CheckUser(args, 0); user.Error != "" {
			b.Fatal(user.Error)
		}
	}
}

// BenchmarkFarewardCharge is the GET base charge every private request sends, at increasing
// concurrency. The backend multiplexes all of them over one connection and the daemon admits
// max_inflight_per_connection (64) at a time, so the step past 64 is the number to watch.
func BenchmarkFarewardCharge(b *testing.B) {
	requireBenchStack(b)
	// Seeding is what sets the company's credit budget; without it every charge is refused.
	seededOrFatal(b, bench.Datasets[0])
	routeID := core.APIRouteID("GET.products")
	for _, concurrency := range []int{1, 64, 512, 2048} {
		b.Run("c"+strconv.Itoa(concurrency), func(b *testing.B) {
			bench.RunAtConcurrency(b, concurrency, func() (int, error) {
				_, err := core.ChargeAPIUsage(context.Background(), 101, bench.BenchUserID, routeID, "GET", 0, nil)
				if err != nil {
					return 0, errors.New("fareward charge: " + err.Error())
				}
				return 0, nil
			})
		})
	}
}
