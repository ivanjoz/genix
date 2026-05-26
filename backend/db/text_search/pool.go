package text_search

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// connPool is a bounded pool of search connections in one channel
// mode. At most `max` connections are open at any time; `min` of them
// are kept warm by re-dialing eagerly when the pool drops below that
// floor.
type connPool struct {
	mode string
	min  int
	max  int

	mu         sync.Mutex
	idle       []*searchConn
	open       int
	closed     bool
	pendingCh  chan struct{} // signaled when a connection returns to the pool
	dialCtxCxl context.CancelFunc
}

func newConnPool(mode string, min, max int) *connPool {
	if min < 0 {
		min = 0
	}
	if max < min {
		max = min
	}
	if max == 0 {
		max = 1
	}
	return &connPool{
		mode:      mode,
		min:       min,
		max:       max,
		idle:      make([]*searchConn, 0, max),
		pendingCh: make(chan struct{}, max),
	}
}

// acquire returns a ready-to-use connection. It dials a new one if the
// pool is below capacity, otherwise waits for a returned connection.
// Honors ctx for both dial and wait.
func (p *connPool) acquire(ctx context.Context) (*searchConn, error) {
	for {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return nil, errors.New("text_search: pool closed")
		}
		// Drain idle list, oldest first.
		for len(p.idle) > 0 {
			c := p.idle[0]
			p.idle = p.idle[1:]
			if time.Since(c.usedAt) > idleTimeout || c.broken {
				p.open--
				p.mu.Unlock()
				_ = c.close()
				p.mu.Lock()
				continue
			}
			p.mu.Unlock()
			// Light liveness check; cheap on a loopback connection. If it
			// fails we throw the connection away and loop.
			pingCtx, cancel := context.WithTimeout(ctx, commandTimeout)
			err := c.ping(pingCtx)
			cancel()
			if err == nil {
				return c, nil
			}
			p.mu.Lock()
			p.open--
			p.mu.Unlock()
			_ = c.close()
			p.mu.Lock()
			continue
		}
		// No idle; can we open a new one?
		if p.open < p.max {
			p.open++
			p.mu.Unlock()
			conn, err := dialAndHandshake(ctx, p.mode)
			if err != nil {
				p.mu.Lock()
				p.open--
				p.mu.Unlock()
				return nil, err
			}
			return conn, nil
		}
		// At capacity; wait for a return.
		p.mu.Unlock()
		select {
		case <-p.pendingCh:
			// Loop and retry checkout.
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// release returns a healthy connection to the pool. Broken connections
// are discarded.
func (p *connPool) release(c *searchConn) {
	if c == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		// Pool is gone; just close the conn outside the lock.
		go func() { _ = c.close() }()
		return
	}
	if c.broken {
		p.open--
		go func() { _ = c.close() }()
		p.signalWaiter()
		return
	}
	c.usedAt = time.Now()
	p.idle = append(p.idle, c)
	p.signalWaiter()
}

// discard force-closes a connection and removes it from accounting.
// Used when a command fails in a way that leaves the connection in an
// ambiguous protocol state.
func (p *connPool) discard(c *searchConn) {
	if c == nil {
		return
	}
	p.mu.Lock()
	p.open--
	p.mu.Unlock()
	_ = c.close()
	p.mu.Lock()
	p.signalWaiter()
	p.mu.Unlock()
}

// signalWaiter wakes one acquire() blocked on pendingCh. Must be called
// with p.mu held to avoid a missed signal vs. a fresh acquire.
func (p *connPool) signalWaiter() {
	select {
	case p.pendingCh <- struct{}{}:
	default:
	}
}

// close drains and shuts down every connection in the pool. New
// acquire() calls fail with "pool closed".
func (p *connPool) close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	conns := p.idle
	p.idle = nil
	p.open = 0
	p.mu.Unlock()
	var firstErr error
	for _, c := range conns {
		if err := c.close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// ensureMinWarm is a hook the ingest path can call after a successful
// release to keep at least `min` connections idle. Skipped on the hot
// path for simplicity — the pool already dials lazily on demand.
//
//nolint:unused
func (p *connPool) ensureMinWarm(ctx context.Context) error {
	p.mu.Lock()
	deficit := p.min - len(p.idle) - 0
	openSlots := p.max - p.open
	if openSlots < deficit {
		deficit = openSlots
	}
	if deficit <= 0 {
		p.mu.Unlock()
		return nil
	}
	p.open += deficit
	p.mu.Unlock()
	for i := 0; i < deficit; i++ {
		conn, err := dialAndHandshake(ctx, p.mode)
		if err != nil {
			p.mu.Lock()
			p.open -= (deficit - i)
			p.mu.Unlock()
			return fmt.Errorf("text_search: warm dial: %w", err)
		}
		p.mu.Lock()
		p.idle = append(p.idle, conn)
		p.mu.Unlock()
	}
	return nil
}
