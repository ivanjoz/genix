// The PCGE counter-account of every cash movement, stored on CashBankMovement.CounterAccountCode.
//
// A withdrawal, a loss and a physical count say nothing of why the money moved: the user picks the
// account when the movement is created. The payment of a document (a sale collection or refund, a
// supplier, expense or asset payment) is stamped by the document's own process with the account
// the document leaves open (DocumentPaymentAccounts), so the Libro Diario posts it without reading
// the document back.

package types

import (
	"app/core"
	"slices"
	"strconv"
)

// PCGE accounts the options below post against. The chart itself is accounting.PCGEAccounts;
// finance cannot import it (accounting/types imports finance/types), so the codes are repeated
// here and TestCashMovementAccountsAreInTheChart keeps each one a postable account of it.
const (
	PCGEShareholdersReceivable int32 = 142  // Accionistas (o socios) — cuentas por cobrar
	PCGEOtherStaffReceivables  int32 = 1419 // Otras cuentas por cobrar al personal
	PCGEShareholdersPayable    int32 = 441  // Accionistas (o socios) — cuentas por pagar (dividendos)
	PCGEOtherOperatingExpenses int32 = 659  // Otros gastos de gestión
	PCGEOtherOperatingIncome   int32 = 759  // Otros ingresos de gestión
	PCGECapital                int32 = 501  // Capital social — aporte de los socios
	PCGELoansReceived          int32 = 451  // Préstamos de instituciones financieras y otras entidades
	PCGEReceivable             int32 = 1212 // Facturas por cobrar — emitidas en cartera
	PCGESuppliers              int32 = 4212 // Facturas por pagar — emitidas
	PCGEOtherPayables          int32 = 4699 // Otras cuentas por pagar diversas
	PCGEAssetPayables          int32 = 4654 // Pasivos por compra de activo inmovilizado — PPE
	PCGEMerchandisePurchase    int32 = 6011 // Compras — mercaderías
	// Payroll (plan/HR_PAYROLL_PLAN.md §5.2).
	PCGEStaffLoans        int32 = 1411  // Préstamos al personal
	PCGEStaffAdvances     int32 = 1412  // Adelanto de remuneraciones
	PCGEEssaludPayable    int32 = 4031  // ESSALUD
	PCGEOnpPayable        int32 = 4032  // ONP
	PCGEIncomeTax5ta      int32 = 40173 // Renta de quinta categoría
	PCGESalariesPayable   int32 = 4111  // Sueldos y salarios por pagar
	PCGEAfpPayable        int32 = 417   // Administradoras de fondos de pensiones
	PCGEOtherStaffPayable int32 = 419   // Otras remuneraciones por pagar (retenciones judiciales)
)

// DocumentPaymentAccounts are the counter-accounts the payment of a document may carry: the
// receivable a sale leaves open, the payable a purchase or an asset does.
var DocumentPaymentAccounts = map[CashMovementType][]int32{
	CashMovementTypeSaleCollection:  {PCGEReceivable},
	CashMovementTypeSaleRefund:      {PCGEReceivable},
	CashMovementTypeSupplierPayment: {PCGESuppliers, PCGEOtherPayables},
	CashMovementTypeExpensePayment:  {PCGESuppliers, PCGEOtherPayables},
	CashMovementTypeAssetPayment:    {PCGEAssetPayables},
	CashMovementTypePayrollPayment:  {PCGESalariesPayable, PCGEStaffAdvances, PCGEStaffLoans},
	CashMovementTypePayrollContributions: {PCGEEssaludPayable, PCGEOnpPayable, PCGEIncomeTax5ta, PCGEAfpPayable,
		PCGEOtherStaffPayable},
}

// PurchasePayableAccount is the payable a purchase leaves open: 4212 when the supplier's comprobante
// is registered, 4699 when it is not. Its payments carry the same account.
func PurchasePayableAccount(docType int8) int32 {
	if docType != PurchaseDocTypeNone {
		return PCGESuppliers
	}
	return PCGEOtherPayables
}

// CashMovementAccountOption is one account a movement type may post against.
//
// Direction narrows an option to the movement's sign: -1 outflow, 1 inflow, 0 either. Only a
// physical count needs it, because a shortage and a surplus are the same type with opposite
// meaning.
type CashMovementAccountOption struct {
	MovementType CashMovementType `cb:"1"`
	Direction    int8             `cb:"2"`
	AccountCode  int32            `cb:"3"`
	Name         string           `cb:"4"`
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

// ResolveCashMovementAccount validates the account the client chose, or the document's process
// stamped, and returns the one to store. A document payment must carry one of its
// DocumentPaymentAccounts. A type with a single option needs no choice, so 0 resolves to it; a type
// with none stores 0 and refuses any account sent for it.
func ResolveCashMovementAccount(movementType CashMovementType, amount int32, accountCode int32) (int32, error) {
	if paymentAccounts, isDocumentPayment := DocumentPaymentAccounts[movementType]; isDocumentPayment {
		if !slices.Contains(paymentAccounts, accountCode) {
			return 0, core.Err("El pago de un documento debe llevar la cuenta que su documento deja abierta, se envió:", accountCode)
		}
		return accountCode, nil
	}
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
