package handlers

import (
	"app/core"
	s "app/types"
)

func GetProductosStock(req *core.HandlerArgs) core.HandlerResponse {
	almacenID := req.GetQueryInt("almacen-id")
	updated := req.GetQueryInt("upd")

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

	err := query.Exec()
	if err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	return req.MakeResponse(almacenProductos)
}
