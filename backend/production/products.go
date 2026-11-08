package production

import (
	business "app/business/types"
	"app/cloud"
	config "app/config/types"
	"app/core"
	"app/db"
	"app/production/types"
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

func GetProducts(req *core.HandlerArgs) core.HandlerResponse {
	loggedProducts := []types.Product{}
	query := db.Query(&loggedProducts).CompanyID.Equals(req.User.CompanyID)
	query.Exclude(query.Stock, query.StockStatus, query.CompanyID, query.Created, query.CreatedBy, query.NameHash)
	// A first sync lists the active products; later syncs read the log, which evicts deleted ones.
	query.Delta(req.GetDeltaCursor("records"), types.ProductStatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los productos:", err)
	}

	// The catalog is Status=1 and nothing else, but the log names every status: a product that
	// became a supply/material (2) comes back as a live record. It must go as an id to drop, never
	// as a body: a supply carries a purchase Price and no FinalPrice, and every screen that reads
	// this cache treats what it finds there as sellable.
	// Filtered in place: a second slice copied every product of a first sync once more.
	productos := loggedProducts[:0]
	evictedProductIDs := []int32{}
	for _, product := range loggedProducts {
		if product.Metadata == nil && product.Status != types.ProductStatusActive {
			evictedProductIDs = append(evictedProductIDs, product.ID)
			continue
		}
		productos = append(productos, product)
	}

	response := map[string]any{
		"records":             &productos,
		"records_IDsToRemove": &evictedProductIDs,
	}

	return req.MakeResponse(&response)
}

func GetProductTextSearch(req *core.HandlerArgs) core.HandlerResponse {
	// Public endpoint (p- prefix): no authenticated user, so the company comes from the query.
	companyID := core.Coalesce(req.GetQueryInt("cid"), req.GetQueryInt("company-id"))
	if companyID <= 0 {
		return req.MakeErr("Company inválida para la búsqueda de productos.")
	}
	query := req.GetQuery("q")
	if len(query) < 2 {
		return req.MakeErr("La búsqueda debe tener al menos 2 caracteres.")
	}
	limit := int(req.GetQueryInt("limit"))
	if limit <= 0 {
		limit = 50
	}

	// Only active products are indexed. Return only ids + scores (no record
	// bodies) — the client resolves names from its by-id cache.
	matches, err := db.SearchTextIDs[types.Product](companyID, query, limit, 0)
	if err != nil {
		return req.MakeErr("Error en la búsqueda de texto de productos:", err)
	}

	return core.MakeResponse(req, &matches)
}

func GetProductsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	// Public route (p- prefix): no token is parsed, so the company comes from the query the app and the
	// storefront both append (?company-id=). Either is fine: the catalog is public.
	cachedIDsRequest, err := req.GetCachedIDsRequest(core.Coalesce(req.GetQueryInt("cid"), req.GetQueryInt("company-id")))
	if err != nil {
		return req.MakeErr(err)
	}
	partitions, err := db.QueryCachedIDs[types.Product](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener los productos.", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}

func PostProducts(req *core.HandlerArgs) core.HandlerResponse {
	// db.SetDebugLogging(2)

	productos := []types.Product{}
	if err := json.Unmarshal([]byte(*req.Body), &productos); err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	nameHashToName := make(map[int32]*types.Product, len(productos))
	// SelfParse each product to populate NombreHash and fail fast on duplicate names in this payload.
	for i := range productos {
		e := &productos[i]
		if len(e.Name) < 4 {
			return req.MakeErr("Faltan propiedades de en el product.")
		}
		// Preserve incoming ID so frontend can map TempID -> ID after merge/upsert.
		e.TempID = e.ID
		e.CompanyID = req.User.CompanyID
		// A product is bought unless the form says it is made; its accounts follow from Origin,
		// so it never carries an account of its own (that column is a supply's default).
		e.Origin = core.If(e.Origin == 0, types.ProductOriginBought, e.Origin)
		if e.Origin != types.ProductOriginBought && e.Origin != types.ProductOriginMade {
			return req.MakeErr(fmt.Sprintf(`El origen del producto "%s" es inválido.`, e.Name))
		}
		e.AccountCode = 0
		e.SelfParse()
		if previousProduct, duplicate := nameHashToName[e.NameHash]; duplicate {
			return req.MakeErr(fmt.Sprintf("Hay nombres duplicados en la solicitud: %s y %s", previousProduct.Name, e.Name))
		}
		nameHashToName[e.NameHash] = e
	}

	brandNamesByID, err := getProductBrandNames(req.User.CompanyID, productos)
	if err != nil {
		return req.MakeErr(err)
	}
	for i := range productos {
		productos[i].BrandName = brandNamesByID[productos[i].BrandID]
	}

	// The company row is locked first, so name uniqueness is checked against every committed save,
	// then the edited rows: the stored columns this form does not own (stock, images, creation) are
	// copied from the locked row, so a sale or an image upload committing meanwhile is not
	// overwritten, and a rejected divisor leaves no row of the batch persisted.
	nowTime := core.SUnixTime()
	core.Log("PostProductos payload:", len(productos))
	savedProducts := []types.Product{}
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		// A rerun after a deadlock starts again from the payload, not from the IDs a failed
		// attempt assigned.
		savedProducts = slices.Clone(productos)
		return saveProductsIn(transaction, req, savedProducts, nowTime)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	productos = savedProducts

	// Register the company for a products .db rebuild. Keyed on Updated (which advances on every
	// product write, incl. price edits) so the snapshot self-heals on any change, matching the
	// ecommerce delta watermark. NameUpdated is reserved for other purposes.
	if len(productos) > 0 {
		if cacheErr := core.SaveCacheGlobal(business.CacheGroupProducts, req.User.CompanyID, nil, nowTime); cacheErr != nil {
			core.Log("PostProducts:: error registrando cambio de productos para ecommerce", cacheErr)
		}
	}

	return req.MakeResponse(productos)
}

// saveProductsIn writes a PostProducts payload in the caller's transaction: existing products are
// locked and rewritten keeping the columns other writers own, new ones are inserted.
func saveProductsIn(transaction *db.Transaction, req *core.HandlerArgs, productos []types.Product, nowTime int32) error {
	if _, err := config.LockCompanyIn(transaction, req.User.CompanyID); err != nil {
		return err
	}
	if err := resolveProductNames(req.User.CompanyID, productos); err != nil {
		return err
	}

	lockedProducts := []types.Product{}
	for _, product := range productos {
		if product.ID > 0 {
			lockedProducts = append(lockedProducts, types.Product{CompanyID: req.User.CompanyID, ID: product.ID})
		}
	}
	slices.SortFunc(lockedProducts, func(a, b types.Product) int { return cmp.Compare(a.ID, b.ID) })
	if err := db.LockByKeys(transaction, &lockedProducts); err != nil {
		return core.Err("Error al obtener los productos:", err)
	}
	storedByID := make(map[int32]types.Product, len(lockedProducts))
	for _, stored := range lockedProducts {
		storedByID[stored.ID] = stored
	}

	if err := validateProductSubUnitDivisors(productos, storedByID); err != nil {
		return err
	}

	newProducts := []types.Product{}
	newProductIndexes := []int{}
	editedProducts := []types.Product{}
	for i := range productos {
		current := &productos[i]
		current.CompanyID = req.User.CompanyID
		current.Updated = nowTime
		current.UpdatedBy = req.User.ID
		if current.ID == 0 {
			current.Created = nowTime
			current.CreatedBy = req.User.ID
			current.NameUpdated = nowTime
			current.Status = 1
			buildProductPresentations(nil, current)
			newProducts = append(newProducts, *current)
			newProductIndexes = append(newProductIndexes, i)
			continue
		}
		stored, found := storedByID[current.ID]
		if !found {
			return core.Err(fmt.Sprintf(`El producto "%s" (ID=%v) no existe.`, current.Name, current.ID))
		}
		current.Created = stored.Created
		current.CreatedBy = stored.CreatedBy
		current.Stock = stored.Stock
		current.ReservedStock = stored.ReservedStock
		current.StockStatus = stored.StockStatus
		current.CategoriesWithStock = stored.CategoriesWithStock
		current.ImageMain = stored.ImageMain
		current.ImageIDs = stored.ImageIDs
		current.ImageDescriptions = stored.ImageDescriptions
		current.NameUpdated = stored.NameUpdated
		// NameUpdated drives the delta of the name-dependent caches.
		if current.Name != stored.Name || current.BrandID != stored.BrandID || !slices.Equal(current.CategoryIDs, stored.CategoryIDs) {
			current.NameUpdated = nowTime
		}
		buildProductPresentations(&stored, current)
		editedProducts = append(editedProducts, *current)
	}

	if err := db.UpsertIn(transaction, &editedProducts); err != nil {
		return core.Err("Error al actualizar los productos:", err)
	}
	if err := db.InsertIn(transaction, &newProducts); err != nil {
		return core.Err("Error al crear los productos:", err)
	}
	// Hand the assigned IDs back: the response maps each TempID to its ID.
	for newIndex, payloadIndex := range newProductIndexes {
		productos[payloadIndex] = newProducts[newIndex]
	}
	return nil
}

// resolveProductNames refuses a name an active product of the company already has, and gives a new
// product the ID of an inactive one with its name, so re-creating a deleted product revives it.
// The reads are plain: the caller holds the company row's lock, which every product save takes, so
// no other save can add a name between these reads and the commit.
func resolveProductNames(companyID int32, productos []types.Product) error {
	nameHashes := core.SliceSet[int32]{}
	for _, product := range productos {
		nameHashes.Add(product.NameHash)
	}
	productsWithNames := []types.Product{}
	if !nameHashes.IsEmpty() {
		query := db.Query(&productsWithNames)
		query.Select(query.NameHash, query.ID, query.Status).
			CompanyID.Equals(companyID).
			NameHash.In(nameHashes.Values...)
		if err := query.Exec(); err != nil {
			return core.Err("Error al validar los nombres de productos:", err)
		}
	}
	existingProductsByHash := make(map[int32][]types.Product, len(productsWithNames))
	for _, product := range productsWithNames {
		existingProductsByHash[product.NameHash] = append(existingProductsByHash[product.NameHash], product)
	}

	for i := range productos {
		currentProduct := &productos[i]
		existingProducts := existingProductsByHash[currentProduct.NameHash]
		for _, candidate := range existingProducts {
			if candidate.Status > 0 && candidate.ID != currentProduct.ID {
				return core.Err(fmt.Sprintf(`Ya existe un producto activo con el nombre "%s". ID=%v`, currentProduct.Name, candidate.ID))
			}
		}
		if currentProduct.ID == 0 {
			for _, candidate := range existingProducts {
				if candidate.Status == 0 {
					currentProduct.ID = candidate.ID
					break
				}
			}
		}
	}
	return nil
}

// buildProductPresentations merges the incoming presentations into the stored ones: a presentation
// keeps its ID by attribute and name, a new one takes the next ID, and one the payload dropped
// stays stored with Status 0, since sales already reference it.
func buildProductPresentations(stored *types.Product, incoming *types.Product) {
	presentationsByID := map[int16]types.ProductPresentation{}
	storedByName := map[string]types.ProductPresentation{}
	maxPresentationID := int16(0)

	if stored != nil {
		for _, storedPresentation := range stored.Presentations {
			storedByName[core.Concatn(storedPresentation.AtributoID, strings.ToLower(storedPresentation.Name))] = storedPresentation
			maxPresentationID = max(maxPresentationID, storedPresentation.ID)
		}
	}

	for _, incomingPresentation := range incoming.Presentations {
		presentationName := core.Concatn(incomingPresentation.AtributoID, strings.ToLower(incomingPresentation.Name))
		if storedPresentation, ok := storedByName[presentationName]; ok && incomingPresentation.ID != 0 {
			incomingPresentation.ID = storedPresentation.ID
		}
		if incomingPresentation.ID <= 0 {
			maxPresentationID++
			incomingPresentation.ID = maxPresentationID
		}
		presentationsByID[incomingPresentation.ID] = incomingPresentation
	}

	for _, storedPresentation := range storedByName {
		if _, ok := presentationsByID[storedPresentation.ID]; !ok {
			storedPresentation.Status = 0
			presentationsByID[storedPresentation.ID] = storedPresentation
		}
	}
	incoming.Presentations = core.MapToSliceT(presentationsByID)
}

// validateProductSubUnitDivisors enforces the refinement-only rule on Product.SbuQuantity,
// which is the divisor every stored SubQuantity for that product is expressed in.
//
// A product may gain a sub-unit freely — the frequent case, an operator deciding to start
// selling an existing product in pieces — because every stored row holds SubQuantity = 0 and
// zero sixths, zero twelfths and zero thousandths are the same zero, so nothing is rewritten.
// Afterwards the divisor may only move to a multiple: 6 -> 12 splits each sixth into two
// twelfths exactly, while 12 -> 6 would turn 9/12 into 4.5 sixths and round away real stock.
func validateProductSubUnitDivisors(productos []types.Product, storedByID map[int32]types.Product) error {
	for i := range productos {
		if err := validateSubUnitDivisorRange(&productos[i]); err != nil {
			return err
		}
	}

	for i := range productos {
		product := &productos[i]
		stored, isUpdate := storedByID[product.ID]
		previousDivisor := stored.SbuQuantity
		// A divisor that was never set leaves every stored row at SubQuantity = 0, so any
		// valid divisor is reachable without changing what a single number means.
		if !isUpdate || previousDivisor <= core.QuantityDivisorNone {
			continue
		}
		if product.SbuQuantity == previousDivisor {
			continue
		}
		if product.SbuQuantity <= core.QuantityDivisorNone {
			return core.Err(fmt.Sprintf(
				`Producto "%s": no se puede quitar la sub-unidad (divisor %v). El stock y los movimientos ya guardados quedarían sin interpretación.`,
				product.Name, previousDivisor))
		}
		if !core.IsQuantityDivisorRefinement(previousDivisor, product.SbuQuantity) {
			return core.Err(fmt.Sprintf(
				`Producto "%s": el divisor de sub-unidad sólo puede refinarse a un múltiplo de %v. Se recibió %v.`,
				product.Name, previousDivisor, product.SbuQuantity))
		}
	}
	return nil
}

func validateSubUnitDivisorRange(product *types.Product) error {
	if product.SbuQuantity == 0 {
		return nil
	}
	if err := core.ValidateQuantityDivisor(product.SbuQuantity); err != nil {
		return core.Err(fmt.Sprintf(`Producto "%s":`, product.Name), err)
	}
	// A sub-unit the operator cannot name is not sellable: the cart, the stock page and the
	// invoice line all label it with SbuUnit.
	if product.SbuQuantity > core.QuantityDivisorNone && len(strings.TrimSpace(product.SbuUnit)) == 0 {
		return core.Err(fmt.Sprintf(
			`Producto "%s": se definió una sub-unidad (divisor %v) sin nombre de sub-unidad.`,
			product.Name, product.SbuQuantity))
	}
	return nil
}

func getProductBrandNames(companyID int32, productos []types.Product) (map[int32]string, error) {
	brandIDs := core.SliceSet[int32]{}
	for _, product := range productos {
		brandIDs.AddIf(product.BrandID)
	}
	if brandIDs.IsEmpty() {
		return map[int32]string{}, nil
	}

	brands := []business.SharedListRecord{}
	query := db.Query(&brands)
	query.Select(query.ID, query.Name).
		CompanyID.Equals(companyID).ID.In(brandIDs.Values...)

	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al obtener las marcas de productos: %w", err)
	}

	brandNamesByID := make(map[int32]string, len(brands))
	for _, brand := range brands {
		brandNamesByID[brand.ID] = brand.Name
	}
	return brandNamesByID, nil
}

type productoImage struct {
	Content       string
	Content_x6    string
	Content_x4    string
	Content_x2    string
	Folder        string
	Description   string
	ProductID     int32
	ImageID       int32 // client-reserved imageID (from GET.image-id-counter)
	ImageToDelete int32 // imageID to remove (autoincrement*10 + configDigit)
}

func PostProductImage(req *core.HandlerArgs) core.HandlerResponse {
	image := productoImage{}
	err := json.Unmarshal([]byte(*req.Body), &image)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body:", err)
	}

	core.Log("Image to delete 1:", image.ImageToDelete)

	if image.ImageToDelete == 0 {
		// imageID is reserved client-side via GET.image-id-counter before uploading.
		if image.ProductID == 0 || image.ImageID == 0 || (len(image.Content) == 0 && len(image.Content_x6) == 0) {
			return req.MakeErr("NO se encontraron los parámetros: [ProductID] [ImageID] [Content]")
		}
	}

	productos := []types.Product{}
	query := db.Query(&productos)
	query.Select().
		CompanyID.Equals(req.User.CompanyID).
		ID.Equals(image.ProductID)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el product:", err)
	}
	if len(productos) == 0 {
		return req.MakeErr("No se encontró el product con ID:", image.ProductID)
	}

	response := map[string]string{}
	if image.ImageToDelete == 0 {
		// imageID is reserved client-side (GET.image-id-counter); derive the base CDN name from it.
		baseName := fmt.Sprintf("%v_%v", req.User.CompanyID, image.ImageID)

		imageArgs := cloud.ImageArgs{
			Content: image.Content, Folder: "img-productos", Name: baseName, Type: "avif",
			// Empty label = base resolution (bare filename); x4/x2 get the "-x4"/"-x2" suffix.
			Resolutions: map[uint16]string{980: "", 570: "x4", 360: "x2"},
		}

		if len(image.Content_x6) > 0 {
			resolutionMap := map[int8]*string{
				6: &image.Content_x6, 4: &image.Content_x4, 2: &image.Content_x2,
			}
			for resolution, content := range resolutionMap {
				if len(*content) < 50 {
					continue
				}
				cloned := imageArgs
				cloned.Resolution = resolution
				cloned.Content = *content
				if _, err = cloud.SaveImage(cloned); err != nil {
					return req.MakeErr("Error al guardar la imagen:", err)
				}
			}
		} else {
			if _, err = cloud.SaveConvertImage(imageArgs); err != nil {
				return req.MakeErr("Error al guardar la imagen:", err)
			}
		}
		response["imageName"] = "img-productos/" + baseName
	}

	// The image list is changed on the locked row and only its columns are written: a product
	// form or a sale committing meanwhile keeps its own columns, and two uploads to one product
	// both stay listed.
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		lockedProducts := []types.Product{{CompanyID: req.User.CompanyID, ID: image.ProductID}}
		if err := db.LockByKeys(transaction, &lockedProducts); err != nil {
			return core.Err("Error al obtener el producto:", err)
		}
		if len(lockedProducts) == 0 {
			return core.Err("No se encontró el producto con ID:", image.ProductID)
		}
		product := lockedProducts[0]
		// Skip the write when only the image bytes were (re)uploaded with no association change.
		if !changeProductImages(&product, image) {
			return nil
		}
		product.Updated = core.SUnixTime()
		product.UpdatedBy = req.User.ID
		table := db.TableOf[types.Product]()
		return db.UpdateIn(transaction, &[]types.Product{product},
			table.ImageIDs, table.ImageDescriptions, table.ImageMain, table.Updated, table.UpdatedBy)
	})
	if err != nil {
		return req.MakeErr("Error al actualizar el producto:", err)
	}
	return req.MakeResponse(response)
}

