package types

import (
	finance "app/finance/types"
	"slices"
	"testing"
	"time"
)

func unixDayOf(year int, month time.Month, day int) int16 {
	return int16(time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

// 2026-11-09 is a Monday of ISO week 2026-46.
var mondayOfWeek2646 = unixDayOf(2026, 11, 9)

func TestClientWeekOfDay(t *testing.T) {
	for dayOffset, wantWeekday := range []int{0, 1, 2, 3, 4, 5, 6} {
		week, weekday := ClientWeekOfDay(mondayOfWeek2646 + int16(dayOffset))
		if week != 2646 || weekday != wantWeekday {
			t.Fatalf("day +%d: week %d weekday %d, want 2646 and %d", dayOffset, week, weekday, wantWeekday)
		}
	}
	// 2027-01-01 (a Friday) belongs to ISO week 2026-53.
	if week, weekday := ClientWeekOfDay(unixDayOf(2027, 1, 1)); week != 2653 || weekday != 4 {
		t.Fatalf("2027-01-01: week %d weekday %d, want 2653 and 4", week, weekday)
	}
}

func TestClientWeekSaleChanges(t *testing.T) {
	created := SaleOrder{ClientID: 7, Date: mondayOfWeek2646, Status: OrderStatusPending, TotalAmount: 1000, DebtAmount: 1000}
	assertChanges := func(name string, got []ClientWeekSaleShare, want ...ClientWeekSaleShare) {
		t.Helper()
		if !slices.Equal(got, want) {
			t.Fatalf("%s: %+v, want %+v", name, got, want)
		}
	}
	assertChanges("a new unpaid sale", ClientWeekSaleChanges(nil, &created),
		ClientWeekSaleShare{ClientID: 7, Date: mondayOfWeek2646, Purchase: 1000})

	partlyPaid := created
	partlyPaid.DebtAmount = 400
	assertChanges("a payment counts on the sale's day", ClientWeekSaleChanges(&created, &partlyPaid),
		ClientWeekSaleShare{ClientID: 7, Date: mondayOfWeek2646, Payment: 600})

	annulled := partlyPaid
	annulled.Status = OrderStatusAnnulled
	assertChanges("an annulment takes back the purchase and its payments", ClientWeekSaleChanges(&partlyPaid, &annulled),
		ClientWeekSaleShare{ClientID: 7, Date: mondayOfWeek2646, Purchase: -1000, Payment: -600})

	movedClient := partlyPaid
	movedClient.ClientID = 8
	assertChanges("a client change moves the whole share", ClientWeekSaleChanges(&partlyPaid, &movedClient),
		ClientWeekSaleShare{ClientID: 7, Date: mondayOfWeek2646, Purchase: -1000, Payment: -600},
		ClientWeekSaleShare{ClientID: 8, Date: mondayOfWeek2646, Purchase: 1000, Payment: 600})

	walkIn := created
	walkIn.ClientID = 0
	assertChanges("a sale without client counts nowhere", ClientWeekSaleChanges(nil, &walkIn))
	assertChanges("a delivery changes nothing", ClientWeekSaleChanges(&created, &created))
}

// A USD sale paid in parts telescopes to its whole total in soles, so annulling it leaves zero.
func TestClientWeekSaleChangesUSDTelescopes(t *testing.T) {
	sale := SaleOrder{ClientID: 7, Date: mondayOfWeek2646, Status: OrderStatusPending, CurrencyType: finance.CurrencyUSD,
		ExchangeRate: 3777, TotalAmount: 1001, DebtAmount: 1001}
	paymentSum := int32(0)
	stored := sale
	for _, debtAfter := range []int32{667, 333, 0} {
		paid := stored
		paid.DebtAmount = debtAfter
		for _, change := range ClientWeekSaleChanges(&stored, &paid) {
			paymentSum += change.Payment
		}
		stored = paid
	}
	if paymentSum != sale.TotalInPEN() {
		t.Fatalf("the payments add up to %d, the total is %d", paymentSum, sale.TotalInPEN())
	}
}

func TestBuildClientWeekSales(t *testing.T) {
	saleOrders := []SaleOrder{
		{ClientID: 7, Date: mondayOfWeek2646, Status: OrderStatusPending, TotalAmount: 1000, DebtAmount: 1000},
		{ClientID: 7, Date: mondayOfWeek2646 + 2, Status: OrderStatusCompleted, TotalAmount: 500},
		{ClientID: 7, Date: mondayOfWeek2646 + 2, Status: OrderStatusAnnulled, TotalAmount: 900},
		{ClientID: 0, Date: mondayOfWeek2646, Status: OrderStatusCompleted, TotalAmount: 300},
		{ClientID: 5, Date: mondayOfWeek2646 - 7, Status: OrderStatusPaid, TotalAmount: 200},
	}
	rows := BuildClientWeekSales(1, saleOrders)
	if len(rows) != 2 || rows[0].Week != 2645 || rows[0].ClientID != 5 || rows[1].Week != 2646 || rows[1].ClientID != 7 {
		t.Fatalf("rows: %+v", rows)
	}
	if !slices.Equal(rows[1].Purchases, []int32{1000, 0, 500, 0, 0, 0, 0}) || !slices.Equal(rows[1].Payments, []int32{0, 0, 500, 0, 0, 0, 0}) {
		t.Fatalf("client 7 week 2646: %v / %v", rows[1].Purchases, rows[1].Payments)
	}
}
