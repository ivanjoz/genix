package finance

import (
	"app/core"
	"github.com/ivanjoz/genix-orm/scylla"
	financeTypes "app/finance/types"
)

func GetCaja(companyID, cashBankID int32) (financeTypes.CashBank, error) {
	cajas := []financeTypes.CashBank{}
	query := scylla.Query(&cajas)
	query.Select().
		CompanyID.Equals(companyID).
		ID.Equals(cashBankID)

	if err := query.Exec(); err != nil {
		return financeTypes.CashBank{}, core.Err("Error al obtener información de la cashBank:", err)
	}
	if len(cajas) == 0 {
		return financeTypes.CashBank{}, core.Err("No se encontró la cashBank")
	}
	return cajas[0], nil
}
