package types

import (
	"app/core"
	logistics "app/logistics/types"
	sales "app/sales/types"
	"fmt"
)

// The limits of a table order's round.
const (
	MaxRoundLines     = 50
	MaxLineNoteLength = 120
	// readyUndoWindowTicks is how long the kitchen can take back a "ready" tap, in SUnixTime ticks
	// (2 s each): 2 minutes.
	readyUndoWindowTicks = 60
)

// AddRoundLines appends a round to the order: one more Round number, LineIDs after the highest,
// every line ordered now (status 2), or waiting for a waiter when it came from a QR whose layout
// requires confirmation (status 1). The lines arrive priced and routed.
func AddRoundLines(order *TableOrder, roundLines []TableOrderLine, needsConfirmation bool, nowTime int32, userID int32) {
	lastLineID, lastRound := int16(0), int8(0)
	for _, line := range order.Lines {
		lastLineID, lastRound = max(lastLineID, line.LineID), max(lastRound, line.Round)
	}
	for _, line := range roundLines {
		lastLineID++
		line.LineID, line.Round, line.UpdatedBy = lastLineID, lastRound+1, userID
		line.Status = LineStatusOrdered
		line.OrderedAt = nowTime
		if needsConfirmation {
			line.Status, line.OrderedAt = LineStatusToConfirm, 0
		}
		order.Lines = append(order.Lines, line)
	}
}

var lineStatusNames = map[int8]string{
	LineStatusCancelled: "anulado", LineStatusToConfirm: "por confirmar", LineStatusOrdered: "en cocina",
	LineStatusReady: "listo", LineStatusServed: "servido",
}

// ApplyLineTransition moves one line to targetStatus (§2.1), stamping when it happened:
//
//	1 → 2 confirm a QR line · 2 → 3 ready · 3 → 4 served · 1/2 → 0 cancel (not once ready)
//	3 → 2 take back a "ready" tap, within 2 minutes of it
func ApplyLineTransition(line *TableOrderLine, targetStatus int8, nowTime int32, userID int32) error {
	switch {
	case line.Status == LineStatusToConfirm && targetStatus == LineStatusOrdered:
		line.OrderedAt = nowTime
	case line.Status == LineStatusOrdered && targetStatus == LineStatusReady:
		line.ReadyAt = nowTime
	case line.Status == LineStatusReady && targetStatus == LineStatusServed:
		line.ServedAt = nowTime
	case (line.Status == LineStatusToConfirm || line.Status == LineStatusOrdered) && targetStatus == LineStatusCancelled:
	case line.Status == LineStatusReady && targetStatus == LineStatusOrdered:
		if nowTime-line.ReadyAt > readyUndoWindowTicks {
			return core.Err("Un plato marcado listo solo se puede devolver a cocina en los 2 minutos siguientes.")
		}
		line.ReadyAt = 0
	default:
		return core.Err(fmt.Sprintf(`Un producto "%s" no puede pasar a "%s".`, lineStatusNames[line.Status], lineStatusNames[targetStatus]))
	}
	line.Status, line.UpdatedBy = targetStatus, userID
	return nil
}

// TableOrderReservationChanges is what the lines hold on warehouseID's stock, times sign (§2.2):
// +1 when a round is added, -1 when a line is cancelled. A cancelled line holds nothing. Counted
// on the Store counter by the till's rule (a line with sub-units holds one more whole unit), so
// the till, the web store and the restaurant read one figure for the warehouse.
func TableOrderReservationChanges(lines []TableOrderLine, warehouseID int32, sign int32) []logistics.PendingDeliveryChange {
	saleLines := make([]sales.SaleOrderLine, 0, len(lines))
	for _, line := range lines {
		if line.Status != LineStatusCancelled {
			saleLines = append(saleLines, sales.SaleOrderLine{ProductID: line.ProductID, PresentationID: line.PresentationID, Quantity: line.Quantity})
		}
	}
	return sales.PendingDeliveryChanges(saleLines, warehouseID, false, sign)
}

// ChargeSaleLines are the sale lines of a bill being charged (§4): its served lines, at the prices
// they were ordered at. Every live line must be served or cancelled first, so a dish still in the
// kitchen is never billed nor left out of the bill.
func ChargeSaleLines(lines []TableOrderLine) ([]sales.SaleOrderLine, error) {
	saleLines := []sales.SaleOrderLine{}
	for _, line := range lines {
		switch line.Status {
		case LineStatusCancelled:
			continue
		case LineStatusServed:
			saleLines = append(saleLines, sales.SaleOrderLine{ProductID: line.ProductID, PresentationID: line.PresentationID,
				Quantity: line.Quantity, SubDivisor: line.SubDivisor, Price: line.Price, SubPrice: line.SubPrice})
		default:
			return nil, core.Err("Hay productos sin servir: márquelos servidos o anúlelos antes de cobrar la mesa.")
		}
	}
	if len(saleLines) == 0 {
		return nil, core.Err("La mesa no tiene productos servidos que cobrar.")
	}
	return saleLines, nil
}

// ValidateRoundLine checks one requested line before it is priced: a product, a positive packed
// quantity and a short note.
func ValidateRoundLine(productID int32, quantity int32, note string) error {
	if productID <= 0 || quantity <= 0 {
		return core.Err("Hay un producto con una cantidad inválida.")
	}
	if len([]rune(note)) > MaxLineNoteLength {
		return core.Err(fmt.Sprintf("La nota de un pedido admite hasta %v caracteres.", MaxLineNoteLength))
	}
	return nil
}
