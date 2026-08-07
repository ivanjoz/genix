package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// One channel is one browser tab. Everything the bridge holds lives here and
// only here: a queue of outbound SSE frames plus the table of replies the
// backend is currently blocked on. Nothing is persisted — a message published
// while the tab is disconnected is dropped by design (the client resynchronizes
// through the normal API).

// frameSendTimeout bounds how long a publisher waits for a wedged client whose
// outbound queue is full. Giving up beats blocking the backend's turn forever.
const frameSendTimeout = 10 * time.Second

// outboundQueueSize is the per-tab burst buffer. A turn emits status events far
// faster than a slow mobile connection drains them.
const outboundQueueSize = 64

// replyEnvelope is the browser's answer to one backend command, correlated by
// the command's ID.
type replyEnvelope struct {
	Kind    string          // "result" | "error"
	Payload json.RawMessage
}

// clientChannel is one connected tab. Its key is the channel token itself
// (channel.go), which decoding proves to be a canonical, one-to-one name for
// the company/user/tab triple — so two spellings can never split one tab into
// two registry entries.
type clientChannel struct {
	key            string
	outboundFrames chan []byte
	closed         chan struct{}
	closeOnce      sync.Once
	connectedAt    time.Time

	pendingMutex   sync.Mutex
	pendingReplies map[uint64]chan replyEnvelope
}

// Close releases every goroutine parked on this channel, exactly once.
func (channel *clientChannel) Close() {
	channel.closeOnce.Do(func() { close(channel.closed) })
}

// SendFrame queues one SSE `data:` frame. The payload is compacted first: a
// frame is terminated by a blank line, so an embedded newline in pretty-printed
// JSON would split one message into two malformed ones.
func (channel *clientChannel) SendFrame(payloadJSON []byte) error {
	compactPayload := bytes.Buffer{}
	if compactError := json.Compact(&compactPayload, payloadJSON); compactError != nil {
		return errors.New("el mensaje a publicar no es JSON válido: " + compactError.Error())
	}

	frame := make([]byte, 0, compactPayload.Len()+8)
	frame = append(frame, "data: "...)
	frame = append(frame, compactPayload.Bytes()...)
	frame = append(frame, '\n', '\n')

	sendTimer := time.NewTimer(frameSendTimeout)
	defer sendTimer.Stop()

	select {
	case channel.outboundFrames <- frame:
		return nil
	case <-channel.closed:
		return errors.New("el stream del cliente ya está cerrado")
	case <-sendTimer.C:
		return errors.New("timeout escribiendo en el stream del cliente")
	}
}

// AwaitReply registers a waiter for a command id and returns the channel its
// reply will arrive on. ReleasePendingReply must always be called afterwards so
// an abandoned waiter (timeout, cancelled turn) doesn't leak.
func (channel *clientChannel) AwaitReply(commandID uint64) chan replyEnvelope {
	replyChannel := make(chan replyEnvelope, 1)
	channel.pendingMutex.Lock()
	channel.pendingReplies[commandID] = replyChannel
	channel.pendingMutex.Unlock()
	return replyChannel
}

func (channel *clientChannel) ReleasePendingReply(commandID uint64) {
	channel.pendingMutex.Lock()
	delete(channel.pendingReplies, commandID)
	channel.pendingMutex.Unlock()
}

// DeliverReply hands the browser's answer to whoever is waiting for it. Returns
// false when nobody is (a late reply after a timeout, or a duplicate).
func (channel *clientChannel) DeliverReply(commandID uint64, envelope replyEnvelope) bool {
	channel.pendingMutex.Lock()
	replyChannel, waiterFound := channel.pendingReplies[commandID]
	if waiterFound {
		delete(channel.pendingReplies, commandID)
	}
	channel.pendingMutex.Unlock()

	if !waiterFound {
		return false
	}
	select {
	case replyChannel <- envelope:
	default:
	}
	return true
}

// FailAllPendingReplies unblocks every waiter with an error. Called when the tab
// disconnects: its commands can never be answered now, and the backend would
// otherwise sit until its own timeout.
func (channel *clientChannel) FailAllPendingReplies(reason string) int {
	errorPayload, _ := json.Marshal(struct{ Message string }{reason})

	channel.pendingMutex.Lock()
	defer channel.pendingMutex.Unlock()

	failedCount := len(channel.pendingReplies)
	for commandID, replyChannel := range channel.pendingReplies {
		select {
		case replyChannel <- replyEnvelope{Kind: "error", Payload: errorPayload}:
		default:
		}
		delete(channel.pendingReplies, commandID)
	}
	return failedCount
}

