package cloud

import (
	"app/core"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ObjectStore is the public object store: the files the CDN serves or the backend keeps as
// artifacts (product images, webpage JSON, company config, invoice XML, backups). It is not the
// ORM's internal file store (utils.FileStore), which has no content type, cache headers or buckets.
//
// Implementations: S3 and R2 for the server (s3.go), IndexedDB for the browser (files_js.go). An
// object's key is always Path + "/" + Name.
type ObjectStore interface {
	PutObject(args SaveFileArgs) error
	GetObject(args SaveFileArgs) ([]byte, error)
	ListObjects(args SaveFileArgs) ([]FileInfo, error)
	// DeleteObject treats a missing object as deleted.
	DeleteObject(args SaveFileArgs) error
	// ObjectExists returns (false, nil) for a confirmed not-found and (false, err) for a failure.
	ObjectExists(args SaveFileArgs) (bool, error)
}

var (
	objectStore         ObjectStore
	objectStoreSelected sync.Once
)

// SetObjectStore installs the object store explicitly. The browser entry point does it at boot; the
// server leaves it to defaultObjectStore, which reads providers.cdn.
func SetObjectStore(store ObjectStore) { objectStore = store }

// currentObjectStore is the installed store, or the build's default chosen on first use — after
// config.toml has been read, which is why it is not chosen at package init.
func currentObjectStore() ObjectStore {
	objectStoreSelected.Do(func() {
		if objectStore == nil {
			objectStore = defaultObjectStore()
		}
	})
	return objectStore
}

func SaveFile(args SaveFileArgs) error { return currentObjectStore().PutObject(args) }

func GetFile(args SaveFileArgs) ([]byte, error) { return currentObjectStore().GetObject(args) }

func ListFiles(args SaveFileArgs) ([]FileInfo, error) { return currentObjectStore().ListObjects(args) }

func DeleteFile(args SaveFileArgs) error { return currentObjectStore().DeleteObject(args) }

func FileExists(args SaveFileArgs) (bool, error) { return currentObjectStore().ObjectExists(args) }

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

	// No Bucket: SaveFile resolves the provider's default one (S3_BUCKET or CLOUDFLARE_BUCKET).
	args := SaveFileArgs{
		Path:        image.Folder,
		FileContent: imageBytes,
		ContentType: fmt.Sprintf("image/%v", image.Type),
		Name:        objectName,
	}
	return args.Name, SaveFile(args)
}
