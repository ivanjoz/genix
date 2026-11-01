//go:build js

package servermetrics

import "errors"

// A browser tab has no filesystem to stat.
func collectDiskMetrics(mountPath string) (DiskMetrics, error) {
	return DiskMetrics{}, errors.New("servermetrics: disk metrics are not available in the browser")
}