// channelRegistry maps channel keys to connected tabs and lets a publisher wait
// for one that is still connecting.
type channelRegistry struct {
	mutex          sync.RWMutex
	channels       map[string]*clientChannel
	arrivalWaiters map[string][]chan struct{}
}

func newChannelRegistry() *channelRegistry {
	return &channelRegistry{
		channels:       map[string]*clientChannel{},
		arrivalWaiters: map[string][]chan struct{}{},
	}
}

// OpenChannel installs a fresh channel for key, evicting any previous one.
// Last-connection-wins: the evicted stream is told why it is ending so a
// duplicated tab (which clones sessionStorage and inherits the tab id) can mint
// a new id instead of the two endlessly kicking each other off.
func (registry *channelRegistry) OpenChannel(key string) *clientChannel {
	channel := &clientChannel{
		key:            key,
		outboundFrames: make(chan []byte, outboundQueueSize),
		closed:         make(chan struct{}),
		connectedAt:    time.Now(),
		pendingReplies: map[uint64]chan replyEnvelope{},
	}

	registry.mutex.Lock()
	previousChannel := registry.channels[key]
	registry.channels[key] = channel
	waitersToWake := registry.arrivalWaiters[key]
	delete(registry.arrivalWaiters, key)
	registry.mutex.Unlock()

	if previousChannel != nil {
		select {
		case previousChannel.outboundFrames <- []byte("data: {\"Type\":\"replaced\"}\n\n"):
		default:
		}
		previousChannel.Close()
		previousChannel.FailAllPendingReplies("el cliente reconectó")
		logInfo("canal reemplazado ::", key)
	}

	for _, waiter := range waitersToWake {
		close(waiter)
	}
	return channel
}

// CloseChannel removes channel from the registry, but only if it is still the
// installed one — a stream that unwinds late must not evict its successor.
func (registry *channelRegistry) CloseChannel(channel *clientChannel) {
	registry.mutex.Lock()
	if registry.channels[channel.key] == channel {
		delete(registry.channels, channel.key)
	}
	registry.mutex.Unlock()

	channel.Close()
	if failedCount := channel.FailAllPendingReplies("el cliente se desconectó"); failedCount > 0 {
		logWarn("canal cerrado con", failedCount, "respuestas pendientes ::", channel.key)
	}
}

func (registry *channelRegistry) FindChannel(key string) *clientChannel {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()
	return registry.channels[key]
}

func (registry *channelRegistry) ConnectedChannelCount() int {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()
	return len(registry.channels)
}

// AwaitChannel returns the channel for key, waiting up to maxWait for it to
// connect. This is the server-side half of the handshake: the client normally
// opens its stream before sending a turn, but a reconnect mid-turn can leave a
// short window where the backend has something to say and nobody to say it to.
func (registry *channelRegistry) AwaitChannel(ctx context.Context, key string, maxWait time.Duration) *clientChannel {
	if channel := registry.FindChannel(key); channel != nil {
		return channel
	}
	if maxWait <= 0 {
		return nil
	}

	arrivalSignal := make(chan struct{})
	registry.mutex.Lock()
	// Re-check under the write lock: the channel may have opened between the
	// lookup above and here, which would close the signal we never registered.
	if channel := registry.channels[key]; channel != nil {
		registry.mutex.Unlock()
		return channel
	}
	registry.arrivalWaiters[key] = append(registry.arrivalWaiters[key], arrivalSignal)
	registry.mutex.Unlock()

	waitTimer := time.NewTimer(maxWait)
	defer waitTimer.Stop()

	select {
	case <-arrivalSignal:
		return registry.FindChannel(key)
	case <-ctx.Done():
		registry.dropArrivalWaiter(key, arrivalSignal)
		return nil
	case <-waitTimer.C:
		registry.dropArrivalWaiter(key, arrivalSignal)
		return nil
	}
}

func (registry *channelRegistry) dropArrivalWaiter(key string, arrivalSignal chan struct{}) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	remainingWaiters := registry.arrivalWaiters[key][:0]
	for _, waiter := range registry.arrivalWaiters[key] {
		if waiter != arrivalSignal {
			remainingWaiters = append(remainingWaiters, waiter)
		}
	}
	if len(remainingWaiters) == 0 {
		delete(registry.arrivalWaiters, key)
	} else {
		registry.arrivalWaiters[key] = remainingWaiters
	}
}
