//go:build !js && !oceanbase && !postgres

package sales

import (
	"app/core"
	"app/db"
	finance "app/finance/types"
	"app/sales/types"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/ivanjoz/genix-orm/mariadbdriver"
)

// The client week hooks against a real MariaDB: GENIX_CLIENT_WEEK_TEST=1 runs them on the local
// load-test container (127.0.0.1:3307, root / genix), in a database of their own that is wiped first.
// Every sale write applies its changes inside its own transaction, as PostSaleOrder and
// PostSaleOrderAnnul do; the rows must then equal what the backfill rebuilds from the stored sales.

const clientWeekTestCompanyID = int32(1)

func connectClientWeekTestDatabase(t *testing.T) {
	if os.Getenv("GENIX_CLIENT_WEEK_TEST") != "1" {
		t.Skip("set GENIX_CLIENT_WEEK_TEST=1 to run against the local MariaDB (127.0.0.1:3307)")
	}
	db.Connect(db.ConnectParams{Keyspace: "genix_client_week_test",
		MariaDB: db.MariaDBParams{Host: "127.0.0.1", Port: 3307, User: "root", Password: "genix"}})
	if err := mariadbdriver.WipeDatabase(); err != nil {
		t.Fatal(err)
	}
	if err := db.Deploy(db.MakeController[types.ClientWeekSales](), db.MakeController[types.SaleOrder]()); err != nil {
		t.Fatal(err)
	}
}

// writeSaleWithChanges stores the sale and its client week changes in one transaction.
func writeSaleWithChanges(t *testing.T, stored *types.SaleOrder, written types.SaleOrder) types.SaleOrder {
	t.Helper()
	written.CompanyID = clientWeekTestCompanyID
	written.StampPeriod()
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		if err := db.UpsertIn(transaction, &[]types.SaleOrder{written}); err != nil {
			return err
		}
		return types.ApplyClientWeekSaleChangesIn(transaction, clientWeekTestCompanyID, types.ClientWeekSaleChanges(stored, &written))
	})
	if err != nil {
		t.Fatalf("writing sale %d: %v", written.ID, err)
	}
	return written
}

func readClientWeekRows(t *testing.T) []types.ClientWeekSales {
	t.Helper()
	rows := []types.ClientWeekSales{}
	if err := db.Query(&rows).CompanyID.Equals(clientWeekTestCompanyID).Exec(); err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(rows, func(left, right types.ClientWeekSales) int {
		if left.Week != right.Week {
			return int(left.Week - right.Week)
		}
		return int(left.ClientID - right.ClientID)
	})
	return rows
}

