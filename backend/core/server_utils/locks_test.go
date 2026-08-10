package server_utils

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// muxDaemonStub speaks the real wire protocol so the client can be exercised without the Rust
// daemon: it reads whole frames, records them, and answers with whatever the test scripted.
type muxDaemonStub struct {
	listener net.Listener
	frames   chan []byte

	mu sync.Mutex
	// answer decides the reply for one request. Returning ok=false withholds the reply entirely,
	// which is how a queued acquire is simulated.
	answer func(sequence uint64, opcode byte, payload []byte) (status byte, detail uint16, ok bool)
	// deferred holds replies the stub chose to withhold, so a test can release them later.
	deferred []deferredReply
	conns    []net.Conn
}

type deferredReply struct {
	connection net.Conn
	sequence   uint64
	status     byte
	detail     uint16
}

func frameSizeFor(opcode byte) int {
	switch opcode {
	case opcodeChargeCredits:
		return 1 + creditChargePayloadSize + serverUtilsAuthTagSize
	case opcodeLockAcquire:
		return 1 + lockAcquirePayloadSize + serverUtilsAuthTagSize
	default:
		return 1 + lockReleasePayloadSize + serverUtilsAuthTagSize
	}
}

func startMuxDaemonStub(t *testing.T) *muxDaemonStub {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stub := &muxDaemonStub{listener: listener, frames: make(chan []byte, 32)}
	stub.answer = func(uint64, byte, []byte) (byte, uint16, bool) { return lockReplyOK, 7, true }
	go stub.serve()
	t.Cleanup(func() { listener.Close() })
	return stub
}

func (stub *muxDaemonStub) serve() {
	for {
		connection, err := stub.listener.Accept()
		if err != nil {
			return
		}
		stub.mu.Lock()
		stub.conns = append(stub.conns, connection)
		stub.mu.Unlock()
		go stub.handle(connection)
	}
}

func (stub *muxDaemonStub) handle(connection net.Conn) {
	defer connection.Close()
	if _, err := connection.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8}); err != nil {
		return
	}
	for sequence := uint64(0); ; sequence++ {
		opcode := []byte{0}
		if _, err := io.ReadFull(connection, opcode); err != nil {
			return
		}
		body := make([]byte, frameSizeFor(opcode[0])-1)
		if _, err := io.ReadFull(connection, body); err != nil {
			return
		}
		stub.frames <- append(opcode, body...)

		stub.mu.Lock()
		answer := stub.answer
		stub.mu.Unlock()
		payload := body[:len(body)-serverUtilsAuthTagSize]
		status, detail, ok := answer(sequence, opcode[0], payload)
		if !ok {
			stub.mu.Lock()
			stub.deferred = append(stub.deferred,
				deferredReply{connection, sequence, lockReplyOK, detail})
			stub.mu.Unlock()
			continue
		}
		if _, err := connection.Write(makeStubReply(sequence, status, detail)); err != nil {
			return
		}
	}
}

// flushDeferred sends the replies the stub withheld earlier.
func (stub *muxDaemonStub) flushDeferred() {
	stub.mu.Lock()
	pending := stub.deferred
	stub.deferred = nil
	stub.mu.Unlock()
	for _, reply := range pending {
		reply.connection.Write(makeStubReply(reply.sequence, reply.status, reply.detail))
	}
}

func (stub *muxDaemonStub) dropConnections() {
	stub.mu.Lock()
	connections := stub.conns
	stub.conns = nil
	stub.mu.Unlock()
	for _, connection := range connections {
		connection.Close()
	}
}

func makeStubReply(sequence uint64, status byte, detail uint16) []byte {
	reply := make([]byte, serverUtilsReplySize)
	binary.BigEndian.PutUint16(reply[0:2], uint16(sequence))
	reply[2] = status
	binary.BigEndian.PutUint16(reply[3:5], detail)
	return reply
}

func (stub *muxDaemonStub) client() *ServerUtilsClient {
	return &ServerUtilsClient{address: stub.listener.Addr().String(), secret: []byte("test-secret")}
}

