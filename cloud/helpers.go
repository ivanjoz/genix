package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/klauspost/compress/zstd"
	"github.com/kr/pretty"
)

// LambdaEnvironmentMaxBytes es el límite duro de AWS para la suma de claves y valores del
// entorno de una función. No hay forma de subirlo, así que es la CONFIG la que tiene que caber.
const LambdaEnvironmentMaxBytes = 4096

// StripTomlComments quita los comentarios de línea completa y las líneas vacías del archivo de
// configuración antes de comprimirlo. config.toml está documentado línea por línea y eso ya no
// cabe: comprimido y en base64 pasa de los 4 KB del entorno de la Lambda. El backend parsea TOML,
// donde un comentario no aporta nada, así que se pierden sólo en la copia que viaja.
//
// Sólo mira el primer carácter no vacío de cada línea, lo que es seguro porque config.toml no usa
// cadenas multilínea ("""): dentro de una, un # inicial sería contenido y no un comentario.
// scripts/deployer/lambda_env.go repite esta función porque es otro módulo Go.
func StripTomlComments(content []byte) []byte {
	keptLines := []string{}
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		keptLines = append(keptLines, line)
	}
	return []byte(strings.Join(keptLines, "\n") + "\n")
}

// AssertLambdaEnvironmentFits falla con el tamaño a la vista en vez de dejar que AWS devuelva su
// InvalidParameterValueException, que no dice qué variable creció ni cuánto margen queda.
func AssertLambdaEnvironmentFits(variables map[string]string) {
	totalBytes := 0
	for name, value := range variables {
		totalBytes += len(name) + len(value)
	}
	if totalBytes <= LambdaEnvironmentMaxBytes {
		fmt.Printf("Entorno de la Lambda: %v de %v bytes\n", totalBytes, LambdaEnvironmentMaxBytes)
		return
	}
	panic(fmt.Sprintf(
		"el entorno de la Lambda mide %v bytes y el límite de AWS es %v: recorte config.toml "+
			"(los comentarios ya no viajan) o mueva alguna sección fuera de la CONFIG",
		totalBytes, LambdaEnvironmentMaxBytes))
}

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
