package handlers

import (
	"app/core"
	"app/db"
	s "app/types"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

func GetProductos(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")

	productos := []s.Producto{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		err := db.SelectRef(&productos, func(q *db.Query[s.Producto], col s.Producto) {
			q.Where(col.EmpresaID_().Equals(req.Usuario.EmpresaID))
			if updated > 0 {
				q.Where(col.Updated_().GreaterEqual(updated))
			} else {
				q.Where(col.Status_().GreaterEqual(1))
			}
		})
		if err != nil {
			err = fmt.Errorf("error al obtener los productos: %v", err)
		}
		return err
	})

	err := errGroup.Wait()
	if err != nil {
		return req.MakeErr(err)
	}

	core.Log("productos obtenidos::", len(productos))

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
		err = db.SelectRef(&productosCurrent, func(q *db.Query[s.Producto], col s.Producto) {
			q.Where(col.ID_().In(productosIDsSet.Values...))
		})
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
		optionMaxID := int16(0)
		propiedadesMap := map[int16]*s.ProductoPropiedades{}

		if current != nil {
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
		}

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
	}

	if err = db.Insert(&productos); err != nil {
		return req.MakeErr("Error al actualizar / insertar la sede: " + err.Error())
	}

	return req.MakeResponse(productos)
}

type productoImage struct {
	Content       string
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
		if image.ProductoID == 0 || len(image.Content) == 0 {
			return req.MakeErr("NO se encontraron los parámetros: [ProductoID] [Content]")
		}
	}

	productos := db.Select(func(q *db.Query[s.Producto], col s.Producto) {
		q.Where(col.EmpresaID_().Equals(req.Usuario.EmpresaID))
		q.Where(col.ID_().Equals(image.ProductoID))
	})

	if productos.Err != nil {
		return req.MakeErr("Error al obtener el producto:", productos.Err)
	}
	if len(productos.Records) == 0 {
		return req.MakeErr("No se encontró el producto con ID:", image.ProductoID)
	}

	producto := productos.Records[0]
	response := map[string]string{}

	if len(image.ImageToDelete) > 0 {
		images := []s.ProductoImagen{}
		for _, e := range producto.Images {
			if e.Name != image.ImageToDelete {
				images = append(images, e)
			}
		}
		producto.Images = images
	} else {
		imageArgs := ImageArgs{
			Content: image.Content,
			Folder:  "img-productos",
			Name:    fmt.Sprintf("%v", time.Now().UnixMilli()),
		}

		_, err = saveImage(imageArgs)
		if err != nil {
			return req.MakeErr("Error al guardar la imagen: " + err.Error())
		}

		response["imageName"] = image.Folder + "/" + imageArgs.Name

		producto.Images = append(producto.Images, s.ProductoImagen{
			Name: imageArgs.Name, Descripcion: image.Description})
	}

	producto.Updated = time.Now().Unix()
	producto.UpdatedBy = req.Usuario.ID

	err = core.DBInsert(&[]s.Producto{producto})

	if err != nil {
		return req.MakeErr("Error al actualizar el producto:", err)
	}

	return req.MakeResponse(response)
}
