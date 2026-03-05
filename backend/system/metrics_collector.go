package system

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ServerMetricsCollector struct {
	selectedInterfaceName string
	previousCPUTotalTicks uint64
	previousCPUIdleTicks  uint64
	previousRXTotalBytes  uint64
	previousTXTotalBytes  uint64
	hasCPUSampleBaseline  bool
	hasNetworkBaseline    bool
}

func NewServerMetricsCollector(interfaceName string) *ServerMetricsCollector {
	return &ServerMetricsCollector{
		selectedInterfaceName: strings.TrimSpace(interfaceName),
	}
}

func (collector *ServerMetricsCollector) CollectSnapshot(mountPath string, activeHTTPConnections int64) (ServerMetricsSnapshot, []string) {
	normalizedMountPath := strings.TrimSpace(mountPath)
	if len(normalizedMountPath) == 0 {
		normalizedMountPath = "/"
	}

	snapshot := ServerMetricsSnapshot{
		TimestampUnix: time.Now().Unix(),
		Connections: ConnectionsMetrics{
			HTTPActive: activeHTTPConnections,
		},
		Disk: DiskMetrics{
			MountPath: normalizedMountPath,
		},
	}

	collectionWarnings := []string{}

	cpuMetric, cpuMetricError := collector.collectCPUMetrics()
	if cpuMetricError != nil {
		collectionWarnings = append(collectionWarnings, "cpu:"+cpuMetricError.Error())
	} else {
		snapshot.CPU = cpuMetric
	}

	memoryMetric, memoryMetricError := collectMemoryMetrics()
	if memoryMetricError != nil {
		collectionWarnings = append(collectionWarnings, "memory:"+memoryMetricError.Error())
	} else {
		snapshot.Memory = memoryMetric
	}

	backendProcessMetric, backendProcessMetricError := collectBackendProcessMetrics()
	if backendProcessMetricError != nil {
		collectionWarnings = append(collectionWarnings, "backend_process:"+backendProcessMetricError.Error())
	} else {
		snapshot.BackendProcess = backendProcessMetric
	}

	diskMetric, diskMetricError := collectDiskMetrics(normalizedMountPath)
	if diskMetricError != nil {
		collectionWarnings = append(collectionWarnings, "disk:"+diskMetricError.Error())
	} else {
		snapshot.Disk = diskMetric
	}

	bandwidthMetric, bandwidthMetricError := collector.collectBandwidthMetrics()
	if bandwidthMetricError != nil {
		collectionWarnings = append(collectionWarnings, "bandwidth:"+bandwidthMetricError.Error())
	} else {
		snapshot.Bandwidth = bandwidthMetric
	}

	return snapshot, collectionWarnings
}

func (collector *ServerMetricsCollector) collectCPUMetrics() (CPUMetrics, error) {
	cpuTotalTicks, cpuIdleTicks, readError := readCPUStatTicks()
	if readError != nil {
		return CPUMetrics{}, readError
	}

	if !collector.hasCPUSampleBaseline {
		collector.previousCPUTotalTicks = cpuTotalTicks
		collector.previousCPUIdleTicks = cpuIdleTicks
		collector.hasCPUSampleBaseline = true
		return CPUMetrics{PercentUsed: 0}, nil
	}

	totalDeltaTicks := cpuTotalTicks - collector.previousCPUTotalTicks
	idleDeltaTicks := cpuIdleTicks - collector.previousCPUIdleTicks

	collector.previousCPUTotalTicks = cpuTotalTicks
	collector.previousCPUIdleTicks = cpuIdleTicks

	if totalDeltaTicks == 0 {
		return CPUMetrics{PercentUsed: 0}, nil
	}

	usagePercent := 100.0 * (float64(totalDeltaTicks-idleDeltaTicks) / float64(totalDeltaTicks))
	if usagePercent < 0 {
		usagePercent = 0
	}
	if usagePercent > 100 {
		usagePercent = 100
	}

	return CPUMetrics{PercentUsed: usagePercent}, nil
}

