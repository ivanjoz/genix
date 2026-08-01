package types

import (
	"app/core"
	"app/db"
)

type CityLocation struct {
	db.TableStruct[CityLocationTable, CityLocation]
	ID             int32         `json:",omitempty"`
	CountryID      int32         `json:",omitempty"`
	Name           string        ``
	ParentID       int32         ``
	Hierarchy      int8          `json:",omitempty"`
	Updated        int32         `json:"upd,omitempty"`
	UpdatedVersion int32         `json:"upv,omitempty"`
	Department     *CityLocation `json:"-"`
	Province       *CityLocation `json:"-"`
	District       *CityLocation `json:"-"`
}

type CityLocationTable struct {
	db.TableStruct[CityLocationTable, CityLocation]
	ID             db.Col[CityLocationTable, int32]
	CountryID      db.Col[CityLocationTable, int32]
	Name           db.Col[CityLocationTable, string]
	ParentID       db.Col[CityLocationTable, int32]
	Hierarchy      db.Col[CityLocationTable, int8]
	Updated        db.Col[CityLocationTable, int32]
	UpdatedVersion db.Col[CityLocationTable, int32]
}

func (e CityLocationTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        8,
		Name:      "city_locations",
		Partition: e.CountryID,
		Keys:      db.Cols(e.ID),
		Indexes: []db.Index{
			// Keyless: this sync filters nothing but its watermark. Read with Delta(upv).
			{Type: db.TypeDelta},
		},
	}
}

type SharedListRecord struct {
	db.TableStruct[SharedListRecordTable, SharedListRecord]
	CompanyID   int32 `json:",omitempty"`
	ID          int32
	ListID      int32    `json:",omitempty"`
	Name        string   `json:",omitempty"`
	Images      []string `json:",omitempty"`
	Description string   `json:",omitempty"`
	NameHash    int32    `json:",omitempty"`
	// General properties
	Status         int8  `json:"ss,omitempty"`
	Updated        int32 `json:"upd,omitempty"`
	UpdatedVersion int32 `json:"upv,omitempty"`
	UpdatedBy      int32 `json:",omitempty"`
}

type SharedListRecordTable struct {
	db.TableStruct[SharedListRecordTable, SharedListRecord]
	CompanyID      db.Col[SharedListRecordTable, int32]
	ID             db.Col[SharedListRecordTable, int32]
	ListID         db.Col[SharedListRecordTable, int32]
	Name           db.Col[SharedListRecordTable, string]
	Images         db.ColSlice[SharedListRecordTable, string]
	Description    db.Col[SharedListRecordTable, string]
	NameHash       db.Col[SharedListRecordTable, int32]
	Status         db.Col[SharedListRecordTable, int8]
	Updated        db.Col[SharedListRecordTable, int32]
	UpdatedVersion db.Col[SharedListRecordTable, int32]
	UpdatedBy      db.Col[SharedListRecordTable, int32]
}

func (e SharedListRecordTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:           29,
		Name:         "shared_list_records",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		// Only Status needs declaring: it is the column Delta() fans a delta sync out over. ListID
		// arrives from the client with no natural ceiling, so it is left undeclared and the delta view
		// gives it the digit remainder.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.NameHash)},
			// Both still read by the ecommerce delta (product-ecommerce.go), which keeps its timestamp
			// watermark because it also drives the prerendered .db snapshot.
			{Type: db.TypeView, Keys: db.Cols(e.ListID.Int32(), e.Status.DecimalSize(2))},
			{Type: db.TypeView, Keys: db.Cols(e.ListID, e.Updated.DecimalSize(10))},
			{Type: db.TypeDelta, Keys: db.Cols(e.Status, e.ListID)},
		},
	}
}

func (e *SharedListRecord) SelfParse() {
	name := core.Concatn(e.ListID, e.Name)
	e.NameHash = core.BasicHashInt(core.NormalizeString(&name))
}

type NewIDToID struct {
	ID     int32
	TempID int32
}
