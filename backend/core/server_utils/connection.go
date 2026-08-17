package server_utils

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// One multiplexed TCP connection to the server-utils daemon, shared by every operation in this
// process: credit charges and locks alike.
//
// Requests travel in order and carry a sequence that both sides advance in lockstep for the
// frame HMAC. Replies do not: an acquire can sit in a lock queue for seconds while charges sent
// after it are answered immediately. Each reply therefore echoes the low 16 bits of its
// request's sequence, and a single reader goroutine uses that to hand the answer to the right
// caller. Nothing extra travels on the wire to make this work — the sequence already existed.
//
// The one hard rule: taking a sequence and writing its frame must be atomic. Two goroutines
// taking 5 and 6 but writing 6, 5 would desynchronize the HMAC and every later frame would fail.
// That is what writeMu guards, and it is held for a socket write, never for a round trip.

const (
	serverUtilsNonceSize   = 8
	serverUtilsAuthTagSize = 8
	// Every reply is [correlation:u16][status:u8][detail:u16].
	serverUtilsReplySize = 5
	// Names the framing of the whole port, request and reply, and is bumped on every wire change
	// so a mismatched peer fails at the first frame instead of misreading bytes.
	serverUtilsAuthDomain = "genix-server-utils:v5"

	opcodeChargeCredits = byte(0x01)
	opcodeLockAcquire   = byte(0x02)
	opcodeLockRelease   = byte(0x03)
	// opcodeLogRequest is the only length-prefixed opcode and the only one the daemon does not
	// answer. Both are consequences of what it carries: a variable-length log record that must
	// never make a response wait.
	opcodeLogRequest          = byte(0x04)
	opcodeMutateCompanyBudget = byte(0x05)

	// Frames are tiny and the daemon is on loopback or a private network, so a write that cannot
	// complete in this long means the connection is gone.
	serverUtilsWriteTimeout = 5 * time.Second
	serverUtilsDialTimeout  = 2 * time.Second
)

// logLine is how this package reports anything: it cannot import core, because core needs
// CreditLimitExceeded for its HTTP error mapping and that would be an import cycle. main pushes
// core.Log in at startup, the same way text_search receives its configuration.
var logLine = func(args ...any) {}

// SetLogger installs the process logger. Called once from main, before any request is served.
func SetLogger(logger func(args ...any)) {
	if logger != nil {
		logLine = logger
	}
}

// ErrServerUtilsUnavailable means no answer arrived. Callers distinguish it from a real verdict:
// a charge treats it as permission to proceed, sign-up treats it as a reason to refuse.
var ErrServerUtilsUnavailable = errors.New("server utils service is unavailable")

type muxReply struct {
	status byte
	detail uint16
}

type pendingRequest struct {
	reply chan muxReply
	// abandoned marks a caller that stopped waiting. The entry stays in the map so a late reply
	// can still be handled: an acquire granted after its caller gave up has to be released, or
	// the key stays locked with nobody holding it.
	abandoned  bool
	opcode     byte
	action     uint16
	identifier int64
}

type muxConnection struct {
	conn  net.Conn
	nonce [serverUtilsNonceSize]byte

	writeMu  sync.Mutex
	sequence uint64

	pendingMu sync.Mutex
	pending   map[uint16]*pendingRequest

	closed    chan struct{}
	closeOnce sync.Once
}

type ServerUtilsClient struct {
	address string
	secret  []byte
	mu      sync.Mutex
	current *muxConnection
}

var (
	configuredServerUtilsMu sync.RWMutex
	configuredServerUtils   *ServerUtilsClient
)

// ConfigureServerUtils installs the process-wide client. One address, one secret, one connection
// for both the credit limiter and the lock service — the opcode decides which.
func ConfigureServerUtils(address, secret string) error {
	address = strings.TrimSpace(address)
	if address == "" {
		return errors.New("server_utils is required by the server-utils client")
	}
	if strings.TrimSpace(secret) == "" {
		return errors.New("internal_apikey is required by the server-utils client")
	}
	client := &ServerUtilsClient{address: address, secret: []byte(secret)}
	configuredServerUtilsMu.Lock()
	previous := configuredServerUtils
	configuredServerUtils = client
	configuredServerUtilsMu.Unlock()
	if previous != nil {
		previous.Close()
	}
	return nil
}

func serverUtils() *ServerUtilsClient {
	configuredServerUtilsMu.RLock()
	defer configuredServerUtilsMu.RUnlock()
	return configuredServerUtils
}

