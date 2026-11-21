package types

import (
	"app/core"
	"app/db"
	"strings"
)

// Loan is money the company borrowed from a bank or another lender, with the installment schedule it
// repays (plan/CASH_FLOW_FORECAST_PLAN.md §8). The lender's official schedule always prevails, so the
// installments are stored as entered: the frontend calculator only proposes them.
type Loan struct {
	db.TableStruct[LoanTable, Loan]
	CompanyID int32  `cb:"1"`
	ID        int32  `cb:"2"`
	Name      string `cb:"3"`
	// LenderID is the bank or lender, a provider in client_provider.
	LenderID int32 `cb:"4"`
	// CashBankID is where the money arrives and the default account installments are paid from.
	CashBankID   int32 `cb:"5"`
	CurrencyType int8  `cb:"6"`
	// Principal is the amount borrowed, in cents.
	Principal        int32 `cb:"7"`
	DisbursementDate int16 `cb:"8"`
	// AnnualRate is the TEA × 10000 (12.5 % → 125000); 0 when not entered. Reference only: the
	// installments carry the interest the lender charges.
	AnnualRate int32 `cb:"9"`
	// Method is how the calculator built the schedule (LoanMethod*); reference only.
	Method       int8              `cb:"10"`
	GraceMonths  int8              `cb:"11"`
	Installments []LoanInstallment `cb:"12"`
	Notes        string            `cb:"13"`
	// IsDisbursementRegistered: the disbursement was written as a cash movement (type 14), so the
	// principal can no longer change. A loan entered mid-life had its money arrive before the ERP.
	IsDisbursementRegistered bool  `cb:"14"`
	Status                   int8  `json:"ss" cb:"ss,15"`
	Updated                  int32 `json:"upd" cb:"upd,16"`
	Version                  int16 `json:"ver" cb:"ver,17"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,18"`
	Created   int32   `cb:"19"`
	CreatedBy int32   `cb:"20"`
	UpdatedBy int32   `cb:"21"`
	// RegisterDisbursement is request-only: on creation, also write the disbursement into CashBankID.
	RegisterDisbursement bool `db:"-" cb:"22"`
}

// LoanInstallment is one scheduled payment. Amounts in cents of the loan's currency. PaidAmount is
// server-authoritative: recomputed from the loan's payments, never taken from the client.
type LoanInstallment struct {
	Number     int16 `cb:"1"`
	DueDate    int16 `cb:"2"`
	Principal  int32 `cb:"3"`
	Interest   int32 `cb:"4"`
	Fees       int32 `cb:"5"`
	PaidAmount int32 `cb:"6"`
}

type LoanTable struct {
	db.TableStruct[LoanTable, Loan]
	CompanyID                db.Col[*LoanTable, int32]
	ID                       db.Col[*LoanTable, int32]
	Name                     db.Col[*LoanTable, string]
	LenderID                 db.Col[*LoanTable, int32]
	CashBankID               db.Col[*LoanTable, int32]
	CurrencyType             db.Col[*LoanTable, int8]
	Principal                db.Col[*LoanTable, int32]
	DisbursementDate         db.Col[*LoanTable, int16]
	AnnualRate               db.Col[*LoanTable, int32]
	Method                   db.Col[*LoanTable, int8]
	GraceMonths              db.Col[*LoanTable, int8]
	Installments             db.Col[*LoanTable, []LoanInstallment]
	Notes                    db.Col[*LoanTable, string]
	IsDisbursementRegistered db.Col[*LoanTable, bool]
	Status                   db.Col[*LoanTable, int8]
	Updated                  db.Col[*LoanTable, int32]
	Version                  db.Col[*LoanTable, int16]
	Created                  db.Col[*LoanTable, int32]
	CreatedBy                db.Col[*LoanTable, int32]
	UpdatedBy                db.Col[*LoanTable, int32]
}

