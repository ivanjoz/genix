package types

import "slices"

// AccesoIDCrmFullReport is "CRM Reporte completo" in backend/access.toml. It has no route of its
// own: the report handlers read it to widen the data scope to every pipeline (plan §8.4).
const AccesoIDCrmFullReport int32 = 57

// ReportablePipelineIDs are the pipelines a user may run CRM reports on: the active ones, all of
// them with the full-report access, otherwise those the user is a member of, at any level. A deleted
// pipeline leaves the reports (it cannot be deleted while it holds open opportunities). The browser
// mirrors this rule in routes/crm/reports/reports.ts (reportablePipelineIDs).
func ReportablePipelineIDs(pipelines []CrmPipeline, userID int32, holdsFullReport bool) []int16 {
	pipelineIDs := []int16{}
	for _, pipeline := range pipelines {
		if pipeline.Status == CrmStatusActive && (holdsFullReport || MemberLevel(pipeline, userID) >= PipelineLevelView) {
			pipelineIDs = append(pipelineIDs, pipeline.ID)
		}
	}
	slices.Sort(pipelineIDs)
	return pipelineIDs
}
