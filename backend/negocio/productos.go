package negocio

import (
	"app/cloud"
	"app/core"
	"app/db"
	negocioTypes "app/negocio/types"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

func GetProductos(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt("updated")

	productos := []negocioTypes.Product{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&productos).EmpresaID.Equals(req.Usuario.EmpresaID)

		query.Exclude(query.Stock, query.StockStatus, query.EmpresaID, query.Created, query.CreatedBy, query.NombreHash)
		
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

	productos := []negocioTypes.Product{}
	err := db.QueryCachedIDs(&productos, cachedIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los productos.", err)
	}

	return core.MakeResponse(req, &productos)
}

func PostProductos(req *core.HandlerArgs) core.HandlerResponse {

	productos := []negocioTypes.Product{}
	if err := json.Unmarshal([]byte(*req.Body), &productos); err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	nameHashToName := make(map[int32]*negocioTypes.Product, len(productos))
	// SelfParse each producto to populate NombreHash and fail fast on duplicate names in this payload.
	for i := range productos {
		e := &productos[i]
		if len(e.Nombre) < 4 {
			return req.MakeErr("Faltan propiedades de en el producto.")
		}
		// Preserve incoming ID so frontend can map TempID -> ID after merge/upsert.
		e.TempID = e.ID
		e.EmpresaID = req.Usuario.EmpresaID
		e.SelfParse()
		if previousProduct, duplicate := nameHashToName[e.NombreHash]; duplicate {
			return req.MakeErr(fmt.Sprintf("Hay nombres duplicados en la solicitud: %s y %s", previousProduct.Nombre, e.Nombre))
		}
		nameHashToName[e.NombreHash] = e
	}

	// Group existing records by NombreHash so we can check active collisions and reuse inactive IDs.
	existingProductsByHash := make(map[int32][]negocioTypes.Product, len(nameHashToName))
	nombreHashesToValidate := make([]int32, 0, len(nameHashToName))
	for nombreHash := range nameHashToName {
		nombreHashesToValidate = append(nombreHashesToValidate, nombreHash)
	}

	existingProducts := []negocioTypes.Product{}
	query := db.Query(&existingProducts)
	query.Select(query.NombreHash, query.ID, query.Status).
		EmpresaID.Equals(req.Usuario.EmpresaID).
		NombreHash.In(nombreHashesToValidate...).AllowFilter()

	if err := query.Exec(); err != nil {
		return req.MakeErr(fmt.Sprintf("Error al validar los nombres de productos: %v", err))
	}

	for _, existingProduct := range existingProducts {
		existingProductsByHash[existingProduct.NombreHash] = append(
			existingProductsByHash[existingProduct.NombreHash], existingProduct)
	}

	// Enforce name uniqueness across the database and reassign inactive IDs when needed.
	for i := range productos {
		currentProduct := &productos[i]
		if existingProducts, found := existingProductsByHash[currentProduct.NombreHash]; found {
			for _, candidate := range existingProducts {
				if candidate.Status > 0 && candidate.ID != currentProduct.ID {
					return req.MakeErr(fmt.Sprintf(`Ya existe un producto activo con el nombre "%s". ID=%v`, currentProduct.Nombre, candidate.ID))
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

	buildPresentaciones := func(current *negocioTypes.Product, incoming *negocioTypes.Product) {
		presentacionesMap := map[int16]negocioTypes.ProductoPesentacion{}
		presentacionesNameMap := map[string]negocioTypes.ProductoPesentacion{}
		presentacionMaxID := int16(0)

		if current != nil {
			for _, presentacionActual := range current.Presentaciones {
				presentacionesNameMap[core.Concatn(presentacionActual.AtributoID, strings.ToLower(presentacionActual.Name))] = presentacionActual
				if presentacionActual.ID > presentacionMaxID {
					presentacionMaxID = presentacionActual.ID
				}
			}
		}

		for _, presentacionNueva := range incoming.Presentaciones {
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

		incoming.Presentaciones = core.MapToSliceT(presentacionesMap)
	}

	t := negocioTypes.ProductoTable{}
	sunixTime := core.SUnixTime()

	// Merge resolves insert/update per primary key and applies only required writes.
	err := db.Merge(&productos,
		[]db.Coln{t.Stock, t.StockReservado, t.StockStatus, t.CategoriasConStock, t.Created, t.CreatedBy, t.Images},
		func(prev, current *negocioTypes.Product) bool {
			current.EmpresaID = req.Usuario.EmpresaID
			current.Created = prev.Created
			current.CreatedBy = prev.CreatedBy
			current.Stock = prev.Stock
			current.StockReservado = prev.StockReservado
			current.StockStatus = prev.StockStatus
			current.CategoriasConStock = prev.CategoriasConStock
			current.Images = prev.Images
			current.NameUpdated = prev.NameUpdated
			buildPresentaciones(prev, current)

			comparableCurrent := *current
			comparableCurrent.TempID = prev.TempID
			comparableCurrent.Updated = prev.Updated
			comparableCurrent.UpdatedBy = prev.UpdatedBy

			// NameUpdated es para el delta si se han actualizado los nombres
			if current.Nombre != prev.Nombre || current.MarcaID != prev.MarcaID || !slices.Equal(current.CategoriasIDs, prev.CategoriasIDs) {
				current.NameUpdated = sunixTime
			}

			current.Updated = nowTime
			current.UpdatedBy = req.Usuario.ID
			return true
		},
		func(current *negocioTypes.Product) {
			current.EmpresaID = req.Usuario.EmpresaID
			current.Created = nowTime
			current.CreatedBy = req.Usuario.ID
			current.Updated = nowTime
			current.NameUpdated = sunixTime
			current.Status = 1
			buildPresentaciones(nil, current)
		},
	)
	if err != nil {
		return req.MakeErr("Error al actualizar / insertar la sede: " + err.Error())
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
	ProductoID    int32
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
		if image.ProductoID == 0 || (len(image.Content) == 0 && len(image.Content_x6) == 0) {
			return req.MakeErr("NO se encontraron los parámetros: [ProductoID] [Content]")
		}
	}

	productos := []negocioTypes.Product{}
	query := db.Query(&productos)
	query.Select().
		EmpresaID.Equals(req.Usuario.EmpresaID).
		ID.Equals(image.ProductoID)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el producto:", err)
	}
	if len(productos) == 0 {
		return req.MakeErr("No se encontró el producto con ID:", image.ProductoID)
	}

	producto := productos[0]
	response := map[string]string{}

	name := core.ToBase36(time.Now().UnixMilli())

	imageArgs := cloud.ImageArgs{
		Content: image.Content, Folder: "img-productos", Name: name, Type: "avif",
		Resolutions: map[uint16]string{980: "x6", 570: "x4", 360: "x2"},
	}

	addImage := func() {
		response["imageName"] = "img-productos/" + name

		pi := negocioTypes.ProductoImagen{Name: name, Descripcion: image.Description}
		producto.Images = append([]negocioTypes.ProductoImagen{pi}, producto.Images...)
	}

	if len(image.ImageToDelete) > 0 {
		images := []negocioTypes.ProductoImagen{}
		for _, e := range producto.Images {
			if e.Name != image.ImageToDelete {
				images = append(images, e)
			}
		}
		producto.Images = images
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

	producto.Updated = core.SUnixTime()
	producto.UpdatedBy = req.Usuario.ID

	core.Print(producto)

	err = db.Insert(&[]negocioTypes.Product{producto})

	if err != nil {
		return req.MakeErr("Error al actualizar el producto:", err)
	}

	return req.MakeResponse(response)
}

func GetProductosCMS(req *core.HandlerArgs) core.HandlerResponse {
	empresaID := req.GetQueryInt("empresa-id")
	// core.Log("usuario::")
	// core.Print(req.Usuario)
	if empresaID == 0 && req.Usuario != nil {
		empresaID = req.Usuario.EmpresaID
	} else if empresaID == 0 {
		empresaID = 1
	}

	categoriaID := req.GetQueryInt("categoria-id")

	productos := []negocioTypes.Product{}
	categorias := []negocioTypes.ListaCompartidaRegistro{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&productos)
		q1 := db.Table[negocioTypes.Product]()
		query.Select(q1.ID, q1.Nombre, q1.Descripcion, q1.Precio, q1.Descuento, q1.PrecioFinal, q1.Images, q1.Stock, q1.CategoriasIDs).
			EmpresaID.Equals(empresaID).
			StockStatus.Equals(1)
		if categoriaID > 0 {
			query.CategoriasIDs.Contains(categoriaID)
		}
		err := query.Exec()
		if err != nil {
			err = fmt.Errorf("error al obtener los productos (store): %v", err)
		}
		return err
	})

	errGroup.Go(func() error {
		query := db.Query(&categorias)
		q1 := db.Table[negocioTypes.ListaCompartidaRegistro]()
		query.Select(q1.ID, q1.Nombre, q1.Descripcion).
			EmpresaID.Equals(empresaID).
			ListaID.Equals(1).
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

	imageName := core.Concat("-", core.ToBase36s(req.Usuario.EmpresaID), image.Order, core.ToBase36(0))
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
