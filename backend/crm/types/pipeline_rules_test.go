package types

import "testing"

func noOpenOpportunities(int16) bool { return false }

func TestOpportunityPermissions(t *testing.T) {
	opportunity := CrmOpportunity{CreatedBy: 7, OwnerUserID: 8}

	if CanMoveOpportunity(PipelineLevelView) || !CanMoveOpportunity(PipelineLevelEdit) || !CanMoveOpportunity(PipelineLevelFull) {
		t.Fatal("moving needs level Edit or Full")
	}
	if CanCreateOpportunity(PipelineLevelView) || !CanCreateOpportunity(PipelineLevelEdit) {
		t.Fatal("creating needs level Edit or Full")
	}
	// Edit changes only what they created or own; Full changes anything.
	if !CanEditOpportunity(PipelineLevelEdit, opportunity, 7) || !CanEditOpportunity(PipelineLevelEdit, opportunity, 8) {
		t.Fatal("creator and owner at Edit can edit")
	}
	if CanEditOpportunity(PipelineLevelEdit, opportunity, 9) || CanEditOpportunity(PipelineLevelView, opportunity, 7) {
		t.Fatal("Edit cannot edit others' opportunities, View edits nothing")
	}
	if !CanEditOpportunity(PipelineLevelFull, opportunity, 9) || !CanChangeOpportunityOwner(PipelineLevelFull) || CanChangeOpportunityOwner(PipelineLevelEdit) {
		t.Fatal("only Full edits any opportunity and changes the owner")
	}
	if CanMoveToPipeline(PipelineLevelEdit, PipelineLevelView) || CanMoveToPipeline(PipelineLevelEdit, 0) || !CanMoveToPipeline(PipelineLevelEdit, PipelineLevelEdit) {
		t.Fatal("a cross-pipeline move needs Edit on the target as well")
	}
}

func TestMemberLevel(t *testing.T) {
	pipeline := CrmPipeline{Members: []CrmPipelineMember{{UserID: 3, Level: PipelineLevelFull}}}
	if MemberLevel(pipeline, 3) != PipelineLevelFull || MemberLevel(pipeline, 4) != 0 {
		t.Fatal("a non-member has level 0")
	}
}

func TestApplyPipelineEditNewPipeline(t *testing.T) {
	saved, err := ApplyPipelineEdit(CrmPipeline{}, CrmPipeline{
		Name:   " Ventas ",
		Stages: []CrmPipelineStage{{Name: "Nuevo", Probability: 10, Status: 1}, {Name: "Propuesta", Probability: 50, Status: 1}},
		Fields: []CrmPipelineField{{FieldID: 1}},
	}, noOpenOpportunities, map[int16]bool{1: true})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Name != "Ventas" || saved.Stages[0].ID != 1 || saved.Stages[1].ID != 2 || saved.Status != CrmStatusActive {
		t.Fatalf("unexpected pipeline: %+v", saved)
	}
}

func TestApplyPipelineEditKeepsStoredMembers(t *testing.T) {
	stored := CrmPipeline{ID: 1, Members: []CrmPipelineMember{{UserID: 1, Level: PipelineLevelFull}},
		Stages: []CrmPipelineStage{{ID: 1, Name: "Nuevo", Status: CrmStatusActive}}}
	saved, err := ApplyPipelineEdit(stored, CrmPipeline{Name: "X", Members: []CrmPipelineMember{{UserID: 9, Level: 3}},
		Stages: []CrmPipelineStage{{ID: 1, Name: "Nuevo", Status: 1}}}, noOpenOpportunities, nil)
	if err != nil || len(saved.Members) != 1 || saved.Members[0].UserID != 1 {
		t.Fatalf("members must come from the stored record: %+v %v", saved.Members, err)
	}
}

func TestApplyPipelineEditRejections(t *testing.T) {
	stored := CrmPipeline{ID: 1, Stages: []CrmPipelineStage{
		{ID: 1, Name: "Nuevo", Status: CrmStatusActive}, {ID: 2, Name: "Cierre", Status: CrmStatusActive}}}
	holdsStageTwo := func(stageID int16) bool { return stageID == 2 }
	validStage := CrmPipelineStage{ID: 1, Name: "Nuevo", Status: 1}

	cases := map[string]CrmPipeline{
		"missing name":             {Stages: []CrmPipelineStage{validStage}},
		"no stages":                {Name: "X"},
		"only deleted stages":      {Name: "X", Stages: []CrmPipelineStage{{ID: 1, Name: "Nuevo", Status: CrmStatusDeleted}}},
		"probability over":         {Name: "X", Stages: []CrmPipelineStage{{ID: 1, Name: "Nuevo", Probability: 101}}},
		"unknown stage ID":         {Name: "X", Stages: []CrmPipelineStage{validStage, {ID: 9, Name: "Otra"}}},
		"stage with cards removed": {Name: "X", Stages: []CrmPipelineStage{validStage}},
		"stage with cards soft-deleted": {Name: "X", Stages: []CrmPipelineStage{validStage,
			{ID: 2, Name: "Cierre", Status: CrmStatusDeleted}}},
		"unknown field": {Name: "X", Stages: []CrmPipelineStage{validStage, {ID: 2, Name: "Cierre", Status: 1}},
			Fields: []CrmPipelineField{{FieldID: 5}}},
	}
	for caseName, payload := range cases {
		if _, err := ApplyPipelineEdit(stored, payload, holdsStageTwo, map[int16]bool{}); err == nil {
			t.Errorf("%s: expected an error", caseName)
		}
	}
}

func TestApplyPipelineEditRemovesEmptyStageAsSoftDelete(t *testing.T) {
	stored := CrmPipeline{ID: 1, Stages: []CrmPipelineStage{
		{ID: 1, Name: "Nuevo", Status: CrmStatusActive}, {ID: 2, Name: "Viejo", Status: CrmStatusActive}}}
	saved, err := ApplyPipelineEdit(stored, CrmPipeline{Name: "X", Stages: []CrmPipelineStage{{ID: 1, Name: "Nuevo", Status: 1}}},
		noOpenOpportunities, nil)
	if err != nil || len(saved.Stages) != 2 || saved.Stages[1].ID != 2 || saved.Stages[1].Status != CrmStatusDeleted {
		t.Fatalf("an empty removed stage must stay as a soft delete: %+v %v", saved.Stages, err)
	}
}

func TestValidatePipelineMembers(t *testing.T) {
	activeUsers := map[int32]bool{1: true, 2: true}
	if err := ValidatePipelineMembers([]CrmPipelineMember{{1, 1}, {2, 3}}, activeUsers); err != nil {
		t.Fatal(err)
	}
	invalidLists := [][]CrmPipelineMember{{{1, 0}}, {{1, 4}}, {{3, 1}}, {{1, 1}, {1, 2}}}
	for _, members := range invalidLists {
		if ValidatePipelineMembers(members, activeUsers) == nil {
			t.Errorf("expected an error for %+v", members)
		}
	}
}
