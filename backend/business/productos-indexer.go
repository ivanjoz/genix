package business

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/libs/index_builder"
	businessTypes "app/business/types"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PRODUCTO_INDEX_VERSION is baked into the uploaded file name. Bumping this constant invalidates
// any cached file-name pointer so the next GetProductsIndex call rebuilds and re-uploads the .idx.
const PRODUCTO_INDEX_VERSION = 1

func GetProductsIndex(req *core.HandlerArgs) core.HandlerResponse {
	companyID := core.Coalesce(
		core.Coalesce(req.GetQueryInt("company-id"), req.GetQueryInt("empresa_id")),
		req.User.CompanyID,
	)
	if companyID <= 0 {
		return req.MakeErr("Company inválida para obtener índice de productos.")
	}

	// Fetch the freshest productos.Updated using the Updated view; this is the change-detection watermark.
	productsLatestRows := []businessTypes.Product{}
	latestQuery := db.Query(&productsLatestRows)
	latestQuery.Select(latestQuery.ID, latestQuery.Updated).
		CompanyID.Equals(companyID).
		Updated.GreaterThan(0).
		OrderDesc().
		Limit(1)
	if latestErr := latestQuery.Exec(); latestErr != nil {
		return req.MakeErr("Error al obtener el último product actualizado.", latestErr)
	}
	productsWatermark := int32(0)
	if len(productsLatestRows) > 0 {
		productsWatermark = productsLatestRows[0].Updated
	}

	// Read last build state: cache.Updated holds the watermark observed when the file was uploaded,
	// cache.Content holds the bucket-derived file name of that upload.
	currentCacheKey := productSortedIDsCacheKey + "_current"
	cacheRows, cacheQueryErr := core.GetCacheByKeys(companyID, currentCacheKey)
	if cacheQueryErr != nil {
		return req.MakeErr("Error al obtener metadata actual del índice de productos.", cacheQueryErr)
	}
	cachedWatermark := int32(0)
	cachedFileName := ""
	if len(cacheRows) > 0 {
		cachedWatermark = cacheRows[0].Updated
		cachedFileName = cacheRows[0].Content
	}

	// Skip the rebuild only when the watermark matches, we have a cached file name, AND that file
	// carries the current PRODUCTO_INDEX_VERSION marker — a version bump invalidates older uploads.
	versionPrefix := productsBucketFileNameVersionPrefix(companyID)
	if cachedFileName != "" &&
		cachedWatermark == productsWatermark &&
		strings.HasPrefix(cachedFileName, versionPrefix) {
		core.Log("GetProductsIndex:: cache hit, no rebuild",
			"| companyID:", companyID,
			"| name:", cachedFileName,
			"| watermark:", productsWatermark)
		return req.MakeResponse(map[string]any{
			"name":    cachedFileName,
			"updated": productsWatermark,
		})
	}

	// Build with a fresh 5-minute-bucket file name; bucket guarantees a new URL each window so CDN/edge caches don't serve stale bytes.
	bucketFileName := buildProductsBucketFileName(companyID)
	if _, buildErr := BuildProductosSearchIndex(BuildProductosSearchIndexArgs{
		CompanyID:         companyID,
		FileName:          bucketFileName,
		ProductsWatermark: productsWatermark,
	}); buildErr != nil {
		return req.MakeErr("Error al reconstruir el índice de productos.", buildErr)
	}

	core.Log("GetProductsIndex:: rebuilt",
		"| companyID:", companyID,
		"| name:", bucketFileName,
		"| watermark:", productsWatermark,
		"| previousWatermark:", cachedWatermark)
	return req.MakeResponse(map[string]any{
		"name":    bucketFileName,
		"updated": productsWatermark,
	})
}

// buildProductsBucketFileName returns "c<company>_products_v<ver>_<bucket>.idx" where
// bucket = ceil(secondsOfDayUTC / 300). The bucket changes every 5 minutes so a freshly built
// index always lands at a never-before-cached URL, and the embedded version lets the cache
// shortcut detect stale entries when PRODUCTO_INDEX_VERSION bumps.
func buildProductsBucketFileName(companyID int32) string {
	now := time.Now().UTC()
	secondsOfDay := now.Hour()*3600 + now.Minute()*60 + now.Second()
	const secondsPerBucket = 5 * 60
	// Ceil division: 0→0, 1..300→1, 301..600→2, etc.
	bucket := (secondsOfDay + secondsPerBucket - 1) / secondsPerBucket
	return fmt.Sprintf("%s%d.idx", productsBucketFileNameVersionPrefix(companyID), bucket)
}

