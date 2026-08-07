package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTP surface of the bridge. Two endpoints face the browser (`/sse`, `/in`) and
// two face the backend (`/publish`, `/rpc`). The bridge never interprets the
// messages it moves: they are opaque JSON objects framed as SSE `data:` events.
// The single exception is the command id, which travels as its own field so a
// reply can be correlated without parsing the payload.

const (
	// keepaliveInterval sends an SSE comment often enough to survive the idle
	// read timeouts of nginx and of mobile carrier NATs.
	keepaliveInterval = 20 * time.Second
	// defaultRPCTimeout applies when the backend doesn't state one. Generous: the
	// browser may be re-rendering a whole page before it can answer.
	defaultRPCTimeout = 60 * time.Second
	// maxRPCTimeout caps whatever the backend asks for, so a bad request can't
	// pin a goroutine and a pending entry for hours.
	maxRPCTimeout = 10 * time.Minute
	// maxChannelWait caps the grace period a publisher may ask the bridge to wait
	// for a reconnecting tab.
	maxChannelWait = 30 * time.Second
)

type bridgeServer struct {
	config    BridgeConfig
	registry  *channelRegistry
	startedAt time.Time
}

func newBridgeServer(config BridgeConfig) *bridgeServer {
	return &bridgeServer{
		config:    config,
		registry:  newChannelRegistry(),
		startedAt: time.Now(),
	}
}

// Routes builds the mux. Client endpoints get CORS because the browser calls
// them cross-origin from the app's own domain; the service endpoints are only
// ever called server-to-server.
func (server *bridgeServer) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/sse", withClientCORS(http.HandlerFunc(server.handleClientStream)))
	mux.Handle("/in", withClientCORS(http.HandlerFunc(server.handleClientInbound)))
	mux.HandleFunc("/publish", server.handleServicePublish)
	mux.HandleFunc("/rpc", server.handleServiceRPC)
	mux.HandleFunc("/health", server.handleHealth)
	return mux
}

// --- Browser-facing endpoints -------------------------------------------------

// resolveClientChannel turns a browser request into the channel it may address.
//
// The channel token names the tab, but it does NOT prove who is asking: it is a
// plain identifier anyone could rewrite. The session token is the proof, so the
// two are cross-checked here. Without this a client could edit the company id
// inside its own token and attach to another tenant's stream.
func (server *bridgeServer) resolveClientChannel(request *http.Request) (string, error) {
	channelToken := strings.TrimSpace(request.URL.Query().Get("ch"))
	if len(channelToken) == 0 {
		return "", errors.New("falta el parámetro ?ch=")
	}

	channelCompanyID, channelUserID, _, decodeError := DecodeChannelToken(channelToken)
	if decodeError != nil {
		return "", decodeError
	}

	userToken, authError := authenticateUserRequest(request, server.config.SecretPhrase)
	if authError != nil {
		return "", authError
	}
	if channelCompanyID != userToken.CompanyID || channelUserID != userToken.ID {
		return "", errors.New("el canal solicitado no pertenece al usuario autenticado")
	}
	return channelToken, nil
}

