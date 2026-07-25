package business

import (
	"app/core"
	"app/db"
	"fmt"
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
