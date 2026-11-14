// The exchange rate a till line priced in the other currency is converted with. The currency rules
// themselves are in sales/types (ValidateSaleCurrency, ResolveSaleExchangeRate).

package sales

import (
	"app/cloud"
	"app/core"
	"app/sales/types"
)

// resolveSaleExchangeRate is types.ResolveSaleExchangeRate with the company's flags loaded.
func resolveSaleExchangeRate(companyID int32, saleCurrency int8) (int32, error) {
	companyConfig, err := cloud.LoadCompanyConfig(companyID)
	if err != nil {
		return 0, core.Err("Error al leer la configuración de la empresa:", err)
	}
	return types.ResolveSaleExchangeRate(companyID, saleCurrency, companyConfig.Company.FlagValues)
}
