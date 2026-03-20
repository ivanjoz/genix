package negocio

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/libs/index_builder"
	negocioTypes "app/negocio/types"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func GetProductsIndex(req *core.HandlerArgs) core.HandlerResponse {
	clientUpdated := core.Coalesce(req.GetQueryInt64("upd"), req.GetQueryInt64("updated"))
	empresaID := core.Coalesce(req.GetQueryInt("empresa_id"), req.Usuario.EmpresaID)

	if empresaID <= 0 {
		return req.MakeErr("Empresa inválida para obtener índice de productos.")
	}

	currentCacheKey := productSortedIDsCacheKey + "_current"
	cacheRows, cacheQueryErr := core.GetCacheByKeys(empresaID, currentCacheKey)
	if cacheQueryErr != nil {
		return req.MakeErr("Error al obtener metadata actual del índice de productos.", cacheQueryErr)
	}
	currentUpdated := int32(0)
	if len(cacheRows) > 0 {
		currentUpdated = cacheRows[0].Updated
	}

	const staleWindowSunix = int32((20 * 60) / 2) // 20 minutes converted to SUnix units.
	nowSunixTime := core.SUnixTime()
	shouldRebuild := currentUpdated <= 0 || (nowSunixTime-currentUpdated) >= staleWindowSunix
	if shouldRebuild {
		buildOutput, buildErr := BuildProductosSearchIndex(empresaID)
		if buildErr != nil {
			return req.MakeErr("Error al reconstruir el índice de productos.", buildErr)
		}
		currentUpdated = buildOutput.IndexBuild.BuildSunixTime
	}

	indexS3Path := core.Concatn("public/c", empresaID, "_productos.idx")
	core.Log("GetProductsIndex:: returning idx path and version",
		"| empresaID:", empresaID,
		"| clientUpdated:", clientUpdated,
		"| currentUpdated:", currentUpdated,
		"| rebuilt:", shouldRebuild,
		"| path:", indexS3Path)
	return req.MakeResponse(map[string]any{
		"path":    indexS3Path,
		"updated": currentUpdated,
	})
}

