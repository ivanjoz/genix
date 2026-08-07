package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ivanjoz/colbin"
)

const testApiKey = "K1OzWIN0yarCc9ge"

// makeTestSessionToken builds a token exactly as security.MakeUsuarioResponse
// does (colbin payload whose Hash is an HMAC over the other fields), so the
// tests exercise the real wire format instead of a stand-in.
func makeTestSessionToken(t *testing.T, companyID, userID int32) string {
	t.Helper()

	sessionToken := UserToken{CompanyID: companyID, ID: userID, Created: 1234, User: "tester"}
	sessionToken.Hash = computeUserTokenHash(sessionToken, testApiKey)

	encodedToken, marshalError := colbin.Marshal(sessionToken)
	if marshalError != nil {
		t.Fatalf("no se pudo serializar el token de prueba: %v", marshalError)
	}
	return base64.StdEncoding.EncodeToString(encodedToken)
}

// makeTestChannel builds the channel token that names one tab.
func makeTestChannel(t *testing.T, companyID, userID int32, tabID string) string {
	t.Helper()

	channelToken := EncodeChannelToken(companyID, userID, tabID)
	if len(channelToken) == 0 {
		t.Fatalf("no se pudo codificar el canal %d/%d/%s", companyID, userID, tabID)
	}
	return channelToken
}

func TestUserTokenAuthentication(t *testing.T) {
	validToken := makeTestSessionToken(t, 7, 42)

	request := httptest.NewRequest(http.MethodGet, "/sse?tab=abc", nil)
	request.Header.Set("Authorization", "Bearer "+validToken)

	authenticatedToken, authError := authenticateUserRequest(request, testApiKey)
	if authError != nil {
		t.Fatalf("un token válido fue rechazado: %v", authError)
	}
	if authenticatedToken.CompanyID != 7 || authenticatedToken.ID != 42 {
		t.Fatalf("identidad incorrecta: company=%d user=%d", authenticatedToken.CompanyID, authenticatedToken.ID)
	}

	// The same token under a different secret must not authenticate: this is what
	// stops a client from minting its own identity.
	if _, wrongSecretError := authenticateUserRequest(request, "otro-secreto"); wrongSecretError == nil {
		t.Fatal("un token firmado con otro secreto fue aceptado")
	}

	requestWithoutHeader := httptest.NewRequest(http.MethodGet, "/sse?tab=abc", nil)
	if _, missingHeaderError := authenticateUserRequest(requestWithoutHeader, testApiKey); missingHeaderError == nil {
		t.Fatal("una petición sin Authorization fue aceptada")
	}
}

func TestServiceAuthentication(t *testing.T) {
	nowUnix := time.Now().Unix()

	freshRequest := httptest.NewRequest(http.MethodPost, "/publish", nil)
	freshRequest.Header.Set(serviceAuthHeaderName, MakeServiceAuthHeader(testApiKey, nowUnix))
	if authError := verifyServiceAuthRequest(freshRequest, testApiKey); authError != nil {
		t.Fatalf("una firma de servicio válida fue rechazada: %v", authError)
	}

	expiredRequest := httptest.NewRequest(http.MethodPost, "/publish", nil)
	expiredRequest.Header.Set(serviceAuthHeaderName, MakeServiceAuthHeader(testApiKey, nowUnix-serviceAuthMaxSkewSeconds-60))
	if authError := verifyServiceAuthRequest(expiredRequest, testApiKey); authError == nil {
		t.Fatal("una firma de servicio expirada fue aceptada")
	}

	tamperedRequest := httptest.NewRequest(http.MethodPost, "/publish", nil)
	tamperedHeader := MakeServiceAuthHeader(testApiKey, nowUnix)
	tamperedRequest.Header.Set(serviceAuthHeaderName, tamperedHeader[:len(tamperedHeader)-2]+"ff")
	if authError := verifyServiceAuthRequest(tamperedRequest, testApiKey); authError == nil {
		t.Fatal("una firma de servicio alterada fue aceptada")
	}
}

// testStream is a connected browser tab under test: it reads SSE frames off the
// response body into a channel, exactly as the frontend client does.
type testStream struct {
	frames   chan map[string]any
	response *http.Response
}

