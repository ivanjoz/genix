package types

import (
	"testing"

	"app/core"
)

func TestPlanPipelineStatusWrites(t *testing.T) {
	const firstDay int16 = 20400
	timeOfDay := func(day int16) int32 { return int32((int64(day)*86400 - 1e9) / 2) }
	logRow := func(kind int8, day int16, fromPipeline, fromStage, toPipeline, toStage int16, toOutcome int8) CrmOpportunityStageLog {
		return CrmOpportunityStageLog{CompanyID: 1, OpportunityID: 20, Kind: kind, Time: timeOfDay(day),
			FromPipelineID: fromPipeline, FromStageID: fromStage, ToPipelineID: toPipeline, ToStageID: toStage, ToOutcome: toOutcome}
	}
	if core.SUnixTimeToUnixDay(timeOfDay(firstDay)) != firstDay {
		t.Fatal("test clock helper is off")
	}

	// Created in pipeline 1, stage 1.
	inserts, updates := PlanPipelineStatusWrites(nil, logRow(StageLogKindCreated, firstDay, 0, 0, 1, 1, OpportunityOutcomeOpen))
	if len(inserts) != 1 || len(updates) != 0 || inserts[0].StartDate != firstDay || inserts[0].PipelineID != 1 {
		t.Fatalf("creation starts a row: %+v %+v", inserts, updates)
	}
	rows := inserts

	// Moved to stage 2 three days later, and back to 1 on the same day: both visits stay, in order.
	_, updates = PlanPipelineStatusWrites(rows, logRow(StageLogKindMoved, firstDay+3, 1, 1, 1, 2, OpportunityOutcomeOpen))
	rows = updates
	_, updates = PlanPipelineStatusWrites(rows, logRow(StageLogKindMoved, firstDay+3, 1, 2, 1, 1, OpportunityOutcomeOpen))
	rows = updates
	want := []CrmOpportunityStageVisit{{StageID: 1}, {StageID: 2, DaysPassed: 3}, {StageID: 1, DaysPassed: 3}}
	if len(rows[0].Visits) != 3 || rows[0].Visits[1] != want[1] || rows[0].Visits[2] != want[2] || rows[0].LastDate != firstDay+3 {
		t.Fatalf("visits = %+v", rows[0].Visits)
	}

	// An edit adds nothing.
	if inserts, updates := PlanPipelineStatusWrites(rows, logRow(StageLogKindEdited, firstDay+4, 1, 1, 1, 1, 0)); len(inserts)+len(updates) != 0 {
		t.Error("an edit must not add a visit")
	}

	// Transfer to pipeline 2: pipeline 1 gets "left", pipeline 2 starts a row of its own.
	inserts, updates = PlanPipelineStatusWrites(rows, logRow(StageLogKindMoved, firstDay+10, 1, 1, 2, 5, OpportunityOutcomeOpen))
	if len(updates) != 1 || updates[0].Visits[3].Outcome != VisitOutcomeLeft || updates[0].Visits[3].DaysPassed != 10 {
		t.Fatalf("source row should end with left: %+v", updates)
	}
	if len(inserts) != 1 || inserts[0].PipelineID != 2 || inserts[0].StartDate != firstDay+10 || inserts[0].Visits[0].StageID != 5 {
		t.Fatalf("target row should start on the move day: %+v", inserts)
	}
	rows = append(updates, inserts...)

	// Won in pipeline 2 two days later: a dated visit with the won outcome, counted from its own start.
	_, updates = PlanPipelineStatusWrites(rows, logRow(StageLogKindClosed, firstDay+12, 2, 5, 2, 5, OpportunityOutcomeWon))
	if len(updates) != 1 || updates[0].PipelineID != 2 || updates[0].Visits[1] != (CrmOpportunityStageVisit{StageID: 5, Outcome: OpportunityOutcomeWon, DaysPassed: 2}) {
		t.Fatalf("won visit = %+v", updates)
	}
}
