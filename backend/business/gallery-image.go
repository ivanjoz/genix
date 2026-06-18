package business

import (
	businessTypes "app/business/types"
	"app/cloud"
	"app/core"
	"app/db"
	"encoding/json"
	"fmt"
)

const galleryImageConfigDigit = 4

type galleryImagePayload struct {
	Content_x8    string
	Content_x4    string
	Description   string
	ImageID       int32
	ImageToDelete int32
}

func PostGalleryImage(req *core.HandlerArgs) core.HandlerResponse {
	payload := galleryImagePayload{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar la imagen:", err)
	}

	if payload.ImageToDelete > 0 {
		return deleteGalleryImage(req, payload.ImageToDelete)
	}

	if len(payload.Content_x8) < 50 || len(payload.Content_x4) < 50 {
		return req.MakeErr("La imagen debe incluir las resoluciones x8 y x4.")
	}

	if payload.ImageID <= 0 || payload.ImageID%10 != galleryImageConfigDigit {
		return req.MakeErr("El ID reservado debe corresponder a la configuración x8/x4.")
	}
	imageID := payload.ImageID
	imageName := fmt.Sprintf("%v_%v", req.User.CompanyID, imageID)

	// The final digit 4 identifies gallery images with exactly x8 and x4 variants.
	for resolution, content := range map[int8]string{8: payload.Content_x8, 4: payload.Content_x4} {
		if _, err := cloud.SaveImage(cloud.ImageArgs{
			Content: content, Folder: "img-galeria", Name: imageName, Type: "avif", Resolution: resolution,
		}); err != nil {
			return req.MakeErr("Error al guardar la imagen:", err)
		}
	}

	galleryImage := businessTypes.GalleryImage{
		CompanyID:   req.User.CompanyID,
		ImageID:     imageID,
		Image:       imageName,
		Description: payload.Description,
		Status:      1,
		Updated:     core.SUnixTime(),
	}
	if err := db.Insert(&[]businessTypes.GalleryImage{galleryImage}); err != nil {
		return req.MakeErr("Error al guardar la imagen en BD.", err)
	}

	core.Log("PostGalleryImage:", "company_id=", req.User.CompanyID, "image_id=", imageID)
	return req.MakeResponse(map[string]any{
		"id": imageID, "imageName": "img-galeria/" + imageName, "description": payload.Description,
	})
}

func deleteGalleryImage(req *core.HandlerArgs, imageID int32) core.HandlerResponse {
	if imageID%10 != galleryImageConfigDigit {
		return req.MakeErr("El ID no corresponde a una imagen de galería.")
	}

	images := []businessTypes.GalleryImage{}
	query := db.Query(&images)
	query.CompanyID.Equals(req.User.CompanyID).
		Image.Equals(fmt.Sprintf("%v_%v", req.User.CompanyID, imageID))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al buscar la imagen:", err)
	}
	if len(images) == 0 {
		return req.MakeErr("No se encontró la imagen de galería.")
	}

	// Soft deletion keeps the delta cache able to evict the image on the next sync.
	images[0].Status = 0
	images[0].Updated = core.SUnixTime()
	if err := db.Insert(&images); err != nil {
		return req.MakeErr("Error al eliminar la imagen:", err)
	}

	core.Log("DeleteGalleryImage:", "company_id=", req.User.CompanyID, "image_id=", imageID)
	return req.MakeResponse(images[0])
}

func GetGalleryImages(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))
	images := []businessTypes.GalleryImage{}
	query := db.Query(&images)
	table := db.Table[businessTypes.GalleryImage]()

	query.Exclude(table.CompanyID).
		CompanyID.Equals(req.User.CompanyID)

	if updated > 0 {
		query.Updated.GreaterThan(updated)
	} else {
		query.Status.GreaterEqual(1)
	}

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las imágenes:", err)
	}

	core.Log("GetGalleryImages:", "company_id=", req.User.CompanyID, "updated=", updated, "count=", len(images))
	return req.MakeResponse(images)
}
