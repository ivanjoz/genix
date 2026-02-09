package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
	"app/operaciones"
	s "app/types"
	"encoding/json"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"
)

func PostSaleOrder(req *core.HandlerArgs) core.HandlerResponse {
	nowTime := core.SUnixTime()
	sale := types.SaleOrder{}
	err := json.Unmarshal([]byte(*req.Body), &sale)
	if err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}

	if len(sale.DetailProductsIDs) != len(sale.DetailPrices) ||
		len(sale.DetailProductsIDs) != len(sale.DetailQuantities) {
		return req.MakeErr("El registro posee propiedades incorrectas.")
	}

	for _, value := range slices.Concat(sale.DetailProductsIDs, sale.DetailQuantities, sale.DetailPrices) {
		if value == 0 {
			return req.MakeErr("Hay un valor incorrecto.")
		}
	}

	sale.EmpresaID = req.Usuario.EmpresaID
	sale.Fecha = core.TimeToFechaUnix(time.Now())
	sale.Created = nowTime
	sale.Updated = nowTime
	sale.UpdatedBy = req.Usuario.ID
	sale.Status = 1
	sales := []types.SaleOrder{sale}

	// Insertar el registro de venta para obtener el ID (autoincrement)
	if err := db.Insert(&sales); err != nil {
		return req.MakeErr("Error al registrar la venta:", err)
	}

	sale.ID = sales[0].ID
	if sale.ID == 0 {
		return req.MakeErr("Error al obtener el ID de la venta.")
	}

	eg := errgroup.Group{}

	// 2 = Pago (Registro en Caja)
	if slices.Contains(sale.ProcessesIncluded_, 2) {
		sale.Status += 1
		if sale.CajaID_ == 0 {
			return req.MakeErr("Se requiere CajaID_ para procesar el pago.")
		}

		montoPago := sale.TotalAmount - sale.DebtAmount
		if montoPago != 0 {
			movimiento := s.CajaMovimientoInterno{
				CajaID:     sale.CajaID_,
				DocumentID: sale.ID,
				Tipo:       8, // Cobro (Venta)
				Monto:      montoPago,
			}

			eg.Go(func() error {
				if err := operaciones.ApplyCajaMovimientos(req, []s.CajaMovimientoInterno{movimiento}); err != nil {
					core.Log("Error al aplicar movimiento de caja:", err)
					return core.Err("Error al registrar el movimiento de caja:", err)
				}
				return nil
			})

			if err := operaciones.ApplyCajaMovimientos(req, []s.CajaMovimientoInterno{movimiento}); err != nil {
				core.Log("Error al aplicar movimiento de caja:", err)
				return req.MakeErr("Error al registrar el movimiento de caja: " + err.Error())
			}
		}
	}

	// 3 = Entrega (Movimiento de Almacén)
	if slices.Contains(sale.ProcessesIncluded_, 3) {
		sale.Status += 2
		if sale.AlmacenID == 0 {
			return req.MakeErr("Se requiere AlmacenID para procesar la entrega.")
		}

		if len(sale.DetailProductsIDs) == 0 {
			return req.MakeErr("No hay productos en el detalle para procesar la entrega.")
		}

		core.Log("Incluyendo movimientos internos...", len(sale.DetailProductsIDs))

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
				DocumentID: sale.ID,
				Tipo:       8,         // Entrega a cliente final (Venta)
				Cantidad:   -cantidad, // Salida de almacén
			})
		}

		core.Print(movimientosInternos)

		if len(movimientosInternos) > 0 {
			eg.Go(func() error {
				if err := operaciones.ApplyMovimientos(req, movimientosInternos); err != nil {
					core.Log("Error al aplicar movimientos de almacén:", err)
					return core.Err("Error al procesar la salida de almacén:", err)
				}
				return nil
			})
		}
	}

	if err := eg.Wait(); err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(sale)
}
