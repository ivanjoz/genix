package types

import "app/core"

// pipelineVisit is one state a stage log row makes the opportunity enter, in one pipeline.
type pipelineVisit struct {
	pipelineID int16
	stageID    int16
	outcome    int8
}

// visitsOfStageLog: the states a change makes the opportunity enter. A move to another pipeline
// enters two: "left" in the source and its new stage in the target. Edits (amount, owner, close
// date) enter none: the timeline draws stages and outcomes only.
func visitsOfStageLog(stageLog CrmOpportunityStageLog) []pipelineVisit {
	if stageLog.Kind == StageLogKindEdited {
		return nil
	}
	entered := pipelineVisit{pipelineID: stageLog.ToPipelineID, stageID: stageLog.ToStageID, outcome: stageLog.ToOutcome}
	if stageLog.Kind == StageLogKindMoved && stageLog.FromPipelineID != stageLog.ToPipelineID {
		left := pipelineVisit{pipelineID: stageLog.FromPipelineID, stageID: stageLog.FromStageID, outcome: VisitOutcomeLeft}
		return []pipelineVisit{left, entered}
	}
	return []pipelineVisit{entered}
}

// PlanPipelineStatusWrites applies one stage log row to the opportunity's pipeline rows (all of
// them, as stored) and answers the rows to insert and to update. A pipeline without a row starts
// one on that day; an opportunity coming back to a pipeline continues its row.
func PlanPipelineStatusWrites(storedRows []CrmPipelineOpportunityStatus, stageLog CrmOpportunityStageLog) (inserts, updates []CrmPipelineOpportunityStatus) {
	day := core.SUnixTimeToUnixDay(stageLog.Time)
	for _, visit := range visitsOfStageLog(stageLog) {
		rowIndex := -1
		for index, row := range storedRows {
			if row.PipelineID == visit.pipelineID {
				rowIndex = index
			}
		}
		if rowIndex < 0 {
			inserts = append(inserts, CrmPipelineOpportunityStatus{
				CompanyID: stageLog.CompanyID, PipelineID: visit.pipelineID, OpportunityID: stageLog.OpportunityID,
				StartDate: day, LastDate: day, Status: CrmStatusActive, UpdatedBy: stageLog.UserID,
				Visits: []CrmOpportunityStageVisit{{StageID: visit.stageID, Outcome: visit.outcome}},
			})
			continue
		}
		row := storedRows[rowIndex]
		row.Visits = append(append([]CrmOpportunityStageVisit{}, row.Visits...),
			CrmOpportunityStageVisit{StageID: visit.stageID, Outcome: visit.outcome, DaysPassed: day - row.StartDate})
		row.LastDate, row.UpdatedBy = day, stageLog.UserID
		storedRows[rowIndex] = row
		updates = append(updates, row)
	}
	return inserts, updates
}
