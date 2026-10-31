// The PCGE chart of accounts: the accounts every movement is classified into, so its journal
// entry can always balance. The list is in pcge_accounts.go; this file holds what the rest of
// the system asks of it.
//
// The chart is fixed in code, not configurable per company. It is a curated subset of the
// official one: every 2- and 3-digit account, plus the subaccounts the ledger rules and an SME's
// usual postings need. Adding an account means adding its row in code order. Its parent must
// already be in the list, and the parent stops being Postable.

package types

import (
	"app/core"
	"strconv"
	"strings"
)

// PCGEAccount is one account of the chart.
//
// Postable marks a leaf: an account a line may be posted to. A parent only groups its children,
// and PLE 5.2 requires the most detailed level in use, so a parent is never posted to. A
// 2-digit account is never a leaf, because PLE 5.4 asks for at least 3 digits.
//
// The pair codes let the ledger derive lines without asking the user, and are set only where
// they apply:
//   - StockCode and VariationCode, on a 60x purchase account: the stock account its reception
//     debits and the 61x it credits (periodic inventory: Dr 20 / Cr 61).
//   - AccumulatedDepreciationCode and DepreciationExpenseCode, on a 33x asset account: the 39x
//     credited and the 68x debited each month it depreciates.
type PCGEAccount struct {
	Code                        int32  `cb:"1"`
	Name                        string `cb:"2"`
	Postable                    bool   `cb:"3"`
	StockCode                   int32  `cb:"4"`
	VariationCode               int32  `cb:"5"`
	AccumulatedDepreciationCode int32  `cb:"6"`
	DepreciationExpenseCode     int32  `cb:"7"`
}

// PCGEAccountKind is what the user is classifying when they pick an account. Each kind may only
// use the accounts under its prefixes (PCGEKindPrefixes), so an expense can never land on a
// sales account. Sales and the derived lines (IGV, payables, receivables, cash) have no kind:
// the system decides those, and the user never picks them.
type PCGEAccountKind int8

const (
	// PCGEKindSupply is the default account of a supply, and the account of a purchase line or
	// inventory expense that buys one.
	PCGEKindSupply PCGEAccountKind = 1
	// PCGEKindExpense is a simple expense.
	PCGEKindExpense PCGEAccountKind = 2
	// PCGEKindAsset is a fixed asset's account.
	PCGEKindAsset PCGEAccountKind = 3
)

// PCGEKindPrefix lets a kind use every postable account whose code starts with Prefix.
type PCGEKindPrefix struct {
	Kind   PCGEAccountKind `cb:"1"`
	Prefix int32           `cb:"2"`
}

// PCGEKindPrefixes is what the backend validates against and — via the //CATALOG: marker in
// frontend/core/pcge.ts — what the pickers filter by.
var PCGEKindPrefixes = []PCGEKindPrefix{
	// A supply is bought into stock (602 materias primas, 603 auxiliares/suministros/repuestos,
	// 604 envases y embalajes), consumed on purchase (656 suministros), or is equipment (33).
	{Kind: PCGEKindSupply, Prefix: 602},
	{Kind: PCGEKindSupply, Prefix: 603},
	{Kind: PCGEKindSupply, Prefix: 604},
	{Kind: PCGEKindSupply, Prefix: 656},
	{Kind: PCGEKindSupply, Prefix: 33},
	// An expense is a cost of the period: staff (62), services (63), taxes (64), other management
	// costs (65, except 655, the net cost of a disposal, which the system derives), and financial
	// costs (67). 40 is included because paying IGV or income tax is not an expense; it settles
	// the tax account.
	{Kind: PCGEKindExpense, Prefix: 62},
	{Kind: PCGEKindExpense, Prefix: 63},
	{Kind: PCGEKindExpense, Prefix: 64},
	{Kind: PCGEKindExpense, Prefix: 651},
	{Kind: PCGEKindExpense, Prefix: 652},
	{Kind: PCGEKindExpense, Prefix: 653},
	{Kind: PCGEKindExpense, Prefix: 654},
	{Kind: PCGEKindExpense, Prefix: 656},
	{Kind: PCGEKindExpense, Prefix: 658},
	{Kind: PCGEKindExpense, Prefix: 659},
	{Kind: PCGEKindExpense, Prefix: 67},
	{Kind: PCGEKindExpense, Prefix: 40},
	// Right-of-use assets (32), property, plant and equipment (33), intangibles (34).
	{Kind: PCGEKindAsset, Prefix: 32},
	{Kind: PCGEKindAsset, Prefix: 33},
	{Kind: PCGEKindAsset, Prefix: 34},
}

