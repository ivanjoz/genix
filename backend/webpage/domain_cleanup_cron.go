package webpage

import (
	"app/core"
	"strings"
)

const (
	previousDomainCleanupActionID = int16(4)
	// The cleanup runs on the next 5-minute frame: releasing the old hostname is not something the
	// browser should wait on, and 5 minutes sits well inside the 60-minute domain change cooldown,
	// so the company cannot re-provision the hostname between the save and the cleanup.
	previousDomainCleanupFrameMinutes = int8(5)
)

func init() {
	core.RegisterActionHandler(
		previousDomainCleanupActionID,
		"Borrar dominio anterior en Cloudflare",
		RemovePreviousDomainHandler,
	)
}

// schedulePreviousDomainCleanup enqueues the Cloudflare cleanup of a hostname the company just
// stopped using. One-shot, not recurring: the row must run once and never re-enqueue itself.
//
// The hostname travels in Param5, which also feeds the cron row ID hash, so two hostnames released
// within the same frame get a row each instead of deduplicating into one and leaking the other.
func schedulePreviousDomainCleanup(companyID int32, previousDomain string) {
	defer func() {
		// ScheduleCronAction panics on a database error. Losing the cleanup leaks a Cloudflare
		// record, which is worth a log line but not worth failing a save that already succeeded.
		if recoveredValue := recover(); recoveredValue != nil {
			core.Log("Error programando el borrado del dominio anterior::", previousDomain, recoveredValue)
		}
	}()

	core.ScheduleCronAction(core.CronAction{
		ActionID:  previousDomainCleanupActionID,
		CompanyID: companyID,
		Params:    core.ExecArgs{Param1: int64(companyID), Param5: previousDomain},
	}, previousDomainCleanupFrameMinutes)

	core.Log("Borrado del dominio anterior programado::", previousDomain)
}

// RemovePreviousDomainHandler drops a released hostname from the storefront Worker and from the
// zone's DNS records. Returning an error leaves the row pending so the executor retries it, which is
// the point of doing this out of band: a Cloudflare outage delays the cleanup instead of failing the
// domain save that triggered it.
func RemovePreviousDomainHandler(args *core.ExecArgs) core.FuncResponse {
	previousDomain := strings.TrimSpace(args.Param5)
	if previousDomain == "" {
		return args.MakeErr("El dominio a borrar es requerido en Param5.")
	}

	// Refuse to release a hostname the company went back to using. The row was written when the
	// domain was still the old one, and a failed attempt keeps retrying for up to an hour, so the
	// current value has to be re-read at execution time rather than trusted from the payload.
	currentDomain, readError := getCompanyDomain(int32(args.Param1))
	if readError != nil {
		return args.MakeErr("Error al obtener el dominio actual:", readError)
	}
	if currentDomain != nil && currentDomain.Value == previousDomain {
		args.AddMessage(core.Concat(" ", "El dominio", previousDomain, "volvió a estar en uso, no se borró"))
		return core.FuncResponse{}
	}

	if removeError := removeStorefrontDomain(previousDomain); removeError != nil {
		return args.MakeErr("Error al borrar el dominio anterior:", removeError)
	}

	args.AddMessage(core.Concat(" ", "Borrado dominio", previousDomain))
	return core.FuncResponse{}
}
