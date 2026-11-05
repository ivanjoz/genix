//go:build !lambda.norpc

// Seeds the product catalog from products.xlsx the way the Productos page's Excel import does:
// resolve category and brand names against the shared lists, create the missing ones through
// POST.shared-lists, then create the products through POST.products.
//
// Kept out of shipping builds (every deploy path builds with lambda.norpc): excelize and the
// embedded workbook belong to the local seed run, never to the Lambda/VPS binary. Shipping builds
// get the stub in import_products_excel_shipping.go.
package sample_records

import (
	"app/business"
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	"app/production"
	productionTypes "app/production/types"
	"bytes"
	_ "embed"
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// products.xlsx is tracked with Git LFS. Without `git lfs pull` the embedded bytes are the LFS
// pointer text, not a workbook.
//
//go:embed products.xlsx
var productsWorkbookBytes []byte

// The workbook is the Productos page export: a title row, then the header row.
const productsWorkbookHeaderRowIndex = 1

// Same batch size the Productos page uses when it saves an import.
const productsImportBatchSize = 500

type productImportRow struct {
	name          string
	categoryNames []string
	brandName     string
	price         int32
	discount      float32
	finalPrice    int32
	volume        float32
	weight        float32
}

// importProductsWorkbook creates every workbook product whose name is not already an active
// product. Existing products are left untouched, so a rerun only fills what is missing.
func (generator *supplyDataGenerator) importProductsWorkbook() (productsImportStats, error) {
	stats := productsImportStats{}

	workbookRows, err := readProductsWorkbook()
	if err != nil {
		return stats, err
	}
	stats.WorkbookRows = len(workbookRows)

	activeProductNameHashes, err := loadActiveProductNameHashes()
	if err != nil {
		return stats, err
	}

	// Only rows that become new products take part, and a name repeated in the workbook (after
	// normalization) is created once: PostProducts rejects a payload with duplicate names.
	newProductRows := []productImportRow{}
	for _, workbookRow := range workbookRows {
		nameHash := productNameHash(workbookRow.name)
		if activeProductNameHashes[nameHash] {
			stats.SkippedExisting++
			continue
		}
		activeProductNameHashes[nameHash] = true
		newProductRows = append(newProductRows, workbookRow)
	}

	sharedListIDByHash, err := generator.resolveSharedListNames(newProductRows, &stats)
	if err != nil {
		return stats, err
	}

	products := make([]productionTypes.Product, 0, len(newProductRows))
	for _, productRow := range newProductRows {
		product := productionTypes.Product{
			Name:       productRow.name,
			Price:      productRow.price,
			Discount:   productRow.discount,
			FinalPrice: productRow.finalPrice,
			Volume:     productRow.volume,
			Weight:     productRow.weight,
		}
		for _, categoryName := range productRow.categoryNames {
			product.CategoryIDs = append(product.CategoryIDs,
				sharedListIDByHash[sharedListNameHash(businessTypes.SharedListCategoryID, categoryName)])
		}
		if productRow.brandName != "" {
			product.BrandID = sharedListIDByHash[sharedListNameHash(businessTypes.SharedListBrandID, productRow.brandName)]
		}
		products = append(products, product)
	}

	if generator.config.dryRun {
		stats.Created = len(products)
		core.Log("GenerateSupplyData:: dry-run importación de productos", "porCrear", stats.Created,
			"existentes", stats.SkippedExisting, "categoríasNuevas", stats.NewCategories, "marcasNuevas", stats.NewBrands)
		return stats, nil
	}

	for batchStart := 0; batchStart < len(products); batchStart += productsImportBatchSize {
		batchEnd := minInt(batchStart+productsImportBatchSize, len(products))
		bodyBytes, err := json.Marshal(products[batchStart:batchEnd])
		if err != nil {
			return stats, err
		}
		request := generator.makeRequest("POST.products", nil, string(bodyBytes))
		if response := production.PostProducts(&request); response.StatusCode != 200 {
			return stats, core.Err(response.Error)
		}
		stats.Created += batchEnd - batchStart
		core.Log("GenerateSupplyData:: productos importados", "hasta", batchEnd, "de", len(products))
	}

	return stats, nil
}

// readProductsWorkbook maps the first sheet by header name, so a column reorder in a future export
// does not shift values into the wrong field.
func readProductsWorkbook() ([]productImportRow, error) {
	if bytes.HasPrefix(productsWorkbookBytes, []byte("version https://git-lfs")) {
		return nil, core.Err("products.xlsx es un puntero de Git LFS; ejecuta `git lfs pull` y recompila")
	}

	workbook, err := excelize.OpenReader(bytes.NewReader(productsWorkbookBytes))
	if err != nil {
		return nil, core.Err("no se pudo abrir products.xlsx:", err)
	}
	defer workbook.Close()

	// Raw values: the formatted ones carry the "#,##0.00" thousands separator.
	sheetRows, err := workbook.GetRows(workbook.GetSheetName(0), excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, core.Err("no se pudo leer products.xlsx:", err)
	}
	if len(sheetRows) <= productsWorkbookHeaderRowIndex {
		return nil, core.Err("products.xlsx no tiene fila de cabeceras")
	}

	columnIndexByHeader := map[string]int{}
	for columnIndex, header := range sheetRows[productsWorkbookHeaderRowIndex] {
		columnIndexByHeader[strings.TrimSpace(header)] = columnIndex
	}
	for _, requiredHeader := range []string{"Producto", "Categorías", "Precio", "Precio Final", "Marca"} {
		if _, found := columnIndexByHeader[requiredHeader]; !found {
			return nil, core.Err("products.xlsx no tiene la columna", requiredHeader)
		}
	}

	readCell := func(row []string, header string) string {
		columnIndex, found := columnIndexByHeader[header]
		if !found || columnIndex >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[columnIndex])
	}

	productRows := []productImportRow{}
	for rowIndex := productsWorkbookHeaderRowIndex + 1; rowIndex < len(sheetRows); rowIndex++ {
		sheetRow := sheetRows[rowIndex]
		excelRowNumber := rowIndex + 1

		productName := readCell(sheetRow, "Producto")
		if productName == "" {
			continue
		}

		// These columns need the frontend's option lists (unit, currency) or an existing product
		// (ID). The workbook leaves them empty; fail loudly instead of dropping data if that changes.
		for _, unsupportedHeader := range []string{"ID", "Unidad", "Moneda", "Sub Unidades"} {
			if readCell(sheetRow, unsupportedHeader) != "" {
				return nil, core.Err("fila", excelRowNumber, ": la columna", unsupportedHeader, "no está soportada por el importador")
			}
		}

		numericValues := map[string]float64{}
		for _, numericHeader := range []string{"Precio", "Descuento", "Precio Final", "Volumen", "Peso"} {
			rawValue := readCell(sheetRow, numericHeader)
			if rawValue == "" {
				continue
			}
			parsedValue, parseError := strconv.ParseFloat(rawValue, 64)
			if parseError != nil {
				return nil, core.Err("fila", excelRowNumber, ": la columna", numericHeader, "no es un número:", rawValue)
			}
			numericValues[numericHeader] = parsedValue
		}

		// Same split as the Productos page import: the cell is a comma-separated name list.
		categoryNames := []string{}
		for _, categoryName := range strings.Split(readCell(sheetRow, "Categorías"), ",") {
			if categoryName = strings.TrimSpace(categoryName); categoryName != "" {
				categoryNames = append(categoryNames, categoryName)
			}
		}

		productRows = append(productRows, productImportRow{
			name:          productName,
			categoryNames: categoryNames,
			brandName:     readCell(sheetRow, "Marca"),
			// Prices are decimals in the workbook and cents in the backend.
			price:      int32(math.Round(numericValues["Precio"] * 100)),
			discount:   float32(numericValues["Descuento"]),
			finalPrice: int32(math.Round(numericValues["Precio Final"] * 100)),
			volume:     float32(numericValues["Volumen"]),
			weight:     float32(numericValues["Peso"]),
		})
	}

	return productRows, nil
}

