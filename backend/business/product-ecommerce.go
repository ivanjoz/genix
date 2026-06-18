package business

import (
	businessTypes "app/business/types"
	"app/cloud"
	"app/core"
	"app/db"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

// Ecommerce product sync replaces the deprecated binary .idx index. The client first downloads a
// plain pipe-separated snapshot file (products-c<companyID>.db, 30-min CDN cache) to seed its local
// delta caches, then calls GetProductsEcommerce for the incremental delta since its watermark.

const (
	// Shared-list IDs distinguish categorías from marcas inside the single shared-list table.
	ecommerceSharedListCategoryID = int32(1)
	ecommerceSharedListBrandID     = int32(2)

	// cache_global group IDs registering, per company, the latest searchable change watermark.
	cacheGroupProducts  = int16(1)
	cacheGroupBrands     = int16(2)
	cacheGroupCategories = int16(3)

	// core.Cache key holding the watermarks observed the last time the .db file was built.
	productsDbBuiltCacheKey = "products_db_built"

	// CDN cache lifetime for the snapshot file.
	productsDbCacheControl = "public, max-age=1800"
)

// productsDbFileName is the per-company snapshot object name under the "live/" CDN path.
func productsDbFileName(companyID int32) string {
	return fmt.Sprintf("products-c%d.db", companyID)
}

// stripPipesAndNewlines removes the field separator and any row-breaking characters from free text
// so a name can never corrupt the pipe/line structure of the .db file.
var dbTextSanitizer = strings.NewReplacer("|", " ", "\n", " ", "\r", " ", "\t", " ")

func sanitizeDbText(text string) string {
	return strings.TrimSpace(dbTextSanitizer.Replace(text))
}

// GetProductsEcommerce returns the incremental delta of products, marcas and categorías changed
// since the client's per-table watermark. Response keys (productos/marcas/categorias) are the same
// names the client sends back as watermark query params on the next call (multi-table delta cache).
func GetProductsEcommerce(req *core.HandlerArgs) core.HandlerResponse {
	companyID := core.Coalesce(req.GetQueryInt("cid"), req.GetQueryInt("company-id"))
	if companyID <= 0 {
		return req.MakeErr("Company inválida para obtener productos de ecommerce.")
	}

	productosWatermark := req.GetQueryInt("productos")
	marcasWatermark := req.GetQueryInt("marcas")
	categoriasWatermark := req.GetQueryInt("categorias")

	// Best-effort lazy rebuild: refresh the snapshot file when source data advanced past the last
	// build. The client sets missingFile=1 when the CDN file was absent, which forces a rebuild
	// even if the stamped watermarks say the build is current. Never fail the delta response if
	// the rebuild errors; the cron is the durable path.
	forceRebuild := req.GetQueryInt("missingFile") > 0
	if rebuildErr := maybeRebuildProductsDbFile(companyID, forceRebuild); rebuildErr != nil {
		core.Log("GetProductsEcommerce:: lazy rebuild skipped", "| companyID:", companyID, "| err:", rebuildErr)
	}

	productos := []businessTypes.Product{}
	marcas := []businessTypes.SharedListRecord{}
	categorias := []businessTypes.SharedListRecord{}
	errGroup := errgroup.Group{}

	// Products delta: keyed by Updated. Only the [company_id, updated] view supports a range scan
	// with full-column projection (NameUpdated is a local index → ALLOW FILTERING). Using Updated
	// also means price-only edits reach already-synced clients.
	errGroup.Go(func() error {
		query := db.Query(&productos)
		query.Select(query.ID, query.Name, query.CategoryIDs, query.BrandID, query.Price, query.FinalPrice, query.ImageMain, query.Updated, query.Status).
			CompanyID.Equals(companyID)
		if productosWatermark > 0 {
			query.Updated.GreaterThan(productosWatermark)
		} else {
			query.Status.GreaterEqual(1)
		}
		if err := query.Exec(); err != nil {
			return fmt.Errorf("error al obtener productos ecommerce: %v", err)
		}
		return nil
	})

	// Marcas + categorías deltas: shared-list rows keyed by Updated, split by ListID.
	errGroup.Go(func() error {
		rows, err := querySharedListDelta(companyID, ecommerceSharedListBrandID, marcasWatermark)
		if err != nil {
			return err
		}
		marcas = rows
		return nil
	})
	errGroup.Go(func() error {
		rows, err := querySharedListDelta(companyID, ecommerceSharedListCategoryID, categoriasWatermark)
		if err != nil {
			return err
		}
		categorias = rows
		return nil
	})

	if err := errGroup.Wait(); err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(map[string]any{
		"productos":  &productos,
		"marcas":     &marcas,
		"categorias": &categorias,
	})
}

// querySharedListDelta returns the delta for a single shared list (marca or categoría). On first
// sync (watermark == 0) it returns active rows only; on delta it includes Status=0 evictions.
func querySharedListDelta(companyID, listID, watermark int32) ([]businessTypes.SharedListRecord, error) {
	records := []businessTypes.SharedListRecord{}
	query := db.Query(&records)
	query.Select(query.ID, query.Name, query.Updated, query.Status).
		CompanyID.Equals(companyID).
		ListID.Equals(listID)
	if watermark > 0 {
		query.Updated.GreaterThan(watermark)
	} else {
		query.Status.Equals(1)
	}
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al obtener lista %d ecommerce: %v", listID, err)
	}
	return records, nil
}

