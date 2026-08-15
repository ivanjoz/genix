package core

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/binary"
	"github.com/ivanjoz/minijson"
	// "encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bytedance/sonic"
)

type HandlerArgs struct {
	Body           *string
	ResponseWriter *http.ResponseWriter
	ReqContext     *http.Request
	Headers        map[string]string
	Query          map[string]string
	QueryString    string
	Method         string
	Route          string
	// ClientIP is the caller's address. Per request and never a global: in server mode one
	// process serves many requests at once, so a package-level field would be a data race and
	// would attribute one caller's address to another.
	ClientIP      string
	Authorization string
	ReqParams     string
	Encoding      string
	User          *UsuarioToken
	StartTime     int64
	// RequestID identifies this request in user_logs and in the log line, so a row read out of
	// ScyllaDB leads straight back to the entry that carries the full message. Per request and
	// never read from a global: in server mode one process serves many at once, and the Env mirror
	// that core.Log prints from is only accurate under Lambda, where invocations are serialized.
	RequestID int64
	// RouteID is the generated number of Method+"."+Route; zero means the path matched no handler.
	RouteID      int16
	accesosNivel []uint16
}

// ClientIPFromRequest resolves the caller's address behind the project's own Nginx.
//
// X-Forwarded-For is deliberately ignored: scripts/configure/configure_server.py sets it with
// $proxy_add_x_forwarded_for, which *appends*, so a client that sends its own header lands first
// in the list. Anything rate-limited by that value would be bypassable with one curl flag.
// X-Real-IP is written from $remote_addr and cannot be forged through the proxy.
func ClientIPFromRequest(request *http.Request) string {
	if realIP := strings.TrimSpace(request.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(request.RemoteAddr)
	}
	return host
}

// ClientIPKey packs the caller's address into the int64 a lock or a counter is keyed by.
//
// IPv6 is keyed by prefix rather than by address: a single residential customer is handed a
// whole /64, often a /56, so limiting per address would be free to bypass. The prefix is shifted
// one bit to stay in positive int64 range, which keys the /63 — still far narrower than any
// block a customer receives. Real IPv6 prefixes start at 2000::/3, so the result cannot collide
// with the IPv4 range that sits below 2^32.
func (req *HandlerArgs) ClientIPKey() (int64, bool) {
	parsed := net.ParseIP(strings.TrimSpace(req.ClientIP))
	if parsed == nil {
		return 0, false
	}
	if asIPv4 := parsed.To4(); asIPv4 != nil {
		return int64(binary.BigEndian.Uint32(asIPv4)), true
	}
	asIPv6 := parsed.To16()
	if asIPv6 == nil {
		return 0, false
	}
	return int64(binary.BigEndian.Uint64(asIPv6[:8]) >> 1), true
}

func makeAccesoNivelUint16(accesoID int32, nivel uint8) uint16 {
	// Pack acceso + nivel into the same sortable representation used by AccesosComputed.
	if nivel < 1 || nivel > 4 {
		nivel = 1
	}
	return uint16(accesoID<<2) | uint16(nivel-1)
}

func getAccesoNivelSearchRange(accesoID int32, nivel uint8) (uint16, uint16) {
	// Require granted levels to be >= the requested level while staying within the same access ID bucket.
	if nivel < 1 || nivel > 4 {
		nivel = 1
	}
	requiredPackedAccesoNivel := makeAccesoNivelUint16(accesoID, nivel)
	maxPackedAccesoNivel := makeAccesoNivelUint16(accesoID, 4)
	return requiredPackedAccesoNivel, maxPackedAccesoNivel
}

func hasPackedAccesoInRange(accesosNivel []uint16, rangeStart uint16, rangeEnd uint16) bool {
	// Use binary search over the sorted packed access slice instead of scanning every user access.
	searchStartIndex, foundExactStart := slices.BinarySearch(accesosNivel, rangeStart)
	if foundExactStart {
		return true
	} else {
		return searchStartIndex < len(accesosNivel) && accesosNivel[searchStartIndex] <= rangeEnd
	}
}

