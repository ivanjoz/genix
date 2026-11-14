package dining

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/dining/types"
	finance "app/finance/types"
	logistics "app/logistics/types"
	production "app/production/types"
	sales "app/sales/types"
	"fmt"
	"slices"
)

// diningMenuProduct is one product of a layout's menu, priced in soles.
type diningMenuProduct struct {
	ID          int32
	Name        string
	Price       int32
	SubPrice    int32
	SubDivisor  int16
	CategoryIDs []int32
	// StationID is the station its lines would go to now.
	StationID     int16
	Presentations []diningMenuPresentation
}

// diningMenuPresentation is one sellable presentation and how many units can still be ordered.
type diningMenuPresentation struct {
	ID       int16
	Name     string
	Sellable int32
}

// GetDiningMenu answers the staff's menu of a layout (?layout-id, docs/DINING_PLAN.md §3): the
// products the web store sells (active, not hidden) that can be ordered now, priced in soles, with
// what the web store's sellable rule leaves of each presentation on the layout's warehouse alone.
// A round is validated with the same rule, so the menu and the order cannot disagree.
func GetDiningMenu(req *core.HandlerArgs) core.HandlerResponse {
	companyID := req.User.CompanyID
	layout, err := readActiveLayout(companyID, int32(req.GetQueryInt("layout-id")))
	if err != nil {
		return req.MakeErr(err)
	}
	products, err := readDiningProducts(companyID, nil)
	if err != nil {
		return req.MakeErr(err)
	}
	exchangeRate, err := resolveDiningExchangeRate(companyID, products)
	if err != nil {
		return req.MakeErr(err)
	}

	// The warehouse's stock rows. A row with Status 0 holds nothing and reserves nothing, so it
	// counts as missing.
	stockRows := []logistics.ProductStock{}
	query := db.Query(&stockRows)
	query.Select(query.ProductID, query.PresentationID, query.Quantity, query.DetailQuantity, query.BlockedQuantity,
		query.PendingDeliveryStore, query.PendingDeliveryOnline).
		CompanyID.Equals(companyID).WarehouseID.Equals(layout.WarehouseID).Status.Equals(1)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el stock del salón:", err)
	}
	stockRowsByProduct := map[int32][]logistics.ProductStock{}
	for _, stockRow := range stockRows {
		stockRowsByProduct[stockRow.ProductID] = append(stockRowsByProduct[stockRow.ProductID], stockRow)
	}

	menu := make([]diningMenuProduct, 0, len(products))
	for _, product := range products {
		pricedLine := sales.SaleOrderLine{Quantity: core.QuantityLineScale}
		if err := sales.PriceSaleOrderLine(&pricedLine, product, finance.CurrencyPEN, exchangeRate); err != nil {
			return req.MakeErr(err)
		}
		presentationIDs := product.WebPresentationIDs()
		sellableByPresentation := logistics.WebSellableStock(stockRowsByProduct[product.ID], presentationIDs, product.WebExtraStock)
		// Only what can be ordered now: a round of it would be refused.
		if !slices.ContainsFunc(presentationIDs, func(presentationID int16) bool { return sellableByPresentation[presentationID] > 0 }) {
			continue
		}
		menuProduct := diningMenuProduct{
			ID: product.ID, Name: product.Name, Price: pricedLine.Price, SubPrice: pricedLine.SubPrice,
			SubDivisor: pricedLine.SubDivisor, CategoryIDs: product.CategoryIDs,
			StationID: types.StationForProduct(layout.Stations, product.ID, product.CategoryIDs),
		}
		for _, presentationID := range presentationIDs {
			menuProduct.Presentations = append(menuProduct.Presentations, diningMenuPresentation{
				ID: presentationID, Name: presentationName(product, presentationID), Sellable: sellableByPresentation[presentationID],
			})
		}
		menu = append(menu, menuProduct)
	}
	return req.MakeResponse(map[string]any{"Products": menu})
}

func presentationName(product production.Product, presentationID int16) string {
	for _, presentation := range product.Presentations {
		if presentation.ID == presentationID {
			return presentation.Name
		}
	}
	return ""
}

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

// readDiningProducts reads the menu's products: active and not hidden from the web store. With
// productIDs, only those (a round); with nil, the whole catalog (the menu).
func readDiningProducts(companyID int32, productIDs []int32) ([]production.Product, error) {
	products := []production.Product{}
	query := db.Query(&products)
	query.Select(query.ID, query.Name, query.FinalPrice, query.SbuQuantity, query.SbuFinalPrice, query.CurrencyID,
		query.WebExtraStock, query.WebHidden, query.Presentations, query.CategoryIDs, query.Status).
		CompanyID.Equals(companyID)
	if productIDs != nil {
		query.ID.In(productIDs...)
	} else {
		query.Status.Equals(production.ProductStatusActive)
	}
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los productos del menú:", err)
	}
	menuProducts := make([]production.Product, 0, len(products))
	for _, product := range products {
		if product.Status == production.ProductStatusActive && !product.WebHidden {
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
