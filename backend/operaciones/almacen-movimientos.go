package operaciones

import (
	"app/core"
	s "app/types"
	"encoding/json"
	"time"

	"golang.org/x/sync/errgroup"
)

func PostAlmacenStock(req *core.HandlerArgs) core.HandlerResponse {

	stock := []s.AlmacenProducto{}
	nowTime := time.Now().Unix()

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

	for _, e := range stock {
		movimiento := s.AlmacenMovimiento{
			EmpresaID:  req.Usuario.EmpresaID,
			ProductoID: e.ProductoID,
			SKU:        e.SKU,
			Lote:       e.Lote,
			Created:    nowTime,
			CreatedBy:  req.Usuario.ID,
		}

		stockCurrent := int32(0)
		if current, ok := currentStockMap[e.ID]; ok {
			stockCurrent = current.Cantidad
		}
		movimiento.Cantidad = e.Cantidad - stockCurrent

		if movimiento.Cantidad > 0 {
			movimiento.Tipo = 1
			movimiento.AlmacenID = e.AlmacenID
			movimiento.AlmacenCantidad = stockCurrent
		} else {
			movimiento.Tipo = 2
			movimiento.AlmacenOrigenID = e.AlmacenID
			movimiento.AlmacenOrigenCantidad = stockCurrent
		}

		movimiento.SelfParse()
		movimiento.AssignID()
		movimientos = append(movimientos, movimiento)
	}

	core.Print(movimientos)

	core.Log("movimientos a insertar::", len(movimientos), "|", len(currentStock))

	err = core.DBInsert(&movimientos)
	if err != nil {
		return req.MakeErr("Error al insertar movimientos:", err)
	}

	err = core.DBInsert(&stock)
	if err != nil {
		return req.MakeErr("Error al insertar el stock de productos:", err)
	}

	return req.MakeResponse(stock)
}

func GetAlmacenMovimientos(req *core.HandlerArgs) core.HandlerResponse {

	almacenID := req.GetQueryInt64("almacen-id")
	fechaHoraInicio := req.GetQueryInt64("fecha-hora-inicio")
	fechaHoraFin := req.GetQueryInt64("fecha-hora-fin")

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
		Where("sk_almacen_created").GreatEq(core.ConcatInt64(almacenID, fechaHoraInicio)).
		Where("sk_almacen_created").LessEq(core.ConcatInt64(almacenID, fechaHoraFin)).
		OrderDescending().Limit(1000).Exec()

	if err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	err = core.DBSelect(&result.Movimientos, "id", "empresa_id").
		Where("empresa_id").Equals(req.Usuario.EmpresaID).
		Where("sk_almacen_origen_created").GreatEq(core.ConcatInt64(almacenID, fechaHoraInicio)).
		Where("sk_almacen_origen_created").LessEq(core.ConcatInt64(almacenID, fechaHoraFin)).
		OrderDescending().Limit(1000).Exec()

	if err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	core.Log("movimientos encontrados:", len(result.Movimientos))

	if len(result.Movimientos) == 0 {
		return req.MakeResponse(result)
	}

	usuariosSet := core.SliceInclude[int32]{}
	productosSet := core.SliceInclude[int32]{}

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
	updated := req.GetQueryInt64("upd")

	almacenProductos := []s.AlmacenProducto{}

	query := core.DBSelect(&almacenProductos).
		Where("empresa_id").Equals(req.Usuario.EmpresaID)

	if updated > 0 {
		query = query.
			Where("sk_almacen_updated").GreatThan(s.ConcatInt64(int64(almacenID), updated)).
			Where("sk_almacen_updated").LessThan(s.ConcatInt64(int64(almacenID+1), 0))
	} else {
		query = query.
			Where("id").GreatThan(core.Concat62(almacenID, 0)).
			Where("id").LessThan(core.Concat62(almacenID+1, 0))
	}

	err := query.Exec()
	if err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	return req.MakeResponse(almacenProductos)
}
