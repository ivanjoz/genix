package types

import "app/db2"

type Increment struct {
	db2.TableStruct[IncrementTable, Increment]
	Name         string
	CurrentValue int64
}

type IncrementTable struct {
	db2.TableStruct[IncrementTable, Increment]
	Name         db2.Col[IncrementTable, string] // `db:"name,pk"`
	CurrentValue db2.Col[IncrementTable, int64]  // `db:"current_value,counter"`
}

func (e IncrementTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:           "sequences",
		Keys:           []db2.Coln{e.Name},
		SequenceColumn: &e.CurrentValue,
	}
}

type PaisCiudad struct {
	db2.TableStruct[PaisCiudadTable, PaisCiudad]
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
	db2.TableStruct[PaisCiudadTable, PaisCiudad]
	PaisID    db2.Col[PaisCiudadTable, int32]
	CiudadID  db2.Col[PaisCiudadTable, string]
	Nombre    db2.Col[PaisCiudadTable, string]
	PadreID   db2.Col[PaisCiudadTable, string]
	Jerarquia db2.Col[PaisCiudadTable, int8]
	Updated   db2.Col[PaisCiudadTable, int64]
}

func (e PaisCiudadTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:      "pais_ciudades",
		Partition: e.PaisID,
		Keys:      []db2.Coln{e.CiudadID},
		Views: []db2.View{
			{Cols: []db2.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type ListaCompartidaRegistro struct {
	db2.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
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
	db2.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
	EmpresaID   db2.Col[ListaCompartidaRegistroTable, int32]
	ID          db2.Col[ListaCompartidaRegistroTable, int32]
	ListaID     db2.Col[ListaCompartidaRegistroTable, int32]
	Nombre      db2.Col[ListaCompartidaRegistroTable, string]
	Images      db2.ColSlice[ListaCompartidaRegistroTable, string]
	Descripcion db2.Col[ListaCompartidaRegistroTable, string]
	Status      db2.Col[ListaCompartidaRegistroTable, int8]
	Updated     db2.Col[ListaCompartidaRegistroTable, int64]
	UpdatedBy   db2.Col[ListaCompartidaRegistroTable, int32]
}

func (e ListaCompartidaRegistroTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:      "lista_compartida_registro",
		Partition: e.EmpresaID,
		Keys:      []db2.Coln{e.ID},
		Views: []db2.View{
			//{Cols: []db.Coln{e.ListaID_(), e.Status_()}, KeepPart: true},
			{Cols: []db2.Coln{e.ListaID, e.Status}, ConcatI32: []int8{2}},
			{Cols: []db2.Coln{e.ListaID, e.Updated}, ConcatI64: []int8{10}},
		},
	}
}

/*
func (e *ListaCompartidaRegistro) GetView(view int8) any {
	if view == 1 {
		return e.ListaID*100 + int32(e.Status)
	} else if view == 2 {
		return int64(e.ListaID)*10_000_000_000 + e.Updated
	} else {
		return 0
	}
}
*/

type NewIDToID struct {
	NewID  int32
	TempID int32
}
