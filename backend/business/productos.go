package business

import (
	"app/cloud"
	"app/core"
	"app/db"
	businessTypes "app/business/types"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

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

func PostProductos(req *core.HandlerArgs) core.HandlerResponse {

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
		NameHash.In(nameHashesToValidate...).AllowFilter()

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
	err := db.Merge(&productos,
		[]db.Coln{t.Stock, t.ReservedStock, t.StockStatus, t.CategoriesWithStock, t.Created, t.CreatedBy, t.Images},
		func(prev, current *businessTypes.Product) bool {
			current.CompanyID = req.User.CompanyID
			current.Created = prev.Created
			current.CreatedBy = prev.CreatedBy
			current.Stock = prev.Stock
			current.ReservedStock = prev.ReservedStock
			current.StockStatus = prev.StockStatus
			current.CategoriesWithStock = prev.CategoriesWithStock
			current.Images = prev.Images
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

	return req.MakeResponse(productos)
}

type productoImage struct {
	Content       string
	Content_x6    string
	Content_x4    string
	Content_x2    string
	Folder        string
	Description   string
	ProductID    int32
	ImageToDelete string
}

func PostProductoImage(req *core.HandlerArgs) core.HandlerResponse {
	image := productoImage{}
	err := json.Unmarshal([]byte(*req.Body), &image)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body:", err)
	}

	core.Log("Image to delete 1:", image.ImageToDelete)

	if len(image.ImageToDelete) == 0 {
		if image.ProductID == 0 || (len(image.Content) == 0 && len(image.Content_x6) == 0) {
			return req.MakeErr("NO se encontraron los parámetros: [ProductID] [Content]")
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

	name := core.ToBase36(time.Now().UnixMilli())

	imageArgs := cloud.ImageArgs{
		Content: image.Content, Folder: "img-productos", Name: name, Type: "avif",
		Resolutions: map[uint16]string{980: "x6", 570: "x4", 360: "x2"},
	}

	addImage := func() {
		response["imageName"] = "img-productos/" + name

		pi := businessTypes.ProductImage{Name: name, Description: image.Description}
		product.Images = append([]businessTypes.ProductImage{pi}, product.Images...)
	}

	if len(image.ImageToDelete) > 0 {
		images := []businessTypes.ProductImage{}
		for _, e := range product.Images {
			if e.Name != image.ImageToDelete {
				images = append(images, e)
			}
		}
		product.Images = images
	} else if len(image.Content_x6) > 0 {

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
			_, err = cloud.SaveImage(cloned)
			if err != nil {
				return req.MakeErr("Error al guardar la imagen:", err)
			}
		}

		addImage()
	} else {
		_, err = cloud.SaveConvertImage(imageArgs)
		if err != nil {
			return req.MakeErr("Error al guardar la imagen:", err)
		}

		addImage()
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

func GetProductosCMS(req *core.HandlerArgs) core.HandlerResponse {
	companyID := req.GetQueryInt("company-id")
	// core.Log("user::")
	// core.Print(req.User)
	if companyID == 0 && req.User != nil {
		companyID = req.User.CompanyID
	} else if companyID == 0 {
		companyID = 1
	}

	categoriaID := req.GetQueryInt("categoria-id")

	productos := []businessTypes.Product{}
	categorias := []businessTypes.SharedListRecord{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&productos)
		q1 := db.Table[businessTypes.Product]()
		query.Select(q1.ID, q1.Name, q1.Description, q1.Price, q1.Discount, q1.FinalPrice, q1.Images, q1.Stock, q1.CategoryIDs).
			CompanyID.Equals(companyID).
			StockStatus.Equals(1)
		if categoriaID > 0 {
			query.CategoryIDs.Contains(categoriaID)
		}
		err := query.Exec()
		if err != nil {
			err = fmt.Errorf("error al obtener los productos (store): %v", err)
		}
		return err
	})

	errGroup.Go(func() error {
		query := db.Query(&categorias)
		q1 := db.Table[businessTypes.SharedListRecord]()
		query.Select(q1.ID, q1.Name, q1.Description).
			CompanyID.Equals(companyID).
			ListID.Equals(1).
			Status.Equals(1)
		err := query.Exec()
		if err != nil {
			err = fmt.Errorf("error al obtener las categorías: %v", err)
		}
		return err
	})

	if err := errGroup.Wait(); err != nil {
		return req.MakeErr(err)
	}
	core.Log("productos obtenidos::", len(productos))

	response := map[string]any{
		"productos":  productos,
		"categorias": categorias,
	}

	return core.MakeResponse(req, &response)
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
