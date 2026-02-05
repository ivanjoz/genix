package handlers

import (
	"app/core"
	"app/db"
	s "app/types"
)

func GetProductosStock(req *core.HandlerArgs) core.HandlerResponse {
	almacenID := req.GetQueryInt("almacen-id")
	updated := req.GetQueryInt("upd")

	almacenProductos := []s.AlmacenProducto{}
	query := db.Query(&almacenProductos)
	query.Select().EmpresaID.Equals(req.Usuario.EmpresaID).AlmacenID.Equals(almacenID)

	if updated > 0 {
		query.Updated.GreaterThan(updated)
	} else {
		query.Status.Equals(0)
	}

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	return req.MakeResponse(almacenProductos)
}