// handleClientStream opens the tab's event stream. It stays open until the
// client disconnects or another connection claims the same channel.
func (server *bridgeServer) handleClientStream(responseWriter http.ResponseWriter, request *http.Request) {
	channelKey, resolveError := server.resolveClientChannel(request)
	if resolveError != nil {
		logWarn("stream rechazado ::", resolveError)
		writeJSONError(responseWriter, http.StatusUnauthorized, resolveError.Error())
		return
	}

	responseFlusher, supportsFlush := responseWriter.(http.Flusher)
	if !supportsFlush {
		writeJSONError(responseWriter, http.StatusInternalServerError, "el servidor no soporta streaming")
		return
	}

	responseWriter.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	responseWriter.Header().Set("Cache-Control", "no-cache")
	responseWriter.Header().Set("Connection", "keep-alive")
	// Defeat reverse-proxy response buffering (nginx honours this), otherwise
	// every event is held back until the buffer fills.
	responseWriter.Header().Set("X-Accel-Buffering", "no")
	responseWriter.WriteHeader(http.StatusOK)
	responseFlusher.Flush()

	channel := server.registry.OpenChannel(channelKey)
	defer server.registry.CloseChannel(channel)

	// Handshake. Queued only after OpenChannel, so a client that has seen this
	// frame is guaranteed to be routable — that is what lets the frontend delay
	// its first turn until the backend can actually reach it.
	if handshakeError := channel.SendFrame([]byte(`{"Type":"bridgeReady"}`)); handshakeError != nil {
		logWarn("handshake fallido ::", channelKey, "::", handshakeError)
		return
	}
	logInfo("stream conectado ::", channelKey, ":: desde", request.RemoteAddr)
	defer func() { logInfo("stream desconectado ::", channelKey) }()

	keepaliveTicker := time.NewTicker(keepaliveInterval)
	defer keepaliveTicker.Stop()

	requestContext := request.Context()
	for {
		select {
		case <-requestContext.Done():
			return

		case <-channel.closed:
			// Drain frames queued right before the close (notably the "replaced"
			// notice) so the client learns why the stream ended.
			for {
				select {
				case frame := <-channel.outboundFrames:
					if _, writeError := responseWriter.Write(frame); writeError != nil {
						return
					}
					responseFlusher.Flush()
				default:
					return
				}
			}

		case frame := <-channel.outboundFrames:
			if _, writeError := responseWriter.Write(frame); writeError != nil {
				logWarn("escritura fallida, cerrando stream ::", channelKey, "::", writeError)
				return
			}
			responseFlusher.Flush()

		case <-keepaliveTicker.C:
			if _, writeError := io.WriteString(responseWriter, ": ping\n\n"); writeError != nil {
				return
			}
			responseFlusher.Flush()
		}
	}
}

// clientInboundMessage is the browser→backend envelope. ID > 0 marks it as the
// reply to a command the backend is blocked on; ID == 0 is an unsolicited event.
type clientInboundMessage struct {
	ID      uint64
	Type    string
	Payload json.RawMessage
}

// handleClientInbound routes the browser's reply to the /rpc call waiting for
// it. Unsolicited events are logged and dropped: the backend that would consume
// them is not connected to the bridge, it only calls in.
func (server *bridgeServer) handleClientInbound(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSONError(responseWriter, http.StatusMethodNotAllowed, "se esperaba POST")
		return
	}

	channelKey, resolveError := server.resolveClientChannel(request)
	if resolveError != nil {
		writeJSONError(responseWriter, http.StatusUnauthorized, resolveError.Error())
		return
	}

	inboundMessage := clientInboundMessage{}
	if decodeError := json.NewDecoder(request.Body).Decode(&inboundMessage); decodeError != nil {
		writeJSONError(responseWriter, http.StatusBadRequest, "cuerpo inválido: "+decodeError.Error())
		return
	}

	if inboundMessage.ID == 0 {
		logInfo("evento no solicitado descartado ::", channelKey, ":: tipo", inboundMessage.Type)
		writeJSON(responseWriter, http.StatusOK, map[string]any{"Delivered": false})
		return
	}

	channel := server.registry.FindChannel(channelKey)
	if channel == nil {
		logWarn("respuesta sin canal ::", channelKey, ":: id", inboundMessage.ID)
		writeJSON(responseWriter, http.StatusOK, map[string]any{"Delivered": false})
		return
	}

	wasDelivered := channel.DeliverReply(inboundMessage.ID, replyEnvelope{
		Kind:    inboundMessage.Type,
		Payload: inboundMessage.Payload,
	})
	if !wasDelivered {
		logWarn("respuesta sin destinatario ::", channelKey, ":: id", inboundMessage.ID, ":: tipo", inboundMessage.Type)
	} else if server.config.VerboseLogs {
		logInfo("respuesta entregada ::", channelKey, ":: id", inboundMessage.ID, ":: tipo", inboundMessage.Type)
	}
	writeJSON(responseWriter, http.StatusOK, map[string]any{"Delivered": wasDelivered})
}

