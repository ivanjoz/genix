package types

import "testing"

func TestBuildCashTransferMovements(t *testing.T) {
	till := CashBank{ID: 1, CurrencyType: 1}
	bank := CashBank{ID: 2, CurrencyType: 1}
	dollarBank := CashBank{ID: 3, CurrencyType: 2}

	movements, err := BuildCashTransferMovements(till, bank, -30000, 70000)
	if err != nil {
		t.Fatalf("valid transfer: %v", err)
	}
	if len(movements) != 2 {
		t.Fatalf("want 2 movements, got %d", len(movements))
	}
	outflow, inflow := movements[0], movements[1]
	if outflow.CashBankID != 1 || outflow.CashBankRefID != 2 || outflow.Amount != -30000 || outflow.FinalAmount != 70000 {
		t.Errorf("source row = %+v", outflow)
	}
	// The destination's FinalAmount stays 0 so ApplyCashBankMovement computes it from its balance.
	if inflow.CashBankID != 2 || inflow.CashBankRefID != 1 || inflow.Amount != 30000 || inflow.FinalAmount != 0 {
		t.Errorf("destination row = %+v", inflow)
	}
	if outflow.Type != CashMovementTypeTransfer || inflow.Type != CashMovementTypeTransfer {
		t.Errorf("both rows must be transfers: %v, %v", outflow.Type, inflow.Type)
	}

	rejected := []struct {
		name        string
		destination CashBank
		amount      int32
	}{
		{"no destination", CashBank{}, -100},
		{"same register", till, -100},
		{"another currency", dollarBank, -100},
		{"an inflow on the source", bank, 100},
	}
	for _, testCase := range rejected {
		if _, err := BuildCashTransferMovements(till, testCase.destination, testCase.amount, 0); err == nil {
			t.Errorf("%s: expected an error", testCase.name)
		}
	}
}
