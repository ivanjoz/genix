//go:build !js

package core

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bytedance/sonic"
)

// The AWS response shapes. They live behind !js so the browser build never links aws-lambda-go:
// the router hands every transport the same *HandlerResponse, and only the Lambda entrypoints in
// main.go (and the agent's) turn it into one of these.

// MakeLambdaResponse renders a finished handler response for a Function URL in BUFFERED invoke mode,
// errors included.
func MakeLambdaResponse(handlerResponse *HandlerResponse) *events.APIGatewayV2HTTPResponse {
	if len(handlerResponse.Error) == 0 {
		return MakeResponseFinal(handlerResponse)
	}
	response := MakeErrRespFinal(errorStatusCode(handlerResponse), handlerResponse.Error)
	for headerName, headerValue := range handlerResponse.Headers {
		response.Headers[headerName] = headerValue
	}
	response.Headers["Access-Control-Expose-Headers"] = "X-Metadata, X-Rate-Limit-Code"
	response.Headers["X-Metadata"] = fmt.Sprintf("%d,%d",
		handlerResponse.PreSerializeMs,
		time.Now().UnixMilli()-handlerResponse.RequestStart,
	)
	return response
}

// MakeLambdaStreamingResponse is MakeLambdaResponse for InvokeMode RESPONSE_STREAM, where the body
// leaves as raw bytes behind a JSON prelude: a different response type, not a variation of the
// buffered one.
func MakeLambdaStreamingResponse(handlerResponse *HandlerResponse) *events.LambdaFunctionURLStreamingResponse {
	if len(handlerResponse.Error) == 0 {
		return MakeStreamingResponseFinal(handlerResponse)
	}
	response := MakeErrStreamingFinal(errorStatusCode(handlerResponse), handlerResponse.Error)
	for headerName, headerValue := range handlerResponse.Headers {
		response.Headers[headerName] = headerValue
	}
	response.Headers["X-Metadata"] = fmt.Sprintf("%d,%d",
		handlerResponse.PreSerializeMs,
		time.Now().UnixMilli()-handlerResponse.RequestStart,
	)
	return response
}

// errorStatusCode is the status of a failed response: 400 unless the handler chose one.
func errorStatusCode(handlerResponse *HandlerResponse) int32 {
	if handlerResponse.StatusCode != 0 {
		return int32(handlerResponse.StatusCode)
	}
	return 400
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
	encodeBase64Body := func(compressed []byte) {
		response.Body = base64.StdEncoding.EncodeToString(compressed)
		response.IsBase64Encoded = true
	}

	// zstd first, gzip as the compatibility fallback. isMaxLen forces compression even when the
	// client advertised nothing, because a payload that large cannot go out uncompressed.
	isCompressed := !handlerResponse.DisableCompression && (isMaxLen ||
		strings.Contains(handlerResponse.Encoding, "zstd") || strings.Contains(handlerResponse.Encoding, "gzip"))
	if handlerResponse.IsBinary && !isCompressed {
		// An uncompressed binary body must be base64'd too: as a plain string API Gateway would
		// corrupt every byte that is not valid UTF-8.
		encodeBase64Body(body)
	} else if handlerResponse.DisableCompression {
		// Preserve raw payloads for endpoints that require an unencoded response.
		response.Body = string(body)
	} else if strings.Contains(handlerResponse.Encoding, "zstd") {
		Log("Enviando respuesta comprimida con zstd")
		CompressZstdPooled(body, encodeBase64Body)
		response.Headers["Content-Encoding"] = "zstd"
	} else if isMaxLen || strings.Contains(handlerResponse.Encoding, "gzip") {
		Log("Enviando respuesta comprimida con gzip")
		if err := CompressGzipPooled(body, encodeBase64Body); err != nil {
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

func MakeErrRespFinal(statusCode int32, body string) *events.APIGatewayV2HTTPResponse {
	response := &events.APIGatewayV2HTTPResponse{}
	response.StatusCode = normalizedErrorStatus(statusCode)
	response.Headers = ensureResponseHeaders(nil)
	response.Headers["Content-Type"] = "application/json; charset=utf-8"
	errorMsg := ErrorMsg{Error: body}
	responseJSON, _ := sonic.Marshal(errorMsg)
	response.Body = string(responseJSON)
	return response
}
