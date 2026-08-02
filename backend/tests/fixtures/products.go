package fixtures

// Package fixtures holds shared benchmark payloads.
//
// MakeProducts: a products-list response shaped like what GET.products actually
// returns (business/products.go:43 → core.MakeResponse(req, &productos)).
//
// The real businessTypes.Product is used rather than a mirror struct so the benchmark cannot
// drift from production: 35+ fields, almost all `omitempty`, three nested slice types, and an
// embedded db.TableStruct whose only exported field is tagged `json:"-"`.
//
// This lives in its own package rather than in a _test.go file so both the serialize-level and
// the core-level benchmarks can share one definition. Importers must be external test packages
// (serialize_test, core_test) because business/types depends on app/core → app/serialize.

import (
	businessTypes "app/business/types"
	"fmt"
)

// MakeProducts builds `count` products with a realistic sparse-field distribution.
// GET.products excludes Stock, StockStatus, CompanyID, Created, CreatedBy and NameHash
// (business/products.go:27), so those stay zero here too.
func MakeProducts(count int) []businessTypes.Product {
	products := make([]businessTypes.Product, 0, count)

	for i := range count {
		product := businessTypes.Product{
			ID:             int32(i + 1),
			Name:           fmt.Sprintf("Producto de prueba %d con nombre largo", i),
			SKU:            fmt.Sprintf("SKU-%06d", i),
			Price:          int32(1000 + i*7),
			FinalPrice:     int32(900 + i*7),
			CurrencyID:     1,
			UnitID:         2,
			Status:         1,
			Updated:        int32(700000 + i),
			UpdatedBy:      3,
			UpdatedVersion: int32(i + 1),
			CategoryIDs:    []int32{int32(i%12 + 1), int32(i%5 + 20)},
			BrandID:        int32(i%40 + 1),
		}

		// Roughly a third of a real catalogue carries a description and rich content.
		if i%3 == 0 {
			product.Description = "Descripción del producto con detalle suficiente para pesar en el payload."
			product.ContentHTML = "<p>Contenido HTML del producto</p><ul><li>Punto uno</li><li>Punto dos</li></ul>"
		}

		// Discounts are the exception, not the rule.
		if i%4 == 0 {
			product.Discount = 10.5
			product.SbuQuantity = 6
			product.SbuUnit = "caja"
			product.SbuPrice = int32(5000 + i)
			product.SbuFinalPrice = int32(4500 + i)
			product.SbuDiscount = 5.25
		}

		// Most products have images.
		if i%5 != 0 {
			product.ImageMain = int32(i*10 + 1)
			product.ImageIDs = []int32{int32(i*10 + 1), int32(i*10 + 2), int32(i*10 + 3)}
			product.ImageDescriptions = []string{"frontal", "lateral", "empaque"}
		}

		// Variants only exist on part of the catalogue, and vary in length.
		if i%2 == 0 {
			presentationCount := i%3 + 1
			for p := range presentationCount {
				product.Presentations = append(product.Presentations, businessTypes.ProductPresentation{
					ID:              int16(p + 1),
					AtributoID:      int16(p%2 + 1),
					Name:            fmt.Sprintf("Presentación %d", p+1),
					Color:           "#3366cc",
					Price:           int32(1200 + p*50),
					PriceDifference: int32(p * 50),
					SKU:             fmt.Sprintf("SKU-%06d-%d", i, p),
					Status:          1,
				})
			}
		}

		if i%6 == 0 {
			product.Properties = []businessTypes.ProductProperties{{
				ID:     1,
				Name:   "Talla",
				Status: 1,
				Options: []businessTypes.ProductProperty{
					{ID: 1, Name: "S", Status: 1},
					{ID: 2, Name: "M", Status: 1},
					{ID: 3, Name: "L", Status: 1},
				},
			}}
		}

		products = append(products, product)
	}

	return products
}
