package types

import (
	"app/core"
	"app/db"
	"errors"
	"slices"
)

// ClientWeekSales is one client's sales of one ISO week, summed per weekday (index 0 = Monday … 6 =
// Sunday), in soles: what the client bought (Purchases) and how much of it is paid (Payments). A
// payment counts on the day of the sale it pays, so Purchases - Payments is always what is still
// owed of that day's sales. Each sale write adds what it changed (ClientWeekSaleChanges) in SQL:
// the days are expanded slices, so a sale is one UPDATE of one row, never a read.
type ClientWeekSales struct {
	db.TableStruct[ClientWeekSalesTable, ClientWeekSales]
	CompanyID int32 `cb:"1"`
	// ID is packed from Week and ClientID (KeyIntPacking): the delta index needs one integer key.
	ID int64 `cb:"2"`
	// Week is the ISO week as YYWW (2646), core.FecSemana.Code.
	Week      int16   `cb:"3"`
	ClientID  int32   `cb:"4"`
	Purchases []int32 `cb:"5"`
	Payments  []int32 `cb:"6"`
	Updated   int32   `json:"upd" cb:"upd,7"`
	Version   int16   `json:"ver" cb:"ver,8"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,9"`
}

type ClientWeekSalesTable struct {
	db.TableStruct[ClientWeekSalesTable, ClientWeekSales]
	CompanyID db.Col[*ClientWeekSalesTable, int32]
	ID        db.Col[*ClientWeekSalesTable, int64]
	Week      db.Col[*ClientWeekSalesTable, int16]
	ClientID  db.Col[*ClientWeekSalesTable, int32]
	Purchases db.ColSlice[*ClientWeekSalesTable, int32]
	Payments  db.ColSlice[*ClientWeekSalesTable, int32]
	Updated   db.Col[*ClientWeekSalesTable, int32]
	Version   db.Col[*ClientWeekSalesTable, int16]
}

// DaysPerWeek is the size of the expanded day slices.
const DaysPerWeek = 7

func (e ClientWeekSalesTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        91,
		Name:      "client_week_sales",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
		// Week leads the packed key, so the weeks of every client are one primary-key range.
		KeyIntPacking:  db.Cols(e.Week.Size(14), e.ClientID),
		ExpandedSlices: db.Cols(e.Purchases.Expand(DaysPerWeek), e.Payments.Expand(DaysPerWeek)),
		// Sizes the Week slot of the delta index: YYWW up to 9953.
		FixedValues: []db.FixedValues{{Col: e.Week, Min: 0, Max: 9953}},
		// The customers list syncs the last 26 weeks: Delta(cursor, weeks...).
		Indexes: []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Week)}},
	}
}

// ClientWeekOfDay is the ISO week (YYWW) of a UnixDay and its weekday index, 0 = Monday.
func ClientWeekOfDay(unixDay int16) (int16, int) {
	// The epoch (UnixDay 0) was a Thursday, index 3.
	return core.MakeSemanaFromFechaUnix(unixDay, false).Code, int((unixDay + 3) % 7)
}

// ClientWeekSaleShare is what one sale counts in its client's week row: its total and the part of
// it paid, in soles, on the sale's day.
type ClientWeekSaleShare struct {
	ClientID int32
	Date     int16
	Purchase int32
	Payment  int32
}

// ClientWeekSaleShare is nothing for a sale without a client or an annulled one: an annulment takes
// back the purchase and its payments from the day of the sale, as if it never happened. The paid
// part is restated in soles as a whole, so every write's share telescopes to the exact total.
func (e *SaleOrder) ClientWeekSaleShare() ClientWeekSaleShare {
	if e.ClientID <= 0 || e.Status == OrderStatusAnnulled {
		return ClientWeekSaleShare{}
	}
	return ClientWeekSaleShare{ClientID: e.ClientID, Date: e.Date,
		Purchase: e.TotalInPEN(), Payment: e.AmountInPEN(e.TotalAmount - e.DebtAmount)}
}

