package types

import (
	"app/db"
	production "app/production/types"
)

// Collections of the record columns of the sale frames, as FrameSQL's resolvers receive them: the
// agent resolves "products" by the product text search, "clients" by the ClientProvider one,
// "categories" by the product categories' shared list.
const (
	FrameSQLCollectionProducts   = "products"
	FrameSQLCollectionClients    = "clients"
	FrameSQLCollectionCategories = "categories"
)

// SaleFrameSQLFrames are the sale DataFrames of one company as FrameSQL reads them: the columns of
// SaleSummaryFrame and SaleClientProductFrame, in their order. A statement sees whole-unit quantities
// only: the sub-unit divisor and the split units/sub sums stay hidden, because sub-units of two
// divisors don't add up (4 sixths + 4 twelfths is not 8 of anything).
func SaleFrameSQLFrames(companyID int32) []db.SQLFrame {
	dateColumn := db.SQLColumn{Name: "date", Label: "Date|Fecha", Kind: db.SQLDay}
	productColumn := db.SQLColumn{Name: "product_id", Label: "Product|Producto", Kind: db.SQLRef, Collection: FrameSQLCollectionProducts}
	clientColumn := db.SQLColumn{Name: "client_id", Label: "Client|Cliente", Kind: db.SQLRef, Collection: FrameSQLCollectionClients}
	divisorColumn := db.SQLColumn{Name: "sub_divisor", Hidden: true}
	sums := []db.SQLColumn{
		{Name: "amount", Label: "Amount|Monto", Kind: db.SQLCents},
		{Name: "debt_amount", Label: "Unpaid|Por cobrar", Kind: db.SQLCents},
		{Name: "sold_units", Hidden: true},
		{Name: "sold_sub", Hidden: true},
		{Name: "pending_units", Hidden: true},
		{Name: "pending_sub", Hidden: true},
		{Name: "quantity", Label: "Quantity|Cantidad", Kind: db.SQLQuantity},
		{Name: "pending_quantity", Label: "Undelivered|Por entregar", Kind: db.SQLQuantity},
	}
	saleSummary := db.SQLFrame{
		Name: "sale-summary", Keys: []db.SQLColumn{dateColumn}, Rows: []db.SQLColumn{productColumn, divisorColumn}, Sums: sums,
		Source: db.FrameSource[SaleOrder](companyID, "sale-summary"),
	}
	saleClientProduct := db.SQLFrame{
		Name: "sale-client-product", Keys: []db.SQLColumn{dateColumn}, Rows: []db.SQLColumn{clientColumn, productColumn, divisorColumn}, Sums: sums,
		Source: db.FrameSource[SaleOrder](companyID, "sale-client-product"),
	}
	// The category views read each product's current categories (ProductCategoryFrame), so a product
	// moved to another category moves its past sales too. A product without one is category 0.
	productCategories := db.SQLDictionary{
		Rows:   []string{"category_id", "product_id"},
		On:     "product_id",
		Column: db.SQLColumn{Name: "category_id", Label: "Category|Categoría", Kind: db.SQLRef, Collection: FrameSQLCollectionCategories},
		// The company's whole frame is its one file, FileBucket 0.
		Selection: db.SQLSelection{FromKey: 0, ToKey: 0},
		Source:    db.FrameSource[production.Product](companyID, "product-categories"),
	}
	return []db.SQLFrame{
		saleSummary,
		saleClientProduct,
		db.SQLView("sale-category", saleSummary, productCategories),
		db.SQLView("sale-client-category", saleClientProduct, productCategories),
	}
}
