package operaciones

import (
	"app/core"
	"app/db"
	s "app/types"
	"encoding/json"
	"slices"

	"golang.org/x/sync/errgroup"
)

func PostAlmacenStock(req *core.HandlerArgs) core.HandlerResponse {

	stock := []s.AlmacenProducto{}
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
	movimientosInternos := []s.MovimientoInterno{}
	for _, e := range stock {
		movimientosInternos = append(movimientosInternos, s.MovimientoInterno{
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
	fechaHoraInicio := core.UnixToSunix(req.GetQueryInt64("fecha-hora-inicio"))
	fechaHoraFin := core.UnixToSunix(req.GetQueryInt64("fecha-hora-fin"))

	if almacenID == 0 || fechaHoraInicio == 0 || fechaHoraFin == 0 {
		return req.MakeErr("Faltan parámetros.")
	}

	type Result struct {
		Movimientos []s.AlmacenMovimiento
		Usuarios    []s.Usuario
		Productos   []s.Producto
	}

	result := Result{}

	query := db.Query(&result.Movimientos)
	query.Select().
		EmpresaID.Equals(req.Usuario.EmpresaID).
		ID.Between(
		core.SUnixTimeUUIDConcatID(almacenID, int64(fechaHoraInicio)),
		core.SUnixTimeUUIDConcatID(almacenID, int64(fechaHoraFin)+1),
	).OrderDesc().Limit(1000)

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

	almacenProductos1 := []s.AlmacenProducto{}
	almacenProductos2 := []s.AlmacenProducto{}

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

func ApplyMovimientos(req *core.HandlerArgs, movimientos []s.MovimientoInterno) error {

	keys := []string{}
	productosIDs := core.SliceSet[int32]{}

	for _, mov := range movimientos {
		keys = append(keys, mov.GetAlmacenProductoID())
		productosIDs.Add(mov.ProductoID)
	}

	// Obtiene información de los productos
	productos := []s.Producto{}
	query := db.Query(&productos)
	q1 := db.Table[s.Producto]()
	query.Select(q1.EmpresaID, q1.ID, q1.CategoriasIDs).
		EmpresaID.Equals(req.Usuario.EmpresaID).
		ID.In(productosIDs.Values...)

	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener los productos (en almacén movimientos):", err)
	}

	productosMap := core.SliceToMapE(productos, func(e s.Producto) int32 { return e.ID })

	// Obtiene el stock actual
	currentStock := []s.AlmacenProducto{}
	currentStockQuery := db.Query(&currentStock)
	currentStockQuery.Select().
		EmpresaID.Equals(req.Usuario.EmpresaID).
		ID.In(keys...)

	if err := currentStockQuery.Exec(); err != nil {
		return core.Err("Error al obtener el stock previo (1):", err)
	}

	currentStockMap := core.SliceToMapK(currentStock,
		func(e s.AlmacenProducto) string { return e.ID })

	// Obtiene el stock del producto en todos los almacenes
	productosStock := []s.AlmacenProducto{}
	productosStockQuery := db.Query(&productosStock)
	qAlm := db.Table[s.AlmacenProducto]()
	productosStockQuery.Select(qAlm.ProductoID, qAlm.AlmacenID, qAlm.Cantidad, qAlm.SubCantidad).
		EmpresaID.Equals(req.Usuario.EmpresaID).
		Status.Equals(1).
		ProductoID.In(productosIDs.Values...)

	core.Log("productos stock::", len(productosStock))

	if err := productosStockQuery.Exec(); err != nil {
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

	for _, e := range productosStock {
		addProductoStock(e.ProductoID, e.AlmacenID, e.Cantidad)
	}

	//Genera los movimientos correspondientes al stock actual
	almacenMovimientos := []s.AlmacenMovimiento{}
	almacenProductos := []s.AlmacenProducto{}
	uuid := core.SUnixTimeUUID()

	for _, e := range movimientos {
		movimiento := s.AlmacenMovimiento{
			ID:             core.SUnixTimeUUIDConcatID(e.AlmacenID, uuid),
			EmpresaID:      req.Usuario.EmpresaID,
			AlmacenID:      e.AlmacenID,
			ProductoID:     e.ProductoID,
			PresentacionID: e.PresentacionID,
			SKU:            e.SKU,
			Lote:           e.Lote,
			Tipo:           core.If(e.Cantidad > 0, int8(1), 2),
			Created:        core.SUnixTime(),
			CreatedBy:      req.Usuario.ID,
		}
		uuid++

		currentCantidad := int32(0)
		almProdID := e.GetAlmacenProductoID()
		if current, ok := currentStockMap[almProdID]; ok {
			currentCantidad = current.Cantidad
		}

		if e.ReemplazarCantidad {
			movimiento.Cantidad = e.Cantidad - currentCantidad
			movimiento.AlmacenCantidad = e.Cantidad
		} else {
			movimiento.Cantidad = e.Cantidad
			movimiento.AlmacenCantidad = currentCantidad + e.Cantidad
		}

		almacenMovimientos = append(almacenMovimientos, movimiento)

		addProductoStock(e.ProductoID, e.AlmacenID, movimiento.Cantidad)

		costoUn := float32(0)
		if cs, ok := currentStockMap[almProdID]; ok {
			costoUn = cs.CostoUn
		}

		almacenProductos = append(almacenProductos, s.AlmacenProducto{
			ID:             almProdID,
			AlmacenID:      e.AlmacenID,
			ProductoID:     e.ProductoID,
			PresentacionID: e.PresentacionID,
			EmpresaID:      req.Usuario.EmpresaID,
			SKU:            e.SKU,
			Lote:           e.Lote,
			Cantidad:       movimiento.AlmacenCantidad,
			// SubCantidad: (???),
			Status:    core.If(movimiento.AlmacenCantidad == 0, int8(0), 1),
			CostoUn:   costoUn,
			Updated:   core.SUnixTime(),
			UpdatedBy: req.Usuario.ID,
		})
	}

	core.Print(almacenProductos)

	// Insert AlmacenProducto using db2
	if err := db.Insert(&almacenProductos); err != nil {
		return core.Err("Error al guardar el stock de productos:", err)
	}

	// Insert AlmacenMovimiento using db2
	if err := db.Insert(&almacenMovimientos); err != nil {
		return core.Err("Error al guardar los movimientos:", err)
	}

	// Agrega los datos del stock a los productos
	productosToUpdate := []s.Producto{}
	for _, e := range productoStock {
		almacenStock := []s.AlmacenStockMin{}
		for almacenID, cant := range e.almacenesStock {
			if cant.cantidad == 0 && cant.subcantidad == 0 {
				continue
			}
			almacenStock = append(almacenStock, s.AlmacenStockMin{
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

	q2 := db.Table[s.Producto]()
	if err := db.Update(&productosToUpdate, q2.Stock, q2.StockStatus); err != nil {
		// Este error es interno, dado que el stock ya se guardó en la tabla principal de almacén_productos
		core.Log("Error al actualizar el stock en productos:", err)
	}

	return nil
}
