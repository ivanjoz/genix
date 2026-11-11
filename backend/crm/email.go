package crm

import (
	"encoding/json"
	"fmt"
	"strings"

	config "app/config/types"
	"app/core"
	coreTypes "app/core/types"
	"app/crm/types"
	"app/db"
)

func GetCrmEmailTemplates(req *core.HandlerArgs) core.HandlerResponse {
	emailTemplates := []types.CrmEmailTemplate{}
	query := db.Query(&emailTemplates)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), types.CrmStatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las plantillas de correo:", err)
	}
	return req.MakeResponse(emailTemplates)
}

// PostCrmEmailTemplate creates or edits an email template; Status 0 deletes it (soft: a stage task
// may still name it, and the pipeline editor then asks for another one).
func PostCrmEmailTemplate(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmEmailTemplate{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	isDelete := payload.ID != 0 && payload.Status == types.CrmStatusDeleted
	if !isDelete {
		if err := types.NormalizeEmailTemplate(&payload); err != nil {
			return req.MakeErr(err)
		}
	}
	payload.CompanyID, payload.UpdatedBy = req.User.CompanyID, req.User.ID
	payload.Status = core.If(isDelete, types.CrmStatusDeleted, types.CrmStatusActive)

	records := []types.CrmEmailTemplate{payload}
	var saveError error
	if payload.ID == 0 {
		saveError = db.Insert(&records)
	} else {
		if _, err := loadEmailTemplate(payload.CompanyID, payload.ID); err != nil {
			return req.MakeErr(err)
		}
		table := db.TableOf[types.CrmEmailTemplate]()
		if isDelete {
			saveError = db.Update(&records, table.Status, table.UpdatedBy)
		} else {
			saveError = db.Update(&records, table.Name, table.Subject, table.BodyHTML, table.Status, table.UpdatedBy)
		}
	}
	if saveError != nil {
		return req.MakeErr("Error al guardar la plantilla de correo:", saveError)
	}
	return req.MakeResponse(records[0])
}

// GetCrmOpportunityEmailDraft renders a template for one opportunity (?opportunity-id=&template-id=):
// the seller reviews and edits it before sending. To is the client's email, when it has one.
func GetCrmOpportunityEmailDraft(req *core.HandlerArgs) core.HandlerResponse {
	opportunity, err := authorizeOpportunityView(req, int32(req.GetQueryInt("opportunity-id")))
	if err != nil {
		return req.MakeErr(err)
	}
	emailTemplate, err := loadEmailTemplate(req.User.CompanyID, int16(req.GetQueryInt("template-id")))
	if err != nil {
		return req.MakeErr(err)
	}
	if emailTemplate.Status != types.CrmStatusActive {
		return req.MakeErr("La plantilla de correo", emailTemplate.ID, "fue eliminada.")
	}
	renderContext, err := loadEmailRenderContext(req.User.CompanyID, opportunity)
	if err != nil {
		return req.MakeErr(err)
	}
	subject, body, err := types.RenderEmailTemplate(emailTemplate, renderContext)
	if err != nil {
		return req.MakeErr(err)
	}
	draft := types.CrmEmailDraft{To: []string{}, Subject: subject, HTMLBody: body}
	if renderContext.ClientEmail != "" {
		draft.To = append(draft.To, renderContext.ClientEmail)
	}
	return req.MakeResponse(draft)
}

// PostCrmOpportunityEmail sends an email from an opportunity and logs it on its timeline as an Email
// activity. The sender is "<company> <platform address>" and answers go to the seller (Reply-To).
// With TaskID the email fulfils that "send email" task.
//
// The email cannot take part in the transaction, so everything that could refuse the write (the task
// check included) runs before it is sent; what is written after it can only fail on the database.
func PostCrmOpportunityEmail(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmOpportunityEmailPayload{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if err := types.NormalizeOpportunityEmail(&payload); err != nil {
		return req.MakeErr(err)
	}
	opportunity, err := authorizeOpportunityView(req, payload.OpportunityID)
	if err != nil {
		return req.MakeErr(err)
	}
	now := core.SUnixTime()
	activity, err := types.PlanActivity(types.CrmActivity{
		OpportunityID: opportunity.ID, TypeID: types.ActivityTypeEmail, Text: types.EmailActivityText(payload),
	}, req.User.CompanyID, opportunity.PipelineID, false, req.User.ID, now)
	if err != nil {
		return req.MakeErr(err)
	}
	if payload.TaskID != 0 {
		task, err := loadTask(req.User.CompanyID, payload.TaskID)
		if err != nil {
			return req.MakeErr(err)
		}
		if _, err := types.PlanActivityTaskDone(task, activity, types.TaskKindSendEmail, req.User.ID, now); err != nil {
			return req.MakeErr(err)
		}
	}

	companyName, senderEmail, err := loadEmailSender(req.User.CompanyID, req.User.ID)
	if err != nil {
		return req.MakeErr(err)
	}
	if err := core.SendEmail(req.User.CompanyID, core.EmailMessage{
		To: payload.To, Cc: payload.Cc, FromName: companyName, ReplyTo: senderEmail,
		Subject: payload.Subject, HTMLBody: payload.HTMLBody,
	}); err != nil {
		return req.MakeErr(err)
	}

	records := []types.CrmActivity{activity}
	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		if err := db.InsertIn(transaction, &records); err != nil {
			return err
		}
		if payload.TaskID == 0 {
			return nil
		}
		task, err := lockTask(transaction, req.User.CompanyID, payload.TaskID)
		if err != nil {
			return err
		}
		doneTask, err := types.PlanActivityTaskDone(task, records[0], types.TaskKindSendEmail, req.User.ID, now)
		if err != nil {
			return err
		}
		return updateClosedTask(transaction, doneTask)
	})
	if saveError != nil {
		// Retrying would send the email twice: say it left.
		core.Log("PostCrmOpportunityEmail:: email sent but not logged, opportunity", opportunity.ID, ":", saveError)
		return req.MakeErr("El correo se envió, pero no se pudo registrar en la oportunidad:", saveError)
	}
	return req.MakeResponse(records[0])
}