// productsBucketFileNameVersionPrefix is the leading "c<company>_products_v<ver>_" segment.
// It's the hook the cache check uses to detect a version mismatch on the cached file name.
func productsBucketFileNameVersionPrefix(companyID int32) string {
	return fmt.Sprintf("c%d_products_v%d_", companyID, PRODUCTO_INDEX_VERSION)
}

func GetProductsIndexDelta(req *core.HandlerArgs) core.HandlerResponse {

	updated := req.GetQueryInt("productos")
	companyID := req.GetQueryInt("company-id")
	if updated == 0 {
		cacheRecords, err := core.GetCacheByKeys(companyID, productSortedIDsCacheKey+"_current")
		if err != nil {
			return req.MakeErr(err)
		}

		core.Log("Cache records::", len(cacheRecords))
		if len(cacheRecords) == 0 {
			return req.MakeResponse([]string{})
		}

		updated = cacheRecords[0].Updated
	}

	productosDelta := []businessTypes.Product{}
	q1 := db.Query(&productosDelta)
	err := q1.Select(q1.Nombre, q1.MarcaID, q1.CategoriasIDs, q1.NameUpdated, q1.Status, q1.StockStatus).
		NameUpdated.GreaterEqual(updated).AllowFilter().Exec()

	if err != nil {
		return req.MakeErr(err)
	}

	if len(productosDelta) == 0 {
		return req.MakeResponse(productosDelta)
	}

	marcasCategoriasIDs := core.SliceSet[int32]{}
	for index := range productosDelta {
		e := &productosDelta[index]
		marcasCategoriasIDs.AddIf(e.MarcaID)
		marcasCategoriasIDs.AddIfBulk(e.CategoriasIDs...)
		e.Updated = e.NameUpdated
		e.NameUpdated = 0
	}

	marcasCategorias := []businessTypes.SharedListRecord{}
	q2 := db.Query(&marcasCategorias)
	err = q2.Select(q2.ID, q2.Nombre, q2.Updated).
		ID.In(marcasCategoriasIDs.Values...).Exec()

	if err != nil {
		return req.MakeErr(err)
	}

	normalizedBrandNameByID := make(map[int32]string, len(marcasCategorias)+1)
	for _, sharedRecord := range marcasCategorias {
		normalizedBrandNameByID[sharedRecord.ID] = index_builder.NormalizeTextForIndex(sharedRecord.Nombre)
	}
	// Fallback for products without explicit brand assignment.
	if _, hasSentinelBrand := normalizedBrandNameByID[0]; !hasSentinelBrand {
		normalizedBrandNameByID[0] = index_builder.NormalizeTextForIndex("Sin marca")
	}

	// Normalize product names and remove brand terms/connectors/single-letter words.
	for productIndex := range productosDelta {
		currentProduct := &productosDelta[productIndex]
		normalizedBrandName := normalizedBrandNameByID[currentProduct.MarcaID]
		currentProduct.Nombre = index_builder.CleanProductTextByBrand(currentProduct.Nombre, normalizedBrandName)
		core.Log("currentProduct.Nombre", currentProduct.Nombre)
	}

	response := map[string]any{
		"productos":        &productosDelta,
		"marcasCategorias": &marcasCategorias,
	}

	return req.MakeResponse(response)
}

const (
	productSharedListCategoriaID = int32(1)
	productSharedListMarcaID     = int32(2)
	productIndexerCacheTTL       = 15 * time.Minute
	productIndexerOutputFilePath = "libs/index_builder/productos.idx"
	productSortedIDsCacheKey     = "productos_sorted_ids"
)

type ProductosIndexBuildOutput struct {
	IndexBuild *index_builder.ProductosIndexBuild
}