// Close drops the current connection, which releases every lock held on it.
func (client *ServerUtilsClient) Close() {
	client.mu.Lock()
	connection := client.current
	client.current = nil
	client.mu.Unlock()
	if connection != nil {
		connection.fail(errors.New("client closed"))
	}
}

// request sends one frame and waits for its reply, retrying once on a connection that turned out
// to be dead. It returns the connection used, because a lock must be released on the same one.
func (client *ServerUtilsClient) request(
	ctx context.Context, opcode byte, payload []byte, wait time.Duration,
	action uint16, identifier int64,
) (muxReply, *muxConnection, error) {
	var lastError error
	for attempt := range 2 {
		connection, reused, err := client.connection(ctx)
		if err != nil {
			return muxReply{}, nil, fmt.Errorf("%w: connect: %v", ErrServerUtilsUnavailable, err)
		}
		reply, err := connection.exchange(
			ctx, client.secret, opcode, payload, wait, action, identifier)
		if err == nil {
			return reply, connection, nil
		}
		lastError = err
		// A pooled connection the daemon closed while idle looks exactly like this. Retrying is
		// safe for an acquire too: ownership is tied to the connection, so anything granted on a
		// dead one was already released with it.
		if reused && attempt == 0 && !errors.Is(err, context.Canceled) &&
			!errors.Is(err, context.DeadlineExceeded) {
			continue
		}
		break
	}
	return muxReply{}, nil, fmt.Errorf("%w: %v", ErrServerUtilsUnavailable, lastError)
}

// requestOnce avoids replaying non-idempotent operations such as increasing a credit balance.
// An ambiguous disconnect is returned to the caller, which must re-read durable state.
func (client *ServerUtilsClient) requestOnce(
	ctx context.Context, opcode byte, payload []byte, wait time.Duration,
) (muxReply, error) {
	connection, _, err := client.connection(ctx)
	if err != nil {
		return muxReply{}, fmt.Errorf("%w: connect: %v", ErrServerUtilsUnavailable, err)
	}
	reply, err := connection.exchange(ctx, client.secret, opcode, payload, wait, 0, 0)
	if err != nil {
		return muxReply{}, fmt.Errorf("%w: %v", ErrServerUtilsUnavailable, err)
	}
	return reply, nil
}

// send writes one frame the daemon will not answer, and returns as soon as the bytes are in the
// socket.
//
// No pending entry is registered, which is the point: the reader would otherwise log every
// unmatched reply, and a caller would be parked waiting for one that never comes. The frame
// sequence still advances under writeMu in lockstep with the daemon's, because that is what the
// HMAC is bound to — a fire-and-forget frame that skipped the sequence would invalidate every
// frame after it on this connection.
//
// One retry, for the same reason a request gets one: a pooled connection the daemon closed while
// idle is indistinguishable from a live one until the write fails.
func (client *ServerUtilsClient) send(ctx context.Context, opcode byte, payload []byte) error {
	var lastError error
	for attempt := range 2 {
		connection, reused, err := client.connection(ctx)
		if err != nil {
			return fmt.Errorf("%w: connect: %v", ErrServerUtilsUnavailable, err)
		}
		if err := connection.write(client.secret, opcode, payload); err == nil {
			return nil
		} else {
			lastError = err
		}
		if reused && attempt == 0 {
			continue
		}
		break
	}
	return fmt.Errorf("%w: %v", ErrServerUtilsUnavailable, lastError)
}

// write builds and writes one length-prefixed frame under the sequence lock.
func (connection *muxConnection) write(secret []byte, opcode byte, payload []byte) error {
	connection.writeMu.Lock()
	if connection.sequence == ^uint64(0) {
		connection.writeMu.Unlock()
		connection.fail(errors.New("frame sequence exhausted"))
		return errors.New("frame sequence exhausted")
	}
	sequence := connection.sequence
	connection.sequence++

	frame := buildServerUtilsLengthPrefixedFrame(
		secret, &connection.nonce, sequence, opcode, payload)
	writeErr := connection.conn.SetWriteDeadline(time.Now().Add(serverUtilsWriteTimeout))
	if writeErr == nil {
		writeErr = writeCompleteFrame(connection.conn, frame)
	}
	connection.writeMu.Unlock()

	if writeErr != nil {
		connection.fail(writeErr)
	}
	return writeErr
}

