//go:build !js

package core

import (
	"app/db"
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ivanjoz/genix-orm/search/genixs3"
)

// textSearchRootSuffix places the text index under the file store's root, beside the delta logs
// and the by-IDs slot files: the project runs on one S3 Express One Zone bucket.
const textSearchRootSuffix = "search/"

// ConfigureTextSearch installs genix-s3 as the ORM's text-search backend, before any write to a
// table with TextSearch. The index lives in the [file_store] bucket under <root>search/; the
// bucket must be in the region of GetAwsConfig.
//
// The ORM packages cannot import core (cycle), so the resolved config is pushed in from here.
func ConfigureTextSearch() {
	if Env.FILE_STORE_BUCKET == "" {
		panic("config.toml: [file_store].bucket is required: the text search index lives there")
	}
	// Compaction defaults to CompactAuto: inside the write on Lambda, in a goroutine on a server.
	backend, err := genixs3.New(context.Background(), genixs3.Config{
		Bucket: Env.FILE_STORE_BUCKET,
		Root:   Env.FILE_STORE_ROOT + textSearchRootSuffix,
		S3:     s3.NewFromConfig(GetAwsConfig()),
		OnCompactError: func(table, partition string, err error) {
			Log("text search: compaction of", table, partition, "failed:", err)
		},
	})
	if err != nil {
		panic("invalid genix-s3 text search configuration: " + err.Error())
	}
	db.SetTextSearchBackend(backend)
}
