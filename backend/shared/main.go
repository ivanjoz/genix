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

func GetCaja(empresaID, cajaID int32) (types.Caja, error) {
	cajas := []types.Caja{}
	err := core.DBSelect(&cajas).Where("empresa_id").Equals(empresaID).
		Where("id").Equals(cajaID).Exec()

	if err != nil {
		return types.Caja{}, core.Err("Error al obtener información de la caja:", err)
	}
	return cajas[0], nil
}
