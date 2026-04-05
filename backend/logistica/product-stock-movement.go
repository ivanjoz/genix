package logistica

import (
	"app/core"
	coretypes "app/core/types"
	"app/db"
	logisticaTypes "app/logistica/types"
	negocioTypes "app/negocio/types"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"
)

func PostAlmacenStock(req *core.HandlerArgs) core.HandlerResponse {

	stock := []logisticaTypes.ProductStock{}
	err := json.Unmarshal([]byte(*req.Body), &stock)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(stock) == 0 {
		return req.MakeErr("No se enviaron registros.")
	}

	for _, e := range stock {
		if e.WarehouseID == 0 || e.ProductID == 0 {
			return req.MakeErr("Hay un registros sin Almacén-ID o Producto-ID")
		}
	}

	// Genera los movimientos internos
	movimientosInternos := []logisticaTypes.MovimientoInterno{}
	for _, e := range stock {
		movimientosInternos = append(movimientosInternos, logisticaTypes.MovimientoInterno{
			ReemplazarCantidad: true,
			WarehouseID:        e.WarehouseID,
			ProductoID:         e.ProductID,
			PresentacionID:     e.PresentationID,
			SKU:                e.SKU,
			Lote:               e.Lote,
			Cantidad:           e.Quantity,
			SubCantidad:        e.SubQuantity,
		})
	}

	if err = ApplyMovimientos(req, movimientosInternos); err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(stock)
}

func GetAlmacenMovimientos(req *core.HandlerArgs) core.HandlerResponse {

	almacenID := req.GetQueryInt("almacen-id")
	fechaInicio := req.GetQueryInt16("fecha-inicio")
	fechaFin := req.GetQueryInt16("fecha-fin")

	if almacenID == 0 || fechaInicio == 0 || fechaFin == 0 {
		return req.MakeErr("Faltan parámetros.")
	}

	type Result struct {
		Movimientos []logisticaTypes.WarehouseProductMovement
		Usuarios    []coretypes.Usuario
		Productos   []negocioTypes.Producto
	}

	result := Result{}

	query := db.Query(&result.Movimientos)

	query.CompanyID.Equals(req.Usuario.EmpresaID).
		WarehouseID.Equals(almacenID).
		Fecha.Between(fechaInicio, fechaFin).OrderDesc().Limit(1000)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	core.Log("movimientos encontrados:", len(result.Movimientos))

	if len(result.Movimientos) == 0 {
		return req.MakeResponse(result)
	}

	core.Print(result.Movimientos[0])

	usuariosSet := core.SliceSet[int32]{}
	productosSet := core.SliceSet[int32]{}

	for _, e := range result.Movimientos {
		usuariosSet.Add(e.CreatedBy)
		productosSet.Add(e.ProductoID)
	}

	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&result.Productos)
		query.Select(query.ID, query.Nombre, query.Precio).
			EmpresaID.Equals(req.Usuario.EmpresaID).
			ID.In(productosSet.Values...)
		err := query.Exec()
		if err != nil {
			err = core.Err("Error al obtener los movimientos de almacnén:", err)
		}
		return err
	})

	errGroup.Go(func() error {
		query := db.Query(&result.Usuarios)
		query.Select(query.ID, query.Usuario, query.Nombres, query.Apellidos).
			EmpresaID.Equals(req.Usuario.EmpresaID).
			ID.In(usuariosSet.Values...)
		err := query.Exec()
		if err != nil {
			err = core.Err("Error al obtener los usuarios:", err)
		}
		return err
	})

	if err := errGroup.Wait(); err != nil {
		return req.MakeErr(err.Error())
	}

	return req.MakeResponse(result)
}

