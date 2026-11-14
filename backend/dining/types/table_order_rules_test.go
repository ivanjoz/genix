package types

import (
	"app/core"
	"testing"
)

func TestAddRoundLinesNumbersRoundsAndLines(t *testing.T) {
	order := TableOrder{Lines: []TableOrderLine{{LineID: 1, Round: 1, Status: LineStatusServed}, {LineID: 4, Round: 1, Status: LineStatusCancelled}}}
	AddRoundLines(&order, []TableOrderLine{{ProductID: 7}, {ProductID: 8}}, false, 1000, 3)
	waiterRound := order.Lines[2:]
	if waiterRound[0].LineID != 5 || waiterRound[1].LineID != 6 || waiterRound[0].Round != 2 || waiterRound[1].Round != 2 {
		t.Fatalf("ids/round: %+v", waiterRound)
	}
	if waiterRound[0].Status != LineStatusOrdered || waiterRound[0].OrderedAt != 1000 || waiterRound[0].UpdatedBy != 3 {
		t.Fatalf("a waiter round goes straight to the kitchen: %+v", waiterRound[0])
	}
	AddRoundLines(&order, []TableOrderLine{{ProductID: 9}}, true, 1010, 0)
	if qrLine := order.Lines[4]; qrLine.Round != 3 || qrLine.Status != LineStatusToConfirm || qrLine.OrderedAt != 0 {
		t.Fatalf("a QR round waiting for confirmation: %+v", qrLine)
	}
}

func TestApplyLineTransition(t *testing.T) {
	cases := []struct {
		name        string
		from, to    int8
		readyAt     int32
		now         int32
		wantAllowed bool
	}{
		{"confirm", LineStatusToConfirm, LineStatusOrdered, 0, 100, true},
		{"ready", LineStatusOrdered, LineStatusReady, 0, 100, true},
		{"served", LineStatusReady, LineStatusServed, 90, 100, true},
		{"cancel ordered", LineStatusOrdered, LineStatusCancelled, 0, 100, true},
		{"cancel to confirm", LineStatusToConfirm, LineStatusCancelled, 0, 100, true},
		{"cancel once ready", LineStatusReady, LineStatusCancelled, 90, 100, false},
		{"undo ready in time", LineStatusReady, LineStatusOrdered, 50, 110, true},
		{"undo ready too late", LineStatusReady, LineStatusOrdered, 50, 111, false},
		{"skip to served", LineStatusOrdered, LineStatusServed, 0, 100, false},
		{"unserve", LineStatusServed, LineStatusReady, 0, 100, false},
	}
	for _, testCase := range cases {
		line := TableOrderLine{LineID: 1, Status: testCase.from, ReadyAt: testCase.readyAt}
		err := ApplyLineTransition(&line, testCase.to, testCase.now, 5)
		if (err == nil) != testCase.wantAllowed {
			t.Errorf("%s: allowed %v, got err %v", testCase.name, testCase.wantAllowed, err)
			continue
		}
		if err == nil && (line.Status != testCase.to || line.UpdatedBy != 5) {
			t.Errorf("%s: status %v updatedBy %v", testCase.name, line.Status, line.UpdatedBy)
		}
	}

	line := TableOrderLine{Status: LineStatusOrdered}
	_ = ApplyLineTransition(&line, LineStatusReady, 200, 1)
	_ = ApplyLineTransition(&line, LineStatusServed, 230, 1)
	if line.ReadyAt != 200 || line.ServedAt != 230 {
		t.Fatalf("stamps: %+v", line)
	}
}

func TestChargeSaleLines(t *testing.T) {
	served := TableOrderLine{ProductID: 7, PresentationID: 2, Quantity: 1001, SubDivisor: 6, Price: 1500, SubPrice: 300, Status: LineStatusServed}
	saleLines, err := ChargeSaleLines([]TableOrderLine{served, {ProductID: 8, Status: LineStatusCancelled}})
	if err != nil || len(saleLines) != 1 {
		t.Fatalf("a cancelled line is left out: %+v %v", saleLines, err)
	}
	if line := saleLines[0]; line.ProductID != 7 || line.PresentationID != 2 || line.Quantity != 1001 ||
		line.SubDivisor != 6 || line.Price != 1500 || line.SubPrice != 300 {
		t.Fatalf("the line keeps the price it was ordered at: %+v", line)
	}
	for _, liveStatus := range []int8{LineStatusToConfirm, LineStatusOrdered, LineStatusReady} {
		if _, err := ChargeSaleLines([]TableOrderLine{served, {ProductID: 9, Status: liveStatus}}); err == nil {
			t.Errorf("a line in status %v must block the charge", liveStatus)
		}
	}
	if _, err := ChargeSaleLines([]TableOrderLine{{ProductID: 8, Status: LineStatusCancelled}}); err == nil {
		t.Error("a bill with nothing served cannot be charged")
	}
}

func TestTableOrderReservationChangesHoldWholeUnits(t *testing.T) {
	lines := []TableOrderLine{
		{ProductID: 7, Quantity: 2 * core.QuantityLineScale, Status: LineStatusOrdered},
		{ProductID: 8, PresentationID: 3, Quantity: 1*core.QuantityLineScale + 2, Status: LineStatusServed},
		{ProductID: 9, Quantity: 5 * core.QuantityLineScale, Status: LineStatusCancelled},
	}
	changes := TableOrderReservationChanges(lines, 4, -1)
	if len(changes) != 2 || changes[0].Store != -2 || changes[1].Store != -2 || changes[1].PresentationID != 3 || changes[0].WarehouseID != 4 || changes[0].Online != 0 {
		t.Fatalf("changes: %+v", changes)
	}
}
