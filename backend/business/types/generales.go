package types

import (
	"app/core"
	"app/db"
)

type CityLocation struct {
	db.TableStruct[CityLocationTable, CityLocation]
	ID         int32         `json:",omitempty"`
	CountryID  int32         `json:",omitempty"`
	Name       string        ``
	ParentID   int32         ``
	Hierarchy  int8          `json:",omitempty"`
	Updated    int32         `json:"upd,omitempty"`
	Department *CityLocation `json:"-"`
	Province   *CityLocation `json:"-"`
	District   *CityLocation `json:"-"`
}

type CityLocationTable struct {
	db.TableStruct[CityLocationTable, CityLocation]
	ID        db.Col[CityLocationTable, int32]
	CountryID db.Col[CityLocationTable, int32]
	Name      db.Col[CityLocationTable, string]
	ParentID  db.Col[CityLocationTable, int32]
	Hierarchy db.Col[CityLocationTable, int8]
	Updated   db.Col[CityLocationTable, int32]
}

func (e CityLocationTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "city_locations",
		Partition: e.CountryID,
		Keys:      []db.Coln{e.ID},
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type SharedListRecord struct {
	db.TableStruct[SharedListRecordTable, SharedListRecord]
	CompanyID   int32    `json:",omitempty"`
	ID          int32
	ListID      int32    `json:",omitempty"`
	Name        string   `json:",omitempty"`
	Images      []string `json:",omitempty"`
	Description string   `json:",omitempty"`
	NameHash    int32    `json:",omitempty"`
	// General properties
	Status    int8  `json:"ss,omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
}

type SharedListRecordTable struct {
	db.TableStruct[SharedListRecordTable, SharedListRecord]
	CompanyID   db.Col[SharedListRecordTable, int32]
	ID          db.Col[SharedListRecordTable, int32]
	ListID      db.Col[SharedListRecordTable, int32]
	Name        db.Col[SharedListRecordTable, string]
	Images      db.ColSlice[SharedListRecordTable, string]
	Description db.Col[SharedListRecordTable, string]
	NameHash    db.Col[SharedListRecordTable, int32]
	Status      db.Col[SharedListRecordTable, int8]
	Updated     db.Col[SharedListRecordTable, int32]
	UpdatedBy   db.Col[SharedListRecordTable, int32]
}

func (e SharedListRecordTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "shared_list_records",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.NameHash}},
			{Type: db.TypeView, Keys: []db.Coln{e.ListID.Int32(), e.Status.DecimalSize(2)}},
			{Type: db.TypeView, Keys: []db.Coln{e.ListID, e.Updated.DecimalSize(10)}},
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
