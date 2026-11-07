package bench

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	fareward "github.com/ivanjoz/fareward/go"

	logistics "app/logistics/types"
	production "app/production/types"
	sales "app/sales/types"
)

// Dataset is one company holding Rows products, Rows stock rows and Rows sale orders. Each size is
// its own company, so a first sync of one never reads another's partition.
type Dataset struct {
	Name      string
	CompanyID int32
	Rows      int
}

// Datasets are the three catalog sizes the benchmarks sweep. GET.sale-orders caps a tab at 5000
// orders (saleOrdersQueryLimit), so the 50k company measures that cap, not 50k orders.
var Datasets = []Dataset{
	{Name: "1k", CompanyID: 101, Rows: 1_000},
	{Name: "10k", CompanyID: 102, Rows: 10_000},
	{Name: "50k", CompanyID: 103, Rows: 50_000},
}

// MultiCompanyDatasets are eight mid-size tenants. A company is one partition and a partition lives
// on one Scylla shard, so a single-company benchmark only ever loads one core of the database; the
// multi-company scenario spreads its requests across these to measure the server as a whole.
//
// The IDs are picked, not consecutive: with the bench's 2 shards, 201–204 hash to shard 1 and 210,
// 211, 212 and 214 to shard 0. Consecutive 201–208 all landed on shard 1, so that scenario loaded
// one shard like a single company does. Check with Scylla's shard of token(company_id).
var MultiCompanyDatasets = func() []Dataset {
	datasets := make([]Dataset, 0, 8)
	for _, companyID := range []int32{201, 202, 203, 204, 210, 211, 212, 214} {
		datasets = append(datasets, Dataset{Name: fmt.Sprintf("multi-%d", companyID), CompanyID: companyID, Rows: 10_000})
	}
	return datasets
}()

// BenchUserID is the session user of every bench company. Not 1: user 1 skips the access gate,
// and the router benchmark must pay for the fareward frame a regular user pays for.
const BenchUserID = int32(2)

// BenchWarehouseID is the one warehouse all stock and sale orders belong to.
const BenchWarehouseID = int32(1)

const seedBatchSize = 1000

// unlimitedBenchCredits is far past what any run spends: a 429 would measure the limiter's policy,
// not its cost.
const unlimitedBenchCredits = uint64(1) << 40

// seededCompanyIDs remembers the datasets this process already verified, so every benchmark can
// call EnsureSeeded without re-counting 50k rows each time.
var seededCompanyIDs sync.Map

// EnsureSeeded writes whatever part of the dataset is missing. It counts the stored rows of each
// table rather than trusting a marker, so a seed interrupted halfway is completed on the next run.
func EnsureSeeded(dataset Dataset) error {
	Setup()
	if _, alreadySeeded := seededCompanyIDs.Load(dataset.CompanyID); alreadySeeded {
		return nil
	}
	if err := seedDataset(dataset); err != nil {
		return err
	}
	seededCompanyIDs.Store(dataset.CompanyID, true)
	return nil
}

// NoFarewardEnv skips the credit budgets when seeding, for the database-only benchmarks (the stage
// and HandlerConcurrent ones never call fareward). The router and fareward benchmarks need them.
const NoFarewardEnv = "GENIX_BENCH_NO_FAREWARD"

func seedDataset(dataset Dataset) error {
	// A company without a company_credit_budget row has a daily allowance of zero, so fareward
	// refuses its first request. Set through the daemon (opcode 0x05, what the SaaS panel sends) so
	// it applies at once, without waiting for a cache reload. The month figure needs the daily one
	// first; it is set too because the monthly ceiling would otherwise end a long run.
	budgetContext, cancelBudget := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelBudget()
	budgetOperations := []fareward.BudgetOperation{fareward.BudgetSetDaily, fareward.BudgetSetCurrent}
	if os.Getenv(NoFarewardEnv) != "" {
		budgetOperations = nil
	}
	for _, budgetOperation := range budgetOperations {
		if err := fareward.MutateCompanyCreditBudget(
			budgetContext, dataset.CompanyID, budgetOperation, unlimitedBenchCredits, unlimitedBenchCredits,
		); err != nil {
			return fmt.Errorf("setting the credit budget (operation %d): %w", budgetOperation, err)
		}
	}

	if err := db.Upsert(&[]coreTypes.User{{
		CompanyID: dataset.CompanyID, ID: BenchUserID, User: "bench", FirstName: "bench",
		Status: 1, Created: core.SUnixTime(), Updated: core.SUnixTime(),
	}}); err != nil {
		return fmt.Errorf("seeding the bench user: %w", err)
	}

	storedProducts := []production.Product{}
	productsQuery := db.Query(&storedProducts)
	if err := productsQuery.Select(productsQuery.ID).CompanyID.Equals(dataset.CompanyID).Exec(); err != nil {
		return fmt.Errorf("counting products: %w", err)
	}
	if len(storedProducts) < dataset.Rows {
		if err := insertInBatches(dataset.Rows, func(from, to int) error {
			products := makeProducts(dataset.CompanyID, from, to)
			return db.Upsert(&products)
		}); err != nil {
			return fmt.Errorf("seeding products: %w", err)
		}
	}

	storedStock := []logistics.ProductStock{}
	stockQuery := db.Query(&storedStock)
	if err := stockQuery.Select(stockQuery.ID).CompanyID.Equals(dataset.CompanyID).Exec(); err != nil {
		return fmt.Errorf("counting stock rows: %w", err)
	}
	if len(storedStock) < dataset.Rows {
		if err := insertInBatches(dataset.Rows, func(from, to int) error {
			stockRows := makeProductStock(dataset.CompanyID, from, to)
			return db.Upsert(&stockRows)
		}); err != nil {
			return fmt.Errorf("seeding stock rows: %w", err)
		}
	}

	storedOrders := []sales.SaleOrder{}
	ordersQuery := db.Query(&storedOrders)
	if err := ordersQuery.Select(ordersQuery.ID).CompanyID.Equals(dataset.CompanyID).Exec(); err != nil {
		return fmt.Errorf("counting sale orders: %w", err)
	}
	if len(storedOrders) < dataset.Rows {
		if err := insertInBatches(dataset.Rows, func(from, to int) error {
			orders := makeSaleOrders(dataset.CompanyID, from, to)
			return db.Upsert(&orders)
		}); err != nil {
			return fmt.Errorf("seeding sale orders: %w", err)
		}
	}
	return nil
}

