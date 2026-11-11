package types

import "app/db"

// CrmEmailTemplate is a reusable email a seller sends from an opportunity (or a stage task sends
// for them). Subject and BodyHTML carry {{placeholders}} that RenderEmailTemplate fills.
type CrmEmailTemplate struct {
	db.TableStruct[CrmEmailTemplateTable, CrmEmailTemplate]
	CompanyID int32  `cb:"1"`
	ID        int16  `cb:"2"`
	Name      string `cb:"3"`
	Subject   string `cb:"4"`
	BodyHTML  string `cb:"5"`
	Status    int8   `json:"ss" cb:"ss,6"`
	Updated   int32  `json:"upd" cb:"upd,7"`
	Version   int16  `json:"ver" cb:"ver,8"`
	UpdatedBy int32  `cb:"9"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,10"`
}

type CrmEmailTemplateTable struct {
	db.TableStruct[CrmEmailTemplateTable, CrmEmailTemplate]
	CompanyID db.Col[*CrmEmailTemplateTable, int32]
	ID        db.Col[*CrmEmailTemplateTable, int16]
	Name      db.Col[*CrmEmailTemplateTable, string]
	Subject   db.Col[*CrmEmailTemplateTable, string]
	BodyHTML  db.Col[*CrmEmailTemplateTable, string]
	Status    db.Col[*CrmEmailTemplateTable, int8]
	Updated   db.Col[*CrmEmailTemplateTable, int32]
	Version   db.Col[*CrmEmailTemplateTable, int16]
	UpdatedBy db.Col[*CrmEmailTemplateTable, int32]
}

func (e CrmEmailTemplateTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          90,
		Name:        "crm_email_templates",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}

// CrmOpportunityEmailPayload is an email a seller sends from an opportunity. TemplateID is only a
// record of where it came from: the subject and body are what the seller left after editing.
// With TaskID the email fulfils that "send email" task.
type CrmOpportunityEmailPayload struct {
	OpportunityID int32
	TemplateID    int16
	To            []string
	Cc            []string
	Subject       string
	HTMLBody      string
	TaskID        int32
}

// CrmEmailDraft is a template rendered for one opportunity, for the seller to review and send.
type CrmEmailDraft struct {
	To       []string
	Subject  string
	HTMLBody string
}
