package crm

import (
	"encoding/json"

	"app/core"
	"app/crm/types"
	"app/db"
)

// GetCrmGoals syncs every goal of the company. Reading a goal is not sensitive for CRM users: the
// scorecards and the forecast page show them next to the results.
func GetCrmGoals(req *core.HandlerArgs) core.HandlerResponse {
	goals := []types.CrmGoal{}
	query := db.Query(&goals)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), types.CrmStatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las metas:", err)
	}
	return req.MakeResponse(goals)
}

// PostCrmGoals saves the cells the goals grid changed. A zero target clears a cell.
func PostCrmGoals(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmGoalsPayload{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if len(payload.Goals) == 0 {
		return req.MakeErr("No hay metas para guardar.")
	}
	companyID := req.User.CompanyID

	pipelines := []types.CrmPipeline{}
	pipelineQuery := db.Query(&pipelines)
	pipelineQuery.Select(pipelineQuery.ID, pipelineQuery.Status, pipelineQuery.GoalPeriodType).CompanyID.Equals(companyID)
	if err := pipelineQuery.Exec(); err != nil {
		return req.MakeErr("Error al leer los pipelines:", err)
	}
	goalPeriodTypeByPipelineID := map[int16]int8{}
	for _, pipeline := range pipelines {
		if pipeline.Status == types.CrmStatusActive {
			goalPeriodTypeByPipelineID[pipeline.ID] = pipeline.GoalPeriodType
		}
	}
	userIDs := make([]int32, 0, len(payload.Goals))
	for _, goal := range payload.Goals {
		userIDs = append(userIDs, goal.UserID)
	}
	activeUserIDs, err := loadActiveUserIDs(companyID, userIDs)
	if err != nil {
		return req.MakeErr(err)
	}
	for _, goal := range payload.Goals {
		if err := types.ValidateGoal(goal, goalPeriodTypeByPipelineID, activeUserIDs); err != nil {
			return req.MakeErr(err)
		}
	}

	var savedGoals []types.CrmGoal
	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		storedGoals := []types.CrmGoal{}
		query := db.Query(&storedGoals)
		query.CompanyID.Equals(companyID)
		if err := query.Exec(); err != nil {
			return err
		}
		inserts, updates, err := types.PlanGoalSave(payload.Goals, storedGoals)
		if err != nil {
			return err
		}
		for index := range inserts {
			inserts[index].CompanyID, inserts[index].UpdatedBy = companyID, req.User.ID
		}
		for index := range updates {
			updates[index].UpdatedBy = req.User.ID
		}
		if len(inserts) > 0 {
			if err := db.InsertIn(transaction, &inserts); err != nil {
				return err
			}
		}
		if len(updates) > 0 {
			table := db.TableOf[types.CrmGoal]()
			if err := db.UpdateIn(transaction, &updates, table.Target, table.Status, table.UpdatedBy); err != nil {
				return err
			}
		}
		savedGoals = append(inserts, updates...)
		return nil
	})
	if saveError != nil {
		return req.MakeErr("Error al guardar las metas:", saveError)
	}
	return req.MakeResponse(savedGoals)
}
