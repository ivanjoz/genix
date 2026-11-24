package types

import (
	"app/core"
	hr "app/hr/types"
	"strconv"
	"strings"
)

// The cash flow forecast's deterministic layers (plan/CASH_FLOW_FORECAST_PLAN.md §2, §3, §12):
// what the company already owes or is owed (Known) and what its rules will generate (Scheduled).
// GET.cash-flow-commitments returns them as CashFlowCommitment rows with the date each one is
// expected to move cash, plus the warnings describing what that timing had to assume.

// Streams: where a commitment comes from. Streams 1–4 and 10–12 are computed by the browser from the
// cash-documents frame; the IDs stay reserved here so the backend never reuses them.
const (
	CashFlowStreamReceivable          int8 = 1  // unpaid sale
	CashFlowStreamPurchaseOrder       int8 = 2  // confirmed purchase order with debt
	CashFlowStreamExpense             int8 = 3  // pending expense
	CashFlowStreamAsset               int8 = 4  // unpaid asset purchase
	CashFlowStreamExpenseSchedule     int8 = 5  // a future period of a recurring bill
	CashFlowStreamPayrollNetPay       int8 = 6  // what the workers are paid on payday
	CashFlowStreamPayrollCts          int8 = 7  // CTS deposit (May, November)
	CashFlowStreamPayrollContribution int8 = 8  // EsSalud, ONP, 5ta, AFP, judicial withholdings
	CashFlowStreamLoanInstallment     int8 = 9  // what an installment still owes
	CashFlowStreamTaxIgv              int8 = 10 // IGV of a closed period
	CashFlowStreamTaxIncome           int8 = 11 // renta prepayment of a closed period
	CashFlowStreamTaxNrus             int8 = 12 // Nuevo RUS cuota of a closed period
)

// Layers: Known is recorded already; Scheduled is what a rule (schedule, payroll, loan) will record.
const (
	CashFlowLayerKnown     int8 = 1
	CashFlowLayerScheduled int8 = 2
)

// CashFlowCommitment is one future movement of cash.
type CashFlowCommitment struct {
	Stream int8
	Layer  int8
	// SiteID is the site the flow belongs to; 0 is the company lane (taxes, payroll, loans).
	SiteID int16
	// SourceID is the record behind it: sale, order, expense, asset, schedule, run period, loan.
	SourceID int64
	Label    string
	// Date is when the cash is expected to move: the due date, or today when that has passed.
	Date int16
	// DueDate is the record's own due date, 0 when it has none.
	DueDate int16
	// Amount is signed cents of CurrencyType: + money in, − money out.
	Amount       int64
	CurrencyType int8
}

// DocumentCashDate is when an open document's money is expected: its due date, or its own date when
// it has none; a date already past moves to today. isOverdue reports that move.
func DocumentCashDate(dueDate, documentDate, today int16) (cashDate int16, isOverdue bool) {
	cashDate = core.If(dueDate > 0, dueDate, documentDate)
	if cashDate < today {
		return today, true
	}
	return cashDate, false
}

// PaydayOf is the company's payday event of the period, or the period's last day when it has none
// (found = false, which raises payday-missing).
func PaydayOf(period int16, paydays []int16) (payday int16, found bool) {
	for _, day := range paydays {
		if core.PeriodOfDay(day) == period {
			return day, true
		}
	}
	return core.PeriodLastDay(period), false
}

// NthBusinessDay is the nth Monday-to-Friday of the period that is not a holiday. The AFP
// contributions fall due within the first five business days of the following month.
func NthBusinessDay(period int16, nth int, holidays map[int16]bool) int16 {
	day := core.PeriodFirstDay(period)
	for count := 0; ; day++ {
		// UnixDay 0 was a Thursday: (day + 4) % 7 is 0 on Sunday and 6 on Saturday.
		weekday := (int(day) + 4) % 7
		if weekday != 0 && weekday != 6 && !holidays[day] {
			count++
			if count == nth {
				return day
			}
		}
	}
}

// PayrollCash is the cash a month's payslips move: the net pay on payday, the CTS deposit, and what
// the deductions and employer contributions owe each payable account.
type PayrollCash struct {
	NetPay        int64
	Cts           int64
	Contributions map[int32]int64 // by payable account
}

