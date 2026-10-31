package types

import (
	finance "app/finance/types"
	"strconv"
	"testing"
)

// TestPCGEAccountsAreConsistent enforces the rules for adding an account (pcge.go): codes in
// order and unique, every parent present, Postable exactly on the leaves of 3+ digits, and every
// pair pointing at a postable account.
func TestPCGEAccountsAreConsistent(t *testing.T) {
	codesWithChildren := map[string]bool{}
	for _, account := range PCGEAccounts {
		digits := strconv.Itoa(int(account.Code))
		codesWithChildren[digits[:len(digits)-1]] = true
	}

	previousDigits := ""
	for _, account := range PCGEAccounts {
		digits := strconv.Itoa(int(account.Code))
		if digits <= previousDigits {
			t.Errorf("account %s is out of code order or repeated (after %s)", digits, previousDigits)
		}
		previousDigits = digits

		if _, parentExists := FindPCGEAccount(account.Code / 10); len(digits) > 2 && !parentExists {
			t.Errorf("account %s has no parent %d in the chart", digits, account.Code/10)
		}
		wantPostable := len(digits) >= 3 && !codesWithChildren[digits]
		if account.Postable != wantPostable {
			t.Errorf("account %s: Postable = %v, want %v", digits, account.Postable, wantPostable)
		}
		if account.Name == "" {
			t.Errorf("account %s has no name", digits)
		}
		for _, pairedCode := range []int32{account.StockCode, account.VariationCode,
			account.AccumulatedDepreciationCode, account.DepreciationExpenseCode} {
			if paired, exists := FindPCGEAccount(pairedCode); pairedCode != 0 && (!exists || !paired.Postable) {
				t.Errorf("account %s pairs with %d, which is not a postable account", digits, pairedCode)
			}
		}
	}
}

// TestPCGEKindPrefixesReachAccounts catches a prefix that matches nothing, e.g. a typo or an
// account removed from the chart.
func TestPCGEKindPrefixesReachAccounts(t *testing.T) {
	for _, kindPrefix := range PCGEKindPrefixes {
		prefixDigits := strconv.Itoa(int(kindPrefix.Prefix))
		reachesAccount := false
		for _, account := range PCGEAccounts {
			digits := strconv.Itoa(int(account.Code))
			if account.Postable && len(digits) >= len(prefixDigits) && digits[:len(prefixDigits)] == prefixDigits {
				reachesAccount = true
				break
			}
		}
		if !reachesAccount {
			t.Errorf("kind %d prefix %d reaches no postable account", kindPrefix.Kind, kindPrefix.Prefix)
		}
	}
}

func TestIsPCGEAccountAllowed(t *testing.T) {
	cases := []struct {
		name    string
		kind    PCGEAccountKind
		code    int32
		allowed bool
	}{
		{"supply bought into stock", PCGEKindSupply, 6032, true},
		{"supply consumed on purchase", PCGEKindSupply, 656, true},
		{"supply that is equipment", PCGEKindSupply, 3361, true},
		{"supply cannot be merchandise", PCGEKindSupply, 6011, false},
		{"expense on rent", PCGEKindExpense, 6352, true},
		{"expense settling IGV", PCGEKindExpense, 40111, true},
		{"expense cannot be the derived disposal cost", PCGEKindExpense, 6551, false},
		{"expense cannot be a parent account", PCGEKindExpense, 635, false},
		{"expense cannot be a sale", PCGEKindExpense, 7012, false},
		{"asset on generic equipment", PCGEKindAsset, 3369, true},
		{"asset cannot be accumulated depreciation", PCGEKindAsset, 39527, false},
		{"unknown code", PCGEKindExpense, 6999, false},
	}
	for _, testCase := range cases {
		if got := IsPCGEAccountAllowed(testCase.kind, testCase.code); got != testCase.allowed {
			t.Errorf("%s: allowed = %v, want %v", testCase.name, got, testCase.allowed)
		}
	}
	if ValidatePCGEAccount(PCGEKindExpense, 0) != nil {
		t.Error("0 (not classified yet) must pass validation")
	}
}

// TestExpenseCategoryDefaultsAreAllowed guards the defaults finance repeats because it cannot
// import the chart: each must be an account a simple expense may use.
func TestExpenseCategoryDefaultsAreAllowed(t *testing.T) {
	for _, category := range finance.ExpenseCategoryOptions {
		if category.DefaultAccountCode != 0 && !IsPCGEAccountAllowed(PCGEKindExpense, category.DefaultAccountCode) {
			t.Errorf("expense category %q defaults to %d, which an expense may not use",
				category.Label, category.DefaultAccountCode)
		}
	}
}

func TestDefaultAccountResolvers(t *testing.T) {
	if !IsPCGEAccountAllowed(PCGEKindSupply, DefaultSupplyAccountCode(0)) ||
		!IsPCGEAccountAllowed(PCGEKindSupply, DefaultSupplyAccountCode(36)) ||
		!IsPCGEAccountAllowed(PCGEKindAsset, DefaultSupplyAccountCode(36)) {
		t.Error("a supply default must be a supply account, and a depreciable one also an asset account")
	}

	assetCases := []struct {
		name               string
		chosen, supplyCode int32
		want               int32
		wantErr            bool
	}{
		{"the form's choice wins", 3361, 3369, 3361, false},
		{"the supply's 33x when none was chosen", 0, 3351, 3351, false},
		{"generic equipment when the supply is not an asset account", 0, 6032, PCGEOtherEquipment, false},
		{"a choice outside 32-34 is refused", 6032, 0, 6032, true},
	}
	for _, testCase := range assetCases {
		got, err := ResolveAssetAccount(testCase.chosen, testCase.supplyCode)
		if got != testCase.want || (err != nil) != testCase.wantErr {
			t.Errorf("asset %s: got %d (err %v), want %d (err %v)", testCase.name, got, err, testCase.want, testCase.wantErr)
		}
	}

	supplyCases := []struct {
		name               string
		chosen, supplyCode int32
		want               int32
		wantErr            bool
	}{
		{"the line's choice wins", 6041, 6032, 6041, false},
		{"the supply's default when the line has none", 0, 6032, 6032, false},
		{"merchandise is not a supply account", 6011, 0, 6011, true},
		{"a supply with no account leaves the line unclassified", 0, 0, 0, false},
	}
	for _, testCase := range supplyCases {
		got, err := ResolveSupplyAccount(testCase.chosen, testCase.supplyCode)
		if got != testCase.want || (err != nil) != testCase.wantErr {
			t.Errorf("supply %s: got %d (err %v), want %d (err %v)", testCase.name, got, err, testCase.want, testCase.wantErr)
		}
	}
}

// TestCashMovementAccountsAreInTheChart guards the codes finance repeats because it cannot
// import the chart.
func TestCashMovementAccountsAreInTheChart(t *testing.T) {
	for _, option := range finance.CashMovementAccountOptions {
		if account, exists := FindPCGEAccount(option.AccountCode); !exists || !account.Postable {
			t.Errorf("cash movement option %q posts to %d, which is not a postable account of the chart",
				option.Name, option.AccountCode)
		}
	}
}
