package handlers

import (
	"app/core"
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
		query := core.DBSelect(&productos, "temp_id", "empresa_id").
			Where("empresa_id").Equals(req.Usuario.EmpresaID)

		if updated > 0 {
			query = query.Where("updated").GreatEq(updated)
		} else {
			query = query.Where("status").Equals(1)
		}
		err := query.Exec()
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

	records := []s.Producto{}
	err := json.Unmarshal([]byte(*req.Body), &records)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	createCounter := 0

	for _, e := range records {
		if e.ID < 1 {
			createCounter++
			e.TempID = e.ID
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

	nowTime := time.Now().Unix()

	for i := range records {
		e := &records[i]
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
	}

	err = core.DBInsert(&records)
	if err != nil {
		return req.MakeErr("Error al actualizar / insertar la sede: " + err.Error())
	}

	return req.MakeResponse(records)
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

	productos := []s.Producto{}

	err = core.DBSelect(&productos).
		Where("empresa_id").Equals(req.Usuario.EmpresaID).
		Where("id").Equals(image.ProductoID).Exec()

	if err != nil {
		return req.MakeErr("Error al obtener el producto:", err)
	}

	if len(productos) == 0 {
		return req.MakeErr("No se encontró el producto con ID:", image.ProductoID)
	}

	producto := productos[0]
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
