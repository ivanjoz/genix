package server_utils

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Ephemeral distributed locks against the Rust daemon, over the connection shared with the
// credit limiter.
//
// The problem it solves: concurrent Lambdas serving the same tenant read the same state, all
// conclude the same thing, and all write. Scylla has no transaction to stop them and the ORM has
// no LWT. A lock orders them so each one re-reads what the previous one wrote.
//
// The daemon ties a lock to the connection it was granted on, so a dead socket frees it at once
// — a killed Lambda blocks nobody. The lease we send is the daemon's own deadline on us, its
// backstop for a process that freezes without closing its socket. Both are why a lock cannot be
// released from a different connection than the one that took it.

const (
	lockAcquirePayloadSize = 15
	lockReleasePayloadSize = 12
)

// The action namespace itself lives in core (enums.go), not here: which features need
// serialization is a property of the application, and this package only carries the number to the
// daemon. Everything below treats an action as an opaque uint16.

var (
	// ErrLockBusy is a real answer from the daemon: the key is taken and the queue is full, or
	// our patience ran out. The caller should reject its client.
	ErrLockBusy = errors.New("lock is busy")
	// ErrLockUnavailable means we got no answer at all. Whether that is fatal is the call site's
	// decision, which is why it is a distinct error: registration fails closed on it, most other
	// callers should carry on unlocked.
	ErrLockUnavailable = ErrServerUtilsUnavailable
)

// Daemon reply statuses. Zero is success for every opcode on this port.
const (
	lockReplyOK          = 0
	lockReplyBusy        = 1
	lockReplyWaitTimeout = 2
	lockReplyCapacity    = 3
	lockReplyMisuse      = 4
)

// LockOptions is the full acquire surface, reached through client.Acquire. Handlers do not build
// one: they call AcquireLock, which fills Wait and Lease with the values below.
type LockOptions struct {
	// MaxWaiters is the queue ceiling. Zero makes the call a try-lock. Callers past the ceiling
	// are refused immediately rather than parked, which is what keeps a flood from becoming a
	// denial of service against the daemon itself.
	MaxWaiters uint8
	// Wait is how long we are willing to queue. It must exceed MaxWaiters × the expected hold,
	// or waiters time out before their turn ever arrives.
	Wait time.Duration
	// Lease is the daemon's deadline on us while we hold. It must exceed the critical section,
	// and the daemon clamps it to its own configured ceiling.
	Lease time.Duration
}

// Lock is one held lock. Release is idempotent and safe to defer.
type Lock struct {
	client     *ServerUtilsClient
	connection *muxConnection
	action     uint16
	identifier int64
	generation uint16

	releaseOnce sync.Once
	lostOnce    sync.Once
	lost        chan struct{}
	done        chan struct{}
}

// Lost closes when this lock is no longer ours: the connection died, so the daemon dropped every
// lock on it, or the lease elapsed and the daemon expired the hold on its own clock.
//
// It is advisory, not authoritative. The lease timer here starts when the grant arrives, a round
// trip after the daemon started counting, and under a partition a holder may already be past its
// check by the time this fires. Work inside a lock has to stay safe to run twice regardless;
// this only narrows the window and makes the failure visible instead of silent.
func (lock *Lock) Lost() <-chan struct{} {
	return lock.lost
}

// Release hands the lock back. It must travel on the connection that took it, because that is
// what the daemon tracks; if that connection is already gone, the lock went with it.
func (lock *Lock) Release() {
	lock.releaseOnce.Do(func() {
		close(lock.done)
		if lock.connection.isClosed() {
			return
		}
		payload := makeLockReleasePayload(lock.action, lock.identifier, lock.generation)
		reply, err := lock.connection.exchange(
			context.Background(), lock.client.secret, opcodeLockRelease, payload,
			serverUtilsWriteTimeout, lock.action, lock.identifier)
		if err != nil {
			logLine("lock release failed::", err)
			return
		}
		if reply.status != lockReplyOK {
			// Misuse here means the daemon no longer had this hold — the lease beat us to it.
			logLine("lock release refused, status::", reply.status)
		}
	})
}

