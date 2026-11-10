package types

import "slices"

// AccesoIDCrmFullReport is "CRM Reporte completo" in backend/access.toml. It has no route of its
// own: the report handlers read it to widen the data scope to every pipeline (plan §8.4).
const AccesoIDCrmFullReport int32 = 57

// ReportablePipelineIDs are the pipelines a user may run CRM reports on: every pipeline with the
// full-report access (soft-deleted ones too, their history still counts), otherwise only the
// active pipelines the user is a member of, at any level.
func ReportablePipelineIDs(pipelines []CrmPipeline, userID int32, holdsFullReport bool) []int16 {
	pipelineIDs := []int16{}
	for _, pipeline := range pipelines {
		if holdsFullReport || (pipeline.Status == CrmStatusActive && MemberLevel(pipeline, userID) >= PipelineLevelView) {
			pipelineIDs = append(pipelineIDs, pipeline.ID)
		}
	}
	slices.Sort(pipelineIDs)
	return pipelineIDs
}

// CrmReportScope is the body of GET.crm-report-scope.
type CrmReportScope struct {
	PipelineIDs     []int16
	HoldsFullReport bool
}
