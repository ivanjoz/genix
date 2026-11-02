//go:build !js

package cloud

import (
	"app/core"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
)

// defaultObjectStore picks the server's object store from providers.cdn in config.toml. It runs
// once, on the first file operation (see currentObjectStore), so the S3 client is built once per
// process instead of once per call.
func defaultObjectStore() ObjectStore {
	switch core.Env.CDN_PROVIDER {
	case "aws":
		return &s3ObjectStore{client: s3.NewFromConfig(core.GetAwsConfig())}
	case "cloudflare":
		return r2ObjectStore{}
	default:
		panic("providers.cdn in config.toml is not set or invalid (must be 'aws' or 'cloudflare')")
	}
}

// readUploadBody is the content of an upload: a local file when LocalFilePath is set, otherwise
// FileContent. The caller closes the returned reader.
func readUploadBody(args SaveFileArgs) (io.ReadCloser, error) {
	if len(args.LocalFilePath) > 1 {
		file, err := os.Open(args.LocalFilePath)
		if err != nil {
			core.Log("Error al abrir el archivo local", err)
			return nil, err
		}
		return file, nil
	}
	if len(args.FileContent) > 0 {
		return io.NopCloser(bytes.NewReader(args.FileContent)), nil
	}
	core.Log("No hay acciones a realizar")
	return nil, errors.New("no hay acciones a realizar")
}

// ─── S3 ────────────────────────────────────────────────────────────────────────

type s3ObjectStore struct {
	client *s3.Client
}

// bucket is the caller's bucket, or the configured S3_BUCKET.
func (store *s3ObjectStore) bucket(args SaveFileArgs) *string {
	if args.Bucket == "" {
		return aws.String(core.Env.S3_BUCKET)
	}
	return aws.String(args.Bucket)
}

func (store *s3ObjectStore) PutObject(args SaveFileArgs) error {
	core.Log("Enviando a s3:", *store.bucket(args), "| Folder:", args.Path, "|", args.Name, "|", args.ContentType)

	body, err := readUploadBody(args)
	if err != nil {
		return err
	}
	defer body.Close()

	input := s3.PutObjectInput{
		Bucket: store.bucket(args),
		Key:    aws.String(args.Path + "/" + args.Name),
		Body:   body,
	}
	if len(args.ContentType) > 0 {
		input.ContentType = &args.ContentType
	}
	if len(args.CacheControl) > 0 {
		input.CacheControl = &args.CacheControl
	}

	output, err := store.client.PutObject(context.TODO(), &input)
	if err != nil {
		core.Log("Error al enviar el archivo a S3", err)
		return err
	}
	core.Log("Respuesta recibida (ok):", output.ETag)
	return nil
}

func (store *s3ObjectStore) GetObject(args SaveFileArgs) ([]byte, error) {
	result, err := store.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: store.bucket(args),
		Key:    aws.String(args.Path + "/" + args.Name),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func (store *s3ObjectStore) ListObjects(args SaveFileArgs) ([]FileInfo, error) {
	input := &s3.ListObjectsV2Input{Bucket: store.bucket(args)}
	if len(args.Prefix) > 0 {
		input.Prefix = &args.Prefix
	}
	if len(args.StartAfter) > 0 {
		input.StartAfter = &args.StartAfter
	}
	if args.MaxKeys > 0 {
		input.MaxKeys = &args.MaxKeys
	}

	result, err := store.client.ListObjectsV2(context.TODO(), input)
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

func (store *s3ObjectStore) DeleteObject(args SaveFileArgs) error {
	_, err := store.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: store.bucket(args),
		Key:    aws.String(args.Path + "/" + args.Name),
	})
	return err
}

