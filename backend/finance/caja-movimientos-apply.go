package finance

import (
	"app/core"
	"app/db"
	financeTypes "app/finance/types"
)

func ApplyCajaMovimientos(req *core.HandlerArgs, movimientos []financeTypes.InternalCashMovement) error {
	if len(movimientos) == 0 {
		return nil
	}

	nowTime := req.EffectiveSUnixTime()
	dateUnix := req.EffectiveFechaUnix()

	// Agrupar movimientos por CajaID para manejar saldos
	cajasIDs := core.SliceSet[int32]{}
	for _, m := range movimientos {
		cajasIDs.Add(m.CajaID)
	}

	cajasMap := make(map[int32]financeTypes.CashBank)
	for _, id := range cajasIDs.Values {
		cashBank, err := GetCaja(req.User.CompanyID, id)
		if err != nil {
			return core.Err("Error al obtener la cashBank:", id, err)
		}
		cajasMap[id] = cashBank
	}

	records := []financeTypes.CashBankMovement{}
	cajasToUpdate := []financeTypes.CashBank{}

	// Para rastrear el saldo actual si hay múltiples movimientos para la misma cashBank
	saldosActuales := make(map[int32]int32)
	for id, cashBank := range cajasMap {
		saldosActuales[id] = cashBank.SaldoCurrent
	}

	for _, m := range movimientos {
		cashBank := cajasMap[m.CajaID]
		saldoPrevio := saldosActuales[m.CajaID]

		// Si se envía un SaldoFinal, validar que coincida con el cálculo
		if m.SaldoFinal != 0 {
			saldoCalculado := saldoPrevio + m.Monto
			if m.SaldoFinal != saldoCalculado {
				// Este caso suele ser para validación de UI (NeedUpdateSaldo)
				// Por simplicidad en batch, podemos retornar error o registrar el saldo enviado
				return core.Err("El saldo final enviado no coincide con el sistema para la cashBank:", m.CajaID)
			}
		} else {
			m.SaldoFinal = saldoPrevio + m.Monto
		}

		record := financeTypes.CashBankMovement{
			ID:          core.SUnixTimeUUIDConcatID(m.CajaID, req.EffectiveSUnixTimeUUID()),
			CompanyID:   req.User.CompanyID,
			CajaID:      m.CajaID,
			CajaRefID:   m.CajaRefID,
			DocumentoID: m.DocumentID,
			Date:       dateUnix,
			Tipo:        m.Tipo,
			Monto:       m.Monto,
			SaldoFinal:  m.SaldoFinal,
			Created:     nowTime,
			CreatedBy:   req.User.ID,
		}

		records = append(records, record)
		saldosActuales[m.CajaID] = m.SaldoFinal

		// Actualizar el objeto cashBank
		cashBank.SaldoCurrent = m.SaldoFinal
		cashBank.Updated = nowTime
		cashBank.UpdatedBy = req.User.ID
		cajasMap[m.CajaID] = cashBank
	}

	// Guardar movimientos
	if err := db.Insert(&records); err != nil {
		return core.Err("Error al insertar movimientos de cashBank:", err)
	}

	// Preparar actualización de cajas
	for _, id := range cajasIDs.Values {
		cajasToUpdate = append(cajasToUpdate, cajasMap[id])
	}

	qCaja := db.Table[financeTypes.CashBank]()
	if err := db.Update(&cajasToUpdate, qCaja.SaldoCurrent, qCaja.Updated, qCaja.UpdatedBy); err != nil {
		return core.Err("Error al actualizar saldo de las cajas:", err)
	}

	return nil
}
