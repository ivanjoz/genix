package handlers

import (
	"app/aws"
	"app/core"
	"app/db"
	s "app/types"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

func GetProductos(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt64("upd"), req.GetQueryInt64("updated"))

	productos := []s.Producto{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&productos)

		query.Exclude(query.Stock, query.StockStatus).
			EmpresaID.Equals(req.Usuario.EmpresaID)
		if updated > 0 {
			query.Updated.GreaterThan(updated)
		} else {
			query.Status.GreaterEqual(1)
		}
		err := query.Exec()
		if err != nil {
			err = fmt.Errorf("error al obtener los productos: %v", err)
		}
		return err
	})

	if err := errGroup.Wait(); err != nil {
		return req.MakeErr(err)
	}

	core.Log("productos obtenidos::", len(productos))
	if len(productos) > 0 {
		core.Print(productos[0])
	}

	return core.MakeResponse(req, &productos)
}

func PostProductos(req *core.HandlerArgs) core.HandlerResponse {

	productos := []s.Producto{}
	err := json.Unmarshal([]byte(*req.Body), &productos)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	productosIDsSet := core.SliceSet[int32]{}
	createCounter := 0

	for i := range productos {
		e := &productos[i]
		if e.ID < 1 {
			createCounter++
			e.TempID = e.ID
		} else {
			productosIDsSet.Add(e.ID)
		}
		if len(e.Nombre) < 4 {
			return req.MakeErr("Faltan propiedades de en el producto.")
		}
	}

	var counter int64
	if createCounter > 0 {
		counter, err = core.GetCounter("productos", createCounter, req.Usuario.EmpresaID)
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
	}

	productosCurrent := []s.Producto{}
	if len(productosIDsSet.Values) > 0 {
		query := db.Query(&productosCurrent)
		query.Select().ID.In(productosIDsSet.Values...)
		err = query.Exec()
		if err != nil {
			return req.MakeErr("Error al obtener los productos actuales:", err)
		}
	}

	productosCurrentMap := core.SliceToMapK(productosCurrent,
		func(e s.Producto) int32 { return e.ID })

	nowTime := time.Now().Unix()

	for i := range productos {
		e := &productos[i]
		if e.ID < 1 {
			e.ID = int32(counter)
			e.Created = nowTime
			e.CreatedBy = req.Usuario.ID
			e.Status = 1
			counter++
		} else {
			e.UpdatedBy = req.Usuario.ID
		}
		e.EmpresaID = req.Usuario.EmpresaID
		e.Updated = nowTime

		current := productosCurrentMap[e.ID]

		// Lógica para hacer un merge de las propiedades de los productos
		// optionMaxID := int16(0)
		// propiedadesMap := map[int16]*s.ProductoPropiedades{}
		presentacionesMap := map[int16]s.ProductoPesentacion{}
		presentacionesNameMap := map[string]s.ProductoPesentacion{}
		presentacionMaxID := int16(0)

		if current != nil {
			// Estas propiedades no cambian
			e.Stock = current.Stock
			e.StockReservado = current.StockReservado
			e.StockStatus = current.StockStatus
			e.CategoriasConStock = current.CategoriasConStock
			e.Created = current.Created
			e.CreatedBy = current.CreatedBy
			e.Images = current.Images
			/*
				for i := range current.Propiedades {
					e := &current.Propiedades[i]
					e.Status = 0
					e.OptionsMap = map[string]*s.ProductoPropiedad{}
					propiedadesMap[e.ID] = e
					for _, opt := range e.Options {
						if opt.ID > optionMaxID {
							optionMaxID = opt.ID
						}
						e.OptionsMap[core.NormaliceString(&opt.Nombre)] = &opt
					}
				}
			*/
			for _, e := range current.Presentaciones {
				presentacionesMap[e.ID] = e
				presentacionesNameMap[core.Concatn(e.AtributoID, strings.ToLower(e.Name))] = e
				if e.ID > presentacionMaxID {
					presentacionMaxID = e.ID
				}
			}
		}

		for _, pr := range e.Presentaciones {
			name := core.Concatn(pr.AtributoID, strings.ToLower(pr.Name))
			if current, ok := presentacionesNameMap[name]; ok && pr.ID != 0 {
				pr.ID = current.ID
			}
			if pr.ID <= 0 {
				presentacionMaxID++
				pr.ID = presentacionMaxID
			}
			presentacionesNameMap[name] = pr
			presentacionesMap[pr.ID] = pr
		}

		e.Presentaciones = core.MapToSliceT(presentacionesMap)

		/*
			for _, propiedad := range e.Propiedades {
				if _, ok := propiedadesMap[propiedad.ID]; !ok {
					propiedadesMap[propiedad.ID] = &s.ProductoPropiedades{
						ID:         propiedad.ID,
						Nombre:     propiedad.Nombre,
						OptionsMap: map[string]*s.ProductoPropiedad{},
						Status:     1,
					}
				}
				propiedadCurrent := propiedadesMap[propiedad.ID]
				propiedadCurrent.Nombre = propiedad.Nombre
				propiedadCurrent.Status = propiedad.Status
				nombresUsedSet := core.SliceSet[string]{}

				for i := range propiedad.Options {
					opt := &propiedad.Options[i]
					if len(opt.Nombre) == 0 {
						continue
					}

					nombre := core.NormaliceString(&opt.Nombre)
					nombresUsedSet.Add(nombre)

					if optCurrent, ok := propiedadCurrent.OptionsMap[nombre]; ok {
						optCurrent.Status = opt.Status
					} else {
						opt.ID = int16(optionMaxID) + 1
						optionMaxID++
						opt.Status = 1
						propiedadCurrent.OptionsMap[nombre] = opt
					}
				}

				for nombre, opt := range propiedadCurrent.OptionsMap {
					if !nombresUsedSet.Include(nombre) {
						opt.Status = 0
					}
				}

				propiedadCurrent.Options = core.MapToSlice(propiedadCurrent.OptionsMap)
			}

			e.Propiedades = core.MapToSlice(propiedadesMap)
		*/
	}

	if err = db.Insert(&productos); err != nil {
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

	productos := []s.Producto{}
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

	imageArgs := aws.ImageArgs{
		Content: image.Content, Folder: "img-productos", Name: name, Type: "avif",
		Resolutions: map[uint16]string{980: "x6", 570: "x4", 360: "x2"},
	}

	addImage := func() {
		response["imageName"] = "img-productos/" + name

		pi := s.ProductoImagen{Name: name, Descripcion: image.Description}
		producto.Images = append([]s.ProductoImagen{pi}, producto.Images...)
	}

	if len(image.ImageToDelete) > 0 {
		images := []s.ProductoImagen{}
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
			_, err = aws.SaveImage(cloned)
			if err != nil {
				return req.MakeErr("Error al guardar la imagen:", err)
			}
		}

		addImage()
	} else {
		_, err = aws.SaveConvertImage(imageArgs)
		if err != nil {
			return req.MakeErr("Error al guardar la imagen:", err)
		}

		addImage()
	}

	producto.Updated = time.Now().Unix()
	producto.UpdatedBy = req.Usuario.ID

	core.Print(producto)

	err = db.Insert(&[]s.Producto{producto})

	if err != nil {
		return req.MakeErr("Error al actualizar el producto:", err)
	}

	return req.MakeResponse(response)
}

