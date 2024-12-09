package shared

import (
	"app/core"
	"app/db"
	"app/types"
	s "app/types"
)

func GetUsuarios(empresaID int32, usuariosIDs []int32) ([]types.Usuario, error) {
	ids := core.MakeSliceInclude(usuariosIDs)

	if len(usuariosIDs) == 0 {
		return []s.Usuario{}, nil
	}

	usuarios := db.Select(func(q *db.Query[s.Usuario], col s.Usuario) {
		q.Where(col.EmpresaID_().Equals(empresaID)).
			Where(col.ID_().In(ids.Values...))
	})

	if usuarios.Err != nil {
		return nil, usuarios.Err
	}
	return usuarios.Records, nil
}

func GetCaja(empresaID, cajaID int32) (s.Caja, error) {

	cajas := db.Select(func(q *db.Query[s.Caja], col s.Caja) {
		q.Where(col.EmpresaID_().Equals(empresaID)).
			Where(col.ID_().Equals(cajaID))
	})

	if cajas.Err != nil {
		return s.Caja{}, core.Err("Error al obtener información de la caja:", cajas.Err)
	}
	return cajas.Records[0], nil
}