// insertInBatches calls insertBatch over [1, rowCount] in ranges of seedBatchSize. IDs are 1-based
// and explicit, so re-running a batch overwrites the same rows instead of duplicating them.
func insertInBatches(rowCount int, insertBatch func(fromID, toID int) error) error {
	for fromID := 1; fromID <= rowCount; fromID += seedBatchSize {
		toID := min(fromID+seedBatchSize-1, rowCount)
		if err := insertBatch(fromID, toID); err != nil {
			return err
		}
	}
	return nil
}

// makeProducts fills the columns a real catalog row carries, with text of realistic length: the
// colbin and zstd costs depend on the bytes, not on the row count.
func makeProducts(companyID int32, fromID, toID int) []production.Product {
	products := make([]production.Product, 0, toID-fromID+1)
	for productID := fromID; productID <= toID; productID++ {
		price := int32(500 + productID%20_000)
		products = append(products, production.Product{
			CompanyID:   companyID,
			ID:          int32(productID),
			Name:        fmt.Sprintf("Producto de prueba %06d presentación estándar", productID),
			Description: fmt.Sprintf("Descripción del producto %d para el benchmark del backend, con un texto de largo típico.", productID),
			SKU:         fmt.Sprintf("SKU-%08d", productID),
			CategoryIDs: []int32{int32(1 + productID%40), int32(41 + productID%15)},
			BrandID:     int32(1 + productID%120),
			BrandName:   fmt.Sprintf("Marca %d", 1+productID%120),
			Price:       price,
			FinalPrice:  price,
			CurrencyID:  1,
			UnitID:      1,
			ImageMain:   int32(productID*10 + 1),
			ImageIDs:    []int32{int32(productID*10 + 1), int32(productID*10 + 2)},
			Origin:      1,
			Status:      production.ProductStatusActive,
		})
	}
	return products
}

func makeProductStock(companyID int32, fromID, toID int) []logistics.ProductStock {
	stockRows := make([]logistics.ProductStock, 0, toID-fromID+1)
	for productID := fromID; productID <= toID; productID++ {
		quantity := int32(1 + productID%500)
		stockRows = append(stockRows, logistics.ProductStock{
			CompanyID:   companyID,
			WarehouseID: BenchWarehouseID,
			ProductID:   int32(productID),
			Quantity:    quantity,
			SubDivisor:  1,
			AverageCost: int32(400 + productID%15_000),
			StockValue:  int64(quantity) * int64(400+productID%15_000),
			Status:      1,
		})
	}
	return stockRows
}

// makeSaleOrders creates pending orders (Status 1) of three lines each, spread over the last year.
// Pending is what the till's "por pagar" tab (pending-status=2) lists.
func makeSaleOrders(companyID int32, fromID, toID int) []sales.SaleOrder {
	today := core.FechaUnix()
	orders := make([]sales.SaleOrder, 0, toID-fromID+1)
	for orderID := fromID; orderID <= toID; orderID++ {
		lines := make([]sales.SaleOrderLine, 3)
		totalAmount := int32(0)
		for lineIndex := range lines {
			productID := int32(1 + (orderID*3+lineIndex)%1000)
			linePrice := 500 + productID*7
			lines[lineIndex] = sales.SaleOrderLine{
				ProductID: productID, Price: linePrice, Quantity: 1 * core.QuantityLineScale, SubDivisor: 1,
				Sku: fmt.Sprintf("SKU-%08d", productID),
			}
			totalAmount += linePrice
		}
		order := sales.SaleOrder{
			CompanyID:    companyID,
			Date:         today - int16(orderID%365),
			WarehouseID:  BenchWarehouseID,
			ID:           int64(orderID),
			Lines:        lines,
			TotalAmount:  totalAmount,
			TaxAmount:    totalAmount * 18 / 118,
			DebtAmount:   totalAmount,
			CurrencyType: 1,
			ClientID:     int32(1 + orderID%300),
			Status:       sales.OrderStatusPending,
		}
		order.StampPeriod()
		orders = append(orders, order)
	}
	return orders
}