func loadActiveProductNameHashes() (map[int32]bool, error) {
	products := []productionTypes.Product{}
	productQuery := db.Query(&products)
	productQuery.Select(productQuery.NameHash, productQuery.Status).
		CompanyID.Equals(supplyCompanyID)
	if err := productQuery.Exec(); err != nil {
		return nil, core.Err("no se pudieron leer los productos existentes:", err)
	}

	activeNameHashes := map[int32]bool{}
	for _, product := range products {
		if product.Status > 0 {
			activeNameHashes[product.NameHash] = true
		}
	}
	return activeNameHashes, nil
}

// resolveSharedListNames returns the ID of every category and brand the new products name,
// creating the missing ones. Names match by the same hash PostSharedLists dedupes on, so
// "Carrefour" in the workbook reuses an existing "CARREFOUR".
func (generator *supplyDataGenerator) resolveSharedListNames(productRows []productImportRow,
	stats *productsImportStats) (map[int32]int32, error) {

	existingRecords := []businessTypes.SharedListRecord{}
	sharedListQuery := db.Query(&existingRecords)
	sharedListQuery.Select(sharedListQuery.ID, sharedListQuery.NameHash, sharedListQuery.Status).
		CompanyID.Equals(supplyCompanyID).
		ListID.In(businessTypes.SharedListCategoryID, businessTypes.SharedListBrandID)
	if err := sharedListQuery.Exec(); err != nil {
		return nil, core.Err("no se pudieron leer las categorías/marcas existentes:", err)
	}

	sharedListIDByHash := map[int32]int32{}
	for _, existingRecord := range existingRecords {
		if existingRecord.Status > 0 {
			sharedListIDByHash[existingRecord.NameHash] = existingRecord.ID
		}
	}

	// Missing names get a negative temporary ID, like the page's temp records; the response maps
	// each TempID to the persisted ID.
	missingRecords := []businessTypes.SharedListRecord{}
	queueIfMissing := func(listID int32, name string) {
		nameHash := sharedListNameHash(listID, name)
		if _, found := sharedListIDByHash[nameHash]; found {
			return
		}
		temporaryID := int32(-(len(missingRecords) + 1))
		sharedListIDByHash[nameHash] = temporaryID
		missingRecords = append(missingRecords, businessTypes.SharedListRecord{
			ID: temporaryID, ListID: listID, Name: name, Status: 1,
		})
		if listID == businessTypes.SharedListCategoryID {
			stats.NewCategories++
		} else {
			stats.NewBrands++
		}
	}
	for _, productRow := range productRows {
		for _, categoryName := range productRow.categoryNames {
			queueIfMissing(businessTypes.SharedListCategoryID, categoryName)
		}
		if productRow.brandName != "" {
			queueIfMissing(businessTypes.SharedListBrandID, productRow.brandName)
		}
	}

	if generator.config.dryRun || len(missingRecords) == 0 {
		return sharedListIDByHash, nil
	}

	persistedIDByTempID := map[int32]int32{}
	for batchStart := 0; batchStart < len(missingRecords); batchStart += productsImportBatchSize {
		batchEnd := minInt(batchStart+productsImportBatchSize, len(missingRecords))
		bodyBytes, err := json.Marshal(missingRecords[batchStart:batchEnd])
		if err != nil {
			return nil, err
		}
		request := generator.makeRequest("POST.shared-lists", nil, string(bodyBytes))
		tempIDMappings := []businessTypes.NewIDToID{}
		if err := decodeResponse(business.PostSharedLists(&request), &tempIDMappings); err != nil {
			return nil, core.Err("no se pudieron crear las categorías/marcas:", err)
		}
		for _, tempIDMapping := range tempIDMappings {
			persistedIDByTempID[tempIDMapping.TempID] = tempIDMapping.ID
		}
	}

	for nameHash, sharedListID := range sharedListIDByHash {
		if sharedListID >= 0 {
			continue
		}
		persistedID := persistedIDByTempID[sharedListID]
		if persistedID <= 0 {
			return nil, core.Err("POST.shared-lists no devolvió el ID del registro temporal", sharedListID)
		}
		sharedListIDByHash[nameHash] = persistedID
	}

	core.Log("GenerateSupplyData:: categorías/marcas creadas", "categorías", stats.NewCategories, "marcas", stats.NewBrands)
	return sharedListIDByHash, nil
}

// Both hashes come from the records' own SelfParse, so they match what the handlers store.
func productNameHash(name string) int32 {
	product := productionTypes.Product{Name: name}
	product.SelfParse()
	return product.NameHash
}

func sharedListNameHash(listID int32, name string) int32 {
	record := businessTypes.SharedListRecord{ListID: listID, Name: name}
	record.SelfParse()
	return record.NameHash
}
