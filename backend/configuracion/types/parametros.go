package types

import "app/db"

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
	Updated   int32
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
	Updated   db.Col[ParametrosTable, int32]
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