// ClientWeekSaleChanges is what a write of a sale moves in the client week rows: its share once
// written minus its share as stored (nil for a new sale). A sale that changed client takes its whole
// share from the old client's row to the new one's. Changes of zero are dropped.
func ClientWeekSaleChanges(stored *SaleOrder, written *SaleOrder) []ClientWeekSaleShare {
	storedShare := ClientWeekSaleShare{}
	if stored != nil {
		storedShare = stored.ClientWeekSaleShare()
	}
	writtenShare := written.ClientWeekSaleShare()
	changes := []ClientWeekSaleShare{}
	if storedShare.ClientID == writtenShare.ClientID && storedShare.Date == writtenShare.Date {
		changes = append(changes, ClientWeekSaleShare{ClientID: writtenShare.ClientID, Date: writtenShare.Date,
			Purchase: writtenShare.Purchase - storedShare.Purchase, Payment: writtenShare.Payment - storedShare.Payment})
	} else {
		changes = append(changes, ClientWeekSaleShare{ClientID: storedShare.ClientID, Date: storedShare.Date,
			Purchase: -storedShare.Purchase, Payment: -storedShare.Payment}, writtenShare)
	}
	return slices.DeleteFunc(changes, func(change ClientWeekSaleShare) bool {
		return change.ClientID <= 0 || (change.Purchase == 0 && change.Payment == 0)
	})
}

// NewClientWeekSales is the row a client's first sale of a week creates, holding that sale's change.
func NewClientWeekSales(companyID int32, change ClientWeekSaleShare) ClientWeekSales {
	week, weekday := ClientWeekOfDay(change.Date)
	row := ClientWeekSales{CompanyID: companyID, Week: week, ClientID: change.ClientID,
		Purchases: make([]int32, DaysPerWeek), Payments: make([]int32, DaysPerWeek)}
	row.Purchases[weekday], row.Payments[weekday] = change.Purchase, change.Payment
	return row
}

// BuildClientWeekSales sums the shares of a company's sales into its week rows, ordered by week and
// client: what the increments of every sale write add up to, which the backfill writes.
func BuildClientWeekSales(companyID int32, saleOrders []SaleOrder) []ClientWeekSales {
	type rowKey struct {
		week     int16
		clientID int32
	}
	rowsByKey := map[rowKey]*ClientWeekSales{}
	for index := range saleOrders {
		share := saleOrders[index].ClientWeekSaleShare()
		if share.ClientID <= 0 {
			continue
		}
		week, weekday := ClientWeekOfDay(share.Date)
		row := rowsByKey[rowKey{week, share.ClientID}]
		if row == nil {
			newRow := NewClientWeekSales(companyID, ClientWeekSaleShare{ClientID: share.ClientID, Date: share.Date})
			row = &newRow
			rowsByKey[rowKey{week, share.ClientID}] = row
		}
		row.Purchases[weekday] += share.Purchase
		row.Payments[weekday] += share.Payment
	}
	rows := make([]ClientWeekSales, 0, len(rowsByKey))
	for _, row := range rowsByKey {
		rows = append(rows, *row)
	}
	slices.SortFunc(rows, func(left, right ClientWeekSales) int {
		if left.Week != right.Week {
			return int(left.Week - right.Week)
		}
		return int(left.ClientID - right.ClientID)
	})
	return rows
}

// ApplyClientWeekSaleChangesIn adds what a sale write changed to its client's week rows, in the
// sale's transaction, so the row can never disagree with the sale. The week's first sale of a client
// creates the row; when two sales race to create it, the loser's duplicate key reruns its whole
// transaction, which then increments the winner's row.
func ApplyClientWeekSaleChangesIn(transaction *db.Transaction, companyID int32, changes []ClientWeekSaleShare) error {
	table := db.TableOf[ClientWeekSales]()
	for _, change := range changes {
		week, weekday := ClientWeekOfDay(change.Date)
		key := ClientWeekSales{CompanyID: companyID, Week: week, ClientID: change.ClientID}
		err := db.IncrementIn(transaction, key, table.Purchases.At(weekday).Add(change.Purchase), table.Payments.At(weekday).Add(change.Payment))
		if errors.Is(err, db.ErrIncrementNotStored) {
			err = db.InsertIn(transaction, &[]ClientWeekSales{NewClientWeekSales(companyID, change)})
		}
		if err != nil {
			return core.Err("Error al acumular las ventas semanales del cliente:", err)
		}
	}
	return nil
}