func buildInputFromProductosData(productsData ProductosData) (index_builder.BuildInput, error) {
	if len(productsData.Productos) == 0 {
		return index_builder.BuildInput{}, fmt.Errorf("no se encontraron productos activos para indexar")
	}

	productIndexRecords := make([]index_builder.RecordInput, 0, len(productsData.Productos))
	for _, currentProduct := range productsData.Productos {
		// Validate source integrity before encoding because record IDs are persisted in section payloads.
		if currentProduct.ID <= 0 {
			return index_builder.BuildInput{}, fmt.Errorf("product con ID inválido=%d", currentProduct.ID)
		}
		productIndexRecords = append(productIndexRecords, index_builder.RecordInput{
			ID:            currentProduct.ID,
			Text:          currentProduct.Nombre,
			BrandID:       currentProduct.MarcaID,
			CategoriesIDs: append([]int32(nil), currentProduct.CategoriasIDs...),
		})
	}

	brandIndexRecords := make([]index_builder.RecordInput, 0, len(productsData.Marcas)+1)
	hasSentinelBrandIDZero := false
	for _, currentBrand := range productsData.Marcas {
		if currentBrand.ID == 0 {
			hasSentinelBrandIDZero = true
		}
		brandIndexRecords = append(brandIndexRecords, index_builder.RecordInput{
			ID: currentBrand.ID, Text: currentBrand.Nombre,
		})
	}
	// Keep MarcaID=0 as a valid sentinel so stage 2 can index products without explicit brand.
	if !hasSentinelBrandIDZero {
		brandIndexRecords = append(brandIndexRecords, index_builder.RecordInput{
			ID: 0, Text: "Sin marca",
		})
	}

	categoryIndexRecords := make([]index_builder.RecordInput, 0, len(productsData.Categorias))
	for _, currentCategory := range productsData.Categorias {
		categoryIndexRecords = append(categoryIndexRecords, index_builder.RecordInput{
			ID: currentCategory.ID, Text: currentCategory.Nombre,
		})
	}

	return index_builder.BuildInput{
		Products:   productIndexRecords,
		Brands:     brandIndexRecords,
		Categories: categoryIndexRecords,
	}, nil
}

func logSourceCardinality(prefix string, buildInput index_builder.BuildInput) {
	core.Log(prefix+":: productos:", len(buildInput.Products),
		"| marcas únicas:", len(buildInput.Brands),
		"| categorías únicas:", len(buildInput.Categories))
}

func logStageOneBuildStats(prefix string, buildArtifacts *index_builder.ProductosIndexBuild) {
	core.Log(prefix+":: text optimization",
		"| changed:", buildArtifacts.OptimizationStats.ChangedProducts,
		"| fallbackOriginal:", buildArtifacts.OptimizationStats.FallbackOriginalProducts)
	core.Log(prefix+":: stage1.sorted_ids_count:", len(buildArtifacts.SortedIDs))
	core.Log(prefix+":: stage1.dictionary_bytes:", buildArtifacts.Stats.DictionaryBytes)
	core.Log(prefix+":: stage1.aliases_bytes:", buildArtifacts.Stats.AliasBytes)
	core.Log(prefix+":: stage1.shapes_bytes:", buildArtifacts.Stats.ShapesBytes)
	core.Log(prefix+":: stage1.content_bytes:", buildArtifacts.Stats.ContentBytes)
	core.Log(prefix+":: stage1.product_ids_bytes:", buildArtifacts.Stats.ProductIDsBytes)
	core.Log(prefix+":: stage1.total_bytes:", buildArtifacts.Stats.TotalBytes)
}

// BuildProductosSearchIndexNoPersist builds the same two-stage index as BuildProductosSearchIndex
// but keeps all artifacts in-memory only:
// - no S3 upload
// - no sorted IDs cache persistence in DB
// - no local gob source cache read/write
func BuildProductosSearchIndexNoPersist(companyID int32) (*ProductosIndexBuildOutput, error) {
	if companyID <= 0 {
		return nil, fmt.Errorf("company ID inválido para construir el índice")
	}

	productsData, loadErr := loadIndexerSourceDataWithoutCache(companyID)
	if loadErr != nil {
		return nil, loadErr
	}
	buildInput, buildInputErr := buildInputFromProductosData(productsData)
	if buildInputErr != nil {
		return nil, buildInputErr
	}

	logSourceCardinality("BuildIndex(NoPersist)", buildInput)
	indexBuildArtifacts, buildErr := index_builder.BuildProductosIndex(buildInput)
	if buildErr != nil {
		return nil, fmt.Errorf("error al construir índice de productos (no persist): %w", buildErr)
	}
	logStageOneBuildStats("BuildIndex(NoPersist)", indexBuildArtifacts)

	combinedIndexBytes, indexBytesErr := indexBuildArtifacts.ToBytes()
	if indexBytesErr != nil {
		return nil, fmt.Errorf("error al serializar índice combinado de productos (no persist): %w", indexBytesErr)
	}
	core.Log("BuildIndex(NoPersist):: final.total_bytes_stage1_plus_stage2:", len(combinedIndexBytes))

	return &ProductosIndexBuildOutput{
		IndexBuild: indexBuildArtifacts,
	}, nil
}

