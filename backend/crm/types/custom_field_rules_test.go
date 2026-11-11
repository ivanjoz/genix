package types

import "testing"

func TestApplyCustomFieldEdit(t *testing.T) {
	saved, err := ApplyCustomFieldEdit(CrmCustomField{}, CrmCustomField{Name: "Origen", Type: CustomFieldSelect,
		Options: []CrmCustomFieldOption{{Name: "Web", Status: 1}, {Name: "Feria", Status: 1}}}, false)
	if err != nil || saved.Options[0].ID != 1 || saved.Options[1].ID != 2 || saved.Options[0].Status != CrmStatusActive {
		t.Fatalf("options get server IDs: %+v %v", saved.Options, err)
	}

	// An option missing from the payload is soft-deleted, never removed.
	resaved, err := ApplyCustomFieldEdit(saved, CrmCustomField{Name: "Origen", Type: CustomFieldSelect,
		Options: []CrmCustomFieldOption{{ID: 2, Name: "Feria", Status: 1}}}, true)
	if err != nil || len(resaved.Options) != 2 || resaved.Options[1].ID != 1 || resaved.Options[1].Status != CrmStatusDeleted {
		t.Fatalf("a removed option must stay as a soft delete: %+v %v", resaved.Options, err)
	}

	// The type is frozen once a value exists.
	saved.ID = 1
	if _, err := ApplyCustomFieldEdit(saved, CrmCustomField{Name: "Origen", Type: CustomFieldText}, true); err == nil {
		t.Fatal("changing the type of a field with values must fail")
	}
	if _, err := ApplyCustomFieldEdit(saved, CrmCustomField{Name: "Origen", Type: CustomFieldText}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyCustomFieldEdit(CrmCustomField{}, CrmCustomField{Name: "X", Type: CustomFieldSelect}, false); err == nil {
		t.Fatal("a select field without options must fail")
	}
	if _, err := ApplyCustomFieldEdit(CrmCustomField{}, CrmCustomField{Name: "X", Type: 99}, false); err == nil {
		t.Fatal("an unknown type must fail")
	}
}

func TestValidateCustomValues(t *testing.T) {
	catalog := map[int16]CrmCustomField{
		1: {ID: 1, Name: "Origen", Type: CustomFieldSelect, Status: CrmStatusActive,
			Options: []CrmCustomFieldOption{{ID: 1, Name: "Web", Status: CrmStatusActive}, {ID: 2, Name: "Feria", Status: CrmStatusDeleted}}},
		2: {ID: 2, Name: "Nota", Type: CustomFieldText, Status: CrmStatusActive},
		3: {ID: 3, Name: "Responsable", Type: CustomFieldUser, Status: CrmStatusActive},
	}
	// The origin is required from the first stage: from creation on.
	requiredOrigin := testPipeline(1)
	requiredOrigin.Fields = []CrmPipelineField{{FieldID: 1, RequiredFromStageID: 1}}
	activeUsers := map[int32]bool{5: true}

	valid := []CrmCustomFieldValue{{FieldID: 1, Number: 1}, {FieldID: 2, Text: "hola"}, {FieldID: 3, Number: 5}}
	if err := ValidateCustomValues(valid, catalog, requiredOrigin, 1, activeUsers); err != nil {
		t.Fatal(err)
	}

	invalidValues := map[string][]CrmCustomFieldValue{
		"required field missing": {{FieldID: 2, Text: "hola"}},
		"deleted option":         {{FieldID: 1, Number: 2}},
		"unknown option":         {{FieldID: 1, Number: 9}},
		"unknown field":          {{FieldID: 1, Number: 1}, {FieldID: 8, Text: "x"}},
		"duplicate field":        {{FieldID: 1, Number: 1}, {FieldID: 1, Number: 1}},
		"empty text":             {{FieldID: 1, Number: 1}, {FieldID: 2}},
		"inactive user":          {{FieldID: 1, Number: 1}, {FieldID: 3, Number: 6}},
	}
	for caseName, values := range invalidValues {
		if ValidateCustomValues(values, catalog, requiredOrigin, 1, activeUsers) == nil {
			t.Errorf("%s: expected an error", caseName)
		}
	}
}

// A field is required to enter its stage or any later one by board position; a standard field is
// required by the stage that lists it and every later stage.
func TestMissingRequiredFieldsFollowsStagePosition(t *testing.T) {
	pipeline := testPipeline(1)
	pipeline.Fields = []CrmPipelineField{{FieldID: 1, RequiredFromStageID: 2}, {FieldID: 2}, {FieldID: 3, RequiredFromStageID: 1}}
	pipeline.Stages[1].RequiredStandardFields = []int8{StandardFieldAmount}
	empty := CrmOpportunity{CustomValues: []CrmCustomFieldValue{{FieldID: 2}}}

	atFirst := MissingRequiredFields(pipeline, 1, empty)
	if len(atFirst.FieldIDs) != 1 || atFirst.FieldIDs[0] != 3 || len(atFirst.StandardFields) != 0 {
		t.Fatalf("first stage requires only field 3: %+v", atFirst)
	}
	atSecond := MissingRequiredFields(pipeline, 2, empty)
	if len(atSecond.FieldIDs) != 2 || len(atSecond.StandardFields) != 1 || atSecond.StandardFields[0] != StandardFieldAmount {
		t.Fatalf("second stage requires fields 1 and 3 and the amount: %+v", atSecond)
	}
	full := CrmOpportunity{Amount: 100, CustomValues: []CrmCustomFieldValue{{FieldID: 1}, {FieldID: 3}}}
	if !MissingRequiredFields(pipeline, 2, full).IsEmpty() {
		t.Fatal("a complete opportunity lacks nothing")
	}
	// A stage that is no longer active requires nothing.
	if !MissingRequiredFields(pipeline, 3, empty).IsEmpty() {
		t.Fatal("a deleted stage requires nothing")
	}
}

func TestPipelineEditValidatesRequiredFromStage(t *testing.T) {
	known := map[int16]bool{1: true}
	stored := CrmPipeline{}
	payload := CrmPipeline{Name: "P", Stages: []CrmPipelineStage{{Name: "A", Status: CrmStatusActive}, {Name: "B", Status: CrmStatusActive, RequiredStandardFields: []int8{StandardFieldExpectedCloseDate}}},
		Fields: []CrmPipelineField{{FieldID: 1, RequiredFromStageID: 2}}}
	saved, err := ApplyPipelineEdit(stored, payload, func(int16) bool { return false }, known)
	if err != nil || saved.Fields[0].RequiredFromStageID != 2 {
		t.Fatalf("a new pipeline may name the stages it allocates (IDs 1..n): %+v %v", saved, err)
	}

	payload.Fields[0].RequiredFromStageID = 9
	if _, err := ApplyPipelineEdit(stored, payload, func(int16) bool { return false }, known); err == nil {
		t.Fatal("a required-from stage that does not exist is refused")
	}
	payload.Fields[0].RequiredFromStageID = 2
	payload.Stages[1].RequiredStandardFields = []int8{7}
	if _, err := ApplyPipelineEdit(stored, payload, func(int16) bool { return false }, known); err == nil {
		t.Fatal("an unknown standard field is refused")
	}
	payload.Stages[1].RequiredStandardFields = []int8{StandardFieldAmount, StandardFieldAmount}
	if _, err := ApplyPipelineEdit(stored, payload, func(int16) bool { return false }, known); err == nil {
		t.Fatal("a repeated standard field is refused")
	}

	// Removing the stage a field is required from is refused while the field still names it.
	payload.Stages[1].RequiredStandardFields = nil
	saved, _ = ApplyPipelineEdit(stored, payload, func(int16) bool { return false }, known)
	payload.Stages = []CrmPipelineStage{saved.Stages[0], {ID: 2, Name: "B", Status: CrmStatusDeleted}}
	if _, err := ApplyPipelineEdit(saved, payload, func(int16) bool { return false }, known); err == nil {
		t.Fatal("deleting the stage a field is required from is refused")
	}
}
