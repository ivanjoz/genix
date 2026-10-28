package types

import "app/db"

// Collections of the record columns of the sale frames, as FrameSQL's resolvers receive them: the
// agent resolves "products" by the product text search, "clients" by the ClientProvider one.
const (
	FrameSQLCollectionProducts = "products"
	FrameSQLCollectionClients  = "clients"
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
	return []db.SQLFrame{{
		Name: "sale-summary", Keys: []db.SQLColumn{dateColumn}, Rows: []db.SQLColumn{productColumn, divisorColumn}, Sums: sums,
		Source: db.FrameSource[SaleOrder](companyID, "sale-summary"),
	}, {
		Name: "sale-client-product", Keys: []db.SQLColumn{dateColumn}, Rows: []db.SQLColumn{clientColumn, productColumn, divisorColumn}, Sums: sums,
		Source: db.FrameSource[SaleOrder](companyID, "sale-client-product"),
	}}
}