// BuildProductosSearchIndexArgs configures a persisted index build:
//   - FileName is the S3 object name to upload under "live/" (e.g. "c1_products_142.idx").
//   - ProductsWatermark is the max productos.Updated observed before the build; persisted in the cache
//     row so subsequent GetProductsIndex calls can skip rebuilding when productos hasn't changed.
type BuildProductosSearchIndexArgs struct {
	CompanyID         int32
	FileName          string
	ProductsWatermark int32
}

// BuildProductosSearchIndex builds both index passes from productos:
// 1) generic text index by product ID + Nombre
// 2) taxonomy pass by querying brand/category names from shared lists.
// The result is uploaded under args.FileName and the cache row is stamped with args.ProductsWatermark + args.FileName.
func BuildProductosSearchIndex(args BuildProductosSearchIndexArgs) (*ProductosIndexBuildOutput, error) {
	if args.CompanyID <= 0 {
		return nil, fmt.Errorf("company ID inválido para construir el índice")
	}
	if args.FileName == "" {
		return nil, fmt.Errorf("nombre de archivo inválido para construir el índice")
	}

	productsData, loadErr := loadIndexerSourceDataWithCache(args.CompanyID)
	if loadErr != nil {
		return nil, loadErr
	}
	buildInput, buildInputErr := buildInputFromProductosData(productsData)
	if buildInputErr != nil {
		return nil, buildInputErr
	}

	logSourceCardinality("BuildIndex", buildInput)
	indexBuildArtifacts, buildErr := index_builder.BuildProductosIndex(buildInput)
	if buildErr != nil {
		return nil, fmt.Errorf("error al construir índice de productos: %w", buildErr)
	}
	logStageOneBuildStats("BuildIndex", indexBuildArtifacts)

	stageOneIndexResult := indexBuildArtifacts
	stageTwoIndexResult := indexBuildArtifacts
	// Persist build time in the text header so the final .idx carries one explicit freshness slot.
	buildSunixTime := core.SUnixTime()
	stageOneIndexResult.BuildSunixTime = buildSunixTime

	brandNamesBytes := 0
	for _, currentBrandName := range stageTwoIndexResult.BrandNames {
		// String column format: 1-byte length prefix + utf8 bytes.
		brandNamesBytes += 1 + len([]byte(currentBrandName))
	}
	categoryNamesBytes := 0
	for _, currentCategoryName := range stageTwoIndexResult.CategoryNames {
		// String column format: 1-byte length prefix + utf8 bytes.
		categoryNamesBytes += 1 + len([]byte(currentCategoryName))
	}
	core.Log("BuildIndex:: stage2.build_sunix_time:", stageOneIndexResult.BuildSunixTime)
	core.Log("BuildIndex:: stage2.brand_unique_ids_count:", len(stageTwoIndexResult.BrandIDs))
	core.Log("BuildIndex:: stage2.brand_names_bytes:", brandNamesBytes)
	core.Log("BuildIndex:: stage2.brand_index_bytes:", stageTwoIndexResult.ProductBrandIndexesBytes())
	core.Log("BuildIndex:: stage2.category_unique_count:", len(stageTwoIndexResult.CategoryIDs))
	core.Log("BuildIndex:: stage2.category_names_bytes:", categoryNamesBytes)
	core.Log("BuildIndex:: stage2.category_index_bytes:", len(stageTwoIndexResult.ProductCategoryIndexes))

	combinedIndexBytes, indexBytesErr := stageOneIndexResult.ToBytes()
	if indexBytesErr != nil {
		return nil, fmt.Errorf("error al serializar índice combinado de productos: %w", indexBytesErr)
	}
	core.Log("BuildIndex:: final.total_bytes_stage1_plus_stage2:", len(combinedIndexBytes))

	// Bucket is left empty so cloud.SaveFile picks the right one for the configured CLOUD_PROVIDER
	// (S3 vs R2). The previous hardcoded S3_BUCKET caused the file to land in S3 even when the
	// frontend was reading from Cloudflare R2.
	uploadArgs := cloud.SaveFileArgs{
		Path:        "live",
		Name:        args.FileName,
		FileContent: combinedIndexBytes,
		ContentType: "application/octet-stream",
	}
	if uploadErr := cloud.SaveFile(uploadArgs); uploadErr != nil {
		return nil, fmt.Errorf("error al guardar índice productos (live/%s): %w", args.FileName, uploadErr)
	}

	// Verify the upload landed: HEAD the same key so we don't stamp the cache with a file the
	// frontend cannot fetch. Any not-found / transport error here aborts before persisting the cache row.
	exists, headErr := cloud.FileExists(uploadArgs)
	if headErr != nil {
		return nil, fmt.Errorf("error al verificar índice productos (live/%s): %w", args.FileName, headErr)
	}
	if !exists {
		return nil, fmt.Errorf("índice productos no se encontró tras subida (live/%s)", args.FileName)
	}
	core.Log("BuildIndex:: upload verificado | live/" + args.FileName)

	if persistUpdatedCacheErr := persistProductosIndexUpdatedCache(args.CompanyID, args.ProductsWatermark, args.FileName); persistUpdatedCacheErr != nil {
		return nil, persistUpdatedCacheErr
	}
	core.Log("BuildIndex:: archivo índice actualizado:", "| key:", "live/"+args.FileName)

	return &ProductosIndexBuildOutput{
		IndexBuild: stageOneIndexResult,
	}, nil
}