func DecompressBase64GzipM(base64String *string, isUrl ...bool) (string, error) {

	if len(isUrl) == 1 && isUrl[0] {
		str := *base64String
		str = strings.ReplaceAll(str, ".", "/")
		str = strings.ReplaceAll(str, "_", "=")
		str = strings.ReplaceAll(str, "-", "+")
		base64String = &str
	}

	// Decode the base64 string
	decodedBytes, err := base64.StdEncoding.DecodeString(*base64String)
	if err != nil {
		return "", errors.New("Error al decodificar: " + err.Error())
	}

	reader := bytes.NewReader(decodedBytes)
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return "", errors.New("Error creating gzip reader: " + err.Error())
	}

	defer gzipReader.Close()

	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		return "", errors.New("Error reading decompressed data: " + err.Error())
	}

	return string(decompressed), nil
}

func MakeResponseFinal(handlerResponse *HandlerResponse) *events.APIGatewayV2HTTPResponse {
	response := &events.APIGatewayV2HTTPResponse{}
	response.StatusCode = http.StatusOK
	response.Headers = ensureResponseHeaders(handlerResponse.Headers)
	if _, ok := response.Headers["Content-Type"]; !ok {
		response.Headers["Content-Type"] = "application/json; charset=utf-8"
	}

	body := *handlerResponse.Body
	handlerResponse.Body = nil

	isMaxLen := len(body) > 5*1000*1000
	Log("Len del body:: ", len(body))

	// API Gateway carries the body inline as a string, so a compressed payload has to be
	// base64'd. EncodeToString is the one copy that has to escape — it produces the string the
	// Lambda runtime serializes — which is why the compressor's buffer can go back to the pool
	// immediately after.
	encodeCompressedBody := func(compressed []byte) {
		response.Body = base64.StdEncoding.EncodeToString(compressed)
		response.IsBase64Encoded = true
	}

	// zstd first, gzip as the compatibility fallback. isMaxLen forces compression even when the
	// client advertised nothing, because a payload that large cannot go out uncompressed.
	if handlerResponse.DisableCompression {
		// Preserve raw payloads for endpoints that require an unencoded response.
		response.Body = string(body)
	} else if strings.Contains(handlerResponse.Encoding, "zstd") {
		Log("Enviando respuesta comprimida con zstd")
		CompressZstdPooled(body, encodeCompressedBody)
		response.Headers["Content-Encoding"] = "zstd"
	} else if isMaxLen || strings.Contains(handlerResponse.Encoding, "gzip") {
		Log("Enviando respuesta comprimida con gzip")
		if err := CompressGzipPooled(body, encodeCompressedBody); err != nil {
			panic("Error al momento de comprimir la respuesta con gzip: " + err.Error())
		}
		response.Headers["Content-Encoding"] = "gzip"
	} else {
		response.Body = string(body)
	}
	setMetadataHeader(response.Headers, handlerResponse.PreSerializeMs, time.Now().UnixMilli()-handlerResponse.RequestStart)
	return response
}

// MakeStreamingResponseFinal is the RESPONSE_STREAM counterpart of MakeResponseFinal, used when
// the Function URL is deployed with InvokeMode RESPONSE_STREAM.
//
// The runtime writes a JSON prelude of status and headers, then the body bytes verbatim. That
// removes the two copies the buffered path cannot avoid: the base64 expansion (+33%) and the
// runtime re-marshalling that base64 string into the response envelope. It also lifts the
// response ceiling from 6 MB to 20 MB.
//
// Note this does not make the body incremental — see MakeResponse; the keys header of the
// compact format is derived from the content, so nothing can be emitted until the whole payload
// has been walked. What streams here is an already-complete buffer.
func MakeStreamingResponseFinal(handlerResponse *HandlerResponse) *events.LambdaFunctionURLStreamingResponse {
	headers := ensureResponseHeaders(handlerResponse.Headers)
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json; charset=utf-8"
	}

	response := &events.LambdaFunctionURLStreamingResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
	}

	if handlerResponse.Body == nil {
		setMetadataHeader(headers, handlerResponse.PreSerializeMs, time.Now().UnixMilli()-handlerResponse.RequestStart)
		return response
	}

	body := *handlerResponse.Body
	handlerResponse.Body = nil

	// Same negotiation as the buffered path: zstd first, gzip as the fallback, and compression
	// forced past 5 MB even when the client advertised nothing.
	isMaxLen := len(body) > 5*1000*1000

	switch {
	case handlerResponse.DisableCompression:
		response.Body = bytes.NewReader(body)
	case strings.Contains(handlerResponse.Encoding, "zstd"):
		response.Body = CompressZstdReader(body)
		headers["Content-Encoding"] = "zstd"
	case isMaxLen || strings.Contains(handlerResponse.Encoding, "gzip"):
		compressedReader, err := CompressGzipReader(body)
		if err != nil {
			panic("Error al momento de comprimir la respuesta con gzip: " + err.Error())
		}
		response.Body = compressedReader
		headers["Content-Encoding"] = "gzip"
	default:
		response.Body = bytes.NewReader(body)
	}

	setMetadataHeader(headers, handlerResponse.PreSerializeMs, time.Now().UnixMilli()-handlerResponse.RequestStart)
	return response
}

