package server_utils

import (
	"encoding/binary"
	"strings"
	"testing"
)

func sampleRecord() RequestLogRecord {
	return RequestLogRecord{
		Date:      20_500,
		RequestID: 1_767_225_600_123,
		RouteID:   102,
		Frame:     41,
		CompanyID: 7,
		UserID:    42,
		ElapsedMs: 318,
		Errors: []RequestLogError{
			{ID: 1_234_567, Line: "responses.go:539", Text: "no se pudo obtener el registro"},
		},
	}
}

// These offsets are the wire contract with server_utils/src/reqlog/protocol.rs. Nothing at runtime
// notices when they drift — the daemon would simply parse different values out of the same bytes
// and write rows that look plausible and are wrong.
func TestEncodeRequestLogWireOffsets(t *testing.T) {
	payload, err := encodeRequestLog(sampleRecord())
	if err != nil {
		t.Fatal(err)
	}

	if got := int16(binary.BigEndian.Uint16(payload[0:2])); got != 20_500 {
		t.Errorf("date = %d at offset 0", got)
	}
	if got := int64(binary.BigEndian.Uint64(payload[2:10])); got != 1_767_225_600_123 {
		t.Errorf("request id = %d at offset 2", got)
	}
	if got := int16(binary.BigEndian.Uint16(payload[10:12])); got != 102 {
		t.Errorf("route id = %d at offset 10", got)
	}
	if payload[12] != 41 {
		t.Errorf("frame = %d at offset 12", payload[12])
	}
	if got := int32(payload[13])<<16 | int32(payload[14])<<8 | int32(payload[15]); got != 7 {
		t.Errorf("company = %d at offset 13", got)
	}
	if got := int32(binary.BigEndian.Uint32(payload[16:20])); got != 42 {
		t.Errorf("user = %d at offset 16", got)
	}
	if got := int16(binary.BigEndian.Uint16(payload[20:22])); got != 318 {
		t.Errorf("elapsed = %d at offset 20", got)
	}
	if payload[22] != 1 {
		t.Errorf("error count = %d at offset 22", payload[22])
	}

	// The error block: id, one-byte line length, line, two-byte text length, text.
	block := payload[requestLogHeaderSize:]
	if got := int32(binary.BigEndian.Uint32(block[0:4])); got != 1_234_567 {
		t.Errorf("error id = %d", got)
	}
	lineLength := int(block[4])
	if line := string(block[5 : 5+lineLength]); line != "responses.go:539" {
		t.Errorf("code line = %q", line)
	}
	textStart := 5 + lineLength
	textLength := int(binary.BigEndian.Uint16(block[textStart : textStart+2]))
	if text := string(block[textStart+2 : textStart+2+textLength]); text != "no se pudo obtener el registro" {
		t.Errorf("text = %q", text)
	}
	// The daemon refuses a payload with bytes left over, so the encoder must produce none.
	if consumed := requestLogHeaderSize + textStart + 2 + textLength; consumed != len(payload) {
		t.Errorf("payload is %d bytes but describes %d", len(payload), consumed)
	}
}

func TestEncodeRequestLogWithNoErrors(t *testing.T) {
	record := sampleRecord()
	record.Errors = nil
	payload, err := encodeRequestLog(record)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) != requestLogHeaderSize {
		t.Fatalf("an error-free record encoded to %d bytes, expected the %d-byte header",
			len(payload), requestLogHeaderSize)
	}
	if payload[22] != 0 {
		t.Fatalf("error count = %d, expected 0", payload[22])
	}
}

// Clamping rather than refusing: a row with four of five errors is worth far more than no row.
func TestEncodeRequestLogClampsInsteadOfFailing(t *testing.T) {
	record := sampleRecord()
	record.Errors = nil
	for index := range 9 {
		record.Errors = append(record.Errors, RequestLogError{
			ID:   int32(index),
			Line: strings.Repeat("x", requestLogMaxLineBytes*2),
			Text: strings.Repeat("y", requestLogMaxTextBytes*3),
		})
	}

	payload, err := encodeRequestLog(record)
	if err != nil {
		t.Fatal(err)
	}
	if payload[22] != requestLogMaxErrors {
		t.Fatalf("error count = %d, expected the cap of %d", payload[22], requestLogMaxErrors)
	}
	if len(payload) > requestLogMaxPayloadSize {
		t.Fatalf("payload is %d bytes, over the %d ceiling the daemon enforces",
			len(payload), requestLogMaxPayloadSize)
	}

	block := payload[requestLogHeaderSize:]
	if lineLength := int(block[4]); lineLength != requestLogMaxLineBytes {
		t.Fatalf("code line was not clamped: %d bytes", lineLength)
	}
	textLength := int(binary.BigEndian.Uint16(block[5+requestLogMaxLineBytes : 7+requestLogMaxLineBytes]))
	if textLength != requestLogMaxTextBytes {
		t.Fatalf("text was not clamped: %d bytes", textLength)
	}
}

// The daemon rejects the whole frame on invalid UTF-8, so a multi-byte rune landing on the
// truncation boundary must not be cut in half — one accented character would cost the entire row.
func TestEncodeRequestLogKeepsRunesWhole(t *testing.T) {
	record := sampleRecord()
	record.Errors = []RequestLogError{{
		ID:   1,
		Line: "responses.go:539",
		Text: strings.Repeat("á", requestLogMaxTextBytes),
	}}

	payload, err := encodeRequestLog(record)
	if err != nil {
		t.Fatal(err)
	}
	block := payload[requestLogHeaderSize:]
	textStart := 5 + int(block[4])
	textLength := int(binary.BigEndian.Uint16(block[textStart : textStart+2]))
	text := string(block[textStart+2 : textStart+2+textLength])

	if textLength > requestLogMaxTextBytes {
		t.Fatalf("text is %d bytes, over the ceiling", textLength)
	}
	if !strings.HasPrefix(record.Errors[0].Text, text) {
		t.Fatal("truncation produced something that is not a prefix of the original")
	}
	if strings.ContainsRune(text, '�') {
		t.Fatal("truncation split a rune")
	}
}

// The frame the daemon reads: opcode, a two-byte length covering only the payload, the payload,
// then the tag. The length is inside the signed bytes, so a peer cannot make the daemon buffer a
// different amount than the one that was authenticated.
func TestLengthPrefixedFrameLayout(t *testing.T) {
	nonce := [serverUtilsNonceSize]byte{1, 2, 3, 4, 5, 6, 7, 8}
	payload, err := encodeRequestLog(sampleRecord())
	if err != nil {
		t.Fatal(err)
	}

	frame := buildServerUtilsLengthPrefixedFrame([]byte("test-secret"), &nonce, 0, opcodeLogRequest, payload)

	if frame[0] != opcodeLogRequest {
		t.Fatalf("opcode = %#x, expected %#x", frame[0], opcodeLogRequest)
	}
	if declared := int(binary.BigEndian.Uint16(frame[1:3])); declared != len(payload) {
		t.Fatalf("declared length %d does not match the %d-byte payload", declared, len(payload))
	}
	if len(frame) != 1+2+len(payload)+serverUtilsAuthTagSize {
		t.Fatalf("frame is %d bytes; opcode + length + payload + tag is %d",
			len(frame), 1+2+len(payload)+serverUtilsAuthTagSize)
	}

	signed := frame[:len(frame)-serverUtilsAuthTagSize]
	expected := serverUtilsAuthTag([]byte("test-secret"), &nonce, 0, signed)
	if string(frame[len(frame)-serverUtilsAuthTagSize:]) != string(expected) {
		t.Fatal("the tag does not cover the opcode, length and payload")
	}
}
