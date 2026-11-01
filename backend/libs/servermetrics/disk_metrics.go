//go:build !js

package servermetrics

import "syscall"

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