func GetProductosCMS(req *core.HandlerArgs) core.HandlerResponse {
	empresaID := req.GetQueryInt("empresa-id")
	// core.Log("usuario::")
	// core.Print(req.Usuario)
	if empresaID == 0 {
		empresaID = req.Usuario.EmpresaID
	}

	categoriaID := req.GetQueryInt("categoria-id")

	productos := []s.Producto{}
	categorias := []s.ListaCompartidaRegistro{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&productos)
		q1 := db.Table[s.Producto]()
		query.Select(q1.ID, q1.Nombre, q1.Descripcion, q1.Precio, q1.Descuento, q1.PrecioFinal, q1.Images, q1.Stock, q1.CategoriasIDs).
			EmpresaID.Equals(empresaID).
			StockStatus.Equals(1)
		if categoriaID > 0 {
			query.CategoriasIDs.Contains(categoriaID)
		}
		err := query.Exec()
		if err != nil {
			err = fmt.Errorf("error al obtener los productos: %v", err)
		}
		return err
	})

	errGroup.Go(func() error {
		query := db.Query(&categorias)
		q1 := db.Table[s.ListaCompartidaRegistro]()
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
	image := aws.ImageArgs{}
	err := json.Unmarshal([]byte(*req.Body), &image)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	imageName := core.Concat("-", core.ToBase36s(req.Usuario.EmpresaID), image.Order, core.ToBase36(0))
	image.Name = imageName
	image.Folder = "img-public"
	image.Resolutions = map[uint16]string{980: "x6", 540: "x4", 340: "x2"}

	if _, err = aws.SaveConvertImage(image); err != nil {
		return req.MakeErr("Error al guardar la imagen: " + err.Error())
	}

	response := map[string]string{
		"imageName": image.Folder + "/" + image.Name,
	}
	return req.MakeResponse(response)
}