func (store *s3ObjectStore) ObjectExists(args SaveFileArgs) (bool, error) {
	_, err := store.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: store.bucket(args),
		Key:    aws.String(args.Path + "/" + args.Name),
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

// ─── Cloudflare R2, through the account management API ─────────────────────────

type r2ObjectStore struct{}

// bucket is CLOUDFLARE_BUCKET unless the caller named a bucket of its own. Callers written for S3
// pass S3_BUCKET explicitly, which means "the default one" here too.
func (r2ObjectStore) bucket(args SaveFileArgs) string {
	if args.Bucket == "" || args.Bucket == core.Env.S3_BUCKET {
		return core.Env.CLOUDFLARE_BUCKET
	}
	return args.Bucket
}

// objectURL preserves slash-separated object paths while escaping special key characters.
func (store r2ObjectStore) objectURL(args SaveFileArgs) string {
	escapedKey := strings.ReplaceAll(url.PathEscape(args.Path+"/"+args.Name), "%2F", "/")
	return fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/r2/buckets/%s/objects/%s",
		core.Env.CLOUDFLARE_ACCOUNT, store.bucket(args), escapedKey)
}

// do sends an authenticated request and returns the response when its status is below 300 or in
// allowedStatus; any other status becomes an error naming action.
func (r2ObjectStore) do(request *http.Request, action string, allowedStatus ...int) (*http.Response, error) {
	request.Header.Set("Authorization", "Bearer "+core.Env.CLOUDFLARE_TOKEN)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusMultipleChoices {
		return response, nil
	}
	for _, status := range allowedStatus {
		if response.StatusCode == status {
			return response, nil
		}
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)
	return nil, fmt.Errorf("R2 %s failed (HTTP %d): %s", action, response.StatusCode, string(responseBody))
}

func (store r2ObjectStore) PutObject(args SaveFileArgs) error {
	core.Log("Enviando a R2:", store.bucket(args), "| Folder:", args.Path, "|", args.Name)

	body, err := readUploadBody(args)
	if err != nil {
		return err
	}
	defer body.Close()

	request, err := http.NewRequest("PUT", store.objectURL(args), body)
	if err != nil {
		return err
	}
	if len(args.ContentType) > 0 {
		request.Header.Set("Content-Type", args.ContentType)
	}
	if len(args.CacheControl) > 0 {
		request.Header.Set("Cache-Control", args.CacheControl)
	}
	response, err := store.do(request, "upload")
	if err != nil {
		return err
	}
	return response.Body.Close()
}

func (store r2ObjectStore) GetObject(args SaveFileArgs) ([]byte, error) {
	request, err := http.NewRequest("GET", store.objectURL(args), nil)
	if err != nil {
		return nil, err
	}
	response, err := store.do(request, "download")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}

func (store r2ObjectStore) ListObjects(args SaveFileArgs) ([]FileInfo, error) {
	requestURL, err := url.Parse(fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/r2/buckets/%s/objects",
		core.Env.CLOUDFLARE_ACCOUNT, store.bucket(args)))
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

	request, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := store.do(request, "list")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var listing struct {
		Result []struct {
			Key          string    `json:"key"`
			Size         int64     `json:"size"`
			LastModified time.Time `json:"last_modified"`
		} `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listing); err != nil {
		return nil, err
	}
	files := make([]FileInfo, 0, len(listing.Result))
	for _, object := range listing.Result {
		files = append(files, FileInfo{Key: object.Key, Size: object.Size, LastModified: object.LastModified})
	}
	return files, nil
}

// DeleteObject treats a missing object as deleted, as S3 does.
func (store r2ObjectStore) DeleteObject(args SaveFileArgs) error {
	request, err := http.NewRequest("DELETE", store.objectURL(args), nil)
	if err != nil {
		return err
	}
	response, err := store.do(request, "delete", http.StatusNotFound)
	if err != nil {
		return err
	}
	return response.Body.Close()
}

func (store r2ObjectStore) ObjectExists(args SaveFileArgs) (bool, error) {
	// The R2 management API only exposes GET/PUT/DELETE for objects (HEAD returns 405).
	// Use GET with a 1-byte Range so we confirm the object without paying the full download.
	request, err := http.NewRequest("GET", store.objectURL(args), nil)
	if err != nil {
		return false, err
	}
	request.Header.Set("Range", "bytes=0-0")
	response, err := store.do(request, "verify", http.StatusNotFound)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	io.Copy(io.Discard, response.Body)
	return response.StatusCode != http.StatusNotFound, nil
}
