// Package text_search talks to a GenixSearch daemon over TCP. It
// preserves the ORM-facing API from the previous FTS5 / Sonic backends
// so call sites in backend/db do not change.
//
// Granularity: one logical index per (table, partition, statusGroup).
// On the wire: collection = {tableName}, bucket = "p{partition}_s{group}".
// Group 0 holds records with status == 0 (soft-deleted / inactive);
// group 1 holds everything else.
//
// GenixSearch exposes only PUSHI (multi-key, replace semantics) and
// single-key POPI; there is no bulk POPI. UpsertBatch issues:
//
//  1. A pipelined POPI run on the opposite-status bucket for every
//     record ID — guards against status flips (the server can't detect
//     a flip on its own).
//  2. A pipelined POPI run on the current-status bucket for records
//     whose SearchText is empty (PUSHI rejects empty payloads).
//  3. One or more PUSHI lines on the current-status bucket batching
//     the non-empty records, chunked to fit the channel buffer.
package text_search

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// ErrNotImplemented is returned by Search until the read path lands.
var ErrNotImplemented = errors.New("text_search: search path not implemented")

// searchConfig holds the GenixSearch endpoint and credentials.
type searchConfig struct {
	host     string
	port     int
	password string
}

var (
	configMu  sync.Mutex
	cfg       searchConfig
	ingestMgr *connPool
	searchMgr *connPool
)

// Configure sets the search endpoint and credentials. The first call
// wins for the lifetime of the process — subsequent calls before any
// shard is opened replace the previous values; afterwards they are
// ignored because the pools have already captured the config.
func Configure(host string, port int, password string) {
	configMu.Lock()
	defer configMu.Unlock()
	cfg = searchConfig{
		host:     strings.TrimSpace(host),
		port:     port,
		password: strings.TrimSpace(password),
	}
}

// getConfig returns a copy of the active config or an error if
// Configure was never called.
func getConfig() (searchConfig, error) {
	configMu.Lock()
	defer configMu.Unlock()
	if cfg.host == "" || cfg.port == 0 {
		return searchConfig{}, errors.New("text_search: Configure(host, port, password) was not called")
	}
	return cfg, nil
}

// initPools creates the ingest and search pools on first use. Safe
// to call concurrently — guarded by configMu. Existing pools (e.g.
// set by tests via an internal hook) are preserved.
func initPools() {
	configMu.Lock()
	defer configMu.Unlock()
	if ingestMgr == nil {
		ingestMgr = newConnPool("ingest", poolMin, poolMax)
	}
	if searchMgr == nil {
		searchMgr = newConnPool("search", poolMin, poolMax)
	}
}

// Close drains both pools. Intended for test teardown and graceful
// shutdown; long-lived processes usually let the OS clean up.
func Close() error {
	configMu.Lock()
	defer configMu.Unlock()
	var firstErr error
	if ingestMgr != nil {
		if err := ingestMgr.close(); err != nil && firstErr == nil {
			firstErr = err
		}
		ingestMgr = nil
	}
	if searchMgr != nil {
		if err := searchMgr.close(); err != nil && firstErr == nil {
			firstErr = err
		}
		searchMgr = nil
	}
	return firstErr
}

// searchConn is one open TCP connection to the GenixSearch daemon,
// bound to a channel mode at handshake time. Not safe for concurrent
// use — the pool checks one out per call site.
type searchConn struct {
	mode       string
	netConn    net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	bufferSize int
	usedAt     time.Time
	broken     bool
}

// exec writes one command line and reads the reply. Single-shot
// commands (OK / RESULT / PONG) consume one line; PENDING-style
// commands are handled by execPending below.
func (c *searchConn) exec(ctx context.Context, line string) (searchResult, error) {
	if c.broken {
		return searchResult{}, fmt.Errorf("%w: connection already broken", ErrProtocol)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.netConn.SetDeadline(deadline)
	} else {
		_ = c.netConn.SetDeadline(time.Now().Add(commandTimeout))
	}
	if err := writeLine(c.writer, line); err != nil {
		c.broken = true
		return searchResult{}, err
	}
	reply, err := readLine(c.reader)
	if err != nil {
		c.broken = true
		return searchResult{}, err
	}
	res, err := parseResult(reply)
	if err != nil {
		// ERR carries a ProtocolError but isn't a transport failure;
		// the connection stays usable. Only mark broken on framing problems.
		if _, ok := err.(*ProtocolError); !ok {
			c.broken = true
		}
	}
	return res, err
}

