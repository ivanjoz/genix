package server_utils

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// Opcode 0x01: [opcode][company:u24][user:u24][group:u8][cpu:u16][inference:u16][hmac:8].
	creditChargePayloadSize = 11

	creditBlockBytes    = 8 * 1024
	apiGroupSmallBytes  = 32 * 1024
	apiGroupMediumBytes = 256 * 1024
)

var ErrCreditLimiterMissing = errors.New("credit rate limiter is not configured")

// CreditLimitExceeded is the authenticated one-byte rejection returned by the Rust service.
type CreditLimitExceeded struct {
	Code      uint8
	Company   bool
	Window    string
	CPU       bool
	Inference bool
}

func (limit *CreditLimitExceeded) Error() string {
	scope := "user"
	if limit.Company {
		scope = "company"
	}
	return fmt.Sprintf("%s %s credit limit exhausted (code=%d)", scope, limit.Window, limit.Code)
}

// APIGroup assigns GET groups 0..2 and POST groups 3..5 from uncompressed payload bytes.
func APIGroup(method string, payloadBytes int) (uint8, error) {
	if payloadBytes < 0 {
		return 0, errors.New("payload size cannot be negative")
	}
	groupOffset := uint8(0)
	switch strings.ToUpper(method) {
	case "GET":
	case "POST":
		groupOffset = 3
	default:
		return 0, fmt.Errorf("credit rate limiting does not support method %q", method)
	}
	if payloadBytes < apiGroupSmallBytes {
		return groupOffset, nil
	}
	if payloadBytes <= apiGroupMediumBytes {
		return groupOffset + 1, nil
	}
	return groupOffset + 2, nil
}

// APICPUCredits applies the GET/POST base charge and rounds each partial extra block up.
func APICPUCredits(method string, payloadBytes int) (uint16, error) {
	if payloadBytes < 0 {
		return 0, errors.New("payload size cannot be negative")
	}
	baseCredits, extraBlockBytes := uint64(0), uint64(0)
	switch strings.ToUpper(method) {
	case "GET":
		baseCredits, extraBlockBytes = 2, 16*1024
	case "POST":
		baseCredits, extraBlockBytes = 5, 8*1024
	default:
		return 0, fmt.Errorf("credit rate limiting does not support method %q", method)
	}
	extraBytes := uint64(0)
	if payloadBytes > creditBlockBytes {
		extraBytes = uint64(payloadBytes - creditBlockBytes)
	}
	credits := baseCredits + ceilDivide(extraBytes, extraBlockBytes)
	if credits > uint64(^uint16(0)) {
		return 0, fmt.Errorf("calculated API credits %d exceed uint16", credits)
	}
	return uint16(credits), nil
}

// InferenceCredits charges one credit per input block and two per output block.
func InferenceCredits(inputBytes, outputBytes int) (uint16, error) {
	if inputBytes < 0 || outputBytes < 0 {
		return 0, errors.New("inference byte sizes cannot be negative")
	}
	credits := ceilDivide(uint64(inputBytes), creditBlockBytes) +
		2*ceilDivide(uint64(outputBytes), creditBlockBytes)
	if credits > uint64(^uint16(0)) {
		return 0, fmt.Errorf("calculated inference credits %d exceed uint16", credits)
	}
	return uint16(credits), nil
}

// ChargeAPIUsage calculates and submits one HTTP request/response CPU charge.
func ChargeAPIUsage(ctx context.Context, companyID, userID int32, method string, payloadBytes int) error {
	apiGroup, err := APIGroup(method, payloadBytes)
	if err != nil {
		return err
	}
	cpuCredits, err := APICPUCredits(method, payloadBytes)
	if err != nil {
		return err
	}
	return chargeConfiguredCredits(ctx, companyID, userID, apiGroup, cpuCredits, 0)
}

type creditRateLimitIdentity struct {
	companyID int32
	userID    int32
	apiGroup  uint8
}

type creditRateLimitIdentityKey struct{}

// WithCreditRateLimitIdentity attributes nested inference calls to the authenticated API request.
func WithCreditRateLimitIdentity(ctx context.Context, companyID, userID int32, apiGroup uint8) context.Context {
	return context.WithValue(ctx, creditRateLimitIdentityKey{}, creditRateLimitIdentity{
		companyID: companyID,
		userID:    userID,
		apiGroup:  apiGroup,
	})
}

