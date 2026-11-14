// The currency and catalog pricing rules every sale-like order shares: the till's sale, the web
// store's order and the restaurant's table order. They live here so the dining module prices a
// round exactly as the till prices a line, without importing the sales body.

package types

import (
	"app/core"
	finance "app/finance/types"
	production "app/production/types"
	"fmt"
	"math"
	"slices"
)

// CurrencyName is how an error names a sale's currency.
func CurrencyName(currencyType int8) string {
	if currencyType == finance.CurrencyUSD {
		return "dólares"
	}
	return "soles"
}

// ValidateSaleCurrency checks the sale's currency. A payment is booked into a cash-bank in that
// cash-bank's currency, so a paid sale must be in the same one: the till sends the cash-bank's
// currency, and a mismatch means it priced the sale in the other.
func ValidateSaleCurrency(companyID int32, sale *SaleOrder) error {
	sale.CurrencyType = NormalizeCurrency(sale.CurrencyType)
	if sale.CurrencyType != finance.CurrencyPEN && sale.CurrencyType != finance.CurrencyUSD {
		return core.Err("La moneda de la venta no es válida (1 = soles, 2 = dólares).")
	}

	if !slices.Contains(sale.ActionsIncluded, 2) || sale.LastPaymentCajaID == 0 {
		return nil
	}
	cashBank, err := finance.GetCaja(companyID, sale.LastPaymentCajaID)
	if err != nil {
		return err
	}
	if cashBankCurrency := NormalizeCurrency(cashBank.CurrencyType); cashBankCurrency != sale.CurrencyType {
		return core.Err(fmt.Sprintf(`La venta está en %v pero la caja "%v" es en %v.`,
			CurrencyName(sale.CurrencyType), cashBank.Name, CurrencyName(cashBankCurrency)))
	}
	return nil
}

// CompanyFlagExchangeRateSpread is the valued company flag (company_flags.toml) holding the
// spread added to the sell rate, typed with 3 decimals — the same scale as a stored rate.
const CompanyFlagExchangeRateSpread = int16(7)

// NormalizeCurrency reads a currency code that was never set as PEN: products and sales created
// before currencies were tracked carry a 0, and every one of them was in soles.
func NormalizeCurrency(currencyType int8) int8 {
	if currencyType == 0 {
		return finance.CurrencyPEN
	}
	return currencyType
}

// ResolveSaleExchangeRate is the effective rate (× 1000) lines are converted into the sale's
// currency with: the company's latest sell rate, moved by its spread in the store's favour.
// companyFlagValues are the company's (cloud.LoadCompanyConfig): cloud imports this package, so
// the caller loads them.
func ResolveSaleExchangeRate(companyID int32, saleCurrency int8, companyFlagValues []core.CompanyFlagValue) (int32, error) {
	sellRate, err := finance.LatestSellRate(companyID, core.FechaUnix(), finance.ExchangeRateMaxAgeDays)
	if err != nil {
		return 0, err
	}
	// A company that never typed a spread converts at the plain sell rate.
	spreadValue, _ := core.GetCompanyFlagValue(companyFlagValues, CompanyFlagExchangeRateSpread)
	spread := int32(math.Round(spreadValue * finance.ExchangeRateScale))

	return finance.EffectiveExchangeRate(sellRate, spread, saleCurrency)
}

// PriceSaleOrderLine resolves one line against its catalog product: whether its sub-unit part is
// allowed, the sub-unit divisor, and the prices restated in the order's currency. Prices always
// come from the catalog, never from the request: a client could otherwise book at any price.
// exchangeRate is required (non-zero) when the product's currency differs from saleCurrency.
func PriceSaleOrderLine(line *SaleOrderLine, product production.Product, saleCurrency int8, exchangeRate int32) error {
	lineQuantity := core.UnpackQuantityLine(line.Quantity)
	productDivisor := product.SbuQuantity
	if productDivisor <= 0 {
		productDivisor = core.QuantityDivisorNone
	}

	if lineQuantity.Sub > 0 {
		if productDivisor <= core.QuantityDivisorNone {
			return core.Err(fmt.Sprintf(
				`El producto "%s" no tiene sub-unidad configurada, pero la venta envía %v sub-unidades.`,
				product.Name, lineQuantity.Sub))
		}
		if product.SbuFinalPrice <= 0 {
			return core.Err(fmt.Sprintf(
				`El producto "%s" no tiene precio de sub-unidad; no se puede vender fraccionado.`,
				product.Name))
		}
		// Packing normalizes, so a Sub at or past the divisor means the client packed a value the
		// catalog cannot represent.
		if lineQuantity.Sub >= int32(productDivisor) {
			return core.Err(fmt.Sprintf(
				`El producto "%s" admite hasta %v sub-unidades por unidad; la venta envía %v.`,
				product.Name, productDivisor-1, lineQuantity.Sub))
		}
	}

	productCurrency := NormalizeCurrency(int8(product.CurrencyID))
	line.Price = finance.ConvertUnitPrice(product.FinalPrice, productCurrency, saleCurrency, exchangeRate)
	line.SubPrice = finance.ConvertUnitPrice(product.SbuFinalPrice, productCurrency, saleCurrency, exchangeRate)
	line.SubDivisor = productDivisor
	return nil
}
