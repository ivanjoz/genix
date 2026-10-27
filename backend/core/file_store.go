package core

import (
	"app/core/filestore"
	"app/db"
	"context"
	"hash/fnv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ConfigureFileStore installs the ORM's file store before any request or CLI function runs. A table
// with a delta index or CacheByIDs can't be written without one, so a missing [file_store].bucket
// stops the boot. Local runs need it too: they write the same keyspace as the deployed backend, so
// they must write the same files.
//
// The DataFrames keep their files in the same store, and their runs take a fareward lock, installed
// here so no entry point that can run a frame misses it.
func ConfigureFileStore() {
	if Env.FILE_STORE_BUCKET == "" {
		panic("config.toml: [file_store].bucket is required: the delta logs and the by-IDs slot files live there")
	}
	db.SetFileStore(filestore.New(s3.NewFromConfig(GetAwsConfig()), Env.FILE_STORE_BUCKET, Env.FILE_STORE_ROOT))
	db.SetDataFrameLock(acquireDataFrameLock)
}

// acquireDataFrameLock is a try-lock on one frame. fareward's lease (15 s, clamped by the daemon) is
// shorter than a long rebuild, and lease is ignored: a lapsed lock lets a second run start, whose
// conditional writes then fail, so it costs work and never a wrong file.
func acquireDataFrameLock(lockName string, _ time.Duration) (func(), bool, error) {
	nameHash := fnv.New64a()
	nameHash.Write([]byte(lockName))
	lock, lockErr := AcquireLock(context.Background(), ActionMaterializeDataFrame, int64(nameHash.Sum64()), 0)
	if lockErr != nil {
		if lockErr.Busy() {
			return nil, false, nil
		}
		return nil, false, lockErr
	}
	return lock.Release, true, nil
}
