package exec

import (

	// sales "app/sales/types"
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	"app/libs"
	"app/sales"
	salesTypes "app/sales/types"
	webpageTypes "app/webpage/types"
	"fmt"
)

type InnerStruct struct {
	Greeting  string  `json:"a,omitempty" ms:"h"`
	Value     float32 `json:"b,omitempty" ms:"v"`
	Greeting2 int     `json:"c,omitempty" ms:"c"`
}

type DemoStruct struct {
	db.TableStruct[DemoStructTable, DemoStruct]
	CompanyID   int32    `db:"empresa_id,pk"`
	ID          int32    `db:"id,pk"`
	ListID      int32    `db:"lista_id,view,view.1,view.2"`
	Name        string   `json:",omitempty" db:"nombre"`
	Images      []string `json:",omitempty" db:"images"`
	Description string   `json:",omitempty" db:"descripcion"`
	DemoColumn  InnerStruct
	// General properties
	Status    int8  `json:"ss,omitempty" db:"status,view.1"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view.2"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
}

type DemoStructTable struct {
	db.TableStruct[DemoStructTable, DemoStruct]
	CompanyID   db.Col[DemoStructTable, int32]
	ID          db.Col[DemoStructTable, int32]
	ListID      db.Col[DemoStructTable, int32]
	Name        db.Col[DemoStructTable, string]
	Images      db.ColSlice[DemoStructTable, string]
	Description db.Col[DemoStructTable, string]
	DemoColumn  db.Col[DemoStructTable, InnerStruct]
	Status      db.Col[DemoStructTable, int8]
	Updated     db.Col[DemoStructTable, int64]
	UpdatedBy   db.Col[DemoStructTable, int32]
}

func (e DemoStructTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "zz_demo_struct",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID},
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: []db.Coln{e.ListID.Int32(), e.Status.DecimalSize(2)}},
			{Type: db.TypeView, Keys: []db.Coln{e.ListID, e.Updated.DecimalSize(10)}},
		},
	}
}

func Test38(args *core.ExecArgs) core.FuncResponse {

	var err error
	/*
		record := DemoStruct{
			CompanyID:   4,
			ID:          1,
			ListID:      11,
			Name:        "Jhon",
			Images:      []string{"image1", "image2"},
			Description: "ddd",
			Status:      2,
			Updated:     123456789,
			UpdatedBy:   99,
			DemoColumn:  InnerStruct{Greeting: "mundo", Value: 0.11, Greeting2: 5555},
		}

		// Insertarndo registros
		fmt.Println("Insertando Registros...")
		err := db.Insert(&[]DemoStruct{record})
		if err != nil {
			panic(err)
		}
	*/

	// Obteniendo registros
	fmt.Println("Obteniendo Registros...")
	recordsGetted := []DemoStruct{}
	query := db.Query(&recordsGetted)
	err = query.Exclude(query.ListID).ID.Equals(1).Exec()
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

func Test41(args *core.ExecArgs) core.FuncResponse {
	/*
		records := []businessTypes.SharedListRecord{}

		query := db.Query(&records)
		err := query.CompanyID.Equals(1).Exec()
		if err != nil {
			panic(err)
		}

		core.Log("registros obtenidos::", len(records))

		for i := range records {
			records[i].Updated = core.SUnixTime()
		}

		err = db.Update(&records, query.ListaID, query.Updated, query.Status)
		if err != nil {
			panic(err)
		}
	*/
	/*
		records := []businessTypes.CityLocation{}

		query := db.Query(&records)
		err := query.Exec()
		if err != nil {
			panic(err)
		}

		core.Log("registros obtenidos::", len(records))

		for i := range records {
			records[i].Updated = core.SUnixTime()
		}

		err = db.Update(&records, query.Updated)
		if err != nil {
			panic(err)
		}
	*/

	records := []businessTypes.Product{}

	query := db.Query(&records)
	err := query.Select(query.ID).Exec()
	if err != nil {
		panic(err)
	}

	core.Log("registros obtenidos::", len(records))

	for i := range records {
		records[i].Updated = core.SUnixTime()
		records[i].Created = core.SUnixTime()
	}

	err = db.Update(&records, query.Updated, query.Created)
	if err != nil {
		panic(err)
	}

	return core.FuncResponse{}
}

func Test44(args *core.ExecArgs) core.FuncResponse {

	s1 := salesTypes.SaleOrderProductStats{
		Quantity:                123,
		QuantityPendingDelivery: 12332,
		TotalAmount:             338829,
		TotalDebtAmount:         78954,
	}

	core.Print(s1)

	bytes1 := libs.SerializeInt30Struct(s1)
	core.Log("struct bytes:", bytes1)

	decodedStruct := salesTypes.SaleOrderProductStats{}
	libs.DeserializeInt30Struct(bytes1, &decodedStruct)

	core.Print(decodedStruct)

	return core.FuncResponse{}
}

func Test45(args *core.ExecArgs) core.FuncResponse {

	sales.SaleOrderReprocess(1, 0)

	return core.FuncResponse{}
}

func Test46(args *core.ExecArgs) core.FuncResponse {

	controller := makeDBController[salesTypes.SaleOrder]()

	err := controller.RecalcVirtualColumns(1)
	if err != nil {
		fmt.Println(err)
	}
	
	return core.FuncResponse{}
}

func Test51(args *core.ExecArgs) core.FuncResponse {


	controller := makeDBController[webpageTypes.Webpage]()

 //	controller.RecalcVirtualColumns(1)
	controller.DeleteViewsAndIndexes()
	// controller.RecalcVirtualColumns(1)
	
//	controller2 := makeDBController[logisticsTypes.ProductStockV2]()
	// controller2.DeleteViewsAndIndexes()

	return core.FuncResponse{}
}