// loadEmailRenderContext gathers the values of the placeholders for one opportunity: its client, its
// owner and the company. A client or owner that is gone leaves its values empty.
func loadEmailRenderContext(companyID int32, opportunity types.CrmOpportunity) (types.EmailRenderContext, error) {
	renderContext := types.EmailRenderContext{
		OpportunityTitle: opportunity.Title, OpportunityAmount: types.EmailAmount(opportunity.Amount),
	}
	clients := []types.ClientProvider{}
	clientQuery := db.Query(&clients)
	clientQuery.Select(clientQuery.ID, clientQuery.Name, clientQuery.Email).CompanyID.Equals(companyID).ID.Equals(opportunity.ClientID)
	if err := clientQuery.Exec(); err != nil {
		return renderContext, fmt.Errorf("error al leer el cliente: %v", err)
	}
	if len(clients) > 0 {
		renderContext.ClientName, renderContext.ClientEmail = clients[0].Name, strings.TrimSpace(clients[0].Email)
	}
	if opportunity.OwnerUserID != 0 {
		owner, err := loadUser(companyID, opportunity.OwnerUserID)
		if err != nil {
			return renderContext, err
		}
		renderContext.OwnerName = strings.TrimSpace(owner.FirstName + " " + owner.LastName)
		renderContext.OwnerEmail = owner.Email
	}
	companyName, err := loadCompanyName(companyID)
	renderContext.CompanyName = companyName
	return renderContext, err
}

// loadEmailSender returns the From name (the company) and the Reply-To (the sending user's email).
func loadEmailSender(companyID, userID int32) (companyName string, senderEmail string, err error) {
	if companyName, err = loadCompanyName(companyID); err != nil {
		return "", "", err
	}
	sender, err := loadUser(companyID, userID)
	return companyName, sender.Email, err
}

func loadUser(companyID, userID int32) (coreTypes.User, error) {
	users := []coreTypes.User{}
	query := db.Query(&users)
	query.Select(query.ID, query.FirstName, query.LastName, query.Email).CompanyID.Equals(companyID).ID.Equals(userID)
	if err := query.Exec(); err != nil {
		return coreTypes.User{}, fmt.Errorf("error al leer el usuario: %v", err)
	}
	if len(users) == 0 {
		return coreTypes.User{}, fmt.Errorf("El usuario %d no existe.", userID)
	}
	return users[0], nil
}

func loadCompanyName(companyID int32) (string, error) {
	companies := []config.Company{}
	query := db.Query(&companies)
	query.Select(query.ID, query.Name).ID.Equals(companyID).Limit(1)
	if err := query.Exec(); err != nil {
		return "", fmt.Errorf("error al leer la empresa: %v", err)
	}
	if len(companies) == 0 {
		return "", fmt.Errorf("La empresa %d no existe.", companyID)
	}
	return companies[0].Name, nil
}

func loadEmailTemplate(companyID int32, emailTemplateID int16) (types.CrmEmailTemplate, error) {
	emailTemplates := []types.CrmEmailTemplate{}
	query := db.Query(&emailTemplates)
	query.CompanyID.Equals(companyID).ID.Equals(emailTemplateID)
	if err := query.Exec(); err != nil {
		return types.CrmEmailTemplate{}, fmt.Errorf("error al leer la plantilla de correo: %v", err)
	}
	if len(emailTemplates) == 0 {
		return types.CrmEmailTemplate{}, fmt.Errorf("La plantilla de correo %d no existe.", emailTemplateID)
	}
	return emailTemplates[0], nil
}

// loadActiveEmailTemplateIDs answers which of the company's email templates (by ID) are active.
func loadActiveEmailTemplateIDs(companyID int32, emailTemplateIDs []int16) (map[int16]bool, error) {
	activeIDs := map[int16]bool{}
	if len(emailTemplateIDs) == 0 {
		return activeIDs, nil
	}
	emailTemplates := []types.CrmEmailTemplate{}
	query := db.Query(&emailTemplates)
	query.Select(query.ID, query.Status).CompanyID.Equals(companyID).ID.In(emailTemplateIDs...)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las plantillas de correo: %v", err)
	}
	for _, emailTemplate := range emailTemplates {
		if emailTemplate.Status == types.CrmStatusActive {
			activeIDs[emailTemplate.ID] = true
		}
	}
	return activeIDs, nil
}
