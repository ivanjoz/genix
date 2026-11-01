// The public object store's arguments and results, shared by both builds: s3.go implements the
// store on S3 / R2 for the server, files_js.go for the browser.
package cloud

import (
	"app/core"
	"fmt"
	"strings"
	"time"
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
