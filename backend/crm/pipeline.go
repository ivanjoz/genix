package crm

import (
	"encoding/json"
	"fmt"

	"app/core"
	coreTypes "app/core/types"
	"app/crm/types"
	"app/db"
)

// GetCrmPipelines syncs every pipeline of the company to every CRM user. Membership only gates
// the opportunities (the sensitive data), so the selector is filtered in the browser.
func GetCrmPipelines(req *core.HandlerArgs) core.HandlerResponse {
	pipelines := []types.CrmPipeline{}
	query := db.Query(&pipelines)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), types.CrmStatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los pipelines:", err)
	}
	return req.MakeResponse(pipelines)
}

// PostCrmPipeline creates or edits a pipeline's definition (name, stages, fields). It never
// writes Members. A pipeline's creator becomes its first Full member, since nobody sees a
// pipeline until it is assigned.
func PostCrmPipeline(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmPipeline{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID

	activeFieldIDs, err := loadActiveCustomFieldIDs(companyID)
	if err != nil {
		return req.MakeErr(err)
	}

	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		stored := types.CrmPipeline{}
		if payload.ID != 0 {
			storedRecords := []types.CrmPipeline{{CompanyID: companyID, ID: payload.ID}}
			if err := db.LockByKeys(transaction, &storedRecords); err != nil {
				return err
			}
			if len(storedRecords) == 0 || storedRecords[0].Status != types.CrmStatusActive {
				return fmt.Errorf("El pipeline %d no existe.", payload.ID)
			}
			stored = storedRecords[0]
		}

		openStageIDs, err := loadOpenOpportunityStageIDs(companyID, stored.ID)
		if err != nil {
			return err
		}
		if payload.ID != 0 && payload.Status == types.CrmStatusDeleted {
			if len(openStageIDs) > 0 {
				return fmt.Errorf("El pipeline todavía tiene oportunidades abiertas; muévalas antes de eliminarlo.")
			}
			stored.Status, stored.UpdatedBy = types.CrmStatusDeleted, req.User.ID
			deleted := []types.CrmPipeline{stored}
			table := db.TableOf[types.CrmPipeline]()
			if err := db.UpdateIn(transaction, &deleted, table.Status, table.UpdatedBy); err != nil {
				return err
			}
			payload = deleted[0]
			return nil
		}

		saved, err := types.ApplyPipelineEdit(stored, payload, func(stageID int16) bool { return openStageIDs[stageID] }, activeFieldIDs)
		if err != nil {
			return err
		}
		if err := validatePipelineReferences(companyID, saved); err != nil {
			return err
		}
		saved.CompanyID, saved.UpdatedBy = companyID, req.User.ID
		records := []types.CrmPipeline{saved}
		if saved.ID == 0 {
			records[0].Members = []types.CrmPipelineMember{{UserID: req.User.ID, Level: types.PipelineLevelFull}}
			if err := db.InsertIn(transaction, &records); err != nil {
				return err
			}
		} else {
			table := db.TableOf[types.CrmPipeline]()
			// Members is left out on purpose: the access page owns it.
			if err := db.UpdateIn(transaction, &records, table.Name, table.Description, table.Color, table.SortOrder,
				table.Stages, table.Fields, table.GoalPeriodType, table.Status, table.UpdatedBy); err != nil {
				return err
			}
		}
		payload = records[0]
		return nil
	})
	if saveError != nil {
		return req.MakeErr("Error al guardar el pipeline:", saveError)
	}
	return req.MakeResponse(payload)
}