// --- Backend-facing endpoints -------------------------------------------------

// publishRequest pushes one message to a tab without expecting an answer.
// WaitMs optionally waits for a reconnecting tab instead of dropping right away.
type publishRequest struct {
	Channel string
	Message json.RawMessage
	WaitMs  int
}

// rpcRequest pushes a command and blocks until the browser answers it. ID must
// match the id inside Message — the browser echoes it back on /in and that is
// what the bridge correlates.
type rpcRequest struct {
	Channel   string
	ID        uint64
	Message   json.RawMessage
	TimeoutMs int
	WaitMs    int
}

func (server *bridgeServer) handleServicePublish(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSONError(responseWriter, http.StatusMethodNotAllowed, "se esperaba POST")
		return
	}
	if authError := verifyServiceAuthRequest(request, server.config.SecretPhrase); authError != nil {
		logWarn("publish rechazado ::", authError)
		writeJSONError(responseWriter, http.StatusUnauthorized, authError.Error())
		return
	}

	publish := publishRequest{}
	if decodeError := json.NewDecoder(request.Body).Decode(&publish); decodeError != nil {
		writeJSONError(responseWriter, http.StatusBadRequest, "cuerpo inválido: "+decodeError.Error())
		return
	}
	channelKey, validationError := validateServiceTarget(publish.Channel, publish.Message)
	if validationError != nil {
		writeJSONError(responseWriter, http.StatusBadRequest, validationError.Error())
		return
	}

	channel := server.registry.AwaitChannel(request.Context(), channelKey, clampDuration(publish.WaitMs, 0, maxChannelWait))
	if channel == nil {
		// Not an error: with no buffering, a message for a disconnected tab is
		// dropped by contract and the backend only needs to know it happened.
		logWarn("publish sin canal conectado ::", channelKey)
		writeJSON(responseWriter, http.StatusOK, map[string]any{"Delivered": false})
		return
	}

	if sendError := channel.SendFrame(publish.Message); sendError != nil {
		logWarn("publish fallido ::", channelKey, "::", sendError)
		writeJSON(responseWriter, http.StatusOK, map[string]any{"Delivered": false, "Error": sendError.Error()})
		return
	}
	if server.config.VerboseLogs {
		logInfo("publish entregado ::", channelKey, "::", len(publish.Message), "bytes")
	}
	writeJSON(responseWriter, http.StatusOK, map[string]any{"Delivered": true})
}