func openTestStream(t *testing.T, baseURL, sessionToken, channelToken string) *testStream {
	t.Helper()

	streamRequest, _ := http.NewRequest(http.MethodGet, baseURL+"/sse?ch="+channelToken, nil)
	streamRequest.Header.Set("Authorization", "Bearer "+sessionToken)

	streamResponse, requestError := http.DefaultClient.Do(streamRequest)
	if requestError != nil {
		t.Fatalf("no se pudo abrir el stream: %v", requestError)
	}
	if streamResponse.StatusCode != http.StatusOK {
		t.Fatalf("el stream respondió %d", streamResponse.StatusCode)
	}

	stream := &testStream{frames: make(chan map[string]any, 16), response: streamResponse}
	go func() {
		defer close(stream.frames)
		bodyScanner := bufio.NewScanner(streamResponse.Body)
		for bodyScanner.Scan() {
			line := bodyScanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			decodedFrame := map[string]any{}
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &decodedFrame) == nil {
				stream.frames <- decodedFrame
			}
		}
	}()

	// The handshake frame proves the channel is registered; every test needs to
	// wait for it before publishing, just like the real client does.
	if handshake := stream.next(t); handshake["Type"] != "bridgeReady" {
		t.Fatalf("se esperaba el handshake bridgeReady, llegó: %v", handshake)
	}
	return stream
}

func (stream *testStream) next(t *testing.T) map[string]any {
	t.Helper()
	select {
	case frame, isOpen := <-stream.frames:
		if !isOpen {
			t.Fatal("el stream se cerró antes de recibir el frame esperado")
		}
		return frame
	case <-time.After(3 * time.Second):
		t.Fatal("timeout esperando un frame del stream")
		return nil
	}
}

func (stream *testStream) close() { _ = stream.response.Body.Close() }

// postService issues one authenticated backend→bridge call.
func postService(t *testing.T, baseURL, path string, body any) (int, map[string]json.RawMessage) {
	t.Helper()

	encodedBody, _ := json.Marshal(body)
	serviceRequest, _ := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(encodedBody))
	serviceRequest.Header.Set(serviceAuthHeaderName, MakeServiceAuthHeader(testApiKey, time.Now().Unix()))

	serviceResponse, requestError := http.DefaultClient.Do(serviceRequest)
	if requestError != nil {
		t.Fatalf("la llamada de servicio a %s falló: %v", path, requestError)
	}
	defer serviceResponse.Body.Close()

	decodedResponse := map[string]json.RawMessage{}
	_ = json.NewDecoder(serviceResponse.Body).Decode(&decodedResponse)
	return serviceResponse.StatusCode, decodedResponse
}

func startTestBridge(t *testing.T) string {
	t.Helper()
	testServer := httptest.NewServer(newBridgeServer(BridgeConfig{ApiKey: testApiKey}).Routes())
	t.Cleanup(testServer.Close)
	return testServer.URL
}

func TestPublishReachesTheConnectedTab(t *testing.T) {
	baseURL := startTestBridge(t)
	sessionToken := makeTestSessionToken(t, 7, 42)

	stream := openTestStream(t, baseURL, sessionToken, makeTestChannel(t, 7, 42, testTabID))
	defer stream.close()

	statusCode, publishResponse := postService(t, baseURL, "/publish", publishRequest{
		Channel: makeTestChannel(t, 7, 42, testTabID),
		Message: json.RawMessage(`{"Type":"agentStatus","Payload":{"State":"thinking"}}`),
	})
	if statusCode != http.StatusOK || string(publishResponse["Delivered"]) != "true" {
		t.Fatalf("publish no entregó el mensaje: status=%d body=%v", statusCode, publishResponse)
	}

	if deliveredFrame := stream.next(t); deliveredFrame["Type"] != "agentStatus" {
		t.Fatalf("llegó un frame inesperado: %v", deliveredFrame)
	}
}

func TestPublishToDisconnectedTabIsDropped(t *testing.T) {
	baseURL := startTestBridge(t)

	statusCode, publishResponse := postService(t, baseURL, "/publish", publishRequest{
		Channel: makeTestChannel(t, 7, 42, "YXVzZW50ZQ"[:8]),
		Message: json.RawMessage(`{"Type":"agentStatus"}`),
	})
	// Dropping is the contract, so this is a 200 that reports non-delivery — not
	// an error the backend has to handle.
	if statusCode != http.StatusOK || string(publishResponse["Delivered"]) != "false" {
		t.Fatalf("se esperaba Delivered:false, llegó status=%d body=%v", statusCode, publishResponse)
	}
}

