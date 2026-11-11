//go:build !js && !oceanbase && !postgres

package exec

import (
	"app/core"
	"app/db"
	sales "app/sales/types"
	"os"
	"slices"
	"testing"

	"github.com/ivanjoz/genix-orm/mariadbdriver"
)

// The backfill against a real MariaDB: GENIX_CLIENT_WEEK_TEST=1 runs it on the local load-test
// container (127.0.0.1:3307, root / genix), in a database of its own that is wiped first.
func TestBackfillClientWeekSales(t *testing.T) {
	if os.Getenv("GENIX_CLIENT_WEEK_TEST") != "1" {
		t.Skip("set GENIX_CLIENT_WEEK_TEST=1 to run against the local MariaDB (127.0.0.1:3307)")
	}
	db.Connect(db.ConnectParams{Keyspace: "genix_client_week_backfill_test",
		MariaDB: db.MariaDBParams{Host: "127.0.0.1", Port: 3307, User: "root", Password: "genix"}})
	if err := mariadbdriver.WipeDatabase(); err != nil {
		t.Fatal(err)
	}
	if err := db.Deploy(db.MakeController[sales.ClientWeekSales](), db.MakeController[sales.SaleOrder]()); err != nil {
		t.Fatal(err)
	}
	const companyID = int32(1)
	today := core.FechaUnix()

	// Sales stored before the table existed: no row was ever incremented for them.
	saleOrders := []sales.SaleOrder{
		{CompanyID: companyID, ID: 1, ClientID: 7, Date: today, Status: sales.OrderStatusPending, TotalAmount: 1000, DebtAmount: 400},
		{CompanyID: companyID, ID: 2, ClientID: 7, Date: today - 7, Status: sales.OrderStatusCompleted, TotalAmount: 250},
		{CompanyID: companyID, ID: 3, ClientID: 9, Date: today, Status: sales.OrderStatusAnnulled, TotalAmount: 900},
	}
	for index := range saleOrders {
		saleOrders[index].StampPeriod()
	}
	if err := db.Insert(&saleOrders); err != nil {
		t.Fatal(err)
	}
	// One drifted row and one no sale accounts for (client 9's sale is annulled).
	drifted := sales.NewClientWeekSales(companyID, sales.ClientWeekSaleShare{ClientID: 7, Date: today, Purchase: 5})
	stale := sales.NewClientWeekSales(companyID, sales.ClientWeekSaleShare{ClientID: 9, Date: today, Purchase: 900})
	if err := db.Upsert(&[]sales.ClientWeekSales{drifted, stale}); err != nil {
		t.Fatal(err)
	}
	readRows := func() []sales.ClientWeekSales {
		rows := []sales.ClientWeekSales{}
		if err := db.Query(&rows).CompanyID.Equals(companyID).Exec(); err != nil {
			t.Fatal(err)
		}
		return rows
	}

	simulated := clientWeekSalesBackfillReport{}
	if err := backfillCompanyClientWeekSales(companyID, false, &simulated); err != nil {
		t.Fatal(err)
	}
	if simulated.rowsBuilt != 2 || simulated.rowsZeroed != 1 || simulated.rowsWritten != 0 || len(readRows()) != 2 {
		t.Fatalf("the simulation: %+v", simulated)
	}

	applied := clientWeekSalesBackfillReport{}
	if err := backfillCompanyClientWeekSales(companyID, true, &applied); err != nil {
		t.Fatal(err)
	}
	if applied.rowsWritten != 3 {
		t.Fatalf("applied: %+v", applied)
	}
	rebuilt := sales.BuildClientWeekSales(companyID, saleOrders)
	for _, row := range readRows() {
		if row.ClientID == 9 {
			if slices.ContainsFunc(row.Purchases, func(amount int32) bool { return amount != 0 }) {
				t.Fatalf("the stale row was not zeroed: %+v", row)
			}
			continue
		}
		position := slices.IndexFunc(rebuilt, func(built sales.ClientWeekSales) bool {
			return built.Week == row.Week && built.ClientID == row.ClientID
		})
		if position < 0 || !slices.Equal(rebuilt[position].Purchases, row.Purchases) || !slices.Equal(rebuilt[position].Payments, row.Payments) {
			t.Fatalf("row %+v differs from the rebuild %+v", row, rebuilt)
		}
	}

	rerun := clientWeekSalesBackfillReport{}
	if err := backfillCompanyClientWeekSales(companyID, true, &rerun); err != nil {
		t.Fatal(err)
	}
	if rerun.rowsUnchanged != 3 || rerun.rowsWritten != 0 {
		t.Fatalf("a re-run should write nothing: %+v", rerun)
	}
}
