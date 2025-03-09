package handlers

import (
	"app/aws"
	"app/core"
	"app/db"
	s "app/types"
	"encoding/json"
	"time"
)

type GaleriaImage struct {
	Content       string
	Folder        string
	Description   string
	ImageToDelete string
}

func PostGaleriaImage(req *core.HandlerArgs) core.HandlerResponse {
	image := productoImage{}
	err := json.Unmarshal([]byte(*req.Body), &image)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body:", err)
	}

	imageName := core.ToBase36(time.Now().UnixMilli())

	// Convierte y guarda la imagen en S3
	imageArgs := aws.ImageArgs{
		Content:     image.Content,
		Folder:      "img-galeria",
		Name:        imageName,
		Resolutions: map[uint16]string{1400: "x8", 540: "x4", 340: "x2"},
	}

	core.Env.IS_LOCAL = false
	_, err = aws.SaveImage(imageArgs)
	if err != nil {
		return req.MakeErr("Error al guardar la imagen: " + err.Error())
	}
	core.Env.IS_LOCAL = true

	// Guarda la imagen en BD
	galeriaImage := s.GaleriaImagen{
		EmpresaID:   req.Usuario.EmpresaID,
		Image:       imageName,
		Description: image.Description,
		Status:      1,
		Updated:     core.SunixTime(),
	}

	err = db.Insert(&[]s.GaleriaImagen{galeriaImage})
	if err != nil {
		core.Log("Error al guardar la imagen.")
		return req.MakeErr("Error al guardar la imagen en BD.", err)
	}

	return req.MakeResponse(map[string]any{"imageName": imageName})
}

func GetGaleriaImages(req *core.HandlerArgs) core.HandlerResponse {

	updated := core.UnixToSunix(req.GetQueryInt64("updated"))

	imagenes, err := db.SelectT(func(query *db.Query[s.GaleriaImagen]) {
		col := query.T
		query.Exclude(col.EmpresaID_())
		query.Where(col.EmpresaID_().Equals(req.Usuario.EmpresaID))
		if updated > 0 {
			query.Where(col.Updated_().GreaterEqual(updated))
		} else {
			query.Where(col.Status_().GreaterEqual(1))
		}
	})

	if err != nil {
		return req.MakeErr("Error al obtener las imágenes:", err)
	}

	return req.MakeResponse(imagenes)
}
