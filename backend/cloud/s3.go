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
	"net/url"
	"os"
	"strings"
	"time"

	"net/http"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
)

// useR2 selects the object store independently from the cloud data mirror.
func useR2() bool {
	switch core.Env.CDN_PROVIDER {
	case "aws":
		return false
	case "cloudflare":
		return true
	default:
		panic("providers.cdn in config.toml is not set or invalid (must be 'aws' or 'cloudflare')")
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

// DeleteFile removes an object (Path + "/" + Name) from the store selected by CDN_PROVIDER. Deleting
// a missing object is not an error on either provider.
func DeleteFile(args SaveFileArgs) error {
	args = resolveStorageBucket(args)
	key := args.Path + "/" + args.Name
	if !useR2() {
		client := s3.NewFromConfig(core.GetAwsConfig())
		_, err := client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{Bucket: &args.Bucket, Key: &key})
		return err
	}
	req, err := http.NewRequest("DELETE", r2ObjectURL(args.Bucket, key), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+core.Env.CLOUDFLARE_TOKEN)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode != http.StatusNotFound {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("R2 delete failed (HTTP %d): %s", resp.StatusCode, string(responseBody))
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
