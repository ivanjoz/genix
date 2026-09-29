package types

import (
	"app/core"
	"fmt"
)

// StockMovementType is WarehouseProductMovement.Type: what happened to the stock and which
// document did it. Each value is written by exactly one code path, so the table DocumentID points
// into is a function of the type and is never stored (docs/ACCOUNTING_MODEL_PLAN.md §4.6). The
// SUNAT Tabla 12 code the Kardex prints is derived from it too, on the accounting side.
//
// The ids are persisted; never renumber them. A new type is added only for a new writer, a new
// source document or a new operational meaning — never for an accounting reason alone.
type StockMovementType int8

const (
	StockMovementTypeManualEntry           StockMovementType = 1  // stock adjustment, upward
	StockMovementTypeManualExit            StockMovementType = 2  // stock adjustment, downward
	StockMovementTypePurchaseReception     StockMovementType = 3  // purchase-order reception
	StockMovementTypeSuppliesPurchase      StockMovementType = 4  // inventory expense (supplies)
	StockMovementTypeAssetAcquisition      StockMovementType = 5  // a fixed asset enters the warehouse
	StockMovementTypeAssetDisposal         StockMovementType = 6  // a fixed asset is written off
	StockMovementTypeAssetTransfer         StockMovementType = 7  // a fixed asset changes warehouse (two rows)
	StockMovementTypeSaleDelivery          StockMovementType = 8  // a sale's goods leave the warehouse
	StockMovementTypeSaleAnnulmentReturn   StockMovementType = 9  // an annulled sale's goods come back
	StockMovementTypeAssetSerialCorrection StockMovementType = 10 // an asset's units move to a corrected serial
	// 11 is reserved for the warehouse-to-warehouse stock transfer; it is added with its screen.
	StockMovementTypeOpeningStock        StockMovementType = 12 // the stock a company starts with, always costed
	StockMovementTypeInternalConsumption StockMovementType = 13 // stock used up by the business itself
	StockMovementTypeShrinkage           StockMovementType = 14 // merma: stock lost or damaged
)

// stockMovementTypeRule is what the engine enforces for a type: the table its DocumentID points
// into ("" = none, so DocumentID must be 0) and the direction its delta must have.
type stockMovementTypeRule struct {
	name          string
	documentTable string
	sign          int8 // 1 inflow only, -1 outflow only, 0 either
}

var stockMovementTypeRules = map[StockMovementType]stockMovementTypeRule{
	StockMovementTypeManualEntry:           {"Entrada manual", "", 1},
	StockMovementTypeManualExit:            {"Salida manual", "", -1},
	StockMovementTypePurchaseReception:     {"Ingreso por orden de compra", "purchase_order", 1},
	StockMovementTypeSuppliesPurchase:      {"Compra de insumos", "expenses", 1},
	StockMovementTypeAssetAcquisition:      {"Alta de activo", "accounting_asset", 1},
	StockMovementTypeAssetDisposal:         {"Baja de activo", "accounting_asset", -1},
	StockMovementTypeAssetTransfer:         {"Traslado de activo", "accounting_asset", 0},
	StockMovementTypeSaleDelivery:          {"Entrega (venta)", "sale_order", -1},
	StockMovementTypeSaleAnnulmentReturn:   {"Reingreso (anulación de venta)", "sale_order", 1},
	StockMovementTypeAssetSerialCorrection: {"Corrección de serie de activo", "accounting_asset", 0},
	StockMovementTypeOpeningStock:          {"Saldo inicial", "", 1},
	StockMovementTypeInternalConsumption:   {"Consumo interno", "", -1},
	StockMovementTypeShrinkage:             {"Merma", "", -1},
}

// StockMovementDocumentTable is the table a movement of this type names in DocumentID, or ""
// when it names none.
func StockMovementDocumentTable(movementType StockMovementType) string {
	return stockMovementTypeRules[movementType].documentTable
}

// validateStockMovementDocument refuses an unknown type, and a DocumentID that contradicts it: a
// type that names a document needs one, a type that names none must not carry one.
func validateStockMovementDocument(movementType StockMovementType, documentID int64) error {
	rule, known := stockMovementTypeRules[movementType]
	if !known {
		return core.Err(fmt.Sprintf("Tipo de movimiento de stock desconocido: %v.", movementType))
	}
	if rule.documentTable != "" && documentID <= 0 {
		return core.Err(fmt.Sprintf("El movimiento \"%s\" necesita el documento que lo origina.", rule.name))
	}
	if rule.documentTable == "" && documentID != 0 {
		return core.Err(fmt.Sprintf("El movimiento \"%s\" no lleva documento, se envió: %v.", rule.name, documentID))
	}
	return nil
}

// validateStockMovementDirection refuses a delta that goes the wrong way for its type. It runs on
// the delta, which for a "set stock to X" movement only the engine knows.
func validateStockMovementDirection(movementType StockMovementType, isInflow bool, productID int32) error {
	rule := stockMovementTypeRules[movementType]
	if rule.sign == 0 || isInflow == (rule.sign > 0) {
		return nil
	}
	return core.Err(fmt.Sprintf("El producto %v: un movimiento \"%s\" no puede %s el stock.",
		productID, rule.name, core.If(isInflow, "aumentar", "disminuir")))
}
