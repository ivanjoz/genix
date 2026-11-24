package types

import (
	"fmt"

	"app/core"
	"app/db"
	production "app/production/types"
)

// ReadSaleProducts loads the catalog rows of a sale's products: the prices and divisor the till
// prices its lines with, whether each one is made or bought, its sellable rule, the categories its
// packs match on, and the names and sub-unit labels the sale's document describes its lines with.
func ReadSaleProducts(companyID int32, lines []SaleOrderLine) (map[int32]production.Product, error) {
	productIDs := core.SliceSet[int32]{}
	for _, line := range lines {
		productIDs.AddIf(line.ProductID)
	}
	productByID := map[int32]production.Product{}
	if productIDs.IsEmpty() {
		return productByID, nil
	}
	products := []production.Product{}
	query := db.Query(&products)
	query.Select(query.ID, query.Name, query.FinalPrice, query.SbuQuantity, query.SbuFinalPrice,
		query.CurrencyID, query.SbuUnit, query.WebExtraStock, query.SalesChannels, query.Presentations, query.Origin, query.CategoryIDs,
		query.TaxGroup, query.DetractionCode).
		CompanyID.Equals(companyID).
		ID.In(productIDs.Values...)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los productos de la venta:", err)
	}
	for _, product := range products {
		productByID[product.ID] = product
	}
	return productByID, nil
}

// ValidateSaleProductsChannel refuses a sale holding a product not sold in the channel it is made
// from (Product.SalesChannels): the channel's list leaves it out, but a request can still name it.
func ValidateSaleProductsChannel(lines []SaleOrderLine, productByID map[int32]production.Product, channel int8) error {
	for _, line := range lines {
		product, found := productByID[line.ProductID]
		if found && !product.IsSoldIn(channel) {
			return core.Err(fmt.Sprintf(`El producto "%v" no se vende en este canal (ver "Mostrar en" del producto).`, product.Name))
		}
	}
	return nil
}

// ReadActiveProductPacks are the company's active packs, read once per sale for ApplySalePacks.
func ReadActiveProductPacks(companyID int32) ([]production.ProductPack, error) {
	packs := []production.ProductPack{}
	query := db.Query(&packs)
	query.Select(query.ID, query.Components, query.PriceType, query.PriceValue, query.RoundingMode, query.Status).
		CompanyID.Equals(companyID).Status.Equals(1)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los packs de la empresa:", err)
	}
	return packs, nil
}

// MadeProductIDsOf are the products the company makes (Origin = 2): their lines are delivered
// without a stock exit, which waits for the daily production close.
func MadeProductIDsOf(productByID map[int32]production.Product) map[int32]bool {
	madeProductIDs := map[int32]bool{}
	for productID, product := range productByID {
		if product.Origin == production.ProductOriginMade {
			madeProductIDs[productID] = true
		}
	}
	return madeProductIDs
}
