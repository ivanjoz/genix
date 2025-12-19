package aws

import (
	"app/core"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/ivanjoz/avif-webp-encoder/imageconv"
	"golang.org/x/sync/errgroup"
)

type FileToS3Args struct {
	Account       uint8
	Bucket        string
	LocalFilePath string
	FileContent   []byte
	Name          string
	Path          string
	Prefix        string
	StartAfter    string
	ContentType   string
	MaxKeys       int32
}

func SendFileToS3(args FileToS3Args) error {
	core.Log("Enviando a s3:", args.Bucket, "| Folder:", args.Path, "|", args.Name, "|", args.ContentType)

	client := s3.NewFromConfig(core.GetAwsConfig())

	key := args.Path + "/" + args.Name

	input := s3.PutObjectInput{
		Bucket: &args.Bucket,
		Key:    &key,
	}

	if len(args.ContentType) > 0 {
		input.ContentType = &args.ContentType
	}

	if len(args.LocalFilePath) > 1 {
		file, err := os.Open(args.LocalFilePath)
		if err != nil {
			core.Log("Error al abrir el archivo local", err)
			return err
		}
		input.Body = file
	} else if len(args.FileContent) > 0 {
		input.Body = bytes.NewReader(args.FileContent)
	} else {
		core.Log("No hay acciones a realizar")
		return errors.New("no hay acciones a realizar")
	}

	output, err := client.PutObject(context.TODO(), &input)
	if err != nil {
		core.Log("Error al enviar el archivo a S3", err)
		return err
	}

	core.Log("Respuesta recibida (ok):", output.ETag)
	return nil
}

func GetFileFromS3(args FileToS3Args) ([]byte, error) {
	core.Log("Obteniendo archvivo a s3 | Path: ", args.Path, " | ", args.Name)
	/*
		// Create a file to download to
		file, err := os.Create(args.Name)
		if err != nil {
			return nil, err
		}
		defer file.Close()
	*/

	buf := manager.NewWriteAtBuffer([]byte{})
	client := s3.NewFromConfig(core.GetAwsConfig())

	downloader := manager.NewDownloader(client)

	_, err := downloader.Download(context.TODO(), buf, &s3.GetObjectInput{
		Bucket: core.PtrString(args.Bucket),
		Key:    core.PtrString(args.Path + "/" + args.Name),
	})

	if err != nil {
		core.Log("error:: ", err)
		return nil, err
	}

	return buf.Bytes(), nil
}

func GetObjectFromFileS3[T any](args FileToS3Args, obj *T) (*T, error) {
	core.Log("Obteniendo archivo de s3::")
	core.Print(args)

	client := s3.NewFromConfig(core.GetAwsConfig())

	requestInput := &s3.GetObjectInput{
		Bucket: core.PtrString(args.Bucket),
		Key:    core.PtrString(args.Path + "/" + args.Name),
	}

	result, err := client.GetObject(context.TODO(), requestInput)
	if err != nil {
		core.Log(err)
	}

	defer result.Body.Close()

	/*body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		core.Log(err)
	}

	bodyString := string(body)
	decoder := json.NewDecoder(strings.NewReader(bodyString))
	err = decoder.Decode(obj)
	if err != nil {
		core.Log("twas an error")
	}*/

	byteValue, _ := io.ReadAll(result.Body)
	err = json.Unmarshal([]byte(byteValue), obj)

	if err != nil {
		core.Log("Error:: ", err.Error())
		return nil, err
	}

	return obj, nil
}

func S3ListFiles(args FileToS3Args) ([]types.Object, error) {
	core.Log("Envío de archivo a s3::")

	client := s3.NewFromConfig(core.GetAwsConfig())
	input := &s3.ListObjectsV2Input{
		Bucket: core.PtrString(args.Bucket),
	}
	if len(args.Prefix) > 0 {
		input.Prefix = &args.Prefix
	}
	if len(args.StartAfter) > 0 {
		input.StartAfter = &args.StartAfter
	}
	if args.MaxKeys > 0 {
		input.MaxKeys = &args.MaxKeys
	}

	result, err := client.ListObjectsV2(context.TODO(), input)
	var contents []types.Object

	if err != nil {
		log.Printf("Couldn't list objects in bucket %v. Here's why: %v\n", args.Bucket, err)
	} else {
		contents = result.Contents
	}

	return contents, nil
}

type ImageArgs struct {
	Order       int32
	Content     string /* Base64 webp image */
	Folder      string
	Name        string
	Description string
	Resolutions map[uint16]string
	Type        string
	Resolution  int8
}

const USE_MULTILAMBDA = true

func SaveConvertImage(args ImageArgs) ([]imageconv.Image, error) {
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

	convertInputBase := imageconv.ImageConvertInput{
		UseWebp:      true,
		UseAvif:      true,
		Resolutions:  resolutions,
		UseDebugLogs: true,
	}

	images := []imageconv.Image{}

	saveImage := func(image imageconv.Image) {
		fmt.Println("args.Folder:", args.Folder)

		args := FileToS3Args{
			Bucket:      core.Env.S3_BUCKET,
			Path:        args.Folder,
			FileContent: image.Content,
			ContentType: fmt.Sprintf("image/%v", image.Format),
			Name: fmt.Sprintf("%v-%v.%v", args.Name,
				args.Resolutions[uint16(image.Resolution)], image.Format),
		}
		SendFileToS3(args)
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
				Param6:        args.Content,
				Param7:        string(convertInputJson),
				ParseResponse: true,
			}

			core.Log("Invocando lambda de conversión de imagen. | Resolution: ", resolution)

			group.Go(func() error {
				lambdaOuput := ExecLambda(lambdaInput)
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

func SaveImage(image ImageArgs) (string, error) {

	fmt.Println("args.Folder:", image.Folder)

	if strings.Contains(image.Content[0:40], "base64,") {
		image.Content = strings.Split(image.Content, "base64,")[1]
	}

	bytes := core.Base64ToBytes(image.Content)

	args := FileToS3Args{
		Bucket:      core.Env.S3_BUCKET,
		Path:        image.Folder,
		FileContent: bytes,
		ContentType: fmt.Sprintf("image/%v", image.Type),
		Name:        fmt.Sprintf("%v-x%v.%v", image.Name, image.Resolution, image.Type),
	}
	err := SendFileToS3(args)
	return args.Name, err
}
