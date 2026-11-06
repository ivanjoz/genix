package bench

import (
	"io"
	"os"
	"runtime/debug"
	"runtime/metrics"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// maxHeapEnv caps the live heap a run may reach before it is stopped. The machine running the
// benchmark also hosts ScyllaDB, so a first sync at high concurrency could otherwise push it into
// the OOM killer instead of reporting where the ceiling was.
const maxHeapEnv = "GENIX_BENCH_MAX_HEAP_MB"

// Operation is one request. It returns the bytes the client would have received.
type Operation func() (responseBytes int, err error)

// RunAtConcurrency runs b.N operations from exactly `concurrency` goroutines and reports what a
// capacity plan needs: throughput, latency percentiles, the peak memory the process reached, and
// how many bytes of stdout logging each request produced.
//
// It does not use b.RunParallel because that sizes its pool as parallelism × GOMAXPROCS; here the
// number of requests in flight is the variable under test, so it is set exactly.
func RunAtConcurrency(b *testing.B, concurrency int, operation Operation) {
	maxHeapBytes := uint64(6144) << 20
	if configuredMB, err := strconv.Atoi(os.Getenv(maxHeapEnv)); err == nil && configuredMB > 0 {
		maxHeapBytes = uint64(configuredMB) << 20
	}

	// Hand the previous run's memory back to the OS first; otherwise peak-go-MB carries the high
	// water mark of whatever sub-benchmark ran before this one.
	debug.FreeOSMemory()

	logCounter := captureStdout()
	defer logCounter.restore()

	sampler := startMemorySampler(maxHeapBytes)

	var remainingOps atomic.Int64
	remainingOps.Store(int64(b.N))
	var failedOps atomic.Int64
	var firstError atomic.Value
	var totalResponseBytes atomic.Int64
	latenciesByWorker := make([][]time.Duration, concurrency)

	b.ResetTimer()
	startedAt := time.Now()

	var workers sync.WaitGroup
	for workerIndex := range concurrency {
		workers.Add(1)
		go func() {
			defer workers.Done()
			workerLatencies := make([]time.Duration, 0, b.N/concurrency+1)
			// A worker stops taking tickets once the heap ceiling is crossed; requests already in
			// flight finish, which is why the ceiling sits below the machine's real limit.
			for remainingOps.Add(-1) >= 0 && !sampler.limitExceeded.Load() {
				requestStartedAt := time.Now()
				responseBytes, err := operation()
				workerLatencies = append(workerLatencies, time.Since(requestStartedAt))
				totalResponseBytes.Add(int64(responseBytes))
				if err != nil {
					failedOps.Add(1)
					firstError.CompareAndSwap(nil, err.Error())
				}
			}
			latenciesByWorker[workerIndex] = workerLatencies
		}()
	}
	workers.Wait()
	elapsed := time.Since(startedAt)
	b.StopTimer()

	peaks := sampler.stop()
	logBytes := logCounter.restore()

	if sampler.limitExceeded.Load() {
		b.Fatalf("live heap crossed %s=%d MB at %d concurrent requests (peak %d MB); raise it only if the machine has the memory",
			maxHeapEnv, maxHeapBytes>>20, concurrency, peaks.heapObjectsBytes>>20)
	}
	// Failures are a result, not an abort: past fareward's in-flight cap some requests are refused
	// by design, and the rate at which that happens is one of the numbers this run is for.
	if failedOps.Load() > 0 {
		b.Logf("%d of %d requests failed; first error: %v", failedOps.Load(), b.N, firstError.Load())
	}
	b.ReportMetric(100*float64(failedOps.Load())/float64(b.N), "err-%")

	allLatencies := slices.Concat(latenciesByWorker...)
	slices.Sort(allLatencies)
	percentileMs := func(percentile float64) float64 {
		if len(allLatencies) == 0 {
			return 0
		}
		index := min(int(float64(len(allLatencies))*percentile), len(allLatencies)-1)
		return float64(allLatencies[index].Microseconds()) / 1000
	}

	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "req/s")
	b.ReportMetric(percentileMs(0.50), "p50-ms")
	b.ReportMetric(percentileMs(0.99), "p99-ms")
	b.ReportMetric(float64(peaks.heapObjectsBytes)/(1<<20), "peak-heap-MB")
	b.ReportMetric(float64(peaks.goMappedBytes)/(1<<20), "peak-go-MB")
	b.ReportMetric(float64(peaks.goroutines), "peak-goroutines")
	b.ReportMetric(float64(totalResponseBytes.Load())/float64(b.N), "resp-B/op")
	b.ReportMetric(float64(logBytes)/float64(b.N), "log-B/op")
}

