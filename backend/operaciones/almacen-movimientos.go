package operaciones

import (
	"app/core"
	s "app/types"
	"encoding/json"
	"slices"

	"golang.org/x/sync/errgroup"
)

func PostAlmacenStock(req *core.HandlerArgs) core.HandlerResponse {

	stock := []s.AlmacenProducto{}
	nowTime := core.SunixTime()

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

	keys := []any{}

	for i := range stock {
		e := &stock[i]
		e.EmpresaID = req.Usuario.EmpresaID
		e.Updated = nowTime
		if e.Cantidad > 0 && e.Status == 0 {
			e.Status = 1
		}
		e.SelfParse()
		keys = append(keys, e.ID)
	}

	// Obtiene el stock actual
	currentStock := []s.AlmacenProducto{}
	err = core.DBSelect(&currentStock).Where("id").In(keys).Exec()
	if err != nil {
		return req.MakeErr("Error al obtener el stock previo:", err)
	}

	currentStockMap := core.SliceToMapK(currentStock,
		func(e s.AlmacenProducto) string { return e.ID })

	//Genera los movimientos correspondientes al stock actual
	movimientos := []s.AlmacenMovimiento{}
	uuid := core.SunixTimeUUIDx3()

	for _, e := range stock {
		movimiento := s.AlmacenMovimiento{
			ID:         core.SunixUUIDx3FromID(e.AlmacenID, uuid),
			EmpresaID:  req.Usuario.EmpresaID,
			ProductoID: e.ProductoID,
			SKU:        e.SKU,
			Lote:       e.Lote,
			Created:    core.SunixTime(),
			CreatedBy:  req.Usuario.ID,
			AlmacenID:  e.AlmacenID,
			Tipo:       core.If(e.Cantidad > 0, int8(1), 2),
		}
		uuid++

		currentCantidad := int32(0)
		if current, ok := currentStockMap[e.ID]; ok {
			currentCantidad = current.Cantidad
		}
		movimiento.Cantidad = e.Cantidad - currentCantidad
		movimiento.AlmacenCantidad = currentCantidad + movimiento.Cantidad
		core.Print(movimiento)
		movimientos = append(movimientos, movimiento)
	}

	statements := slices.Concat(
		core.MakeInsertQuery(&movimientos), core.MakeInsertQuery(&stock))
	core.Print(statements)

	err = core.ExecuteStatements(statements)
	if err != nil {
		return req.MakeErr("Error al obtener el stock previo:", err)
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

	err := core.DBSelect(&result.Movimientos, "id", "empresa_id").
		Where("empresa_id").Equals(req.Usuario.EmpresaID).
		Where("id").GreatEq(core.SunixUUIDx3FromID(almacenID, int64(fechaHoraInicio)*1e6)).
		Where("id").LessEq(core.SunixUUIDx3FromID(almacenID+1, int64(0))).
		OrderDescending().Limit(1000).Exec()

	if err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	err = core.DBSelect(&result.Movimientos, "id", "empresa_id").
		Where("empresa_id").Equals(req.Usuario.EmpresaID).
		Where("almacen_ref_id", "created").GreatEq(almacenID, fechaHoraInicio).
		Where("almacen_ref_id", "created").LessEq(almacenID, fechaHoraFin).
		OrderDescending().Limit(1000).Exec()

	if err != nil {
		return req.MakeErr("Error al obtener los registros del almacén (2):", err)
	}

	core.Log("movimientos encontrados:", len(result.Movimientos))

	if len(result.Movimientos) == 0 {
		return req.MakeResponse(result)
	}

	usuariosSet := core.SliceSet[int32]{}
	productosSet := core.SliceSet[int32]{}

	for _, e := range result.Movimientos {
		usuariosSet.Add(e.CreatedBy)
		productosSet.Add(e.ProductoID)
	}

	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		err := core.DBSelect(&result.Productos).
			Columns("id", "nombre", "precio").
			Where("empresa_id").Equals(req.Usuario.EmpresaID).
			Where("id").In(core.ToAny(productosSet.Values)).Exec()

		if err != nil {
			err = core.Err("Error al obtener los productos:", err)
		}
		return err
	})

	errGroup.Go(func() error {
		err := core.DBSelect(&result.Usuarios).
			Columns("id", "usuario", "nombres", "apellidos").
			Where("empresa_id").Equals(req.Usuario.EmpresaID).
			Where("id").In(core.ToAny(usuariosSet.Values)).Exec()

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
	updated := core.UnixToSunix(req.GetQueryInt64("upd"))

	almacenProductos := []s.AlmacenProducto{}

	query := core.DBSelect(&almacenProductos).
		Where("empresa_id").Equals(req.Usuario.EmpresaID)

	if updated > 0 {
		query = query.
			Where("almacen_id", "updated").GreatEq(almacenID, updated).
			Where("almacen_id", "updated").LessThan(almacenID+1, 0)
	} else {
		query = query.
			Where("id").GreatThan(core.Concat62(almacenID, 0)).
			Where("id").LessThan(core.Concat62(almacenID+1, 0))
	}

	core.Log("ejecutando query: 1")
	err := query.Exec()
	if err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	return req.MakeResponse(almacenProductos)
}
