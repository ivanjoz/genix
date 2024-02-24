package exec

import (
	"app/core"
	"encoding/json"
	"strings"

	"github.com/ivanjoz/avif-webp-encoder/imageconv"
)

func CompressImage(args *core.ExecArgs) core.FuncResponse {

	input := imageconv.ImageConvertInput{}
	err := json.Unmarshal([]byte(args.Param7), &input)
	if err != nil {
		return args.MakeErr("Error al parsear los parámetros: ", err)
	}

	if len(args.Param6) < 40 {
		return args.MakeErr("No se ha recibido el contenido de la imagen")
	}

	if strings.Contains(args.Param6[0:40], "base64,") {
		args.Param6 = strings.Split(args.Param6, "base64,")[1]
	}

	input.Image = core.Base64ToBytes(args.Param6)

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
