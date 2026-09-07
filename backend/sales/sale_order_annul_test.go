package sales

import (
	finance "app/finance/types"
	logistics "app/logistics/types"
	"app/sales/types"
	"testing"
)

// The refund is what the sale collected minus what it already gave back. Reading it from the
// ledger instead of from Status is what lets a retry after a partial failure give back only the
// part that never left.
func TestNetCashRefundAmount(t *testing.T) {
	collection := finance.CashBankMovement{Type: finance.CashMovementTypeSaleCollection, Amount: 5000}
	refund := finance.CashBankMovement{Type: finance.CashMovementTypeSaleRefund, Amount: -5000}
	unrelated := finance.CashBankMovement{Type: finance.CashMovementTypeExpensePayment, Amount: -900}

	checks := []struct {
		name      string
		movements []finance.CashBankMovement
		expected  int32
	}{
		{"never paid", nil, 0},
		{"paid once", []finance.CashBankMovement{collection}, 5000},
		{"paid twice", []finance.CashBankMovement{collection, collection}, 10000},
		{"already refunded", []finance.CashBankMovement{collection, refund}, 0},
		{"partially refunded", []finance.CashBankMovement{collection,
			{Type: finance.CashMovementTypeSaleRefund, Amount: -2000}}, 3000},
		// A movement of another type sharing this DocumentID says nothing about the sale.
		{"unrelated type ignored", []finance.CashBankMovement{collection, unrelated}, 5000},
		// Refunded more than collected: the sale is owed nothing, and a negative refund would
		// mean charging the client on the way out.
		{"over-refunded clamps", []finance.CashBankMovement{collection,
			{Type: finance.CashMovementTypeSaleRefund, Amount: -8000}}, 0},
	}

	for _, check := range checks {
		if got := netCashRefundAmount(check.movements); got != check.expected {
			t.Fatalf("%s: expected refund %v, got %v", check.name, check.expected, got)
		}
	}
}

// The refund may be paid from a different register than the one that collected, so the sum has to
// ignore CashBankID. Grouping by it would leave an unsettled collection on one bank and an
// unexplained outflow on the other, and the retry would refund a second time.
func TestNetCashRefundAmountIgnoresCashBank(t *testing.T) {
	movements := []finance.CashBankMovement{
		{CashBankID: 4, Type: finance.CashMovementTypeSaleCollection, Amount: 5000},
		{CashBankID: 9, Type: finance.CashMovementTypeSaleRefund, Amount: -5000},
	}
	if got := netCashRefundAmount(movements); got != 0 {
		t.Fatalf("expected a refund from another cash bank to settle the sale, got %v", got)
	}
}

// Deliveries are negative on the ledger; the return is the negation of what is still out.
func TestNetStockReturnsReversesWhatIsStillDelivered(t *testing.T) {
	delivered := []logistics.WarehouseProductMovement{
		{Type: 8, WarehouseID: 3, ProductID: 77, PresentationID: 2, SubDivisor: 6,
			Quantity: -4, SubQuantity: -2},
	}

	returns := netStockReturns(delivered, 5150)
	if len(returns) != 1 {
		t.Fatalf("expected one return movement, got %v", len(returns))
	}
	stockReturn := returns[0]
	if stockReturn.Quantity != 4 || stockReturn.SubQuantity != 2 {
		t.Fatalf("expected to return {4,2}, got {%v,%v}", stockReturn.Quantity, stockReturn.SubQuantity)
	}
	if stockReturn.WarehouseID != 3 || stockReturn.ProductID != 77 || stockReturn.PresentationID != 2 {
		t.Fatalf("expected the bucket the goods left from, got %+v", stockReturn)
	}
	if stockReturn.SubDivisor != 6 {
		t.Fatalf("expected the divisor the movement was written in, got %v", stockReturn.SubDivisor)
	}
	if stockReturn.Type != warehouseMovementTypeSaleAnnulment {
		t.Fatalf("expected the annulment type, got %v", stockReturn.Type)
	}
	// Without the document id the next annulment could not find this row and would return the
	// stock a second time.
	if stockReturn.DocumentID != 5150 {
		t.Fatalf("expected the reversal to carry the sale id, got %v", stockReturn.DocumentID)
	}
}

