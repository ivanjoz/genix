package bench

import (
	"app/db"

	logistics "app/logistics/types"
	production "app/production/types"
	sales "app/sales/types"
)

// ScanOnly runs the same ORM query each endpoint's handler runs on a first sync, and stops there:
// no colbin, no compression, no router. The gap between this and the handler benchmark is what
// serialization costs. Keep each query in step with its handler (production/products.go,
// sales/sale_orders_status.go, logistics/product-stock-movement.go).
var ScanOnly = map[string]func(companyID int32) (int, error){
	"products": func(companyID int32) (int, error) {
		products := []production.Product{}
		query := db.Query(&products).CompanyID.Equals(companyID)
		query.Exclude(query.Stock, query.StockStatus, query.CompanyID, query.Created, query.CreatedBy, query.NameHash)
		query.Delta(db.DeltaCursor{}, production.ProductStatusActive)
		err := query.Exec()
		return len(products), err
	},
	"sale-orders": func(companyID int32) (int, error) {
		orders := []sales.SaleOrder{}
		query := db.Query(&orders).Limit(5000).OrderDesc()
		query.Exclude(query.CompanyID)
		query.CompanyID.Equals(companyID).Delta(db.DeltaCursor{}, sales.OrderStatusPending, sales.OrderStatusDelivered)
		err := query.Exec()
		return len(orders), err
	},
	"products-stock": func(companyID int32) (int, error) {
		stockRows := []logistics.ProductStock{}
		query := db.Query(&stockRows)
		query.Select().CompanyID.Equals(companyID).Delta(db.DeltaCursor{}, 1)
		err := query.Exec()
		return len(stockRows), err
	},
}
