package production

import (
	"app/core"
	"app/db"
	logistics "app/logistics/types"
	"app/production/types"
	"fmt"
	"maps"
	"slices"
)

// The shared receipt of a production (docs/MANUFACTURING_PLAN.md §6.2, steps 4–6): the supplies leave
// the supplies warehouse (type 15), each product is valued and enters the order's warehouse (type 16).
// A production order's receipt and the daily close both write through it.

// receiptConsumption is what a receipt takes of a supply from one stock bucket, and the product whose
// value it adds to.
type receiptConsumption struct {
	ProductID    int32
	SupplyID     int32
	LotID        int32
	SerialNumber string
	SubUnits     int32
	SubDivisor   int16
}

// receiptProduct is one product a receipt enters, and what its value is made of beyond its consumption.
type receiptProduct struct {
	ProductID int32
	Name      string
	// SubDivisor is the product's own sub-unit divisor, which its stock rows take.
	SubDivisor             int16
	UnitsByPresentation    map[int16]int32
	ScrappedByPresentation map[int16]int32
	CostSource             int8
	// StandardValue is the whole value of a CostSourceStandard product. The others are added to a
	// CostSourceRecipe product's consumed supplies.
	StandardValue    int64
	LaborValue       int64
	SubcontractValue int64
	OverheadValue    int64
	// TracksLots enters the units into the receipt's lot, OP<orderID>-<N> (docs/MANUFACTURING_PLAN.md
	// §10), which the engine creates with the product's ShelfLifeDays and QualityHoldOnEntry.
	TracksLots         bool
	ShelfLifeDays      int16
	QualityHoldOnEntry bool
}

type productionReceiptInput struct {
	OrderID             int32
	WarehouseID         int32
	SuppliesWarehouseID int32
	N                   int16
	// MovementDate is the date the stock movements take.
	MovementDate int16
	Consumptions []receiptConsumption
	Products     []receiptProduct
	// FollowingMovements are applied right after the entries, in the same engine call: the close's sale
	// exits then leave at the cost the entries set.
	FollowingMovements []logistics.InternalMovement
}

