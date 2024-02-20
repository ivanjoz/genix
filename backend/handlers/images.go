package handlers

import (
	"app/aws"
	"app/core"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"app/imageconv"
)

type imageBody struct {
	Content string `json:"content"` /* Base64 webp image */
	Folder  string `json:"filter"`
	Name    string `json:"name"`
}

func PostImage(req *core.HandlerArgs) core.HandlerResponse {
	core.Env.LOGS_FULL = true
	fmt.Println("hola!")

	resolutionsMap := map[uint16]string{
		900: "x6", 500: "x4", 340: "x2",
	}

	resolutions := []uint16{}
	for r := range resolutionsMap {
		resolutions = append(resolutions, r)
	}

	image := imageBody{}
	err := json.Unmarshal([]byte(*req.Body), &image)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(image.Content) < 40 {
		return req.MakeErr("No se ha recibido el contenido de la imagen")
	}

	if strings.Contains(image.Content[0:40], "base64,") {
		image.Content = strings.Split(image.Content, "base64,")[1]
	}

	bytes := core.Base64ToBytes(image.Content)

	if len(bytes) == 0 {
		return req.MakeErr("Error al convertir el contenido de la imagen a bytes")
	}

	imageName := fmt.Sprintf("%v", time.Now().UnixMilli())

	images, err := imageconv.Convert(imageconv.ImageConvertInput{
		Image:              bytes,
		UseWebp:            true,
		UseAvif:            true,
		Resolutions:        resolutions,
		UseDebugLogs:       true,
		TempDirOfExecution: core.Env.TMP_DIR,
	})

	if err != nil {
		return req.MakeErr("Error al convertir la imagen: " + err.Error())
	}

	for _, e := range images {
		core.Log("image:: ", e.Name, e.Format, e.Resolution, " | Size:", len(e.Content))
	}

	for _, e := range images {
		aws.SendFileToS3(aws.FileToS3Args{
			Account:     1,
			Bucket:      core.Env.S3_BUCKET,
			Path:        "img-productos",
			FileContent: e.Content,
			Name:        fmt.Sprintf("%v-%v.%v", imageName, resolutionsMap[uint16(e.Resolution)], e.Format),
		})
	}

	return req.MakeResponse(map[string]int{"ok": 1})
}
