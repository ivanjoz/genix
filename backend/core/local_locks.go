package core

import (
	"context"
	"sync"
	"time"

	fareward "github.com/ivanjoz/fareward/go"
)

// In-process locks for the VPS server.
//
// fareward's lock service exists for Lambda, where concurrent requests run in separate execution
// environments and only a daemon can serialize them. The VPS backend is one process serving every
// request, so the same mutual exclusion is a mutex away — without a TCP round trip per acquire and
// release, and without a daemon outage turning into 503s. The semantics the call sites rely on are
// kept: a queue ceiling (MaxWaiters, past it the caller is refused as busy), a bounded wait, and a
// lease that frees the key if a holder never releases it.
//
// The scope is this process. Anything else that takes the same locks on the same database — an
// `fn-*` exec run, another backend instance — is not serialized against it.

type localLockKey struct {
	action     uint16
	identifier int64
}

// localLockState is one key. token holds a value while the key is free; generation identifies the
// current hold, so a lease that fires after its holder released (or a release after the lease)
// does nothing.
type localLockState struct {
	token      chan struct{}
	waiters    int
	generation uint64
}

var (
	localLocksMu sync.Mutex
	localLocks   = map[localLockKey]*localLockState{}
)

// localLock is a held in-process lock. Release is idempotent and safe to defer.
type localLock struct {
	releaseOnce sync.Once
	release     func()
}

func (lock *localLock) Release() { lock.releaseOnce.Do(lock.release) }

// acquireLocalLock mirrors fareward.AcquireLockWithOptions: fareward.ErrLockBusy when the queue is
// full or the wait runs out, which is what LockError.Busy and the data-frame caller test for.
func acquireLocalLock(ctx context.Context, action uint16, identifier int64, options LockOptions) (Lock, error) {
	key := localLockKey{action: action, identifier: identifier}

	localLocksMu.Lock()
	state := localLocks[key]
	if state == nil {
		state = &localLockState{token: make(chan struct{}, 1)}
		state.token <- struct{}{}
		localLocks[key] = state
	}
	select {
	case <-state.token:
		lock := holdLocalLock(key, state, options.Lease)
		localLocksMu.Unlock()
		return lock, nil
	default:
	}
	if state.waiters >= int(options.MaxWaiters) {
		localLocksMu.Unlock()
		return nil, fareward.ErrLockBusy
	}
	state.waiters++
	localLocksMu.Unlock()

	waitTimer := time.NewTimer(options.Wait)
	defer waitTimer.Stop()
	var waitErr error
	select {
	case <-state.token:
	case <-waitTimer.C:
		waitErr = fareward.ErrLockBusy
	case <-ctx.Done():
		waitErr = ctx.Err()
	}

	localLocksMu.Lock()
	defer localLocksMu.Unlock()
	state.waiters--
	if waitErr != nil {
		// A release may have handed the token over just as this waiter gave up. With nobody left
		// waiting it would sit in a key nobody holds, so the key is dropped instead.
		if state.waiters == 0 {
			select {
			case <-state.token:
				delete(localLocks, key)
			default:
			}
		}
		return nil, waitErr
	}
	return holdLocalLock(key, state, options.Lease), nil
}

// holdLocalLock records a new hold on a key whose token the caller just took. Called with
// localLocksMu held.
func holdLocalLock(key localLockKey, state *localLockState, lease time.Duration) Lock {
	state.generation++
	heldGeneration := state.generation

	releaseHold := func() {
		localLocksMu.Lock()
		defer localLocksMu.Unlock()
		if state.generation != heldGeneration || localLocks[key] != state {
			return // the lease already freed this hold
		}
		state.generation++
		if state.waiters > 0 {
			state.token <- struct{}{} // hand the key to the next waiter
			return
		}
		delete(localLocks, key)
	}
	leaseTimer := time.AfterFunc(lease, func() {
		Log("lock lease elapsed, releasing it:: action", key.action, "identifier", key.identifier, "warn")
		releaseHold()
	})
	return &localLock{release: func() {
		leaseTimer.Stop()
		releaseHold()
	}}
}
