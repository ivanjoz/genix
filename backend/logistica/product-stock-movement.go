package logistica

import (
	"app/core"
	coretypes "app/core/types"
	"app/db"
	logisticaTypes "app/logistica/types"
	negocioTypes "app/negocio/types"
	"encoding/json"
	"maps"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"
)

func PostAlmacenStock(req *core.HandlerArgs) core.HandlerResponse {

	stock := []logisticaTypes.ProductoStock{}
	err := json.Unmarshal([]byte(*req.Body), &stock)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(stock) == 0 {
		return req.MakeErr("No se enviaron registros.")
	}

	for _, e := range stock {
		if e.AlmacenID == 0 || e.ProductoID == 0 {
			return req.MakeErr("Hay un registros sin Almacén-ID o Producto-ID")
		}
	}

	// Genera los movimientos internos
	movimientosInternos := []logisticaTypes.MovimientoInterno{}
	for _, e := range stock {
		movimientosInternos = append(movimientosInternos, logisticaTypes.MovimientoInterno{
			ReemplazarCantidad: true,
			AlmacenID:          e.AlmacenID,
			ProductoID:         e.ProductoID,
			PresentacionID:     e.PresentacionID,
			SKU:                e.SKU,
			Lote:               e.Lote,
			Cantidad:           e.Cantidad,
			SubCantidad:        e.SubCantidad,
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
		Movimientos []logisticaTypes.AlmacenMovimiento
		Usuarios    []coretypes.Usuario
		Productos   []negocioTypes.Producto
	}

	result := Result{}

	query := db.Query(&result.Movimientos)

	query.EmpresaID.Equals(req.Usuario.EmpresaID).
		AlmacenID.Equals(almacenID).
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

	almacenProductos1 := []logisticaTypes.ProductoStock{}
	almacenProductos2 := []logisticaTypes.ProductoStock{}

	eg := errgroup.Group{}

	// Si se necesita obtener un delta
	if updated > 0 {
		eg.Go(func() error {
			query := db.Query(&almacenProductos1)
			query.Select().
				EmpresaID.Equals(req.Usuario.EmpresaID).
				AlmacenID.Equals(int32(almacenID)).
				Status.Equals(1).
				Updated.Between(int32(updated), int32(999999999))
			return query.Exec()
		})
		eg.Go(func() error {
			query := db.Query(&almacenProductos2)
			query.Select().
				EmpresaID.Equals(req.Usuario.EmpresaID).
				AlmacenID.Equals(int32(almacenID)).
				Status.Equals(0).
				Updated.Between(int32(updated), int32(999999999))
			return query.Exec()
		})
	} else {
		eg.Go(func() error {
			query := db.Query(&almacenProductos2)
			query.Select().
				EmpresaID.Equals(req.Usuario.EmpresaID).
				AlmacenID.Equals(int32(almacenID)).
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

func ApplyMovimientos(req *core.HandlerArgs, movimientos []logisticaTypes.MovimientoInterno) error {

	keysPendientes := []string{}
	productosIDs := core.SliceSet[int32]{}
	currentStockMap := map[string]logisticaTypes.ProductoStock{}
	reusedStockCount := 0

	for _, mov := range movimientos {
		productosIDs.Add(mov.ProductoID)
		if mov.Stock != nil {
			reusedStockCount++
			currentStockMap[mov.GetAlmacenProductoID()] = *mov.Stock
		}
	}

	for _, mov := range movimientos {
		almacenProductoID := mov.GetAlmacenProductoID()
		if _, exists := currentStockMap[almacenProductoID]; !exists {
			keysPendientes = append(keysPendientes, almacenProductoID)
		}
	}

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

	// Consulta solo los stocks que no llegaron adjuntos en el movimiento.
	if len(keysPendientes) > 0 {
		currentStock := []logisticaTypes.ProductoStock{}
		currentStockQuery := db.Query(&currentStock)
		currentStockQuery.Select().
			EmpresaID.Equals(req.Usuario.EmpresaID).
			ID.In(keysPendientes...)

		if err := currentStockQuery.Exec(); err != nil {
			return core.Err("Error al obtener el stock previo (1):", err)
		}

		for _, stockActual := range currentStock {
			currentStockMap[stockActual.ID] = stockActual
		}
	}
	core.Log("ApplyMovimientos stock reutilizado:", reusedStockCount, "stock consultado:", len(keysPendientes))

	// Obtiene el stock del producto en todos los almacenes
	almacenProductosStock := []logisticaTypes.ProductoStock{}
	apTable := db.Query(&almacenProductosStock)

	apTable.Select(apTable.ProductoID, apTable.AlmacenID, apTable.Cantidad, apTable.SubCantidad).
		EmpresaID.Equals(req.Usuario.EmpresaID).
		Status.Equals(1).
		ProductoID.In(productosIDs.Values...)

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
		addProductoStock(e.ProductoID, e.AlmacenID, e.Cantidad)
	}

	//Genera los movimientos correspondientes al stock actual
	almacenMovimientos := []logisticaTypes.AlmacenMovimiento{}
	updatedStockTime := core.SUnixTime()
	// Clona el stock actual para acumular movimientos repetidos sin mutar el mapa base.
	almacenProductosMap := maps.Clone(currentStockMap)

	for _, e := range movimientos {
		movimiento := logisticaTypes.AlmacenMovimiento{
			DocumentID:     e.DocumentID,
			EmpresaID:      req.Usuario.EmpresaID,
			AlmacenID:      e.AlmacenID,
			ProductoID:     e.ProductoID,
			PresentacionID: e.PresentacionID,
			SKU:            e.SKU,
			Lote:           e.Lote,
			Tipo:           core.Coalesce(e.Tipo, core.If(e.Cantidad > 0, int8(1), 2)),
			Fecha:          core.TimeToFechaUnix(time.Now()),
			Created:        core.SUnixTime(),
			CreatedBy:      req.Usuario.ID,
		}

		almProdID := e.GetAlmacenProductoID()
		stockActual := almacenProductosMap[almProdID]
		currentCantidad := stockActual.Cantidad

		if e.ReemplazarCantidad {
			movimiento.Cantidad = e.Cantidad - currentCantidad
			movimiento.AlmacenCantidad = e.Cantidad
		} else {
			movimiento.Cantidad = e.Cantidad
			movimiento.AlmacenCantidad = currentCantidad + e.Cantidad
		}

		almacenMovimientos = append(almacenMovimientos, movimiento)

		addProductoStock(e.ProductoID, e.AlmacenID, movimiento.Cantidad)

		// Mantiene un acumulado por stock ID para que movimientos repetidos sumen sobre el saldo ya ajustado.
		stockActual.ID = almProdID
		stockActual.AlmacenID = e.AlmacenID
		stockActual.ProductoID = e.ProductoID
		stockActual.EmpresaID = req.Usuario.EmpresaID
		stockActual.Cantidad = movimiento.AlmacenCantidad
		stockActual.Status = core.If(movimiento.AlmacenCantidad == 0, int8(0), 1)
		stockActual.Updated = updatedStockTime
		almacenProductosMap[almProdID] = stockActual
	}

	almacenProductos := make([]logisticaTypes.ProductoStock, 0, len(almacenProductosMap))
	for _, almacenProducto := range almacenProductosMap {
		almacenProductos = append(almacenProductos, almacenProducto)
	}

	core.Print(almacenProductos)

	// Actualiza solo el estado mutable del stock para no reescribir columnas estables.
	almacenProductoTable := db.Table[logisticaTypes.ProductoStock]()
	if err := db.Update(&almacenProductos,
		almacenProductoTable.AlmacenID,
		almacenProductoTable.ProductoID,
		almacenProductoTable.Cantidad,
		almacenProductoTable.Status,
		almacenProductoTable.Updated,
	); err != nil {
		return core.Err("Error al guardar el stock de productos:", err)
	}

	// Insert AlmacenMovimiento using db2
	if err := db.Insert(&almacenMovimientos); err != nil {
		return core.Err("Error al guardar los movimientos:", err)
	}

	// Agrega los datos del stock a los productos
	productosToUpdate := []negocioTypes.Producto{}
	for _, e := range productoStock {
		almacenStock := []negocioTypes.AlmacenStockMin{}
		for almacenID, cant := range e.almacenesStock {
			if cant.cantidad == 0 && cant.subcantidad == 0 {
				continue
			}
			almacenStock = append(almacenStock, negocioTypes.AlmacenStockMin{
				AlmacenID: almacenID,
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

	return nil
}