// buildProductsDbFile renders the 3-section snapshot from active products/marcas/categorías and
// uploads it to live/products-c<companyID>.db. Returns the max watermark observed per section.
func buildProductsDbFile(companyID int32) (productosWm, marcasWm, categoriasWm int32, err error) {
	if companyID <= 0 {
		return 0, 0, 0, fmt.Errorf("company ID inválido para construir el archivo de productos")
	}

	productos := []businessTypes.Product{}
	sharedRows := []businessTypes.SharedListRecord{}
	loadGroup := errgroup.Group{}

	loadGroup.Go(func() error {
		query := db.Query(&productos)
		query.Select(query.ID, query.Name, query.CategoryIDs, query.BrandID, query.Price, query.FinalPrice, query.ImageMain, query.Updated, query.Status).
			CompanyID.Equals(companyID).
			Status.GreaterThan(0)
		if execErr := query.Exec(); execErr != nil {
			return fmt.Errorf("error al obtener productos para archivo: %w", execErr)
		}
		return nil
	})
	loadGroup.Go(func() error {
		query := db.Query(&sharedRows)
		query.Select(query.ID, query.Name, query.ListID, query.Updated, query.Status).
			CompanyID.Equals(companyID).
			ListID.In(ecommerceSharedListCategoryID, ecommerceSharedListBrandID).
			Status.GreaterThan(0).
			AllowFilter()
		if execErr := query.Exec(); execErr != nil {
			return fmt.Errorf("error al obtener marcas/categorías para archivo: %w", execErr)
		}
		return nil
	})
	if waitErr := loadGroup.Wait(); waitErr != nil {
		return 0, 0, 0, waitErr
	}

	fileBuilder := strings.Builder{}

	// >>>productos: ID|Name|categoriesIDs(csv)|BrandID|Price|FinalPrice|Image|Updated|Status
	// Image is the first image's name only (flat string), enough to render a search-card thumbnail.
	fileBuilder.WriteString(">>>productos\n")
	for index := range productos {
		product := &productos[index]
		if product.Updated > productosWm {
			productosWm = product.Updated
		}
		categoryIDsCsv := make([]string, 0, len(product.CategoryIDs))
		for _, categoryID := range product.CategoryIDs {
			categoryIDsCsv = append(categoryIDsCsv, strconv.Itoa(int(categoryID)))
		}
		// TODO(frontend pass): emit the main imageID; the frontend will rebuild the
		// "<companyID>_<imageID>.avif" URL from it. Empty when the product has no image.
		imageName := ""
		if product.ImageMain > 0 {
			imageName = strconv.Itoa(int(product.ImageMain))
		}
		fileBuilder.WriteString(strconv.Itoa(int(product.ID)))
		fileBuilder.WriteByte('|')
		fileBuilder.WriteString(sanitizeDbText(product.Name))
		fileBuilder.WriteByte('|')
		fileBuilder.WriteString(strings.Join(categoryIDsCsv, ","))
		fileBuilder.WriteByte('|')
		fileBuilder.WriteString(strconv.Itoa(int(product.BrandID)))
		fileBuilder.WriteByte('|')
		fileBuilder.WriteString(strconv.Itoa(int(product.Price)))
		fileBuilder.WriteByte('|')
		fileBuilder.WriteString(strconv.Itoa(int(product.FinalPrice)))
		fileBuilder.WriteByte('|')
		fileBuilder.WriteString(sanitizeDbText(imageName))
		fileBuilder.WriteByte('|')
		fileBuilder.WriteString(strconv.Itoa(int(product.Updated)))
		fileBuilder.WriteByte('|')
		fileBuilder.WriteString(strconv.Itoa(int(product.Status)))
		fileBuilder.WriteByte('\n')
	}

	// Marcas / categorías rows share the same column layout: ID|Name|Updated|Status.
	marcasBuilder := strings.Builder{}
	categoriasBuilder := strings.Builder{}
	writeSharedRow := func(builder *strings.Builder, row *businessTypes.SharedListRecord) {
		builder.WriteString(strconv.Itoa(int(row.ID)))
		builder.WriteByte('|')
		builder.WriteString(sanitizeDbText(row.Name))
		builder.WriteByte('|')
		builder.WriteString(strconv.Itoa(int(row.Updated)))
		builder.WriteByte('|')
		builder.WriteString(strconv.Itoa(int(row.Status)))
		builder.WriteByte('\n')
	}
	for index := range sharedRows {
		row := &sharedRows[index]
		if row.ListID == ecommerceSharedListBrandID {
			if row.Updated > marcasWm {
				marcasWm = row.Updated
			}
			writeSharedRow(&marcasBuilder, row)
		} else if row.ListID == ecommerceSharedListCategoryID {
			if row.Updated > categoriasWm {
				categoriasWm = row.Updated
			}
			writeSharedRow(&categoriasBuilder, row)
		}
	}

	fileBuilder.WriteString(">>>marcas\n")
	fileBuilder.WriteString(marcasBuilder.String())
	fileBuilder.WriteString(">>>categorias\n")
	fileBuilder.WriteString(categoriasBuilder.String())

	uploadArgs := cloud.SaveFileArgs{
		Path:         "live",
		Name:         productsDbFileName(companyID),
		FileContent:  []byte(fileBuilder.String()),
		ContentType:  "text/plain; charset=utf-8",
		CacheControl: productsDbCacheControl,
	}
	if uploadErr := cloud.SaveFile(uploadArgs); uploadErr != nil {
		return 0, 0, 0, fmt.Errorf("error al guardar archivo productos (live/%s): %w", uploadArgs.Name, uploadErr)
	}

	core.Log("buildProductsDbFile:: uploaded",
		"| companyID:", companyID,
		"| productos:", len(productos),
		"| productosWm:", productosWm,
		"| marcasWm:", marcasWm,
		"| categoriasWm:", categoriasWm)
	return productosWm, marcasWm, categoriasWm, nil
}

