package types

import (
	"app/core"
	"app/db"
)

type PaisCiudad struct {
	db.TableStruct[PaisCiudadTable, PaisCiudad]
	PaisID       int32       `json:",omitempty" db:"pais_id,pk"`
	CiudadID     string      `json:"ID" db:"ciudad_id,pk"`
	Nombre       string      `db:"nombre"`
	PadreID      string      `db:"padre_id"`
	Jerarquia    int8        `json:",omitempty" db:"jerarquia"`
	Updated      int32       `json:"upd,omitempty" db:"updated,view"`
	Departamento *PaisCiudad `json:"-"`
	Provincia    *PaisCiudad `json:"-"`
}

type PaisCiudadTable struct {
	db.TableStruct[PaisCiudadTable, PaisCiudad]
	PaisID    db.Col[PaisCiudadTable, int32]
	CiudadID  db.Col[PaisCiudadTable, string]
	Nombre    db.Col[PaisCiudadTable, string]
	PadreID   db.Col[PaisCiudadTable, string]
	Jerarquia db.Col[PaisCiudadTable, int8]
	Updated   db.Col[PaisCiudadTable, int32]
}

func (e PaisCiudadTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "pais_ciudades",
		Partition: e.PaisID,
		Keys:      []db.Coln{e.CiudadID},
		Views: []db.View{
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type ListaCompartidaRegistro struct {
	db.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
	EmpresaID   int32 `json:",omitempty"`
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

type ListaCompartidaRegistroTable struct {
	db.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
	EmpresaID   db.Col[ListaCompartidaRegistroTable, int32]
	ID          db.Col[ListaCompartidaRegistroTable, int32]
	ListaID     db.Col[ListaCompartidaRegistroTable, int32]
	Nombre      db.Col[ListaCompartidaRegistroTable, string]
	Images      db.ColSlice[ListaCompartidaRegistroTable, string]
	Descripcion db.Col[ListaCompartidaRegistroTable, string]
	NombreHash  db.Col[ListaCompartidaRegistroTable, int32]
	Status      db.Col[ListaCompartidaRegistroTable, int8]
	Updated     db.Col[ListaCompartidaRegistroTable, int32]
	UpdatedBy   db.Col[ListaCompartidaRegistroTable, int32]
}

func (e ListaCompartidaRegistroTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "lista_compartida_registro",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes:      [][]db.Coln{{e.NombreHash}},
		Views: []db.View{
			{Cols: []db.Coln{e.ListaID.Int32(), e.Status.DecimalSize(2)}},
			{Cols: []db.Coln{e.ListaID, e.Updated.DecimalSize(10)}},
		},
	}
}

func (e *ListaCompartidaRegistro) SelfParse() {
	name := core.Concatn(e.ListaID, e.Nombre)
	e.NombreHash = core.BasicHashInt(core.NormalizeString(&name))
}

type NewIDToID struct {
	ID     int32
	TempID int32
}
