package crm

import (
	"fmt"
	"slices"

	"app/core"
	coreTypes "app/core/types"
	"app/crm/types"
	"app/db"
)

// GetCrmReportStageLog returns the stage log of one pipeline for a range of months of ONE year,
// as cache groups by (Period, pipeline): a closed month never changes, so the browser downloads it
// once. Every report (net progress, daily state, waterfall, funnel, scorecard, lost reasons) is
// computed in the browser from these rows, which carry both states of each change.
func GetCrmReportStageLog(req *core.HandlerArgs) core.HandlerResponse {
	groupRequest, err := parseCrmReportGroupRequest(req)
	if err != nil {
		return req.MakeErr(err)
	}
	stageLogGroups := []db.RecordGroup[types.CrmOpportunityStageLog]{}
	query := db.QueryIndexGroup(&stageLogGroups).CompanyID.Equals(req.User.CompanyID)
	for _, heldGroup := range groupRequest.heldGroups {
		query.IncludeCachedGroup(heldGroup.GroupHash, heldGroup.LastWrite)
	}
	query.Period.Between(groupRequest.periodStart, groupRequest.periodEnd).PipelineIDs.Contains(int32(groupRequest.pipelineID))
	if err := query.Exec(); err != nil {
		core.Log("Error querying CRM stage log groups:", err)
		return req.MakeErr("Error al obtener el historial de oportunidades.")
	}
	return core.MakeResponse(req, &stageLogGroups)
}

// GetCrmReportActivities returns the activity entries written in one pipeline for a range of
// months of ONE year, as cache groups by (Period, PipelineID): the seller scorecard counts them by
// author. Only the counting fields travel; the text and the files stay in the opportunity panel.
func GetCrmReportActivities(req *core.HandlerArgs) core.HandlerResponse {
	groupRequest, err := parseCrmReportGroupRequest(req)
	if err != nil {
		return req.MakeErr(err)
	}
	activityGroups := []db.RecordGroup[types.CrmActivity]{}
	query := db.QueryIndexGroup(&activityGroups).CompanyID.Equals(req.User.CompanyID)
	for _, heldGroup := range groupRequest.heldGroups {
		query.IncludeCachedGroup(heldGroup.GroupHash, heldGroup.LastWrite)
	}
	query.Period.Between(groupRequest.periodStart, groupRequest.periodEnd).PipelineID.Equals(groupRequest.pipelineID)
	if err := query.Exec(); err != nil {
		core.Log("Error querying CRM activity groups:", err)
		return req.MakeErr("Error al obtener las actividades.")
	}
	for groupIndex := range activityGroups {
		for recordIndex := range activityGroups[groupIndex].Records {
			activity := &activityGroups[groupIndex].Records[recordIndex]
			activity.Text, activity.Attachments = "", nil
		}
	}
	return core.MakeResponse(req, &activityGroups)
}

type crmReportGroupRequest struct {
	pipelineID  int16
	periodStart int16
	periodEnd   int16
	heldGroups  []db.GroupIndexCache
}

// parseCrmReportGroupRequest reads ?pipeline-id&period-start&period-end of a cache-groups report
// route and refuses a pipeline outside the user's report scope (plan §8.4).
func parseCrmReportGroupRequest(req *core.HandlerArgs) (crmReportGroupRequest, error) {
	groupRequest := crmReportGroupRequest{
		pipelineID:  int16(req.GetQueryInt("pipeline-id")),
		periodStart: req.GetQueryInt16("period-start"),
		periodEnd:   req.GetQueryInt16("period-end"),
	}
	if groupRequest.pipelineID <= 0 {
		return groupRequest, fmt.Errorf("Debe enviar pipeline-id.")
	}
	// YYMM periods: within one year the steps are contiguous, so each step is one real month.
	periodStart, periodEnd := groupRequest.periodStart, groupRequest.periodEnd
	if periodStart <= 0 || periodEnd < periodStart || periodStart/100 != periodEnd/100 || periodEnd%100 > 12 || periodStart%100 < 1 {
		return groupRequest, fmt.Errorf("El rango de periodos debe estar dentro de un mismo año (YYMM).")
	}
	reportablePipelineIDs, err := loadReportablePipelineIDs(req.User.CompanyID, req.User.ID)
	if err != nil {
		return groupRequest, err
	}
	if !slices.Contains(reportablePipelineIDs, groupRequest.pipelineID) {
		return groupRequest, fmt.Errorf("No tiene acceso a los reportes de este pipeline.")
	}
	groupRequest.heldGroups, err = core.ExtractGroupIndexCacheValues(req)
	return groupRequest, err
}

// loadReportablePipelineIDs answers the pipelines the user may report on (types.ReportablePipelineIDs).
// The browser derives the same list from its pipelines cache (routes/crm/reports/reports.ts,
// reportablePipelineIDs); this is the server's own check.
func loadReportablePipelineIDs(companyID, userID int32) ([]int16, error) {
	pipelines := []types.CrmPipeline{}
	query := db.Query(&pipelines)
	query.Select(query.ID, query.Status, query.Members).CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los pipelines: %v", err)
	}
	holdsFullReport, err := userHoldsAccess(companyID, userID, types.AccesoIDCrmFullReport)
	if err != nil {
		return nil, err
	}
	return types.ReportablePipelineIDs(pipelines, userID, holdsFullReport), nil
}

// userHoldsAccess reads the user's computed grants. The request gate only answers about the
// accesses that authorized the route, so an access with no route of its own (like the full
// report) has to be read from the user record. User 1 holds every access.
func userHoldsAccess(companyID, userID, accesoID int32) (bool, error) {
	if userID == 1 {
		return true, nil
	}
	users := []coreTypes.User{}
	query := db.Query(&users)
	query.Select(query.ID, query.Status, query.AccesosComputed, query.AccesosSubComputed).
		CompanyID.Equals(companyID).ID.Equals(userID)
	if err := query.Exec(); err != nil {
		return false, fmt.Errorf("error al leer el usuario: %v", err)
	}
	if len(users) == 0 || users[0].Status != 1 {
		return false, nil
	}
	grants, err := core.DecodeAccesosGrants(users[0].AccesosComputed, users[0].AccesosSubComputed)
	if err != nil {
		return false, fmt.Errorf("error al leer los accesos del usuario: %v", err)
	}
	return slices.ContainsFunc(grants, func(grant core.AccesoGrant) bool { return grant.AccesoID == accesoID }), nil
}
