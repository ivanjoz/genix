//go:build avif

package cloud

import (
	"app/core"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ivanjoz/avif-webp-encoder/imageconv"
	"golang.org/x/sync/errgroup"
)

const USE_MULTILAMBDA = true

// ConvertImageForLambda is the entry point the `_2` conversion Lambda handler calls.
func ConvertImageForLambda(input ImageConvertInput) ([]Image, error) {
	return convertImage(input)
}

// convertImage is the only place in the repo that calls the encoder. Everything else goes through
// the local DTOs in image_convert_types.go, so this file is the single import of imageconv and the
// single reason the 4.17 MB embedded binary is linked. Present only under `-tags avif`.
func convertImage(input ImageConvertInput) ([]Image, error) {
	converted, err := imageconv.Convert(imageconv.ImageConvertInput(input))
	if err != nil {
		return nil, err
	}
	images := make([]Image, len(converted))
	for i, c := range converted {
		images[i] = Image(c)
	}
	return images, nil
}

func SaveConvertImage(args ImageArgs) ([]Image, error) {
	fmt.Println("API de conversión de imágenes. Usando Multilambda:", USE_MULTILAMBDA)

	/*
		resolutionsMap := map[uint16]string{
			980: "x6", 540: "x4", 340: "x2",
		}
	*/

	resolutions := []uint16{}
	for r := range args.Resolutions {
		resolutions = append(resolutions, r)
	}

	if len(args.Content) < 40 {
		return nil, core.Err("No se ha recibido el contenido de la imagen")
	}

	convertInputBase := ImageConvertInput{
		UseWebp:      true,
		UseAvif:      true,
		Resolutions:  resolutions,
		UseDebugLogs: true,
	}

	images := []Image{}

	saveImage := func(image Image) {
		fmt.Println("args.Folder:", args.Folder)

		// An empty resolution label marks the base image, which is stored without a
		// suffix (e.g. "<companyID>_<imageID>.avif"); other resolutions get "-<label>".
		resolutionLabel := args.Resolutions[uint16(image.Resolution)]
		objectName := fmt.Sprintf("%v.%v", args.Name, image.Format)
		if len(resolutionLabel) > 0 {
			objectName = fmt.Sprintf("%v-%v.%v", args.Name, resolutionLabel, image.Format)
		}

		args := SaveFileArgs{
			Bucket:      core.Env.S3_BUCKET,
			Path:        args.Folder,
			FileContent: image.Content,
			ContentType: fmt.Sprintf("image/%v", image.Format),
			Name:        objectName,
		}
		SaveFile(args)
		image.Content = nil
		images = append(images, image)
	}

	if USE_MULTILAMBDA {
		group := errgroup.Group{}

		for resolution := range args.Resolutions {
			convertInput := convertInputBase
			convertInput.Resolutions = []uint16{resolution}

			convertInputJson, err := json.Marshal(convertInput)

			if err != nil {
				return nil, core.Err("No pudo convertir el input de la Lambda a JSON (Imágenes)")
			}

			lambdaInput := core.ExecArgs{
				LambdaName:    core.Env.LAMBDA_NAME + "_2",
				FuncToExec:    "compress-image",
				Param5:        args.Content,
				Param6:        string(convertInputJson),
				ParseResponse: true,
			}

			core.Log("Invocando lambda de conversión de imagen. | Resolution: ", resolution)

			group.Go(func() error {
				lambdaOuput := ExecLambda(lambdaInput)
				if len(lambdaOuput.Error) > 0 {
					return fmt.Errorf("%v", lambdaOuput.Error)
				}

				images := []Image{}
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

		images, err := convertImage(ImageConvertInput{
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
