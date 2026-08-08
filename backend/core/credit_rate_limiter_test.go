package core

import (
	"context"
	"io"
	"net"
	"testing"
)

func TestAPIGroupsUseDocumentedBoundaries(t *testing.T) {
	tests := []struct {
		method string
		bytes  int
		want   uint8
	}{
		{"GET", 32*1024 - 1, 0}, {"GET", 32 * 1024, 1}, {"GET", 256 * 1024, 1},
		{"GET", 256*1024 + 1, 2}, {"POST", 0, 3}, {"POST", 32 * 1024, 4},
		{"POST", 256*1024 + 1, 5},
	}
	for _, test := range tests {
		got, err := APIGroup(test.method, test.bytes)
		if err != nil || got != test.want {
			t.Fatalf("APIGroup(%q, %d) = %d, %v; want %d", test.method, test.bytes, got, err, test.want)
		}
	}
}

func TestCreditFormulasRoundPartialBlocksUp(t *testing.T) {
	checks := []struct {
		method string
		bytes  int
		want   uint16
	}{
		{"GET", 0, 2}, {"GET", 8 * 1024, 2}, {"GET", 8*1024 + 1, 3},
		{"GET", 24 * 1024, 3}, {"GET", 24*1024 + 1, 4},
		{"POST", 0, 5}, {"POST", 8 * 1024, 5}, {"POST", 8*1024 + 1, 6},
		{"POST", 16 * 1024, 6}, {"POST", 16*1024 + 1, 7},
	}
	for _, check := range checks {
		got, err := APICPUCredits(check.method, check.bytes)
		if err != nil || got != check.want {
			t.Fatalf("APICPUCredits(%q, %d) = %d, %v; want %d", check.method, check.bytes, got, err, check.want)
		}
	}
	inference, err := InferenceCredits(8*1024+1, 8*1024+1)
	if err != nil || inference != 6 {
		t.Fatalf("InferenceCredits() = %d, %v; want 6", inference, err)
	}
}

func TestClientReusesNonceAndAdvancesSequence(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	secret := "test-secret"
	nonce := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	expectedFrames := [2][creditRateLimitFrameSize]byte{
		{0x12, 0x34, 0x56, 0x00, 0x00, 0x2A, 0x04, 0x00, 0x07, 0x00, 0x09, 0x37, 0x79, 0x1B, 0xC2, 0x18, 0x3B, 0x8F, 0xE8},
		{0x12, 0x34, 0x56, 0x00, 0x00, 0x2A, 0x04, 0x00, 0x07, 0x00, 0x09, 0x32, 0x8B, 0x6F, 0x7F, 0xCE, 0xD3, 0xC4, 0x07},
	}
	serverError := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverError <- acceptErr
			return
		}
		defer connection.Close()
		if _, writeErr := connection.Write(nonce[:]); writeErr != nil {
			serverError <- writeErr
			return
		}
		for sequence := uint64(0); sequence < 2; sequence++ {
			frame := [creditRateLimitFrameSize]byte{}
			if _, readErr := io.ReadFull(connection, frame[:]); readErr != nil {
				serverError <- readErr
				return
			}
			if frame != expectedFrames[sequence] {
				serverError <- io.ErrUnexpectedEOF
				return
			}
			if _, writeErr := connection.Write([]byte{0}); writeErr != nil {
				serverError <- writeErr
				return
			}
		}
		serverError <- nil
	}()

	client, err := NewCreditRateLimiterClient(listener.Addr().String(), secret)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for range 2 {
		if err := client.Charge(context.Background(), 0x12_3456, 42, 4, 7, 9); err != nil {
			t.Fatal(err)
		}
	}
	if err := <-serverError; err != nil {
		t.Fatal(err)
	}
}

func TestClientDecodesExhaustionBits(t *testing.T) {
	violation, err := decodeCreditLimitResponse(0b1_1011)
	if err != nil {
		t.Fatal(err)
	}
	if violation.Company || violation.Window != "1 hour" || !violation.CPU || !violation.Inference {
		t.Fatalf("unexpected violation: %+v", violation)
	}
}

func TestCreditLimitResponseUsesHTTP429AndRawCodeHeader(t *testing.T) {
	request := HandlerArgs{Route: "products"}
	response := request.MakeCreditRateLimitResponse(&CreditLimitExceeded{Code: 0b1_1011})
	if response.StatusCode != 429 || response.Headers["X-Rate-Limit-Code"] != "27" {
		t.Fatalf("unexpected handler response: %+v", response)
	}
	lambdaResponse := MakeErrRespFinal(int32(response.StatusCode), response.Error)
	if lambdaResponse.StatusCode != 429 {
		t.Fatalf("Lambda status = %d; want 429", lambdaResponse.StatusCode)
	}
}