// persistProductosIndexUpdatedCache stamps the cache row used by GetProductsIndex to short-circuit rebuilds.
// Updated holds the productos watermark observed at build time; Content holds the uploaded file name.
func persistProductosIndexUpdatedCache(companyID int32, productsWatermark int32, fileName string) error {
	if companyID <= 0 {
		return fmt.Errorf("company ID inválido para guardar updated de índice de productos")
	}
	if fileName == "" {
		return fmt.Errorf("nombre de archivo inválido para guardar updated de índice de productos")
	}

	cacheRowCurrent := core.Cache{
		CompanyID: companyID,
		ID:        core.BasicHashInt(productSortedIDsCacheKey + "_current"),
		Key:       productSortedIDsCacheKey + "_current",
		Updated:   productsWatermark,
		Content:   fileName,
	}

	if insertErr := db.Insert(&[]core.Cache{cacheRowCurrent}); insertErr != nil {
		core.Log(insertErr)
		return fmt.Errorf("error al guardar updated de índice de productos en cache (company=%d): %w", companyID, insertErr)
	}

	core.Log("BuildIndex:: updated cache actualizado",
		"| key:", cacheRowCurrent.Key,
		"| updated:", cacheRowCurrent.Updated,
		"| name:", cacheRowCurrent.Content)
	return nil
}

type ProductosData struct {
	Productos  []businessTypes.Product
	Categorias []businessTypes.SharedListRecord
	Marcas     []businessTypes.SharedListRecord
}

