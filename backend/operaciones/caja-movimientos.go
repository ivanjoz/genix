package operaciones

import (
	"app/core"
	"app/db"
	"app/shared"
	s "app/types"
	"time"
)

func ApplyCajaMovimientos(req *core.HandlerArgs, movimientos []s.CajaMovimientoInterno) error {
	if len(movimientos) == 0 {
		return nil
	}

	nowTime := core.SUnixTime()
	fechaUnix := core.TimeToFechaUnix(time.Now())
	
	// Agrupar movimientos por CajaID para manejar saldos
	cajasIDs := core.SliceSet[int32]{}
	for _, m := range movimientos {
		cajasIDs.Add(m.CajaID)
	}

	cajasMap := make(map[int32]s.Caja)
	for _, id := range cajasIDs.Values {
		caja, err := shared.GetCaja(req.Usuario.EmpresaID, id)
		if err != nil {
			return core.Err("Error al obtener la caja:", id, err)
		}
		cajasMap[id] = caja
	}

	records := []s.CajaMovimiento{}
	cajasToUpdate := []s.Caja{}
	
	// Para rastrear el saldo actual si hay múltiples movimientos para la misma caja
	saldosActuales := make(map[int32]int32)
	for id, caja := range cajasMap {
		saldosActuales[id] = caja.SaldoCurrent
	}

	for _, m := range movimientos {
		caja := cajasMap[m.CajaID]
		saldoPrevio := saldosActuales[m.CajaID]

		// Si se envía un SaldoFinal, validar que coincida con el cálculo
		if m.SaldoFinal != 0 {
			saldoCalculado := saldoPrevio + m.Monto
			if m.SaldoFinal != saldoCalculado {
				// Este caso suele ser para validación de UI (NeedUpdateSaldo)
				// Por simplicidad en batch, podemos retornar error o registrar el saldo enviado
				return core.Err("El saldo final enviado no coincide con el sistema para la caja:", m.CajaID)
			}
		} else {
			m.SaldoFinal = saldoPrevio + m.Monto
		}

		record := s.CajaMovimiento{
			ID:         core.SUnixTimeUUIDConcatID(m.CajaID),
			EmpresaID:  req.Usuario.EmpresaID,
			CajaID:     m.CajaID,
			CajaRefID:  m.CajaRefID,
			VentaID:    m.DocumentID,
			Fecha:      fechaUnix,
			Tipo:       m.Tipo,
			Monto:      m.Monto,
			SaldoFinal: m.SaldoFinal,
			Created:    nowTime,
			CreatedBy:  req.Usuario.ID,
		}

		records = append(records, record)
		saldosActuales[m.CajaID] = m.SaldoFinal
		
		// Actualizar el objeto caja
		caja.SaldoCurrent = m.SaldoFinal
		caja.Updated = nowTime
		caja.UpdatedBy = req.Usuario.ID
		cajasMap[m.CajaID] = caja
	}

	// Guardar movimientos
	if err := db.Insert(&records); err != nil {
		return core.Err("Error al insertar movimientos de caja:", err)
	}

	// Preparar actualización de cajas
	for _, id := range cajasIDs.Values {
		cajasToUpdate = append(cajasToUpdate, cajasMap[id])
	}

	qCaja := db.Table[s.Caja]()
	if err := db.Update(&cajasToUpdate, qCaja.SaldoCurrent, qCaja.Updated, qCaja.UpdatedBy); err != nil {
		return core.Err("Error al actualizar saldo de las cajas:", err)
	}

	return nil
}
