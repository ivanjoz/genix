package types

import "app/core"

// BuildCashTransferMovements returns the two ledger rows of a transfer between registers: the
// outflow on the source and the matching inflow on the destination, each pointing at the other
// through CashBankRefID.
//
// Two rows rather than one row read from both columns: each register's balance, history and
// running FinalAmount then come from its own movements, like every other type. The Libro Diario
// posts the transfer once, from the source row (the negative one).
//
// amount is the source's signed amount and must be an outflow. sourceFinalAmount is the balance
// the client computed for the source; the destination's is left for ApplyCashBankMovement to
// compute.
func BuildCashTransferMovements(
	source, destination CashBank, amount, sourceFinalAmount int32,
) ([]InternalCashMovement, error) {

	if destination.ID == 0 {
		return nil, core.Err("Las transferencias necesitan especificar una caja de destino.")
	}
	if amount >= 0 {
		return nil, core.Err("El monto de una transferencia debe ser una salida (negativo) de la caja de origen.")
	}
	if destination.ID == source.ID {
		return nil, core.Err("La caja de destino debe ser distinta a la caja de origen.")
	}
	// A cross-currency move is a currency exchange: it needs a rate the movement cannot carry.
	if destination.CurrencyType != source.CurrencyType {
		return nil, core.Err("La caja de destino debe tener la misma moneda que la caja de origen.")
	}

	return []InternalCashMovement{
		{
			CashBankID:    source.ID,
			CashBankRefID: destination.ID,
			Type:          CashMovementTypeTransfer,
			Amount:        amount,
			FinalAmount:   sourceFinalAmount,
		},
		{
			CashBankID:    destination.ID,
			CashBankRefID: source.ID,
			Type:          CashMovementTypeTransfer,
			Amount:        -amount,
		},
	}, nil
}