func (e LoanTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        130,
		Name:      "loans",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		Indexes: []db.Index{
			// Active and paid-off loans, delta-synced: the page lists both.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// Loan statuses.
const (
	LoanStatusActive  = int8(1)
	LoanStatusPaidOff = int8(2)
)

// How the calculator built a schedule.
const (
	LoanMethodFrench = int8(1) // constant installment
	LoanMethodGerman = int8(2) // constant principal
	LoanMethodBullet = int8(3) // interest only, principal at the end
)

// Total is what the installment asks for: principal, interest and fees.
func (e LoanInstallment) Total() int32 {
	return e.Principal + e.Interest + e.Fees
}

// paidParts splits what an installment has received the way a lender applies a payment: fees first,
// then interest, then principal.
func (e LoanInstallment) paidParts(paidAmount int32) (fees, interest, principal int32) {
	fees = min(paidAmount, e.Fees)
	interest = min(paidAmount-fees, e.Interest)
	principal = paidAmount - fees - interest
	return fees, interest, principal
}

// LoanPaymentSplits divides a payment of an installment into the accounts it clears, after what the
// installment already received: fees (671) first, then interest (673), then principal (451). The
// splits are negative, like the payment's cash movement. The payment may not exceed what is pending.
func LoanPaymentSplits(installment LoanInstallment, amount int32) ([]CashMovementAccountSplit, error) {
	if amount <= 0 {
		return nil, core.Err("El monto del pago debe ser mayor a 0.")
	}
	if installment.PaidAmount+amount > installment.Total() {
		return nil, core.Err("El pago supera lo pendiente de la cuota", installment.Number, ":",
			installment.Total()-installment.PaidAmount)
	}
	feesBefore, interestBefore, principalBefore := installment.paidParts(installment.PaidAmount)
	feesAfter, interestAfter, principalAfter := installment.paidParts(installment.PaidAmount + amount)

	splits := []CashMovementAccountSplit{}
	for _, part := range []CashMovementAccountSplit{
		{AccountCode: PCGELoanCharges, Amount: feesAfter - feesBefore},
		{AccountCode: PCGELoanInterest, Amount: interestAfter - interestBefore},
		{AccountCode: PCGELoansReceived, Amount: principalAfter - principalBefore},
	} {
		if part.Amount > 0 {
			splits = append(splits, CashMovementAccountSplit{AccountCode: part.AccountCode, Amount: -part.Amount})
		}
	}
	return splits, nil
}

// IsPaidOff: every installment received its total.
func (e *Loan) IsPaidOff() bool {
	for _, installment := range e.Installments {
		if installment.PaidAmount < installment.Total() {
			return false
		}
	}
	return len(e.Installments) > 0
}

// FindInstallment returns the installment with that number, or nil.
func (e *Loan) FindInstallment(number int16) *LoanInstallment {
	for index := range e.Installments {
		if e.Installments[index].Number == number {
			return &e.Installments[index]
		}
	}
	return nil
}

// ValidateLoan checks a loan about to be saved active, trimming its texts in place: the required
// fields, installments numbered 1..n with strictly increasing due dates after the disbursement, no
// negative amounts, principals adding up to the loan's, and no installment asking for less than it
// already received.
func ValidateLoan(loan *Loan) error {
	loan.Name, loan.Notes = strings.TrimSpace(loan.Name), strings.TrimSpace(loan.Notes)
	if loan.Name == "" {
		return core.Err("El préstamo debe tener un nombre.")
	}
	if loan.LenderID <= 0 || loan.CashBankID <= 0 {
		return core.Err("Seleccione la entidad que presta y la caja o banco del préstamo.")
	}
	if loan.Principal <= 0 || loan.DisbursementDate <= 0 {
		return core.Err("Indique el monto del préstamo y la fecha del desembolso.")
	}
	if loan.AnnualRate < 0 || loan.GraceMonths < 0 {
		return core.Err("La tasa y los meses de gracia no pueden ser negativos.")
	}
	if len(loan.Installments) == 0 {
		return core.Err("El préstamo debe tener su cronograma de cuotas.")
	}
	principalTotal := int32(0)
	previousDueDate := loan.DisbursementDate
	for index, installment := range loan.Installments {
		if installment.Number != int16(index+1) {
			return core.Err("Las cuotas deben numerarse del 1 al", len(loan.Installments), "en orden.")
		}
		if installment.DueDate <= previousDueDate {
			return core.Err("La cuota", installment.Number, "debe vencer después del desembolso y de la cuota anterior.")
		}
		if installment.Principal < 0 || installment.Interest < 0 || installment.Fees < 0 {
			return core.Err("La cuota", installment.Number, "tiene montos negativos.")
		}
		if installment.PaidAmount > installment.Total() {
			return core.Err("La cuota", installment.Number, "ya recibió", installment.PaidAmount,
				"y no puede pedir menos que eso.")
		}
		principalTotal += installment.Principal
		previousDueDate = installment.DueDate
	}
	if principalTotal != loan.Principal {
		return core.Err("El capital de las cuotas suma", principalTotal, "y el préstamo es de", loan.Principal)
	}
	return nil
}
