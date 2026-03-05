package system

import (
	"runtime"
	"sort"
	"strings"
	"time"
)

type GoHeapPackageUsage struct {
	PackageName string  `json:"package_name"`
	InuseBytes  uint64  `json:"inuse_bytes"`
	InuseMiB    float64 `json:"inuse_mib"`
	Percent     float64 `json:"percent"`
}

type GoHeapPackageReport struct {
	TimestampUnix    int64                `json:"timestamp_unix"`
	PackageLimit     int                  `json:"package_limit"`
	GoHeapInuseBytes uint64               `json:"go_heap_inuse_bytes"`
	GoHeapObjects    uint64               `json:"go_heap_objects"`
	BackendRSSBytes  uint64               `json:"backend_rss_bytes"`
	TopPackages      []GoHeapPackageUsage `json:"top_packages"`
}

func CollectGoHeapPackageReport(requestedPackageLimit int) (GoHeapPackageReport, []string) {
	normalizedPackageLimit := requestedPackageLimit
	if normalizedPackageLimit <= 0 {
		normalizedPackageLimit = 20
	}
	if normalizedPackageLimit > 100 {
		normalizedPackageLimit = 100
	}

	reportWarnings := []string{}
	memoryStatistics := runtime.MemStats{}
	runtime.ReadMemStats(&memoryStatistics)

	heapAllocationByPackage := collectHeapAllocationByPackage()
	sortedPackageUsage := make([]GoHeapPackageUsage, 0, len(heapAllocationByPackage))

	for packageName, inuseBytes := range heapAllocationByPackage {
		usagePercent := 0.0
		if memoryStatistics.HeapInuse > 0 {
			usagePercent = (float64(inuseBytes) / float64(memoryStatistics.HeapInuse)) * 100.0
		}

		sortedPackageUsage = append(sortedPackageUsage, GoHeapPackageUsage{
			PackageName: packageName,
			InuseBytes:  inuseBytes,
			InuseMiB:    float64(inuseBytes) / (1024.0 * 1024.0),
			Percent:     usagePercent,
		})
	}

	sort.Slice(sortedPackageUsage, func(leftIndex int, rightIndex int) bool {
		return sortedPackageUsage[leftIndex].InuseBytes > sortedPackageUsage[rightIndex].InuseBytes
	})

	if len(sortedPackageUsage) > normalizedPackageLimit {
		sortedPackageUsage = sortedPackageUsage[:normalizedPackageLimit]
	}

	report := GoHeapPackageReport{
		TimestampUnix:    time.Now().Unix(),
		PackageLimit:     normalizedPackageLimit,
		GoHeapInuseBytes: memoryStatistics.HeapInuse,
		GoHeapObjects:    memoryStatistics.HeapObjects,
		TopPackages:      sortedPackageUsage,
	}

	backendProcessMetric, backendProcessMetricError := collectBackendProcessMetrics()
	if backendProcessMetricError != nil {
		reportWarnings = append(reportWarnings, "backend_process:"+backendProcessMetricError.Error())
	} else {
		report.BackendRSSBytes = backendProcessMetric.RSSBytes
	}

	return report, reportWarnings
}

func collectHeapAllocationByPackage() map[string]uint64 {
	// Runtime returns sampled heap records and may require retries if the destination is too small.
	heapProfileRecordCount, _ := runtime.MemProfile(nil, false)
	if heapProfileRecordCount == 0 {
		return map[string]uint64{}
	}

	heapProfileRecords := make([]runtime.MemProfileRecord, heapProfileRecordCount*2)
	for {
		retrievedRecordCount, ok := runtime.MemProfile(heapProfileRecords, false)
		if ok {
			heapProfileRecords = heapProfileRecords[:retrievedRecordCount]
			break
		}
		heapProfileRecords = make([]runtime.MemProfileRecord, retrievedRecordCount*2)
	}

	heapAllocationByPackage := map[string]uint64{}
	for _, heapProfileRecord := range heapProfileRecords {
		if heapProfileRecord.InUseBytes() <= 0 {
			continue
		}

		owningPackageName := inferOwningPackageFromStack(heapProfileRecord.Stack())
		if len(owningPackageName) == 0 {
			owningPackageName = "unknown"
		}

		heapAllocationByPackage[owningPackageName] += uint64(heapProfileRecord.InUseBytes())
	}

	return heapAllocationByPackage
}

func inferOwningPackageFromStack(stackProgramCounters []uintptr) string {
	for _, stackProgramCounter := range stackProgramCounters {
		runtimeFunction := runtime.FuncForPC(stackProgramCounter)
		if runtimeFunction == nil {
			continue
		}

		fullFunctionName := runtimeFunction.Name()
		if len(fullFunctionName) == 0 {
			continue
		}
		if isRuntimeSupportFunction(fullFunctionName) {
			continue
		}

		owningPackageName := extractPackageNameFromFunction(fullFunctionName)
		if len(owningPackageName) > 0 {
			return owningPackageName
		}
	}

	return ""
}

func isRuntimeSupportFunction(fullFunctionName string) bool {
	return strings.HasPrefix(fullFunctionName, "runtime.") ||
		strings.HasPrefix(fullFunctionName, "internal/") ||
		strings.HasPrefix(fullFunctionName, "reflect.")
}

func extractPackageNameFromFunction(fullFunctionName string) string {
	lastPathSeparatorIndex := strings.LastIndex(fullFunctionName, "/")
	firstSymbolSearchIndex := 0
	if lastPathSeparatorIndex >= 0 {
		firstSymbolSearchIndex = lastPathSeparatorIndex + 1
	}

	packageAndFunctionSuffix := fullFunctionName[firstSymbolSearchIndex:]
	firstSymbolSeparatorIndex := strings.Index(packageAndFunctionSuffix, ".")
	if firstSymbolSeparatorIndex <= 0 {
		return ""
	}

	packageNameEndIndex := firstSymbolSearchIndex + firstSymbolSeparatorIndex
	return fullFunctionName[:packageNameEndIndex]
}