func TestAcquireAndReleaseFramesMatchTheRustVectors(t *testing.T) {
	stub := startMuxDaemonStub(t)
	client := stub.client()

	lock, err := client.Acquire(context.Background(), 7, -42, LockOptions{
		MaxWaiters: 3,
		Wait:       5000 * time.Millisecond,
		Lease:      15000 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Pinned byte for byte against service/auth.rs.
	expected := []byte{
		0x02, 0x00, 0x07, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xD6, 0x03, 0x13, 0x88,
		0x3A, 0x98, 0x89, 0x9E, 0x18, 0x73, 0xDC, 0xDF, 0x40, 0x29,
	}
	if frame := <-stub.frames; string(frame) != string(expected) {
		t.Fatalf("acquire frame = % X; want % X", frame, expected)
	}

	// The release must carry the key and the generation the daemon handed back (7 here).
	lock.Release()
	release := <-stub.frames
	if release[0] != opcodeLockRelease || len(release) != frameSizeFor(opcodeLockRelease) {
		t.Fatalf("release frame = % X; want a %d-byte 0x03 frame", release, frameSizeFor(opcodeLockRelease))
	}
	if action := binary.BigEndian.Uint16(release[1:3]); action != 7 {
		t.Fatalf("release action = %d; want 7", action)
	}
	if identifier := int64(binary.BigEndian.Uint64(release[3:11])); identifier != -42 {
		t.Fatalf("release identifier = %d; want -42", identifier)
	}
	if generation := binary.BigEndian.Uint16(release[11:13]); generation != 7 {
		t.Fatalf("release generation = %d; want the granted 7", generation)
	}

	// Release is idempotent: deferring it next to an early return is the normal usage.
	lock.Release()
	select {
	case extra := <-stub.frames:
		t.Fatalf("release is not idempotent, it sent % X again", extra)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestRepliesCorrelateToTheRightCallerOutOfOrder(t *testing.T) {
	// The property multiplexing rests on: a request parked in a lock queue must not stop later
	// requests from being answered, and each caller must get its own answer.
	stub := startMuxDaemonStub(t)
	stub.answer = func(sequence uint64, opcode byte, _ []byte) (byte, uint16, bool) {
		if opcode == opcodeLockAcquire {
			return 0, 11, false // withheld, like an acquire sitting in the queue
		}
		return 0, 0, true
	}
	client := stub.client()

	acquireDone := make(chan error, 1)
	go func() {
		_, err := client.Acquire(context.Background(), 1, 500, LockOptions{
			MaxWaiters: 4, Wait: 3 * time.Second, Lease: 15 * time.Second,
		})
		acquireDone <- err
	}()
	<-stub.frames // the acquire reached the stub and is now parked

	// A charge sent afterwards must be answered while the acquire still waits.
	charged := make(chan error, 1)
	go func() { charged <- client.Charge(context.Background(), 1, 1, 0, 2, 0) }()
	select {
	case err := <-charged:
		if err != nil {
			t.Fatalf("charge overtook the acquire but failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the charge was blocked behind the parked acquire")
	}
	select {
	case <-acquireDone:
		t.Fatal("the acquire should still be waiting")
	default:
	}

	stub.flushDeferred()
	if err := <-acquireDone; err != nil {
		t.Fatalf("the parked acquire never got its own reply: %v", err)
	}
}

func TestConnectionDeathFailsPendingCallersAndLosesLocks(t *testing.T) {
	stub := startMuxDaemonStub(t)
	client := stub.client()

	lock, err := client.Acquire(context.Background(), 1, 900, LockOptions{
		MaxWaiters: 1, Wait: time.Second, Lease: 30 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	<-stub.frames

	select {
	case <-lock.Lost():
		t.Fatal("the lock is still held; Lost must not be closed yet")
	default:
	}

	// The daemon drops every lock held on a connection that dies, so the holder has to be told.
	stub.dropConnections()
	select {
	case <-lock.Lost():
	case <-time.After(2 * time.Second):
		t.Fatal("a dead connection must close Lost()")
	}
}

func TestALeaseElapsingClosesLostWithoutAnyFrame(t *testing.T) {
	// The daemon expires a hold on its own clock and has no way to push that to us, so the client
	// arms its own timer. Advisory, but without it a holder would keep believing it holds the key.
	stub := startMuxDaemonStub(t)
	client := stub.client()

	lock, err := client.Acquire(context.Background(), 1, 901, LockOptions{
		MaxWaiters: 1, Wait: time.Second, Lease: 150 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-lock.Lost():
	case <-time.After(2 * time.Second):
		t.Fatal("an elapsed lease must close Lost()")
	}
}

func TestAnAbandonedAcquireGrantedLateIsReleasedAutomatically(t *testing.T) {
	// A caller whose context is cancelled after its frame went out must not leave the key held by
	// nobody until the lease runs out.
	stub := startMuxDaemonStub(t)
	stub.answer = func(_ uint64, opcode byte, _ []byte) (byte, uint16, bool) {
		if opcode == opcodeLockAcquire {
			return 0, 33, false // granted, but only after the caller has given up
		}
		return 0, 0, true
	}
	client := stub.client()

	ctx, cancel := context.WithCancel(context.Background())
	acquireDone := make(chan error, 1)
	go func() {
		_, err := client.Acquire(ctx, 1, 902, LockOptions{
			MaxWaiters: 4, Wait: 5 * time.Second, Lease: 30 * time.Second,
		})
		acquireDone <- err
	}()
	<-stub.frames
	cancel()
	if err := <-acquireDone; err == nil {
		t.Fatal("the cancelled acquire should have returned an error")
	}

	// Now the grant finally lands. The client must hand it straight back.
	stub.flushDeferred()
	select {
	case frame := <-stub.frames:
		if frame[0] != opcodeLockRelease {
			t.Fatalf("expected an automatic release, got opcode %d", frame[0])
		}
		if generation := binary.BigEndian.Uint16(frame[11:13]); generation != 33 {
			t.Fatalf("release generation = %d; want the granted 33", generation)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a lock granted after its caller gave up must be released automatically")
	}
}

func TestConcurrentSendersKeepTheSequenceInLockstep(t *testing.T) {
	// Taking a sequence and writing its frame must be atomic: interleaved writes would
	// desynchronize the HMAC and every later frame would fail authentication.
	stub := startMuxDaemonStub(t)
	client := stub.client()

	var waitGroup sync.WaitGroup
	for range 20 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			_ = client.Charge(context.Background(), 1, 1, 0, 1, 0)
		}()
	}
	waitGroup.Wait()

	// The stub verifies nothing itself; what matters is that it could frame all 20 requests,
	// which is only possible if their bytes never interleaved.
	for range 20 {
		select {
		case frame := <-stub.frames:
			if frame[0] != opcodeChargeCredits {
				t.Fatalf("frame boundaries drifted: leading byte %d", frame[0])
			}
		case <-time.After(2 * time.Second):
			t.Fatal("not every concurrent request reached the daemon intact")
		}
	}
}

func TestAnUnreachableDaemonIsDistinguishableFromBusy(t *testing.T) {
	// A port nobody listens on: the caller must be able to tell "no answer" from "taken", since
	// that is what decides whether it fails open or closed.
	client := &ServerUtilsClient{address: "127.0.0.1:1", secret: []byte("test-secret")}
	_, err := client.Acquire(context.Background(), 1, 5, LockOptions{Lease: time.Second})
	if !errors.Is(err, ErrLockUnavailable) {
		t.Fatalf("err = %v; want ErrLockUnavailable", err)
	}
	if errors.Is(err, ErrLockBusy) {
		t.Fatal("an unreachable daemon must never look like a busy lock")
	}
}

func TestBusyAndTimeoutRepliesBothReportBusy(t *testing.T) {
	for _, status := range []byte{lockReplyBusy, lockReplyWaitTimeout} {
		stub := startMuxDaemonStub(t)
		stub.answer = func(uint64, byte, []byte) (byte, uint16, bool) { return status, 0, true }
		client := stub.client()
		lock, err := client.Acquire(context.Background(), 1, 5, LockOptions{
			MaxWaiters: 1, Wait: 100 * time.Millisecond, Lease: time.Second,
		})
		if !errors.Is(err, ErrLockBusy) {
			t.Fatalf("status %d gave err = %v; want ErrLockBusy", status, err)
		}
		if lock != nil {
			t.Fatal("a refused acquire must not return a lock")
		}
	}
}

func TestLeaseAndWaitMustFitTheWireWidth(t *testing.T) {
	client := &ServerUtilsClient{address: "127.0.0.1:1", secret: []byte("s")}
	if _, err := client.Acquire(context.Background(), 1, 5, LockOptions{Lease: 0}); err == nil {
		t.Fatal("a zero lease must be rejected before dialing")
	}
	_, err := client.Acquire(context.Background(), 1, 5, LockOptions{Lease: 90 * time.Second})
	if err == nil || errors.Is(err, ErrLockUnavailable) {
		t.Fatalf("err = %v; want a validation error about the 65535 ms ceiling", err)
	}
}
