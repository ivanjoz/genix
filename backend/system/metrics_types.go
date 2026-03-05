package system

type CPUMetrics struct {
	PercentUsed float64 `json:"percent_used"`
}

type MemoryMetrics struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	PercentUsed    float64 `json:"percent_used"`
}

type BackendProcessMetrics struct {
	RSSBytes uint64 `json:"rss_bytes"`
}

type DiskMetrics struct {
	MountPath   string  `json:"mount_path"`
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	PercentUsed float64 `json:"percent_used"`
}

type ConnectionsMetrics struct {
	HTTPActive int64 `json:"http_active"`
}

type BandwidthMetrics struct {
	InterfaceName  string `json:"interface_name"`
	RXBytesPerSec  uint64 `json:"rx_bytes_per_sec"`
	TXBytesPerSec  uint64 `json:"tx_bytes_per_sec"`
	TotalRXBytes   uint64 `json:"total_rx_bytes"`
	TotalTXBytes   uint64 `json:"total_tx_bytes"`
}

type ServerMetricsSnapshot struct {
	TimestampUnix int64              `json:"timestamp_unix"`
	CPU           CPUMetrics         `json:"cpu"`
	Memory        MemoryMetrics      `json:"memory"`
	BackendProcess BackendProcessMetrics `json:"backend_process"`
	Disk          DiskMetrics        `json:"disk"`
	Connections   ConnectionsMetrics `json:"connections"`
	Bandwidth     BandwidthMetrics   `json:"bandwidth"`
}