func GetProductosStock(req *core.HandlerArgs) core.HandlerResponse {
	almacenID := req.GetQueryInt("almacen-id")
	updated := req.GetQueryInt("updated")
	core.Log("updated::", updated)

	almacenProductos1 := []logisticaTypes.ProductStock{}
	almacenProductos2 := []logisticaTypes.ProductStock{}

	eg := errgroup.Group{}

	// Si se necesita obtener un delta
	if updated > 0 {
		eg.Go(func() error {
			query := db.Query(&almacenProductos1)
			query.Select().
				CompanyID.Equals(req.Usuario.EmpresaID).
				WarehouseID.Equals(int32(almacenID)).
				Status.Equals(1).
				Updated.Between(int32(updated), int32(999999999))
			return query.Exec()
		})
		eg.Go(func() error {
			query := db.Query(&almacenProductos2)
			query.Select().
				CompanyID.Equals(req.Usuario.EmpresaID).
				WarehouseID.Equals(int32(almacenID)).
				Status.Equals(0).
				Updated.Between(int32(updated), int32(999999999))
			return query.Exec()
		})
	} else {
		eg.Go(func() error {
			query := db.Query(&almacenProductos2)
			query.Select().
				CompanyID.Equals(req.Usuario.EmpresaID).
				WarehouseID.Equals(int32(almacenID)).
				Status.Equals(1).
				Updated.Between(int32(0), int32(999999999))
			return query.Exec()
		})
	}

	if err := eg.Wait(); err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	almacenProductos1 = slices.Concat(almacenProductos1, almacenProductos2)

	return req.MakeResponse(almacenProductos1)
}

type almacenStockCount struct {
	cantidad    int32
	subcantidad int32
}

type productoStockCounter struct {
	productoID     int32
	almacenesStock map[int32]almacenStockCount
}

var applyMovimientosLockByCompany = map[int32]*sync.Mutex{}
var applyMovimientosLockMapMu sync.Mutex

func getApplyMovimientosCompanyLock(companyID int32) *sync.Mutex {
	// Reuse one mutex per company so stock writes for the same tenant stay serialized.
	applyMovimientosLockMapMu.Lock()
	companyLock := applyMovimientosLockByCompany[companyID]
	if companyLock == nil {
		companyLock = &sync.Mutex{}
		applyMovimientosLockByCompany[companyID] = companyLock
	}
	applyMovimientosLockMapMu.Unlock()
	return companyLock
}

