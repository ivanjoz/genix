package sample_records

import (
	accountingTypes "app/accounting/types"
	financeTypes "app/finance/types"
	"encoding/json"
	"testing"
	"time"
)

// TestExpenseTemplatesAreBookable checks the catalog against the rules the handlers enforce, so a
// bad template fails here instead of halfway through a 60-day run.
func TestExpenseTemplatesAreBookable(t *testing.T) {
	supplierSeeds := []partySeed{}
	if err := json.Unmarshal(expenseSupplierSeedJSON, &supplierSeeds); err != nil {
		t.Fatal(err)
	}
	seededRUCs := map[string]bool{}
	for _, seed := range supplierSeeds {
		seededRUCs[seed.RegistryNumber] = true
	}

	seenKeys := map[string]bool{}
	for _, template := range expenseTemplates {
		if seenKeys[template.key] {
			t.Errorf("%s: duplicated key", template.key)
		}
		seenKeys[template.key] = true

		// An inventory purchase takes the supply's own account, so it names none.
		if !template.isInventory && !accountingTypes.IsPCGEAccountAllowed(accountingTypes.PCGEKindExpense, template.accountCode) {
			t.Errorf("%s: account %d is not an allowed expense account", template.key, template.accountCode)
		}
		if template.docType != financeTypes.PurchaseDocTypeNone && !seededRUCs[template.supplierRUC] {
			t.Errorf("%s: a comprobante needs a seeded supplier, got %q", template.key, template.supplierRUC)
		}
		if template.cadence != cadenceMonthlyTax && !template.isInventory &&
			(template.minAmount <= 0 || template.maxAmount < template.minAmount) {
			t.Errorf("%s: invalid amount range %d..%d", template.key, template.minAmount, template.maxAmount)
		}

		// The comprobante must add up to the expense amount, or PostExpenses rejects it.
		document := makePurchaseDocument(template.docType, template.docSeries, 1, 20000, 123457)
		if template.docType != financeTypes.PurchaseDocTypeNone && document.Total() != 123457 {
			t.Errorf("%s: document total %d differs from the amount", template.key, document.Total())
		}
	}
}

func TestIsMonthDayClampsToTheMonthEnd(t *testing.T) {
	february28 := time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC)
	if !isMonthDay(february28, 31) || !isMonthDay(february28, 28) || isMonthDay(february28, 27) {
		t.Error("day 31 and day 28 must land on February 28, day 27 must not")
	}
}
