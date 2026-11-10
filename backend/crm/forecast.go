package crm

import (
	"encoding/json"
	"slices"

	"app/core"
	"app/crm/types"
	"app/db"
)

// GetCrmForecastSubmissions returns every submission of one pipeline and one month, oldest first
// in the browser's view. Reading needs the same scope as the reports (plan §8.4).
func GetCrmForecastSubmissions(req *core.HandlerArgs) core.HandlerResponse {
	pipelineID := int16(req.GetQueryInt("pipeline-id"))
	period := req.GetQueryInt16("period")
	if pipelineID <= 0 || period <= 0 {
		return req.MakeErr("Debe enviar pipeline-id y period.")
	}
	scope, err := loadCrmReportScope(req.User.CompanyID, req.User.ID)
	if err != nil {
		return req.MakeErr(err)
	}
	if !slices.Contains(scope.PipelineIDs, pipelineID) {
		return req.MakeErr("No tiene acceso a los reportes de este pipeline.")
	}
	submissions := []types.CrmForecastSubmission{}
	query := db.Query(&submissions)
	query.Select().CompanyID.Equals(req.User.CompanyID).PipelineID.Equals(pipelineID).Period.Equals(period)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los forecasts enviados:", err)
	}
	return req.MakeResponse(submissions)
}

// PostCrmForecastSubmission appends one forecast submission. The server recomputes the forecast
// from the opportunities at this moment and stores it next to what the user declared, so the
// report can contrast "what was said" with "what the data said then".
func PostCrmForecastSubmission(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmForecastSubmissionPayload{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	pipeline, err := loadPipeline(companyID, payload.PipelineID)
	if err != nil {
		return req.MakeErr(err)
	}

	teams, err := loadActiveTeams(companyID)
	if err != nil {
		return req.MakeErr(err)
	}
	var team types.CrmTeam
	ownerIDs := map[int32]bool{req.User.ID: true}
	if payload.TeamID != 0 {
		for _, candidate := range teams {
			if candidate.ID == payload.TeamID {
				team = candidate
			}
		}
		ownerIDs = map[int32]bool{}
		for _, memberID := range team.MemberUserIDs {
			ownerIDs[memberID] = true
		}
	}

	opportunities := []types.CrmOpportunity{}
	query := db.Query(&opportunities)
	query.Select(query.ID, query.PipelineID, query.OwnerUserID, query.StageID, query.Amount, query.Outcome,
		query.ExpectedCloseDate, query.ClosedTime).CompanyID.Equals(companyID).PipelineID.Equals(pipeline.ID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al leer las oportunidades:", err)
	}
	computed := types.ComputeForecast(opportunities, pipeline, payload.Period, ownerIDs)

	submission, err := types.PlanForecastSubmission(payload, pipeline.GoalPeriodType, computed, req.User.ID,
		types.MemberLevel(pipeline, req.User.ID), team, core.SUnixTime())
	if err != nil {
		return req.MakeErr(err)
	}
	submission.CompanyID = companyID
	records := []types.CrmForecastSubmission{submission}
	if err := db.Insert(&records); err != nil {
		return req.MakeErr("Error al guardar el forecast:", err)
	}
	return req.MakeResponse(records[0])
}
