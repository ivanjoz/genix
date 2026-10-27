package core

import (
	"app/core/filestore"
	"app/db"
	"context"
	"errors"
	"hash/fnv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	fareward "github.com/ivanjoz/fareward/go"
)

// ConfigureFileStore installs the ORM's file store before any request or CLI function runs. A table
// with a delta index or CacheByIDs can't be written without one, so a missing [file_store].bucket
// stops the boot. Local runs need it too: they write the same keyspace as the deployed backend, so
// they must write the same files.
//
// The DataFrames keep their files in the same store, and their compactions take a fareward lock,
// installed here so no entry point that can compact a frame misses it.
func ConfigureFileStore() {
	if Env.FILE_STORE_BUCKET == "" {
		panic("config.toml: [file_store].bucket is required: the delta logs and the by-IDs slot files live there")
	}
	db.SetFileStore(filestore.New(s3.NewFromConfig(GetAwsConfig()), Env.FILE_STORE_BUCKET, Env.FILE_STORE_ROOT))
	db.SetDataFrameLock(acquireDataFrameLock)
}

// dataFrameLockMaxWaiters bounds the reads of one company queued behind a compaction of one frame:
// each one waits for the files the holder writes. Past it a read is refused the lock and answers
// from the log instead, which stays correct.
const dataFrameLockMaxWaiters = 16

// acquireDataFrameLock takes the lock of one frame partition, queuing up to wait behind its holder.
// lease bounds the hold (the daemon clamps it to its own ceiling, 60 s by default): a holder that
// dies or overruns lets the next waiter in, whose conditional writes win over the lapsed holder's.
func acquireDataFrameLock(lockName string, wait, lease time.Duration) (func(), bool, error) {
	nameHash := fnv.New64a()
	nameHash.Write([]byte(lockName))
	lock, err := fareward.AcquireLockWithOptions(context.Background(), uint16(ActionCompactDataFrame), int64(nameHash.Sum64()),
		fareward.LockOptions{MaxWaiters: dataFrameLockMaxWaiters, Wait: wait, Lease: lease})
	if errors.Is(err, fareward.ErrLockBusy) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return lock.Release, true, nil
}
