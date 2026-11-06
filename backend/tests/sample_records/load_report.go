package sample_records

// The load test's measurements: every request's latency and outcome, and the watched processes'
// memory and CPU, split by ramp stage.

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
)

type loadOutcome int8

const (
	loadOutcomeOK loadOutcome = iota
	loadOutcomeRefused
	loadOutcomeError
)

// clockTicksPerSecond is the unit of utime/stime in /proc/<pid>/stat (USER_HZ, 100 on Linux).
const clockTicksPerSecond = 100

type loadOperationSamples struct {
	durations          []time.Duration
	refused, errors    int
	responseBytesTotal int64
}

type loadProcessSamples struct {
	maxRSSBytes                 int64
	firstCPUTicks, lastCPUTicks int64
}

// loadRecorder collects into the current stage. Stage -1 is the setup, which is not reported.
type loadRecorder struct {
	mutex        sync.Mutex
	currentStage int
	operations   map[int]map[string]*loadOperationSamples
	processes    map[int]map[string]*loadProcessSamples
	// The live window: what happened since the last live status line (load_flow.go), and each
	// watched process's latest sample.
	live loadLiveWindow
}

type loadLiveWindow struct {
	saleDurations             []time.Duration
	requests, errors, refused int
	rssBytesByProcess         map[string]int64
	cpuTicksByProcess         map[string]int64
}

func newLoadRecorder() *loadRecorder {
	recorder := &loadRecorder{}
	recorder.reset()
	return recorder
}

func (recorder *loadRecorder) reset() {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.currentStage = -1
	recorder.operations = map[int]map[string]*loadOperationSamples{}
	recorder.processes = map[int]map[string]*loadProcessSamples{}
}

func (recorder *loadRecorder) startStage(stageIndex int) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.currentStage = stageIndex
}

func (recorder *loadRecorder) record(operation string, elapsed time.Duration, outcome loadOutcome, responseBytes int) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	stageOperations := recorder.operations[recorder.currentStage]
	if stageOperations == nil {
		stageOperations = map[string]*loadOperationSamples{}
		recorder.operations[recorder.currentStage] = stageOperations
	}
	samples := stageOperations[operation]
	if samples == nil {
		samples = &loadOperationSamples{}
		stageOperations[operation] = samples
	}
	samples.durations = append(samples.durations, elapsed)
	samples.responseBytesTotal += int64(responseBytes)
	recorder.live.requests++
	switch outcome {
	case loadOutcomeRefused:
		samples.refused++
		recorder.live.refused++
	case loadOutcomeError:
		samples.errors++
		recorder.live.errors++
	case loadOutcomeOK:
		if operation == "POST.sale-order" {
			recorder.live.saleDurations = append(recorder.live.saleDurations, elapsed)
		}
	}
}

// takeLiveWindow returns what happened since the previous call and starts a new window. The
// process maps are the latest samples, kept across windows.
func (recorder *loadRecorder) takeLiveWindow() loadLiveWindow {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	window := recorder.live
	window.rssBytesByProcess = maps.Clone(recorder.live.rssBytesByProcess)
	window.cpuTicksByProcess = maps.Clone(recorder.live.cpuTicksByProcess)
	recorder.live = loadLiveWindow{
		rssBytesByProcess: recorder.live.rssBytesByProcess, cpuTicksByProcess: recorder.live.cpuTicksByProcess,
	}
	return window
}

func (recorder *loadRecorder) sampleProcess(processName string, rssBytes, cpuTicks int64) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	stageProcesses := recorder.processes[recorder.currentStage]
	if stageProcesses == nil {
		stageProcesses = map[string]*loadProcessSamples{}
		recorder.processes[recorder.currentStage] = stageProcesses
	}
	samples := stageProcesses[processName]
	if samples == nil {
		samples = &loadProcessSamples{firstCPUTicks: cpuTicks}
		stageProcesses[processName] = samples
	}
	samples.maxRSSBytes = max(samples.maxRSSBytes, rssBytes)
	samples.lastCPUTicks = cpuTicks
	if recorder.live.rssBytesByProcess == nil {
		recorder.live.rssBytesByProcess, recorder.live.cpuTicksByProcess = map[string]int64{}, map[string]int64{}
	}
	recorder.live.rssBytesByProcess[processName] = rssBytes
	recorder.live.cpuTicksByProcess[processName] = cpuTicks
}

type loadOperationResult struct {
	name                   string
	count, refused, errors int
	perSecond              float64
	p50, p99               time.Duration
	averageResponseKB      float64
}

type loadProcessResult struct {
	name       string
	maxRSSMB   int64
	cpuPercent float64
}