// MakeErrStreamingFinal renders an error under RESPONSE_STREAM. Errors are small and never
// compressed, so the body is a plain reader over the JSON.
func MakeErrStreamingFinal(statusCode int32, message string) *events.LambdaFunctionURLStreamingResponse {
	headers := ensureResponseHeaders(nil)
	headers["Content-Type"] = "application/json; charset=utf-8"

	responseStatus := normalizedErrorStatus(statusCode)

	responseJSON, _ := sonic.Marshal(ErrorMsg{Error: message})
	return &events.LambdaFunctionURLStreamingResponse{
		StatusCode: responseStatus,
		Headers:    headers,
		Body:       bytes.NewReader(responseJSON),
	}
}

type ErrorMsg struct {
	Error string `json:"error"`
}

func MakeErrRespFinal(statusCode int32, body string) *events.APIGatewayV2HTTPResponse {
	response := &events.APIGatewayV2HTTPResponse{}
	response.StatusCode = normalizedErrorStatus(statusCode)
	response.Headers = ensureResponseHeaders(nil)
	// responseErr.Headers["Content-Type"] = "plain/text"
	response.Headers["Content-Type"] = "application/json; charset=utf-8"
	errorMsg := ErrorMsg{Error: body}
	responseJSON, _ := sonic.Marshal(errorMsg)
	response.Body = string(responseJSON)
	// Log("Error a enviar::", body)
	return response
}

func normalizedErrorStatus(statusCode int32) int {
	// Preserve deliberate 4xx/5xx responses such as rate-limit 429 and dependency 503.
	if statusCode >= 400 && statusCode <= 599 {
		return int(statusCode)
	}
	return http.StatusBadRequest
}

func (e HandlerArgs) HasAcceso(accesosIDs ...int32) bool {
	if e.User == nil || e.User.ID == 0 || len(e.accesosNivel) == 0 {
		return false
	}

	for _, accesoID := range accesosIDs {
		// Plain access checks require at least level 1 for the requested access ID.
		if e.HasAccesoNivel(accesoID, 1) {
			return true
		}
	}
	return false
}

func (e HandlerArgs) HasAccesoNivel(accesoID int32, nivel uint8) bool {
	if e.User == nil || e.User.ID == 0 || len(e.accesosNivel) == 0 || accesoID <= 0 {
		return false
	}

	rangeStart, rangeEnd := getAccesoNivelSearchRange(accesoID, nivel)
	return hasPackedAccesoInRange(e.accesosNivel, rangeStart, rangeEnd)
}

func (e HandlerArgs) IsUser(usuarioIDs ...int32) bool {
	usuariosIDsInclude := MakeSliceInclude(usuarioIDs)
	return usuariosIDsInclude.Include(e.User.ID)
}

func (e HandlerArgs) GetQueryInt64(key string) int64 {
	if strVar, ok := e.Query[key]; ok {
		value, err := strconv.ParseInt(strVar, 10, 64)
		if err == nil {
			return value
		}
	}
	return 0
}

func (e HandlerArgs) GetQueryInt16(key string) int16 {
	if strVar, ok := e.Query[key]; ok {
		value, err := strconv.ParseInt(strVar, 10, 16)
		if err == nil {
			return int16(value)
		}
	}
	return 0
}

func (e HandlerArgs) GetQueryInt(key string) int32 {
	if strVar, ok := e.Query[key]; ok {
		value, err := strconv.Atoi(strVar)
		if err == nil {
			return int32(value)
		}
	}
	return 0
}

