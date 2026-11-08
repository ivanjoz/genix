package core

import (
	"bufio"
	"context"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// Admission control for the VPS server: how many requests run at once, and how much memory the Go
// heap may grow to before the GC fights back.
//
// Without it a burst reaches the OOM killer instead of a queue. A first sync of a 50k catalog holds
// ~110 MB of live heap while it runs, so a few dozen of them at once exhaust an 8 GB box that also
// hosts the database; an incremental sync holds a few KB but saturates the CPU at ~16 per core, past
// which more concurrency only lengthens the queue inside the runtime. Lambda needs none of this:
// one invocation runs per execution environment.

const (
	// requestsPerProcessor is the default slot count per GOMAXPROCS. The benchmark saturated a
	// 4-core server's CPU at 64 concurrent incremental syncs; more slots added latency, not
	// throughput.
	requestsPerProcessor = 16
	// admissionWait is how long a request queues for a slot before it is answered 503. Long
	// enough to ride out a burst, short enough that a client retrying is better served.
	admissionWait = 4 * time.Second
	// memoryLimitShare is the part of the memory found at startup that the Go heap may use. The
	// rest is headroom: the soft limit is not a ceiling on RSS (stacks, buffers outside the heap).
	memoryLimitShare = 0.7
)

var admissionSlots chan struct{}

// ConfigureServerAdmission sizes the request slots and the Go memory limit. Called once at VPS
// startup, before the listener opens.
func ConfigureServerAdmission() {
	maxConcurrentRequests := int(Env.SERVER_MAX_CONCURRENT_REQUESTS)
	if maxConcurrentRequests <= 0 {
		maxConcurrentRequests = requestsPerProcessor * runtime.GOMAXPROCS(0)
	}
	admissionSlots = make(chan struct{}, maxConcurrentRequests)

	// An explicit GOMEMLIMIT in the environment is the operator's own decision; the runtime has
	// already applied it.
	memoryLimitBytes := int64(Env.SERVER_MEMORY_LIMIT_MB) << 20
	if os.Getenv("GOMEMLIMIT") == "" {
		if memoryLimitBytes <= 0 {
			memoryLimitBytes = int64(float64(startupMemoryBudgetBytes()) * memoryLimitShare)
		}
		if memoryLimitBytes > 0 {
			debug.SetMemoryLimit(memoryLimitBytes)
		}
	}
	Log("*admission:: max_concurrent_requests", maxConcurrentRequests,
		"memory_limit_mb", debug.SetMemoryLimit(-1)>>20)
}

// AdmitRequest takes a request slot, waiting up to admissionWait or until the client goes away.
// It returns the release func, or false when the server is saturated and the request must be
// refused. With admission never configured (tests, exec runs) every request passes.
func AdmitRequest(ctx context.Context) (func(), bool) {
	if admissionSlots == nil {
		return func() {}, true
	}
	select {
	case admissionSlots <- struct{}{}:
		return func() { <-admissionSlots }, true
	default:
	}

	waitTimer := time.NewTimer(admissionWait)
	defer waitTimer.Stop()
	select {
	case admissionSlots <- struct{}{}:
		return func() { <-admissionSlots }, true
	case <-waitTimer.C:
		return nil, false
	case <-ctx.Done():
		return nil, false
	}
}

// startupMemoryBudgetBytes is what this process can count on: MemAvailable, not MemTotal, because
// a database on the same host reserves its memory (a buffer pool) when it starts, and the cgroup limit when
// the service has one. Zero when neither can be read, which leaves the runtime default (no limit).
func startupMemoryBudgetBytes() int64 {
	budgetBytes := readMemAvailableBytes()
	if cgroupLimitBytes := readCgroupMemoryLimitBytes(); cgroupLimitBytes > 0 && (budgetBytes == 0 || cgroupLimitBytes < budgetBytes) {
		budgetBytes = cgroupLimitBytes
	}
	return budgetBytes
}

func readMemAvailableBytes() int64 {
	meminfo, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer meminfo.Close()
	scanner := bufio.NewScanner(meminfo)
	for scanner.Scan() {
		// "MemAvailable:    5123456 kB"
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "MemAvailable:" {
			kilobytes, _ := strconv.ParseInt(fields[1], 10, 64)
			return kilobytes << 10
		}
	}
	return 0
}

// readCgroupMemoryLimitBytes reads the cgroup v2 limit of this process ("0::/system.slice/genix.service"
// in /proc/self/cgroup names its directory); "max" or anything unreadable means none.
func readCgroupMemoryLimitBytes() int64 {
	cgroupMembership, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return 0
	}
	cgroupPath, isCgroupV2 := strings.CutPrefix(strings.TrimSpace(string(cgroupMembership)), "0::")
	if !isCgroupV2 {
		return 0
	}
	content, err := os.ReadFile("/sys/fs/cgroup" + cgroupPath + "/memory.max")
	if err != nil {
		return 0
	}
	limitBytes, err := strconv.ParseInt(strings.TrimSpace(string(content)), 10, 64)
	if err != nil {
		return 0
	}
	return limitBytes
}
