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
	requiredOrigin := []CrmPipelineField{{FieldID: 1, Required: true}}
	activeUsers := map[int32]bool{5: true}

	valid := []CrmCustomFieldValue{{FieldID: 1, Number: 1}, {FieldID: 2, Text: "hola"}, {FieldID: 3, Number: 5}}
	if err := ValidateCustomValues(valid, catalog, requiredOrigin, activeUsers); err != nil {
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
		if ValidateCustomValues(values, catalog, requiredOrigin, activeUsers) == nil {
			t.Errorf("%s: expected an error", caseName)
		}
	}

	missing := MissingRequiredFields([]CrmCustomFieldValue{{FieldID: 2}}, []CrmPipelineField{{FieldID: 1, Required: true}, {FieldID: 2, Required: true}})
	if len(missing) != 1 || missing[0] != 1 {
		t.Fatalf("missing required fields: %v", missing)
	}
}
