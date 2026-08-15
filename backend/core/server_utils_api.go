package core

import (
	server_utils "app/core/server_utils"
	"context"
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
// The re-exports keep call sites saying core.X like everything else in this backend. Most are
// aliases rather than wrappers: core.LockOptions and server_utils.LockOptions are the same type,
// and core.ErrLockBusy is the same sentinel, so errors.Is and type assertions work across both
// names. AcquireLock is the exception, and only because it has work to do on this side of the
// seam — the LockAction namespace and the HandlerResponse mapping both live here.

type (
	Lock                = server_utils.Lock
	LockOptions         = server_utils.LockOptions
	CreditLimitExceeded = server_utils.CreditLimitExceeded
	ServerUtilsClient   = server_utils.ServerUtilsClient
	RequestLogRecord    = server_utils.RequestLogRecord
	RequestLogEntry     = server_utils.RequestLogError
)

var (
	// ErrLockBusy is a real answer: the key is taken and the queue is full, or our patience ran
	// out. ErrLockUnavailable is the absence of an answer, which each call site judges for
	// itself — sign-up refuses, most others carry on unlocked.
	ErrLockBusy             = server_utils.ErrLockBusy
	ErrLockUnavailable      = server_utils.ErrLockUnavailable
	ErrCreditLimiterMissing = server_utils.ErrCreditLimiterMissing

	ConfigureServerUtils = server_utils.ConfigureServerUtils

	SendRequestLog              = server_utils.SendRequestLog
	ChargeAPIUsage              = server_utils.ChargeAPIUsage
	ChargeInferenceUsage        = server_utils.ChargeInferenceUsage
	WithCreditRateLimitIdentity = server_utils.WithCreditRateLimitIdentity
	IsCreditRateLimitError      = server_utils.IsCreditRateLimitError
	APICPUCredits               = server_utils.APICPUCredits
	InferenceCredits            = server_utils.InferenceCredits
)

// LockError is every way AcquireLock can fail to hand back a lock. It exists so a handler can turn
// a refusal into its HTTP answer in one line instead of restating the same mapping at each call
// site — the wrapper below is what lets it live here, where HandlerResponse is in scope, rather
// than in the protocol package that cannot import core.
//
// It keeps the sentinel underneath reachable through errors.Is, because Response is a policy, not
// the only one: it fails closed on an unavailable daemon, which is right for anything guarding a
// side effect a caller must not get twice, and wrong for a caller that would rather run unlocked.
// The latter checks ErrLockUnavailable itself and carries on.
type LockError struct{ err error }

func (lockErr *LockError) Error() string { return lockErr.err.Error() }

func (lockErr *LockError) Unwrap() error { return lockErr.err }

// Busy separates the two answers: the daemon refused us because the key is taken and the queue is
// full, versus the daemon never answered at all.
func (lockErr *LockError) Busy() bool { return errors.Is(lockErr.err, server_utils.ErrLockBusy) }

// Response is the fail-closed mapping: a contended key is the caller's problem (429), a daemon we
// cannot reach is ours (503), and neither runs the work the lock was protecting.
func (lockErr *LockError) Response(req *HandlerArgs) HandlerResponse {
	if lockErr.Busy() {
		return req.MakeErrCode("Demasiadas solicitudes simultáneas. Intente nuevamente.", 429)
	}
	Log("lock service unavailable, refusing the request::", lockErr.err)
	return req.MakeErrCode("El servicio no está disponible.", 503)
}

// AcquireLock wraps the protocol call so handlers get a LockError they can answer with directly,
// and so the action stays a LockAction (enums.go) up to the wire, where it becomes the opaque
// uint16 the daemon keys on.
//
// The concrete error type is deliberate: a nil *LockError compares nil correctly, which the
// interface would not if this ever returned a typed nil.
func AcquireLock(
	ctx context.Context, action LockAction, identifier int64, maxWaiters uint8,
) (*Lock, *LockError) {
	lock, err := server_utils.AcquireLock(ctx, uint16(action), identifier, maxWaiters)
	if err != nil {
		return nil, &LockError{err: err}
	}
	return lock, nil
}

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