// Obtiene un parámetro como un slice de enteros desde un string separado por comas
func (e HandlerArgs) GetQueryIntSliceBase(key string, sep string) []int32 {
	intSlice := []int32{}
	if strVar, ok := e.Query[key]; ok && !(strVar == "" || strVar == "0") {
		if strings.Contains(key, "aplicacion") {
			Log(key, strVar[:4], strVar)
		}

		if len(strVar) > 4 && strVar[:4] == "gz--" {
			strVar = strVar[4:]
			newStrVar, err := DecompressBase64GzipM(&strVar, true)
			if err == nil {
				strVar = newStrVar
			} else {
				Log("No se pudo descomprimir:: ", strVar)
			}
		}

		values := strings.Split(strVar, sep)
		for _, value := range values {
			if len(value) == 0 {
				continue
			}
			valueInt, err := strconv.Atoi(value)
			if err == nil {
				intSlice = append(intSlice, int32(valueInt))
			} else {
				Log("No es un número:: ", value)
			}
		}
	}
	return intSlice
}

func (e HandlerArgs) GetQueryIntSlice(key string) []int32 {
	return e.GetQueryIntSliceBase(key, ",")
}

func (e HandlerArgs) GetQueryIntSliceB64(key string) []int32 {
	content := e.GetQuery(key)
	if len(content) == 0 {
		return []int32{}
	}
	content = strings.ReplaceAll(content, "-", "+")
	content = strings.ReplaceAll(content, "_", "=")
	content = strings.ReplaceAll(content, ".", "/")
	decompressed, err := DecompressBase64GzipM(&content)
	if err != nil {
		Log(err)
		return []int32{}
	}

	intSlice := []int32{}
	for _, value := range strings.Split(decompressed, ",") {
		if len(value) == 0 || value == "0" {
			continue
		}
		valueInt, err := strconv.Atoi(value)
		if err == nil {
			intSlice = append(intSlice, int32(valueInt))
		}
	}
	return intSlice
}

func (e HandlerArgs) GetQueryIntPairsBase(key string, separator string) [][2]int32 {

	intsPairs := [][2]int32{}

	if strVar, ok := e.Query[key]; ok {
		values := strings.Split(strVar, ",")
		for _, value := range values {
			if !strings.Contains(value, separator) {
				Log("El string: " + value + " no contiene un ',' para poder separarlo")
				return intsPairs
			}
			valueInts := strings.Split(value, separator)

			intsPairs = append(intsPairs, [2]int32{SrtToInt32(valueInts[0]), SrtToInt32(valueInts[1])})
		}
	}
	return intsPairs
}

func (e HandlerArgs) GetQueryIntPairs(key string) [][2]int32 {
	return e.GetQueryIntPairsBase(key, ".")
}

func (e HandlerArgs) GetQueryInt64PairsBase(key string, separator string) [][2]int64 {

	intsPairs := [][2]int64{}

	if strVar, ok := e.Query[key]; ok {
		values := strings.Split(strVar, ",")
		for _, value := range values {
			if !strings.Contains(value, separator) {
				Log("El string: " + value + " no contiene un ',' para poder separarlo")
				return intsPairs
			}
			valueInts := strings.Split(value, separator)
			value1, err1 := strconv.ParseInt(valueInts[0], 10, 64)
			value2, err2 := strconv.ParseInt(valueInts[1], 10, 64)

			if err1 != nil || err2 != nil {
				Log("El valor " + valueInts[0] + " o el " + valueInts[1] + " no se pudieron convertir a int")
				return intsPairs
			}
			intsPairs = append(intsPairs, [2]int64{value1, value2})
		}
	}
	return intsPairs
}

func (e HandlerArgs) GetQueryInt64Pairs(key string) [][2]int64 {
	return e.GetQueryInt64PairsBase(key, ".")
}

// Obtiene un parámetro del query como un slice de 3 int32 (trio de enteros)
func (e HandlerArgs) GetQueryIntThreeBase(key string, separator string) [][3]int32 {
	intsPairs := [][3]int32{}

	if strVar, ok := e.Query[key]; ok {
		values := strings.Split(strVar, ",")
		for _, value := range values {
			if !strings.Contains(value, separator) {
				Log("El string: " + value + " no contiene un ',' para poder separarlo")
				return intsPairs
			}
			valueInts := strings.Split(value, separator)

			intsPairs = append(intsPairs, [3]int32{SrtToInt32(valueInts[0]), SrtToInt32(valueInts[1]), SrtToInt32(valueInts[2])})
		}
	}
	return intsPairs
}

