package seguridad

import (
	"app/core"
	coretypes "app/core/types"
	"app/db"
)

func GetUsuariosList(empresaID int32, usuariosIDs []int32) ([]coretypes.Usuario, error) {
	ids := core.MakeSliceInclude(usuariosIDs)

	if len(usuariosIDs) == 0 {
		return []coretypes.Usuario{}, nil
	}

	usuarios := []coretypes.Usuario{}
	query := db.Query(&usuarios)
	query.Select().
		EmpresaID.Equals(empresaID).
		ID.In(ids.Values...)

	if err := query.Exec(); err != nil {
		return nil, err
	}
	return usuarios, nil
}
