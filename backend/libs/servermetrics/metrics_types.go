package servermetrics

type CPUMetrics struct {
	PercentUsed float64 `json:"percent_used" cb:"percent_used,1"`
}

type MemoryMetrics struct {
	TotalBytes     uint64  `json:"total_bytes" cb:"total_bytes,1"`
	UsedBytes      uint64  `json:"used_bytes" cb:"used_bytes,2"`
	AvailableBytes uint64  `json:"available_bytes" cb:"available_bytes,3"`
	PercentUsed    float64 `json:"percent_used" cb:"percent_used,4"`
}

type BackendProcessMetrics struct {
	RSSBytes uint64 `json:"rss_bytes" cb:"rss_bytes,1"`
}

type DiskMetrics struct {
	MountPath   string  `json:"mount_path" cb:"mount_path,1"`
	TotalBytes  uint64  `json:"total_bytes" cb:"total_bytes,2"`
	UsedBytes   uint64  `json:"used_bytes" cb:"used_bytes,3"`
	FreeBytes   uint64  `json:"free_bytes" cb:"free_bytes,4"`
	PercentUsed float64 `json:"percent_used" cb:"percent_used,5"`
}

type ConnectionsMetrics struct {
	HTTPActive int64 `json:"http_active" cb:"http_active,1"`
}

type BandwidthMetrics struct {
	InterfaceName string `json:"interface_name" cb:"interface_name,1"`
	RXBytesPerSec uint64 `json:"rx_bytes_per_sec" cb:"rx_bytes_per_sec,2"`
	TXBytesPerSec uint64 `json:"tx_bytes_per_sec" cb:"tx_bytes_per_sec,3"`
	TotalRXBytes  uint64 `json:"total_rx_bytes" cb:"total_rx_bytes,4"`
	TotalTXBytes  uint64 `json:"total_tx_bytes" cb:"total_tx_bytes,5"`
}

type ServerMetricsSnapshot struct {
	TimestampUnix  int64                 `json:"timestamp_unix" cb:"timestamp_unix,1"`
	CPU            CPUMetrics            `json:"cpu" cb:"cpu,2"`
	Memory         MemoryMetrics         `json:"memory" cb:"memory,3"`
	BackendProcess BackendProcessMetrics `json:"backend_process" cb:"backend_process,4"`
	Disk           DiskMetrics           `json:"disk" cb:"disk,5"`
	Connections    ConnectionsMetrics    `json:"connections" cb:"connections,6"`
	Bandwidth      BandwidthMetrics      `json:"bandwidth" cb:"bandwidth,7"`
}
