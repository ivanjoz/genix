package finanzas

import (
	"app/core"
	"app/db"
	finanzasTypes "app/finanzas/types"
)

func GetCaja(empresaID, cajaID int32) (finanzasTypes.Caja, error) {
	cajas := []finanzasTypes.Caja{}
	query := db.Query(&cajas)
	query.Select().
		EmpresaID.Equals(empresaID).
		ID.Equals(cajaID)

	if err := query.Exec(); err != nil {
		return finanzasTypes.Caja{}, core.Err("Error al obtener información de la caja:", err)
	}
	if len(cajas) == 0 {
		return finanzasTypes.Caja{}, core.Err("No se encontró la caja")
	}
	return cajas[0], nil
}
