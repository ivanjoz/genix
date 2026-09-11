package production

import (
	business "app/business/types"
	"app/cloud"
	"app/core"
	"app/db"
	"app/production/types"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/sync/errgroup"
)

func GetProducts(req *core.HandlerArgs) core.HandlerResponse {
	// Delta syncs are watermarked by "upv", the write sequence number, not by a timestamp: two
	// writes in the same second are distinguishable, so nothing is re-sent and nothing is skipped.
	updatedSince := req.GetUpVersion()

	productos := []types.Product{}
	// The catalog is Status=1 and nothing else. A deleted row (0) and a supply/material (2) live
	// in the same table, so a delta still has to mention them — but as ids to drop, never as
	// record bodies: a supply carries a purchase Price and no FinalPrice, and every screen that
	// reads this cache treats what it finds there as sellable.
	evictedStatuses := []int8{types.ProductStatusInactive, types.ProductStatusSupply}
	evictedIDsByStatus := make([][]int32, len(evictedStatuses))
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&productos).CompanyID.Equals(req.User.CompanyID)

		query.Exclude(query.Stock, query.StockStatus, query.CompanyID, query.Created, query.CreatedBy, query.NameHash)

		// Pinning Status leaves the delta index's every key bound, so Delta() only adds the
		// watermark and the read stays inside the active bucket on a first sync and a delta alike.
		query.Status.Equals(types.ProductStatusActive)
		query.Delta(updatedSince)

		if err := query.Exec(); err != nil {
			return fmt.Errorf("error al obtener los productos: %v", err)
		}
		return nil
	})

	// A first sync has no cached rows to evict, so the eviction scans are pure delta work.
	if updatedSince > 0 {
		for statusIndex, evictedStatus := range evictedStatuses {
			errGroup.Go(func() error {
				query := db.Query(&[]types.Product{}).CompanyID.Equals(req.User.CompanyID)
				query.Select(query.ID)
				query.Status.Equals(evictedStatus)
				query.Delta(updatedSince)

				// Only the ids matter, so every decoded row is discarded as it is scanned.
				return query.ExecScan(func(record *types.Product) bool {
					evictedIDsByStatus[statusIndex] = append(evictedIDsByStatus[statusIndex], record.ID)
					return true
				})
			})
		}
	}

	if err := errGroup.Wait(); err != nil {
		return req.MakeErr(err)
	}

	evictedProductIDs := []int32{}
	for _, evictedIDs := range evictedIDsByStatus {
		evictedProductIDs = append(evictedProductIDs, evictedIDs...)
	}

	// The watermark the client sent, next to what it bought: upv=0 means the client asked for a
	// first sync, so a 10k-row answer is the request being obeyed, not the delta failing.
	core.Log("GET.products:: upv recibido::", updatedSince, "| registros devueltos::", len(productos),
		"| ids a evictar::", len(evictedProductIDs))

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

	// Active products live in status group 1. Return only ids + weights (no
	// record bodies) — the client resolves names from its by-id cache.
	matches, err := db.SearchTextIDs[types.Product](companyID, query, 1, limit)
	if err != nil {
		return req.MakeErr("Error en la búsqueda de texto de productos:", err)
	}

	return core.MakeResponse(req, &matches)
}

func GetProductsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDs := req.ExtractUpdatedVersionValues()

	if len(cachedIDs) == 0 {
		return req.MakeErr("No se enviaron ids a buscar.")
	}

	productos := []types.Product{}
	err := db.QueryCachedIDs(&productos, cachedIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los productos.", err)
	}

	return core.MakeResponse(req, &productos)
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
		productos[i].BrandName_ = brandNamesByID[productos[i].BrandID]
	}

	// Group existing records by NameHash so we can check active collisions and reuse inactive IDs.
	existingProductsByHash := make(map[int32][]types.Product, len(nameHashToName))
	nameHashesToValidate := make([]int32, 0, len(nameHashToName))
	for nameHash := range nameHashToName {
		nameHashesToValidate = append(nameHashesToValidate, nameHash)
	}

	existingProducts := []types.Product{}
	query := db.Query(&existingProducts)
	query.Select(query.NameHash, query.ID, query.Status).
		CompanyID.Equals(req.User.CompanyID).
		NameHash.In(nameHashesToValidate...)

	if err := query.Exec(); err != nil {
		return req.MakeErr(fmt.Sprintf("Error al validar los nombres de productos: %v", err))
	}

	for _, existingProduct := range existingProducts {
		existingProductsByHash[existingProduct.NameHash] = append(
			existingProductsByHash[existingProduct.NameHash], existingProduct)
	}

	// Enforce name uniqueness across the database and reassign inactive IDs when needed.
	for i := range productos {
		currentProduct := &productos[i]
		if existingProducts, found := existingProductsByHash[currentProduct.NameHash]; found {
			for _, candidate := range existingProducts {
				if candidate.Status > 0 && candidate.ID != currentProduct.ID {
					return req.MakeErr(fmt.Sprintf(`Ya existe un product activo con el nombre "%s". ID=%v`, currentProduct.Name, candidate.ID))
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
	}

	// Runs before db.Merge: the merge callback fires during the write, so a divisor rejected
	// there would leave the other rows already persisted.
	if err := validateProductSubUnitDivisors(req.User.CompanyID, productos); err != nil {
		return req.MakeErr(err)
	}

	nowTime := core.SUnixTime()
	core.Log("PostProductos merge payload:", len(productos))

	buildPresentaciones := func(current *types.Product, incoming *types.Product) {
		presentacionesMap := map[int16]types.ProductPresentation{}
		presentacionesNameMap := map[string]types.ProductPresentation{}
		presentacionMaxID := int16(0)

		if current != nil {
			for _, presentacionActual := range current.Presentations {
				presentacionesNameMap[core.Concatn(presentacionActual.AtributoID, strings.ToLower(presentacionActual.Name))] = presentacionActual
				if presentacionActual.ID > presentacionMaxID {
					presentacionMaxID = presentacionActual.ID
				}
			}
		}

		for _, presentacionNueva := range incoming.Presentations {
			presentacionName := core.Concatn(presentacionNueva.AtributoID, strings.ToLower(presentacionNueva.Name))
			if current, ok := presentacionesNameMap[presentacionName]; ok && presentacionNueva.ID != 0 {
				presentacionNueva.ID = current.ID
			}
			if presentacionNueva.ID <= 0 {
				presentacionMaxID++
				presentacionNueva.ID = presentacionMaxID
			}
			presentacionesMap[presentacionNueva.ID] = presentacionNueva
		}

		// Add the removed presentaciones with status = 0
		for _, presentacionActual := range presentacionesNameMap {
			if _, ok := presentacionesMap[presentacionActual.ID]; !ok {
				presentacionActual.Status = 0
				presentacionesMap[presentacionActual.ID] = presentacionActual
			}
		}

		incoming.Presentations = core.MapToSliceT(presentacionesMap)
	}

	t := types.ProductTable{}
	sunixTime := core.SUnixTime()

	// Merge resolves insert/update per primary key and applies only required writes.
	err = db.Merge(&productos,
		db.Cols(t.Stock, t.ReservedStock, t.StockStatus, t.CategoriesWithStock, t.Created, t.CreatedBy, t.ImageMain, t.ImageIDs, t.ImageDescriptions),
		func(prev, current *types.Product) bool {
			current.CompanyID = req.User.CompanyID
			current.Created = prev.Created
			current.CreatedBy = prev.CreatedBy
			current.Stock = prev.Stock
			current.ReservedStock = prev.ReservedStock
			current.StockStatus = prev.StockStatus
			current.CategoriesWithStock = prev.CategoriesWithStock
			current.ImageMain = prev.ImageMain
			current.ImageIDs = prev.ImageIDs
			current.ImageDescriptions = prev.ImageDescriptions
			current.NameUpdated = prev.NameUpdated
			buildPresentaciones(prev, current)

			comparableCurrent := *current
			comparableCurrent.TempID = prev.TempID
			comparableCurrent.Updated = prev.Updated
			comparableCurrent.UpdatedBy = prev.UpdatedBy

			// NameUpdated es para el delta si se han actualizado los nombres
			if current.Name != prev.Name || current.BrandID != prev.BrandID || !slices.Equal(current.CategoryIDs, prev.CategoryIDs) {
				current.NameUpdated = sunixTime
			}

			current.Updated = nowTime
			current.UpdatedBy = req.User.ID
			return true
		},
		func(current *types.Product) {
			current.CompanyID = req.User.CompanyID
			current.Created = nowTime
			current.CreatedBy = req.User.ID
			current.Updated = nowTime
			current.NameUpdated = sunixTime
			current.Status = 1
			buildPresentaciones(nil, current)
		},
	)
	if err != nil {
		return req.MakeErr("Error al actualizar / insertar la site: " + err.Error())
	}

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

// validateProductSubUnitDivisors enforces the refinement-only rule on Product.SbuQuantity,
// which is the divisor every stored SubQuantity for that product is expressed in.
//
// A product may gain a sub-unit freely — the frequent case, an operator deciding to start
// selling an existing product in pieces — because every stored row holds SubQuantity = 0 and
// zero sixths, zero twelfths and zero thousandths are the same zero, so nothing is rewritten.
// Afterwards the divisor may only move to a multiple: 6 -> 12 splits each sixth into two
// twelfths exactly, while 12 -> 6 would turn 9/12 into 4.5 sixths and round away real stock.
func validateProductSubUnitDivisors(companyID int32, productos []types.Product) error {
	productIDsToCheck := core.SliceSet[int32]{}
	for i := range productos {
		if err := validateSubUnitDivisorRange(&productos[i]); err != nil {
			return err
		}
		if productos[i].ID > 0 {
			productIDsToCheck.Add(productos[i].ID)
		}
	}
	if productIDsToCheck.IsEmpty() {
		return nil
	}

	existing := []types.Product{}
	query := db.Query(&existing)
	query.Select(query.ID, query.SbuQuantity).
		CompanyID.Equals(companyID).
		ID.In(productIDsToCheck.Values...)
	if err := query.Exec(); err != nil {
		return core.Err("Error al validar la sub-unidad de los productos:", err)
	}

	previousDivisorByID := make(map[int32]int16, len(existing))
	for _, product := range existing {
		previousDivisorByID[product.ID] = product.SbuQuantity
	}

	for i := range productos {
		product := &productos[i]
		previousDivisor, isUpdate := previousDivisorByID[product.ID]
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

	product := productos[0]
	response := map[string]string{}
	productChanged := false

	if image.ImageToDelete > 0 {
		productChanged = true
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
	} else {
		// imageID is reserved client-side (GET.image-id-counter); derive the base CDN name from it.
		imageID := image.ImageID
		baseName := fmt.Sprintf("%v_%v", req.User.CompanyID, imageID)

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
		// Dedupe guard: the product may already reference this imageID (saved optimistically
		// with the client-reserved ID before the bytes finished uploading). Only associate it
		// when absent, so a retried upload can't double-append.
		if !slices.Contains(product.ImageIDs, imageID) {
			// Prepend the new image; it becomes the main image.
			product.ImageIDs = append([]int32{imageID}, product.ImageIDs...)
			product.ImageDescriptions = append([]string{image.Description}, product.ImageDescriptions...)
			product.ImageMain = imageID
			productChanged = true
		}
	}

	// Skip the product write when only the image bytes were (re)uploaded with no association change.
	if !productChanged {
		return req.MakeResponse(response)
	}

	product.Updated = core.SUnixTime()
	product.UpdatedBy = req.User.ID

	core.Print(product)

	err = db.Insert(&[]types.Product{product})

	if err != nil {
		return req.MakeErr("Error al actualizar el product:", err)
	}

	return req.MakeResponse(response)
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