var pcgeAccountsByCode = func() map[int32]PCGEAccount {
	accountsByCode := make(map[int32]PCGEAccount, len(PCGEAccounts))
	for _, account := range PCGEAccounts {
		accountsByCode[account.Code] = account
	}
	return accountsByCode
}()

// FindPCGEAccount returns the account with this code, and whether the chart has it.
func FindPCGEAccount(code int32) (PCGEAccount, bool) {
	account, exists := pcgeAccountsByCode[code]
	return account, exists
}

// IsPCGEAccountAllowed reports whether a line of this kind may be posted to the account: the
// account is a leaf, and its code starts with one of the kind's prefixes. The comparison is on
// the digits: prefix 33 admits 3369, and prefix 659 does not admit 65.
func IsPCGEAccountAllowed(kind PCGEAccountKind, code int32) bool {
	account, exists := pcgeAccountsByCode[code]
	if !exists || !account.Postable {
		return false
	}
	codeDigits := strconv.Itoa(int(code))
	for _, kindPrefix := range PCGEKindPrefixes {
		if kindPrefix.Kind == kind && strings.HasPrefix(codeDigits, strconv.Itoa(int(kindPrefix.Prefix))) {
			return true
		}
	}
	return false
}

// Accounts the defaults below fall back to.
const (
	PCGEOtherEquipment int32 = 3369 // Otros equipos — an asset nobody classified more precisely
	PCGESupplies       int32 = 6032 // Suministros — a supply bought into stock
)

// DefaultSupplyAccountCode is the account a supply gets when its form names none: equipment when
// it depreciates (it becomes an asset), otherwise suministros bought into stock.
func DefaultSupplyAccountCode(depreciationMonths int16) int32 {
	if depreciationMonths > 0 {
		return PCGEOtherEquipment
	}
	return PCGESupplies
}

// ResolveAssetAccount returns the account to store on an asset: the one picked on the form, else
// the supply's own account when it is an asset account, else generic equipment.
func ResolveAssetAccount(chosenCode, supplyAccountCode int32) (int32, error) {
	if chosenCode != 0 {
		return chosenCode, ValidatePCGEAccount(PCGEKindAsset, chosenCode)
	}
	if IsPCGEAccountAllowed(PCGEKindAsset, supplyAccountCode) {
		return supplyAccountCode, nil
	}
	return PCGEOtherEquipment, nil
}

// ResolveSupplyAccount returns the account a purchase of a supply posts to: the one picked on the
// line, else the supply's default. Either way it must be a supply account.
func ResolveSupplyAccount(chosenCode, supplyAccountCode int32) (int32, error) {
	accountCode := chosenCode
	if accountCode == 0 {
		accountCode = supplyAccountCode
	}
	return accountCode, ValidatePCGEAccount(PCGEKindSupply, accountCode)
}

// ValidatePCGEAccount refuses an account the kind may not use. 0 passes: it means "not
// classified yet", which the consolidation page reports instead of blocking the save.
func ValidatePCGEAccount(kind PCGEAccountKind, code int32) error {
	if code == 0 || IsPCGEAccountAllowed(kind, code) {
		return nil
	}
	if _, exists := pcgeAccountsByCode[code]; !exists {
		return core.Err("La cuenta contable no existe en el plan de cuentas:", code)
	}
	return core.Err("La cuenta contable no corresponde a este tipo de operación, o no es una cuenta de detalle:", code)
}
