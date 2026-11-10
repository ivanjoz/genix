package types

import "testing"

func testPipeline(id int16, members ...CrmPipelineMember) CrmPipeline {
	return CrmPipeline{ID: id, Status: CrmStatusActive, Members: members, Stages: []CrmPipelineStage{
		{ID: 1, Name: "Nuevo", Probability: 10, Status: CrmStatusActive},
		{ID: 2, Name: "Propuesta", Probability: 50, Status: CrmStatusActive},
		{ID: 3, Name: "Viejo", Probability: 5, Status: CrmStatusDeleted},
	}}
}

func TestPlanOpportunityCreate(t *testing.T) {
	pipeline := testPipeline(1, CrmPipelineMember{UserID: 7, Level: PipelineLevelEdit}, CrmPipelineMember{UserID: 8, Level: PipelineLevelView})
	payload := CrmOpportunity{Title: " Venta ", ClientID: 3, StageID: 1, Probability: 10, Outcome: OpportunityOutcomeWon}

	created, err := PlanOpportunityCreate(payload, pipeline, 7, 100)
	if err != nil || created.Title != "Venta" || created.OwnerUserID != 7 || created.CreatedBy != 7 ||
		created.Outcome != OpportunityOutcomeOpen || created.StageEnteredTime != 100 || created.PipelineID != 1 {
		t.Fatalf("unexpected creation: %+v %v", created, err)
	}
	if _, err := PlanOpportunityCreate(payload, pipeline, 8, 100); err == nil {
		t.Fatal("a View member cannot create")
	}
	if _, err := PlanOpportunityCreate(payload, pipeline, 99, 100); err == nil {
		t.Fatal("a non-member cannot create")
	}
	if _, err := PlanOpportunityCreate(CrmOpportunity{Title: "x", ClientID: 3, StageID: 3}, pipeline, 7, 100); err == nil {
		t.Fatal("a deleted stage is refused")
	}
	other := payload
	other.OwnerUserID = 8
	if _, err := PlanOpportunityCreate(other, pipeline, 7, 100); err == nil {
		t.Fatal("Edit cannot assign another owner")
	}
	for _, invalid := range []CrmOpportunity{{ClientID: 3, StageID: 1}, {Title: "x", StageID: 1}, {Title: "x", ClientID: 3, StageID: 1, Probability: 101}} {
		if _, err := PlanOpportunityCreate(invalid, pipeline, 7, 100); err == nil {
			t.Fatalf("expected a validation error for %+v", invalid)
		}
	}
}

func TestPlanOpportunityEdit(t *testing.T) {
	pipeline := testPipeline(1, CrmPipelineMember{UserID: 7, Level: PipelineLevelEdit}, CrmPipelineMember{UserID: 9, Level: PipelineLevelFull})
	stored := CrmOpportunity{ID: 1, PipelineID: 1, StageID: 1, Title: "A", ClientID: 3, CreatedBy: 7, OwnerUserID: 7}
	payload := CrmOpportunity{Title: "B", ClientID: 3, PipelineID: 2, StageID: 2}

	edited, err := PlanOpportunityEdit(stored, payload, pipeline, 7)
	if err != nil || edited.Title != "B" || edited.PipelineID != 1 || edited.StageID != 1 {
		t.Fatalf("stage and pipeline must not change on edit: %+v %v", edited, err)
	}
	stranger := stored
	stranger.CreatedBy, stranger.OwnerUserID = 1, 1
	if _, err := PlanOpportunityEdit(stranger, payload, pipeline, 7); err == nil {
		t.Fatal("Edit cannot edit another user's opportunity")
	}
	if _, err := PlanOpportunityEdit(stranger, payload, pipeline, 9); err != nil {
		t.Fatal("Full edits any opportunity:", err)
	}
	reassigned := payload
	reassigned.OwnerUserID = 9
	if _, err := PlanOpportunityEdit(stored, reassigned, pipeline, 7); err == nil {
		t.Fatal("Edit cannot change the owner")
	}
	closed := stored
	closed.Outcome = OpportunityOutcomeWon
	if _, err := PlanOpportunityEdit(closed, payload, pipeline, 7); err == nil {
		t.Fatal("a closed opportunity cannot be edited")
	}
}

