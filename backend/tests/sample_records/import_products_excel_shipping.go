//go:build lambda.norpc

package sample_records

import "app/core"

// Shipping builds leave excelize and products.xlsx out (see import_products_excel.go), so the
// supply seed only runs from a local, untagged build.
func (generator *supplyDataGenerator) importProductsWorkbook() (productsImportStats, error) {
	return productsImportStats{}, core.Err("la importación de products.xlsx sólo está disponible en un build local (sin el tag lambda.norpc)")
}