// PayrollCashOf sums a month's payslips. Only the payroll contribution payables count: a deduction
// that settles an advance or a staff loan moves no cash on its own.
func PayrollCashOf(payslips []hr.Payslip) PayrollCash {
	contributionAccounts := DocumentPaymentAccounts[CashMovementTypePayrollContributions]
	cash := PayrollCash{Contributions: map[int32]int64{}}
	for _, payslip := range payslips {
		cash.NetPay += int64(payslip.NetPay)
		for _, line := range payslip.Lines {
			if line.OutsideNetPay {
				cash.Cts += int64(line.Amount)
			}
			isContribution := line.Kind == hr.ConceptKindDeduction || line.Kind == hr.ConceptKindEmployer
			if isContribution && core.Contains(contributionAccounts, line.PayableAccountCode) {
				cash.Contributions[line.PayableAccountCode] += int64(line.Amount)
			}
		}
	}
	return cash
}

// TaxRules are the company's tax setup, already resolved from its regime.
type TaxRules struct {
	ChargesIgv bool
	// IncomeTaxRate is the monthly renta prepayment × 10000 (1.5 % → 150); 0 for none.
	IncomeTaxRate int32
	// NrusQuota is the Nuevo RUS fixed monthly cuota in cents; 0 outside Nuevo RUS.
	NrusQuota int64
}

// CashFlowTaxOutlook is what the browser needs to compute the taxes from the cash-documents frame
// (plan/CASH_DOCUMENTS_FRAME_PLAN.md §6): the company's rules and the periods to compute.
type CashFlowTaxOutlook struct {
	Rules TaxRules
	// Periods are the last closed periods, then the open one and the following ones whose due date
	// falls in the window, in period order.
	Periods []CashFlowTaxPeriod
}

// CashFlowTaxPeriod is a tax period's SUNAT due date and what the 401x expenses dated in its due month
// already paid, in local currency cents.
type CashFlowTaxPeriod struct {
	Period          int16
	DueDate         int16
	PaidIgvLocal    int64
	PaidIncomeLocal int64
}

// CashFlowOldPayables are the open payables dated before the cash-documents window the browser reads,
// summed by currency (cents): local, and dollars of the documents in USD.
type CashFlowOldPayables struct {
	Local   int64
	Dollars int64
}

// Tax payable accounts the paid-tax detection reads: 4011x IGV, 4017x renta.
func IsIgvPayableAccount(accountCode int32) bool {
	return strings.HasPrefix(strconv.Itoa(int(accountCode)), "4011")
}

func IsIncomeTaxPayableAccount(accountCode int32) bool {
	return strings.HasPrefix(strconv.Itoa(int(accountCode)), "4017")
}

// AmountInPEN restates an amount at its document rate (× 1000); a dollar amount without a rate is
// taken as is, which is good enough for a warning's amount at stake.
func AmountInPEN(currencyType int8, exchangeRate int32, amount int64) int64 {
	if currencyType == CurrencyUSD && exchangeRate > 0 {
		return amount * int64(exchangeRate) / 1000
	}
	return amount
}

// CashFlowWarning is one gap the forecast had to assume around, with its offending records. The
// frontend's catalog gives each Code its severity, message and the page where it is fixed.
type CashFlowWarning struct {
	Code             string
	AmountAtStakePEN int64
	Records          []CashFlowWarningRecord // the first CashFlowWarningRecordLimit
	RecordCount      int32
}

type CashFlowWarningRecord struct {
	ID    int64
	Label string
}

const CashFlowWarningRecordLimit = 50

// CashFlowWarnings collects warnings by code, in the order each code first appears.
type CashFlowWarnings struct {
	warnings []CashFlowWarning
}

// Add counts one offending record under a code.
func (collector *CashFlowWarnings) Add(code string, amountPEN int64, recordID int64, label string) {
	index := -1
	for warningIndex := range collector.warnings {
		if collector.warnings[warningIndex].Code == code {
			index = warningIndex
		}
	}
	if index < 0 {
		collector.warnings = append(collector.warnings, CashFlowWarning{Code: code})
		index = len(collector.warnings) - 1
	}
	warning := &collector.warnings[index]
	warning.AmountAtStakePEN += max(amountPEN, -amountPEN)
	warning.RecordCount++
	if len(warning.Records) < CashFlowWarningRecordLimit {
		warning.Records = append(warning.Records, CashFlowWarningRecord{ID: recordID, Label: label})
	}
}

func (collector *CashFlowWarnings) List() []CashFlowWarning {
	return collector.warnings
}
