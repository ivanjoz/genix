package core

import (
	"context"
	"errors"
	"fmt"
	fareward "github.com/ivanjoz/fareward/go"
	"strings"
)

// The seam between core and the fareward client.
//
// The client is no longer part of this module: it lives in the daemon's own repository as
// github.com/ivanjoz/fareward/go, replaced into the fareward submodule, so the wire protocol and
// its reference implementation sit in one place and a non-Genix backend can depend on the client
// without depending on this repository. It imports only the standard library.
//
// That it is external also settles what used to be an internal rule. The client must not import
// core — core needs CreditLimitExceeded for MakeCreditRateLimitResponse below, and that would be
// an import cycle — so the dependency runs core -> fareward only and core.Log is pushed in from
// main. A module boundary now enforces what a convention used to. Same shape as text_search,
// which cannot import core either.
//
// The re-exports keep call sites saying core.X like everything else in this backend. Most are
// aliases rather than wrappers: core.LockOptions and fareward.LockOptions are the same type,
// and core.ErrLockBusy is the same sentinel, so errors.Is and type assertions work across both
// names. AcquireLock is the exception, and only because it has work to do on this side of the
// seam — the LockAction namespace and the HandlerResponse mapping both live here.

type (
	Lock                = fareward.Lock
	LockOptions         = fareward.LockOptions
	CreditLimitExceeded = fareward.CreditLimitExceeded
	AccessDenied        = fareward.AccessDenied
	BudgetOperation     = fareward.BudgetOperation
	FarewardClient      = fareward.FarewardClient
	RequestLogRecord    = fareward.RequestLogRecord
	RequestLogEntry     = fareward.RequestLogError
)

const (
	// InvalidateAllCompanyUsers drops every cached user of a company instead of one.
	InvalidateAllCompanyUsers = fareward.InvalidateAllCompanyUsers

	// MaxRequiredAccess bounds how many accesses one route may map to. The gate refuses to encode
	// more, and TestEveryRouteFitsTheRequiredAccessSlots keeps access_list.yml inside it.
	MaxRequiredAccess = fareward.MaxRequiredAccess

	BudgetSetDaily        = fareward.BudgetSetDaily
	BudgetSetCurrent      = fareward.BudgetSetCurrent
	BudgetIncreaseCurrent = fareward.BudgetIncreaseCurrent
)

var (
	// ErrLockBusy is a real answer: the key is taken and the queue is full, or our patience ran
	// out. ErrLockUnavailable is the absence of an answer, which each call site judges for
	// itself — sign-up refuses, most others carry on unlocked.
	ErrLockBusy                 = fareward.ErrLockBusy
	ErrLockUnavailable          = fareward.ErrLockUnavailable
	ErrCreditLimiterMissing     = fareward.ErrCreditLimiterMissing
	ErrBudgetMonthNotConfigured = fareward.ErrBudgetMonthNotConfigured
	ErrBudgetMutationOverflow   = fareward.ErrBudgetMutationOverflow

	ConfigureFareward = fareward.ConfigureFareward

	SendRequestLog              = fareward.SendRequestLog
	InvalidateUserAccess        = fareward.InvalidateUserAccess
	ChargeAPIUsage              = fareward.ChargeAPIUsage
	ChargeAPICredits            = fareward.ChargeAPICredits
	ChargeAPIAccessOnly         = fareward.ChargeAPIAccessOnly
	APICPUBaseCredits           = fareward.APICPUBaseCredits
	IsAccessDeniedError         = fareward.IsAccessDeniedError
	ChargeInferenceUsage        = fareward.ChargeInferenceUsage
	WithCreditRateLimitIdentity = fareward.WithCreditRateLimitIdentity
	IsCreditRateLimitError      = fareward.IsCreditRateLimitError
	APICPUCredits               = fareward.APICPUCredits
	InferenceCredits            = fareward.InferenceCredits
	MutateCompanyCreditBudget   = fareward.MutateCompanyCreditBudget
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
func (lockErr *LockError) Busy() bool { return errors.Is(lockErr.err, fareward.ErrLockBusy) }

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
	lock, err := fareward.AcquireLock(ctx, uint16(action), identifier, maxWaiters)
	if err != nil {
		return nil, &LockError{err: err}
	}
	return lock, nil
}

// MakeCreditRateLimitResponse maps the daemon's decision to a stable HTTP error and a diagnostic
// header. It stays on this side of the seam because it is an HTTP concern: it takes HandlerArgs
// and returns a HandlerResponse, neither of which the protocol package knows about.
func (req *HandlerArgs) MakeCreditRateLimitResponse(err error) HandlerResponse {
	var exceeded *fareward.CreditLimitExceeded
	if errors.As(err, &exceeded) {
		message := "Límite de créditos agotado."
		if exceeded.Window == "month" {
			message = "Presupuesto mensual de créditos agotado."
		}
		response := req.MakeErrCode(message, 429)
		response.Headers["X-Rate-Limit-Code"] = fmt.Sprint(exceeded.Code)
		return response
	}
	Log("credit rate limiter unavailable::", err)
	return req.MakeErrCode("El servicio de límites de crédito no está disponible.", 503)
}

// MakeAccessDeniedResponse maps the daemon's authorization refusal to an HTTP answer.
//
// The two codes are not interchangeable. A missing or inactive user is a statement about the
// session, so it is a 401 and the client must re-authenticate; lacking a permission is a 403 and
// re-authenticating would achieve nothing. accessNames comes from the caller because the daemon
// never sees them — it holds no copy of access_list.yml, deliberately.
func (req *HandlerArgs) MakeAccessDeniedResponse(err error, accessNames []string) HandlerResponse {
	var denied *fareward.AccessDenied
	if !errors.As(err, &denied) {
		Log("credit rate limiter unavailable during authorization::", err)
		return req.MakeErrCode("El servicio de límites de crédito no está disponible.", 503)
	}
	if denied.IdentityFailed() {
		return req.MakeErrCode("La sesión no es válida o el user está inactivo.", 401)
	}
	return req.MakeErrCode(
		fmt.Sprintf("El user no posee alguno de los accesos: %s", strings.Join(accessNames, ", ")),
		403)
}