// validatePipelineReferences checks the users, activity types and email templates the pipeline's
// stages name (approvers, task assignees, what a task asks for) against the database.
func validatePipelineReferences(companyID int32, pipeline types.CrmPipeline) error {
	userIDs, activityTypeIDs, emailTemplateIDs := types.PipelineReferenceIDs(pipeline)
	activeUserIDs, err := loadActiveUserIDs(companyID, userIDs)
	if err != nil {
		return err
	}
	activeActivityTypeIDs, err := loadActiveActivityTypeIDs(companyID, activityTypeIDs)
	if err != nil {
		return err
	}
	activeEmailTemplateIDs, err := loadActiveEmailTemplateIDs(companyID, emailTemplateIDs)
	if err != nil {
		return err
	}
	return types.ValidatePipelineReferences(pipeline, types.PipelineReferences{
		ActiveUserIDs: activeUserIDs, ActiveActivityTypeIDs: activeActivityTypeIDs, ActiveEmailTemplateIDs: activeEmailTemplateIDs,
	})
}

// PostCrmPipelineMembers replaces who can work in one pipeline and at what level. It is the only
// writer of Members.
func PostCrmPipelineMembers(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmPipelineMembersPayload{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if payload.PipelineID == 0 {
		return req.MakeErr("Debe indicar el pipeline.")
	}
	companyID := req.User.CompanyID

	memberUserIDs := make([]int32, 0, len(payload.Members))
	for _, member := range payload.Members {
		memberUserIDs = append(memberUserIDs, member.UserID)
	}
	activeUserIDs, err := loadActiveUserIDs(companyID, memberUserIDs)
	if err != nil {
		return req.MakeErr(err)
	}
	if err := types.ValidatePipelineMembers(payload.Members, activeUserIDs); err != nil {
		return req.MakeErr(err)
	}

	var savedPipeline types.CrmPipeline
	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		storedRecords := []types.CrmPipeline{{CompanyID: companyID, ID: payload.PipelineID}}
		if err := db.LockByKeys(transaction, &storedRecords); err != nil {
			return err
		}
		if len(storedRecords) == 0 || storedRecords[0].Status != types.CrmStatusActive {
			return fmt.Errorf("El pipeline %d no existe.", payload.PipelineID)
		}
		storedRecords[0].Members, storedRecords[0].UpdatedBy = payload.Members, req.User.ID
		table := db.TableOf[types.CrmPipeline]()
		if err := db.UpdateIn(transaction, &storedRecords, table.Members, table.UpdatedBy); err != nil {
			return err
		}
		savedPipeline = storedRecords[0]
		return nil
	})
	if saveError != nil {
		return req.MakeErr("Error al guardar los accesos del pipeline:", saveError)
	}
	return req.MakeResponse(savedPipeline)
}

// loadOpenOpportunityStageIDs returns the stages (of one pipeline) that still hold open
// opportunities. A zero pipeline ID has none.
func loadOpenOpportunityStageIDs(companyID int32, pipelineID int16) (map[int16]bool, error) {
	stageIDs := map[int16]bool{}
	if pipelineID == 0 {
		return stageIDs, nil
	}
	// Read through the board's delta index (PipelineID, Status=open) with an empty cursor: it reads
	// only the open opportunities, not the pipeline's whole closed history.
	opportunities := []types.CrmOpportunity{}
	query := db.Query(&opportunities)
	query.Select(query.ID, query.StageID).CompanyID.Equals(companyID).PipelineID.Equals(pipelineID).
		Delta(db.DeltaCursor{}, types.OpportunityStatusOpen)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las oportunidades: %v", err)
	}
	for _, opportunity := range opportunities {
		// Skip the metadata record the Delta() read appends.
		if opportunity.Metadata == nil {
			stageIDs[opportunity.StageID] = true
		}
	}
	return stageIDs, nil
}

// loadActiveUserIDs returns which of userIDs are active users of the company.
func loadActiveUserIDs(companyID int32, userIDs []int32) (map[int32]bool, error) {
	activeUserIDs := map[int32]bool{}
	if len(userIDs) == 0 {
		return activeUserIDs, nil
	}
	users := []coreTypes.User{}
	query := db.Query(&users)
	query.Select(query.ID, query.Status).CompanyID.Equals(companyID).ID.In(userIDs...)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los usuarios: %v", err)
	}
	for _, user := range users {
		if user.Status == types.CrmStatusActive {
			activeUserIDs[user.ID] = true
		}
	}
	return activeUserIDs, nil
}