func loadIndexerSourceDataWithCache(
	companyID int32,
) (ProductosData, error) {
	cacheFilePath := filepath.Join("tmp", fmt.Sprintf("productos-indexer-%d.gob", companyID))
	if cachedPayload, cacheHit, cacheErr := loadProductosIndexerCache(cacheFilePath); cacheErr == nil && cacheHit {
		core.Log("BuildIndex:: cache hit:", cacheFilePath, "| productos:", len(cachedPayload.Productos))
		return cachedPayload, nil
	}

	productos := []businessTypes.Product{}
	query := db.Query(&productos)
	query.Select(query.ID, query.Nombre, query.CategoriasIDs, query.MarcaID).
		CompanyID.Equals(companyID).
		Status.GreaterThan(0)
	if queryErr := query.Exec(); queryErr != nil {
		return ProductosData{}, fmt.Errorf("error al obtener productos para indexado: %w", queryErr)
	}

	marcas := []businessTypes.SharedListRecord{}
	categorias := []businessTypes.SharedListRecord{}
	// Query all active categories and brands to avoid large clustering-key IN cartesian products.
	listRows := []businessTypes.SharedListRecord{}
	listQuery := db.Query(&listRows)
	listQuery.Select(listQuery.ID, listQuery.Nombre, listQuery.ListaID).
		CompanyID.Equals(companyID).
		ListaID.In(productSharedListCategoriaID, productSharedListMarcaID).
		Status.GreaterThan(0).
		AllowFilter()
	if queryErr := listQuery.Exec(); queryErr != nil {
		return ProductosData{}, fmt.Errorf("error al obtener marcas/categorías para índice: %w", queryErr)
	}

	for _, listRow := range listRows {
		if listRow.ListaID == productSharedListMarcaID {
			marcas = append(marcas, listRow)
		} else if listRow.ListaID == productSharedListCategoriaID {
			categorias = append(categorias, listRow)
		}
	}

	productsData := ProductosData{
		Productos:  productos,
		Marcas:     marcas,
		Categorias: categorias,
	}

	// Persist the full source payload as a short-lived cache to avoid repeated DB scans.
	if mkdirErr := os.MkdirAll(filepath.Dir(cacheFilePath), 0o755); mkdirErr != nil {
		core.Log("BuildIndex:: no se pudo crear carpeta cache:", mkdirErr)
		return productsData, nil
	}
	cacheFile, createErr := os.Create(cacheFilePath)
	if createErr != nil {
		core.Log("BuildIndex:: no se pudo crear archivo cache:", createErr)
		return productsData, nil
	}
	defer cacheFile.Close()

	cacheEncoder := gob.NewEncoder(cacheFile)
	if encodeErr := cacheEncoder.Encode(productsData); encodeErr != nil {
		core.Log("BuildIndex:: no se pudo serializar cache gob:", encodeErr)
	}

	core.Log("loadIndexerSourceDataWithCache:: marcas:", len(marcas), "| categorías:", len(categorias))
	return productsData, nil
}

func loadIndexerSourceDataWithoutCache(
	companyID int32,
) (ProductosData, error) {
	productos := []businessTypes.Product{}
	query := db.Query(&productos)
	query.Select(query.ID, query.Nombre, query.CategoriasIDs, query.MarcaID).
		CompanyID.Equals(companyID).
		Status.GreaterThan(0)
	if queryErr := query.Exec(); queryErr != nil {
		return ProductosData{}, fmt.Errorf("error al obtener productos para indexado: %w", queryErr)
	}

	marcas := []businessTypes.SharedListRecord{}
	categorias := []businessTypes.SharedListRecord{}
	// Query all active categories and brands to avoid large clustering-key IN cartesian products.
	listRows := []businessTypes.SharedListRecord{}
	listQuery := db.Query(&listRows)
	listQuery.Select(listQuery.ID, listQuery.Nombre, listQuery.ListaID).
		CompanyID.Equals(companyID).
		ListaID.In(productSharedListCategoriaID, productSharedListMarcaID).
		Status.GreaterThan(0).
		AllowFilter()
	if queryErr := listQuery.Exec(); queryErr != nil {
		return ProductosData{}, fmt.Errorf("error al obtener marcas/categorías para índice: %w", queryErr)
	}

	for _, listRow := range listRows {
		if listRow.ListaID == productSharedListMarcaID {
			marcas = append(marcas, listRow)
		} else if listRow.ListaID == productSharedListCategoriaID {
			categorias = append(categorias, listRow)
		}
	}

	productsData := ProductosData{
		Productos:  productos,
		Marcas:     marcas,
		Categorias: categorias,
	}

	core.Log("loadIndexerSourceDataWithoutCache:: marcas:", len(marcas), "| categorías:", len(categorias))
	return productsData, nil
}

func loadProductosIndexerCache(cacheFilePath string) (ProductosData, bool, error) {
	cacheInfo, statErr := os.Stat(cacheFilePath)
	if statErr != nil {
		return ProductosData{}, false, statErr
	}
	if time.Since(cacheInfo.ModTime()) > productIndexerCacheTTL {
		return ProductosData{}, false, fmt.Errorf("cache expired")
	}

	cacheFile, openErr := os.Open(cacheFilePath)
	if openErr != nil {
		return ProductosData{}, false, openErr
	}
	defer cacheFile.Close()

	decoder := gob.NewDecoder(cacheFile)
	cachePayload := ProductosData{}
	if decodeErr := decoder.Decode(&cachePayload); decodeErr != nil {
		return ProductosData{}, false, decodeErr
	}
	return cachePayload, true, nil
}
