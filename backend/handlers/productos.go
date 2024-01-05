package handlers

import (
	"app/core"
	s "app/types"
	"fmt"

	"golang.org/x/sync/errgroup"
)

func GetProductos(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")

	productos := []s.Producto{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := core.DBSelect(&productos).
			Where("empresa_id").Equals(req.Usuario.EmpresaID)

		if updated > 0 {
			query = query.Where("updated").GreatEq(updated)
		} else {
			query = query.Where("status").Equals(1)
		}
		err := query.Exec()
		if err != nil {
			err = fmt.Errorf("error al obtener los productos: %v", err)
		}
		return err
	})

	err := errGroup.Wait()
	if err != nil {
		return req.MakeErr(err)
	}

	return core.MakeResponse(req, &productos)
}
