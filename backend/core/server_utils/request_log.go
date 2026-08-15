package server_utils

import (
	"context"
	"encoding/binary"
	"errors"
)

// Opcode 0x04: the end-of-request record.
//
// The only variable-length frame on this port, and the only one the daemon never answers. Both
// follow from what it is for: a log row carries strings, and making a response wait for an
// acknowledgement that a log was stored would put the daemon's latency on the critical path of
// every request in the system. The client writes the frame and returns.
//
//	[opcode:1][length:u16][payload:length][hmac:8]
//
// The payload layout is mirrored in server_utils/src/reqlog/protocol.rs. Every field is
// big-endian, like the rest of this port.

const (
	// date i16, request id i64, route i16, frame u8, company u24, user i32, elapsed u16, errors u8.
	requestLogHeaderSize = 2 + 8 + 2 + 1 + 3 + 4 + 2 + 1

	// The daemon enforces the same three ceilings and refuses anything past them, so these are a
	// contract and not a local preference.
	requestLogMaxErrors    = 4
	requestLogMaxLineBytes = 64
	requestLogMaxTextBytes = 200

	requestLogMaxErrorBlockSize = 4 + 1 + requestLogMaxLineBytes + 2 + requestLogMaxTextBytes
	requestLogMaxPayloadSize    = requestLogHeaderSize + requestLogMaxErrors*requestLogMaxErrorBlockSize
)

// RequestLogError is one failing code line, already hashed by the caller.
type RequestLogError struct {
	ID   int32
	Line string
	Text string
}

// RequestLogRecord is everything one finished request contributes to user_logs.
type RequestLogRecord struct {
	Date      int16
	RequestID int64
	RouteID   int16
	Frame     uint8
	CompanyID int32
	UserID    int32
	ElapsedMs int16
	Errors    []RequestLogError
}

var (
	ErrRequestLogTooLarge = errors.New("request log payload exceeds the protocol ceiling")
	// ErrRequestLogNotConfigured means no daemon address was installed at startup — a local run
	// without server_utils, most often. Requests still work; they simply leave no row.
	ErrRequestLogNotConfigured = errors.New("server utils client is not configured")
)

// SendRequestLog writes one record and returns without waiting for anything.
//
// It reports an error only when the frame could not be written at all, and every caller ignores
// it beyond logging: a request that has already produced its response must not fail because its
// log row did not land.
func SendRequestLog(ctx context.Context, record RequestLogRecord) error {
	client := serverUtils()
	if client == nil {
		return ErrRequestLogNotConfigured
	}
	payload, err := encodeRequestLog(record)
	if err != nil {
		return err
	}
	return client.send(ctx, opcodeLogRequest, payload)
}

// encodeRequestLog builds the payload, clamping rather than refusing.
//
// A record that violates a ceiling is still worth writing without the part that violated it: an
// over-long preview truncated to 200 bytes still says what happened, and a fifth error dropped
// still leaves four. Refusing outright would throw away the row over its least important field.
func encodeRequestLog(record RequestLogRecord) ([]byte, error) {
	errorsToSend := record.Errors
	if len(errorsToSend) > requestLogMaxErrors {
		errorsToSend = errorsToSend[:requestLogMaxErrors]
	}

	payload := make([]byte, 0, requestLogHeaderSize+len(errorsToSend)*64)
	payload = binary.BigEndian.AppendUint16(payload, uint16(record.Date))
	payload = binary.BigEndian.AppendUint64(payload, uint64(record.RequestID))
	payload = binary.BigEndian.AppendUint16(payload, uint16(record.RouteID))
	payload = append(payload, record.Frame)
	// Three bytes, same width the charge opcode uses for a company.
	payload = append(payload,
		byte(record.CompanyID>>16), byte(record.CompanyID>>8), byte(record.CompanyID))
	payload = binary.BigEndian.AppendUint32(payload, uint32(record.UserID))
	payload = binary.BigEndian.AppendUint16(payload, uint16(record.ElapsedMs))
	payload = append(payload, byte(len(errorsToSend)))

	for _, requestError := range errorsToSend {
		payload = binary.BigEndian.AppendUint32(payload, uint32(requestError.ID))
		line := truncateUTF8(requestError.Line, requestLogMaxLineBytes)
		payload = append(payload, byte(len(line)))
		payload = append(payload, line...)
		text := truncateUTF8(requestError.Text, requestLogMaxTextBytes)
		payload = binary.BigEndian.AppendUint16(payload, uint16(len(text)))
		payload = append(payload, text...)
	}

	// Unreachable given the clamping above; kept because the daemon closes the connection on an
	// oversized declared length, and a silent framing bug here would take the charges and locks
	// on that connection down with it.
	if len(payload) > requestLogMaxPayloadSize {
		return nil, ErrRequestLogTooLarge
	}
	return payload, nil
}

// truncateUTF8 cuts to at most limit bytes without splitting a rune. A half rune would travel as
// invalid UTF-8 and the daemon would refuse the whole frame over one character.
func truncateUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	cut := limit
	// Continuation bytes are 10xxxxxx; back up to the start of the rune they belong to.
	for cut > 0 && value[cut]&0xC0 == 0x80 {
		cut--
	}
	return value[:cut]
}
