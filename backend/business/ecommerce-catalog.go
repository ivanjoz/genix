// The web store catalog: a snapshot file on R2 with what the store can sell right now, and a delta
// endpoint that brings a browser from its cursor to now. docs/ECOMMERCE_CATALOG_SNAPSHOT_PLAN.md.
//
// The file holds only sellable products, each with its stock per presentation (the sellable rule,
// logistics.WebSellableStock). A browser keeps the catalog in IndexedDB and asks the delta from its
// `since`; every product the delta touches is sent whole, or as removed, so a product restocked after
// the file was built reaches it complete. A delta past ecommerceCatalogLargeDelta sends the browser to
// the file instead, when the file is newer: heavy data comes from R2, not from here.

package business

import (
	"app/business/types"
	"app/cloud"
	"app/core"
	"app/db"
	logistics "app/logistics/types"
	production "app/production/types"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ivanjoz/colbin"
	"golang.org/x/sync/errgroup"
)

const (
	// Every cursor starts this many ticks (2 s each) before the read that produced it: a write stamped
	// before the read but committed after it is read again by the next delta.
	ecommerceCatalogMarginTicks = int32(4)
	// A delta past this many records sends the browser to the file, and makes the file rebuild.
	ecommerceCatalogLargeDelta = 100
	// With at least one change since the file, it is rebuilt once it is this old (10 minutes).
	ecommerceCatalogMaxFileAgeTicks = int32(300)
	// Short enough that a rebuilt file reaches every browser within ~30 s; R2 takes the reads.
	ecommerceCatalogCacheControl = "public, max-age=15"

	// core.Cache keys: the file currently on R2 ("created,updated,generation"), and the company's
	// catalog generation, bumped when a warehouse change moves every product's sellable stock.
	ecommerceCatalogFileCacheKey       = "ecommerce_catalog_file"
	ecommerceCatalogGenerationCacheKey = "ecommerce_catalog_generation"
)

func ecommerceCatalogFileName(companyID int32) string {
	return fmt.Sprintf("ecommerce-c%d.cb", companyID)
}

// ecommerceCatalogFile is the file currently on R2. Updated is where a browser seeded from it starts
// its delta; Generation is the catalog generation it was built for.
type ecommerceCatalogFile struct {
	Created    int32
	Updated    int32
	Generation int32
}

// ecommerceCatalogFileContent is the file: a self-describing colbin message (the schema travels
// inside it), with the same record shapes the delta sends. Only sellable products.
type ecommerceCatalogFileContent struct {
	Meta       ecommerceCatalogFile
	Productos  []ecommerceCatalogProduct
	Marcas     []ecommerceCatalogListRecord
	Categorias []ecommerceCatalogListRecord
}

// ecommerceCatalogProduct is one product as the store gets it, in the file and in the delta. Stock
// lists every sellable presentation. Status 0 tells the browser to drop it.
type ecommerceCatalogProduct struct {
	ID          int32
	Name        string
	CategoryIDs []int32
	BrandID     int32
	Price       int32
	FinalPrice  int32
	Image       int32
	Stock       []ecommerceCatalogStock
	Status      int8 `json:"ss"`
}

// ecommerceCatalogStock is what the store can sell of one presentation (0: no presentations).
type ecommerceCatalogStock struct {
	PresentationID int16
	Units          int32
}

// ecommerceCatalogListRecord is one brand or category. Status 0 tells the browser to drop it.
type ecommerceCatalogListRecord struct {
	ID     int32
	Name   string
	Status int8 `json:"ss"`
}

// ecommerceCatalogDelta is the delta endpoint's answer. With UseFile set the browser reloads the
// file (FileCreated busts the CDN cache) and asks again from its Updated; the rest is empty.
type ecommerceCatalogDelta struct {
	UseFile     bool
	FileCreated int32
	Generation  int32
	NextSince   int32
	Productos   []ecommerceCatalogProduct
	Marcas      []ecommerceCatalogListRecord
	Categorias  []ecommerceCatalogListRecord
}

// ── State rows ──────────────────────────────────────────────────────────────────────────────────

