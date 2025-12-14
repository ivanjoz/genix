package shared

import (
	"app/core"
	"app/db2"
	"app/types"
	s "app/types"
)

func GetUsuarios(empresaID int32, usuariosIDs []int32) ([]types.Usuario, error) {
	ids := core.MakeSliceInclude(usuariosIDs)

	if len(usuariosIDs) == 0 {
		return []s.Usuario{}, nil
	}

	usuarios := []s.Usuario{}
	query := db2.Query(&usuarios)
	query.Select().
		EmpresaID.Equals(empresaID).
		ID.In(ids.Values...)

	if err := query.Exec(); err != nil {
		return nil, err
	}
	return usuarios, nil
}

func GetCaja(empresaID, cajaID int32) (s.Caja, error) {

	cajas := []s.Caja{}
	query := db2.Query(&cajas)
	query.Select().
		EmpresaID.Equals(empresaID).
		ID.Equals(cajaID)

	if err := query.Exec(); err != nil {
		return s.Caja{}, core.Err("Error al obtener información de la caja:", err)
	}
	return cajas[0], nil
}