func (e HandlerArgs) GetQueryIntThree(key string) [][3]int32 {
	return e.GetQueryIntThreeBase(key, ".")
}

// Obtiene un parámetro del query como un string
func (e HandlerArgs) GetQuery(key string) string {
	if strVar, ok := e.Query[key]; ok {
		return strVar
	}
	return ""
}

// Obtiene un parámetro del query como un slice de strings
func (e HandlerArgs) GetQuerySlice(key string) []string {
	if strVar, ok := e.Query[key]; ok {
		srtVarSlice := strings.Split(strVar, ",")
		srtVarSliceFiltered := []string{}
		for _, elm := range srtVarSlice {
			if len(elm) > 0 {
				srtVarSliceFiltered = append(srtVarSliceFiltered, elm)
			}
		}
		return srtVarSliceFiltered
	}
	return []string{}
}

type HandlerResponse struct {
	Body               *[]byte
	StatusCode         int
	Error              string
	Encoding           string
	Headers            map[string]string
	RequestStart       int64
	PreSerializeMs     int64
	Route              string
	StreamHandled      bool
	DisableCompression bool
}

type MainResponse struct {
	LambdaResponse *events.APIGatewayV2HTTPResponse
	// LambdaStreamingResponse is set instead of LambdaResponse when the Function URL is deployed
	// with InvokeMode RESPONSE_STREAM (Env.LAMBDA_RESPONSE_STREAMING). The two response shapes
	// are mutually exclusive: returning the wrong one for the deployed mode breaks every request.
	LambdaStreamingResponse *events.LambdaFunctionURLStreamingResponse
	Error                   error
}

// Every handler failure in this backend funnels through these four, which is what makes them the
// place to record one. Each resolves the code line itself and passes it down, rather than letting
// makeErrorResponse count frames: the depth differs depending on which wrapper was used, and a
// frame count that is right for one and wrong for the other blames this file instead of the
// handler.

func (req *HandlerArgs) MakeErrCode(message string, code int32) HandlerResponse {
	return req.makeErrorResponse(message, code, CallerCodeLine(1))
}

func (req *HandlerArgs) MakeErr(message ...any) HandlerResponse {
	return req.makeErrorResponse(Concat(" ", message...), 400, CallerCodeLine(1))
}

func (req *HandlerArgs) MakeErr401(message ...any) HandlerResponse {
	return req.makeErrorResponse(Concat(" ", message...), 401, CallerCodeLine(1))
}

func (req *HandlerArgs) MakeErr500(message ...any) HandlerResponse {
	return req.makeErrorResponse(Concat(" ", message...), 500, CallerCodeLine(1))
}

func (req *HandlerArgs) makeErrorResponse(message string, code int32, codeLine string) HandlerResponse {
	response := HandlerResponse{Headers: makeHeaders()}

	response.Error = message
	response.Route = req.Route
	RegisterRequestErrorAt(codeLine, message)
	// Deliberately not Log: the line above already recorded this failure against the handler, and
	// the heuristic would record it a second time against this file.
	logNoCapture("Req Error:: ", message)

	response.StatusCode = normalizedErrorStatus(code)
	return response
}

func makeHeaders() map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	}
	return headers
}

func ensureResponseHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		headers = map[string]string{}
	}

	// Browsers only expose custom headers cross-origin when the server whitelists them.
	headers["Access-Control-Expose-Headers"] = "X-Metadata, X-Rate-Limit-Code"
	return headers
}

func setMetadataHeader(headers map[string]string, preSerializeMs, totalMs int64) {
	if headers == nil {
		return
	}

	// Format: "<before-json-and-compression-ms>,<total-response-ms>".
	headers["X-Metadata"] = fmt.Sprintf("%d,%d", preSerializeMs, totalMs)
}

// Crea una respuesta serializando un struct
func (req *HandlerArgs) MakeResponse(respStruct any) HandlerResponse {
	return MakeResponse(req, &respStruct)
}