func TestPublishFromAnotherTenantCannotReachTheTab(t *testing.T) {
	baseURL := startTestBridge(t)
	sessionToken := makeTestSessionToken(t, 7, 42)

	stream := openTestStream(t, baseURL, sessionToken, makeTestChannel(t, 7, 42, testTabID))
	defer stream.close()

	// Same tab id, different company: the channel key differs, so there is
	// nothing to deliver to.
	_, publishResponse := postService(t, baseURL, "/publish", publishRequest{
		Channel: makeTestChannel(t, 8, 42, testTabID),
		Message: json.RawMessage(`{"Type":"agentStatus"}`),
	})
	if string(publishResponse["Delivered"]) != "false" {
		t.Fatalf("un publish de otra company alcanzó el tab: %v", publishResponse)
	}
}

// The channel token is an identifier, not a credential: editing the company id
// inside it must not open another tenant's channel. The session token is the
// proof of identity and the two have to agree.
func TestClientCannotOpenAChannelOfAnotherIdentity(t *testing.T) {
	baseURL := startTestBridge(t)
	sessionToken := makeTestSessionToken(t, 7, 42)

	for caseName, forgedChannel := range map[string]string{
		"otra company": makeTestChannel(t, 8, 42, testTabID),
		"otro usuario": makeTestChannel(t, 7, 43, testTabID),
	} {
		streamRequest, _ := http.NewRequest(http.MethodGet, baseURL+"/sse?ch="+forgedChannel, nil)
		streamRequest.Header.Set("Authorization", "Bearer "+sessionToken)

		streamResponse, requestError := http.DefaultClient.Do(streamRequest)
		if requestError != nil {
			t.Fatalf("la petición falló (%s): %v", caseName, requestError)
		}
		streamResponse.Body.Close()

		if streamResponse.StatusCode != http.StatusUnauthorized {
			t.Fatalf("se abrió un canal ajeno (%s): status %d", caseName, streamResponse.StatusCode)
		}
	}
}

