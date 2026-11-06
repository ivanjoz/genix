package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	fareward "github.com/ivanjoz/fareward/go"
)

func testLockOptions(maxWaiters uint8, wait, lease time.Duration) LockOptions {
	return LockOptions{MaxWaiters: maxWaiters, Wait: wait, Lease: lease}
}

// The point of the lock: never two holders of one key at once, however many goroutines contend.
func TestLocalLockIsMutuallyExclusive(t *testing.T) {
	var holders, maxHolders atomic.Int32
	var contenders sync.WaitGroup
	for range 50 {
		contenders.Add(1)
		go func() {
			defer contenders.Done()
			lock, err := acquireLocalLock(context.Background(), 900, 1, testLockOptions(255, 5*time.Second, 5*time.Second))
			if err != nil {
				t.Errorf("acquire failed: %v", err)
				return
			}
			current := holders.Add(1)
			for {
				seen := maxHolders.Load()
				if current <= seen || maxHolders.CompareAndSwap(seen, current) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			holders.Add(-1)
			lock.Release()
		}()
	}
	contenders.Wait()
	if maxHolders.Load() != 1 {
		t.Fatalf("saw %d holders at once", maxHolders.Load())
	}
	if _, isLeft := localLocks[localLockKey{action: 900, identifier: 1}]; isLeft {
		t.Fatal("a released key with no waiters stayed in the map")
	}
}

// MaxWaiters 0 is a try-lock, and a full queue refuses at once: the abuse pattern the call sites
// (sign-up and contact by IP) rely on being cut off fast.
func TestLocalLockRefusesPastTheQueueCeiling(t *testing.T) {
	held, err := acquireLocalLock(context.Background(), 901, 1, testLockOptions(0, time.Second, 5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()

	startedAt := time.Now()
	if _, err := acquireLocalLock(context.Background(), 901, 1, testLockOptions(0, time.Second, 5*time.Second)); !errors.Is(err, fareward.ErrLockBusy) {
		t.Fatalf("a try-lock on a held key answered %v, not busy", err)
	}
	if time.Since(startedAt) > 100*time.Millisecond {
		t.Fatal("the try-lock waited instead of refusing at once")
	}
}

func TestLocalLockWaitRunsOutAsBusy(t *testing.T) {
	held, _ := acquireLocalLock(context.Background(), 902, 1, testLockOptions(1, time.Second, 5*time.Second))
	defer held.Release()

	_, err := acquireLocalLock(context.Background(), 902, 1, testLockOptions(1, 50*time.Millisecond, 5*time.Second))
	if !errors.Is(err, fareward.ErrLockBusy) {
		t.Fatalf("a waiter whose wait ran out answered %v, not busy", err)
	}
}

func TestLocalLockHandsTheKeyToTheWaiter(t *testing.T) {
	held, _ := acquireLocalLock(context.Background(), 903, 1, testLockOptions(1, time.Second, 5*time.Second))
	acquired := make(chan error, 1)
	go func() {
		lock, err := acquireLocalLock(context.Background(), 903, 1, testLockOptions(1, time.Second, 5*time.Second))
		if err == nil {
			defer lock.Release()
		}
		acquired <- err
	}()
	time.Sleep(20 * time.Millisecond)
	held.Release()
	held.Release() // idempotent: must not hand the key out twice
	if err := <-acquired; err != nil {
		t.Fatalf("the waiter did not get the released key: %v", err)
	}
}

// A holder that never releases must not keep the key forever.
func TestLocalLockLeaseFreesAnAbandonedHold(t *testing.T) {
	if _, err := acquireLocalLock(context.Background(), 904, 1, testLockOptions(1, time.Second, 30*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	lock, err := acquireLocalLock(context.Background(), 904, 1, testLockOptions(1, time.Second, 5*time.Second))
	if err != nil {
		t.Fatalf("the lease did not free the key: %v", err)
	}
	lock.Release()
}
