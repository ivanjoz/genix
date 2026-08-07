package cloud

import (
	"app/core"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"net/http"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
	"github.com/ivanjoz/avif-webp-encoder/imageconv"
	"golang.org/x/sync/errgroup"
)

type SaveFileArgs struct {
	Account       uint8
	Bucket        string
	LocalFilePath string
	FileContent   []byte
	Name          string
	Path          string
	Prefix        string
	StartAfter    string
	ContentType   string
	CacheControl  string
	MaxKeys       int32
}

// FileInfo keeps object metadata independent from the storage provider.
type FileInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// useR2 selects the object store independently from the cloud data mirror.
func useR2() bool {
	switch core.Env.CDN_PROVIDER {
	case "aws":
		return false
	case "cloudflare":
		return true
	default:
		panic("CDN_PROVIDER in credentials.json is not set or invalid (must be 'aws' or 'cloudflare')")
	}
}

// resolveStorageBucket applies the provider's configured default bucket consistently.
func resolveStorageBucket(args SaveFileArgs) SaveFileArgs {
	if useR2() {
		if args.Bucket == "" || args.Bucket == core.Env.S3_BUCKET {
			args.Bucket = core.Env.CLOUDFLARE_BUCKET
		}
	} else if args.Bucket == "" {
		args.Bucket = core.Env.S3_BUCKET
	}
	return args
}

// r2ObjectURL preserves slash-separated object paths while escaping special key characters.
func r2ObjectURL(bucket, key string) string {
	escapedKey := strings.ReplaceAll(url.PathEscape(key), "%2F", "/")
	return fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/r2/buckets/%s/objects/%s",
		core.Env.CLOUDFLARE_ACCOUNT, bucket, escapedKey)
}

// SaveFile uploads to the object store selected by CDN_PROVIDER.
func SaveFile(args SaveFileArgs) error {
	args = resolveStorageBucket(args)
	if useR2() {
		return SaveFileToR2(args)
	}
	return saveFileToS3(args)
}