// loadEcommerceCatalogState reads the current file (hasFile false before the first build) and the
// company's catalog generation (0 until a warehouse change bumps it).
func loadEcommerceCatalogState(companyID int32) (catalogFile ecommerceCatalogFile, hasFile bool, generation int32, err error) {
	rows, err := core.GetCacheByKeys(companyID, ecommerceCatalogFileCacheKey, ecommerceCatalogGenerationCacheKey)
	if err != nil {
		return catalogFile, false, 0, core.Err("Error al leer el estado del catálogo web:", err)
	}
	for _, row := range rows {
		values := strings.Split(row.Content, ",")
		switch {
		case row.Key == ecommerceCatalogFileCacheKey && len(values) == 3:
			catalogFile.Created, catalogFile.Updated, catalogFile.Generation = parseInt32(values[0]), parseInt32(values[1]), parseInt32(values[2])
			hasFile = true
		case row.Key == ecommerceCatalogGenerationCacheKey:
			generation = parseInt32(values[0])
		}
	}
	return catalogFile, hasFile, generation, nil
}

func parseInt32(text string) int32 {
	value, _ := strconv.Atoi(text)
	return int32(value)
}

func saveEcommerceCatalogCacheRow(companyID int32, key string, content string) error {
	row := core.Cache{CompanyID: companyID, ID: core.BasicHashInt(key), Key: key, Content: content, Updated: core.SUnixTime()}
	if err := db.Upsert(&[]core.Cache{row}); err != nil {
		return core.Err("Error al guardar el estado del catálogo web:", err)
	}
	return nil
}

// bumpEcommerceCatalogGeneration marks every browser's catalog stale: a warehouse opened to or closed
// to online sales changes every product's sellable stock without touching a product or stock row, so
// no delta could carry it. The generation is the bump's time: monotonic, with no read to race.
func bumpEcommerceCatalogGeneration(companyID int32) error {
	return saveEcommerceCatalogCacheRow(companyID, ecommerceCatalogGenerationCacheKey, strconv.Itoa(int(core.SUnixTime())))
}

// ── Sellable records ────────────────────────────────────────────────────────────────────────────

// readOnlineWarehouseIDs reads the warehouses whose stock the web store sells.
func readOnlineWarehouseIDs(companyID int32) ([]int32, error) {
	warehouses := []types.Warehouse{}
	query := db.Query(&warehouses)
	query.Select(query.ID, query.Status, query.ExcludeFromOnlineSales).CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los almacenes:", err)
	}
	return types.OnlineWarehouseIDs(warehouses), nil
}

// selectCatalogProductColumns is the projection of every product read here.
func selectCatalogProductColumns(query *production.ProductTable) {
	query.Select(query.ID, query.Name, query.CategoryIDs, query.BrandID, query.Price, query.FinalPrice, query.ImageMain,
		query.Presentations, query.WebExtraStock, query.WebHidden, query.Status, query.Updated)
}

// makeEcommerceCatalogProduct applies the sellable rule to one product: Status 0 when the store cannot
// sell it (inactive, a supply, hidden, nothing sellable in any presentation).
func makeEcommerceCatalogProduct(product *production.Product, stockRows []logistics.ProductStock) ecommerceCatalogProduct {
	catalogProduct := ecommerceCatalogProduct{ID: product.ID}
	if product.Status != production.ProductStatusActive || product.WebHidden {
		return catalogProduct
	}
	presentationIDs := product.WebPresentationIDs()
	sellableByPresentation := logistics.WebSellableStock(stockRows, presentationIDs, product.WebExtraStock)
	for _, presentationID := range presentationIDs {
		if sellableUnits := sellableByPresentation[presentationID]; sellableUnits > 0 {
			catalogProduct.Stock = append(catalogProduct.Stock, ecommerceCatalogStock{PresentationID: presentationID, Units: sellableUnits})
		}
	}
	if len(catalogProduct.Stock) == 0 {
		return catalogProduct
	}
	catalogProduct.Name, catalogProduct.CategoryIDs, catalogProduct.BrandID = product.Name, product.CategoryIDs, product.BrandID
	catalogProduct.Price, catalogProduct.FinalPrice, catalogProduct.Image = product.Price, product.FinalPrice, product.ImageMain
	catalogProduct.Status = 1
	return catalogProduct
}

// toCatalogListRecords keeps a shared list's rows as the store gets them.
func toCatalogListRecords(rows []types.SharedListRecord, listID int32) []ecommerceCatalogListRecord {
	records := []ecommerceCatalogListRecord{}
	for _, row := range rows {
		if row.ListID == listID {
			records = append(records, ecommerceCatalogListRecord{ID: row.ID, Name: row.Name, Status: core.If(row.Status > 0, int8(1), int8(0))})
		}
	}
	return records
}

