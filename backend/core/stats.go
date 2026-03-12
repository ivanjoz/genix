package core

import (
	"sync"
	types "app/core/types"
)

type UsageCounter struct {
	// Bandwidth is stored in 4 KB increments.
	GetBandwith int32
	// Bandwidth is stored in 4 KB increments.
	PostBandwith int32
	// CPU time is stored in 4 ms increments.
	GetCpuTimeUsage int32
	// CPU time is stored in 4 ms increments.
	PostCpuTimeUsage int32
}

type CompanyUsageBucket struct {
	Mu           sync.Mutex
	UsageLogID   int32
	IsDirty      bool
	UserUsageMap map[int32]*UsageCounter
}

// UsageCompanyUserMap stores company usage buckets as:
// companyID -> *CompanyUsageBucket
var UsageCompanyUserMap = map[int32]*CompanyUsageBucket{}
var usageCompanyUserMapMu sync.Mutex
var pendingUsageLogsMu sync.Mutex
var pendingUsageLogsByKey = map[string]types.UsageLog{}
var usageLastSaved int32 = 0

const usageLogWindowTicks int32 = 150 // 5 minutes with 2-second SUnixTime ticks

// requestType = 1: GET | requestType = 2: POST
// bandwith uses 4 KB increments and usageTime uses 4 ms increments.
func AddRequestUsage(companyID, usuarioID, bandwith, usageTime int32, requestType int8) {
	companyUsageBucket := getOrCreateCompanyUsageBucket(companyID)
	currentUsageLogID := GetCurrentUsageLogID()

	companyUsageBucket.Mu.Lock()
	defer companyUsageBucket.Mu.Unlock()

	if companyUsageBucket.UsageLogID == 0 {
		companyUsageBucket.UsageLogID = currentUsageLogID
	}
	if companyUsageBucket.UsageLogID != currentUsageLogID {
		if usageLog, hasData := buildUsageLog(companyID, companyUsageBucket.UsageLogID, companyUsageBucket.UserUsageMap); hasData {
			storePendingUsageLog(usageLog)
		}
		companyUsageBucket.UsageLogID = currentUsageLogID
		companyUsageBucket.UserUsageMap = map[int32]*UsageCounter{}
		companyUsageBucket.IsDirty = false
	}

	updateUsageCounter := func(targetUsuarioID int32) {
		usageCounter := companyUsageBucket.UserUsageMap[targetUsuarioID]
		if usageCounter == nil {
			usageCounter = &UsageCounter{}
			companyUsageBucket.UserUsageMap[targetUsuarioID] = usageCounter
		}

		if requestType == 1 {
			usageCounter.GetBandwith += bandwith
			usageCounter.GetCpuTimeUsage += usageTime
		} else if requestType == 2 {
			usageCounter.PostBandwith += bandwith
			usageCounter.PostCpuTimeUsage += usageTime
		}
	}

	// Update both the specific user and the company aggregate entry (`-1`).
	updateUsageCounter(usuarioID)
	updateUsageCounter(-1)
	companyUsageBucket.IsDirty = true
}

func getOrCreateCompanyUsageBucket(companyID int32) *CompanyUsageBucket {
	usageCompanyUserMapMu.Lock()
	companyUsageBucket := UsageCompanyUserMap[companyID]
	if companyUsageBucket == nil {
		companyUsageBucket = &CompanyUsageBucket{
			UserUsageMap: map[int32]*UsageCounter{},
		}
		UsageCompanyUserMap[companyID] = companyUsageBucket
	}
	usageCompanyUserMapMu.Unlock()
	return companyUsageBucket
}

func TakeUsageLogs() []types.UsageLog {
	usageLogs := takePendingUsageLogs()

	usageCompanyUserMapMu.Lock()
	companyBucketsByID := make(map[int32]*CompanyUsageBucket, len(UsageCompanyUserMap))
	for companyID, companyUsageBucket := range UsageCompanyUserMap {
		companyBucketsByID[companyID] = companyUsageBucket
	}
	usageCompanyUserMapMu.Unlock()

	for companyID, companyUsageBucket := range companyBucketsByID {
		companyUsageBucket.Mu.Lock()
		if !companyUsageBucket.IsDirty || companyUsageBucket.UsageLogID == 0 || len(companyUsageBucket.UserUsageMap) == 0 {
			companyUsageBucket.Mu.Unlock()
			continue
		}

		usageLog, hasData := buildUsageLog(companyID, companyUsageBucket.UsageLogID, companyUsageBucket.UserUsageMap)
		companyUsageBucket.IsDirty = false
		companyUsageBucket.Mu.Unlock()
		if hasData {
			usageLogs = append(usageLogs, usageLog)
		}
	}

	return usageLogs
}