func TestNetStockReturnsSkipsWhatWasNeverDeliveredOrAlreadyReturned(t *testing.T) {
	if returns := netStockReturns(nil, 1); len(returns) != 0 {
		t.Fatalf("expected no returns for a sale that never shipped, got %v", len(returns))
	}

	// Already annulled once: delivery and reversal cancel, so a retry moves nothing.
	settled := []logistics.WarehouseProductMovement{
		{Type: 8, WarehouseID: 3, ProductID: 77, SubDivisor: 6, Quantity: -4, SubQuantity: -2},
		{Type: 9, WarehouseID: 3, ProductID: 77, SubDivisor: 6, Quantity: 4, SubQuantity: 2},
	}
	if returns := netStockReturns(settled, 1); len(returns) != 0 {
		t.Fatalf("expected an already-returned sale to move nothing, got %v", len(returns))
	}

	// Add carries nothing, so a partial return sits at {0,-4}: still four sub-units owed, and a
	// plain Units check would call that settled.
	partial := []logistics.WarehouseProductMovement{
		{Type: 8, WarehouseID: 3, ProductID: 77, SubDivisor: 6, Quantity: -1, SubQuantity: 0},
		{Type: 9, WarehouseID: 3, ProductID: 77, SubDivisor: 6, Quantity: 1, SubQuantity: -4},
	}
	returns := netStockReturns(partial, 1)
	if len(returns) != 1 || returns[0].Quantity != 0 || returns[0].SubQuantity != 4 {
		t.Fatalf("expected a partial return of {0,4}, got %+v", returns)
	}
}

// Each lot and serial is its own physical place: returning them merged would put stock back
// where it never was.
func TestNetStockReturnsKeepsLotsAndSerialsApart(t *testing.T) {
	delivered := []logistics.WarehouseProductMovement{
		{Type: 8, WarehouseID: 3, ProductID: 77, LotID: 11, SubDivisor: 1, Quantity: -2},
		{Type: 8, WarehouseID: 3, ProductID: 77, LotID: 12, SubDivisor: 1, Quantity: -5},
		{Type: 8, WarehouseID: 3, ProductID: 77, SerialNumber: "SN-1", SubDivisor: 1, Quantity: -1},
	}

	returns := netStockReturns(delivered, 1)
	if len(returns) != 3 {
		t.Fatalf("expected one return per physical bucket, got %v", len(returns))
	}
	if returns[0].LotID != 11 || returns[0].Quantity != 2 {
		t.Fatalf("expected lot 11 to get 2 back, got %+v", returns[0])
	}
	if returns[1].LotID != 12 || returns[1].Quantity != 5 {
		t.Fatalf("expected lot 12 to get 5 back, got %+v", returns[1])
	}
	if returns[2].SerialNumber != "SN-1" || returns[2].Quantity != 1 {
		t.Fatalf("expected serial SN-1 to get 1 back, got %+v", returns[2])
	}
}

// A row written before the product had a sub-unit carries divisor 0, which means the same as 1.
// Left unnormalized they would be two buckets, and the net would never settle.
func TestNetStockReturnsTreatsDivisorZeroAsOne(t *testing.T) {
	movements := []logistics.WarehouseProductMovement{
		{Type: 8, WarehouseID: 3, ProductID: 77, SubDivisor: 0, Quantity: -2},
		{Type: 9, WarehouseID: 3, ProductID: 77, SubDivisor: 1, Quantity: 2},
	}
	if returns := netStockReturns(movements, 1); len(returns) != 0 {
		t.Fatalf("expected divisor 0 and 1 to net against each other, got %+v", returns)
	}
}

