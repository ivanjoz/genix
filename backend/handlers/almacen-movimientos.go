package handlers

import (
	"app/core"
	"app/db"
	s "app/types"
)

func GetProductosStock(req *core.HandlerArgs) core.HandlerResponse {
	almacenID := req.GetQueryInt("almacen-id")
	updated := req.GetQueryInt("upd")

	almacenProductos := db.Select(func(q *db.Query[s.AlmacenProducto], col s.AlmacenProducto) {
		q.Where(col.EmpresaID_().Equals(req.Usuario.EmpresaID))

		if updated > 0 {
			q.Between(col.AlmacenID_().Equals(almacenID), col.Updated_().Equals(updated)).
				And(col.AlmacenID_().Equals(almacenID+1), col.Updated_().Equals(0))
		} else {
			q.Where(col.ID_().Between(
				core.Concat62(almacenID, 0), core.Concat62(almacenID+1, 0)))
		}
	})

	if almacenProductos.Err != nil {
		return req.MakeErr("Error al obtener los registros del almacén:", almacenProductos.Err)
	}

	return req.MakeResponse(almacenProductos.Records)
}