func RestoreUsageLogs(usageLogs []types.UsageLog) {
	for _, usageLog := range usageLogs {
		companyUsageBucket := getOrCreateCompanyUsageBucket(usageLog.CompanyID)
		companyUsageBucket.Mu.Lock()
		if companyUsageBucket.UsageLogID == usageLog.ID {
			companyUsageBucket.IsDirty = true
			companyUsageBucket.Mu.Unlock()
			continue
		}
		companyUsageBucket.Mu.Unlock()

		storePendingUsageLog(usageLog)
	}
}

func SetUsageLastSaved(savedAt int32) {
	usageCompanyUserMapMu.Lock()
	usageLastSaved = savedAt
	usageCompanyUserMapMu.Unlock()
}

func GetCurrentUsageLogID() int32 {
	nowTime := SUnixTime() // SUnixTime uses 2-second ticks.
	return nowTime - (nowTime % usageLogWindowTicks)
}

func buildUsageLog(companyID, usageLogID int32, usageMap map[int32]*UsageCounter) (types.UsageLog, bool) {
	usageLog := types.UsageLog{
		CompanyID:              companyID,
		ID:                     usageLogID,
		DetailUserID:           make([]int32, 0, len(usageMap)),
		DetailGetBandwith:      make([]int32, 0, len(usageMap)),
		DetailPostBandwith:     make([]int32, 0, len(usageMap)),
		DetailGetCpuTimeUsage:  make([]int32, 0, len(usageMap)),
		DetailPostCpuTimeUsage: make([]int32, 0, len(usageMap)),
	}

	totalUsageCounter := usageMap[-1]
	if totalUsageCounter != nil {
		usageLog.GetBandwith = totalUsageCounter.GetBandwith
		usageLog.PostBandwith = totalUsageCounter.PostBandwith
		usageLog.GetCpuTimeUsage = totalUsageCounter.GetCpuTimeUsage
		usageLog.PostCpuTimeUsage = totalUsageCounter.PostCpuTimeUsage
	}

	for usuarioID, usageCounter := range usageMap {
		if usuarioID == -1 || usageCounter == nil {
			continue
		}
		usageLog.DetailUserID = append(usageLog.DetailUserID, usuarioID)
		usageLog.DetailGetBandwith = append(usageLog.DetailGetBandwith, usageCounter.GetBandwith)
		usageLog.DetailPostBandwith = append(usageLog.DetailPostBandwith, usageCounter.PostBandwith)
		usageLog.DetailGetCpuTimeUsage = append(usageLog.DetailGetCpuTimeUsage, usageCounter.GetCpuTimeUsage)
		usageLog.DetailPostCpuTimeUsage = append(usageLog.DetailPostCpuTimeUsage, usageCounter.PostCpuTimeUsage)
	}

	hasData := usageLog.GetBandwith != 0 ||
		usageLog.PostBandwith != 0 ||
		usageLog.GetCpuTimeUsage != 0 ||
		usageLog.PostCpuTimeUsage != 0 ||
		len(usageLog.DetailUserID) > 0
	return usageLog, hasData
}

func storePendingUsageLog(usageLog types.UsageLog) {
	pendingUsageLogsMu.Lock()
	pendingUsageLogsByKey[makeUsageLogKey(usageLog.CompanyID, usageLog.ID)] = usageLog
	pendingUsageLogsMu.Unlock()
}

func takePendingUsageLogs() []types.UsageLog {
	pendingUsageLogsMu.Lock()
	defer pendingUsageLogsMu.Unlock()

	usageLogs := make([]types.UsageLog, 0, len(pendingUsageLogsByKey))
	for usageLogKey, usageLog := range pendingUsageLogsByKey {
		usageLogs = append(usageLogs, usageLog)
		delete(pendingUsageLogsByKey, usageLogKey)
	}
	return usageLogs
}

func makeUsageLogKey(companyID, usageLogID int32) string {
	return Concat("-", companyID, usageLogID)
}
