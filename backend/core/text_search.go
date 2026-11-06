//go:build !js

package core

import (
	"app/db"
	"context"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ivanjoz/genix-orm/search/genixs3"
	genix "github.com/ivanjoz/genix-search/genix-s3"
)

// textSearchRootSuffix places the text index beside the delta logs and the by-IDs slot files: under
// the bucket's root on Lambda, under vps_dir on a VPS.
const textSearchRootSuffix = "search/"

// ConfigureTextSearch installs genix-s3 as the ORM's text-search backend, before any write to a
// table with TextSearch. Like the file store (ConfigureFileStore), the index lives in the
// [file_store] bucket on Lambda (in the region of GetAwsConfig) and on the VPS's own disk otherwise.
//
// The ORM packages cannot import core (cycle), so the resolved config is pushed in from here.
func ConfigureTextSearch() {
	searchConfig := genixs3.Config{
		Root: Env.FILE_STORE_ROOT + textSearchRootSuffix,
		OnCompactError: func(table, partition string, err error) {
			Log("text search: compaction of", table, partition, "failed:", err)
		},
	}
	// Compaction defaults to CompactAuto: inside the write on Lambda, in a goroutine on a server.
	var backend *genixs3.Backend
	var err error
	if Env.IS_SERVERLESS {
		if Env.FILE_STORE_BUCKET == "" {
			panic("config.toml: [file_store].bucket is required on Lambda: the text search index lives there")
		}
		searchConfig.Bucket = Env.FILE_STORE_BUCKET
		searchConfig.S3 = s3.NewFromConfig(GetAwsConfig())
		backend, err = genixs3.New(context.Background(), searchConfig)
	} else {
		searchConfig.Root = ""
		var directoryStore genix.Blob
		if directoryStore, err = genix.NewFSStore(filepath.Join(VPSFileStoreDir(), textSearchRootSuffix)); err == nil {
			backend, err = genixs3.NewWithStore(directoryStore, searchConfig)
		}
	}
	if err != nil {
		panic("invalid genix-s3 text search configuration: " + err.Error())
	}
	db.SetTextSearchBackend(backend)
}
