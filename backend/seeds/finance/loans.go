package finance

import (
	"app/core"
	crmTypes "app/crm/types"
	financeModule "app/finance"
	financeTypes "app/finance/types"
	"app/seeds/base"
	"app/seeds/runner"
	"math"
	"time"
)

// The sample bank loan (plan/CASH_FLOW_FORECAST_PLAN.md §8): S/ 150,000 from BCP at a TEA of 18 %,
// 24 monthly French installments, disbursed into the bank before the seeded window so the window
// already finds installments paid and one falling due every month.
const (
	sampleLoanName             = "Préstamo capital de trabajo BCP"
	sampleLoanPrincipalCents   = int32(15_000_000)
	sampleLoanAnnualRate       = int32(180_000) // TEA 18 % × 10000
	sampleLoanInstallmentCount = 24
	sampleLoanDaysBeforeWindow = 120
	sampleLenderRUC            = "20100047218"
)

// prepareSampleLoan makes sure the sample loan exists and that every installment due before the
// window is paid, each on its due date. Both checks read what is stored, so a resumed run repeats
// nothing. Returns the loan the day step pays from.
func prepareSampleLoan(run *runner.Run) (financeTypes.Loan, error) {
	lenders, err := base.SaveAndLoadParties(run, []runner.PartySeed{{
		Name: "Banco de Crédito del Perú", PersonType: 2, RegistryNumber: sampleLenderRUC, CountryID: 604, CityID: "150101",
	}}, crmTypes.ClientProviderTypeProvider)
	if err != nil {
		return financeTypes.Loan{}, core.Err("No se pudo sembrar el banco que presta:", err)
	}

	storedLoans := []financeTypes.Loan{}
	if err := runner.CallAs(run.UserToken, "GET.loans", financeModule.GetLoans, nil, nil, &storedLoans); err != nil {
		return financeTypes.Loan{}, err
	}
	loan := financeTypes.Loan{}
	for _, storedLoan := range storedLoans {
		if storedLoan.Name == sampleLoanName && storedLoan.Status > 0 {
			loan = storedLoan
		}
	}

	if loan.ID == 0 {
		bank, err := financeTypes.GetCaja(run.UserToken.CompanyID, run.BankCashBankID)
		if err != nil {
			return loan, err
		}
		disbursementDate := run.FirstDay() - sampleLoanDaysBeforeWindow
		newLoan := financeTypes.Loan{
			Name: sampleLoanName, LenderID: lenders[0].ID, CashBankID: bank.ID, CurrencyType: bank.CurrencyType,
			Principal: sampleLoanPrincipalCents, DisbursementDate: disbursementDate, AnnualRate: sampleLoanAnnualRate,
			Method: financeTypes.LoanMethodFrench, Status: financeTypes.LoanStatusActive, RegisterDisbursement: true,
			Installments: frenchSchedule(run, sampleLoanPrincipalCents, sampleLoanAnnualRate, disbursementDate, sampleLoanInstallmentCount),
		}
		if err := runner.CallAs(run.UserToken, "POST.loans", financeModule.PostLoans, nil, newLoan, &loan); err != nil {
			return loan, core.Err("No se pudo registrar el préstamo de ejemplo:", err)
		}
	}

	for _, installment := range loan.Installments {
		if installment.DueDate >= run.FirstDay() {
			break
		}
		if loan, err = paySampleLoanInstallment(run, loan, installment); err != nil {
			return loan, err
		}
	}
	return loan, nil
}

// paySampleLoanInstallment pays what an installment still owes from the loan's own account, on its
// due date. A paid installment is left alone.
func paySampleLoanInstallment(run *runner.Run, loan financeTypes.Loan, installment financeTypes.LoanInstallment) (financeTypes.Loan, error) {
	pendingAmount := installment.Total() - installment.PaidAmount
	if pendingAmount <= 0 {
		return loan, nil
	}
	// The handler refuses to leave the account negative: top it up first, as the other payment steps do.
	if err := run.EnsureCashBalance(loan.CashBankID, pendingAmount, core.Now().Add(-time.Second)); err != nil {
		return loan, err
	}
	payment := financeModule.LoanPaymentBody{
		LoanID: loan.ID, InstallmentNumber: installment.Number, CashBankID: loan.CashBankID,
		Amount: pendingAmount, Date: installment.DueDate,
	}
	paidLoan := financeTypes.Loan{}
	if err := runner.CallAs(run.UserToken, "POST.loan-payment", financeModule.PostLoanPayment, nil, payment, &paidLoan); err != nil {
		return loan, core.Err("No se pudo pagar la cuota", installment.Number, "del préstamo:", err)
	}
	return paidLoan, nil
}

// runLoanPaymentsStep pays the sample loan's installments that fall due on the simulated day.
func runLoanPaymentsStep(run *runner.Run, loan *financeTypes.Loan, dayStart time.Time) error {
	today := core.TimeToFechaUnix(dayStart)
	for _, installment := range loan.Installments {
		if installment.DueDate != today {
			continue
		}
		paidLoan, err := paySampleLoanInstallment(run, *loan, installment)
		if err != nil {
			return err
		}
		*loan = paidLoan
	}
	return nil
}

// frenchSchedule is the lender's constant-installment schedule: one installment a calendar month
// after the disbursement, the TEA turned into its equivalent monthly rate, and the last installment
// absorbing the rounding so the principals add up to the loan.
func frenchSchedule(run *runner.Run, principal, annualRate int32, disbursementDate int16, count int) []financeTypes.LoanInstallment {
	monthlyRate := math.Pow(1+float64(annualRate)/1_000_000, 1.0/12) - 1
	installmentAmount := float64(principal) * monthlyRate / (1 - math.Pow(1+monthlyRate, -float64(count)))
	disbursementTime := run.DayStartTime(disbursementDate)

	schedule := make([]financeTypes.LoanInstallment, 0, count)
	outstanding := principal
	for number := 1; number <= count; number++ {
		interest := int32(math.Round(float64(outstanding) * monthlyRate))
		principalPart := int32(math.Round(installmentAmount)) - interest
		if number == count {
			principalPart = outstanding
		}
		outstanding -= principalPart
		schedule = append(schedule, financeTypes.LoanInstallment{
			Number: int16(number), DueDate: core.TimeToFechaUnix(disbursementTime.AddDate(0, number, 0)),
			Principal: principalPart, Interest: interest,
		})
	}
	return schedule
}