func collectMemoryMetrics() (MemoryMetrics, error) {
	memInfoMap, readError := readMemInfoValues()
	if readError != nil {
		return MemoryMetrics{}, readError
	}

	totalBytes := memInfoMap["MemTotal"] * 1024
	availableBytes := memInfoMap["MemAvailable"] * 1024
	if totalBytes == 0 {
		return MemoryMetrics{}, errors.New("MemTotal es 0")
	}

	usedBytes := totalBytes - availableBytes
	usagePercent := (float64(usedBytes) / float64(totalBytes)) * 100.0

	return MemoryMetrics{
		TotalBytes:     totalBytes,
		UsedBytes:      usedBytes,
		AvailableBytes: availableBytes,
		PercentUsed:    usagePercent,
	}, nil
}

func collectDiskMetrics(mountPath string) (DiskMetrics, error) {
	var fileSystemStats syscall.Statfs_t
	if statError := syscall.Statfs(mountPath, &fileSystemStats); statError != nil {
		return DiskMetrics{}, statError
	}

	totalBytes := fileSystemStats.Blocks * uint64(fileSystemStats.Bsize)
	freeBytes := fileSystemStats.Bavail * uint64(fileSystemStats.Bsize)
	usedBytes := totalBytes - freeBytes

	usagePercent := 0.0
	if totalBytes > 0 {
		usagePercent = (float64(usedBytes) / float64(totalBytes)) * 100.0
	}

	return DiskMetrics{
		MountPath:   mountPath,
		TotalBytes:  totalBytes,
		UsedBytes:   usedBytes,
		FreeBytes:   freeBytes,
		PercentUsed: usagePercent,
	}, nil
}

func collectBackendProcessMetrics() (BackendProcessMetrics, error) {
	statusFileBytes, readError := os.ReadFile("/proc/self/status")
	if readError != nil {
		return BackendProcessMetrics{}, readError
	}

	statusFileLines := strings.Split(string(statusFileBytes), "\n")
	for _, statusLine := range statusFileLines {
		cleanStatusLine := strings.TrimSpace(statusLine)
		if !strings.HasPrefix(cleanStatusLine, "VmRSS:") {
			continue
		}

		statusLineFields := strings.Fields(cleanStatusLine)
		// Expected: ["VmRSS:", "<value>", "kB"]
		if len(statusLineFields) < 2 {
			return BackendProcessMetrics{}, errors.New("formato VmRSS inválido")
		}

		rssKilobytes, parseError := strconv.ParseUint(statusLineFields[1], 10, 64)
		if parseError != nil {
			return BackendProcessMetrics{}, parseError
		}

		return BackendProcessMetrics{
			RSSBytes: rssKilobytes * 1024,
		}, nil
	}

	return BackendProcessMetrics{}, errors.New("no se encontró VmRSS en /proc/self/status")
}

func (collector *ServerMetricsCollector) collectBandwidthMetrics() (BandwidthMetrics, error) {
	interfaceName, totalRXBytes, totalTXBytes, readError := readNetworkCounters(collector.selectedInterfaceName)
	if readError != nil {
		return BandwidthMetrics{}, readError
	}

	rxBytesPerSecond := uint64(0)
	txBytesPerSecond := uint64(0)
	if collector.hasNetworkBaseline {
		if totalRXBytes >= collector.previousRXTotalBytes {
			rxBytesPerSecond = totalRXBytes - collector.previousRXTotalBytes
		}
		if totalTXBytes >= collector.previousTXTotalBytes {
			txBytesPerSecond = totalTXBytes - collector.previousTXTotalBytes
		}
	}

	collector.selectedInterfaceName = interfaceName
	collector.previousRXTotalBytes = totalRXBytes
	collector.previousTXTotalBytes = totalTXBytes
	collector.hasNetworkBaseline = true

	return BandwidthMetrics{
		InterfaceName: interfaceName,
		RXBytesPerSec: rxBytesPerSecond,
		TXBytesPerSec: txBytesPerSecond,
		TotalRXBytes:  totalRXBytes,
		TotalTXBytes:  totalTXBytes,
	}, nil
}