// Whatever the summary added when the sale was created is exactly what the annulment has to take
// off, at every status the sale can be annulled from.
func TestAnnulmentSummaryChangeCancelsTheCreationChange(t *testing.T) {
	sale := types.SaleOrder{
		CompanyID:         1,
		Date:              20000,
		DetailProductsIDs: []int32{77, 78},
		// Packed lines: 3002 is three units plus two sub-units at divisor 6; 2000 is two whole.
		DetailQuantities: []int32{3002, 2000},
		DetailPrices:      []int32{500, 1200},
		DetailSubPrices:   []int32{100, 0},
		DetailSubDivisor:  []int16{6, 1},
	}

	for _, status := range []int8{types.OrderStatusPending, types.OrderStatusPaid,
		types.OrderStatusDelivered, types.OrderStatusCompleted} {

		sale.Status = status
		actions := summaryActionsOfStatus(status)
		created := MakeSummaryChangeFromOSaleOrder(sale, actions...)
		annulled := negateSummaryChanges(MakeSummaryChangeFromOSaleOrder(sale, actions...))

		if len(created) == 0 || len(created) != len(annulled) {
			t.Fatalf("status %v: expected matching change sets, got %v and %v",
				status, len(created), len(annulled))
		}

		// MakeSummaryChangeFromOSaleOrder drains a map, so the two slices are not in the same
		// order and have to be paired by product.
		annulledByProduct := map[int32]ProductSummaryChange{}
		for _, annulledChange := range annulled {
			annulledByProduct[annulledChange.productID] = annulledChange
		}

		for index, createdChange := range created {
			annulledChange, found := annulledByProduct[createdChange.productID]
			if !found {
				t.Fatalf("status %v: product %v is missing from the annulment",
					status, createdChange.productID)
			}
			sumOf := func(created, annulled int32) int32 { return created + annulled }
			if sumOf(createdChange.Quantity, annulledChange.Quantity) != 0 ||
				sumOf(int32(createdChange.SubQuantity), int32(annulledChange.SubQuantity)) != 0 ||
				sumOf(createdChange.QuantityPendingDelivery, annulledChange.QuantityPendingDelivery) != 0 ||
				sumOf(int32(createdChange.SubQuantityPendingDelivery), int32(annulledChange.SubQuantityPendingDelivery)) != 0 ||
				sumOf(createdChange.TotalAmount, annulledChange.TotalAmount) != 0 ||
				sumOf(createdChange.TotalDebtAmount, annulledChange.TotalDebtAmount) != 0 {
				t.Fatalf("status %v: change %v does not cancel: %+v vs %+v",
					status, index, createdChange, annulledChange)
			}
			// The divisor is the unit the counters are expressed in, not a counter: negating it
			// would make the change unmergeable with the stored row.
			if annulledChange.SubDivisor != createdChange.SubDivisor {
				t.Fatalf("status %v: the divisor must survive negation, got %v",
					status, annulledChange.SubDivisor)
			}
		}
	}
}

// Status is the only surviving record of what happened to a sale, so the annulment reads the
// actions back out of it.
func TestSummaryActionsOfStatus(t *testing.T) {
	checks := map[int8][]int8{
		types.OrderStatusPending:   {1},
		types.OrderStatusPaid:      {1, 2},
		types.OrderStatusDelivered: {1, 3},
		types.OrderStatusCompleted: {1, 2, 3},
	}

	for status, expected := range checks {
		actions := summaryActionsOfStatus(status)
		if len(actions) != len(expected) {
			t.Fatalf("status %v: expected actions %v, got %v", status, expected, actions)
		}
		for index, expectedAction := range expected {
			if actions[index] != expectedAction {
				t.Fatalf("status %v: expected actions %v, got %v", status, expected, actions)
			}
		}
	}
}
