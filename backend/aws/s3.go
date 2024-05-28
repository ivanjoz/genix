package aws

import (
	"app/core"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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
