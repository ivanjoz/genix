package agent

// Agent traffic over the SSE bridge (see fareward/README.md).
//
// In Lambda the backend cannot hold the browser's stream open: an invocation
// ends when the handler returns, and the browser's answer to a command would
// arrive at a different execution environment. The bridge is a small process on
// a normal server that keeps the connection and acts as the rendezvous point:
//
//	bridgePublish → fareward.BridgePublish → fire-and-forget event down the tab's stream
//	bridgeCommand → fareward.BridgeRPC     → command + blocking wait for the tab's reply
//
// Outside Lambda this file is inert: transport stays on the local clientConn in
// ws.go, which is exactly the same protocol over a stream this process owns.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	fareward "github.com/ivanjoz/fareward/go"

	"app/core"
)

const (
	// bridgeChannelWaitMs lets the bridge hold a message for a tab that is
	// reconnecting instead of dropping it. Short: the client opens its stream
	// before sending a turn, so this only covers a mid-turn reconnect.
	bridgeChannelWaitMs = 3000
	// bridgeCallTimeout bounds the non-blocking calls (/publish). /rpc gets the
	// caller's context instead, since it legitimately waits on a human-speed UI.
	bridgeCallTimeout = 15 * time.Second
)

// BridgeEnabled reports whether agent traffic must be relayed. Only in Lambda:
// a backend that can hold its own streams has no reason to add a hop.
func BridgeEnabled() bool {
	return core.Env.IS_SERVERLESS && fareward.BridgeConfigured()
}

// bridgePublish relays one already-framed event to the channel's stream. A tab
// that isn't connected is not an error: events are best-effort by design (no
// buffer), exactly like the local push when no stream is open.
func bridgePublish(channelToken string, messageJSON []byte) error {
	if len(channelToken) == 0 {
		return errors.New("la sesión no tiene un channel token; no se puede publicar")
	}
	callContext, cancelCall := context.WithTimeout(context.Background(), bridgeCallTimeout)
	defer cancelCall()

	delivered, publishError := fareward.BridgePublish(callContext, channelToken, messageJSON, bridgeChannelWaitMs)
	if publishError != nil {
		core.Log("agent.bridge publish error channel::", channelToken, " err::", publishError)
		return publishError
	}
	if !delivered {
		core.Log("agent.bridge publish dropped (canal sin stream) channel::", channelToken, " bytes::", len(messageJSON))
	}
	return nil
}

// makeBridgeCommandID mints the id that correlates a command with its reply.
// Two constraints shape it: it must survive JSON in the browser (numbers above
// 2^53 lose precision), and Lambda instances mint ids independently, so a
// counter would collide. 40 random bits satisfies both.
func makeBridgeCommandID() uint64 {
	return uint64(rand.Int64N(1<<40)) + 1
}

// bridgeCommand is the bridged half of request(): push a command to the tab and
// block on the bridge until the browser's reply comes back, decoding it into
// commandResult.
func bridgeCommand(ctx context.Context, tabID, commandType string, commandPayload any, commandResult any) error {
	session := lookupChatSession(tabID)
	if session == nil || len(session.ChannelToken) == 0 {
		return fmt.Errorf("no hay sesión de chat para el tab %q; no se puede resolver el canal del bridge", tabID)
	}

	commandID := makeBridgeCommandID()
	commandMessage, marshalError := json.Marshal(Message{ID: commandID, Type: commandType, Payload: commandPayload})
	if marshalError != nil {
		return fmt.Errorf("serializar el comando: %w", marshalError)
	}

	// The bridge must give up before the caller does, otherwise a timeout surfaces
	// as a cancelled HTTP call with no diagnosis of who stalled.
	timeoutMilliseconds := 0
	if callDeadline, hasDeadline := ctx.Deadline(); hasDeadline {
		timeoutMilliseconds = int(time.Until(callDeadline).Milliseconds()) - 1000
		if timeoutMilliseconds <= 0 {
			return context.DeadlineExceeded
		}
	}

	core.Log("agent.bridge request send tab::", shortTabID(tabID), " cmd::", commandType, " id::", commandID, " payload_bytes::", len(commandMessage))

	rpcReply, callError := fareward.BridgeRPC(ctx, session.ChannelToken, commandID, commandMessage, timeoutMilliseconds, bridgeChannelWaitMs)
	if callError != nil {
		core.Log("agent.bridge request error tab::", shortTabID(tabID), " cmd::", commandType, " id::", commandID, " err::", callError)
		return callError
	}

	if rpcReply.Kind == TypeError {
		browserError := struct{ Message string }{}
		_ = json.Unmarshal(rpcReply.Payload, &browserError)
		if len(browserError.Message) == 0 {
			browserError.Message = "agent error"
		}
		core.Log("agent.bridge request browser-error tab::", shortTabID(tabID), " cmd::", commandType, " id::", commandID, " err::", browserError.Message)
		return errors.New(browserError.Message)
	}

	core.Log("agent.bridge request ok tab::", shortTabID(tabID), " cmd::", commandType, " id::", commandID, " payload_bytes::", len(rpcReply.Payload))
	if commandResult != nil && len(rpcReply.Payload) > 0 {
		if decodeError := json.Unmarshal(rpcReply.Payload, commandResult); decodeError != nil {
			return fmt.Errorf("decodificar la respuesta del navegador: %w", decodeError)
		}
	}
	return nil
}
