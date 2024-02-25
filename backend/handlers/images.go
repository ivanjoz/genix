package handlers

import (
	"app/aws"
	"app/core"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ivanjoz/avif-webp-encoder/imageconv"
	"golang.org/x/sync/errgroup"
)

type imageBody struct {
	Content string `json:"content"` /* Base64 webp image */
	Folder  string `json:"filter"`
	Name    string `json:"name"`
}

const USE_MULTILAMBDA = true
const IMAGE_S3_PATH = "img-productos"

func PostImage(req *core.HandlerArgs) core.HandlerResponse {
	core.Env.LOGS_FULL = true
	fmt.Println("API de conversión de imágenes. Usando Multilambda:", USE_MULTILAMBDA)

	resolutionsMap := map[uint16]string{
		960: "x6", 520: "x4", 340: "x2",
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

	convertInputBase := imageconv.ImageConvertInput{
		UseWebp:      true,
		UseAvif:      true,
		Resolutions:  resolutions,
		UseDebugLogs: true,
	}

	imageName := fmt.Sprintf("%v", time.Now().UnixMilli())

	saveImage := func(image imageconv.Image) {
		args := aws.FileToS3Args{
			Bucket:      core.Env.S3_BUCKET,
			Path:        IMAGE_S3_PATH,
			FileContent: image.Content,
			ContentType: fmt.Sprintf("image/%v", image.Format),
			Name: fmt.Sprintf("%v-%v.%v", imageName,
				resolutionsMap[uint16(image.Resolution)], image.Format),
		}
		aws.SendFileToS3(args)
	}

	if USE_MULTILAMBDA {
		group := errgroup.Group{}

		for resolution := range resolutionsMap {
			convertInput := convertInputBase
			convertInput.Resolutions = []uint16{resolution}

			convertInputJson, err := json.Marshal(convertInput)

			if err != nil {
				return req.MakeErr("No pudo convertir el input de la Lambda a JSON (Imágenes)")
			}

			lambdaInput := core.ExecArgs{
				LambdaName:    core.Env.LAMBDA_NAME + "_2",
				FuncToExec:    "compress-image",
				Param6:        image.Content,
				Param7:        string(convertInputJson),
				ParseResponse: true,
			}

			core.Log("Invocando lambda de conversión de imagen. | Resolution: ", resolution)

			group.Go(func() error {
				lambdaOuput := aws.ExecLambda(lambdaInput)
				if len(lambdaOuput.Error) > 0 {
					return fmt.Errorf("%v", lambdaOuput.Error)
				}

				images := []imageconv.Image{}
				err = json.Unmarshal([]byte(lambdaOuput.Response.ContentJson), &images)

				if err != nil {
					core.Log("*" + core.StrCut(lambdaOuput.Response.ContentJson, 400))
					return fmt.Errorf("%v", "No se pudo parsear la respuesta como JSON (Imágenes)")
				}
				for _, e := range images {
					saveImage(e)
				}
				return nil
			})
		}

		if err := group.Wait(); err != nil {
			return req.MakeErr(err)
		}
	} else {
		if strings.Contains(image.Content[0:40], "base64,") {
			image.Content = strings.Split(image.Content, "base64,")[1]
		}

		bytes := core.Base64ToBytes(image.Content)

		if len(bytes) == 0 {
			return req.MakeErr("Error al convertir el contenido de la imagen a bytes")
		}

		images, err := imageconv.Convert(imageconv.ImageConvertInput{
			Image:        bytes,
			UseWebp:      true,
			UseAvif:      true,
			Resolutions:  resolutions,
			UseDebugLogs: true,
		})

		if err != nil {
			return req.MakeErr("Error al convertir la imagen: " + err.Error())
		}

		for _, e := range images {
			core.Log("image:: ", e.Name, e.Format, e.Resolution, " | Size:", len(e.Content))
		}

		for _, e := range images {
			saveImage(e)
		}
	}

	response := map[string]string{"imageName": IMAGE_S3_PATH + "/" + imageName}
	return req.MakeResponse(response)
}
