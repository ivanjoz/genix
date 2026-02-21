package handlers

import (
	"app/aws"
	"app/core"
	"app/db"
	"app/libs/index_builder"
	s "app/types"
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
	if len(productosData.Productos) == 0 {
		return nil, fmt.Errorf("no se encontraron productos activos para indexar")
	}

	productIndexRecords := make([]index_builder.RecordInput, 0, len(productosData.Productos))
	for _, producto := range productosData.Productos {
		if producto.ID <= 0 {
			return nil, fmt.Errorf("producto con ID inválido=%d", producto.ID)
		}

		productIndexRecords = append(productIndexRecords, index_builder.RecordInput{
			ID:            producto.ID,
			Text:          producto.Nombre,
			BrandID:       producto.MarcaID,
			CategoriesIDs: append([]int32(nil), producto.CategoriasIDs...),
		})
	}

	brandIndexRecords := make([]index_builder.RecordInput, 0, len(productosData.Marcas))
	hasSentinelBrandZero := false
	for _, marca := range productosData.Marcas {
		if marca.ID == 0 {
			hasSentinelBrandZero = true
		}
		brandIndexRecords = append(brandIndexRecords, index_builder.RecordInput{
			ID: marca.ID, Text: marca.Nombre,
		})
	}
	// Keep MarcaID=0 as a valid sentinel so stage 2 can index products without explicit brand.
	if !hasSentinelBrandZero {
		brandIndexRecords = append(brandIndexRecords, index_builder.RecordInput{
			ID: 0, Text: "Sin marca",
		})
	}

	categoryIndexRecords := make([]index_builder.RecordInput, 0, len(productosData.Categorias))
	for _, categoria := range productosData.Categorias {
		categoryIndexRecords = append(categoryIndexRecords, index_builder.RecordInput{
			ID: categoria.ID, Text: categoria.Nombre,
		})
	}

	core.Log("BuildIndex:: productos:", len(productosData.Productos),
		"| marcas únicas:", len(brandIndexRecords),
		"| categorías únicas:", len(categoryIndexRecords))
	buildInput := index_builder.BuildInput{
		Products: productIndexRecords, Brands: brandIndexRecords, Categories: categoryIndexRecords,
	}
	indexBuildArtifacts, buildErr := index_builder.BuildProductosIndex(buildInput)
	if buildErr != nil {
		return nil, fmt.Errorf("error al construir índice de productos: %w", buildErr)
	}
	core.Log("BuildIndex:: text optimization",
		"| changed:", indexBuildArtifacts.OptimizationStats.ChangedProducts,
		"| fallbackOriginal:", indexBuildArtifacts.OptimizationStats.FallbackOriginalProducts)

	stageOneIndexResult := indexBuildArtifacts
	core.Log("BuildIndex:: stage1.sorted_ids_count:", len(stageOneIndexResult.SortedIDs))
	core.Log("BuildIndex:: stage1.dictionary_count:", stageOneIndexResult.Stats.DictionaryCount)
	core.Log("BuildIndex:: stage1.shapes_bytes:", stageOneIndexResult.Stats.ShapesBytes)
	core.Log("BuildIndex:: stage1.content_bytes:", stageOneIndexResult.Stats.ContentBytes)
	core.Log("BuildIndex:: stage1.total_bytes:", stageOneIndexResult.Stats.TotalBytes)

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

	indexFileName := "c" + fmt.Sprintf("%v",empresaID) + "_products.idx"
	
	if uploadErr := aws.SaveFile(aws.SaveFileArgs{
		Bucket:      core.Env.S3_BUCKET,
		Path:        "public",
		Name:        indexFileName,
		FileContent: combinedIndexBytes,
		ContentType: "application/octet-stream",
	}); uploadErr != nil {
		return nil, fmt.Errorf("error al guardar índice productos en S3 (public/%s): %w", indexFileName, uploadErr)
	}

	if persistSortedIDsErr := persistProductosSortedIDsCache(empresaID, stageOneIndexResult.SortedIDs, buildSunixTime); persistSortedIDsErr != nil {
		return nil, persistSortedIDsErr
	}
	core.Log("BuildIndex:: archivo índice actualizado:","| s3:", "public/" + indexFileName)

	return &ProductosIndexBuildOutput{
		IndexBuild: stageOneIndexResult,
	}, nil
}

func persistProductosSortedIDsCache(empresaID int32, sortedProductIDs []int32, updatedSunixTime int32) error {
	if empresaID <= 0 {
		return fmt.Errorf("empresa ID inválido para guardar sorted IDs de productos")
	}
	if len(sortedProductIDs) == 0 {
		return fmt.Errorf("no hay sorted IDs de productos para guardar")
	}
	if updatedSunixTime <= 0 {
		return fmt.Errorf("updated sunix inválido para guardar sorted IDs de productos")
	}

	cacheKey := core.Concatn(productSortedIDsCacheKey, updatedSunixTime)
	// Cache row ID is deterministic per key+version, isolated by EmpresaID partition.
	cacheRecordID := core.BasicHashInt(cacheKey)

	cacheRow := core.Cache{
		EmpresaID:    empresaID,
		ID:           cacheRecordID,
		Key:          cacheKey,
		ContentBytes: s.EncodeIDs(sortedProductIDs),
		// Keep cache metadata timestamp equal to index header build timestamp.
		Updated: updatedSunixTime,
	}

	cacheRowCurrent := core.Cache{
		EmpresaID: empresaID,
		ID:        core.BasicHashInt(productSortedIDsCacheKey + "_current"),
		Key:       productSortedIDsCacheKey + "_current",
		Updated:   updatedSunixTime,
	}

	if insertErr := db.Insert(&[]core.Cache{cacheRow, cacheRowCurrent}); insertErr != nil {
		core.Log(insertErr)
		return fmt.Errorf("error al guardar sorted IDs de productos en cache (empresa=%d, cache_id=%d): %w", empresaID, cacheRecordID, insertErr)
	}

	core.Log("BuildIndex:: sorted IDs cache actualizado",
		"| key:", cacheKey,
		"| bytes:", len(cacheRow.ContentBytes))
	return nil
}

type ProductosData struct {
	Productos  []s.Producto
	Categorias []s.ListaCompartidaRegistro
	Marcas     []s.ListaCompartidaRegistro
}

func loadIndexerSourceDataWithCache(
	empresaID int32,
) (ProductosData, error) {
	cacheFilePath := filepath.Join("tmp", fmt.Sprintf("productos-indexer-%d.gob", empresaID))
	if cachedPayload, cacheHit, cacheErr := loadProductosIndexerCache(cacheFilePath); cacheErr == nil && cacheHit {
		core.Log("BuildIndex:: cache hit:", cacheFilePath, "| productos:", len(cachedPayload.Productos))
		return cachedPayload, nil
	}

	productos := []s.Producto{}
	query := db.Query(&productos)
	query.Select(query.ID, query.Nombre, query.CategoriasIDs, query.MarcaID).
		EmpresaID.Equals(empresaID).
		Status.GreaterThan(0)
	if queryErr := query.Exec(); queryErr != nil {
		return ProductosData{}, fmt.Errorf("error al obtener productos para indexado: %w", queryErr)
	}

	marcas := []s.ListaCompartidaRegistro{}
	categorias := []s.ListaCompartidaRegistro{}
	// Query all active categories and brands to avoid large clustering-key IN cartesian products.
	listRows := []s.ListaCompartidaRegistro{}
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