func ApplyMovimientos(req *core.HandlerArgs, movimientos []logisticaTypes.MovimientoInterno) error {
	companyID := req.Usuario.EmpresaID
	companyLock := getApplyMovimientosCompanyLock(companyID)

	core.Log("ApplyMovimientos esperando lock de empresa:", companyID, "movimientos:", len(movimientos))
	companyLock.Lock()
	// Keep the critical section scoped to the full stock update pipeline for one company.
	defer companyLock.Unlock()
	core.Log("ApplyMovimientos lock adquirido para empresa:", companyID)

	pendingWarehouseProductIDs := core.SliceSet[string]{}
	productosIDs := core.SliceSet[int32]{}
	currentStockMap := map[string]*logisticaTypes.ProductStock{}

	for _, mov := range movimientos {
		if mov.Cantidad == 0 {
			continue
		}
		productosIDs.Add(mov.ProductoID)
	}

	for _, mov := range movimientos {
		if mov.Cantidad == 0 {
			continue
		}
		for _, key := range []string{mov.GetAlmacenProductoID(), mov.GetAlmacenProductoGrupoID()} {
			pendingWarehouseProductIDs.Add(key)
		}
	}

	// Load every stock row involved in this batch from DB and ignore any embedded stock in the movement payload.
	if !pendingWarehouseProductIDs.IsEmpty() {
		currentStock := []logisticaTypes.ProductStock{}
		currentStockQuery := db.Query(&currentStock)
		currentStockQuery.Select().
			CompanyID.Equals(req.Usuario.EmpresaID).
			ID.In(pendingWarehouseProductIDs.Values...)

		if err := currentStockQuery.Exec(); err != nil {
			return core.Err("Error al obtener el stock previo (1):", err)
		}

		for i := range currentStock {
			currentStockMap[currentStock[i].ID] = &currentStock[i]
		}
	}
	core.Log("ApplyMovimientos stock consultado:", len(pendingWarehouseProductIDs.Values))

	for _, key := range pendingWarehouseProductIDs.Values {
		if currentStockMap[key] == nil {
			keyParser := db.KeyParser{Key: key}

			currentStockMap[key] = &logisticaTypes.ProductStock{
				CompanyID:      req.Usuario.EmpresaID,
				ID:             key,
				WarehouseID:    int32(keyParser.GetNumber(0)),
				ProductID:      int32(keyParser.GetNumber(1)),
				PresentationID: int16(keyParser.GetNumber(2)),
				SKU:            keyParser.GetString(3),
				Lote:           keyParser.GetString(4),
				IsNewRecord:    true,
			}
		}
	}

	// Obtiene el stock del producto en todos los almacenes
	almacenProductosStock := []logisticaTypes.ProductStock{}
	apTable := db.Query(&almacenProductosStock)

	apTable.Select(apTable.ProductID, apTable.WarehouseID, apTable.Quantity, apTable.SubQuantity).
		CompanyID.Equals(req.Usuario.EmpresaID).
		Status.Equals(1).
		ProductID.In(productosIDs.Values...)

	core.Log("productos stock::", len(almacenProductosStock))

	if err := apTable.Exec(); err != nil {
		return core.Err("Error al obtener el stock previo (2):", err)
	}

	productoStock := map[int32]*productoStockCounter{}

	addProductoStock := func(productoID, almacenID, cantidad int32) {
		ps := productoStock[productoID]
		if ps == nil {
			ps = &productoStockCounter{
				productoID:     productoID,
				almacenesStock: map[int32]almacenStockCount{},
			}
			productoStock[productoID] = ps
		}

		count := ps.almacenesStock[almacenID]
		count.cantidad += cantidad
		ps.almacenesStock[almacenID] = count
	}

	for _, e := range almacenProductosStock {
		addProductoStock(e.ProductID, e.WarehouseID, e.Quantity)
	}

	//Genera los movimientos correspondientes al stock actual
	almacenMovimientos := []logisticaTypes.WarehouseProductMovement{}
	updatedStockTime := req.EffectiveSUnixTime()
	// Clona el stock actual para acumular movimientos repetidos sin mutar el mapa base.
	almacenProductosMap := maps.Clone(currentStockMap)

	for _, e := range movimientos {
		movimiento := logisticaTypes.WarehouseProductMovement{
			DocumentID:     e.DocumentID,
			CompanyID:      req.Usuario.EmpresaID,
			WarehouseID:    e.WarehouseID,
			ProductoID:     e.ProductoID,
			PresentacionID: e.PresentacionID,
			SKU:            e.SKU,
			Lote:           e.Lote,
			Tipo:           core.Coalesce(e.Tipo, core.If(e.Cantidad > 0, int8(1), 2)),
			Fecha:          req.EffectiveFechaUnix(),
			Created:        req.EffectiveSUnixTime(),
			CreatedBy:      req.Usuario.ID,
		}

		almProdID := e.GetAlmacenProductoID()
		stockActual := almacenProductosMap[almProdID]
		currentCantidad := stockActual.Quantity

		if e.ReemplazarCantidad {
			movimiento.Quantity = e.Cantidad - currentCantidad
			movimiento.WarehouseQuantity = e.Cantidad
		} else {
			movimiento.Quantity = e.Cantidad
			movimiento.WarehouseQuantity = currentCantidad + e.Cantidad
		}

		almacenMovimientos = append(almacenMovimientos, movimiento)

		addProductoStock(e.ProductoID, e.WarehouseID, movimiento.Quantity)

		// Mantiene un acumulado por stock ID para que movimientos repetidos sumen sobre el saldo ya ajustado.
		stockActual.ID = almProdID
		stockActual.WarehouseID = e.WarehouseID
		stockActual.ProductID = e.ProductoID
		stockActual.CompanyID = req.Usuario.EmpresaID
		stockActual.Quantity = movimiento.WarehouseQuantity
		stockActual.Status = core.If(movimiento.WarehouseQuantity == 0, int8(0), 1)
		stockActual.Updated = updatedStockTime

		// Computa la cantidad del grupo
		almProdGrupoID := e.GetAlmacenProductoGrupoID()
		stockActualGrupo := almacenProductosMap[almProdGrupoID]
		stockActualGrupo.WarehouseProductQuantity = stockActualGrupo.WarehouseProductQuantity + movimiento.Quantity
		stockActual.Updated = updatedStockTime
	}

	almacenProductosToInsert := []logisticaTypes.ProductStock{}
	almacenProductosToUpdate := []logisticaTypes.ProductStock{}

	for _, e := range core.MapToSlice(almacenProductosMap) {
		if e.Quantity < 0 {
			return core.Err(fmt.Sprintf(`El producto %v | almacen %v | presentacion %v, resulta en un stock menor a 0.`, e.ProductID, e.WarehouseID, e.PresentationID))
		}

		if e.IsNewRecord {
			almacenProductosToInsert = append(almacenProductosToInsert, e)
		} else {
			almacenProductosToUpdate = append(almacenProductosToUpdate, e)
		}
	}

	core.Log("almacenProductosToInsert:", len(almacenProductosToInsert), "|", "almacenProductosToUpdate:", len(almacenProductosToUpdate))

	// Actualiza solo el estado mutable del stock para no reescribir columnas estables.
	apT := db.Table[logisticaTypes.ProductStock]()

	if len(almacenProductosToUpdate) > 0 {
		if err := db.Update(&almacenProductosToUpdate,
			apT.WarehouseID, apT.ProductID, apT.Quantity, apT.Status, apT.Updated, apT.WarehouseProductQuantity, apT.IsWarehouseProductStatus,
		); err != nil {
			return core.Err("Error al guardar el stock de productos:", err)
		}
	}

	if len(almacenProductosToInsert) > 0 {
		if err := db.Insert(&almacenProductosToInsert); err != nil {
			return core.Err("Error al guardar el stock de productos:", err)
		}
	}

	// Insert AlmacenMovimiento using db
	if err := db.Insert(&almacenMovimientos); err != nil {
		return core.Err("Error al guardar los movimientos:", err)
	}

	/*
		// Obtiene información de los productos
		productos := []negocioTypes.Producto{}
		query := db.Query(&productos)
		q1 := db.Table[negocioTypes.Producto]()
		query.Select(q1.EmpresaID, q1.ID, q1.CategoriasIDs).
			EmpresaID.Equals(req.Usuario.EmpresaID).
			ID.In(productosIDs.Values...)

		if err := query.Exec(); err != nil {
			return core.Err("Error al obtener los productos (en almacén movimientos):", err)
		}

		productosMap := core.SliceToMapE(productos, func(e negocioTypes.Producto) int32 { return e.ID })

		// Agrega los datos del stock a los productos
		productosToUpdate := []negocioTypes.Producto{}
		for _, e := range productoStock {
			almacenStock := []negocioTypes.AlmacenStockMin{}
			for almacenID, cant := range e.almacenesStock {
				if cant.cantidad == 0 && cant.subcantidad == 0 {
					continue
				}
				almacenStock = append(almacenStock, negocioTypes.AlmacenStockMin{
					WarehouseID: almacenID,
					Cantidad:  cant.cantidad,
				})
			}

			producto := productosMap[e.productoID]
			producto.Stock = almacenStock
			producto.StockStatus = core.If(almacenStock == nil, int8(0), 1)
			producto.FillCategoriasConStock()

			productosToUpdate = append(productosToUpdate, producto)
		}

		//core.Log("productos for update...")
		// core.Print(productosToUpdate)

		q2 := db.Table[negocioTypes.Producto]()
		if err := db.Update(&productosToUpdate, q2.Stock, q2.StockStatus, q2.CategoriasConStock); err != nil {
			// Este error es interno, dado que el stock ya se guardó en la tabla principal de almacén_productos
			core.Log("Error al actualizar el stock en productos:", err)
		}
	*/
	return nil
}
