package logistica

import (
	"app/core"
	"app/db"
	logisticaTypes "app/logistica/types"
	negocioTypes "app/negocio/types"
	"fmt"
)

func RecalcProductStockByMovements(empresaID int32) error {

	// 1. Fetch all movements for the company
	movimientos := []logisticaTypes.AlmacenMovimiento{}
	query := db.Query(&movimientos)
	query.Select().EmpresaID.Equals(empresaID)

	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener movimientos:", err)
	}

	// 2. Compute stock
	computedStocks := make(map[string]*logisticaTypes.ProductoStock)
	productoTotalStock := make(map[int32]int32)

	for _, mov := range movimientos {
		key := core.Concat62(mov.AlmacenID, mov.ProductoID, mov.PresentacionID, mov.SKU, mov.Lote)
		if _, exists := computedStocks[key]; !exists {
			computedStocks[key] = &logisticaTypes.ProductoStock{
				EmpresaID:      empresaID,
				ID:             key,
				AlmacenID:      mov.AlmacenID,
				ProductoID:     mov.ProductoID,
				PresentacionID: mov.PresentacionID,
				SKU:            mov.SKU,
				Lote:           mov.Lote,
			}
		}

		computedStocks[key].Cantidad += mov.Cantidad
		computedStocks[key].SubCantidad += mov.SubCantidad

		productoTotalStock[mov.ProductoID] += mov.Cantidad
	}

	// 3. Prepare the new slice
	newStocks := make([]logisticaTypes.ProductoStock, 0, len(computedStocks))
	updatedTime := core.SUnixTime()

	for _, stock := range computedStocks {
		if stock.Cantidad > 0 || stock.SubCantidad > 0 {
			stock.Status = 1
		} else {
			stock.Status = 0
		}
		stock.Updated = updatedTime
		newStocks = append(newStocks, *stock)
	}

	// 4. Overwrite ProductoStock table
	// Delete all existing stock for the company (we use partition key)
	delStatement := fmt.Sprintf("DELETE FROM genix.almacen_producto WHERE empresa_id = %d", empresaID)
	if err := db.QueryExec(delStatement); err != nil {
		return core.Err("Error al eliminar el stock actual:", err)
	}

	// Insert the recalculated stocks
	if len(newStocks) > 0 {
		if err := db.Insert(&newStocks); err != nil {
			return core.Err("Error al insertar el nuevo stock:", err)
		}
	}

	// 5. Update Producto.StockStatus and CategoriasConStock
	productos := []negocioTypes.Producto{}
	qProd := db.Query(&productos)
	q1 := db.Table[negocioTypes.Producto]()
	qProd.Select(q1.ID, q1.EmpresaID, q1.CategoriasIDs, q1.StockStatus).
		EmpresaID.Equals(empresaID)

	if err := qProd.Exec(); err != nil {
		return core.Err("Error al obtener productos:", err)
	}

	productosToUpdate := []negocioTypes.Producto{}
	for _, p := range productos {
		totalCantidad := productoTotalStock[p.ID]
		oldStatus := p.StockStatus

		if totalCantidad > 0 {
			p.StockStatus = 1
		} else {
			p.StockStatus = 0
		}

		// Always update if we need to ensure consistency, 
		// or at least when status changed. We'll update all for safety.
		p.FillCategoriasConStock()

		// Only append if something logically changed, or just update all
		if oldStatus != p.StockStatus || len(p.CategoriasConStock) > 0 || oldStatus > 0 {
			productosToUpdate = append(productosToUpdate, p)
		}
	}

	if len(productosToUpdate) > 0 {
		qTable := db.Table[negocioTypes.Producto]()
		if err := db.Update(&productosToUpdate, qTable.StockStatus, qTable.CategoriasConStock); err != nil {
			return core.Err("Error al actualizar el stock y categorias en productos:", err)
		}
	}

	return nil
}
