package exec

import (
	"app/core"
	"encoding/json"
	"strings"

	"github.com/ivanjoz/avif-webp-encoder/imageconv"
)

func CompressImage(args *core.ExecArgs) core.FuncResponse {

	input := imageconv.ImageConvertInput{}
	// Param6 stores the JSON conversion options in the compact ExecArgs payload.
	err := json.Unmarshal([]byte(args.Param6), &input)
	if err != nil {
		return args.MakeErr("Error al parsear los parámetros: ", err)
	}

	// Param5 carries the base64 image content.
	if len(args.Param5) < 40 {
		return args.MakeErr("No se ha recibido el contenido de la imagen")
	}

	if strings.Contains(args.Param5[0:40], "base64,") {
		args.Param5 = strings.Split(args.Param5, "base64,")[1]
	}

	input.Image = core.Base64ToBytes(args.Param5)

	if len(input.Image) == 0 {
		return args.MakeErr("Error al convertir el contenido de la imagen a bytes")
	}

	images, err := imageconv.Convert(input)

	if err != nil {
		return args.MakeErr("Error al convertir la imagen.", err)
	}

	imagesJson, err := json.Marshal(images)

	if err != nil {
		return args.MakeErr("Error al convertir las imágenes a JSON.", err)
	}

	response := core.FuncResponse{ContentJson: string(imagesJson)}
	return response
}
