package operaciones

import (
	"app/core"
	s "app/types"
	"encoding/json"
	"time"
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
			AlmacenID:  e.AlmacenID,
			ProductoID: e.ProductoID,
			SKU:        e.SKU,
			Lote:       e.Lote,
			Cantidad:   e.Cantidad,
			Tipo:       1, // Movimiento Manual
			Created:    nowTime,
		}
		if current, ok := currentStockMap[e.ID]; ok {
			movimiento.AlmacenCantidad = current.Cantidad
			movimiento.Cantidad = e.Cantidad - current.Cantidad
		}
		movimiento.SelfParse()
		movimiento.AssignID()
		movimientos = append(movimientos, movimiento)
	}

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

	almacenMovimientos := []s.AlmacenMovimiento{}
	query := core.DBSelect(&almacenMovimientos).
		Where("empresa_id").Equals(req.Usuario.EmpresaID).
		Where("sk_almacen_created").GreatEq(core.ConcatInt64(almacenID, fechaHoraInicio)).
		Where("sk_almacen_created").LessEq(core.ConcatInt64(almacenID, fechaHoraFin))

	query.Limit = 1000
	err := query.Exec()
	if err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	core.Log("movimientos encontrados:", len(almacenMovimientos))

	return req.MakeResponse(almacenMovimientos)
}
