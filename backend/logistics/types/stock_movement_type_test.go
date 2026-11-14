package types

import (
	"app/core"
	"testing"
)

// The type names the source document, so a DocumentID that disagrees with it would make the
// ledger point at the wrong table — or at none.
func TestValidateStockMovementDocument(t *testing.T) {
	cases := []struct {
		name         string
		movementType StockMovementType
		documentID   int64
		wantErr      bool
	}{
		{"reception carries its order", StockMovementTypePurchaseReception, 55, false},
		{"reception without its order", StockMovementTypePurchaseReception, 0, true},
		{"sale delivery without its sale", StockMovementTypeSaleDelivery, 0, true},
		{"manual entry without a document", StockMovementTypeManualEntry, 0, false},
		{"opening stock refuses a document", StockMovementTypeOpeningStock, 9, true},
		{"a missing type is refused", 0, 0, true},
		{"the reserved transfer type is refused", 11, 0, true},
		{"a production consumption names its close", StockMovementTypeProductionConsumption, 3, false},
		{"a production entry without its close", StockMovementTypeProductionEntry, 0, true},
	}
	for _, testCase := range cases {
		err := validateStockMovementDocument(testCase.movementType, testCase.documentID)
		if (err != nil) != testCase.wantErr {
			t.Errorf("%s: error = %v, want error %v", testCase.name, err, testCase.wantErr)
		}
	}
}

// A "set stock to X" adjustment only learns its direction inside the engine; a reason that
// contradicts it — a merma that adds stock — is refused rather than recorded.
func TestValidateStockMovementDirection(t *testing.T) {
	cases := []struct {
		name         string
		movementType StockMovementType
		isInflow     bool
		wantErr      bool
	}{
		{"manual entry adds", StockMovementTypeManualEntry, true, false},
		{"manual entry cannot remove", StockMovementTypeManualEntry, false, true},
		{"shrinkage removes", StockMovementTypeShrinkage, false, false},
		{"shrinkage cannot add", StockMovementTypeShrinkage, true, true},
		{"consumption cannot add", StockMovementTypeInternalConsumption, true, true},
		{"opening stock cannot remove", StockMovementTypeOpeningStock, false, true},
		{"an asset transfer goes both ways", StockMovementTypeAssetTransfer, false, false},
		{"an asset transfer goes both ways, inflow", StockMovementTypeAssetTransfer, true, false},
		{"a production consumption removes", StockMovementTypeProductionConsumption, false, false},
		{"a production entry cannot remove", StockMovementTypeProductionEntry, false, true},
	}
	for _, testCase := range cases {
		err := validateStockMovementDirection(testCase.movementType, testCase.isInflow, 7)
		if (err != nil) != testCase.wantErr {
			t.Errorf("%s: error = %v, want error %v", testCase.name, err, testCase.wantErr)
		}
	}
}

// Only the production close dates its movements, and only today or yesterday: a later close of day D
// still books on D, but never in a day already past.
func TestValidateStockMovementDate(t *testing.T) {
	today := core.FechaUnix()
	cases := []struct {
		name         string
		movementType StockMovementType
		date         int16
		wantErr      bool
	}{
		{"no date is today", StockMovementTypeManualEntry, 0, false},
		{"a production entry dated yesterday", StockMovementTypeProductionEntry, today - 1, false},
		{"a close's sale exit dated today", StockMovementTypeSaleDelivery, today, false},
		{"two days back is refused", StockMovementTypeProductionConsumption, today - 2, true},
		{"tomorrow is refused", StockMovementTypeProductionEntry, today + 1, true},
		{"a manual entry cannot be dated", StockMovementTypeManualEntry, today - 1, true},
	}
	for _, testCase := range cases {
		err := validateStockMovementDate(testCase.movementType, testCase.date)
		if (err != nil) != testCase.wantErr {
			t.Errorf("%s: error = %v, want error %v", testCase.name, err, testCase.wantErr)
		}
	}
}
