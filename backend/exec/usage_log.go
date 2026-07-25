package exec

import (
	"app/core"
	"github.com/ivanjoz/genix-orm/scylla"
	"time"
)

const usageLogFlushInterval = 20 * time.Second

func StartUsageLogFlushWorker() {
	go func() {
		flushTicker := time.NewTicker(usageLogFlushInterval)
		defer flushTicker.Stop()

		for range flushTicker.C {
			FlushUsageLogs()
		}
	}()
}

func FlushUsageLogs() {
	usageLogsToSave := core.TakeUsageLogs()
	if len(usageLogsToSave) == 0 {
		return
	}

	core.Log("Saving usage logs:", len(usageLogsToSave))
	if insertErr := scylla.Insert(&usageLogsToSave); insertErr != nil {
		core.Log("Error saving usage logs:", insertErr.Error())
		core.RestoreUsageLogs(usageLogsToSave)
		return
	}

	savedAt := core.SUnixTime()
	core.SetUsageLastSaved(savedAt)
	core.Log("Usage logs saved at:", savedAt)
}