// ChargeInferenceUsage submits actual successful provider input/output bytes when identity exists.
func ChargeInferenceUsage(ctx context.Context, inputBytes, outputBytes int) error {
	identity, ok := ctx.Value(creditRateLimitIdentityKey{}).(creditRateLimitIdentity)
	if !ok {
		return nil
	}
	inferenceCredits, err := InferenceCredits(inputBytes, outputBytes)
	if err != nil {
		return err
	}
	if inferenceCredits == 0 {
		return nil
	}
	return chargeConfiguredCredits(
		ctx, identity.companyID, identity.userID, identity.apiGroup, 0, inferenceCredits,
	)
}

// IsCreditRateLimitError identifies both quota exhaustion and fail-closed transport failures.
func IsCreditRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	var exceeded *CreditLimitExceeded
	return errors.As(err, &exceeded) || errors.Is(err, ErrCreditLimiterMissing) ||
		strings.Contains(err.Error(), "credit rate limiter")
}

// chargeConfiguredCredits fails open: only an authenticated CreditLimitExceeded
// violation blocks the caller. Any unavailability (unconfigured client, dial
// timeout, connection reset, etc.) is logged and treated as an allowed charge,
// since credit accounting must never take the API down with it.
func chargeConfiguredCredits(
	ctx context.Context,
	companyID, userID int32,
	apiGroup uint8,
	cpuCredits, inferenceCredits uint16,
) error {
	client := serverUtils()
	if client == nil {
		logLine("credit rate limiter not configured, allowing request::", ErrCreditLimiterMissing)
		return nil
	}
	err := client.Charge(ctx, companyID, userID, apiGroup, cpuCredits, inferenceCredits)
	if err == nil {
		return nil
	}
	var exceeded *CreditLimitExceeded
	if errors.As(err, &exceeded) {
		return err
	}
	logLine("credit rate limiter unavailable, allowing request::", err)
	return nil
}

// Charge sends one authenticated frame and returns nil only for status zero.
func (client *ServerUtilsClient) Charge(
	ctx context.Context,
	companyID, userID int32,
	apiGroup uint8,
	cpuCredits, inferenceCredits uint16,
) error {
	if companyID <= 0 || companyID > 0xFF_FFFF || userID <= 0 || userID > 0xFF_FFFF {
		return errors.New("company and user IDs must fit positive uint24")
	}
	if apiGroup > 5 {
		return fmt.Errorf("API group %d is outside 0..5", apiGroup)
	}
	if cpuCredits == 0 && inferenceCredits == 0 {
		return errors.New("at least one credit amount must be positive")
	}

	payload := make([]byte, creditChargePayloadSize)
	writeUint24(payload[0:3], uint32(companyID))
	writeUint24(payload[3:6], uint32(userID))
	payload[6] = apiGroup
	binary.BigEndian.PutUint16(payload[7:9], cpuCredits)
	binary.BigEndian.PutUint16(payload[9:11], inferenceCredits)

	// A charge is answered without queueing, so it needs no patience beyond the round trip.
	reply, _, err := client.request(ctx, opcodeChargeCredits, payload, chargeWait(ctx), 0, 0)
	if err != nil {
		return err
	}
	if reply.status == 0 {
		return nil
	}
	return decodeCreditLimitResponse(reply.status)
}

// chargeWait keeps a charge inside the caller's own deadline when it has one.
func chargeWait(ctx context.Context) time.Duration {
	wait := 3 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < wait {
			return remaining
		}
	}
	return wait
}

func decodeCreditLimitResponse(code uint8) error {
	windowCode := (code >> 1) & 0b11
	// 0xFF is the daemon saying it could not answer; anything else malformed is treated the same
	// way, as unavailability rather than as a verdict.
	if code&0b1110_0000 != 0 || windowCode == 3 || code&0b0001_1000 == 0 {
		return fmt.Errorf("%w: credit limiter returned status %d", ErrServerUtilsUnavailable, code)
	}
	windows := [...]string{"10 seconds", "1 hour", "24 hours"}
	return &CreditLimitExceeded{
		Code:      code,
		Company:   code&1 == 0,
		Window:    windows[windowCode],
		Inference: code&(1<<3) != 0,
		CPU:       code&(1<<4) != 0,
	}
}

func writeUint24(target []byte, value uint32) {
	target[0], target[1], target[2] = byte(value>>16), byte(value>>8), byte(value)
}

func ceilDivide(value, divisor uint64) uint64 {
	if value == 0 {
		return 0
	}
	return 1 + (value-1)/divisor
}

func connectionDeadline(ctx context.Context) time.Time {
	deadline := time.Now().Add(3 * time.Second)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		return contextDeadline
	}
	return deadline
}