type loadStageResult struct {
	stage, cashiersPerCompany, companies int
	salesPerSecond, requestsPerSecond    float64
	saleP50, saleP99                     time.Duration
	errorPct                             float64
	operations                           []loadOperationResult
	processes                            []loadProcessResult
}

func (recorder *loadRecorder) summarizeStage(stageIndex, cashiersPerCompany, companies, stageSeconds int) loadStageResult {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	result := loadStageResult{stage: stageIndex + 1, cashiersPerCompany: cashiersPerCompany, companies: companies}

	totalRequests, totalErrors := 0, 0
	for operationName, samples := range recorder.operations[stageIndex] {
		sortedDurations := slices.Clone(samples.durations)
		slices.Sort(sortedDurations)
		operation := loadOperationResult{
			name: operationName, count: len(sortedDurations), refused: samples.refused, errors: samples.errors,
			perSecond: float64(len(sortedDurations)) / float64(stageSeconds),
			p50:       percentileOf(sortedDurations, 0.50),
			p99:       percentileOf(sortedDurations, 0.99),
		}
		if operation.count > 0 {
			operation.averageResponseKB = float64(samples.responseBytesTotal) / float64(operation.count) / 1024
		}
		result.operations = append(result.operations, operation)
		totalRequests += operation.count
		totalErrors += operation.errors
		if operationName == "POST.sale-order" {
			result.salesPerSecond = float64(operation.count-operation.refused-operation.errors) / float64(stageSeconds)
			result.saleP50, result.saleP99 = operation.p50, operation.p99
		}
	}
	slices.SortFunc(result.operations, func(left, right loadOperationResult) int { return right.count - left.count })
	result.requestsPerSecond = float64(totalRequests) / float64(stageSeconds)
	if totalRequests > 0 {
		result.errorPct = 100 * float64(totalErrors) / float64(totalRequests)
	}

	for processName, samples := range recorder.processes[stageIndex] {
		result.processes = append(result.processes, loadProcessResult{
			name:     processName,
			maxRSSMB: samples.maxRSSBytes >> 20,
			cpuPercent: 100 * float64(samples.lastCPUTicks-samples.firstCPUTicks) /
				clockTicksPerSecond / float64(stageSeconds),
		})
	}
	slices.SortFunc(result.processes, func(left, right loadProcessResult) int { return strings.Compare(left.name, right.name) })
	return result
}

func percentileOf(sortedDurations []time.Duration, fraction float64) time.Duration {
	if len(sortedDurations) == 0 {
		return 0
	}
	return sortedDurations[min(len(sortedDurations)-1, int(fraction*float64(len(sortedDurations))))]
}

func (result loadStageResult) String() string {
	text := strings.Builder{}
	fmt.Fprintf(&text, "LoadTest:: etapa %d — %d cajeros × %d empresas: %.1f ventas/s, %.1f req/s, venta p50 %v p99 %v, errores %.2f%%\n",
		result.stage, result.cashiersPerCompany, result.companies, result.salesPerSecond, result.requestsPerSecond,
		result.saleP50.Round(time.Millisecond), result.saleP99.Round(time.Millisecond), result.errorPct)
	fmt.Fprintf(&text, "  %-42s %7s %8s %9s %9s %8s %7s %9s\n", "operation", "count", "req/s", "p50", "p99", "refused", "errors", "avg KB")
	for _, operation := range result.operations {
		fmt.Fprintf(&text, "  %-42s %7d %8.2f %9v %9v %8d %7d %9.1f\n", operation.name, operation.count, operation.perSecond,
			operation.p50.Round(time.Millisecond), operation.p99.Round(time.Millisecond), operation.refused, operation.errors,
			operation.averageResponseKB)
	}
	for _, process := range result.processes {
		fmt.Fprintf(&text, "  process %-12s peak RSS %5d MB, CPU %5.0f%%\n", process.name, process.maxRSSMB, process.cpuPercent)
	}
	return text.String()
}

// formatLoadReport is the closing table: one row per stage.
func formatLoadReport(stages []loadStageResult) string {
	text := strings.Builder{}
	text.WriteString("LoadTest:: resumen\n")
	fmt.Fprintf(&text, "  %6s %10s %9s %8s %9s %9s %8s\n", "stage", "cashiers", "sales/s", "req/s", "sale p50", "sale p99", "error%")
	for _, stage := range stages {
		fmt.Fprintf(&text, "  %6d %10d %9.1f %8.1f %9v %9v %8.2f\n", stage.stage, stage.cashiersPerCompany*stage.companies,
			stage.salesPerSecond, stage.requestsPerSecond, stage.saleP50.Round(time.Millisecond),
			stage.saleP99.Round(time.Millisecond), stage.errorPct)
	}
	return text.String()
}
