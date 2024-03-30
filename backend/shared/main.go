package shared

import (
	"app/core"
	"app/types"
)

func GetUsuarios(empresaID int32, usuariosIDs []int32) ([]types.Usuario, error) {
	ids := core.MakeSliceInclude(usuariosIDs)

	usuarios := []types.Usuario{}
	if len(usuariosIDs) == 0 {
		return usuarios, nil
	}

	err := core.DBSelect(&usuarios).
		Where("empresa_id").Equals(empresaID).Where("id").In(ids.ToAny()).Exec()

	if err != nil {
		return nil, err
	}

	return usuarios, err
}