// ── File ────────────────────────────────────────────────────────────────────────────────────────

// buildEcommerceCatalogFile reads every sellable product, writes the file to R2 and records it. Its
// Created is taken before the first read, so anything written during the build is in the next delta.
func buildEcommerceCatalogFile(companyID int32, generation int32) (ecommerceCatalogFile, error) {
	catalogFile := ecommerceCatalogFile{Created: core.SUnixTime(), Generation: generation}
	catalogFile.Updated = catalogFile.Created - ecommerceCatalogMarginTicks

	onlineWarehouseIDs, err := readOnlineWarehouseIDs(companyID)
	if err != nil {
		return catalogFile, err
	}
	products := []production.Product{}
	stockRows := []logistics.ProductStock{}
	sharedRows := []types.SharedListRecord{}
	loadGroup := errgroup.Group{}
	loadGroup.Go(func() error {
		query := db.Query(&products)
		selectCatalogProductColumns(query)
		query.CompanyID.Equals(companyID).Status.Equals(production.ProductStatusActive)
		return query.Exec()
	})
	// The whole company's stock: one partition read beats packing every product × warehouse ID.
	loadGroup.Go(func() error {
		query := db.Query(&stockRows)
		query.Select(query.WarehouseID, query.ProductID, query.PresentationID, query.Quantity, query.DetailQuantity,
			query.BlockedQuantity, query.PendingDeliveryStore, query.PendingDeliveryOnline).CompanyID.Equals(companyID)
		return query.Exec()
	})
	loadGroup.Go(func() error {
		query := db.Query(&sharedRows)
		query.Select(query.ID, query.ListID, query.Name, query.Status).CompanyID.Equals(companyID).
			ListID.In(types.SharedListBrandID, types.SharedListCategoryID).Status.Equals(1)
		return query.Exec()
	})
	if err := loadGroup.Wait(); err != nil {
		return catalogFile, core.Err("Error al leer el catálogo web:", err)
	}
	onlineStockByProduct := map[int32][]logistics.ProductStock{}
	for _, stockRow := range stockRows {
		if slices.Contains(onlineWarehouseIDs, stockRow.WarehouseID) {
			onlineStockByProduct[stockRow.ProductID] = append(onlineStockByProduct[stockRow.ProductID], stockRow)
		}
	}

	fileContent := ecommerceCatalogFileContent{
		Meta:       catalogFile,
		Marcas:     toCatalogListRecords(sharedRows, types.SharedListBrandID),
		Categorias: toCatalogListRecords(sharedRows, types.SharedListCategoryID),
	}
	for index := range products {
		catalogProduct := makeEcommerceCatalogProduct(&products[index], onlineStockByProduct[products[index].ID])
		if catalogProduct.Status > 0 {
			fileContent.Productos = append(fileContent.Productos, catalogProduct)
		}
	}
	fileBytes, err := colbin.MarshalSelfDescribing(fileContent)
	if err != nil {
		return catalogFile, core.Err("Error al serializar el catálogo web:", err)
	}
	if err := cloud.SaveFile(cloud.SaveFileArgs{
		Path: "live", Name: ecommerceCatalogFileName(companyID), FileContent: fileBytes,
		ContentType: core.ColbinMediaType, CacheControl: ecommerceCatalogCacheControl,
	}); err != nil {
		return catalogFile, core.Err("Error al subir el catálogo web:", err)
	}
	content := fmt.Sprintf("%d,%d,%d", catalogFile.Created, catalogFile.Updated, catalogFile.Generation)
	if err := saveEcommerceCatalogCacheRow(companyID, ecommerceCatalogFileCacheKey, content); err != nil {
		return catalogFile, err
	}
	core.Log("buildEcommerceCatalogFile:: empresa", companyID, "productos vendibles", len(fileContent.Productos), "de", len(products),
		"created", catalogFile.Created, "generation", generation)
	return catalogFile, nil
}

