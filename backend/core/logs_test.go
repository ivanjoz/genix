package core

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The prefix is the whole point of the CloudWatch half: these are the tokens a filter matches on,
// so their exact spelling is the contract with whoever is reading the log group.
func TestMakeLogPrefixTokens(t *testing.T) {
	previousEnv := Env
	Env = &EnvStruct{IS_SERVERLESS: true}
	atomic.StoreInt32(LogCounter, 0)
	t.Cleanup(func() { Env = previousEnv; atomic.StoreInt32(LogCounter, 0) })

	SetLogRequest(1767225600123456789, 118)
	SetLogUser(7, 42)

	prefix := makeLogPrefix()
	for _, token := range []string{"#1767225600123456789#", "#c7#", "#u7_42#", "#r118#|"} {
		if !strings.Contains(prefix, token) {
			t.Errorf("prefix %q is missing %q", prefix, token)
		}
	}

	// The counter distinguishes lines within one request, so it has to advance.
	if second := makeLogPrefix(); second == prefix {
		t.Fatalf("two lines of the same request got an identical prefix: %q", prefix)
	}
}

// A request rejected at the token check has a route and no user. Carrying the previous caller's
// company into those lines would attribute one tenant's failures to another.
func TestSetLogRequestClearsThePreviousIdentity(t *testing.T) {
	previousEnv := Env
	Env = &EnvStruct{IS_SERVERLESS: true}
	t.Cleanup(func() { Env = previousEnv })

	SetLogRequest(1, 10)
	SetLogUser(7, 42)
	SetLogRequest(2, 11)

	if LogCompanyID != 0 || LogUserID != 0 {
		t.Fatalf("company %d user %d survived into the next request", LogCompanyID, LogUserID)
	}
}

// On the VPS these globals are never read, so writing them from many goroutines at once would be
// a data race for no benefit. Run this with -race; it is the whole assertion.
func TestSetLogIdentityIsInertOffLambda(t *testing.T) {
	previousEnv := Env
	Env = &EnvStruct{IS_SERVERLESS: false}
	LogCompanyID, LogUserID, LogRouteID = 0, 0, 0
	t.Cleanup(func() { Env = previousEnv })

	waitGroup := sync.WaitGroup{}
	for worker := range 8 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			SetLogRequest(int64(worker), int16(worker))
			SetLogUser(int32(worker), int32(worker))
		}()
	}
	waitGroup.Wait()

	if Env.REQUEST_ID != 0 || LogCompanyID != 0 || LogRouteID != 0 {
		t.Fatal("the setters wrote package state while off Lambda")
	}
}

// Within one process the counter is what makes collision impossible, in both modes — the salt only
// separates one Lambda environment from another. 500 draws stays under the serverless mode's
// 1024-per-millisecond counter width, which is the real guarantee being asserted here.
func TestMakeRequestIDIsUniqueWithinAProcess(t *testing.T) {
	for _, serverless := range []bool{true, false} {
		previousEnv := Env
		Env = &EnvStruct{IS_SERVERLESS: serverless}

		seen := map[int64]bool{}
		for range 500 {
			id := MakeRequestID()
			if id <= 0 {
				t.Fatalf("serverless=%v: MakeRequestID returned %d; the row key must stay positive",
					serverless, id)
			}
			if seen[id] {
				t.Fatalf("serverless=%v: MakeRequestID repeated %d", serverless, id)
			}
			seen[id] = true
		}
		Env = previousEnv
	}
}

// The timestamp in the high digits is what puts the clustering key of user_logs in write order,
// and it is meant to be recoverable by dividing the ID back down. If the tail ever outgrew its
// budget it would start eating into the time and both properties would go at once.
func TestMakeRequestIDCarriesARecoverableTimestamp(t *testing.T) {
	for _, serverless := range []bool{true, false} {
		previousEnv := Env
		Env = &EnvStruct{IS_SERVERLESS: serverless}

		span := int64(vpsRequestsPerTick)
		if serverless {
			span = serverlessTailSpan
		}

		before := SUnixTime()
		earlier := MakeRequestID()
		// Longer than one SUnixTime tick, which is two seconds.
		time.Sleep(2100 * time.Millisecond)
		later := MakeRequestID()
		after := SUnixTime()

		if later <= earlier {
			t.Fatalf("serverless=%v: a later request did not sort after an earlier one (%d vs %d)",
				serverless, later, earlier)
		}
		if later/span <= earlier/span {
			t.Fatalf("serverless=%v: the time digits did not advance (%d vs %d)",
				serverless, earlier/span, later/span)
		}

		// The VPS form divides back to exactly the SUnixTime it was minted in.
		if !serverless {
			if tick := int32(earlier / span); tick < before || tick > after {
				t.Fatalf("the ID decoded to SUnixTime %d, outside the [%d, %d] it was minted in",
					tick, before, after)
			}
		}
		Env = previousEnv
	}
}

// The first request of a tick starts that tick's range, so an ID reads as "this instant, request N
// of it" rather than continuing from whatever the process had reached.
func TestVPSRequestIDRestartsEachTick(t *testing.T) {
	previousEnv := Env
	Env = &EnvStruct{IS_SERVERLESS: false}
	t.Cleanup(func() { Env = previousEnv })

	for range 5 {
		MakeRequestID()
	}
	time.Sleep(2100 * time.Millisecond)

	first := MakeRequestID()
	if counter := first % vpsRequestsPerTick; counter != 0 {
		t.Fatalf("the first ID of a new tick had counter %d, expected the tick base", counter)
	}
	if second := MakeRequestID(); second != first+1 {
		t.Fatalf("the second ID of a tick was %d, expected %d", second, first+1)
	}
}

// The VPS counter is shared by every goroutine of one process, which is the only thing keeping two
// requests that start in the same second apart.
func TestMakeRequestIDIsUniqueAcrossGoroutines(t *testing.T) {
	previousEnv := Env
	Env = &EnvStruct{IS_SERVERLESS: false}
	t.Cleanup(func() { Env = previousEnv })

	const workers, perWorker = 8, 500
	minted := make(chan int64, workers*perWorker)
	waitGroup := sync.WaitGroup{}
	for range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for range perWorker {
				minted <- MakeRequestID()
			}
		}()
	}
	waitGroup.Wait()
	close(minted)

	seen := map[int64]bool{}
	for id := range minted {
		if seen[id] {
			t.Fatalf("two goroutines minted the same request ID: %d", id)
		}
		seen[id] = true
	}
}
