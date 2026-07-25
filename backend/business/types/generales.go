package types

import (
	"app/core"
	"github.com/ivanjoz/genix-orm/scylla"
)

type CityLocation struct {
	scylla.TableStruct[CityLocationTable, CityLocation]
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
	scylla.TableStruct[CityLocationTable, CityLocation]
	ID        scylla.Col[CityLocationTable, int32]
	CountryID scylla.Col[CityLocationTable, int32]
	Name      scylla.Col[CityLocationTable, string]
	ParentID  scylla.Col[CityLocationTable, int32]
	Hierarchy scylla.Col[CityLocationTable, int8]
	Updated   scylla.Col[CityLocationTable, int32]
}

func (e CityLocationTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "city_locations",
		Partition: e.CountryID,
		Keys:      scylla.Cols(e.ID),
		Indexes: []scylla.Index{
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}

type SharedListRecord struct {
	scylla.TableStruct[SharedListRecordTable, SharedListRecord]
	CompanyID   int32 `json:",omitempty"`
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
	scylla.TableStruct[SharedListRecordTable, SharedListRecord]
	CompanyID   scylla.Col[SharedListRecordTable, int32]
	ID          scylla.Col[SharedListRecordTable, int32]
	ListID      scylla.Col[SharedListRecordTable, int32]
	Name        scylla.Col[SharedListRecordTable, string]
	Images      scylla.ColSlice[SharedListRecordTable, string]
	Description scylla.Col[SharedListRecordTable, string]
	NameHash    scylla.Col[SharedListRecordTable, int32]
	Status      scylla.Col[SharedListRecordTable, int8]
	Updated     scylla.Col[SharedListRecordTable, int32]
	UpdatedBy   scylla.Col[SharedListRecordTable, int32]
}

func (e SharedListRecordTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:         "shared_list_records",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			{Type: scylla.TypeLocalIndex, Keys: scylla.Cols(e.NameHash)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.ListID.Int32(), e.Status.DecimalSize(2))},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.ListID, e.Updated.DecimalSize(10))},
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
