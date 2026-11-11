package crm

import (
	"encoding/json"
	"fmt"

	"app/core"
	"app/crm/types"
	"app/db"
)

// GetCrmOpportunities syncs the open opportunities of ONE pipeline (?pipeline-id=): the board, and
// the reports, which rebuild past days backwards from them. The sensitive data is gated here: only
// members of the pipeline and users who may report on it (full-report access) get it.
func GetCrmOpportunities(req *core.HandlerArgs) core.HandlerResponse {
	pipelineID := int16(req.GetQueryInt("pipeline-id"))
	if pipelineID <= 0 {
		return req.MakeErr("Debe enviar pipeline-id.")
	}
	pipeline, err := loadPipeline(req.User.CompanyID, pipelineID)
	if err != nil {
		return req.MakeErr(err)
	}
	if types.MemberLevel(pipeline, req.User.ID) < types.PipelineLevelView {
		// Not a member: only the full-report access opens the pipeline (read-only, for the reports).
		holdsFullReport, err := userHoldsAccess(req.User.CompanyID, req.User.ID, types.AccesoIDCrmFullReport)
		if err != nil {
			return req.MakeErr(err)
		}
		if !holdsFullReport {
			return req.MakeErr("No tiene acceso a este pipeline.")
		}
	}

	opportunities := []types.CrmOpportunity{}
	query := db.Query(&opportunities)
	query.Select().CompanyID.Equals(req.User.CompanyID).PipelineID.Equals(pipelineID).
		Delta(req.GetDeltaCursor(), types.OpportunityStatusOpen)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las oportunidades:", err)
	}
	return req.MakeResponse(opportunities)
}

// GetCrmOpportunityLog returns the stage history of one opportunity (?opportunity-id=), for users
// who can see its pipeline, as the opportunity's single cache group: the browser keeps it and
// downloads it again only after a new change of that opportunity.
func GetCrmOpportunityLog(req *core.HandlerArgs) core.HandlerResponse {
	opportunity, err := authorizeOpportunityView(req, int32(req.GetQueryInt("opportunity-id")))
	if err != nil {
		return req.MakeErr(err)
	}
	heldGroups, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}
	logGroups := []db.RecordGroup[types.CrmOpportunityStageLog]{}
	query := db.QueryIndexGroup(&logGroups).CompanyID.Equals(req.User.CompanyID)
	for _, heldGroup := range heldGroups {
		query.IncludeCachedGroup(heldGroup.GroupHash, heldGroup.LastWrite)
	}
	query.OpportunityID.Between(opportunity.ID, opportunity.ID)
	if err := query.Exec(); err != nil {
		core.Log("Error querying CRM opportunity log group:", err)
		return req.MakeErr("Error al obtener el historial.")
	}
	return core.MakeResponse(req, &logGroups)
}

// PostCrmOpportunity creates or edits an opportunity's own fields. Stage changes go through
// PostCrmOpportunityMove so they are logged.
func PostCrmOpportunity(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmOpportunity{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID, userID := req.User.CompanyID, req.User.ID

	stored := types.CrmOpportunity{}
	pipelineID, stageID := payload.PipelineID, payload.StageID
	if payload.ID != 0 {
		var err error
		if stored, err = loadOpportunity(companyID, payload.ID); err != nil {
			return req.MakeErr(err)
		}
		// An edit never changes the stage: the required fields are those of the stored one.
		pipelineID, stageID = stored.PipelineID, stored.StageID
	}
	pipeline, err := loadPipeline(companyID, pipelineID)
	if err != nil {
		return req.MakeErr(err)
	}
	if err := validateClientIsActive(companyID, payload.ClientID); err != nil {
		return req.MakeErr(err)
	}
	if err := validateOpportunityCustomValues(companyID, payload.CustomValues, pipeline, stageID); err != nil {
		return req.MakeErr(err)
	}

	now := core.SUnixTime()
	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		if payload.ID == 0 {
			created, err := types.PlanOpportunityCreate(payload, pipeline, userID, now)
			if err != nil {
				return err
			}
			created.CompanyID = companyID
			records := []types.CrmOpportunity{created}
			if err := db.InsertIn(transaction, &records); err != nil {
				return err
			}
			logs := []types.CrmOpportunityStageLog{types.StageLogOfCreation(records[0], pipeline, userID, now)}
			payload = records[0]
			if err := insertStageLogs(transaction, companyID, logs); err != nil {
				return err
			}
			// Entering its first stage proposes that stage's tasks.
			return insertTasks(transaction, types.StageEntryTasks(records[0], pipeline, records[0].StageID, nil, now), userID)
		}

		edited, err := types.PlanOpportunityEdit(stored, payload, pipeline, userID)
		if err != nil {
			return err
		}
		records := []types.CrmOpportunity{edited}
		table := db.TableOf[types.CrmOpportunity]()
		if err := db.UpdateIn(transaction, &records, table.Title, table.ClientID, table.OwnerUserID, table.Amount,
			table.Probability, table.ExpectedCloseDate, table.CustomValues, table.UpdatedBy); err != nil {
			return err
		}
		payload = records[0]
		// The task cards show the opportunity's title, client and amount: resend its open tasks.
		if types.EditChangesTaskCard(stored, edited) {
			if err := touchPendingTasks(transaction, companyID, edited.ID, userID); err != nil {
				return err
			}
		}
		// Amount, owner and close date feed the reports (waterfall, owner transfers, slippage).
		if !types.EditChangesReportedState(stored, edited) {
			return nil
		}
		logs := []types.CrmOpportunityStageLog{types.BuildStageLogRow(types.StageLogKindEdited, stored, pipeline, edited, pipeline, userID, now)}
		return insertStageLogs(transaction, companyID, logs)
	})
	if saveError != nil {
		return req.MakeErr("Error al guardar la oportunidad:", saveError)
	}
	return req.MakeResponse(payload)
}

