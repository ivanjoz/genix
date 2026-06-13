package business

import (
	"app/cloud"
	"app/core"
	"app/db"
	"encoding/json"
	"fmt"
	"time"
)

// GetImageIdCounter reserves ONE per-company image autoincrement and returns the derived
// imageID + base CDN name. Universal: any uploader prefetches its final ID before converting,
// so the client can show the image as "saved" immediately and upload it in the background.
func GetImageIdCounter(req *core.HandlerArgs) core.HandlerResponse {
	// configDigit selects the resolution-set scheme (default 7 = full product upload).
	configDigit := int32(req.GetQueryInt64("config"))
	if configDigit == 0 {
		configDigit = imageConfigDigitFull
	}

	autoincrement, err := db.GetAutoincrementID(fmt.Sprintf("images_%v", req.User.CompanyID), 1)
	if err != nil {
		return req.MakeErr("Error al reservar el id de imagen:", err)
	}

	imageID := int32(autoincrement*10) + configDigit
	return req.MakeResponse(map[string]any{
		"id":   imageID,
		"name": fmt.Sprintf("%v_%v", req.User.CompanyID, imageID), // base CDN name (no folder/ext)
	})
}

func PostImage(req *core.HandlerArgs) core.HandlerResponse {
	image := cloud.ImageArgs{}
	err := json.Unmarshal([]byte(*req.Body), &image)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	image.Name = fmt.Sprintf("%v", time.Now().UnixMilli())
	image.Folder = "img-productos"
	image.Resolutions = map[uint16]string{980: "x6", 540: "x4", 340: "x2"}

	_, err = cloud.SaveConvertImage(image)
	if err != nil {
		return req.MakeErr("Error al guardar la imagen: " + err.Error())
	}

	response := map[string]string{
		"imageName": image.Folder + "/" + image.Name,
	}
	return req.MakeResponse(response)
}
