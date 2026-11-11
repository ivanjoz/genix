package config

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
)

// The Email Inbox page (frontend/routes/system/email-inbox) reads the sent_emails table that
// core.SendEmail writes: every email of a company, sent, failed, or only stored under
// [smtp].store_only. SaaS-only routes, so the company comes from the query, not the session.

const sentEmailsPageSize = 100

// GetSentEmails returns a page of a company's emails, newest first, without their bodies:
// ?target-company-id=<ID>&before=<ID> pages back from the oldest one already shown.
func GetSentEmails(req *core.HandlerArgs) core.HandlerResponse {
	companyID := req.GetQueryInt("target-company-id")
	if companyID <= 0 {
		return req.MakeErr("Debe enviar el target-company-id.")
	}

	sentEmails := []coreTypes.SentEmail{}
	query := db.Query(&sentEmails)
	query.Exclude(query.HTMLBody).CompanyID.Equals(companyID)
	if beforeID := req.GetQueryInt("before"); beforeID > 0 {
		query.ID.LessThan(beforeID)
	}
	query.OrderDesc().Limit(sentEmailsPageSize)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los correos enviados:", err)
	}
	return req.MakeResponse(sentEmails)
}

// GetSentEmail returns one email with its body (?target-company-id=&id=), for the inbox preview.
func GetSentEmail(req *core.HandlerArgs) core.HandlerResponse {
	companyID := req.GetQueryInt("target-company-id")
	sentEmailID := req.GetQueryInt("id")
	if companyID <= 0 || sentEmailID <= 0 {
		return req.MakeErr("Debe enviar el target-company-id y el id del correo.")
	}

	sentEmails := []coreTypes.SentEmail{}
	query := db.Query(&sentEmails)
	query.CompanyID.Equals(companyID).ID.Equals(sentEmailID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el correo:", err)
	}
	if len(sentEmails) == 0 {
		return req.MakeErr("El correo", sentEmailID, "no existe.")
	}
	return req.MakeResponse(sentEmails[0])
}
