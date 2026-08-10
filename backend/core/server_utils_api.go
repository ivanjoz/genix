package core

import (
	server_utils "app/core/server_utils"
	"errors"
	"fmt"
)

// The seam between core and the server-utils client.
//
// The implementation lives in app/core/server_utils because it must not import core: core needs
// CreditLimitExceeded for MakeCreditRateLimitResponse below, and that would be an import cycle.
// The dependency therefore runs core -> server_utils only, and core.Log is pushed in from main.
// Same shape as text_search, which cannot import core either.
//
// The re-exports keep call sites saying core.X like everything else in this backend. They are
// aliases, not wrappers: core.LockOptions and server_utils.LockOptions are the same type, and
// core.ErrLockBusy is the same sentinel, so errors.Is and type assertions work across both names.

type (
	Lock                = server_utils.Lock
	LockOptions         = server_utils.LockOptions
	CreditLimitExceeded = server_utils.CreditLimitExceeded
	ServerUtilsClient   = server_utils.ServerUtilsClient
)

const ActionSignUpByIP = server_utils.ActionSignUpByIP

var (
	// ErrLockBusy is a real answer: the key is taken and the queue is full, or our patience ran
	// out. ErrLockUnavailable is the absence of an answer, which each call site judges for
	// itself — sign-up refuses, most others carry on unlocked.
	ErrLockBusy             = server_utils.ErrLockBusy
	ErrLockUnavailable      = server_utils.ErrLockUnavailable
	ErrCreditLimiterMissing = server_utils.ErrCreditLimiterMissing

	ConfigureServerUtils = server_utils.ConfigureServerUtils
	AcquireLock          = server_utils.AcquireLock

	ChargeAPIUsage              = server_utils.ChargeAPIUsage
	ChargeInferenceUsage        = server_utils.ChargeInferenceUsage
	WithCreditRateLimitIdentity = server_utils.WithCreditRateLimitIdentity
	IsCreditRateLimitError      = server_utils.IsCreditRateLimitError
	APIGroup                    = server_utils.APIGroup
	APICPUCredits               = server_utils.APICPUCredits
	InferenceCredits            = server_utils.InferenceCredits
)

// MakeCreditRateLimitResponse maps the daemon's decision to a stable HTTP error and a diagnostic
// header. It stays on this side of the seam because it is an HTTP concern: it takes HandlerArgs
// and returns a HandlerResponse, neither of which the protocol package knows about.
func (req *HandlerArgs) MakeCreditRateLimitResponse(err error) HandlerResponse {
	var exceeded *server_utils.CreditLimitExceeded
	if errors.As(err, &exceeded) {
		response := req.MakeErrCode("Límite de créditos agotado.", 429)
		response.Headers["X-Rate-Limit-Code"] = fmt.Sprint(exceeded.Code)
		return response
	}
	Log("credit rate limiter unavailable::", err)
	return req.MakeErrCode("El servicio de límites de crédito no está disponible.", 503)
}
