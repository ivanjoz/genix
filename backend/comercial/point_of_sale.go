package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
	"app/operaciones"
	"encoding/json"
	s "app/types"
	"slices"
	"time"
)

func PostSaleOrder(req *core.HandlerArgs) core.HandlerResponse {
	nowTime := core.SUnixTime()
	sale := types.SaleOrder{}
	err := json.Unmarshal([]byte(*req.Body), &sale)
	if err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}

	sale.EmpresaID = req.Usuario.EmpresaID
	sale.Fecha = core.TimeToFechaUnix(time.Now())
	sale.Created = nowTime
	sale.Updated = nowTime
	sale.UpdatedBy = req.Usuario.ID
	sale.Status = 1

	// Insertar el registro de venta para obtener el ID (autoincrement)
	if err := db.Insert(&[]types.SaleOrder{sale}); err != nil {
		return req.MakeErr("Error al registrar la venta:", err)
	}

	// 2 = Pago (Registro en Caja)
	if slices.Contains(sale.ProcessesIncluded_, 2) {
		if sale.CajaID_ == 0 {
			return req.MakeErr("Se requiere CajaID_ para procesar el pago.")
		}

		montoPago := sale.TotalAmount - sale.DebtAmount
		if montoPago != 0 {
			movimiento := s.CajaMovimientoInterno{
				CajaID:  sale.CajaID_,
				VentaID: sale.ID,
				Tipo:    8, // Cobro (Venta)
				Monto:   montoPago,
			}

			if err := operaciones.ApplyCajaMovimientos(req, []s.CajaMovimientoInterno{movimiento}); err != nil {
				core.Log("Error al aplicar movimiento de caja:", err)
				return req.MakeErr("Error al registrar el movimiento de caja: " + err.Error())
			}
		}
	}

	// 3 = Entrega (Movimiento de Almacén)
	if slices.Contains(sale.ProcessesIncluded_, 3) {
		if sale.AlmacenID == 0 {
			return req.MakeErr("Se requiere AlmacenID para procesar la entrega.")
		}

		if len(sale.DetailProductsIDs) == 0 {
			return req.MakeErr("No hay productos en el detalle para procesar la entrega.")
		}

		movimientosInternos := []s.MovimientoInterno{}
		for i, productoID := range sale.DetailProductsIDs {
			if i >= len(sale.DetailQuantities) {
				break
			}
			cantidad := sale.DetailQuantities[i]
			if cantidad == 0 {
				continue
			}

			movimientosInternos = append(movimientosInternos, s.MovimientoInterno{
				AlmacenID:  sale.AlmacenID,
				ProductoID: productoID,
				Cantidad:   -cantidad, // Salida de almacén
			})
		}

		if len(movimientosInternos) > 0 {
			if err := operaciones.ApplyMovimientos(req, movimientosInternos); err != nil {
				core.Log("Error al aplicar movimientos de almacén:", err)
				return req.MakeErr("Error al procesar la salida de almacén: " + err.Error())
			}
		}
	}

	return req.MakeResponse(sale)
}
