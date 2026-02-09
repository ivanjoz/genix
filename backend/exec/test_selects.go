package exec

import (
	"app/core"
	"app/db"
	"app/types"
	comercial "app/comercial/types"
	"fmt"
)

// TestSelects demonstrates complex select queries including the new KeyConcatenated smart logic.
func TestSelects(args *core.ExecArgs) core.FuncResponse {
	var err error

	fmt.Println("\n--- Test 7: Range query in int packet column local index ---")
	recordSalesOrders2 := []comercial.SaleOrder{}
	q7 := db.Query(&recordSalesOrders2)
	err = q7.EmpresaID.Equals(1).
		Status.Equals(1).
		Updated.Between(385298000, 385299000).
		Exec()

	if err != nil {
		fmt.Println("Error in Test 6:", err)
	} else {
		fmt.Printf("Found %d records in range\n", len(recordSalesOrders2))
	}
	
	return  core.FuncResponse{}
	
	// 6. Test bucket query CONTAINS + "RANGE" with hash index
	fmt.Println("\n--- Test 5: Range Query (Between) ---")
	recordSalesOrders := []comercial.SaleOrder{}
	q6 := db.Query(&recordSalesOrders)
	err = q6.EmpresaID.Equals(1).
		DetailProductsIDs.Contains(1).
		Fecha.Between(20475, 20495).
		Exec()

	if err != nil {
		fmt.Println("Error in Test 6:", err)
	} else {
		fmt.Printf("Found %d records in range\n", len(recordSalesOrders))
	}
	
	// 1. Test AlmacenProducto with KeyConcatenated Smart Logic
	// This should trigger a range query on the 'id' column because it's the first column of KeyConcatenated.
	fmt.Println("\n--- Test 1: AlmacenProducto (Smart ORM for KeyConcatenated) ---")
	productos := []types.AlmacenProducto{}
	q1 := db.Query(&productos)
	err = q1.EmpresaID.Equals(1).
		AlmacenID.Equals(1). // This is the first column in KeyConcatenated for AlmacenProducto
		Limit(5).Exec()

	if err != nil {
		fmt.Println("Error in Test 1:", err)
	} else {
		fmt.Printf("Found %d products\n", len(productos))
	}

	// 2. Test AlmacenProducto with multiple prefix columns
	fmt.Println("\n--- Test 2: AlmacenProducto (Multiple prefix columns) ---")
	productos2 := []types.AlmacenProducto{}
	q2 := db.Query(&productos2)
	err = q2.EmpresaID.Equals(1).
		AlmacenID.Equals(1).
		ProductoID.GreaterEqual(8).
		Limit(5).Exec()

	if err != nil {
		fmt.Println("Error in Test 2:", err)
	} else {
		fmt.Printf("Found %d products\n", len(productos2))
	}
	
	// New test
	fmt.Println("\n--- Test 21: AlmacenProducto. Using view: []db.Coln{e.AlmacenID, e.Status, e.Updated} ---")
	productos21 := []types.AlmacenProducto{}
	q21 := db.Query(&productos21)
	err = q21.EmpresaID.Equals(1).
		AlmacenID.Equals(1).Status.Equals(1).Updated.GreaterEqual(1000).
		Limit(5).Exec()

	if err != nil {
		fmt.Println("Error in Test 21:", err)
	} else {
		fmt.Printf("Found %d products\n", len(productos2))
	}

	// 3. Test ListaCompartidaRegistro with complex view concatenation
	fmt.Println("\n--- Test 3: ListaCompartidaRegistro (Complex View/Concatenation) ---")
	registros := []types.ListaCompartidaRegistro{}
	q3 := db.Query(&registros)
	// This query should use a view that concatenates ListaID and Status or Updated
	err = q3.EmpresaID.Equals(1).
		ListaID.Equals(1).
		Status.Equals(1).
		Exec()

	if err != nil {
		fmt.Println("Error in Test 3:", err)
	} else {
		fmt.Printf("Found %d records\n", len(registros))
	}

	// 4. Test CajaMovimiento with View
	fmt.Println("\n--- Test 4: CajaMovimiento (Query using View) ---")
	movimientos := []types.CajaMovimiento{}
	q4 := db.Query(&movimientos)
	err = q4.EmpresaID.Equals(1).
		DocumentoID.Equals(12345). // This uses a view defined in CajaMovimientoTable
		Exec()

	if err != nil {
		fmt.Println("Error in Test 4:", err)
	} else {
		fmt.Printf("Found %d movements\n", len(movimientos))
	}

	// 5. Test with range query (Between)
	fmt.Println("\n--- Test 5: Range Query (Between) ---")
	recordRegistrosListas := []types.ListaCompartidaRegistro{}
	q5 := db.Query(&recordRegistrosListas)
	err = q5.EmpresaID.Equals(1).
		ListaID.Equals(1).
		Updated.Between(1000000000, 2000000000).
		Exec()

	if err != nil {
		fmt.Println("Error in Test 5:", err)
	} else {
		fmt.Printf("Found %d records in range\n", len(recordRegistrosListas))
	}


	return core.FuncResponse{}
}
