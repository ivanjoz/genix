package types

import (
	"slices"
	"testing"
)

func TestStagePositionSkipsInactiveStages(t *testing.T) {
	pipeline := CrmPipeline{Stages: []CrmPipelineStage{
		{ID: 5, Status: CrmStatusActive}, {ID: 9, Status: CrmStatusDeleted}, {ID: 2, Status: CrmStatusActive},
	}}
	if StagePosition(pipeline, 5) != 1 || StagePosition(pipeline, 2) != 2 || StagePosition(pipeline, 9) != 0 || StagePosition(pipeline, 77) != 0 {
		t.Fatal("positions count active stages in board order")
	}
}

func TestBuildStageLogRowCarriesBothStates(t *testing.T) {
	source := testPipeline(1)
	target := testPipeline(2)
	before := CrmOpportunity{CompanyID: 3, ID: 4, PipelineID: 1, StageID: 2, OwnerUserID: 7, Amount: 100, ExpectedCloseDate: 20000}
	after := before
	after.PipelineID, after.StageID, after.OwnerUserID, after.Amount = 2, 1, 8, 250

	row := BuildStageLogRow(StageLogKindMoved, before, source, after, target, 9, 0)
	if row.FromPosition != 2 || row.ToPosition != 1 || row.FromOwnerUserID != 7 || row.ToOwnerUserID != 8 ||
		row.FromAmount != 100 || row.ToAmount != 250 || row.UserID != 9 || row.CompanyID != 3 || row.OpportunityID != 4 {
		t.Fatalf("unexpected row: %+v", row)
	}
	if !slices.Equal(row.PipelineIDs(), []int32{2, 1}) {
		t.Fatalf("a transfer belongs to both pipelines' groups: %v", row.PipelineIDs())
	}

	created := BuildStageLogRow(StageLogKindCreated, before, source, before, source, 9, 0)
	if created.FromPipelineID != 0 || created.FromPosition != 0 || created.FromAmount != 0 || !slices.Equal(created.PipelineIDs(), []int32{1}) {
		t.Fatalf("a creation has no before-state: %+v", created)
	}
}

func TestBuildStageLogRowStoresLostReasonOnlyWhenLost(t *testing.T) {
	pipeline := testPipeline(1)
	open := CrmOpportunity{PipelineID: 1, StageID: 1, LostReasonID: 4}
	lost := open
	lost.Outcome = OpportunityOutcomeLost
	if BuildStageLogRow(StageLogKindClosed, open, pipeline, lost, pipeline, 1, 0).LostReasonID != 4 {
		t.Fatal("lost rows carry their reason")
	}
	if BuildStageLogRow(StageLogKindClosed, lost, pipeline, open, pipeline, 1, 0).LostReasonID != 0 {
		t.Fatal("a reopen carries no reason")
	}
}

func TestEditChangesReportedState(t *testing.T) {
	stored := CrmOpportunity{Title: "A", Amount: 10, OwnerUserID: 1, ExpectedCloseDate: 5}
	retitled := stored
	retitled.Title = "B"
	if EditChangesReportedState(stored, retitled) {
		t.Fatal("a title edit is not reported")
	}
	for _, change := range []func(*CrmOpportunity){
		func(o *CrmOpportunity) { o.Amount = 11 },
		func(o *CrmOpportunity) { o.OwnerUserID = 2 },
		func(o *CrmOpportunity) { o.ExpectedCloseDate = 6 },
	} {
		edited := stored
		change(&edited)
		if !EditChangesReportedState(stored, edited) {
			t.Fatalf("expected a logged edit for %+v", edited)
		}
	}
}
