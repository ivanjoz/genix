package agent

// Client of the SSE bridge (see sse_bridge/ and PLAN_SSE_BRIDGE.md).
//
// In Lambda the backend cannot hold the browser's stream open: an invocation
// ends when the handler returns, and the browser's answer to a command would
// arrive at a different execution environment. The bridge is a small process on
// a normal server that keeps the connection and acts as the rendezvous point:
//
//	bridgePublish → POST /publish → fire-and-forget event down the tab's stream
//	bridgeCommand → POST /rpc     → command + blocking wait for the tab's reply
//
// Outside Lambda this file is inert: transport stays on the local clientConn in
// ws.go, which is exactly the same protocol over a stream this process owns.

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"app/core"
)

const (
	// bridgeServiceAuthHeaderName and bridgeServiceAuthMessagePrefix mirror
	// sse_bridge/auth.go. The bridge is a separate Go module, so the compiler
	// cannot enforce this — changing one side requires changing the other.
	bridgeServiceAuthHeaderName    = "X-Bridge-Auth"
	bridgeServiceAuthMessagePrefix = "sse-bridge:v1|"

	// bridgeChannelWaitMs lets the bridge hold a message for a tab that is
	// reconnecting instead of dropping it. Short: the client opens its stream
	// before sending a turn, so this only covers a mid-turn reconnect.
	bridgeChannelWaitMs = 3000
	// bridgeCallTimeout bounds the non-blocking calls (/publish). /rpc gets the
	// caller's context instead, since it legitimately waits on a human-speed UI.
	bridgeCallTimeout = 15 * time.Second
)

// bridgeHTTPClient has no client-level timeout on purpose: /rpc blocks until the
// browser answers, and the per-call context is what bounds it.
var bridgeHTTPClient = &http.Client{}

// BridgeEnabled reports whether agent traffic must be relayed. Only in Lambda:
// a backend that can hold its own streams has no reason to add a hop.
func BridgeEnabled() bool {
	return core.Env.IS_SERVERLESS && len(bridgeBaseURL()) > 0
}

func bridgeBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(core.Env.SSE_BRIDGE_URL), "/")
}

// makeBridgeServiceAuthHeader signs the current timestamp with SECRET_PHRASE.
// Both processes read the same config.toml, so no extra secret has to be
// provisioned for the bridge.
func makeBridgeServiceAuthHeader() string {
	timestampText := strconv.FormatInt(time.Now().Unix(), 10)
	hashMac := hmac.New(sha256.New, []byte(core.Env.SECRET_PHRASE))
	hashMac.Write([]byte(bridgeServiceAuthMessagePrefix + timestampText))
	return timestampText + "." + hex.EncodeToString(hashMac.Sum(nil))
}

// bridgeCall posts one authenticated request to the bridge and decodes its JSON
// answer into responseTarget.
func bridgeCall(ctx context.Context, path string, requestBody any, responseTarget any) error {
	encodedBody, marshalError := json.Marshal(requestBody)
	if marshalError != nil {
		return fmt.Errorf("serializar la petición al bridge: %w", marshalError)
	}

	bridgeRequest, requestError := http.NewRequestWithContext(ctx, http.MethodPost, bridgeBaseURL()+path, bytes.NewReader(encodedBody))
	if requestError != nil {
		return fmt.Errorf("construir la petición al bridge: %w", requestError)
	}
	bridgeRequest.Header.Set("Content-Type", "application/json")
	bridgeRequest.Header.Set(bridgeServiceAuthHeaderName, makeBridgeServiceAuthHeader())

	bridgeResponse, callError := bridgeHTTPClient.Do(bridgeRequest)
	if callError != nil {
		return fmt.Errorf("el bridge no respondió: %w", callError)
	}
	defer bridgeResponse.Body.Close()

	responseBody, readError := io.ReadAll(bridgeResponse.Body)
	if readError != nil {
		return fmt.Errorf("leer la respuesta del bridge: %w", readError)
	}
	if bridgeResponse.StatusCode != http.StatusOK {
		errorDetail := struct{ Error string }{}
		_ = json.Unmarshal(responseBody, &errorDetail)
		if len(errorDetail.Error) == 0 {
			errorDetail.Error = core.StrCut(string(responseBody), 200)
		}
		return fmt.Errorf("el bridge respondió %d: %s", bridgeResponse.StatusCode, errorDetail.Error)
	}
	if responseTarget == nil {
		return nil
	}
	return json.Unmarshal(responseBody, responseTarget)
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

	publishResponse := struct{ Delivered bool }{}
	publishError := bridgeCall(callContext, "/publish", map[string]any{
		"Channel": channelToken,
		"Message": json.RawMessage(messageJSON),
		"WaitMs":  bridgeChannelWaitMs,
	}, &publishResponse)

	if publishError != nil {
		core.Log("agent.bridge publish error channel::", channelToken, " err::", publishError)
		return publishError
	}
	if !publishResponse.Delivered {
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

	rpcResponse := struct {
		Kind    string
		Payload json.RawMessage
	}{}
	callError := bridgeCall(ctx, "/rpc", map[string]any{
		"Channel":   session.ChannelToken,
		"ID":        commandID,
		"Message":   json.RawMessage(commandMessage),
		"TimeoutMs": timeoutMilliseconds,
		"WaitMs":    bridgeChannelWaitMs,
	}, &rpcResponse)
	if callError != nil {
		core.Log("agent.bridge request error tab::", shortTabID(tabID), " cmd::", commandType, " id::", commandID, " err::", callError)
		return callError
	}

	if rpcResponse.Kind == TypeError {
		browserError := struct{ Message string }{}
		_ = json.Unmarshal(rpcResponse.Payload, &browserError)
		if len(browserError.Message) == 0 {
			browserError.Message = "agent error"
		}
		core.Log("agent.bridge request browser-error tab::", shortTabID(tabID), " cmd::", commandType, " id::", commandID, " err::", browserError.Message)
		return errors.New(browserError.Message)
	}

	core.Log("agent.bridge request ok tab::", shortTabID(tabID), " cmd::", commandType, " id::", commandID, " payload_bytes::", len(rpcResponse.Payload))
	if commandResult != nil && len(rpcResponse.Payload) > 0 {
		if decodeError := json.Unmarshal(rpcResponse.Payload, commandResult); decodeError != nil {
			return fmt.Errorf("decodificar la respuesta del navegador: %w", decodeError)
		}
	}
	return nil
}