// The timings every call site gets. They are not parameters because there is nothing a handler
// knows that would make it pick different ones: both are properties of this daemon and of how long
// a critical section behind it is allowed to run, not of the feature taking the lock.
//
// lockLease has to outlast the longest critical section any caller puts under a lock, or the daemon
// hands the key to the next caller while the first is still working — the exact race the lock
// exists to prevent. Sign-up is the current bound: core.SendEmail's 4s connect + 6s send plus its
// queries. Anything slower than that under a lock needs this raised, or its own lease.
//
// lockWait is deliberately shorter than that hold. Contention here is the abuse pattern, so
// refusing the extras fast is the wanted behavior; a queue that patiently absorbs a flood is doing
// the attacker's work. LockOptions remains for tests, which need to drive the edges.
const (
	lockWait  = 5 * time.Second
	lockLease = 15 * time.Second
)

// AcquireLock blocks until the key is free, the queue refuses us, or lockWait elapses.
//
// maxWaiters is the queue ceiling for this key, and the only knob a call site gets. Zero makes it a
// try-lock; callers arriving past the ceiling are refused immediately instead of parked.
func AcquireLock(
	ctx context.Context, action uint16, identifier int64, maxWaiters uint8,
) (*Lock, error) {
	client := serverUtils()
	if client == nil {
		return nil, fmt.Errorf("%w: not configured", ErrLockUnavailable)
	}
	return client.Acquire(ctx, action, identifier, LockOptions{
		MaxWaiters: maxWaiters,
		Wait:       lockWait,
		Lease:      lockLease,
	})
}

func (client *ServerUtilsClient) Acquire(
	ctx context.Context, action uint16, identifier int64, options LockOptions,
) (*Lock, error) {
	waitMillis, err := lockDurationToMillis(options.Wait, "Wait")
	if err != nil {
		return nil, err
	}
	leaseMillis, err := lockDurationToMillis(options.Lease, "Lease")
	if err != nil {
		return nil, err
	}
	if leaseMillis == 0 {
		return nil, errors.New("LockOptions.Lease must be positive")
	}

	payload := make([]byte, lockAcquirePayloadSize)
	binary.BigEndian.PutUint16(payload[0:2], action)
	binary.BigEndian.PutUint64(payload[2:10], uint64(identifier))
	payload[10] = options.MaxWaiters
	binary.BigEndian.PutUint16(payload[11:13], waitMillis)
	binary.BigEndian.PutUint16(payload[13:15], leaseMillis)

	// The daemon holds the frame for up to Wait before answering, so our patience has to outlast
	// the queue, not the round trip.
	reply, connection, err := client.request(
		ctx, opcodeLockAcquire, payload, options.Wait+3*time.Second, action, identifier)
	if err != nil {
		return nil, err
	}

	switch reply.status {
	case lockReplyOK:
		return newLock(client, connection, action, identifier, reply.detail, options.Lease), nil
	case lockReplyBusy, lockReplyWaitTimeout:
		return nil, ErrLockBusy
	case lockReplyCapacity:
		return nil, fmt.Errorf("%w: daemon at capacity", ErrLockUnavailable)
	default:
		return nil, fmt.Errorf("%w: unexpected reply status %d", ErrLockUnavailable, reply.status)
	}
}

func newLock(
	client *ServerUtilsClient, connection *muxConnection,
	action uint16, identifier int64, generation uint16, lease time.Duration,
) *Lock {
	lock := &Lock{
		client:     client,
		connection: connection,
		action:     action,
		identifier: identifier,
		generation: generation,
		lost:       make(chan struct{}),
		done:       make(chan struct{}),
	}
	// Watches for the two ways this hold can end without us releasing it. Exits as soon as the
	// lock is released normally, so it costs nothing in the common case.
	go func() {
		timer := time.NewTimer(lease)
		defer timer.Stop()
		select {
		case <-lock.done:
			return
		case <-connection.closed:
		case <-timer.C:
		}
		lock.lostOnce.Do(func() { close(lock.lost) })
	}()
	return lock
}

func makeLockReleasePayload(action uint16, identifier int64, generation uint16) []byte {
	payload := make([]byte, lockReleasePayloadSize)
	binary.BigEndian.PutUint16(payload[0:2], action)
	binary.BigEndian.PutUint64(payload[2:10], uint64(identifier))
	binary.BigEndian.PutUint16(payload[10:12], generation)
	return payload
}

// lockDurationToMillis enforces the uint16 milliseconds the wire carries.
func lockDurationToMillis(value time.Duration, name string) (uint16, error) {
	if value < 0 {
		return 0, fmt.Errorf("LockOptions.%s cannot be negative", name)
	}
	millis := value.Milliseconds()
	if millis > int64(^uint16(0)) {
		return 0, fmt.Errorf("LockOptions.%s exceeds the 65535 ms the protocol carries", name)
	}
	return uint16(millis), nil
}