// rebuildEcommerceCatalogFile builds the file under the company's build lock. waitForBuild queues
// behind a build under way and then reuses its file when it already covers generation (a request
// that needs a file to answer); without it a build under way means this one is skipped (a request
// that only noticed the file aging). Returns the file on R2 afterwards.
func rebuildEcommerceCatalogFile(companyID int32, generation int32, waitForBuild bool) (ecommerceCatalogFile, bool, error) {
	// A whole-company build outlasts the default 15 s lease; a waiter outlasts one build.
	lockOptions := core.LockOptions{MaxWaiters: 0, Lease: 60 * time.Second}
	if waitForBuild {
		lockOptions.MaxWaiters, lockOptions.Wait = 16, 30*time.Second
	}
	buildLock, err := core.AcquireLockWithOptions(context.Background(), core.ActionBuildEcommerceCatalog, int64(companyID), lockOptions)
	if err != nil {
		if !waitForBuild {
			return ecommerceCatalogFile{}, false, nil
		}
		return ecommerceCatalogFile{}, false, core.Err("El catálogo web se está generando, intente nuevamente:", err)
	}
	defer buildLock.Release()

	// The build this request queued behind may have produced what it needs.
	if waitForBuild {
		catalogFile, hasFile, _, err := loadEcommerceCatalogState(companyID)
		if err != nil {
			return catalogFile, false, err
		}
		if hasFile && catalogFile.Generation == generation {
			return catalogFile, true, nil
		}
	}
	catalogFile, err := buildEcommerceCatalogFile(companyID, generation)
	return catalogFile, err == nil, err
}

// ── Delta ───────────────────────────────────────────────────────────────────────────────────────

// GetEcommerceCatalog answers a browser's delta: GET p-ecommerce-catalog?company-id=&since=&gen=. A
// browser with nothing saved sends no `since`; one whose generation is stale, or whose delta would be
// large while the file is newer, is told to reload the file.
func GetEcommerceCatalog(req *core.HandlerArgs) core.HandlerResponse {
	companyID := req.GetQueryInt("company-id")
	if companyID <= 0 {
		return req.MakeErr("Empresa inválida para obtener el catálogo web.")
	}
	since, clientGeneration := int32(req.GetQueryInt("since")), int32(req.GetQueryInt("gen"))
	// Before any read: the next cursor starts the margin before it.
	readTime := core.SUnixTime()

	catalogFile, hasFile, generation, err := loadEcommerceCatalogState(companyID)
	if err != nil {
		return req.MakeErr(err)
	}
	// No file yet, or one built before a warehouse change: build it now; the browser needs it.
	if !hasFile || catalogFile.Generation != generation {
		if catalogFile, _, err = rebuildEcommerceCatalogFile(companyID, generation, true); err != nil {
			return req.MakeErr(err)
		}
	}
	useFile := ecommerceCatalogDelta{UseFile: true, FileCreated: catalogFile.Created, Generation: catalogFile.Generation}
	if since <= 0 || clientGeneration != catalogFile.Generation {
		return req.MakeResponse(useFile)
	}

	delta, changesSinceFile, err := readEcommerceCatalogDelta(companyID, since, catalogFile.Updated)
	if err != nil {
		return req.MakeErr(err)
	}
	delta.Generation = catalogFile.Generation
	delta.NextSince = readTime - ecommerceCatalogMarginTicks
	deltaSize := len(delta.Productos) + len(delta.Marcas) + len(delta.Categorias)

	// Rebuild rule: over the large-delta threshold since the file, or anything at all once the file
	// is ten minutes old. Built here, not scheduled: a scheduled action waits for the next 5-minute
	// frame, and on Lambda nothing may run after the response.
	isFileAged := readTime-catalogFile.Created >= ecommerceCatalogMaxFileAgeTicks
	if changesSinceFile > ecommerceCatalogLargeDelta || (changesSinceFile > 0 && isFileAged) {
		rebuiltFile, isRebuilt, err := rebuildEcommerceCatalogFile(companyID, catalogFile.Generation, false)
		if err != nil {
			core.Log("GetEcommerceCatalog:: no se pudo reconstruir el catálogo web", companyID, err)
		} else if isRebuilt {
			catalogFile = rebuiltFile
			useFile.FileCreated = catalogFile.Created
		}
	}
	// A large delta, and a file newer than the browser's cursor: R2 serves it instead.
	if deltaSize > ecommerceCatalogLargeDelta && catalogFile.Updated > since {
		return req.MakeResponse(useFile)
	}
	return req.MakeResponse(delta)
}