// connection returns the shared connection, dialing one if none is healthy, and reports whether
// it was already open.
func (client *ServerUtilsClient) connection(ctx context.Context) (*muxConnection, bool, error) {
	// The dial happens under the lock on purpose. Releasing it first lets every concurrent
	// caller open its own socket and then throw all but one away — a burst of six requests on a
	// cold client opened six connections. Waiting behind one dial is what they would have spent
	// anyway, and it is bounded by serverUtilsDialTimeout.
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.current != nil && !client.current.isClosed() {
		return client.current, true, nil
	}

	dialed, err := client.dial(ctx)
	if err != nil {
		return nil, false, err
	}
	client.current = dialed
	go dialed.readLoop(client)
	return dialed, false, nil
}

func (client *ServerUtilsClient) dial(ctx context.Context) (*muxConnection, error) {
	dialer := net.Dialer{Timeout: serverUtilsDialTimeout, KeepAlive: 30 * time.Second}
	socket, err := dialer.DialContext(ctx, "tcp", client.address)
	if err != nil {
		return nil, err
	}
	connection := &muxConnection{
		conn:    socket,
		pending: map[uint16]*pendingRequest{},
		closed:  make(chan struct{}),
	}
	if err := socket.SetReadDeadline(time.Now().Add(serverUtilsDialTimeout)); err != nil {
		socket.Close()
		return nil, err
	}
	if _, err := io.ReadFull(socket, connection.nonce[:]); err != nil {
		socket.Close()
		return nil, fmt.Errorf("read server nonce: %w", err)
	}
	// Clear it again: from here the reader blocks indefinitely and per-request deadlines are
	// enforced by the callers, since one socket now carries many requests with different ones.
	if err := socket.SetReadDeadline(time.Time{}); err != nil {
		socket.Close()
		return nil, err
	}
	return connection, nil
}

func (connection *muxConnection) exchange(
	ctx context.Context, secret []byte, opcode byte, payload []byte, wait time.Duration,
	action uint16, identifier int64,
) (muxReply, error) {
	pending := &pendingRequest{
		// Buffered, so the reader never blocks handing over an answer nobody is waiting for yet.
		reply:      make(chan muxReply, 1),
		opcode:     opcode,
		action:     action,
		identifier: identifier,
	}

	connection.writeMu.Lock()
	if connection.sequence == ^uint64(0) {
		connection.writeMu.Unlock()
		connection.fail(errors.New("frame sequence exhausted"))
		return muxReply{}, errors.New("frame sequence exhausted")
	}
	sequence := connection.sequence
	correlation := uint16(sequence)
	connection.pendingMu.Lock()
	if _, taken := connection.pending[correlation]; taken {
		connection.pendingMu.Unlock()
		connection.writeMu.Unlock()
		return muxReply{}, errors.New("too many requests in flight on one connection")
	}
	connection.pending[correlation] = pending
	connection.pendingMu.Unlock()
	connection.sequence++

	frame := buildServerUtilsFrame(secret, &connection.nonce, sequence, opcode, payload)
	writeErr := connection.conn.SetWriteDeadline(time.Now().Add(serverUtilsWriteTimeout))
	if writeErr == nil {
		writeErr = writeCompleteFrame(connection.conn, frame)
	}
	connection.writeMu.Unlock()

	if writeErr != nil {
		connection.forget(correlation)
		connection.fail(writeErr)
		return muxReply{}, writeErr
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case reply := <-pending.reply:
		return reply, nil
	case <-connection.closed:
		connection.forget(correlation)
		return muxReply{}, errors.New("connection closed while waiting for a reply")
	case <-ctx.Done():
		connection.abandon(correlation)
		return muxReply{}, ctx.Err()
	case <-timer.C:
		connection.abandon(correlation)
		return muxReply{}, errors.New("timed out waiting for a reply")
	}
}