// applyProductionReceiptIn writes one receipt inside transaction and answers it, every line and
// consumption valued. A product's value is spread over its presentations by good units, the last
// taking the rounding remainder; scrapped units carry none, the good ones absorb the whole cost.
func applyProductionReceiptIn(transaction *db.Transaction, req *core.HandlerArgs, input productionReceiptInput) (types.ProductionReceipt, error) {
	receipt := types.ProductionReceipt{N: input.N, Date: input.MovementDate, CreatedBy: req.User.ID}

	// The supplies leave first: what the engine values their exits at is what the products cost. A
	// zero consumption would be dropped by the engine and shift the ledger rows, so it is never sent.
	consumptions := slices.DeleteFunc(slices.Clone(input.Consumptions), func(consumption receiptConsumption) bool {
		return consumption.SubUnits <= 0
	})
	consumptionMovements := make([]logistics.InternalMovement, 0, len(consumptions))
	for _, consumption := range consumptions {
		quantity := core.Quantity{
			Units: consumption.SubUnits / int32(consumption.SubDivisor), Sub: consumption.SubUnits % int32(consumption.SubDivisor),
		}.Negate()
		consumptionMovements = append(consumptionMovements, logistics.InternalMovement{
			WarehouseID: input.SuppliesWarehouseID, ProductID: consumption.SupplyID,
			LotID: consumption.LotID, SerialNumber: consumption.SerialNumber,
			Quantity: quantity.Units, SubQuantity: quantity.Sub, SubDivisor: consumption.SubDivisor,
			Type: logistics.StockMovementTypeProductionConsumption, DocumentID: int64(input.OrderID), Date: input.MovementDate,
		})
	}
	consumptionLedger, err := logistics.ApplyMovimientosWithLedgerIn(transaction, req, consumptionMovements)
	if err != nil {
		return receipt, core.Err("Error al consumir los insumos:", err)
	}
	consumedValueByProduct := map[int32]int64{}
	for index, ledgerRow := range consumptionLedger {
		consumption := consumptions[index]
		consumedValueByProduct[consumption.ProductID] += int64(ledgerRow.MonetaryValue)
		receipt.SuppliesValue += int64(ledgerRow.MonetaryValue)
		receipt.Consumptions = append(receipt.Consumptions, types.ProductionReceiptConsumption{
			SupplyID: consumption.SupplyID, LotID: consumption.LotID, SerialNumber: consumption.SerialNumber,
			SubUnits: consumption.SubUnits, SubDivisor: consumption.SubDivisor, Value: ledgerRow.MonetaryValue,
		})
	}

	entryMovements := []logistics.InternalMovement{}
	// The receipt line each entry movement fills, to record the lot the engine resolved for it.
	entryLineIndexes := []int{}
	// One lot per receipt, shared by every product and presentation it enters: the lot's key is
	// (date, supplier 0, name), so each product's detail rows reference the same lot.
	receiptLotName := fmt.Sprintf("OP%v-%v", input.OrderID, input.N)
	for _, product := range input.Products {
		value := product.StandardValue
		if product.CostSource == types.CostSourceRecipe {
			value = consumedValueByProduct[product.ProductID] + product.LaborValue + product.SubcontractValue + product.OverheadValue
			receipt.LaborValue += product.LaborValue
			receipt.SubcontractValue += product.SubcontractValue
			receipt.OverheadValue += product.OverheadValue
		}
		goodUnits := int32(0)
		for _, units := range product.UnitsByPresentation {
			goodUnits += units
		}
		// The stock engine refuses an uncosted inflow into a row with no average, and a value with no
		// good unit to carry it would vanish.
		if goodUnits > 0 && value <= 0 {
			return receipt, core.Err(fmt.Sprintf(`"%s" no tiene costo para su ingreso de producción.`, product.Name))
		}
		if goodUnits == 0 && value > 0 {
			return receipt, core.Err(fmt.Sprintf(`"%s" no tiene unidades buenas que absorban su costo de producción.`, product.Name))
		}

		presentationIDs := slices.Sorted(maps.Keys(product.UnitsByPresentation))
		for presentationID := range product.ScrappedByPresentation {
			if _, hasGoodUnits := product.UnitsByPresentation[presentationID]; !hasGoodUnits {
				presentationIDs = append(presentationIDs, presentationID)
			}
		}
		slices.Sort(presentationIDs)
		goodPresentationCount := 0
		for _, presentationID := range presentationIDs {
			if product.UnitsByPresentation[presentationID] > 0 {
				goodPresentationCount++
			}
		}
		valueLeft := value
		for _, presentationID := range presentationIDs {
			units := product.UnitsByPresentation[presentationID]
			receiptLine := types.ProductionReceiptLine{
				ProductID: product.ProductID, PresentationID: presentationID, Units: units,
				Scrapped: product.ScrappedByPresentation[presentationID],
			}
			if units > 0 {
				goodPresentationCount--
				lineValue := value * int64(units) / int64(goodUnits)
				if goodPresentationCount == 0 {
					lineValue = valueLeft
				}
				valueLeft -= lineValue
				receiptLine.Value = int32(lineValue)
				entryMovement := logistics.InternalMovement{
					WarehouseID: input.WarehouseID, ProductID: product.ProductID, PresentationID: presentationID,
					Quantity: units, SubDivisor: max(product.SubDivisor, core.QuantityDivisorNone),
					InflowValue: receiptLine.Value, Type: logistics.StockMovementTypeProductionEntry,
					DocumentID: int64(input.OrderID), Date: input.MovementDate,
				}
				if product.TracksLots {
					entryMovement.LotName = receiptLotName
					entryMovement.LotShelfLifeDays, entryMovement.LotQualityHold = product.ShelfLifeDays, product.QualityHoldOnEntry
				}
				entryMovements = append(entryMovements, entryMovement)
				entryLineIndexes = append(entryLineIndexes, len(receipt.Lines))
			}
			receipt.Lines = append(receipt.Lines, receiptLine)
		}
	}
	// Every entry carries units, so the engine drops none: its first ledger rows are the entries, in
	// order, each with the LotID it resolved.
	entryLedger, err := logistics.ApplyMovimientosWithLedgerIn(transaction, req, append(entryMovements, input.FollowingMovements...))
	if err != nil {
		return receipt, core.Err("Error al registrar el ingreso de producción:", err)
	}
	for entryIndex, lineIndex := range entryLineIndexes {
		receipt.Lines[lineIndex].LotID = entryLedger[entryIndex].LotID
	}
	return receipt, nil
}
