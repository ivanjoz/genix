package dining

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/dining/types"
	finance "app/finance/types"
	production "app/production/types"
	sales "app/sales/types"
	"fmt"
)

// The reads a round is validated with. The staff's menu is built in the browser from the delta-cached
// catalog and stock (frontend floor-service.ts diningMenu) with these same rules, so the menu and
// the order cannot disagree.

// readActiveLayout reads one active layout of the company.
func readActiveLayout(companyID int32, layoutID int32) (types.DiningLayout, error) {
	if layoutID <= 0 {
		return types.DiningLayout{}, core.Err("Debe indicar el salón.")
	}
	layouts := []types.DiningLayout{}
	query := db.Query(&layouts)
	query.Select().CompanyID.Equals(companyID).ID.Equals(layoutID)
	if err := query.Exec(); err != nil {
		return types.DiningLayout{}, core.Err("Error al obtener el salón:", err)
	}
	if len(layouts) == 0 || layouts[0].Status == 0 {
		return types.DiningLayout{}, core.Err("El salón no existe o fue eliminado.")
	}
	return layouts[0], nil
}

// readDiningProducts reads the round's products that are active and sold in the restaurant.
func readDiningProducts(companyID int32, productIDs []int32) ([]production.Product, error) {
	products := []production.Product{}
	query := db.Query(&products)
	query.Select(query.ID, query.Name, query.FinalPrice, query.SbuQuantity, query.SbuFinalPrice, query.CurrencyID,
		query.WebExtraStock, query.SalesChannels, query.Presentations, query.CategoryIDs, query.Status, query.TaxGroup).
		CompanyID.Equals(companyID).ID.In(productIDs...)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los productos del menú:", err)
	}
	menuProducts := make([]production.Product, 0, len(products))
	for _, product := range products {
		if product.Status == production.ProductStatusActive && product.IsSoldIn(production.ProductChannelRestaurant) {
			menuProducts = append(menuProducts, product)
		}
	}
	return menuProducts, nil
}

// resolveDiningExchangeRate is the rate a USD-priced product is converted into soles with, 0 when
// every product is already in soles (table orders are in soles).
func resolveDiningExchangeRate(companyID int32, products []production.Product) (int32, error) {
	for _, product := range products {
		if sales.NormalizeCurrency(int8(product.CurrencyID)) == finance.CurrencyPEN {
			continue
		}
		companyConfig, err := cloud.LoadCompanyConfig(companyID)
		if err != nil {
			return 0, core.Err("Error al leer la configuración de la empresa:", err)
		}
		exchangeRate, err := sales.ResolveSaleExchangeRate(companyID, finance.CurrencyPEN, companyConfig.Company.FlagValues)
		if err != nil {
			return 0, core.Err(fmt.Sprintf(`"%s" tiene precio en dólares:`, product.Name), err)
		}
		return exchangeRate, nil
	}
	return 0, nil
}
