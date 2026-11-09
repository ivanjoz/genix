package types

import "app/db"

// Custom field types, see docs/CRM_PLAN.md §5 for the value column each one uses.
const (
	CustomFieldText        int8 = 1
	CustomFieldLongText    int8 = 2
	CustomFieldInteger     int8 = 3
	CustomFieldDecimal     int8 = 4
	CustomFieldMoney       int8 = 5
	CustomFieldDate        int8 = 6
	CustomFieldSelect      int8 = 7
	CustomFieldMultiSelect int8 = 8
	CustomFieldCheckbox    int8 = 9
	CustomFieldUser        int8 = 10
)

// CrmCustomField is a company-wide catalog entry. Each pipeline picks the fields it shows, and a
// value is keyed by the company-unique ID, so it survives a move between pipelines.
type CrmCustomField struct {
	db.TableStruct[CrmCustomFieldTable, CrmCustomField]
	CompanyID int32                  `cb:"1"`
	ID        int16                  `cb:"2"`
	Name      string                 `cb:"3"`
	Type      int8                   `cb:"4"`
	Options   []CrmCustomFieldOption `cb:"5"` // select / multi-select only
	Status    int8                   `json:"ss" cb:"ss,6"`
	Updated   int32                  `json:"upd" cb:"upd,7"`
	Version   int16                  `json:"ver" cb:"ver,8"`
	UpdatedBy int32                  `cb:"9"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,10"`
}

type CrmCustomFieldOption struct {
	ID     int16  `cb:"1"` // stable, server-allocated
	Name   string `cb:"2"`
	Color  string `cb:"3"`
	Status int8   `cb:"4"`
}

// CrmCustomFieldValue is one value per field on an opportunity. Only the column matching the
// field's type is set.
type CrmCustomFieldValue struct {
	FieldID   int16   `cb:"1"`
	Number    int64   `cb:"2"` // integer, decimal (x100), money, date (UnixDay), checkbox (0/1), user ID, option ID
	Text      string  `cb:"3"` // text, long text
	OptionIDs []int16 `cb:"4"` // multi-select
}

type CrmCustomFieldTable struct {
	db.TableStruct[CrmCustomFieldTable, CrmCustomField]
	CompanyID db.Col[*CrmCustomFieldTable, int32]
	ID        db.Col[*CrmCustomFieldTable, int16]
	Name      db.Col[*CrmCustomFieldTable, string]
	Type      db.Col[*CrmCustomFieldTable, int8]
	Options   db.Col[*CrmCustomFieldTable, []CrmCustomFieldOption]
	Status    db.Col[*CrmCustomFieldTable, int8]
	Updated   db.Col[*CrmCustomFieldTable, int32]
	Version   db.Col[*CrmCustomFieldTable, int16]
	UpdatedBy db.Col[*CrmCustomFieldTable, int32]
}

func (e CrmCustomFieldTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          68,
		Name:        "crm_custom_fields",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}