func (server *bridgeServer) handleServiceRPC(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSONError(responseWriter, http.StatusMethodNotAllowed, "se esperaba POST")
		return
	}
	if authError := verifyServiceAuthRequest(request, server.config.SecretPhrase); authError != nil {
		logWarn("rpc rechazado ::", authError)
		writeJSONError(responseWriter, http.StatusUnauthorized, authError.Error())
		return
	}

	rpc := rpcRequest{}
	if decodeError := json.NewDecoder(request.Body).Decode(&rpc); decodeError != nil {
		writeJSONError(responseWriter, http.StatusBadRequest, "cuerpo inválido: "+decodeError.Error())
		return
	}
	channelKey, validationError := validateServiceTarget(rpc.Channel, rpc.Message)
	if validationError != nil {
		writeJSONError(responseWriter, http.StatusBadRequest, validationError.Error())
		return
	}
	if rpc.ID == 0 {
		writeJSONError(responseWriter, http.StatusBadRequest, "ID es obligatorio para correlacionar la respuesta")
		return
	}

	channel := server.registry.AwaitChannel(request.Context(), channelKey, clampDuration(rpc.WaitMs, 0, maxChannelWait))
	if channel == nil {
		logWarn("rpc sin canal conectado ::", channelKey, ":: id", rpc.ID)
		writeJSONError(responseWriter, http.StatusConflict, "no hay ningún cliente conectado para ese tab")
		return
	}

	// The waiter is registered *before* the command goes out, otherwise a very
	// fast browser could answer before there is anything to answer to.
	replyChannel := channel.AwaitReply(rpc.ID)
	defer channel.ReleasePendingReply(rpc.ID)

	if sendError := channel.SendFrame(rpc.Message); sendError != nil {
		logWarn("rpc no se pudo enviar ::", channelKey, ":: id", rpc.ID, "::", sendError)
		writeJSONError(responseWriter, http.StatusBadGateway, sendError.Error())
		return
	}

	replyTimeout := clampDuration(rpc.TimeoutMs, defaultRPCTimeout, maxRPCTimeout)
	replyTimer := time.NewTimer(replyTimeout)
	defer replyTimer.Stop()

	select {
	case replyEnvelope := <-replyChannel:
		if server.config.VerboseLogs {
			logInfo("rpc respondido ::", channelKey, ":: id", rpc.ID, ":: tipo", replyEnvelope.Kind)
		}
		writeJSON(responseWriter, http.StatusOK, map[string]any{
			"Kind":    replyEnvelope.Kind,
			"Payload": replyEnvelope.Payload,
		})

	case <-channel.closed:
		logWarn("rpc abortado: el cliente se desconectó ::", channelKey, ":: id", rpc.ID)
		writeJSONError(responseWriter, http.StatusConflict, "el cliente se desconectó antes de responder")

	case <-request.Context().Done():
		logWarn("rpc cancelado por el backend ::", channelKey, ":: id", rpc.ID)

	case <-replyTimer.C:
		logWarn("rpc expiró ::", channelKey, ":: id", rpc.ID, "::", replyTimeout)
		writeJSONError(responseWriter, http.StatusGatewayTimeout, "el cliente no respondió en "+replyTimeout.String())
	}
}

func (server *bridgeServer) handleHealth(responseWriter http.ResponseWriter, _ *http.Request) {
	writeJSON(responseWriter, http.StatusOK, map[string]any{
		"Ok":            true,
		"Channels":      server.registry.ConnectedChannelCount(),
		"UptimeSeconds": int64(time.Since(server.startedAt).Seconds()),
	})
}

// --- Shared helpers -----------------------------------------------------------

// validateServiceTarget checks a backend call's target. The channel token is
// decoded (and thereby proven canonical) before being used as a registry key,
// even though the backend is trusted: a malformed token would otherwise create
// a channel entry nobody can ever connect to.
func validateServiceTarget(channelToken string, message json.RawMessage) (string, error) {
	channelToken = strings.TrimSpace(channelToken)
	if len(channelToken) == 0 {
		return "", errors.New("Channel es obligatorio")
	}
	if _, _, _, decodeError := DecodeChannelToken(channelToken); decodeError != nil {
		return "", decodeError
	}
	if len(message) == 0 {
		return "", errors.New("Message es obligatorio")
	}
	return channelToken, nil
}

// clampDuration turns a millisecond field into a duration, applying a default
// when unset and never exceeding the ceiling the bridge is willing to hold.
func clampDuration(milliseconds int, whenUnset, maximum time.Duration) time.Duration {
	if milliseconds <= 0 {
		return whenUnset
	}
	requested := time.Duration(milliseconds) * time.Millisecond
	if requested > maximum {
		return maximum
	}
	return requested
}

// withClientCORS answers preflights and whitelists the Authorization header the
// browser must send — EventSource cannot set headers, so the frontend reads the
// stream with fetch() and needs the header allowed explicitly.
func withClientCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Access-Control-Allow-Origin", "*")
		responseWriter.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		responseWriter.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		responseWriter.Header().Set("Access-Control-Max-Age", "86400")
		if request.Method == http.MethodOptions {
			responseWriter.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(responseWriter, request)
	})
}

func writeJSON(responseWriter http.ResponseWriter, statusCode int, value any) {
	responseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	responseWriter.WriteHeader(statusCode)
	_ = json.NewEncoder(responseWriter).Encode(value)
}

func writeJSONError(responseWriter http.ResponseWriter, statusCode int, message string) {
	writeJSON(responseWriter, statusCode, map[string]any{"Error": message})
}
