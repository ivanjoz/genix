package exec

import (

	// comercial "app/comercial/types"
	"app/comercial"
	comercialTypes "app/comercial/types"
	"app/core"
	"app/db"
	"app/libs"
	logisticaTypes "app/logistica/types"
	"app/negocio"
	negocioTypes "app/negocio/types"
	"fmt"
)

type InnerStruct struct {
	Hola  string  `json:"a,omitempty" ms:"h"`
	Value float32 `json:"b,omitempty" ms:"v"`
	Hola2 int     `json:"c,omitempty" ms:"c"`
}

type DemoStruct struct {
	db.TableStruct[DemoStructTable, DemoStruct]
	EmpresaID   int32    `db:"empresa_id,pk"`
	ID          int32    `db:"id,pk"`
	ListaID     int32    `db:"lista_id,view,view.1,view.2"`
	Nombre      string   `json:",omitempty" db:"nombre"`
	Images      []string `json:",omitempty" db:"images"`
	Descripcion string   `json:",omitempty" db:"descripcion"`
	DemoColumn  InnerStruct
	// Propiedades generales
	Status    int8  `json:"ss,omitempty" db:"status,view.1"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view.2"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
}

type DemoStructTable struct {
	db.TableStruct[DemoStructTable, DemoStruct]
	EmpresaID   db.Col[DemoStructTable, int32]
	ID          db.Col[DemoStructTable, int32]
	ListaID     db.Col[DemoStructTable, int32]
	Nombre      db.Col[DemoStructTable, string]
	Images      db.ColSlice[DemoStructTable, string]
	Descripcion db.Col[DemoStructTable, string]
	DemoColumn  db.Col[DemoStructTable, InnerStruct]
	Status      db.Col[DemoStructTable, int8]
	Updated     db.Col[DemoStructTable, int64]
	UpdatedBy   db.Col[DemoStructTable, int32]
}

func (e DemoStructTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "zz_demo_struct",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.ID},
		Indexes: []db.Index{
			//{Cols: []db.Coln{e.ListaID_(), e.Status_()}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.ListaID.Int32(), e.Status.DecimalSize(2)}},
			{Type: db.TypeView, Keys: []db.Coln{e.ListaID, e.Updated.DecimalSize(10)}},
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
			Descripcion: "ddd",
			Status:      2,
			Updated:     123456789,
			UpdatedBy:   99,
			DemoColumn:  InnerStruct{Hola: "mundo", Value: 0.11, Hola2: 5555},
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

func Test41(args *core.ExecArgs) core.FuncResponse {
	/*
		records := []negocioTypes.ListaCompartidaRegistro{}

		query := db.Query(&records)
		err := query.EmpresaID.Equals(1).Exec()
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
		records := []negocioTypes.PaisCiudad{}

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

	records := []negocioTypes.Producto{}

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

func Test43(args *core.ExecArgs) core.FuncResponse {
	indexOutput, indexErr := negocio.BuildProductosSearchIndex(1)
	if indexErr != nil {
		return core.FuncResponse{Error: indexErr.Error()}
	}
	indexBuild := indexOutput.IndexBuild

	brandNamesBytes := 0
	for _, brandName := range indexBuild.BrandNames {
		// String column wire size uses 1-byte length prefix + UTF-8 bytes.
		brandNamesBytes += 1 + len([]byte(brandName))
	}
	categoryNamesBytes := 0
	for _, categoryName := range indexBuild.CategoryNames {
		// String column wire size uses 1-byte length prefix + UTF-8 bytes.
		categoryNamesBytes += 1 + len([]byte(categoryName))
	}
	brandIDsBytes := len(indexBuild.BrandIDs) * 2
	categoryIDsBytes := len(indexBuild.CategoryIDs) * 2
	brandIndexesBytes := indexBuild.ProductBrandIndexesBytes()
	categoryCountBytes := len(indexBuild.ProductCategoryCount)
	categoryIndexesBytes := len(indexBuild.ProductCategoryIndexes)
	stage2TotalBytes := brandNamesBytes + categoryNamesBytes + brandIDsBytes + categoryIDsBytes +
		brandIndexesBytes + categoryCountBytes + categoryIndexesBytes

	return core.FuncResponse{
		Message: "Índice de productos generado.",
		Content: map[string]any{
			"stage1_input_records":          indexBuild.Stats.InputRecords,
			"stage1_encoded_records":        indexBuild.Stats.EncodedRecords,
			"stage1_dictionary_count":       indexBuild.Stats.DictionaryCount,
			"stage1_shapes_bytes":           indexBuild.Stats.ShapesBytes,
			"stage1_content_bytes":          indexBuild.Stats.ContentBytes,
			"stage1_total_bytes":            indexBuild.Stats.TotalBytes,
			"stage2_brand_ids":              len(indexBuild.BrandIDs),
			"stage2_category_ids":           len(indexBuild.CategoryIDs),
			"stage2_brand_index_mode":       indexBuild.BrandIndexEncodingName(),
			"stage2_brand_index_flag":       indexBuild.BrandIndexEncodingFlag,
			"stage2_brand_indexes_count":    indexBuild.ProductBrandIndexesCount(),
			"stage2_category_count":         len(indexBuild.ProductCategoryCount),
			"stage2_category_indexes":       len(indexBuild.ProductCategoryIndexes),
			"stage2_brand_ids_bytes":        brandIDsBytes,
			"stage2_brand_names_bytes":      brandNamesBytes,
			"stage2_category_ids_bytes":     categoryIDsBytes,
			"stage2_category_names_bytes":   categoryNamesBytes,
			"stage2_brand_indexes_bytes":    brandIndexesBytes,
			"stage2_category_count_bytes":   categoryCountBytes,
			"stage2_category_indexes_bytes": categoryIndexesBytes,
			"stage2_total_bytes":            stage2TotalBytes,
		},
	}
}

func Test44(args *core.ExecArgs) core.FuncResponse {

	s1 := comercialTypes.SaleOrderProductStats{
		Quantity:                123,
		QuantityPendingDelivery: 12332,
		TotalAmount:             338829,
		TotalDebtAmount:         78954,
	}

	core.Print(s1)

	bytes1 := libs.SerializeInt30Struct(s1)
	core.Log("struct bytes:", bytes1)

	decodedStruct := comercialTypes.SaleOrderProductStats{}
	libs.DeserializeInt30Struct(bytes1, &decodedStruct)

	core.Print(decodedStruct)

	return core.FuncResponse{}
}

func Test45(args *core.ExecArgs) core.FuncResponse {

	comercial.SaleOrderReprocess(1, 0)

	return core.FuncResponse{}
}

func Test46(args *core.ExecArgs) core.FuncResponse {

	controller := makeDBController[comercialTypes.SaleOrder]()

	err := controller.RecalcGroupIndexHashes(1)
	if err != nil {
		fmt.Println(err)
	}
	
	return core.FuncResponse{}
}

func Test51(args *core.ExecArgs) core.FuncResponse {

	orders := []comercialTypes.SaleOrder{}
	db.Query(&orders).CompanyID.Equals(1)

	controller := makeDBController[logisticaTypes.WarehouseProductMovement]()

	controller.RecalcVirtualColumns(1)

	return core.FuncResponse{}
}
