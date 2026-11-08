//go:build mariadb || postgres

package types

import (
	"app/core"
	"app/db"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Concurrent stock writers against a real database. There is no process lock any more: these
// writers are serialized only by the stock row locks, the plain INSERT of a new stock row and the
// lot's unique index, exactly as writers in different backend processes would be.
//
//	GENIX_STOCK_DB_TEST=1 go test -tags mariadb  -run TestStockConcurrency ./logistics/types/   (127.0.0.1:3309)
//	GENIX_STOCK_DB_TEST=1 go test -tags postgres -run TestStockConcurrency ./logistics/types/   (127.0.0.1:5434)

func connectStockTestDatabase(t *testing.T) int32 {
	t.Helper()
	if os.Getenv("GENIX_STOCK_DB_TEST") != "1" {
		t.Skip("set GENIX_STOCK_DB_TEST=1 to run against the test database")
	}
	portFromEnv := func(name string, fallback int) int {
		if port, err := strconv.Atoi(os.Getenv(name)); err == nil {
			return port
		}
		return fallback
	}
	db.Connect(db.ConnectParams{
		Keyspace:  "genix_stock_concurrency",
		Scripting: true,
		MariaDB:   db.MariaDBParams{Host: "127.0.0.1", Port: portFromEnv("GENIX_MARIADB_PORT", 3309), User: "root", Password: "genix"},
		Postgres:  db.PostgresParams{Host: "127.0.0.1", Port: portFromEnv("GENIX_POSTGRES_PORT", 5434), User: "postgres", Password: "genix"},
	})
	if err := db.Deploy(
		db.MakeController[ProductStock](),
		db.MakeController[ProductStockDetail](),
		db.MakeController[ProductStockLot](),
		db.MakeController[WarehouseProductMovement](),
	); err != nil {
		t.Fatal(err)
	}
	// A fresh company per run, so nothing from an earlier run is read back.
	return int32(time.Now().UnixNano()%1_000_000) + 10_000
}

// applyConcurrently runs every movement batch in its own goroutine, all at once, and returns each
// batch's error.
func applyConcurrently(companyID int32, batches [][]InternalMovement) []error {
	errorsByBatch := make([]error, len(batches))
	start := make(chan struct{})
	var writers sync.WaitGroup
	for batchIndex, batch := range batches {
		writers.Add(1)
		go func() {
			defer writers.Done()
			<-start
			req := &core.HandlerArgs{User: &core.UsuarioToken{CompanyID: companyID, ID: 1}}
			errorsByBatch[batchIndex] = ApplyMovimientos(req, batch)
		}()
	}
	close(start)
	writers.Wait()
	return errorsByBatch
}

func readStockRow(t *testing.T, companyID int32, stockID int64) ProductStock {
	t.Helper()
	stocks := []ProductStock{}
	query := db.Query(&stocks)
	if err := query.CompanyID.Equals(companyID).ID.Equals(stockID).Exec(); err != nil {
		t.Fatal(err)
	}
	if len(stocks) != 1 {
		t.Fatalf("stock row %d: %d rows", stockID, len(stocks))
	}
	return stocks[0]
}

func TestStockConcurrency(t *testing.T) {
	companyID := connectStockTestDatabase(t)
	const warehouseID, productID = int32(3), int32(77)
	stockID := PackProductStockID(warehouseID, productID, 0)

	// 20 writers all create the same missing stock row: one INSERT wins, the others fail on the
	// duplicate key, rerun, and add to the winner's row. No unit may be lost.
	inflows := make([][]InternalMovement, 20)
	for i := range inflows {
		inflows[i] = []InternalMovement{{Type: StockMovementTypeManualEntry, WarehouseID: warehouseID, ProductID: productID, Quantity: 1, Price: 100}}
	}
	for batchIndex, err := range applyConcurrently(companyID, inflows) {
		if err != nil {
			t.Fatalf("inflow %d: %v", batchIndex, err)
		}
	}
	if stock := readStockRow(t, companyID, stockID); stock.Quantity != 20 || stock.StockValue != 2000 {
		t.Fatalf("after 20 concurrent inflows of 1: Quantity %d StockValue %d, want 20 and 2000", stock.Quantity, stock.StockValue)
	}

	// 30 writers take 1 unit each from 20: the check runs on the locked row, so exactly 20 succeed.
	outflows := make([][]InternalMovement, 30)
	for i := range outflows {
		outflows[i] = []InternalMovement{{Type: StockMovementTypeManualExit, WarehouseID: warehouseID, ProductID: productID, Quantity: -1}}
	}
	succeeded := 0
	for _, err := range applyConcurrently(companyID, outflows) {
		if err == nil {
			succeeded++
		}
	}
	if stock := readStockRow(t, companyID, stockID); succeeded != 20 || stock.Quantity != 0 || stock.StockValue != 0 {
		t.Fatalf("30 concurrent outflows of 1 from 20: %d succeeded, Quantity %d StockValue %d; want 20, 0, 0",
			succeeded, stock.Quantity, stock.StockValue)
	}

	// 10 receptions name the same new lot at once: the unique index keeps one lot, and every unit
	// lands in its one detail row.
	const lotProductID = int32(78)
	lotReceptions := make([][]InternalMovement, 10)
	for i := range lotReceptions {
		lotReceptions[i] = []InternalMovement{{Type: StockMovementTypeManualEntry, WarehouseID: warehouseID,
			ProductID: lotProductID, Quantity: 1, Price: 100, SupplierID: 5, LotName: "L-race"}}
	}
	for batchIndex, err := range applyConcurrently(companyID, lotReceptions) {
		if err != nil {
			t.Fatalf("lot reception %d: %v", batchIndex, err)
		}
	}
	lots := []ProductStockLot{}
	lotQuery := db.Query(&lots)
	if err := lotQuery.CompanyID.Equals(companyID).Name.Equals("L-race").Exec(); err != nil {
		t.Fatal(err)
	}
	if len(lots) != 1 {
		t.Fatalf("10 concurrent receptions of a new lot created %d lots, want 1", len(lots))
	}
	if stock := readStockRow(t, companyID, PackProductStockID(warehouseID, lotProductID, 0)); stock.DetailQuantity != 10 {
		t.Fatalf("lot stock DetailQuantity %d, want 10", stock.DetailQuantity)
	}
}