func MakeResponse[T any](req *HandlerArgs, respStruct *T) HandlerResponse {

	response := HandlerResponse{
		StatusCode: http.StatusOK,
		Headers:    makeHeaders(),
		Route:      req.Route,
	}

	// A string response is already the payload. Everything else goes through the compact
	// [keys, content] encoder that the frontend's unmarshal() expects — one path for every
	// size, so a large response can never silently fall back to a different wire format.
	if fmt.Sprintf("%T", *new(T)) == "string" {
		body := []byte(fmt.Sprintf("%v", *respStruct))
		response.Body = &body
		return response
	}

	bodyBytes, err := minijson.Marshal(respStruct)
	if err != nil {
		return req.MakeErr("No se pudo serializar respuesta:", err)
	}
	if bytes.Equal(bodyBytes, []byte("null")) {
		bodyBytes = []byte("[]")
	}
	response.Body = &bodyBytes

	return response
}

func (req *HandlerArgs) MakeResponsePlain(body *[]byte) HandlerResponse {
	response := HandlerResponse{
		Body:       body,
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/plain; charset=utf-8",
		},
	}

	return response
}

func SendLocalResponse(args HandlerArgs, response HandlerResponse) {
	respWriter := *args.ResponseWriter
	respWriter.Header().Set("Access-Control-Allow-Origin", "*")
	respWriter.Header().Set("Access-Control-Expose-Headers", "X-Metadata, X-Rate-Limit-Code")

	// Setea los headers de la respuesta
	if response.Headers != nil {
		for key, value := range response.Headers {
			respWriter.Header().Set(key, value)
		}
	}
	if len(respWriter.Header().Get("Content-Type")) == 0 {
		respWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	// Revisa si hay que enviar error
	if len(response.Error) > 0 {
		statusCode := http.StatusBadRequest
		if response.StatusCode != 0 {
			statusCode = response.StatusCode
		}
		elapsed := time.Now().UnixMilli() - args.StartTime
		respWriter.Header().Set("X-Metadata", fmt.Sprintf("%d,%d", response.PreSerializeMs, elapsed))
		respWriter.WriteHeader(statusCode)
		errorMap := map[string]string{
			"error": response.Error,
		}
		errorJson, _ := sonic.Marshal(errorMap)

		Log("enviando error:: ", response.Error)
		_, err := respWriter.Write(errorJson)
		if err != nil {
			Log("Hubo un error al enviar la respuesta::", err)
		}
		return
	}

	// On this path the bytes go straight to the socket, so unlike the Lambda path there is no
	// base64 step and nothing needs to escape. sendBody is invoked while the compressor's
	// pooled buffer is still borrowed, and sets every header just before the first Write —
	// which also keeps the elapsed timing inclusive of compression, as it was before pooling.
	sendBody := func(bodyToSend []byte, contentEncoding string) {
		if len(contentEncoding) > 0 {
			respWriter.Header().Set("Content-Encoding", contentEncoding)
		}

		elapsed := time.Now().UnixMilli() - args.StartTime
		respWriter.Header().Set("Server", fmt.Sprintf("Genix-v1.0:%v", elapsed))
		respWriter.Header().Set("X-Metadata", fmt.Sprintf("%d,%d", response.PreSerializeMs, elapsed))

		if _, err := respWriter.Write(bodyToSend); err != nil {
			Log("Hubo un error al enviar la respuesta::", err)
		}
	}

	// Envía respuesta ok: zstd first, gzip as the compatibility fallback.
	if response.Body == nil {
		Log("El body es nil!")
		sendBody(nil, "")
	} else if response.DisableCompression {
		// Write raw bytes without adding a Content-Encoding header.
		sendBody(*response.Body, "")
	} else if strings.Contains(args.Encoding, "zstd") {
		Log("Comprimiendo body con: zstd")
		CompressZstdPooled(*response.Body, func(compressed []byte) {
			sendBody(compressed, "zstd")
		})
	} else {
		Log("Comprimiendo body con: gzip")
		if err := CompressGzipPooled(*response.Body, func(compressed []byte) {
			sendBody(compressed, "gzip")
		}); err != nil {
			Log("Error al comprimir la respuesta con gzip::", err)
		}
	}
}

// SendServerEvent writes one SSE event to the current HTTP response stream.
// It is safe to call repeatedly (for example every second in a ticker loop).
func SendServerEvent(args *HandlerArgs, eventName string, payload any) error {
	if args == nil || args.ResponseWriter == nil {
		return errors.New("no se encontró ResponseWriter para enviar SSE")
	}

	respWriter := *args.ResponseWriter
	flusher, ok := respWriter.(http.Flusher)
	if !ok {
		return errors.New("el servidor no soporta streaming SSE (http.Flusher)")
	}

	// Set SSE headers once and keep them stable for the stream lifetime.
	if len(respWriter.Header().Get("Content-Type")) == 0 {
		respWriter.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	}
	respWriter.Header().Set("Cache-Control", "no-cache")
	respWriter.Header().Set("Connection", "keep-alive")
	respWriter.Header().Set("X-Accel-Buffering", "no")
	respWriter.Header().Set("Access-Control-Allow-Origin", "*")

	payloadJson, err := sonic.Marshal(payload)
	if err != nil {
		return errors.New("no se pudo serializar el payload SSE: " + err.Error())
	}

	// SSE format: optional event name, then one or more data lines, ending with blank line.
	var eventBuilder strings.Builder
	if len(eventName) > 0 {
		eventBuilder.WriteString("event: ")
		eventBuilder.WriteString(eventName)
		eventBuilder.WriteString("\n")
	}
	eventBuilder.WriteString("data: ")
	eventBuilder.Write(payloadJson)
	eventBuilder.WriteString("\n\n")

	if _, err := respWriter.Write([]byte(eventBuilder.String())); err != nil {
		return errors.New("error al escribir el evento SSE: " + err.Error())
	}

	flusher.Flush()
	Log("SSE enviado | route:", args.Route, " | event:", eventName, " | bytes:", len(payloadJson))
	return nil
}

// SendServerComment writes an SSE comment line, useful for lightweight keepalive heartbeats.
func SendServerComment(args *HandlerArgs, commentBody string) error {
	if args == nil || args.ResponseWriter == nil {
		return errors.New("no se encontró ResponseWriter para enviar comentario SSE")
	}

	respWriter := *args.ResponseWriter
	flusher, ok := respWriter.(http.Flusher)
	if !ok {
		return errors.New("el servidor no soporta streaming SSE (http.Flusher)")
	}

	if len(respWriter.Header().Get("Content-Type")) == 0 {
		respWriter.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	}
	respWriter.Header().Set("Cache-Control", "no-cache")
	respWriter.Header().Set("Connection", "keep-alive")
	respWriter.Header().Set("X-Accel-Buffering", "no")
	respWriter.Header().Set("Access-Control-Allow-Origin", "*")

	cleanCommentBody := strings.ReplaceAll(commentBody, "\n", " ")
	if _, err := respWriter.Write([]byte(": " + cleanCommentBody + "\n\n")); err != nil {
		return errors.New("error al escribir comentario SSE: " + err.Error())
	}

	flusher.Flush()
	return nil
}

// TODO: deprecar después
type MergedRoute struct {
	Id       int32             `json:"id"`
	FuncPath string            `json:"funcPath"`
	Route    string            `json:"route"`
	Query    map[string]string `json:"query"`
}

type MergedResponse struct {
	Route      string `json:"route"`
	Id         int32  `json:"id"`
	Body       string `json:"body"`
	StatusCode int32  `json:"statusCode"`
	Message    string `json:"message"`
}

func ParseMergedUri(query map[string]string) []MergedRoute {
	mapOfRoutes := map[int]MergedRoute{}
	mapOfRoutesValues := map[string]string{}

	for key, value := range query {
		// si es una ruta
		if key[0:3] == "i--" {
			routeID, err := strconv.Atoi(key[3:])
			if err != nil {
				Log("Hubo un error al convertir ", key[3:], " en int")
			}
			mapOfRoutes[routeID] = MergedRoute{
				Id:       int32(routeID),
				Route:    value,
				FuncPath: "GET." + value,
				Query:    map[string]string{},
			}
		} else if strings.Contains(key, "--") {
			mapOfRoutesValues[key] = value
		}
	}

	for key, value := range mapOfRoutesValues {
		routeIDParamName := strings.Split(key, "--")
		routeID, err := strconv.Atoi(routeIDParamName[0])
		if err != nil {
			Log("Hubo un error al convertir ", key[3:], " en int")
		}
		mergedRoute := mapOfRoutes[routeID]
		paramName := routeIDParamName[1]
		mergedRoute.Query[paramName] = value
	}

	mergedRoutes := []MergedRoute{}
	for _, value := range mapOfRoutes {
		mergedRoutes = append(mergedRoutes, value)
	}

	return mergedRoutes
}