// memoryPeaks are the maxima the sampler saw. heapObjectsBytes is the live heap (what GOMEMLIMIT
// and the OOM killer care about first); goMappedBytes is everything the runtime has mapped, the
// closest in-process figure to the RSS the server would show.
type memoryPeaks struct {
	heapObjectsBytes uint64
	goMappedBytes    uint64
	goroutines       uint64
}

type memorySampler struct {
	limitExceeded atomic.Bool
	done          chan struct{}
	finished      chan memoryPeaks
}

// startMemorySampler polls runtime/metrics every 5 ms. runtime/metrics and not ReadMemStats,
// because ReadMemStats stops the world and would itself distort a run with thousands of goroutines.
func startMemorySampler(maxHeapBytes uint64) *memorySampler {
	sampler := &memorySampler{done: make(chan struct{}), finished: make(chan memoryPeaks, 1)}
	samples := []metrics.Sample{
		{Name: "/memory/classes/heap/objects:bytes"},
		{Name: "/memory/classes/total:bytes"},
		{Name: "/sched/goroutines:goroutines"},
	}
	go func() {
		peaks := memoryPeaks{}
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			metrics.Read(samples)
			peaks.heapObjectsBytes = max(peaks.heapObjectsBytes, samples[0].Value.Uint64())
			peaks.goMappedBytes = max(peaks.goMappedBytes, samples[1].Value.Uint64())
			peaks.goroutines = max(peaks.goroutines, samples[2].Value.Uint64())
			if peaks.heapObjectsBytes > maxHeapBytes {
				sampler.limitExceeded.Store(true)
			}
			select {
			case <-sampler.done:
				sampler.finished <- peaks
				return
			case <-ticker.C:
			}
		}
	}()
	return sampler
}

func (sampler *memorySampler) stop() memoryPeaks {
	close(sampler.done)
	return <-sampler.finished
}

// stdoutCounter swaps os.Stdout for a pipe and counts what crosses it. core.Log prints every line
// on a VPS (doLog is true whenever IS_SERVERLESS is false), so the formatting and the write are part
// of each request's cost; the counter turns that into a per-request byte figure. The bytes are
// discarded, which is cheaper than the journald the production server writes to.
type stdoutCounter struct {
	originalStdout *os.File
	pipeWriter     *os.File
	countedBytes   chan int64
	restored       bool
	finalBytes     int64
}

func captureStdout() *stdoutCounter {
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		panic("bench: creating the stdout pipe: " + err.Error())
	}
	counter := &stdoutCounter{originalStdout: os.Stdout, pipeWriter: pipeWriter, countedBytes: make(chan int64, 1)}
	go func() {
		copiedBytes, _ := io.Copy(io.Discard, pipeReader)
		pipeReader.Close()
		counter.countedBytes <- copiedBytes
	}()
	os.Stdout = pipeWriter
	return counter
}

// restore puts os.Stdout back and returns the bytes logged; calling it twice is safe, which lets
// the caller both defer it and read it.
func (counter *stdoutCounter) restore() int64 {
	if counter.restored {
		return counter.finalBytes
	}
	counter.restored = true
	os.Stdout = counter.originalStdout
	counter.pipeWriter.Close()
	counter.finalBytes = <-counter.countedBytes
	return counter.finalBytes
}
