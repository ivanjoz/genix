// The currency half of creating a sale: which currency it is charged in, and the
// exchange rate a line priced in the other currency is converted with.
//
// The till previews the same rules, but the prices, the total and the cash movement
// are settled here — a client that sent its own rate could book a sale at any price.

package sales

import (
	"app/cloud"
	"app/core"
	finance "app/finance/types"
	"app/sales/types"
	"fmt"
	"math"
	"slices"
)

// companyFlagExchangeRateSpread is the valued company flag (company_flags.toml) holding the
// spread added to the sell rate, typed with 3 decimals — the same scale as a stored rate.
const companyFlagExchangeRateSpread = int16(7)

// normalizeCurrency reads a currency code that was never set as PEN: products and sales
// created before currencies were tracked carry a 0, and every one of them was in soles.
func normalizeCurrency(currencyType int8) int8 {
	if currencyType == 0 {
		return finance.CurrencyPEN
	}
	return currencyType
}

func currencyName(currencyType int8) string {
	if currencyType == finance.CurrencyUSD {
		return "dólares"
	}
	return "soles"
}

// resolveSaleOrderCurrency checks the sale's currency. A payment is booked into a cash-bank
// in that cash-bank's currency, so a paid sale must be in the same one: the till sends the
// cash-bank's currency, and a mismatch means it priced the sale in the other.
func resolveSaleOrderCurrency(req *core.HandlerArgs, sale *types.SaleOrder) error {
	sale.CurrencyType = normalizeCurrency(sale.CurrencyType)
	if sale.CurrencyType != finance.CurrencyPEN && sale.CurrencyType != finance.CurrencyUSD {
		return core.Err("La moneda de la venta no es válida (1 = soles, 2 = dólares).")
	}

	if !slices.Contains(sale.ActionsIncluded, 2) || sale.LastPaymentCajaID == 0 {
		return nil
	}
	cashBank, err := finance.GetCaja(req.User.CompanyID, sale.LastPaymentCajaID)
	if err != nil {
		return err
	}
	if cashBankCurrency := normalizeCurrency(cashBank.CurrencyType); cashBankCurrency != sale.CurrencyType {
		return core.Err(fmt.Sprintf(`La venta está en %v pero la caja "%v" es en %v.`,
			currencyName(sale.CurrencyType), cashBank.Name, currencyName(cashBankCurrency)))
	}
	return nil
}

// resolveSaleExchangeRate is the effective rate (× 1000) lines are converted into the sale's
// currency with: the company's latest sell rate, moved by its spread in the store's favour.
func resolveSaleExchangeRate(companyID int32, saleCurrency int8) (int32, error) {
	sellRate, err := finance.LatestSellRate(companyID, core.FechaUnix(), finance.ExchangeRateMaxAgeDays)
	if err != nil {
		return 0, err
	}

	companyConfig, err := cloud.LoadCompanyConfig(companyID)
	if err != nil {
		return 0, core.Err("Error al leer la configuración de la empresa:", err)
	}
	// A company that never typed a spread converts at the plain sell rate.
	spreadValue, _ := core.GetCompanyFlagValue(companyConfig.Company.FlagValues, companyFlagExchangeRateSpread)
	spread := int32(math.Round(spreadValue * finance.ExchangeRateScale))

	return finance.EffectiveExchangeRate(sellRate, spread, saleCurrency)
}
