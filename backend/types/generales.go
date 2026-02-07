package types

import "app/db"

type Increment struct {
	db.TableStruct[IncrementTable, Increment]
	Name         string
	CurrentValue int64
}

type IncrementTable struct {
	db.TableStruct[IncrementTable, Increment]
	Name         db.Col[IncrementTable, string]
	CurrentValue db.Col[IncrementTable, int64]
}

func (e IncrementTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:           "sequences",
		Keys:           []db.Coln{e.Name},
		SequenceColumn: &e.CurrentValue,
	}
}

type PaisCiudad struct {
	db.TableStruct[PaisCiudadTable, PaisCiudad]
	PaisID       int32       `json:",omitempty" db:"pais_id,pk"`
	CiudadID     string      `json:"ID" db:"ciudad_id,pk"`
	Nombre       string      `db:"nombre"`
	PadreID      string      `db:"padre_id"`
	Jerarquia    int8        `json:",omitempty" db:"jerarquia"`
	Updated      int64       `json:"upd,omitempty" db:"updated,view"`
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
	Updated   db.Col[PaisCiudadTable, int64]
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
	EmpresaID   int32    `db:"empresa_id,pk"`
	ID          int32    `db:"id,pk"`
	ListaID     int32    `db:"lista_id,view,view.1,view.2"`
	Nombre      string   `json:",omitempty" db:"nombre"`
	Images      []string `json:",omitempty" db:"images"`
	Descripcion string   `json:",omitempty" db:"descripcion"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty" db:"status,view.1"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view.2"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
}

type ListaCompartidaRegistroTable struct {
	db.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
	EmpresaID   db.Col[ListaCompartidaRegistroTable, int32]
	ID          db.Col[ListaCompartidaRegistroTable, int32]
	ListaID     db.Col[ListaCompartidaRegistroTable, int32]
	Nombre      db.Col[ListaCompartidaRegistroTable, string]
	Images      db.ColSlice[ListaCompartidaRegistroTable, string]
	Descripcion db.Col[ListaCompartidaRegistroTable, string]
	Status      db.Col[ListaCompartidaRegistroTable, int8]
	Updated     db.Col[ListaCompartidaRegistroTable, int64]
	UpdatedBy   db.Col[ListaCompartidaRegistroTable, int32]
}

func (e ListaCompartidaRegistroTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "lista_compartida_registro",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Views: []db.View{
			//{Cols: []db.Coln{e.ListaID_(), e.Status_()}, KeepPart: true},
			{Cols: []db.Coln{e.ListaID, e.Status}, ConcatI32: []int8{2}},
			{Cols: []db.Coln{e.ListaID, e.Updated}, ConcatI64: []int8{10}},
		},
	}
}

type NewIDToID struct {
	NewID  int32
	TempID int32
}

type Parametros struct {
	db.TableStruct[ParametrosTable, Parametros]
	EmpresaID int32
	Grupo     int32
	Key       string
	Valor     string
	ValorInt  int32
	Valores   []int32
	// Propiedades generales
	Status    int8
	Updated   int64
	UpdatedBy int32
}

type ParametrosTable struct {
	db.TableStruct[ParametrosTable, Parametros]
	EmpresaID db.Col[ParametrosTable, int32]
	Grupo     db.Col[ParametrosTable, int32]
	Key       db.Col[ParametrosTable, string]
	Valor     db.Col[ParametrosTable, string]
	ValorInt  db.Col[ParametrosTable, int32]
	Valores   db.ColSlice[ParametrosTable, int32]
	Status    db.Col[ParametrosTable, int8]
	Updated   db.Col[ParametrosTable, int64]
	UpdatedBy db.Col[ParametrosTable, int32]
}

func (e ParametrosTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "parametros",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.Grupo, e.Key},
		Views:        []db.View{},
	}
}
