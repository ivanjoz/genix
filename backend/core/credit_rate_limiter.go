package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	creditRateLimitFrameSize   = 19
	creditRateLimitPayloadSize = 11
	creditRateLimitNonceSize   = 8
	creditRateLimitDomain      = "genix-rate-limiter:v1"
	creditBlockBytes           = 8 * 1024
	apiGroupSmallBytes         = 32 * 1024
	apiGroupMediumBytes        = 256 * 1024
)

var (
	configuredCreditLimiterMu sync.RWMutex
	configuredCreditLimiter   *CreditRateLimiterClient
	ErrCreditLimiterMissing   = errors.New("credit rate limiter is not configured")
)

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

// CreditRateLimiterClient serializes frames on one persistent sequence-bound TCP connection.
type CreditRateLimiterClient struct {
	address  string
	secret   []byte
	mu       sync.Mutex
	conn     net.Conn
	nonce    [creditRateLimitNonceSize]byte
	sequence uint64
}

// NewCreditRateLimiterClient validates immutable connection settings without dialing eagerly.
func NewCreditRateLimiterClient(address, secret string) (*CreditRateLimiterClient, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, errors.New("rate_limit.address is required")
	}
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("secret_phrase is required by the credit rate limiter")
	}
	return &CreditRateLimiterClient{address: address, secret: []byte(secret)}, nil
}

// ConfigureCreditRateLimiter installs the process-wide client used by API and inference calls.
func ConfigureCreditRateLimiter(address, secret string) error {
	client, err := NewCreditRateLimiterClient(address, secret)
	if err != nil {
		return err
	}
	configuredCreditLimiterMu.Lock()
	previousClient := configuredCreditLimiter
	configuredCreditLimiter = client
	configuredCreditLimiterMu.Unlock()
	if previousClient != nil {
		previousClient.Close()
	}
	return nil
}

// Close drops the current connection; the next charge reconnects with a fresh nonce and sequence.
func (client *CreditRateLimiterClient) Close() {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.closeConnection()
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

// MakeCreditRateLimitResponse maps the TCP decision to a stable HTTP error and diagnostic header.
func (req *HandlerArgs) MakeCreditRateLimitResponse(err error) HandlerResponse {
	var exceeded *CreditLimitExceeded
	if errors.As(err, &exceeded) {
		response := req.MakeErrCode("Límite de créditos agotado.", 429)
		response.Headers["X-Rate-Limit-Code"] = fmt.Sprint(exceeded.Code)
		return response
	}
	Log("credit rate limiter unavailable::", err)
	return req.MakeErrCode("El servicio de límites de crédito no está disponible.", 503)
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
	configuredCreditLimiterMu.RLock()
	client := configuredCreditLimiter
	configuredCreditLimiterMu.RUnlock()
	if client == nil {
		Log("credit rate limiter not configured, allowing request::", ErrCreditLimiterMissing)
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
	Log("credit rate limiter unavailable, allowing request::", err)
	return nil
}

// Charge sends one authenticated frame and returns nil only for response byte zero.
func (client *CreditRateLimiterClient) Charge(
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
	client.mu.Lock()
	defer client.mu.Unlock()

	if err := client.ensureConnected(ctx); err != nil {
		return fmt.Errorf("credit rate limiter connect: %w", err)
	}
	frame := client.makeFrame(companyID, userID, apiGroup, cpuCredits, inferenceCredits)
	if err := client.conn.SetDeadline(connectionDeadline(ctx)); err != nil {
		client.closeConnection()
		return fmt.Errorf("credit rate limiter deadline: %w", err)
	}
	// Never retry after writing: the server may have charged a frame whose response was lost.
	if err := writeCompleteFrame(client.conn, frame[:]); err != nil {
		client.closeConnection()
		return fmt.Errorf("credit rate limiter write: %w", err)
	}
	response := []byte{0}
	if _, err := io.ReadFull(client.conn, response); err != nil {
		client.closeConnection()
		return fmt.Errorf("credit rate limiter response: %w", err)
	}
	if client.sequence == ^uint64(0) {
		// The server cannot advance past this response either, so force a fresh nonce next time.
		client.closeConnection()
	} else {
		client.sequence++
	}
	if response[0] == 0 {
		return nil
	}
	violation, err := decodeCreditLimitResponse(response[0])
	if err != nil {
		client.closeConnection()
		return err
	}
	return violation
}

func (client *CreditRateLimiterClient) ensureConnected(ctx context.Context) error {
	if client.conn != nil {
		return nil
	}
	dialer := net.Dialer{Timeout: 2 * time.Second, KeepAlive: 30 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", client.address)
	if err != nil {
		return err
	}
	if err := connection.SetDeadline(connectionDeadline(ctx)); err != nil {
		connection.Close()
		return err
	}
	if _, err := io.ReadFull(connection, client.nonce[:]); err != nil {
		connection.Close()
		return fmt.Errorf("read server nonce: %w", err)
	}
	client.conn = connection
	client.sequence = 0
	return nil
}

func (client *CreditRateLimiterClient) makeFrame(
	companyID, userID int32,
	apiGroup uint8,
	cpuCredits, inferenceCredits uint16,
) [creditRateLimitFrameSize]byte {
	frame := [creditRateLimitFrameSize]byte{}
	writeUint24(frame[0:3], uint32(companyID))
	writeUint24(frame[3:6], uint32(userID))
	frame[6] = apiGroup
	frame[7], frame[8] = byte(cpuCredits>>8), byte(cpuCredits)
	frame[9], frame[10] = byte(inferenceCredits>>8), byte(inferenceCredits)
	mac := hmac.New(sha256.New, client.secret)
	mac.Write([]byte(creditRateLimitDomain))
	mac.Write(client.nonce[:])
	sequenceBytes := [8]byte{
		byte(client.sequence >> 56), byte(client.sequence >> 48), byte(client.sequence >> 40),
		byte(client.sequence >> 32), byte(client.sequence >> 24), byte(client.sequence >> 16),
		byte(client.sequence >> 8), byte(client.sequence),
	}
	mac.Write(sequenceBytes[:])
	mac.Write(frame[:creditRateLimitPayloadSize])
	copy(frame[creditRateLimitPayloadSize:], mac.Sum(nil)[:8])
	return frame
}

func (client *CreditRateLimiterClient) closeConnection() {
	if client.conn != nil {
		client.conn.Close()
		client.conn = nil
	}
}

func writeCompleteFrame(connection net.Conn, frame []byte) error {
	for len(frame) > 0 {
		written, err := connection.Write(frame)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrUnexpectedEOF
		}
		frame = frame[written:]
	}
	return nil
}

func decodeCreditLimitResponse(code uint8) (*CreditLimitExceeded, error) {
	windowCode := (code >> 1) & 0b11
	if code&0b1110_0000 != 0 || windowCode == 3 || code&0b0001_1000 == 0 {
		return nil, fmt.Errorf("credit rate limiter returned invalid response byte %d", code)
	}
	windows := [...]string{"10 seconds", "1 hour", "24 hours"}
	return &CreditLimitExceeded{
		Code:      code,
		Company:   code&1 == 0,
		Window:    windows[windowCode],
		Inference: code&(1<<3) != 0,
		CPU:       code&(1<<4) != 0,
	}, nil
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