func GetProductsIndexDelta(req *core.HandlerArgs) core.HandlerResponse {

	updated := req.GetQueryInt("productos")
	empresaID := req.GetQueryInt("empresa-id")
	if updated == 0 {
		cacheRecords, err := core.GetCacheByKeys(empresaID, productSortedIDsCacheKey+"_current")
		if err != nil {
			return req.MakeErr(err)
		}

		core.Log("Cache records::", len(cacheRecords))
		if len(cacheRecords) == 0 {
			return req.MakeResponse([]string{})
		}

		updated = cacheRecords[0].Updated
	}

	productosDelta := []negocioTypes.Producto{}
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

	marcasCategorias := []negocioTypes.ListaCompartidaRegistro{}
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

func buildInputFromProductosData(productosData ProductosData) (index_builder.BuildInput, error) {
	if len(productosData.Productos) == 0 {
		return index_builder.BuildInput{}, fmt.Errorf("no se encontraron productos activos para indexar")
	}

	productIndexRecords := make([]index_builder.RecordInput, 0, len(productosData.Productos))
	for _, currentProduct := range productosData.Productos {
		// Validate source integrity before encoding because record IDs are persisted in section payloads.
		if currentProduct.ID <= 0 {
			return index_builder.BuildInput{}, fmt.Errorf("producto con ID inválido=%d", currentProduct.ID)
		}
		productIndexRecords = append(productIndexRecords, index_builder.RecordInput{
			ID:            currentProduct.ID,
			Text:          currentProduct.Nombre,
			BrandID:       currentProduct.MarcaID,
			CategoriesIDs: append([]int32(nil), currentProduct.CategoriasIDs...),
		})
	}

	brandIndexRecords := make([]index_builder.RecordInput, 0, len(productosData.Marcas)+1)
	hasSentinelBrandIDZero := false
	for _, currentBrand := range productosData.Marcas {
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

	categoryIndexRecords := make([]index_builder.RecordInput, 0, len(productosData.Categorias))
	for _, currentCategory := range productosData.Categorias {
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
func BuildProductosSearchIndexNoPersist(empresaID int32) (*ProductosIndexBuildOutput, error) {
	if empresaID <= 0 {
		return nil, fmt.Errorf("empresa ID inválido para construir el índice")
	}

	productosData, loadErr := loadIndexerSourceDataWithoutCache(empresaID)
	if loadErr != nil {
		return nil, loadErr
	}
	buildInput, buildInputErr := buildInputFromProductosData(productosData)
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

// BuildProductosSearchIndex builds both index passes from productos:
// 1) generic text index by product ID + Nombre
// 2) taxonomy pass by querying brand/category names from shared lists.
func BuildProductosSearchIndex(empresaID int32) (*ProductosIndexBuildOutput, error) {
	if empresaID <= 0 {
		return nil, fmt.Errorf("empresa ID inválido para construir el índice")
	}

	productosData, loadErr := loadIndexerSourceDataWithCache(empresaID)
	if loadErr != nil {
		return nil, loadErr
	}
	buildInput, buildInputErr := buildInputFromProductosData(productosData)
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

	indexFileName := "c" + fmt.Sprintf("%v", empresaID) + "_products.idx"

	if uploadErr := cloud.SaveFile(cloud.SaveFileArgs{
		Bucket:      core.Env.S3_BUCKET,
		Path:        "live",
		Name:        indexFileName,
		FileContent: combinedIndexBytes,
		ContentType: "application/octet-stream",
	}); uploadErr != nil {
		return nil, fmt.Errorf("error al guardar índice productos en S3 (public/%s): %w", indexFileName, uploadErr)
	}

	if persistUpdatedCacheErr := persistProductosIndexUpdatedCache(empresaID, buildSunixTime); persistUpdatedCacheErr != nil {
		return nil, persistUpdatedCacheErr
	}
	core.Log("BuildIndex:: archivo índice actualizado:", "| s3:", "public/"+indexFileName)

	return &ProductosIndexBuildOutput{
		IndexBuild: stageOneIndexResult,
	}, nil
}

func persistProductosIndexUpdatedCache(empresaID int32, updatedSunixTime int32) error {
	if empresaID <= 0 {
		return fmt.Errorf("empresa ID inválido para guardar updated de índice de productos")
	}
	if updatedSunixTime <= 0 {
		return fmt.Errorf("updated sunix inválido para guardar updated de índice de productos")
	}

	cacheRowCurrent := core.Cache{
		EmpresaID: empresaID,
		ID:        core.BasicHashInt(productSortedIDsCacheKey + "_current"),
		Key:       productSortedIDsCacheKey + "_current",
		// Keep cache metadata timestamp equal to index header build timestamp.
		Updated: updatedSunixTime,
	}

	if insertErr := db.Insert(&[]core.Cache{cacheRowCurrent}); insertErr != nil {
		core.Log(insertErr)
		return fmt.Errorf("error al guardar updated de índice de productos en cache (empresa=%d): %w", empresaID, insertErr)
	}

	core.Log("BuildIndex:: updated cache actualizado", "| key:", cacheRowCurrent.Key, "| updated:", cacheRowCurrent.Updated)
	return nil
}

type ProductosData struct {
	Productos  []negocioTypes.Producto
	Categorias []negocioTypes.ListaCompartidaRegistro
	Marcas     []negocioTypes.ListaCompartidaRegistro
}

func loadIndexerSourceDataWithCache(
	empresaID int32,
) (ProductosData, error) {
	cacheFilePath := filepath.Join("tmp", fmt.Sprintf("productos-indexer-%d.gob", empresaID))
	if cachedPayload, cacheHit, cacheErr := loadProductosIndexerCache(cacheFilePath); cacheErr == nil && cacheHit {
		core.Log("BuildIndex:: cache hit:", cacheFilePath, "| productos:", len(cachedPayload.Productos))
		return cachedPayload, nil
	}

	productos := []negocioTypes.Producto{}
	query := db.Query(&productos)
	query.Select(query.ID, query.Nombre, query.CategoriasIDs, query.MarcaID).
		EmpresaID.Equals(empresaID).
		Status.GreaterThan(0)
	if queryErr := query.Exec(); queryErr != nil {
		return ProductosData{}, fmt.Errorf("error al obtener productos para indexado: %w", queryErr)
	}

	marcas := []negocioTypes.ListaCompartidaRegistro{}
	categorias := []negocioTypes.ListaCompartidaRegistro{}
	// Query all active categories and brands to avoid large clustering-key IN cartesian products.
	listRows := []negocioTypes.ListaCompartidaRegistro{}
	listQuery := db.Query(&listRows)
	listQuery.Select(listQuery.ID, listQuery.Nombre, listQuery.ListaID).
		EmpresaID.Equals(empresaID).
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

	productosData := ProductosData{
		Productos:  productos,
		Marcas:     marcas,
		Categorias: categorias,
	}

	// Persist the full source payload as a short-lived cache to avoid repeated DB scans.
	if mkdirErr := os.MkdirAll(filepath.Dir(cacheFilePath), 0o755); mkdirErr != nil {
		core.Log("BuildIndex:: no se pudo crear carpeta cache:", mkdirErr)
		return productosData, nil
	}
	cacheFile, createErr := os.Create(cacheFilePath)
	if createErr != nil {
		core.Log("BuildIndex:: no se pudo crear archivo cache:", createErr)
		return productosData, nil
	}
	defer cacheFile.Close()

	cacheEncoder := gob.NewEncoder(cacheFile)
	if encodeErr := cacheEncoder.Encode(productosData); encodeErr != nil {
		core.Log("BuildIndex:: no se pudo serializar cache gob:", encodeErr)
	}

	core.Log("loadIndexerSourceDataWithCache:: marcas:", len(marcas), "| categorías:", len(categorias))
	return productosData, nil
}

func loadIndexerSourceDataWithoutCache(
	empresaID int32,
) (ProductosData, error) {
	productos := []negocioTypes.Producto{}
	query := db.Query(&productos)
	query.Select(query.ID, query.Nombre, query.CategoriasIDs, query.MarcaID).
		EmpresaID.Equals(empresaID).
		Status.GreaterThan(0)
	if queryErr := query.Exec(); queryErr != nil {
		return ProductosData{}, fmt.Errorf("error al obtener productos para indexado: %w", queryErr)
	}

	marcas := []negocioTypes.ListaCompartidaRegistro{}
	categorias := []negocioTypes.ListaCompartidaRegistro{}
	// Query all active categories and brands to avoid large clustering-key IN cartesian products.
	listRows := []negocioTypes.ListaCompartidaRegistro{}
	listQuery := db.Query(&listRows)
	listQuery.Select(listQuery.ID, listQuery.Nombre, listQuery.ListaID).
		EmpresaID.Equals(empresaID).
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

	productosData := ProductosData{
		Productos:  productos,
		Marcas:     marcas,
		Categorias: categorias,
	}

	core.Log("loadIndexerSourceDataWithoutCache:: marcas:", len(marcas), "| categorías:", len(categorias))
	return productosData, nil
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