// changeProductImages applies an upload or a removal to the product's image list, and reports
// whether the list changed.
func changeProductImages(product *types.Product, image productoImage) bool {
	if image.ImageToDelete > 0 {
		// Drop the imageID from the parallel ImageIDs/ImageDescriptions arrays.
		imageIDs := []int32{}
		imageDescriptions := []string{}
		for index, id := range product.ImageIDs {
			if id == image.ImageToDelete {
				continue
			}
			imageIDs = append(imageIDs, id)
			if index < len(product.ImageDescriptions) {
				imageDescriptions = append(imageDescriptions, product.ImageDescriptions[index])
			}
		}
		product.ImageIDs = imageIDs
		product.ImageDescriptions = imageDescriptions
		// Repoint the main image if the deleted one was primary.
		if product.ImageMain == image.ImageToDelete {
			product.ImageMain = 0
			if len(imageIDs) > 0 {
				product.ImageMain = imageIDs[0]
			}
		}
		return true
	}
	// Dedupe guard: the product may already reference this imageID (saved optimistically
	// with the client-reserved ID before the bytes finished uploading). Only associate it
	// when absent, so a retried upload can't double-append.
	if slices.Contains(product.ImageIDs, image.ImageID) {
		return false
	}
	// Prepend the new image; it becomes the main image.
	product.ImageIDs = append([]int32{image.ImageID}, product.ImageIDs...)
	product.ImageDescriptions = append([]string{image.Description}, product.ImageDescriptions...)
	product.ImageMain = image.ImageID
	return true
}