// execPending issues a command that produces PENDING + EVENT (QUERY).
// Returns the EVENT result.
func (c *searchConn) execPending(ctx context.Context, line string) (searchResult, error) {
	pending, err := c.exec(ctx, line)
	if err != nil {
		return searchResult{}, err
	}
	if pending.kind != kindPending {
		c.broken = true
		return searchResult{}, fmt.Errorf("%w: expected PENDING, got %v", ErrProtocol, pending.kind)
	}
	event, err := readLine(c.reader)
	if err != nil {
		c.broken = true
		return searchResult{}, err
	}
	res, err := parseResult(event)
	if err != nil {
		if _, ok := err.(*ProtocolError); !ok {
			c.broken = true
		}
		return res, err
	}
	if res.kind != kindEvent || res.marker != pending.marker {
		c.broken = true
		return res, fmt.Errorf("%w: EVENT marker mismatch (%q vs %q)", ErrProtocol, res.marker, pending.marker)
	}
	return res, nil
}

// dialAndHandshake opens a TCP connection to the daemon, performs the
// CONNECTED -> START -> STARTED handshake, and returns a ready-to-use
// connection bound to the requested mode ("ingest" / "search").
func dialAndHandshake(ctx context.Context, mode string) (*searchConn, error) {
	conf, err := getConfig()
	if err != nil {
		return nil, err
	}
	d := net.Dialer{Timeout: dialTimeout}
	addr := fmt.Sprintf("%s:%d", conf.host, conf.port)
	netConn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("text_search: dial %s: %w", addr, err)
	}
	c := &searchConn{
		mode:    mode,
		netConn: netConn,
		reader:  bufio.NewReaderSize(netConn, 8192),
		writer:  bufio.NewWriterSize(netConn, 8192),
		usedAt:  time.Now(),
	}
	_ = netConn.SetDeadline(time.Now().Add(commandTimeout))
	// 1. Wait for CONNECTED greeting.
	greeting, err := readLine(c.reader)
	if err != nil {
		_ = netConn.Close()
		return nil, fmt.Errorf("text_search: read greeting from %s: %w", addr, err)
	}
	if res, perr := parseResult(greeting); perr != nil || res.kind != kindConnected {
		_ = netConn.Close()
		return nil, fmt.Errorf("text_search: unexpected greeting %q from %s", greeting, addr)
	}
	// 2. Send START <mode> <password>.
	startLine := fmt.Sprintf("START %s %s", mode, conf.password)
	if err := writeLine(c.writer, startLine); err != nil {
		_ = netConn.Close()
		return nil, fmt.Errorf("text_search: send START: %w", err)
	}
	// 3. Wait for STARTED, capture buffer size.
	startedLine, err := readLine(c.reader)
	if err != nil {
		_ = netConn.Close()
		return nil, fmt.Errorf("text_search: read STARTED: %w", err)
	}
	res, err := parseResult(startedLine)
	if err != nil {
		_ = netConn.Close()
		return nil, fmt.Errorf("text_search: parse STARTED %q: %w", startedLine, err)
	}
	if res.kind != kindStarted {
		_ = netConn.Close()
		return nil, fmt.Errorf("text_search: expected STARTED, got %q", startedLine)
	}
	c.bufferSize = res.bufferSize
	// Clear deadline for subsequent use; the pool will set per-command deadlines.
	_ = netConn.SetDeadline(time.Time{})
	return c, nil
}

// ping verifies a pooled connection is still alive. Used at checkout.
func (c *searchConn) ping(ctx context.Context) error {
	res, err := c.exec(ctx, "PING")
	if err != nil {
		return err
	}
	if res.kind != kindPong {
		c.broken = true
		return fmt.Errorf("%w: expected PONG, got %v", ErrProtocol, res.kind)
	}
	return nil
}

// close sends QUIT (best-effort) and closes the underlying socket.
func (c *searchConn) close() error {
	_ = c.netConn.SetDeadline(time.Now().Add(quitTimeout))
	_ = writeLine(c.writer, "QUIT")
	return c.netConn.Close()
}

// Timeouts and pool sizing. Knobs concentrated here so they're easy
// to revisit.
const (
	poolMin        = 2
	poolMax        = 8
	dialTimeout    = 3 * time.Second
	commandTimeout = 10 * time.Second
	quitTimeout    = 500 * time.Millisecond
	idleTimeout    = 300 * time.Second
)
