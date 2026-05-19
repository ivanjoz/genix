package exec

import (
	sales "app/sales/types"
	"app/core"
	"app/db"
	financeTypes "app/finance/types"
	logisticsTypes "app/logistics/types"
	businessTypes "app/business/types"
	"fmt"
)

// TestSelects demonstrates complex select queries including the new KeyConcatenated smart logic.
func TestSelects(args *core.ExecArgs) core.FuncResponse {
	var err error
	
	traceSales := []db.RecordGroup[sales.SaleOrder]{}
	query := db.QueryIndexGroup(&traceSales)
	
	query.IncludeCachedGroup(12340001, 1001)
	query.IncludeCachedGroup(12340002, 1002)
	
	query.CompanyID.Equals(1).DetailProductsIDs.Contains(3372).Date.Between(20500, 20558)

	
	if err := query.Exec(); err != nil {
		fmt.Println("Error in Test 6:", err)
	} else {
		fmt.Printf("Found %d traceSales in range\n", len(traceSales))
	}
	
	for _, e := range traceSales {
		fmt.Println(e.GroupHash,"|",e.IndexGroupValues,"| Records:",len(e.Records))
	}
		
	// 6. Test bucket query CONTAINS + "RANGE" with hash index
	fmt.Println("\n--- Test 5: Range Query (Between) ---")
	recordSalesOrders := []sales.SaleOrder{}
	q6 := db.Query(&recordSalesOrders)
	err = q6.CompanyID.Equals(1).
		DetailProductsIDs.Contains(3372).
		Date.Between(20530, 20540). //.AllowFilter().
		Exec()

	if err != nil {
		fmt.Println("Error in Test 6:", err)
	} else {
		fmt.Printf("Found %d records in range\n", len(recordSalesOrders))
	}

	// return core.FuncResponse{}
	
	fmt.Println("\n--- Test 7: Range query in int packet column local index ---")
	recordSalesOrders2 := []sales.SaleOrder{}
	q7 := db.Query(&recordSalesOrders2)
	err = q7.CompanyID.Equals(1).
		Status.Equals(1).
		Updated.Between(385298000, 385299000).AllowFilter().
		Exec()

	if err != nil {
		fmt.Println("Error in Test 6:", err)
	} else {
		fmt.Printf("Found %d records in range\n", len(recordSalesOrders2))
	}

	// 1. Test AlmacenProducto with KeyConcatenated Smart Logic
	// This should trigger a range query on the 'id' column because it's the first column of KeyConcatenated.
	fmt.Println("\n--- Test 1: AlmacenProducto (Smart ORM for KeyConcatenated) ---")
	productos := []logisticsTypes.ProductStock{}
	q1 := db.Query(&productos)
	err = q1.CompanyID.Equals(1).
		WarehouseID.Equals(1). // This is the first column in KeyConcatenated for AlmacenProducto
		Limit(5).Exec()

	if err != nil {
		fmt.Println("Error in Test 1:", err)
	} else {
		fmt.Printf("Found %d products\n", len(productos))
	}

	// 2. Test AlmacenProducto with multiple prefix columns
	fmt.Println("\n--- Test 2: AlmacenProducto (Multiple prefix columns) ---")
	productos2 := []logisticsTypes.ProductStock{}
	q2 := db.Query(&productos2)
	err = q2.CompanyID.Equals(1).
		WarehouseID.Equals(1).
		ProductID.GreaterEqual(8).
		Limit(5).Exec()

	if err != nil {
		fmt.Println("Error in Test 2:", err)
	} else {
		fmt.Printf("Found %d products\n", len(productos2))
	}

	// New test
	fmt.Println("\n--- Test 21: AlmacenProducto. Using view: []db.Coln{e.WarehouseID, e.Status, e.Updated} ---")
	productos21 := []logisticsTypes.ProductStock{}
	q21 := db.Query(&productos21)
	err = q21.CompanyID.Equals(1).
		WarehouseID.Equals(1).Status.Equals(1).Updated.GreaterEqual(1000).
		Limit(5).Exec()

	if err != nil {
		fmt.Println("Error in Test 21:", err)
	} else {
		fmt.Printf("Found %d products\n", len(productos2))
	}

	// 3. Test SharedListRecord with complex view concatenation
	fmt.Println("\n--- Test 3: SharedListRecord (Complex View/Concatenation) ---")
	registros := []businessTypes.SharedListRecord{}
	q3 := db.Query(&registros)
	// This query should use a view that concatenates ListaID and Status or Updated
	err = q3.CompanyID.Equals(1).
		ListID.Equals(1).
		Status.Equals(1).
		Exec()

	if err != nil {
		fmt.Println("Error in Test 3:", err)
	} else {
		fmt.Printf("Found %d records\n", len(registros))
	}

	// 4. Test CashBankMovement with View
	fmt.Println("\n--- Test 4: CashBankMovement (Query using View) ---")
	movimientos := []financeTypes.CashBankMovement{}
	q4 := db.Query(&movimientos)
	err = q4.CompanyID.Equals(1).
		DocumentID.Equals(12345). // This uses a view defined in CashBankMovementTable
		Exec()

	if err != nil {
		fmt.Println("Error in Test 4:", err)
	} else {
		fmt.Printf("Found %d movements\n", len(movimientos))
	}

	// 5. Test with range query (Between)
	fmt.Println("\n--- Test 5: Range Query (Between) ---")
	recordRegistrosListas := []businessTypes.SharedListRecord{}
	q5 := db.Query(&recordRegistrosListas)
	err = q5.CompanyID.Equals(1).
		ListID.Equals(1).
		Updated.Between(1000000000, 2000000000).
		Exec()

	if err != nil {
		fmt.Println("Error in Test 5:", err)
	} else {
		fmt.Printf("Found %d records in range\n", len(recordRegistrosListas))
	}

	return core.FuncResponse{}
}


// TestSelects demonstrates complex select queries including the new KeyConcatenated smart logic.
func TestSelects2(args *core.ExecArgs) core.FuncResponse {
	var err error

	movimientos := []logisticsTypes.WarehouseProductMovement{}

	query := db.Query(&movimientos).
		CompanyID.Equals(1).
		Date.GreaterEqual(1827)

	if err := query.GroupBy(query.Date, query.ProductID, query.Type, query.Quantity.Sum()).Exec(); err != nil {
		panic(err)
	}
	
	// 3. Test SharedListRecord with complex view concatenation
	fmt.Println("\n--- Test 3: SharedListRecord (Complex View/Concatenation) ---")
	registros := []businessTypes.SharedListRecord{}
	q3 := db.Query(&registros)
	// This query should use a view that concatenates ListaID and Status or Updated
	err = q3.CompanyID.Equals(1).
		ListID.Equals(1).
		Status.Equals(1).
		Exec()

	if err != nil {
		fmt.Println("Error in Test 3:", err)
	} else {
		fmt.Printf("Found %d records\n", len(registros))
	}

	// 5. Test with range query (Between)
	fmt.Println("\n--- Test 5: Range Query (Between) ---")
	recordRegistrosListas := []businessTypes.SharedListRecord{}
	q5 := db.Query(&recordRegistrosListas)
	err = q5.CompanyID.Equals(1).
		ListID.Equals(1).
		Updated.Between(1000000000, 2000000000).
		Exec()

	if err != nil {
		fmt.Println("Error in Test 5:", err)
	} else {
		fmt.Printf("Found %d records in range\n", len(recordRegistrosListas))
	}

	return core.FuncResponse{}
}