func TestRPCCorrelatesTheBrowserReply(t *testing.T) {
	baseURL := startTestBridge(t)
	sessionToken := makeTestSessionToken(t, 7, 42)

	stream := openTestStream(t, baseURL, sessionToken, makeTestChannel(t, 7, 42, testTabID))
	defer stream.close()

	type rpcOutcome struct {
		statusCode int
		body       map[string]json.RawMessage
	}
	rpcResult := make(chan rpcOutcome, 1)
	go func() {
		statusCode, body := postService(t, baseURL, "/rpc", rpcRequest{
			Channel: makeTestChannel(t, 7, 42, testTabID), ID: 9,
			Message:   json.RawMessage(`{"ID":9,"Type":"navigate","Payload":{"Route":"/negocio/productos"}}`),
			TimeoutMs: 3000,
		})
		rpcResult <- rpcOutcome{statusCode, body}
	}()

	commandFrame := stream.next(t)
	if commandFrame["Type"] != "navigate" || commandFrame["ID"].(float64) != 9 {
		t.Fatalf("el comando no llegó al navegador: %v", commandFrame)
	}

	// The browser answers on its own short request, which is what unblocks /rpc.
	replyBody, _ := json.Marshal(clientInboundMessage{
		ID: 9, Type: "result", Payload: json.RawMessage(`{"Route":"/negocio/productos"}`),
	})
	replyRequest, _ := http.NewRequest(http.MethodPost, baseURL+"/in?ch="+makeTestChannel(t, 7, 42, testTabID), bytes.NewReader(replyBody))
	replyRequest.Header.Set("Authorization", "Bearer "+sessionToken)
	replyResponse, replyError := http.DefaultClient.Do(replyRequest)
	if replyError != nil {
		t.Fatalf("la respuesta del navegador falló: %v", replyError)
	}
	replyResponse.Body.Close()

	select {
	case outcome := <-rpcResult:
		if outcome.statusCode != http.StatusOK {
			t.Fatalf("rpc respondió %d: %v", outcome.statusCode, outcome.body)
		}
		if string(outcome.body["Kind"]) != `"result"` {
			t.Fatalf("rpc devolvió un Kind inesperado: %v", outcome.body)
		}
		if !strings.Contains(string(outcome.body["Payload"]), "/negocio/productos") {
			t.Fatalf("rpc devolvió un payload inesperado: %v", outcome.body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("rpc no retornó tras la respuesta del navegador")
	}
}

func TestRPCWithoutConnectedTabFails(t *testing.T) {
	baseURL := startTestBridge(t)

	statusCode, _ := postService(t, baseURL, "/rpc", rpcRequest{
		Channel: makeTestChannel(t, 7, 42, "YXVzZW50ZQ"[:8]), ID: 1,
		Message: json.RawMessage(`{"ID":1,"Type":"navigate"}`),
	})
	if statusCode != http.StatusConflict {
		t.Fatalf("se esperaba 409 sin cliente conectado, llegó %d", statusCode)
	}
}

func TestRPCTimesOutWhenTheBrowserStaysSilent(t *testing.T) {
	baseURL := startTestBridge(t)
	sessionToken := makeTestSessionToken(t, 7, 42)

	stream := openTestStream(t, baseURL, sessionToken, makeTestChannel(t, 7, 42, testTabID))
	defer stream.close()

	statusCode, _ := postService(t, baseURL, "/rpc", rpcRequest{
		Channel: makeTestChannel(t, 7, 42, testTabID), ID: 5,
		Message:   json.RawMessage(`{"ID":5,"Type":"getMenu"}`),
		TimeoutMs: 200,
	})
	if statusCode != http.StatusGatewayTimeout {
		t.Fatalf("se esperaba 504 por timeout, llegó %d", statusCode)
	}
}

func TestReconnectReplacesThePreviousStream(t *testing.T) {
	baseURL := startTestBridge(t)
	sessionToken := makeTestSessionToken(t, 7, 42)

	firstStream := openTestStream(t, baseURL, sessionToken, makeTestChannel(t, 7, 42, testTabID))
	defer firstStream.close()

	secondStream := openTestStream(t, baseURL, sessionToken, makeTestChannel(t, 7, 42, testTabID))
	defer secondStream.close()

	// The evicted stream is told why it ended so a duplicated tab can rotate its
	// id instead of the two fighting over it.
	if replacedFrame := firstStream.next(t); replacedFrame["Type"] != "replaced" {
		t.Fatalf("el stream reemplazado no fue notificado: %v", replacedFrame)
	}

	_, publishResponse := postService(t, baseURL, "/publish", publishRequest{
		Channel: makeTestChannel(t, 7, 42, testTabID),
		Message: json.RawMessage(`{"Type":"agentStatus"}`),
	})
	if string(publishResponse["Delivered"]) != "true" {
		t.Fatalf("el stream nuevo no recibió el mensaje: %v", publishResponse)
	}
	if deliveredFrame := secondStream.next(t); deliveredFrame["Type"] != "agentStatus" {
		t.Fatalf("frame inesperado en el stream nuevo: %v", deliveredFrame)
	}
}

func TestPublishWaitsForAReconnectingTab(t *testing.T) {
	baseURL := startTestBridge(t)
	sessionToken := makeTestSessionToken(t, 7, 42)

	// The tab connects only after the publish is already waiting — the safety net
	// behind the client handshake.
	lateStream := make(chan *testStream, 1)
	go func() {
		time.Sleep(150 * time.Millisecond)
		lateStream <- openTestStream(t, baseURL, sessionToken, makeTestChannel(t, 7, 42, "dGFyZGlv"))
	}()
	// The stream has to be closed before the test server shuts down: an open SSE
	// handler keeps httptest.Server.Close blocking forever.
	defer func() { (<-lateStream).close() }()

	_, publishResponse := postService(t, baseURL, "/publish", publishRequest{
		Channel: makeTestChannel(t, 7, 42, "dGFyZGlv"),
		Message: json.RawMessage(`{"Type":"agentStatus"}`),
		WaitMs:  3000,
	})
	if string(publishResponse["Delivered"]) != "true" {
		t.Fatalf("publish no esperó al tab que reconectaba: %v", publishResponse)
	}
}

func TestUnauthenticatedClientIsRejected(t *testing.T) {
	baseURL := startTestBridge(t)

	streamResponse, requestError := http.Get(baseURL + "/sse?ch=" + makeTestChannel(t, 7, 42, testTabID))
	if requestError != nil {
		t.Fatalf("la petición falló: %v", requestError)
	}
	defer streamResponse.Body.Close()

	if streamResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("se esperaba 401 sin token, llegó %d", streamResponse.StatusCode)
	}
}

func TestUnauthenticatedServiceCallIsRejected(t *testing.T) {
	baseURL := startTestBridge(t)

	encodedBody, _ := json.Marshal(publishRequest{Channel: makeTestChannel(t, 7, 42, testTabID), Message: json.RawMessage(`{}`)})
	publishResponse, requestError := http.Post(baseURL+"/publish", "application/json", bytes.NewReader(encodedBody))
	if requestError != nil {
		t.Fatalf("la petición falló: %v", requestError)
	}
	defer publishResponse.Body.Close()

	if publishResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("se esperaba 401 sin firma de servicio, llegó %d", publishResponse.StatusCode)
	}
}