// readLoop is the only reader of this socket. It dispatches by correlation, which is what lets
// several callers share the connection.
func (connection *muxConnection) readLoop(client *ServerUtilsClient) {
	for {
		reply := [serverUtilsReplySize]byte{}
		if _, err := io.ReadFull(connection.conn, reply[:]); err != nil {
			connection.fail(err)
			return
		}
		correlation := binary.BigEndian.Uint16(reply[0:2])
		answer := muxReply{status: reply[2], detail: binary.BigEndian.Uint16(reply[3:5])}

		connection.pendingMu.Lock()
		request, known := connection.pending[correlation]
		if known {
			delete(connection.pending, correlation)
		}
		connection.pendingMu.Unlock()

		if !known {
			// Nobody is waiting for this. Not fatal — the caller may have been abandoned and
			// already cleaned up — but it should never happen in a healthy stream.
			logLine("server utils reply with no matching request::", correlation)
			continue
		}
		if request.abandoned {
			// The caller gave up, but the daemon may still have granted the lock. Hand it back
			// straight away instead of leaving the key held by nobody until its lease expires.
			if request.opcode == opcodeLockAcquire && answer.status == lockReplyOK {
				go client.releaseAbandoned(connection, request.action, request.identifier, answer.detail)
			}
			continue
		}
		request.reply <- answer
	}
}

// releaseAbandoned returns a lock that was granted to a caller which had already stopped waiting.
func (client *ServerUtilsClient) releaseAbandoned(
	connection *muxConnection, action uint16, identifier int64, generation uint16,
) {
	logLine("server utils releasing a lock granted after its caller gave up::", action, identifier)
	payload := makeLockReleasePayload(action, identifier, generation)
	_, err := connection.exchange(
		context.Background(), client.secret, opcodeLockRelease, payload,
		serverUtilsWriteTimeout, action, identifier)
	if err != nil {
		// Not recoverable, and not fatal: the lease is the backstop.
		logLine("server utils could not release an abandoned lock::", err)
	}
}

func (connection *muxConnection) forget(correlation uint16) {
	connection.pendingMu.Lock()
	delete(connection.pending, correlation)
	connection.pendingMu.Unlock()
}

func (connection *muxConnection) abandon(correlation uint16) {
	connection.pendingMu.Lock()
	if request, known := connection.pending[correlation]; known {
		request.abandoned = true
	}
	connection.pendingMu.Unlock()
}

// fail tears the connection down once and wakes everyone waiting on it. A dead socket must never
// leave a goroutine blocked forever, and it also means the daemon dropped every lock held here.
func (connection *muxConnection) fail(cause error) {
	connection.closeOnce.Do(func() {
		connection.conn.Close()
		close(connection.closed)
		logLine("server utils connection closed::", cause)
	})
}

func (connection *muxConnection) isClosed() bool {
	select {
	case <-connection.closed:
		return true
	default:
		return false
	}
}

func buildServerUtilsFrame(
	secret []byte, nonce *[serverUtilsNonceSize]byte, sequence uint64, opcode byte, payload []byte,
) []byte {
	frame := make([]byte, 0, 1+len(payload)+serverUtilsAuthTagSize)
	frame = append(frame, opcode)
	frame = append(frame, payload...)
	return append(frame, serverUtilsAuthTag(secret, nonce, sequence, frame)...)
}

// buildServerUtilsLengthPrefixedFrame is the variable-width form: the payload's length travels
// between the opcode and the payload. The tag covers the length header too, so a peer cannot make
// the daemon buffer a different amount than the one that was signed.
func buildServerUtilsLengthPrefixedFrame(
	secret []byte, nonce *[serverUtilsNonceSize]byte, sequence uint64, opcode byte, payload []byte,
) []byte {
	frame := make([]byte, 0, 1+2+len(payload)+serverUtilsAuthTagSize)
	frame = append(frame, opcode)
	frame = binary.BigEndian.AppendUint16(frame, uint16(len(payload)))
	frame = append(frame, payload...)
	return append(frame, serverUtilsAuthTag(secret, nonce, sequence, frame)...)
}

// serverUtilsAuthTag signs one frame for one position in one connection's stream. Binding the
// tag to both the server nonce and the frame sequence is what stops a captured frame from being
// replayed, on this connection or any other.
func serverUtilsAuthTag(
	secret []byte, nonce *[serverUtilsNonceSize]byte, sequence uint64, signed []byte,
) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(serverUtilsAuthDomain))
	mac.Write(nonce[:])
	sequenceBytes := [8]byte{}
	binary.BigEndian.PutUint64(sequenceBytes[:], sequence)
	mac.Write(sequenceBytes[:])
	mac.Write(signed)
	return mac.Sum(nil)[:serverUtilsAuthTagSize]
}

func writeCompleteFrame(connection net.Conn, frame []byte) error {
	for len(frame) > 0 {
		written, err := connection.Write(frame)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrUnexpectedEOF
		}
		frame = frame[written:]
	}
	return nil
}
