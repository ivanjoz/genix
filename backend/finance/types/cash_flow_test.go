package types

import (
	"app/core"
	hr "app/hr/types"
	"testing"
)

func TestDocumentCashDate(t *testing.T) {
	today := int16(20000)
	if date, overdue := DocumentCashDate(20010, 19990, today); date != 20010 || overdue {
		t.Fatalf("future due date: got %d %v", date, overdue)
	}
	if date, overdue := DocumentCashDate(0, 20005, today); date != 20005 || overdue {
		t.Fatalf("no due date falls back to the document date: got %d %v", date, overdue)
	}
	if date, overdue := DocumentCashDate(19995, 19990, today); date != today || !overdue {
		t.Fatalf("a past due date moves to today: got %d %v", date, overdue)
	}
}

func TestNthBusinessDayAndPayday(t *testing.T) {
	// December 2026 starts on a Tuesday; the 8th is a holiday (Inmaculada Concepción).
	period := int16(2612)
	first := core.PeriodFirstDay(period)
	holidays := map[int16]bool{first + 7: true}
	// Business days: 1 Tue, 2 Wed, 3 Thu, 4 Fri, 7 Mon → the 5th is the 7th.
	if day := NthBusinessDay(period, 5, holidays); day != first+6 {
		t.Fatalf("5th business day: got day %d of the month", day-first+1)
	}
	if day := NthBusinessDay(period, 6, holidays); day != first+8 {
		t.Fatalf("the holiday is skipped: got day %d of the month", day-first+1)
	}
	if payday, found := PaydayOf(period, []int16{first - 3, first + 27}); payday != first+27 || !found {
		t.Fatalf("payday event of the period: got %d %v", payday, found)
	}
	if payday, found := PaydayOf(period, nil); payday != core.PeriodLastDay(period) || found {
		t.Fatalf("no payday falls back to the last day: got %d %v", payday, found)
	}
}

func TestPayrollCashOf(t *testing.T) {
	payslips := []hr.Payslip{{NetPay: 250000, Lines: []hr.PayslipLine{
		{Kind: hr.ConceptKindEarning, Amount: 300000},
		{Kind: hr.ConceptKindEarning, Amount: 120000, OutsideNetPay: true},
		{Kind: hr.ConceptKindDeduction, Amount: 39000, PayableAccountCode: PCGEOnpPayable},
		{Kind: hr.ConceptKindDeduction, Amount: 11000, PayableAccountCode: PCGEStaffAdvances},
		{Kind: hr.ConceptKindEmployer, Amount: 27000, PayableAccountCode: PCGEEssaludPayable},
	}}}
	cash := PayrollCashOf(payslips)
	if cash.NetPay != 250000 || cash.Cts != 120000 {
		t.Fatalf("net pay / CTS: got %d %d", cash.NetPay, cash.Cts)
	}
	if cash.Contributions[PCGEOnpPayable] != 39000 || cash.Contributions[PCGEEssaludPayable] != 27000 || len(cash.Contributions) != 2 {
		t.Fatalf("an advance settled by a deduction moves no cash: got %v", cash.Contributions)
	}
}

func TestCashFlowWarnings(t *testing.T) {
	collector := CashFlowWarnings{}
	for index := range CashFlowWarningRecordLimit + 5 {
		collector.Add("payable-overdue", -100, int64(index), "")
	}
	collector.Add("loan-rate-missing", 0, 1, "Préstamo")
	warnings := collector.List()
	if len(warnings) != 2 || warnings[0].RecordCount != CashFlowWarningRecordLimit+5 ||
		len(warnings[0].Records) != CashFlowWarningRecordLimit || warnings[0].AmountAtStakePEN != 100*int64(CashFlowWarningRecordLimit+5) {
		t.Fatalf("warnings grouped by code, records capped: got %+v", warnings[0])
	}
}

func TestValidateCashFlowEntry(t *testing.T) {
	entry := CashFlowEntry{Name: " Dividendos ", Kind: CashFlowEntryDividend, Date: 20000, CurrencyType: CurrencyPEN, Amount: 5000}
	if err := ValidateCashFlowEntry(&entry); err == nil {
		t.Fatal("a dividend must be an outflow")
	}
	entry.Amount = -5000
	if err := ValidateCashFlowEntry(&entry); err != nil || entry.Name != "Dividendos" {
		t.Fatalf("valid entry rejected: %v", err)
	}
}
