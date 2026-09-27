package types

import "testing"

func TestResolveCashMovementAccount(t *testing.T) {
	cases := []struct {
		name         string
		movementType CashMovementType
		amount       int32
		accountCode  int16
		wantCode     int16
		wantErr      bool
	}{
		{"withdrawal with a valid choice", CashMovementTypeWithdrawal, -500, PCGEShareholdersPayable, PCGEShareholdersPayable, false},
		{"withdrawal without a choice", CashMovementTypeWithdrawal, -500, 0, 0, true},
		{"loss with an account of another type", CashMovementTypeLoss, -500, PCGEOtherOperatingIncome, 0, true},
		{"count shortage charged to the cashier", CashMovementTypePhysicalCount, -300, PCGEOtherStaffReceivables, PCGEOtherStaffReceivables, false},
		{"count surplus resolves its only option", CashMovementTypePhysicalCount, 300, 0, PCGEOtherOperatingIncome, false},
		{"count surplus refuses a shortage account", CashMovementTypePhysicalCount, 300, PCGEOtherOperatingExpenses, 0, true},
		{"count without difference stores nothing", CashMovementTypePhysicalCount, 0, 0, 0, false},
		{"unclassified type stores nothing", CashMovementTypeSaleCollection, 900, 0, 0, false},
		{"unclassified type refuses an account", CashMovementTypeSaleCollection, 900, PCGEOtherOperatingIncome, 0, true},
	}
	for _, testCase := range cases {
		gotCode, err := ResolveCashMovementAccount(testCase.movementType, testCase.amount, testCase.accountCode)
		if (err != nil) != testCase.wantErr {
			t.Errorf("%s: error = %v, want error %v", testCase.name, err, testCase.wantErr)
		}
		if gotCode != testCase.wantCode {
			t.Errorf("%s: account = %d, want %d", testCase.name, gotCode, testCase.wantCode)
		}
	}
}