// maybeRebuildProductsDbFile rebuilds + reuploads the snapshot only when any source group watermark
// (productos/marcas/categorías) advanced beyond the watermarks stamped at the last build.
func maybeRebuildProductsDbFile(companyID int32, force bool) error {
	sourceProductos := latestGroupWatermark(cacheGroupProducts, companyID)
	sourceMarcas := latestGroupWatermark(cacheGroupBrands, companyID)
	sourceCategorias := latestGroupWatermark(cacheGroupCategories, companyID)

	builtProductos, builtMarcas, builtCategorias, hasBuilt := loadBuiltWatermarks(companyID)

	// Rebuild when forced (client reported the CDN file missing), on first build, or whenever any
	// source watermark moved forward.
	needsRebuild := force ||
		!hasBuilt ||
		sourceProductos > builtProductos ||
		sourceMarcas > builtMarcas ||
		sourceCategorias > builtCategorias
	if !needsRebuild {
		return nil
	}

	productosWm, marcasWm, categoriasWm, buildErr := buildProductsDbFile(companyID)
	if buildErr != nil {
		return buildErr
	}
	return saveBuiltWatermarks(companyID, productosWm, marcasWm, categoriasWm)
}

// latestGroupWatermark reads a single company's registered change watermark for a cache group.
func latestGroupWatermark(groupID int16, companyID int32) int32 {
	rows, err := core.GetCacheGlobal(groupID, companyID)
	if err != nil || len(rows) == 0 {
		return 0
	}
	return rows[0].Updated
}

// loadBuiltWatermarks reads the "p,m,c" watermarks stamped when the .db file was last built.
func loadBuiltWatermarks(companyID int32) (productosWm, marcasWm, categoriasWm int32, found bool) {
	rows, err := core.GetCacheByKeys(companyID, productsDbBuiltCacheKey)
	if err != nil || len(rows) == 0 {
		return 0, 0, 0, false
	}
	parts := strings.Split(rows[0].Content, ",")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	p, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	c, _ := strconv.Atoi(parts[2])
	return int32(p), int32(m), int32(c), true
}

// saveBuiltWatermarks stamps the watermarks observed during the build so future calls can skip
// rebuilding until the source advances again.
func saveBuiltWatermarks(companyID, productosWm, marcasWm, categoriasWm int32) error {
	row := core.Cache{
		CompanyID: companyID,
		ID:        core.BasicHashInt(productsDbBuiltCacheKey),
		Key:       productsDbBuiltCacheKey,
		Content:   fmt.Sprintf("%d,%d,%d", productosWm, marcasWm, categoriasWm),
		Updated:   core.SUnixTime(),
	}
	if err := db.Insert(&[]core.Cache{row}); err != nil {
		return fmt.Errorf("error al guardar watermarks de archivo productos (company=%d): %w", companyID, err)
	}
	return nil
}
