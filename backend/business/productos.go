package business

import (
	businessTypes "app/business/types"
	"app/cloud"
	"app/core"
	"app/db"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/sync/errgroup"
)

func GetProductos(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt("updated")

	productos := []businessTypes.Product{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&productos).CompanyID.Equals(req.User.CompanyID)

		query.Exclude(query.Stock, query.StockStatus, query.CompanyID, query.Created, query.CreatedBy, query.NameHash)

		if updated > 0 {
			query.Updated.GreaterThan(updated)
		} else {
			query.Status.GreaterEqual(1)
		}

		if err := query.Exec(); err != nil {
			return fmt.Errorf("error al obtener los productos: %v", err)
		}
		return nil
	})

	if err := errGroup.Wait(); err != nil {
		return req.MakeErr(err)
	}

	return core.MakeResponse(req, &productos)
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
	matches, err := db.SearchTextIDs[businessTypes.Product](companyID, query, 1, limit)
	if err != nil {
		return req.MakeErr("Error en la búsqueda de texto de productos:", err)
	}

	return core.MakeResponse(req, &matches)
}

func GetProductosByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDs := req.ExtractCacheVersionValues()

	if len(cachedIDs) == 0 {
		return req.MakeErr("No se enviaron ids a buscar.")
	}

	// core.Log("buscando ids::", len(cachedIDs), "|", cachedIDs)

	productos := []businessTypes.Product{}
	err := db.QueryCachedIDs(&productos, cachedIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los productos.", err)
	}

	return core.MakeResponse(req, &productos)
}

func PostProducts(req *core.HandlerArgs) core.HandlerResponse {
	// db.SetDebugLogging(2)
	
	productos := []businessTypes.Product{}
	if err := json.Unmarshal([]byte(*req.Body), &productos); err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	nameHashToName := make(map[int32]*businessTypes.Product, len(productos))
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
	existingProductsByHash := make(map[int32][]businessTypes.Product, len(nameHashToName))
	nameHashesToValidate := make([]int32, 0, len(nameHashToName))
	for nameHash := range nameHashToName {
		nameHashesToValidate = append(nameHashesToValidate, nameHash)
	}

	existingProducts := []businessTypes.Product{}
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

	nowTime := core.SUnixTime()
	core.Log("PostProductos merge payload:", len(productos))

	buildPresentaciones := func(current *businessTypes.Product, incoming *businessTypes.Product) {
		presentacionesMap := map[int16]businessTypes.ProductPresentation{}
		presentacionesNameMap := map[string]businessTypes.ProductPresentation{}
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

	t := businessTypes.ProductTable{}
	sunixTime := core.SUnixTime()

	// Merge resolves insert/update per primary key and applies only required writes.
	err = db.Merge(&productos,
		[]db.Coln{t.Stock, t.ReservedStock, t.StockStatus, t.CategoriesWithStock, t.Created, t.CreatedBy, t.ImageMain, t.ImageIDs, t.ImageDescriptions},
		func(prev, current *businessTypes.Product) bool {
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
		func(current *businessTypes.Product) {
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
		if cacheErr := core.SaveCacheGlobal(cacheGroupProductos, req.User.CompanyID, nil, nowTime); cacheErr != nil {
			core.Log("PostProducts:: error registrando cambio de productos para ecommerce", cacheErr)
		}
	}

	return req.MakeResponse(productos)
}

func getProductBrandNames(companyID int32, productos []businessTypes.Product) (map[int32]string, error) {
	brandIDs := core.SliceSet[int32]{}
	for _, product := range productos {
		brandIDs.AddIf(product.BrandID)
	}
	if brandIDs.IsEmpty() {
		return map[int32]string{}, nil
	}

	brands := []businessTypes.SharedListRecord{}
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

// imageConfigDigitFull is the last digit of the imageID for the standard product
// upload (base x6 + x4 + x2). The digit→resolution-set dictionary is defined later.
const imageConfigDigitFull = 7

func PostProductoImage(req *core.HandlerArgs) core.HandlerResponse {
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

	productos := []businessTypes.Product{}
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

	err = db.Insert(&[]businessTypes.Product{product})

	if err != nil {
		return req.MakeErr("Error al actualizar el product:", err)
	}

	return req.MakeResponse(response)
}

func PostProductoCategoriaImage(req *core.HandlerArgs) core.HandlerResponse {
	image := cloud.ImageArgs{}
	err := json.Unmarshal([]byte(*req.Body), &image)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	imageName := core.Concat("-", core.ToBase36s(req.User.CompanyID), image.Order, core.ToBase36(0))
	image.Name = imageName
	image.Folder = "img-public"
	image.Resolutions = map[uint16]string{980: "x6", 540: "x4", 340: "x2"}

	if _, err = cloud.SaveConvertImage(image); err != nil {
		return req.MakeErr("Error al guardar la imagen: " + err.Error())
	}

	response := map[string]string{
		"imageName": image.Folder + "/" + image.Name,
	}
	return req.MakeResponse(response)
}