func readCPUStatTicks() (uint64, uint64, error) {
	cpuStatFile, openError := os.Open("/proc/stat")
	if openError != nil {
		return 0, 0, openError
	}
	defer cpuStatFile.Close()

	statScanner := bufio.NewScanner(cpuStatFile)
	for statScanner.Scan() {
		statLine := strings.TrimSpace(statScanner.Text())
		if !strings.HasPrefix(statLine, "cpu ") {
			continue
		}

		statFields := strings.Fields(statLine)
		if len(statFields) < 5 {
			return 0, 0, errors.New("formato inválido en /proc/stat")
		}

		totalTicks := uint64(0)
		idleTicks := uint64(0)
		for index := 1; index < len(statFields); index++ {
			parsedTicks, parseError := strconv.ParseUint(statFields[index], 10, 64)
			if parseError != nil {
				return 0, 0, parseError
			}
			totalTicks += parsedTicks
			if index == 4 {
				idleTicks = parsedTicks
			}
		}
		return totalTicks, idleTicks, nil
	}

	if scanError := statScanner.Err(); scanError != nil {
		return 0, 0, scanError
	}
	return 0, 0, errors.New("no se encontró línea cpu en /proc/stat")
}

func readMemInfoValues() (map[string]uint64, error) {
	memInfoFile, openError := os.Open("/proc/meminfo")
	if openError != nil {
		return nil, openError
	}
	defer memInfoFile.Close()

	memInfoValues := map[string]uint64{}
	memInfoScanner := bufio.NewScanner(memInfoFile)
	for memInfoScanner.Scan() {
		memInfoLine := strings.TrimSpace(memInfoScanner.Text())
		if len(memInfoLine) == 0 {
			continue
		}

		memInfoParts := strings.SplitN(memInfoLine, ":", 2)
		if len(memInfoParts) != 2 {
			continue
		}

		metricName := strings.TrimSpace(memInfoParts[0])
		metricValueFields := strings.Fields(strings.TrimSpace(memInfoParts[1]))
		if len(metricValueFields) == 0 {
			continue
		}

		metricValue, parseError := strconv.ParseUint(metricValueFields[0], 10, 64)
		if parseError != nil {
			continue
		}
		memInfoValues[metricName] = metricValue
	}

	if scanError := memInfoScanner.Err(); scanError != nil {
		return nil, scanError
	}

	if _, foundTotal := memInfoValues["MemTotal"]; !foundTotal {
		return nil, errors.New("no se encontró MemTotal")
	}
	if _, foundAvailable := memInfoValues["MemAvailable"]; !foundAvailable {
		return nil, errors.New("no se encontró MemAvailable")
	}
	return memInfoValues, nil
}

func readNetworkCounters(requestedInterfaceName string) (string, uint64, uint64, error) {
	netDevFile, openError := os.Open("/proc/net/dev")
	if openError != nil {
		return "", 0, 0, openError
	}
	defer netDevFile.Close()

	cleanRequestedInterfaceName := strings.TrimSpace(requestedInterfaceName)
	interfaceScanner := bufio.NewScanner(netDevFile)

	for interfaceScanner.Scan() {
		interfaceLine := strings.TrimSpace(interfaceScanner.Text())
		if len(interfaceLine) == 0 || !strings.Contains(interfaceLine, ":") {
			continue
		}

		interfaceParts := strings.SplitN(interfaceLine, ":", 2)
		if len(interfaceParts) != 2 {
			continue
		}

		parsedInterfaceName := strings.TrimSpace(interfaceParts[0])
		if parsedInterfaceName == "lo" {
			continue
		}
		if len(cleanRequestedInterfaceName) > 0 && cleanRequestedInterfaceName != parsedInterfaceName {
			continue
		}

		counterFields := strings.Fields(strings.TrimSpace(interfaceParts[1]))
		if len(counterFields) < 16 {
			continue
		}

		totalRXBytes, rxParseError := strconv.ParseUint(counterFields[0], 10, 64)
		if rxParseError != nil {
			return "", 0, 0, rxParseError
		}
		totalTXBytes, txParseError := strconv.ParseUint(counterFields[8], 10, 64)
		if txParseError != nil {
			return "", 0, 0, txParseError
		}
		return parsedInterfaceName, totalRXBytes, totalTXBytes, nil
	}

	if scanError := interfaceScanner.Err(); scanError != nil {
		return "", 0, 0, scanError
	}
	if len(cleanRequestedInterfaceName) > 0 {
		return "", 0, 0, errors.New("no se encontró la interfaz solicitada: " + cleanRequestedInterfaceName)
	}
	return "", 0, 0, errors.New("no se encontró una interfaz de red activa")
}
