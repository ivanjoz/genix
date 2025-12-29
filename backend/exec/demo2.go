package exec

import (
	"app/aws"
	"app/core"
	"app/db2"
	"app/handlers"
	"fmt"
)

type InnerStruct struct {
	Hola  string  `json:"a,omitempty" ms:"h"`
	Value float32 `json:"b,omitempty" ms:"v"`
	Hola2 int     `json:"c,omitempty" ms:"c"`
}

type DemoStruct struct {
	db2.TableStruct[DemoStructTable, DemoStruct]
	EmpresaID   int32    `db:"empresa_id,pk"`
	ID          int32    `db:"id,pk"`
	ListaID     int32    `db:"lista_id,view,view.1,view.2"`
	Nombre      string   `json:",omitempty" db:"nombre"`
	Images      []string `json:",omitempty" db:"images"`
	RolesIDs    []int32  `json:",omitempty" db:"roles_ids"`
	Descripcion string   `json:",omitempty" db:"descripcion"`
	DemoColumn  InnerStruct
	// Propiedades generales
	Status    int8  `json:"ss,omitempty" db:"status,view.1"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view.2"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
}

type DemoStructTable struct {
	db2.TableStruct[DemoStructTable, DemoStruct]
	EmpresaID   db2.Col[DemoStructTable, int32]
	ID          db2.Col[DemoStructTable, int32]
	ListaID     db2.Col[DemoStructTable, int32]
	Nombre      db2.Col[DemoStructTable, string]
	Images      db2.ColSlice[DemoStructTable, string]
	RolesIDs    db2.ColSlice[DemoStructTable, int32]
	Descripcion db2.Col[DemoStructTable, string]
	Status      db2.Col[DemoStructTable, int8]
	Updated     db2.Col[DemoStructTable, int64]
	UpdatedBy   db2.Col[DemoStructTable, int32]
	DemoColumn  db2.Col[DemoStructTable, InnerStruct]
}

func (e DemoStructTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:      "zz_demo_struct",
		Partition: e.EmpresaID,
		Keys:      []db2.Coln{e.ID},
		Views: []db2.View{
			//{Cols: []db.Coln{e.ListaID_(), e.Status_()}, KeepPart: true},
			{Cols: []db2.Coln{e.ListaID, e.Status}, ConcatI32: []int8{2}},
			{Cols: []db2.Coln{e.ListaID, e.Updated}, ConcatI64: []int8{10}},
		},
	}
}

func Test38(args *core.ExecArgs) core.FuncResponse {

	var err error
	/*
		record := DemoStruct{
			EmpresaID:   4,
			ID:          1,
			ListaID:     11,
			Nombre:      "Jhon",
			Images:      []string{"image1", "image2"},
			RolesIDs:    []int32{4, 6, 9, 11, 14},
			Descripcion: "ddd",
			Status:      2,
			Updated:     123456789,
			UpdatedBy:   99,
			DemoColumn:  InnerStruct{Hola: "mundo", Value: 0.11, Hola2: 5555},
		}

		// Insertarndo registros
		fmt.Println("Insertando Registros...")
		err := db2.Insert(&[]DemoStruct{record})
		if err != nil {
			panic(err)
		}
	*/

	// Obteniendo registros
	fmt.Println("Obteniendo Registros...")
	recordsGetted := []DemoStruct{}
	query := db2.Query(&recordsGetted)
	err = query.Exclude(query.ListaID).ID.Equals(1).Exec()
	if err != nil {
		panic(err)
	}

	core.Print(recordsGetted[0])

	return core.FuncResponse{}
}

func Test39(args *core.ExecArgs) core.FuncResponse {

	value := int64(65)

	fmt.Println(string(core.Int64ToBase64Bytes(value)))

	return core.FuncResponse{}
}

func Test40(args *core.ExecArgs) core.FuncResponse {

	query := aws.DynamoQueryParam{Index: "ix3", GreaterThan: "0"}
	query.GreaterThan = fmt.Sprintf("1_%v", 0)

	dynamoTable := handlers.MakeUsuarioTable(1)
	records, err := dynamoTable.QueryBatch([]aws.DynamoQueryParam{query})

	if err != nil {
		panic(err)
	}

	if err = db2.Insert(&records); err != nil {
		panic(err.Error())
	}

	return core.FuncResponse{}
}

func Test41(args *core.ExecArgs) core.FuncResponse {

	RecalcSequences(1)

	return core.FuncResponse{}
}
