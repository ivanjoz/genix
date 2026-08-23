package finance

import (
	"app/core"
	"app/db"
	"app/finance/types"
)

func GetCaja(companyID, cashBankID int32) (types.CashBank, error) {
	cajas := []types.CashBank{}
	query := db.Query(&cajas)
	query.Select().
		CompanyID.Equals(companyID).
		ID.Equals(cashBankID)

	if err := query.Exec(); err != nil {
		return types.CashBank{}, core.Err("Error al obtener información de la cashBank:", err)
	}
	if len(cajas) == 0 {
		return types.CashBank{}, core.Err("No se encontró la cashBank")
	}
	return cajas[0], nil
}