func TestClientWeekSalesHooks(t *testing.T) {
	connectClientWeekTestDatabase(t)
	today := core.FechaUnix()
	lastWeek := today - 7

	// An unpaid sale, then two payments of it: they count on the sale's day.
	unpaid := writeSaleWithChanges(t, nil, types.SaleOrder{ID: 101, ClientID: 7, Date: today, Status: types.OrderStatusPending, TotalAmount: 1000, DebtAmount: 1000})
	partlyPaid := unpaid
	partlyPaid.DebtAmount = 400
	partlyPaid = writeSaleWithChanges(t, &unpaid, partlyPaid)
	paid := partlyPaid
	paid.DebtAmount, paid.Status = 0, types.OrderStatusPaid
	writeSaleWithChanges(t, &partlyPaid, paid)

	// A USD sale paid at once, restated in soles with its rate.
	writeSaleWithChanges(t, nil, types.SaleOrder{ID: 102, ClientID: 7, Date: lastWeek, Status: types.OrderStatusPaid,
		CurrencyType: finance.CurrencyUSD, ExchangeRate: 3777, TotalAmount: 1001})

	// A sale whose client changes on a later update moves to the new client's row.
	firstClient := writeSaleWithChanges(t, nil, types.SaleOrder{ID: 103, ClientID: 8, Date: today, Status: types.OrderStatusPending, TotalAmount: 500, DebtAmount: 500})
	movedClient := firstClient
	movedClient.ClientID = 9
	writeSaleWithChanges(t, &firstClient, movedClient)

	// An annulled sale leaves nothing behind.
	toAnnul := writeSaleWithChanges(t, nil, types.SaleOrder{ID: 104, ClientID: 7, Date: today, Status: types.OrderStatusPaid, TotalAmount: 300})
	annulled := toAnnul
	annulled.Status = types.OrderStatusAnnulled
	writeSaleWithChanges(t, &toAnnul, annulled)

	// Eight first sales of one client and week at once: one creates the row, the others' duplicate key
	// reruns their transaction, which then increments it.
	var concurrentWrites sync.WaitGroup
	for saleIndex := range 8 {
		concurrentWrites.Add(1)
		go func() {
			defer concurrentWrites.Done()
			writeSaleWithChanges(t, nil, types.SaleOrder{ID: int64(200 + saleIndex), ClientID: 20, Date: today,
				Status: types.OrderStatusPending, TotalAmount: 100, DebtAmount: 100})
		}()
	}
	concurrentWrites.Wait()

	storedSales := []types.SaleOrder{}
	if err := db.Query(&storedSales).CompanyID.Equals(clientWeekTestCompanyID).Exec(); err != nil {
		t.Fatal(err)
	}
	rebuilt := types.BuildClientWeekSales(clientWeekTestCompanyID, storedSales)
	incremented := readClientWeekRows(t)
	_, todayWeekday := types.ClientWeekOfDay(today)
	for _, row := range incremented {
		// Client 8's row is left at zero by the move; the rebuild has no row for it.
		if row.ClientID == 8 {
			if slices.ContainsFunc(row.Purchases, func(amount int32) bool { return amount != 0 }) {
				t.Fatalf("client 8 kept the moved sale: %+v", row)
			}
			continue
		}
		position := slices.IndexFunc(rebuilt, func(built types.ClientWeekSales) bool {
			return built.Week == row.Week && built.ClientID == row.ClientID
		})
		if position < 0 || !slices.Equal(rebuilt[position].Purchases, row.Purchases) || !slices.Equal(rebuilt[position].Payments, row.Payments) {
			t.Fatalf("incremented row %+v differs from the rebuild %+v", row, rebuilt)
		}
	}
	if len(incremented) != len(rebuilt)+1 {
		t.Fatalf("%d incremented rows, the rebuild has %d (+1 zeroed)", len(incremented), len(rebuilt))
	}
	todayWeek, _ := types.ClientWeekOfDay(today)
	lastWeekCode, _ := types.ClientWeekOfDay(lastWeek)
	rowOf := func(clientID int32, week int16) types.ClientWeekSales {
		return incremented[slices.IndexFunc(incremented, func(row types.ClientWeekSales) bool { return row.ClientID == clientID && row.Week == week })]
	}
	// The annulled sale's 300 is gone; the paid sale's 1000 is fully paid.
	if client7Today := rowOf(7, todayWeek); client7Today.Purchases[todayWeekday] != 1000 || client7Today.Payments[todayWeekday] != 1000 {
		t.Fatalf("client 7 today: %+v", client7Today)
	}
	// 1001 USD cents at 3.777 are 3781 soles cents.
	if client7LastWeek := rowOf(7, lastWeekCode); client7LastWeek.Purchases[todayWeekday] != 3781 || client7LastWeek.Payments[todayWeekday] != 3781 {
		t.Fatalf("client 7 last week: %+v", client7LastWeek)
	}
	client20 := incremented[slices.IndexFunc(incremented, func(row types.ClientWeekSales) bool { return row.ClientID == 20 })]
	if client20.Purchases[todayWeekday] != 800 {
		t.Fatalf("the concurrent first sales add up to %d, want 800", client20.Purchases[todayWeekday])
	}

	// The customers list's delta read of the last weeks returns them.
	synced := []types.ClientWeekSales{}
	query := db.Query(&synced)
	query.CompanyID.Equals(clientWeekTestCompanyID).Delta(db.DeltaCursor{}, lastClientWeeks(today)...)
	if err := query.Exec(); err != nil {
		t.Fatal(err)
	}
	if syncedRows := slices.DeleteFunc(synced, func(row types.ClientWeekSales) bool { return row.ID <= 0 }); len(syncedRows) != len(incremented) {
		t.Fatalf("the delta read returned %d rows, want %d", len(syncedRows), len(incremented))
	}
}