// PostCrmOpportunityMove moves an opportunity to another stage or pipeline, and logs it in the
// same transaction.
func PostCrmOpportunityMove(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmOpportunityMovePayload{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID, userID := req.User.CompanyID, req.User.ID

	var moved types.CrmOpportunity
	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		stored, err := lockOpportunity(transaction, companyID, payload.ID)
		if err != nil {
			return err
		}
		source, err := loadPipeline(companyID, stored.PipelineID)
		if err != nil {
			return err
		}
		target := source
		if payload.PipelineID != 0 && payload.PipelineID != source.ID {
			if target, err = loadPipeline(companyID, payload.PipelineID); err != nil {
				return err
			}
		}
		now := core.SUnixTime()
		movedRecord, stageLog, err := types.PlanOpportunityMove(stored, source, target, payload.StageID, userID, now)
		if err != nil {
			return err
		}
		approverUserID, err := types.ApprovalGate(stored, source, target, payload.StageID)
		if err != nil {
			return err
		}
		if approverUserID != 0 {
			// The stage has an approver: the card stays where it is and waits on an approval task.
			moved, err = requestApproval(transaction, stored, target, payload.StageID, approverUserID, userID, now)
			return err
		}
		moved = movedRecord
		return persistStageChange(transaction, movedRecord, stageLog, target, userID, now)
	})
	if saveError != nil {
		return req.MakeErr("Error al mover la oportunidad:", saveError)
	}
	return req.MakeResponse(moved)
}

// PostCrmOpportunityClose closes an opportunity as won or lost, or reopens it, and logs it.
// Winning also clears the prospect flag of its client: the lead is now a customer.
func PostCrmOpportunityClose(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmOpportunityClosePayload{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID, userID := req.User.CompanyID, req.User.ID

	lostReasonIsActive := false
	if payload.Outcome == types.OpportunityOutcomeLost {
		var err error
		if lostReasonIsActive, err = lostReasonExists(companyID, payload.LostReasonID); err != nil {
			return req.MakeErr(err)
		}
	}

	var closed types.CrmOpportunity
	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		stored, err := lockOpportunity(transaction, companyID, payload.ID)
		if err != nil {
			return err
		}
		pipeline, err := loadPipeline(companyID, stored.PipelineID)
		if err != nil {
			return err
		}
		now := core.SUnixTime()
		closedRecord, stageLog, err := types.PlanOpportunityClose(stored, pipeline, payload, lostReasonIsActive, userID, now)
		if err != nil {
			return err
		}
		records := []types.CrmOpportunity{closedRecord}
		table := db.TableOf[types.CrmOpportunity]()
		if err := db.UpdateIn(transaction, &records, table.Outcome, table.LostReasonID, table.LostNote, table.ClosedTime,
			table.StageEnteredTime, table.Status, table.UpdatedBy); err != nil {
			return err
		}
		if closedRecord.Outcome == types.OpportunityOutcomeWon {
			clients := []types.ClientProvider{{CompanyID: companyID, ID: closedRecord.ClientID, IsProspect: false, UpdatedBy: userID}}
			clientTable := db.TableOf[types.ClientProvider]()
			if err := db.UpdateIn(transaction, &clients, clientTable.IsProspect, clientTable.UpdatedBy); err != nil {
				return err
			}
		}
		logs := []types.CrmOpportunityStageLog{stageLog}
		closed = records[0]
		if err := insertStageLogs(transaction, companyID, logs); err != nil {
			return err
		}
		// Lost: what it still owed is cancelled. Reopened: it re-enters its stage, which proposes
		// that stage's tasks again (plan P3). Won leaves them as they are.
		switch closedRecord.Outcome {
		case types.OpportunityOutcomeLost:
			return syncStageTasks(transaction, closedRecord, pipeline, 0, userID, now)
		case types.OpportunityOutcomeOpen:
			return syncStageTasks(transaction, closedRecord, pipeline, closedRecord.StageID, userID, now)
		}
		return nil
	})
	if saveError != nil {
		return req.MakeErr("Error al cerrar la oportunidad:", saveError)
	}
	return req.MakeResponse(closed)
}

