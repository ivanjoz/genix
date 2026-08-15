// Package browser is a minimal Chrome DevTools Protocol client — just enough to launch a headless
// Chrome, screenshot it, evaluate a snippet and listen to console events.
//
// It is deliberately not a browser automation library: the Genix app exposes its own action API
// (POST /agent), so nothing here needs selectors, waiting strategies or input synthesis. The
// browser is a renderer and a session holder; the app drives itself.
package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Screenshots arrive base64-encoded inside the websocket frame, so the default 32KB read limit
// would truncate every capture. 64MB covers a full-page shot of a long report at 2x scale.
const maxFrameBytes = 64 << 20

// Target is one entry of Chrome's /json/list — a tab, a worker or an extension page.
type Target struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// ListTargets asks the running Chrome what it has open. Also the readiness probe: it only answers
// once the debugging port is listening.
func ListTargets(ctx context.Context, debugPort int) ([]Target, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/json/list", debugPort), nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	targets := []Target{}
	if err := json.NewDecoder(response.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("no se pudo leer la lista de targets: %w", err)
	}
	return targets, nil
}

// FirstPageTarget picks the tab to drive. Chrome opens exactly one page target for the URL passed
// on the command line, so the first one is the app; workers and devtools pages are skipped.
func FirstPageTarget(ctx context.Context, debugPort int) (Target, error) {
	targets, err := ListTargets(ctx, debugPort)
	if err != nil {
		return Target{}, err
	}
	for _, target := range targets {
		if target.Type == "page" && target.WebSocketDebuggerURL != "" {
			return target, nil
		}
	}
	return Target{}, fmt.Errorf("chrome está corriendo en el puerto %d pero no tiene ninguna pestaña abierta", debugPort)
}

// WaitForDebugPort blocks until Chrome's debugging port answers or the deadline passes.
func WaitForDebugPort(ctx context.Context, debugPort int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := ListTargets(ctx, debugPort); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("chrome no abrió el puerto de depuración %d en %s", debugPort, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

type pendingCall struct {
	result chan json.RawMessage
	fail   chan error
}

// Client is one CDP session over a websocket. Safe for concurrent use: a single read loop fans
// replies out to their waiting call and events out to the subscribers.
type Client struct {
	conn        *websocket.Conn
	ctx         context.Context
	cancel      context.CancelFunc
	mutex       sync.Mutex
	nextID      int64
	pending     map[int64]pendingCall
	subscribers []chan Event
	readErr     error
}

// Event is a CDP notification: a message with a method but no id.
type Event struct {
	Method string
	Params json.RawMessage
}

// Connect opens a CDP session against a target's websocket URL.
func Connect(ctx context.Context, webSocketURL string) (*Client, error) {
	sessionCtx, cancel := context.WithCancel(ctx)
	conn, _, err := websocket.Dial(sessionCtx, webSocketURL, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("no se pudo conectar al target de chrome: %w", err)
	}
	conn.SetReadLimit(maxFrameBytes)

	client := &Client{
		conn:    conn,
		ctx:     sessionCtx,
		cancel:  cancel,
		pending: map[int64]pendingCall{},
	}
	go client.readLoop()
	return client, nil
}

// ConnectToPage is the common entry point: find the app tab and open a session on it.
func ConnectToPage(ctx context.Context, debugPort int) (*Client, Target, error) {
	target, err := FirstPageTarget(ctx, debugPort)
	if err != nil {
		return nil, Target{}, err
	}
	client, err := Connect(ctx, target.WebSocketDebuggerURL)
	return client, target, err
}

func (c *Client) readLoop() {
	for {
		_, data, err := c.conn.Read(c.ctx)
		if err != nil {
			c.failAll(err)
			return
		}

		var message struct {
			ID     int64           `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Params json.RawMessage `json:"params"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(data, &message); err != nil {
			continue
		}

		// id present → the reply to one of our calls. id absent → an event.
		if message.ID != 0 {
			c.mutex.Lock()
			call, waiting := c.pending[message.ID]
			delete(c.pending, message.ID)
			c.mutex.Unlock()
			if !waiting {
				continue
			}
			if message.Error != nil {
				call.fail <- fmt.Errorf("cdp: %s", message.Error.Message)
				continue
			}
			call.result <- message.Result
			continue
		}

		if message.Method == "" {
			continue
		}
		c.mutex.Lock()
		subscribers := append([]chan Event{}, c.subscribers...)
		c.mutex.Unlock()
		for _, subscriber := range subscribers {
			// Non-blocking: a slow consumer drops events instead of stalling the read loop, which
			// would also stall every in-flight call.
			select {
			case subscriber <- Event{Method: message.Method, Params: message.Params}:
			default:
			}
		}
	}
}

// failAll releases every waiting call when the connection dies, so no caller blocks forever.
func (c *Client) failAll(err error) {
	c.mutex.Lock()
	c.readErr = err
	for id, call := range c.pending {
		call.fail <- err
		delete(c.pending, id)
	}
	subscribers := c.subscribers
	c.subscribers = nil
	c.mutex.Unlock()
	for _, subscriber := range subscribers {
		close(subscriber)
	}
}

// Call sends a CDP command and waits for its reply.
func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if params == nil {
		params = map[string]any{}
	}

	c.mutex.Lock()
	if c.readErr != nil {
		c.mutex.Unlock()
		return nil, c.readErr
	}
	c.nextID++
	id := c.nextID
	call := pendingCall{result: make(chan json.RawMessage, 1), fail: make(chan error, 1)}
	c.pending[id] = call
	c.mutex.Unlock()

	payload, err := json.Marshal(map[string]any{"id": id, "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	if err := c.conn.Write(ctx, websocket.MessageText, payload); err != nil {
		return nil, fmt.Errorf("no se pudo enviar %s: %w", method, err)
	}

	select {
	case result := <-call.result:
		return result, nil
	case err := <-call.fail:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// CallInto is Call plus decoding into a typed result.
func (c *Client) CallInto(ctx context.Context, method string, params any, out any) error {
	result, err := c.Call(ctx, method, params)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(result, out)
}

// Subscribe returns a channel receiving every event from now on. Buffered because the read loop
// drops rather than blocks; the buffer absorbs a burst of console output at page load.
func (c *Client) Subscribe() <-chan Event {
	events := make(chan Event, 256)
	c.mutex.Lock()
	c.subscribers = append(c.subscribers, events)
	c.mutex.Unlock()
	return events
}

func (c *Client) Close() {
	_ = c.conn.Close(websocket.StatusNormalClosure, "")
	c.cancel()
}
