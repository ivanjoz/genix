package handlers

import (
	"app/aws"
	"app/core"
	"encoding/json"
	"fmt"
	"time"
)

func PostImage(req *core.HandlerArgs) core.HandlerResponse {
	image := aws.ImageArgs{}
	err := json.Unmarshal([]byte(*req.Body), &image)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	image.Name = fmt.Sprintf("%v", time.Now().UnixMilli())
	image.Folder = "img-productos"
	image.Resolutions = map[uint16]string{980: "x6", 540: "x4", 340: "x2"}

	_, err = aws.SaveConvertImage(image)
	if err != nil {
		return req.MakeErr("Error al guardar la imagen: " + err.Error())
	}

	response := map[string]string{
		"imageName": image.Folder + "/" + image.Name,
	}
	return req.MakeResponse(response)
}
