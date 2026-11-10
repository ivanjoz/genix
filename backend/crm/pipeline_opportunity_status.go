package crm

import (
	"fmt"
	"slices"

	"app/core"
	"app/crm/types"
	"app/db"
)

// insertStageLogs writes the stage log rows of one change and, in the same transaction, the
// opportunity's per-pipeline status rows the timeline reads. Every writer of the log goes through
// here, so the two never drift apart. The caller holds the opportunity (locked or just created).
func insertStageLogs(transaction *db.Transaction, companyID int32, stageLogs []types.CrmOpportunityStageLog) error {
	if err := db.InsertIn(transaction, &stageLogs); err != nil {
		return err
	}
	for _, stageLog := range stageLogs {
		storedRows := []types.CrmPipelineOpportunityStatus{}
		query := db.Query(&storedRows)
		query.CompanyID.Equals(companyID).OpportunityID.Equals(stageLog.OpportunityID)
		if err := query.Exec(); err != nil {
			return fmt.Errorf("error al leer el estado de la oportunidad por pipeline: %v", err)
		}
		inserts, updates := types.PlanPipelineStatusWrites(storedRows, stageLog)
		if len(inserts) > 0 {
			if err := db.InsertIn(transaction, &inserts); err != nil {
				return err
			}
		}
		if len(updates) > 0 {
			table := db.TableOf[types.CrmPipelineOpportunityStatus]()
			if err := db.UpdateIn(transaction, &updates, table.LastDate, table.Visits, table.UpdatedBy); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetCrmPipelineOpportunityStatus syncs, by delta, the path of every opportunity that went through
// one pipeline (?pipeline-id=): the timeline report's cards. The browser filters by StartDate /
// LastDate and joins titles and owners through GET.crm-opportunity-ids.
func GetCrmPipelineOpportunityStatus(req *core.HandlerArgs) core.HandlerResponse {
	pipelineID := int16(req.GetQueryInt("pipeline-id"))
	scope, err := loadCrmReportScope(req.User.CompanyID, req.User.ID)
	if err != nil {
		return req.MakeErr(err)
	}
	if !slices.Contains(scope.PipelineIDs, pipelineID) {
		return req.MakeErr("No tiene acceso a los reportes de este pipeline.")
	}
	statusRows := []types.CrmPipelineOpportunityStatus{}
	query := db.Query(&statusRows)
	query.Select().CompanyID.Equals(req.User.CompanyID).PipelineID.Equals(pipelineID).
		Delta(req.GetDeltaCursor(), types.CrmStatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el recorrido de las oportunidades:", err)
	}
	return req.MakeResponse(statusRows)
}

// GetCrmOpportunitiesByIDs resolves opportunity IDs for the reports (title, owner), versioned per
// slot. Only opportunities that went through a pipeline the user may report on come back, so a
// hand-made request cannot read other pipelines.
func GetCrmOpportunitiesByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	partitions, err := db.QueryCachedIDs[types.CrmOpportunity](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener las oportunidades.", err)
	}

	scope, err := loadCrmReportScope(req.User.CompanyID, req.User.ID)
	if err != nil {
		return req.MakeErr(err)
	}
	opportunityIDs := []int32{}
	for _, partition := range partitions {
		for _, opportunity := range partition.Records {
			opportunityIDs = append(opportunityIDs, opportunity.ID)
		}
	}
	allowedIDs := map[int32]bool{}
	if len(opportunityIDs) > 0 {
		statusRows := []types.CrmPipelineOpportunityStatus{}
		query := db.Query(&statusRows)
		query.Select(query.ID, query.PipelineID, query.OpportunityID).
			CompanyID.Equals(req.User.CompanyID).OpportunityID.In(opportunityIDs...)
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al validar el acceso a las oportunidades:", err)
		}
		for _, statusRow := range statusRows {
			if slices.Contains(scope.PipelineIDs, statusRow.PipelineID) {
				allowedIDs[statusRow.OpportunityID] = true
			}
		}
	}
	for index := range partitions {
		partitions[index].Records = slices.DeleteFunc(partitions[index].Records, func(opportunity types.CrmOpportunity) bool {
			return !allowedIDs[opportunity.ID]
		})
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}