// readEcommerceCatalogDelta reads what changed at or after since and rebuilds every touched product
// whole. changesSinceFile counts the changed rows stamped at or after the file's Updated: exact when
// since is older than the file, a lower bound otherwise (rows between the file and since are not read).
func readEcommerceCatalogDelta(companyID, since, fileUpdated int32) (ecommerceCatalogDelta, int, error) {
	delta := ecommerceCatalogDelta{}
	onlineWarehouseIDs, err := readOnlineWarehouseIDs(companyID)
	if err != nil {
		return delta, 0, err
	}
	changedProducts := []production.Product{}
	changedStock := []logistics.ProductStock{}
	changedBrands := []types.SharedListRecord{}
	changedCategories := []types.SharedListRecord{}
	readGroup := errgroup.Group{}
	readGroup.Go(func() error {
		query := db.Query(&changedProducts)
		selectCatalogProductColumns(query)
		return query.CompanyID.Equals(companyID).ChangedSince(since).Exec()
	})
	readGroup.Go(func() error {
		query := db.Query(&changedStock)
		query.Select(query.ID, query.WarehouseID, query.ProductID, query.Updated, query.Status)
		return query.CompanyID.Equals(companyID).ChangedSince(since).Exec()
	})
	for _, list := range []struct {
		listID int32
		rows   *[]types.SharedListRecord
	}{{types.SharedListBrandID, &changedBrands}, {types.SharedListCategoryID, &changedCategories}} {
		readGroup.Go(func() error {
			query := db.Query(list.rows)
			query.Select(query.ID, query.ListID, query.Name, query.Status, query.Updated)
			return query.CompanyID.Equals(companyID).ListID.Equals(list.listID).ChangedSince(since).Exec()
		})
	}
	if err := readGroup.Wait(); err != nil {
		return delta, 0, core.Err("Error al leer los cambios del catálogo web:", err)
	}

	changesSinceFile := 0
	countIfSinceFile := func(updated int32) {
		if updated >= fileUpdated {
			changesSinceFile++
		}
	}
	productByID := map[int32]*production.Product{}
	for index := range changedProducts {
		productByID[changedProducts[index].ID] = &changedProducts[index]
		countIfSinceFile(changedProducts[index].Updated)
	}
	// A stock row moves its product: read the products only their stock touched.
	stockOnlyProductIDs := core.SliceSet[int32]{}
	for _, stockRow := range changedStock {
		if !slices.Contains(onlineWarehouseIDs, stockRow.WarehouseID) {
			continue
		}
		countIfSinceFile(stockRow.Updated)
		if productByID[stockRow.ProductID] == nil {
			stockOnlyProductIDs.Add(stockRow.ProductID)
		}
	}
	if !stockOnlyProductIDs.IsEmpty() {
		stockOnlyProducts := []production.Product{}
		query := db.Query(&stockOnlyProducts)
		selectCatalogProductColumns(query)
		if err := query.CompanyID.Equals(companyID).ID.In(stockOnlyProductIDs.Values...).Exec(); err != nil {
			return delta, 0, core.Err("Error al leer los productos del catálogo web:", err)
		}
		for index := range stockOnlyProducts {
			productByID[stockOnlyProducts[index].ID] = &stockOnlyProducts[index]
		}
	}

	// Every touched product is recomputed over all of its online stock rows, not only the changed ones.
	presentationIDsByProduct := map[int32][]int16{}
	for productID, product := range productByID {
		presentationIDsByProduct[productID] = product.WebPresentationIDs()
	}
	stockRowsByProduct, err := logistics.ReadWebStockRows(companyID, onlineWarehouseIDs, presentationIDsByProduct)
	if err != nil {
		return delta, 0, err
	}
	for productID, product := range productByID {
		// A supply (Status 2) never reaches the store, not even as a removal.
		if product.Status == production.ProductStatusSupply {
			continue
		}
		delta.Productos = append(delta.Productos, makeEcommerceCatalogProduct(product, stockRowsByProduct[productID]))
	}
	delta.Marcas = toCatalogListRecords(changedBrands, types.SharedListBrandID)
	delta.Categorias = toCatalogListRecords(changedCategories, types.SharedListCategoryID)
	for _, row := range slices.Concat(changedBrands, changedCategories) {
		countIfSinceFile(row.Updated)
	}
	return delta, changesSinceFile, nil
}
