package crm

import (
	"encoding/json"
	"slices"

	"app/core"
	"app/crm/types"
	"app/db"
)

// GetCrmForecastSubmissions returns the submissions of one pipeline for a range of weeks
// (?week-start&week-end), as cache groups by (Week, PipelineID). The browser asks for the weeks its
// goal period spans and keeps only that period's rows. Reading needs the same scope as the reports
// (plan §8.4).
func GetCrmForecastSubmissions(req *core.HandlerArgs) core.HandlerResponse {
	pipelineID := int16(req.GetQueryInt("pipeline-id"))
	weekStart, weekEnd := req.GetQueryInt16("week-start"), req.GetQueryInt16("week-end")
	if pipelineID <= 0 || weekStart <= 0 || weekEnd <= 0 {
		return req.MakeErr("Debe enviar pipeline-id, week-start y week-end.")
	}
	// A month spans at most 6 weeks; each week is one group to probe.
	if weekEnd < weekStart || weekEnd-weekStart > 5 {
		return req.MakeErr("El rango de semanas debe cubrir como máximo 6 semanas.")
	}
	heldGroups, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}
	reportablePipelineIDs, err := loadReportablePipelineIDs(req.User.CompanyID, req.User.ID)
	if err != nil {
		return req.MakeErr(err)
	}
	if !slices.Contains(reportablePipelineIDs, pipelineID) {
		return req.MakeErr("No tiene acceso a los reportes de este pipeline.")
	}
	submissionGroups := []db.RecordGroup[types.CrmForecastSubmission]{}
	query := db.QueryIndexGroup(&submissionGroups).CompanyID.Equals(req.User.CompanyID)
	for _, heldGroup := range heldGroups {
		query.IncludeCachedGroup(heldGroup.GroupHash, heldGroup.LastWrite)
	}
	query.Week.Between(weekStart, weekEnd).PipelineID.Equals(pipelineID)
	if err := query.Exec(); err != nil {
		core.Log("Error querying CRM forecast submission groups:", err)
		return req.MakeErr("Error al obtener los forecasts enviados.")
	}
	return core.MakeResponse(req, &submissionGroups)
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
