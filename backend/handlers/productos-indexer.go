package handlers

import (
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

const (
	productSharedListCategoriaID = int32(1)
	productSharedListMarcaID     = int32(2)
	productIndexerCacheTTL       = 15 * time.Minute
)

type ProductosIndexBuildOutput struct {
	TextIndexResult     *index_builder.BuildResult
	TaxonomyIndexResult *index_builder.TaxonomyBuildResult
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

	core.Log("BuildProductosSearchIndex:: productos:", len(productosData.Productos),
		"| marcas únicas:", len(brandIndexRecords),
		"| categorías únicas:", len(categoryIndexRecords))

	stageOneOptions := index_builder.DefaultOptions()
	stageOneIndexResult, stageOneErr := index_builder.Build(productIndexRecords, stageOneOptions)
	if stageOneErr != nil {
		return nil, fmt.Errorf("error al construir índice base de texto: %w", stageOneErr)
	}
	core.Log("BuildProductosSearchIndex:: stage 1 completado",
		"| sorted IDs:", len(stageOneIndexResult.SortedIDs),
		"| dict:", stageOneIndexResult.Stats.DictionaryCount,
		"| shapes bytes:", stageOneIndexResult.Stats.ShapesBytes,
		"| content bytes:", stageOneIndexResult.Stats.ContentBytes,
		"| total bytes:", stageOneIndexResult.Stats.TotalBytes)

	stageTwoInput := index_builder.BuildInput{
		Products: productIndexRecords, Brands: brandIndexRecords, Categories: categoryIndexRecords,
	}

	stageTwoIndexResult, stageTwoErr := index_builder.BuildTaxonomySecondPass(stageOneIndexResult.SortedIDs, stageTwoInput)
	if stageTwoErr != nil {
		return nil, fmt.Errorf("error al construir índice de taxonomía: %w", stageTwoErr)
	}

	core.Log("BuildProductosSearchIndex:: stage 2 completado",
		"| brandIDs:", len(stageTwoIndexResult.BrandIDs),
		"| categoryIDs:", len(stageTwoIndexResult.CategoryIDs),
		"| brandIdxU8:", len(stageTwoIndexResult.ProductBrandIndexesU8),
		"| brandIdxU16:", len(stageTwoIndexResult.ProductBrandIndexesU16),
		"| catCountPacked:", len(stageTwoIndexResult.ProductCategoryCount),
		"| catIndexes:", len(stageTwoIndexResult.ProductCategoryIndexes))
	core.Log("BuildProductosSearchIndex:: completado")

	return &ProductosIndexBuildOutput{
		TextIndexResult:     stageOneIndexResult,
		TaxonomyIndexResult: stageTwoIndexResult,
	}, nil
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
		core.Log("BuildProductosSearchIndex:: cache hit:", cacheFilePath, "| productos:", len(cachedPayload.Productos))
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
		core.Log("BuildProductosSearchIndex:: no se pudo crear carpeta cache:", mkdirErr)
		return productosData, nil
	}
	cacheFile, createErr := os.Create(cacheFilePath)
	if createErr != nil {
		core.Log("BuildProductosSearchIndex:: no se pudo crear archivo cache:", createErr)
		return productosData, nil
	}
	defer cacheFile.Close()

	cacheEncoder := gob.NewEncoder(cacheFile)
	if encodeErr := cacheEncoder.Encode(productosData); encodeErr != nil {
		core.Log("BuildProductosSearchIndex:: no se pudo serializar cache gob:", encodeErr)
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