func TestPlanOpportunityMove(t *testing.T) {
	source := testPipeline(1, CrmPipelineMember{UserID: 7, Level: PipelineLevelEdit}, CrmPipelineMember{UserID: 8, Level: PipelineLevelView})
	target := testPipeline(2, CrmPipelineMember{UserID: 7, Level: PipelineLevelEdit})
	target.Fields = []CrmPipelineField{{FieldID: 4, Required: true}}
	opportunity := CrmOpportunity{ID: 1, PipelineID: 1, StageID: 1, Probability: 33}

	moved, log, err := PlanOpportunityMove(opportunity, source, source, 2, 7, 500)
	if err != nil || moved.StageID != 2 || moved.Probability != 50 || moved.PreviousPipelineID != 1 || moved.PreviousStageID != 1 || moved.StageEnteredTime != 500 {
		t.Fatalf("stage move: %+v %v", moved, err)
	}
	if log.FromStageID != 1 || log.ToStageID != 2 || log.FromPipelineID != 1 || log.ToPipelineID != 1 || log.UserID != 7 ||
		log.Kind != StageLogKindMoved || log.FromPosition != 1 || log.ToPosition != 2 {
		t.Fatalf("stage move log: %+v", log)
	}
	if _, _, err := PlanOpportunityMove(opportunity, source, source, 1, 7, 500); err == nil {
		t.Fatal("moving to the same place is refused")
	}
	if _, _, err := PlanOpportunityMove(opportunity, source, source, 2, 8, 500); err == nil {
		t.Fatal("a View member cannot move")
	}
	if _, _, err := PlanOpportunityMove(opportunity, source, source, 3, 7, 500); err == nil {
		t.Fatal("a deleted stage is refused")
	}
	if _, _, err := PlanOpportunityMove(opportunity, source, target, 1, 7, 500); err == nil {
		t.Fatal("the target pipeline's required field is missing")
	}
	opportunity.CustomValues = []CrmCustomFieldValue{{FieldID: 4, Number: 1}}
	crossMoved, _, err := PlanOpportunityMove(opportunity, source, target, 1, 7, 500)
	if err != nil || crossMoved.PipelineID != 2 || crossMoved.PreviousPipelineID != 1 {
		t.Fatalf("cross-pipeline move: %+v %v", crossMoved, err)
	}
	viewOnlyTarget := testPipeline(2, CrmPipelineMember{UserID: 7, Level: PipelineLevelView})
	if _, _, err := PlanOpportunityMove(opportunity, source, viewOnlyTarget, 1, 7, 500); err == nil {
		t.Fatal("a move needs Edit on the target pipeline too")
	}
	opportunity.Outcome = OpportunityOutcomeLost
	if _, _, err := PlanOpportunityMove(opportunity, source, source, 2, 7, 500); err == nil {
		t.Fatal("a closed opportunity cannot move")
	}
}

func TestPlanOpportunityClose(t *testing.T) {
	pipeline := testPipeline(1, CrmPipelineMember{UserID: 7, Level: PipelineLevelEdit}, CrmPipelineMember{UserID: 8, Level: PipelineLevelView})
	opportunity := CrmOpportunity{ID: 1, PipelineID: 1, StageID: 2}

	won, log, err := PlanOpportunityClose(opportunity, pipeline, CrmOpportunityClosePayload{Outcome: OpportunityOutcomeWon, LostReasonID: 9}, false, 7, 600)
	if err != nil || won.Outcome != OpportunityOutcomeWon || won.ClosedTime != 600 || won.LostReasonID != 0 || won.Status != OpportunityStatusClosed || log.ToOutcome != OpportunityOutcomeWon ||
		log.FromOutcome != OpportunityOutcomeOpen || log.Kind != StageLogKindClosed || log.FromPosition != 2 || log.ToPosition != 2 {
		t.Fatalf("won: %+v %v", won, err)
	}
	if _, _, err := PlanOpportunityClose(opportunity, pipeline, CrmOpportunityClosePayload{Outcome: OpportunityOutcomeLost}, false, 7, 600); err == nil {
		t.Fatal("lost needs a valid reason")
	}
	lost, _, err := PlanOpportunityClose(opportunity, pipeline, CrmOpportunityClosePayload{Outcome: OpportunityOutcomeLost, LostReasonID: 2, LostNote: " caro "}, true, 7, 600)
	if err != nil || lost.LostReasonID != 2 || lost.LostNote != "caro" {
		t.Fatalf("lost: %+v %v", lost, err)
	}
	if _, _, err := PlanOpportunityClose(opportunity, pipeline, CrmOpportunityClosePayload{Outcome: OpportunityOutcomeWon}, false, 8, 600); err == nil {
		t.Fatal("a View member cannot close")
	}
	reopened, _, err := PlanOpportunityClose(lost, pipeline, CrmOpportunityClosePayload{Outcome: OpportunityOutcomeOpen}, false, 7, 700)
	if err != nil || reopened.Outcome != OpportunityOutcomeOpen || reopened.ClosedTime != 0 || reopened.LostReasonID != 0 || reopened.Status != OpportunityStatusOpen || reopened.StageEnteredTime != 700 {
		t.Fatalf("reopen: %+v %v", reopened, err)
	}
	if _, _, err := PlanOpportunityClose(opportunity, pipeline, CrmOpportunityClosePayload{Outcome: OpportunityOutcomeOpen}, false, 7, 700); err == nil {
		t.Fatal("reopening an open opportunity is refused")
	}
	onDeletedStage := lost
	onDeletedStage.StageID = 3
	if _, _, err := PlanOpportunityClose(onDeletedStage, pipeline, CrmOpportunityClosePayload{Outcome: OpportunityOutcomeOpen}, false, 7, 700); err == nil {
		t.Fatal("cannot reopen onto a deleted stage")
	}
}
