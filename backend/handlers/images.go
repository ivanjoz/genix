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

type ImageArgs struct {
	Content     string /* Base64 webp image */
	Folder      string
	Name        string
	Description string
}

const USE_MULTILAMBDA = true

func PostImage(req *core.HandlerArgs) core.HandlerResponse {
	image := ImageArgs{}
	err := json.Unmarshal([]byte(*req.Body), &image)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	image.Name = fmt.Sprintf("%v", time.Now().UnixMilli())
	image.Folder = "img-productos"
	_, err = saveImage(image)
	if err != nil {
		return req.MakeErr("Error al guardar la imagen: " + err.Error())
	}

	response := map[string]string{
		"imageName": image.Folder + "/" + image.Name,
	}
	return req.MakeResponse(response)
}

func saveImage(args ImageArgs) ([]imageconv.Image, error) {
	fmt.Println("API de conversión de imágenes. Usando Multilambda:", USE_MULTILAMBDA)

	resolutionsMap := map[uint16]string{
		980: "x6", 540: "x4", 340: "x2",
	}

	resolutions := []uint16{}
	for r := range resolutionsMap {
		resolutions = append(resolutions, r)
	}

	if len(args.Content) < 40 {
		return nil, core.Err("No se ha recibido el contenido de la imagen")
	}

	convertInputBase := imageconv.ImageConvertInput{
		UseWebp:      true,
		UseAvif:      true,
		Resolutions:  resolutions,
		UseDebugLogs: true,
	}

	images := []imageconv.Image{}

	saveImage := func(image imageconv.Image) {
		args := aws.FileToS3Args{
			Bucket:      core.Env.S3_BUCKET,
			Path:        args.Folder,
			FileContent: image.Content,
			ContentType: fmt.Sprintf("image/%v", image.Format),
			Name: fmt.Sprintf("%v-%v.%v", args.Name,
				resolutionsMap[uint16(image.Resolution)], image.Format),
		}
		aws.SendFileToS3(args)
		image.Content = nil
		images = append(images, image)
	}

	if USE_MULTILAMBDA {
		group := errgroup.Group{}

		for resolution := range resolutionsMap {
			convertInput := convertInputBase
			convertInput.Resolutions = []uint16{resolution}

			convertInputJson, err := json.Marshal(convertInput)

			if err != nil {
				return nil, core.Err("No pudo convertir el input de la Lambda a JSON (Imágenes)")
			}

			lambdaInput := core.ExecArgs{
				LambdaName:    core.Env.LAMBDA_NAME + "_2",
				FuncToExec:    "compress-image",
				Param6:        args.Content,
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
			return nil, err
		}
	} else {
		if strings.Contains(args.Content[0:40], "base64,") {
			args.Content = strings.Split(args.Content, "base64,")[1]
		}

		bytes := core.Base64ToBytes(args.Content)

		if len(bytes) == 0 {
			return nil, core.Err("Error al convertir el contenido de la imagen a bytes")
		}

		images, err := imageconv.Convert(imageconv.ImageConvertInput{
			Image:        bytes,
			UseWebp:      true,
			UseAvif:      true,
			Resolutions:  resolutions,
			UseDebugLogs: true,
		})

		if err != nil {
			return nil, core.Err("Error al convertir la imagen:", err)
		}

		for _, e := range images {
			core.Log("image:: ", e.Name, e.Format, e.Resolution, " | Size:", len(e.Content))
		}

		for _, e := range images {
			saveImage(e)
		}
	}
	return images, nil
}
