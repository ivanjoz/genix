package handlers

import (
	"app/core"
	"app/db2"
	s "app/types"
)

func GetProductosStock(req *core.HandlerArgs) core.HandlerResponse {
	almacenID := req.GetQueryInt("almacen-id")
	updated := req.GetQueryInt("upd")

	almacenProductos := []s.AlmacenProducto{}
	query := db2.Query(&almacenProductos)
	query.Select().EmpresaID.Equals(req.Usuario.EmpresaID)

	if updated > 0 {
		query.AlmacenID.Equals(int32(almacenID)).
			Updated.Between(int32(updated), int32(0))
	} else {
		query.ID.Between(
			core.Concat62(almacenID, 0), core.Concat62(almacenID+1, 0))
	}

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", err)
	}

	return req.MakeResponse(almacenProductos)
}
