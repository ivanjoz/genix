package types

import (
	"app/core"
	"app/db"
)

type CityLocation struct {
	db.TableStruct[CityLocationTable, CityLocation]
	ID           int32       `json:",omitempty"`
	PaisID       int32       `json:",omitempty"`
	CiudadID     string      `json:",omitempty"`
	Nombre       string      ``
	PadreID      string      ``
	Jerarquia    int8        `json:",omitempty"`
	Updated      int32       `json:"upd,omitempty"`
	Departamento *CityLocation `json:"-"`
	Provincia    *CityLocation `json:"-"`
}

type CityLocationTable struct {
	db.TableStruct[CityLocationTable, CityLocation]
	PaisID    db.Col[CityLocationTable, int32]
	CiudadID  db.Col[CityLocationTable, string]
	Nombre    db.Col[CityLocationTable, string]
	PadreID   db.Col[CityLocationTable, string]
	Jerarquia db.Col[CityLocationTable, int8]
	Updated   db.Col[CityLocationTable, int32]
}

func (e CityLocationTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "pais_ciudades",
		Partition: e.PaisID,
		Keys:      []db.Coln{e.CiudadID},
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type SharedListRecord struct {
	db.TableStruct[SharedListRecordTable, SharedListRecord]
	CompanyID   int32 `json:",omitempty"`
	ID          int32
	ListaID     int32    `json:",omitempty"`
	Nombre      string   `json:",omitempty"`
	Images      []string `json:",omitempty"`
	Descripcion string   `json:",omitempty"`
	NombreHash  int32    `json:",omitempty"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
}

type SharedListRecordTable struct {
	db.TableStruct[SharedListRecordTable, SharedListRecord]
	CompanyID   db.Col[SharedListRecordTable, int32]
	ID          db.Col[SharedListRecordTable, int32]
	ListaID     db.Col[SharedListRecordTable, int32]
	Nombre      db.Col[SharedListRecordTable, string]
	Images      db.ColSlice[SharedListRecordTable, string]
	Descripcion db.Col[SharedListRecordTable, string]
	NombreHash  db.Col[SharedListRecordTable, int32]
	Status      db.Col[SharedListRecordTable, int8]
	Updated     db.Col[SharedListRecordTable, int32]
	UpdatedBy   db.Col[SharedListRecordTable, int32]
}

func (e SharedListRecordTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "lista_compartida_registro",
		Partition:    e.CompanyID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: []db.Coln{e.NombreHash}},
			{Type: db.TypeView, Keys: []db.Coln{e.ListaID.Int32(), e.Status.DecimalSize(2)}},
			{Type: db.TypeView, Keys: []db.Coln{e.ListaID, e.Updated.DecimalSize(10)}},
		},
	}
}

func (e *SharedListRecord) SelfParse() {
	name := core.Concatn(e.ListaID, e.Nombre)
	e.NombreHash = core.BasicHashInt(core.NormalizeString(&name))
}

type NewIDToID struct {
	ID     int32
	TempID int32
}
