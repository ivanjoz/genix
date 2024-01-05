package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/klauspost/compress/zstd"
	"github.com/kr/pretty"
)

func MakeAwsConfig(profile, region string) (aws.Config, error) {
	var cfg aws.Config
	var err error

	setConfig := func(lo *config.LoadOptions) error {
		lo.Region = region
		return nil
	}

	cfg, err = config.LoadDefaultConfig(
		context.TODO(), config.WithSharedConfigProfile(profile), setConfig)

	return cfg, err
}

func ReadFile(filePath string) ([]byte, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.New("Error opening file: " + err.Error())
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.New("Error reading file: " + err.Error())
	}

	return fileBytes, nil
}

type StackEventsLogger struct {
	Client    *cloudformation.Client
	StackName string
	LogsMap   map[string]types.StackEvent
	UnixTime  int64
	Start     bool
}

func (e *StackEventsLogger) GetCurrentEvents() {

	if e.LogsMap == nil {
		e.LogsMap = map[string]types.StackEvent{}
		if e.UnixTime == 0 {
			e.UnixTime = time.Now().Unix() - 5
		}
	}

	input := cloudformation.DescribeStackEventsInput{
		StackName: &e.StackName,
	}
	result, err := e.Client.DescribeStackEvents(context.TODO(), &input)

	if err != nil {
		fmt.Println("Error al obtener los eventos del Stack: ", e.StackName)
		fmt.Println(err)
		return
	}

	for _, se := range result.StackEvents {
		if se.Timestamp.Unix() < e.UnixTime {
			continue
		}
		if _, ok := e.LogsMap[*se.EventId]; ok {
			continue
		}
		e.LogsMap[*se.EventId] = se

		resourceId := ""
		if se.LogicalResourceId != nil {
			resourceId = *se.LogicalResourceId
		}
		statusReason := ""
		if se.ResourceStatusReason != nil {
			statusReason = *se.ResourceStatusReason
		}

		fmt.Println(*se.Timestamp, se.ResourceStatus, resourceId, statusReason)
	}
}

func (e *StackEventsLogger) GetCurrentEventsAll() {
	e.Start = true
	for {
		if !e.Start {
			break
		}

		fmt.Println("Consultando Eventos...")
		e.GetCurrentEvents()
		time.Sleep(2 * time.Second)
	}
}

func Print(Struct any) {
	pretty.Println(Struct)
}

type FileToS3Args struct {
	Account       uint8
	Bucket        string
	LocalFilePath string
	FilePath      string
}

// Envío o actualización de archivos en S3
func SendFileToS3Client(args FileToS3Args, client *s3.Client) error {

	file, err := os.Open(args.LocalFilePath)
	if err != nil {
		panic("Error al abrir el archivo local" + err.Error())
	}

	input := s3.PutObjectInput{
		Bucket:   &args.Bucket,
		Metadata: map[string]string{},
		Key:      &args.FilePath,
		Body:     file,
	}

	_, err = client.PutObject(context.TODO(), &input)
	if err != nil {
		panic("Error al enviar el archivo a S3" + err.Error())
	}

	fmt.Println("S3 File Saved!", args.Bucket, args.FilePath)
	return nil
}

// Compresión con Zstd
func CompressZstd(content *string) []byte {
	encoder, _ := zstd.NewWriter(nil)
	src := []byte(*content)
	compressed := encoder.EncodeAll(src, make([]byte, 0, len(src)))
	return compressed
}

func DecompressZstd(content *[]byte) string {
	decoder, _ := zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	decompressed, err := decoder.DecodeAll(*content, nil)
	if err != nil {
		fmt.Println("Error al descomprimir: " + err.Error())
		return ""
	}
	return string(decompressed)
}

func MakeB64UrlEncode(contentString string) string {
	contentString = strings.ReplaceAll(contentString, "/", "_")
	contentString = strings.ReplaceAll(contentString, "+", "-")
	contentString = strings.ReplaceAll(contentString, "=", "~")
	return contentString
}

func MakeB64UrlDecode(contentString string) string {
	contentString = strings.ReplaceAll(contentString, "_", "/")
	contentString = strings.ReplaceAll(contentString, "-", "+")
	contentString = strings.ReplaceAll(contentString, "~", "=")
	return contentString
}

func BytesToBase64(source []byte, useUrlEncoded ...bool) string {
	contentString := base64.StdEncoding.EncodeToString(source)
	if len(useUrlEncoded) == 1 && useUrlEncoded[0] {
		contentString = MakeB64UrlEncode(contentString)
	}
	return contentString
}