func PostProductCategoryImage(req *core.HandlerArgs) core.HandlerResponse {
	image := productoImage{}
	err := json.Unmarshal([]byte(*req.Body), &image)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if image.ImageID <= 0 || image.ImageID%10 != business.ImageConfigDigitFull {
		return req.MakeErr("El ID reservado no corresponde a una imagen completa.")
	}
	if len(image.Content_x6) < 50 || len(image.Content_x4) < 50 || len(image.Content_x2) < 50 {
		return req.MakeErr("La imagen debe incluir las resoluciones x6, x4 y x2.")
	}

	imageName := fmt.Sprintf("%v_%v", req.User.CompanyID, image.ImageID)
	for resolution, content := range map[int8]string{
		6: image.Content_x6, 4: image.Content_x4, 2: image.Content_x2,
	} {
		if _, err := cloud.SaveImage(cloud.ImageArgs{
			Content: content, Folder: "img-public", Name: imageName, Type: "avif", Resolution: resolution,
		}); err != nil {
			return req.MakeErr("Error al guardar la imagen:", err)
		}
	}

	core.Log("PostProductCategoryImage:", "company_id=", req.User.CompanyID, "image_id=", image.ImageID)
	response := map[string]string{
		"imageName": "img-public/" + imageName,
	}
	return req.MakeResponse(response)
}
