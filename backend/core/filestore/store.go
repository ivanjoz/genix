// Package filestore is the ORM's file store on S3 ([file_store]): the by-IDs slot files (ids-cache/)
// and the delta logs (delta-log/). It implements utils.FileStore on one bucket, in Express or
// compatibility mode by the bucket's name. core.ConfigureFileStore installs it at boot.
package filestore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/ivanjoz/genix-orm/utils"
)

// The two modes of AppendBytesToFile, picked by the bucket's name:
//
//   - Express mode: a bucket named "<name>--<zone>--x-s3" is an S3 Express One Zone directory bucket,
//     where S3 appends natively: a PutObject with WriteOffsetBytes set to the file's current size adds
//     the bytes at its end. Two appenders that read the same size race for that offset: one lands, the
//     other gets InvalidWriteOffset, reads the new size and tries again.
//   - Compatibility mode: any other bucket is a general purpose one, which has no append. An append is
//     a GET of the whole file and a PUT of it plus the new bytes, conditioned on the ETag it read; the
//     loser of a round reads again. Each round costs the whole file twice, so it suits small files.
//
// Both modes retry a lost round appendAttempts times, pausing a random time below attempt ×
// appendPauseStep (full jitter) so the losers don't retry in lockstep: 30 attempts serialize about 20
// concurrent appenders of one file.
const (
	expressBucketSuffix = "--x-s3"
	appendAttempts      = 30
	appendPauseStep     = 50 * time.Millisecond
)

// s3FileStore is the ORM's FileStore on one bucket: every key lives under root.
type s3FileStore struct {
	client    *s3.Client
	bucket    string
	root      string
	isExpress bool
}

// New is the file store on bucket, every key under root ("" for the bucket's top level). The bucket's
// name picks the append mode (see above).
func New(client *s3.Client, bucket, root string) utils.FileStore {
	return &s3FileStore{client: client, bucket: bucket, root: root, isExpress: strings.HasSuffix(bucket, expressBucketSuffix)}
}

func (store *s3FileStore) objectKey(key string) *string { return aws.String(store.root + key) }

func (store *s3FileStore) GetFile(key string) ([]byte, string, error) {
	return store.getFile(context.Background(), key)
}

func (store *s3FileStore) getFile(ctx context.Context, key string) ([]byte, string, error) {
	out, err := store.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(store.bucket), Key: store.objectKey(key)})
	var noSuchKey *s3types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return nil, "", fmt.Errorf("%s: %w", key, utils.ErrFileNotFound)
	}
	if err != nil {
		return nil, "", err
	}
	defer out.Body.Close()
	content, err := io.ReadAll(out.Body)
	return content, aws.ToString(out.ETag), err
}

// PutFileIfMatch uses S3's conditional writes: If-Match on the ETag, or If-None-Match: * for "must
// not exist". A 409 means another conditional write to the key was in flight: the caller treats it as
// a lost condition.
func (store *s3FileStore) PutFileIfMatch(ctx context.Context, key string, content []byte, etag string) (string, error) {
	input := &s3.PutObjectInput{Bucket: aws.String(store.bucket), Key: store.objectKey(key), Body: bytes.NewReader(content)}
	if etag == "" {
		input.IfNoneMatch = aws.String("*")
	} else {
		input.IfMatch = aws.String(etag)
	}
	out, err := store.client.PutObject(ctx, input)
	if hasHTTPStatus(err, http.StatusPreconditionFailed) || hasHTTPStatus(err, http.StatusConflict) {
		return "", fmt.Errorf("%s: %w", key, utils.ErrFilePreconditionFailed)
	}
	if err != nil {
		return "", err
	}
	return aws.ToString(out.ETag), nil
}