func saveFileToS3(args SaveFileArgs) error {
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
	if len(args.CacheControl) > 0 {
		input.CacheControl = &args.CacheControl
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

// GetFile downloads an object from the store selected by CDN_PROVIDER.
func GetFile(args SaveFileArgs) ([]byte, error) {
	args = resolveStorageBucket(args)
	if useR2() {
		return getFileFromR2(args)
	}
	return getFileFromS3(args)
}

func getFileFromS3(args SaveFileArgs) ([]byte, error) {
	client := s3.NewFromConfig(core.GetAwsConfig())
	result, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: core.PtrString(args.Bucket),
		Key:    core.PtrString(args.Path + "/" + args.Name),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func getFileFromR2(args SaveFileArgs) ([]byte, error) {
	req, err := http.NewRequest("GET", r2ObjectURL(args.Bucket, args.Path+"/"+args.Name), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+core.Env.CLOUDFLARE_TOKEN)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("R2 download failed (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}
	return io.ReadAll(resp.Body)
}

// ListFiles lists provider-neutral object metadata from the configured object store.
func ListFiles(args SaveFileArgs) ([]FileInfo, error) {
	args = resolveStorageBucket(args)
	if useR2() {
		return listFilesFromR2(args)
	}
	return listFilesFromS3(args)
}

func listFilesFromS3(args SaveFileArgs) ([]FileInfo, error) {
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
	if err != nil {
		return nil, err
	}
	files := make([]FileInfo, 0, len(result.Contents))
	for _, object := range result.Contents {
		files = append(files, FileInfo{
			Key:          aws.ToString(object.Key),
			Size:         aws.ToInt64(object.Size),
			LastModified: aws.ToTime(object.LastModified),
		})
	}
	return files, nil
}

func listFilesFromR2(args SaveFileArgs) ([]FileInfo, error) {
	requestURL, err := url.Parse(fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/r2/buckets/%s/objects", core.Env.CLOUDFLARE_ACCOUNT, args.Bucket))
	if err != nil {
		return nil, err
	}
	query := requestURL.Query()
	if args.Prefix != "" {
		query.Set("prefix", args.Prefix)
	}
	if args.StartAfter != "" {
		query.Set("start_after", args.StartAfter)
	}
	if args.MaxKeys > 0 {
		query.Set("per_page", fmt.Sprint(args.MaxKeys))
	}
	requestURL.RawQuery = query.Encode()
	req, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+core.Env.CLOUDFLARE_TOKEN)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("R2 list failed (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}
	var response struct {
		Result []struct {
			Key          string    `json:"key"`
			Size         int64     `json:"size"`
			LastModified time.Time `json:"last_modified"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	files := make([]FileInfo, 0, len(response.Result))
	for _, object := range response.Result {
		files = append(files, FileInfo{Key: object.Key, Size: object.Size, LastModified: object.LastModified})
	}
	return files, nil
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

func SaveFileToR2(args SaveFileArgs) error {
	key := args.Path + "/" + args.Name
	url := r2ObjectURL(args.Bucket, key)

	core.Log("Enviando a R2:", args.Bucket, "| Folder:", args.Path, "|", args.Name)

	var body io.Reader
	if len(args.LocalFilePath) > 1 {
		file, err := os.Open(args.LocalFilePath)
		if err != nil {
			return err
		}
		defer file.Close()
		body = file
	} else if len(args.FileContent) > 0 {
		body = bytes.NewReader(args.FileContent)
	} else {
		return errors.New("no hay acciones a realizar")
	}

	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+core.Env.CLOUDFLARE_TOKEN)
	if len(args.ContentType) > 0 {
		req.Header.Set("Content-Type", args.ContentType)
	}
	if len(args.CacheControl) > 0 {
		req.Header.Set("Cache-Control", args.CacheControl)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("R2 upload failed (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	return nil
}

// FileExists checks whether an object exists in the object store selected by CDN_PROVIDER.
// Returns (false, nil) for a confirmed not-found; (false, err) for transport errors.
func FileExists(args SaveFileArgs) (bool, error) {
	args = resolveStorageBucket(args)
	if useR2() {
		return fileExistsR2(args)
	}
	return fileExistsS3(args)
}

func fileExistsS3(args SaveFileArgs) (bool, error) {
	client := s3.NewFromConfig(core.GetAwsConfig())
	key := args.Path + "/" + args.Name
	_, err := client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: &args.Bucket,
		Key:    &key,
	})
	if err == nil {
		return true, nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey", "404":
			return false, nil
		}
	}
	return false, err
}

func fileExistsR2(args SaveFileArgs) (bool, error) {
	key := args.Path + "/" + args.Name
	url := r2ObjectURL(args.Bucket, key)

	// The R2 management API only exposes GET/PUT/DELETE for objects (HEAD returns 405).
	// Use GET with a 1-byte Range so we confirm the object without paying the full download.
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+core.Env.CLOUDFLARE_TOKEN)
	req.Header.Set("Range", "bytes=0-0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	}
	respBytes, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("R2 verify failed (HTTP %d): %s", resp.StatusCode, string(respBytes))
}

func SaveImage(image ImageArgs) (string, error) {

	fmt.Println("args.Folder:", image.Folder)

	if strings.Contains(image.Content[0:40], "base64,") {
		image.Content = strings.Split(image.Content, "base64,")[1]
	}

	imageBytes := core.Base64ToBytes(image.Content)

	// Resolution 6 (x6) is the base image, stored without a suffix; lower resolutions
	// (x4, x2) keep the "-x<resolution>" suffix so the frontend can derive their names.
	objectName := fmt.Sprintf("%v.%v", image.Name, image.Type)
	if image.Resolution != 6 {
		objectName = fmt.Sprintf("%v-x%v.%v", image.Name, image.Resolution, image.Type)
	}

	args := SaveFileArgs{
		Path:        image.Folder,
		FileContent: imageBytes,
		ContentType: fmt.Sprintf("image/%v", image.Type),
		Name:        objectName,
	}

	var err error
	if useR2() {
		args.Bucket = core.Env.CLOUDFLARE_BUCKET
		err = SaveFileToR2(args)
	} else {
		args.Bucket = core.Env.S3_BUCKET
		err = SaveFile(args)
	}
	return args.Name, err
}