func loadPipeline(companyID int32, pipelineID int16) (types.CrmPipeline, error) {
	pipelines := []types.CrmPipeline{}
	query := db.Query(&pipelines)
	query.CompanyID.Equals(companyID).ID.Equals(pipelineID)
	if err := query.Exec(); err != nil {
		return types.CrmPipeline{}, fmt.Errorf("error al leer el pipeline: %v", err)
	}
	if len(pipelines) == 0 || pipelines[0].Status != types.CrmStatusActive {
		return types.CrmPipeline{}, fmt.Errorf("El pipeline %d no existe.", pipelineID)
	}
	return pipelines[0], nil
}

func loadOpportunity(companyID, opportunityID int32) (types.CrmOpportunity, error) {
	opportunities := []types.CrmOpportunity{}
	query := db.Query(&opportunities)
	query.CompanyID.Equals(companyID).ID.Equals(opportunityID)
	if err := query.Exec(); err != nil {
		return types.CrmOpportunity{}, fmt.Errorf("error al leer la oportunidad: %v", err)
	}
	if len(opportunities) == 0 {
		return types.CrmOpportunity{}, fmt.Errorf("La oportunidad %d no existe.", opportunityID)
	}
	return opportunities[0], nil
}

// lockOpportunity reads the stored opportunity under lock, so two moves of the same card are serialized.
func lockOpportunity(transaction *db.Transaction, companyID, opportunityID int32) (types.CrmOpportunity, error) {
	records := []types.CrmOpportunity{{CompanyID: companyID, ID: opportunityID}}
	if err := db.LockByKeys(transaction, &records); err != nil {
		return types.CrmOpportunity{}, err
	}
	if len(records) == 0 {
		return types.CrmOpportunity{}, fmt.Errorf("La oportunidad %d no existe.", opportunityID)
	}
	return records[0], nil
}

func loadOpportunityWithPipeline(companyID, opportunityID int32) (types.CrmOpportunity, types.CrmPipeline, error) {
	opportunity, err := loadOpportunity(companyID, opportunityID)
	if err != nil {
		return types.CrmOpportunity{}, types.CrmPipeline{}, err
	}
	pipeline, err := loadPipeline(companyID, opportunity.PipelineID)
	return opportunity, pipeline, err
}

// validateClientIsActive: the client (or prospect) must be an active client of the company.
func validateClientIsActive(companyID, clientID int32) error {
	clients := []types.ClientProvider{}
	query := db.Query(&clients)
	query.Select(query.ID, query.Type, query.Status).CompanyID.Equals(companyID).ID.Equals(clientID)
	if err := query.Exec(); err != nil {
		return fmt.Errorf("error al leer el cliente: %v", err)
	}
	if len(clients) == 0 || clients[0].Status != types.CrmStatusActive || clients[0].Type != types.ClientProviderTypeClient {
		return fmt.Errorf("El cliente %d no existe.", clientID)
	}
	return nil
}

func validateOpportunityCustomValues(companyID int32, values []types.CrmCustomFieldValue, pipeline types.CrmPipeline, stageID int16) error {
	customFields := []types.CrmCustomField{}
	query := db.Query(&customFields)
	query.Select().CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return fmt.Errorf("error al leer los campos personalizados: %v", err)
	}
	catalog := map[int16]types.CrmCustomField{}
	for _, customField := range customFields {
		catalog[customField.ID] = customField
	}
	userIDs := []int32{}
	for _, value := range values {
		if catalog[value.FieldID].Type == types.CustomFieldUser {
			userIDs = append(userIDs, int32(value.Number))
		}
	}
	activeUserIDs, err := loadActiveUserIDs(companyID, userIDs)
	if err != nil {
		return err
	}
	return types.ValidateCustomValues(values, catalog, pipeline, stageID, activeUserIDs)
}

func lostReasonExists(companyID int32, lostReasonID int16) (bool, error) {
	lostReasons := []types.CrmLostReason{}
	query := db.Query(&lostReasons)
	query.CompanyID.Equals(companyID).ID.Equals(lostReasonID)
	if err := query.Exec(); err != nil {
		return false, fmt.Errorf("error al leer el motivo de pérdida: %v", err)
	}
	return len(lostReasons) > 0 && lostReasons[0].Status == types.CrmStatusActive, nil
}
