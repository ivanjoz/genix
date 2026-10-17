package core

import (
	"app/core/filestore"
	"app/db"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ConfigureFileStore installs the ORM's file store before any request or CLI function runs. A table
// with a delta index or CacheByIDs can't be written without one, so a missing [file_store].bucket
// stops the boot. Local runs need it too: they write the same keyspace as the deployed backend, so
// they must write the same files.
func ConfigureFileStore() {
	if Env.FILE_STORE_BUCKET == "" {
		panic("config.toml: [file_store].bucket is required: the delta logs and the by-IDs slot files live there")
	}
	db.SetFileStore(filestore.New(s3.NewFromConfig(GetAwsConfig()), Env.FILE_STORE_BUCKET, Env.FILE_STORE_ROOT))
}
