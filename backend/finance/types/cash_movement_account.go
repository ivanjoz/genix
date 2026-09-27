// The PCGE counter-account of the cash movements a user classifies.
//
// A sale collection, a supplier payment or an expense payment says why the money moved, so the
// Libro Diario derives the other side of the entry from the type and its document. A withdrawal,
// a loss and a physical count say nothing of the kind: the user picks the account when the
// movement is created, and it is stored on CashBankMovement.AccountCode.

package types

import (
	"app/core"
	"strconv"
)

// PCGE accounts the options below post against. Only the ones this file needs: the full chart
// belongs to the Libro Diario.
const (
	PCGEShareholdersReceivable int16 = 142  // Accionistas (o socios) — cuentas por cobrar
	PCGEOtherStaffReceivables  int16 = 1419 // Otras cuentas por cobrar al personal
	PCGEShareholdersPayable    int16 = 441  // Accionistas (o socios) — cuentas por pagar (dividendos)
	PCGEOtherOperatingExpenses int16 = 659  // Otros gastos de gestión
	PCGEOtherOperatingIncome   int16 = 759  // Otros ingresos de gestión
	PCGECapital                int16 = 50   // Capital — aporte de los socios
	PCGELoansReceived          int16 = 451  // Préstamos de instituciones financieras y otras entidades
)

// CashMovementAccountOption is one account a movement type may post against.
//
// Direction narrows an option to the movement's sign: -1 outflow, 1 inflow, 0 either. Only a
// physical count needs it, because a shortage and a surplus are the same type with opposite
// meaning.
type CashMovementAccountOption struct {
	MovementType CashMovementType `json:",omitempty"`
	Direction    int8             `json:",omitempty"`
	AccountCode  int16            `json:",omitempty"`
	Name         string           `json:",omitempty"`
}

// CashMovementAccountOptions is what the backend validates against and — via the //CATALOG:
// marker in frontend/routes/finance/cash-banks/cajas.svelte.ts — what the forms offer. The
// first option of each type and direction is the default the backfill assigns.
var CashMovementAccountOptions = []CashMovementAccountOption{
	{MovementType: CashMovementTypeWithdrawal, AccountCode: PCGEShareholdersReceivable, Name: "Owner or shareholder withdrawal|Retiro del socio o accionista"},
	{MovementType: CashMovementTypeWithdrawal, AccountCode: PCGEShareholdersPayable, Name: "Dividend payment|Pago de dividendos"},
	{MovementType: CashMovementTypeLoss, AccountCode: PCGEOtherOperatingExpenses, Name: "Other expense|Otro gasto"},
	{MovementType: CashMovementTypeLoss, AccountCode: PCGEOtherStaffReceivables, Name: "Charged to an employee|Cargo a un trabajador"},
	{MovementType: CashMovementTypePhysicalCount, Direction: -1, AccountCode: PCGEOtherOperatingExpenses, Name: "Other expense|Otro gasto"},
	{MovementType: CashMovementTypePhysicalCount, Direction: -1, AccountCode: PCGEOtherStaffReceivables, Name: "Charged to the cashier|Cargo al cajero"},
	{MovementType: CashMovementTypePhysicalCount, Direction: 1, AccountCode: PCGEOtherOperatingIncome, Name: "Other income|Otro ingreso"},
	// A manual Cobro is money coming in that no sale explains; the account says from where.
	{MovementType: CashMovementTypeCollection, AccountCode: PCGEOtherOperatingIncome, Name: "Other income|Otro ingreso"},
	{MovementType: CashMovementTypeCollection, AccountCode: PCGEShareholdersReceivable, Name: "Owner or shareholder repayment|Devolución del socio o accionista"},
	{MovementType: CashMovementTypeCollection, AccountCode: PCGEOtherStaffReceivables, Name: "Repayment by an employee|Devolución de un trabajador"},
	{MovementType: CashMovementTypeCollection, AccountCode: PCGECapital, Name: "Capital contribution|Aporte de capital"},
	{MovementType: CashMovementTypeCollection, AccountCode: PCGELoansReceived, Name: "Loan received|Préstamo recibido"},
}

// ManualCashMovementTypes are the types a user registers by hand on a cash register, with the
// sign each must carry: -1 money out, 1 money in. Every other type is written by the document
// that explains it (a sale, a purchase order, an expense, an asset) or by a physical count; a
// manual one would move cash with nothing behind it.
var ManualCashMovementTypes = map[CashMovementType]int8{
	CashMovementTypeTransfer:   -1,
	CashMovementTypeWithdrawal: -1,
	CashMovementTypeLoss:       -1,
	CashMovementTypeCollection: 1,
}

// ValidateManualCashMovement refuses a type the user may not register by hand, and an amount
// whose sign contradicts the type.
func ValidateManualCashMovement(movementType CashMovementType, amount int32) error {
	requiredSign, isManual := ManualCashMovementTypes[movementType]
	if !isManual {
		return core.Err("Este tipo de movimiento no se registra manualmente, se registra desde su propio documento.")
	}
	if amount == 0 || (amount < 0) != (requiredSign < 0) {
		return core.Err(core.If(requiredSign < 0,
			"El monto de este movimiento debe ser negativo: es una salida de la caja.",
			"El monto de este movimiento debe ser positivo: es un ingreso a la caja."))
	}
	return nil
}

// CashMovementAccountOptionsFor lists the accounts a movement of this type and amount may post
// against. Empty for a type the user does not classify, and for a zero amount.
func CashMovementAccountOptionsFor(movementType CashMovementType, amount int32) []CashMovementAccountOption {
	if amount == 0 {
		return nil
	}
	direction := int8(1)
	if amount < 0 {
		direction = -1
	}
	matching := []CashMovementAccountOption{}
	for _, option := range CashMovementAccountOptions {
		if option.MovementType == movementType && (option.Direction == 0 || option.Direction == direction) {
			matching = append(matching, option)
		}
	}
	return matching
}

// ResolveCashMovementAccount validates the account the client chose and returns the one to
// store. A type with a single option needs no choice, so 0 resolves to it; a type with none
// stores 0 and refuses any account sent for it.
func ResolveCashMovementAccount(movementType CashMovementType, amount int32, accountCode int16) (int16, error) {
	options := CashMovementAccountOptionsFor(movementType, amount)
	if len(options) == 0 {
		if accountCode != 0 {
			return 0, core.Err("Este tipo de movimiento no lleva cuenta contable, se envió:", accountCode)
		}
		return 0, nil
	}
	if accountCode == 0 && len(options) == 1 {
		return options[0].AccountCode, nil
	}
	allowedCodes := make([]string, 0, len(options))
	for _, option := range options {
		if option.AccountCode == accountCode {
			return accountCode, nil
		}
		allowedCodes = append(allowedCodes, strconv.Itoa(int(option.AccountCode)))
	}
	return 0, core.Err("Seleccione la cuenta contable del movimiento. Cuentas permitidas:", allowedCodes)
}
