package types

import (
	"testing"

	"app/core"
)

const testFileKey = "crm/c1/o7/abcdefghij012345/acta.pdf"

func TestPlanActivityAcceptsSystemAndCompanyTypes(t *testing.T) {
	planned, err := PlanActivity(CrmActivity{ID: 99, OpportunityID: 7, TypeID: ActivityTypeCall, Text: "  Llamó  "}, 1, 2, false, 3, 500)
	if err != nil {
		t.Fatalf("system type rejected: %v", err)
	}
	// The pipeline and the report period come from the server, never from the payload.
	if planned.ID != 0 || planned.Text != "Llamó" || planned.AuthorID != 3 || planned.Created != 500 || planned.CompanyID != 1 ||
		planned.PipelineID != 2 || planned.Period != core.PeriodOfDay(core.SUnixTimeToUnixDay(500)) {
		t.Fatalf("unexpected planned activity: %+v", planned)
	}
	if _, err := PlanActivity(CrmActivity{OpportunityID: 7, TypeID: 41, Text: "x"}, 1, 2, true, 3, 500); err != nil {
		t.Fatalf("active company type rejected: %v", err)
	}
}

func TestPlanActivityRejectsUnknownTypes(t *testing.T) {
	cases := []struct {
		typeID              int16
		companyTypeIsActive bool
	}{
		{0, true}, {39, true}, {41, false}, {-1, true},
	}
	for _, testCase := range cases {
		if _, err := PlanActivity(CrmActivity{OpportunityID: 7, TypeID: testCase.typeID, Text: "x"}, 1, 2, testCase.companyTypeIsActive, 3, 500); err == nil {
			t.Fatalf("type %d (company active %v) should be rejected", testCase.typeID, testCase.companyTypeIsActive)
		}
	}
}

func TestPlanActivityNeedsTextOrFile(t *testing.T) {
	if _, err := PlanActivity(CrmActivity{OpportunityID: 7, TypeID: ActivityTypeNote, Text: "   "}, 1, 2, false, 3, 500); err == nil {
		t.Fatal("an empty activity should be rejected")
	}
	fileOnly := CrmActivity{OpportunityID: 7, TypeID: ActivityTypeNote,
		Attachments: []CrmAttachment{{Key: testFileKey, Name: "acta.pdf", Size: 10, UploadedBy: 99}}}
	planned, err := PlanActivity(fileOnly, 1, 2, false, 3, 500)
	if err != nil {
		t.Fatalf("a file-only activity should be accepted: %v", err)
	}
	if planned.Attachments[0].UploadedBy != 3 || planned.Attachments[0].UploadedAt != 500 {
		t.Fatalf("the uploader is not stamped by the server: %+v", planned.Attachments[0])
	}
}

func TestPlanActivityRejectsForeignFiles(t *testing.T) {
	foreignKeys := []string{
		"crm/c2/o7/abcdefghij012345/acta.pdf", // another company
		"crm/c1/o8/abcdefghij012345/acta.pdf", // another opportunity
		"projects/c1/p7/abcdefghij012345/acta.pdf",
		"crm/c1/o7/short/acta.pdf",
	}
	for _, key := range foreignKeys {
		activity := CrmActivity{OpportunityID: 7, TypeID: ActivityTypeNote, Attachments: []CrmAttachment{{Key: key, Name: "acta.pdf", Size: 10}}}
		if _, err := PlanActivity(activity, 1, 2, false, 3, 500); err == nil {
			t.Fatalf("file %q should be rejected", key)
		}
	}
	tooBig := CrmActivity{OpportunityID: 7, TypeID: ActivityTypeNote, Attachments: []CrmAttachment{{Key: testFileKey, Name: "a", Size: MaxAttachmentBytes + 1}}}
	if _, err := PlanActivity(tooBig, 1, 2, false, 3, 500); err == nil {
		t.Fatal("an oversized file should be rejected")
	}
}

func TestNormalizeActivityType(t *testing.T) {
	valid := CrmActivityType{Name: "  Cata  ", Icon: "icon-[fa--flag]"}
	if err := NormalizeActivityType(&valid); err != nil || valid.Name != "Cata" {
		t.Fatalf("valid type rejected or not trimmed: %v %+v", err, valid)
	}
	invalidTypes := []CrmActivityType{
		{Name: "", Icon: "icon-[fa--flag]"},
		{Name: "Cata", Icon: "icon-[fa--rocket]"},
		{ID: ActivityTypeCall, Name: "Llamada", Icon: "icon-[fa--phone]"},
	}
	for _, invalidType := range invalidTypes {
		if err := NormalizeActivityType(&invalidType); err == nil {
			t.Fatalf("type %+v should be rejected", invalidType)
		}
	}
}

func TestSystemActivityTypesUseKnownIconsAndReservedIDs(t *testing.T) {
	for _, option := range SystemActivityTypes {
		if option.ID < 1 || option.ID > ActivityTypeSystemMaxID || !IsActivityIcon(option.Icon) {
			t.Fatalf("system type out of range or with an unknown icon: %+v", option)
		}
	}
}
