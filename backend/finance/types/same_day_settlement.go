// Same-day settlement: the payment that alone pays a whole document on the document's own
// accounting date. The accounting operations (GET.accounting-operations) never download such a
// payment: its movement carries SameDayMovement = 1, which keeps it out of the delta log they read,
// and the document carries the movement's cash bank in its SameDayCashBankID, from which they infer
// the entry. Most sales are paid at the till the moment they are made, so this is most of the cash.
//
// A payment is decided when it is written: a document's first payment is its same-day settlement
// when it pays the whole total on the accounting date (IsSameDaySettlement). A flagged payment
// leaves no debt, so no later payment can follow it and unflag it. What can unflag it is a change
// to the document — its date, its total, an annulment's refund — and that path re-applies the rule
// over the stored payments (RestampDocumentPayments).

package types

import (
	"app/core"
	"app/db"
	"slices"
)

// SameDaySettlement returns the payment that alone settled a document of documentTotal on its
// accountingDate, or nil. payments are every payment of the document, refunds included: a second
// movement of any kind means no single one settled it.
func SameDaySettlement(accountingDate int16, documentTotal int32, payments []CashBankMovement) *CashBankMovement {
	if len(payments) != 1 || documentTotal <= 0 {
		return nil
	}
	payment := &payments[0]
	// A collection is positive and a payment negative: the size of the movement is what settles.
	paidAmount := core.If(payment.Amount < 0, -payment.Amount, payment.Amount)
	if payment.Date != accountingDate || paidAmount != documentTotal {
		return nil
	}
	return payment
}

// IsSameDaySettlement decides a payment being written: paidBefore is what the document was paid
// before it, so 0 means it is the first and only payment.
func IsSameDaySettlement(accountingDate int16, documentTotal, paidBefore int32, paymentDate int16, paymentAmount int32) bool {
	if paidBefore != 0 {
		return false
	}
	return SameDaySettlement(accountingDate, documentTotal,
		[]CashBankMovement{{Date: paymentDate, Amount: paymentAmount}}) != nil
}

// RestampDocumentPayments re-applies SameDaySettlement and the counter-account to every stored
// payment of one document, after a change to the document that can break either: its accounting
// date, its total, its comprobante (counterAccountCode) or an annulment's refund. paymentTypes are
// the movement types that pay this kind of document, since DocumentID alone is shared between the
// registers. It writes only the movements that changed and returns the cash bank of the same-day
// payment, 0 when none: the caller stores it in the document's SameDayCashBankID.
func RestampDocumentPayments(
	companyID int32, documentID int64, paymentTypes []CashMovementType,
	accountingDate int16, documentTotal int32, counterAccountCode int32,
) (int32, error) {
	stored := []CashBankMovement{}
	query := db.Query(&stored)
	query.Select().CompanyID.Equals(companyID).DocumentID.Equals(documentID)
	if err := query.Exec(); err != nil {
		return 0, core.Err("Error al obtener los pagos del documento:", documentID, err)
	}
	payments := []CashBankMovement{}
	for _, movement := range stored {
		if slices.Contains(paymentTypes, movement.Type) {
			payments = append(payments, movement)
		}
	}

	sameDayCashBankID := int32(0)
	settledPayment := SameDaySettlement(accountingDate, documentTotal, payments)
	if settledPayment != nil {
		sameDayCashBankID = settledPayment.CashBankID
	}

	changed := []CashBankMovement{}
	for _, payment := range payments {
		sameDayMovement := int8(core.If(settledPayment != nil && settledPayment.ID == payment.ID, 1, 0))
		period := core.PeriodOfDay(payment.Date)
		if payment.SameDayMovement == sameDayMovement && payment.CounterAccountCode == counterAccountCode &&
			payment.Period == period {
			continue
		}
		payment.SameDayMovement = sameDayMovement
		payment.CounterAccountCode = counterAccountCode
		payment.Period = period
		changed = append(changed, payment)
	}
	if len(changed) == 0 {
		return sameDayCashBankID, nil
	}
	// The period delta index packs Period, SameDayMovement and Type: they are written together.
	table := db.TableOf[CashBankMovement]()
	if err := db.Update(&changed, table.Period, table.SameDayMovement, table.Type, table.CounterAccountCode); err != nil {
		return 0, core.Err("Error al actualizar los pagos del documento:", documentID, err)
	}
	return sameDayCashBankID, nil
}