// AppendBytesToFile adds content at the end of the file, creating it when it is missing, in the
// bucket's mode. It gives up when ctx ends: every request and pause takes it.
func (store *s3FileStore) AppendBytesToFile(ctx context.Context, key string, content []byte) error {
	appendRound := store.appendRoundCompatible
	if store.isExpress {
		appendRound = store.appendRoundExpress
	}
	for attempt := 1; ; attempt++ {
		isLost, err := appendRound(ctx, key, content)
		if !isLost || attempt == appendAttempts {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(rand.Int64N(int64(attempt) * int64(appendPauseStep)))):
		}
	}
}

// appendRoundCompatible is one round of the compatibility mode: GET the file, PUT it back with content
// at its end only while its ETag is unchanged. isLost reports a concurrent write won the round.
func (store *s3FileStore) appendRoundCompatible(ctx context.Context, key string, content []byte) (isLost bool, err error) {
	storedContent, etag, err := store.getFile(ctx, key)
	if err != nil && !errors.Is(err, utils.ErrFileNotFound) {
		return false, err
	}
	_, err = store.PutFileIfMatch(ctx, key, append(storedContent, content...), etag)
	return errors.Is(err, utils.ErrFilePreconditionFailed), err
}

// appendRoundExpress is one round of the Express mode: HEAD the file for its size, then append at that
// offset. A missing file is created with a PUT that must not find one. isLost reports another
// appender moved the size (InvalidWriteOffset, or a 409 for a write in flight) or created the file
// first.
func (store *s3FileStore) appendRoundExpress(ctx context.Context, key string, content []byte) (isLost bool, err error) {
	head, err := store.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(store.bucket), Key: store.objectKey(key)})
	var notFound *s3types.NotFound
	if errors.As(err, &notFound) {
		_, err = store.PutFileIfMatch(ctx, key, content, "")
		return errors.Is(err, utils.ErrFilePreconditionFailed), err
	}
	if err != nil {
		return false, err
	}
	_, err = store.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(store.bucket), Key: store.objectKey(key), Body: bytes.NewReader(content),
		WriteOffsetBytes: aws.Int64(aws.ToInt64(head.ContentLength)),
	})
	var invalidWriteOffset *s3types.InvalidWriteOffset
	return errors.As(err, &invalidWriteOffset) || hasHTTPStatus(err, http.StatusConflict), err
}

// ListFiles returns every key under prefix, relative to root. A directory bucket only lists prefixes
// that end in "/".
func (store *s3FileStore) ListFiles(prefix string) ([]string, error) {
	var keys []string
	pages := s3.NewListObjectsV2Paginator(store.client, &s3.ListObjectsV2Input{Bucket: aws.String(store.bucket), Prefix: store.objectKey(prefix)})
	for pages.HasMorePages() {
		page, err := pages.NextPage(context.Background())
		if err != nil {
			return nil, err
		}
		for _, object := range page.Contents {
			keys = append(keys, strings.TrimPrefix(aws.ToString(object.Key), store.root))
		}
	}
	return keys, nil
}

// DeleteFiles removes the keys with DeleteObjects, 1000 per call (its cap). A missing key is a success.
func (store *s3FileStore) DeleteFiles(ctx context.Context, keys ...string) error {
	for start := 0; start < len(keys); start += 1000 {
		batchKeys := keys[start:min(start+1000, len(keys))]
		objects := make([]s3types.ObjectIdentifier, len(batchKeys))
		for i, key := range batchKeys {
			objects[i] = s3types.ObjectIdentifier{Key: store.objectKey(key)}
		}
		out, err := store.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(store.bucket), Delete: &s3types.Delete{Objects: objects, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return err
		}
		if len(out.Errors) > 0 {
			return fmt.Errorf("filestore: deleting %s: %s", aws.ToString(out.Errors[0].Key), aws.ToString(out.Errors[0].Message))
		}
	}
	return nil
}

// hasHTTPStatus reports whether err is an S3 response with that HTTP status.
func hasHTTPStatus(err error, statusCode int) bool {
	var responseErr *awshttp.ResponseError
	return errors.As(err, &responseErr) && responseErr.HTTPStatusCode() == statusCode
}
