//go:build !js

package core

import (
	"app/core/filestore"
	"app/db"
	"context"
	"errors"
	"hash/fnv"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	fareward "github.com/ivanjoz/fareward/go"
	"github.com/ivanjoz/genix-orm/utils"
)

// ConfigureFileStore installs the ORM's file store before any request or CLI function runs. A table
// with a delta index or CacheByIDs can't be written without one.
//
// Lambda keeps the files in S3 ([file_store].bucket). A VPS keeps them on its own disk under
// [file_store].vps_dir: the backend, the database and every cache file live on the same machine, so every
// process that writes a VPS keyspace must run on that VPS (the store locks across processes).
//
// The DataFrames keep their files in the same store, and their compactions take a fareward lock,
// installed here so no entry point that can compact a frame misses it.
func ConfigureFileStore() {
	if Env.IS_SERVERLESS {
		if Env.FILE_STORE_BUCKET == "" {
			panic("config.toml: [file_store].bucket is required on Lambda: the delta logs and the by-IDs slot files live there")
		}
		// The snapshot files spare a cold Lambda instance its database reads, so they live in S3
		// with everything else.
		db.SetFileStore(filestore.New(s3.NewFromConfig(GetAwsConfig()), Env.FILE_STORE_BUCKET, Env.FILE_STORE_ROOT))
		db.SetRecordSnapshotsEnabled(true)
	} else {
		durableStore, err := filestore.NewLocalDisk(filepath.Join(VPSFileStoreDir(), "orm"))
		if err != nil {
			panic("config.toml: [file_store].vps_dir is unusable: " + err.Error())
		}
		ConfigureVPSSnapshotCache(durableStore)
	}
	db.SetDataFrameLock(acquireDataFrameLock)
}

// VPSFileStoreDir is the directory a VPS keeps the ORM's files in: orm/ (delta logs, by-IDs and
// cache-group files), snapshots/ (the snapshot cache's disk tier) and search/ (the text index).
func VPSFileStoreDir() string {
	if Env.FILE_STORE_VPS_DIR != "" {
		return Env.FILE_STORE_VPS_DIR
	}
	return filepath.Join(ProjectTmpDir(), "file-store")
}


// ConfigureVPSSnapshotCache installs durableStore with the snapshot files kept apart: in memory up to
// a cap, then on disk up to a second cap, the least recently used dropped. A snapshot turns a repeated
// first sync into a read of what changed since; being a cache, it is bounded instead of piling up in
// the durable store. [file_store].vps_snapshot_cache_mb = 0 turns snapshots off and reads every
// replace from the database. The in-process benchmarks call this directly, with an in-memory store.
func ConfigureVPSSnapshotCache(durableStore utils.FileStore) {
	if Env.FILE_STORE_VPS_SNAPSHOT_CACHE_MB <= 0 {
		db.SetFileStore(durableStore)
		db.SetRecordSnapshotsEnabled(false)
		return
	}
	directory := filepath.Join(VPSFileStoreDir(), "snapshots")
	localSnapshots, err := filestore.NewLocalSnapshots(
		Env.FILE_STORE_VPS_SNAPSHOT_CACHE_MB<<20, Env.FILE_STORE_VPS_SNAPSHOT_DISK_MB<<20, directory)
	if err != nil {
		// A disk tier that can't start must not stop the server: keep the memory tier.
		Log("Error: local snapshot directory unusable, keeping snapshots in memory only:", err)
		localSnapshots, _ = filestore.NewLocalSnapshots(Env.FILE_STORE_VPS_SNAPSHOT_CACHE_MB<<20, 0, "")
	}
	db.SetFileStore(filestore.WithLocalSnapshots(durableStore, localSnapshots, db.IsRecordSnapshotKey))
	db.SetRecordSnapshotsEnabled(true)
	files, _, diskBytes := localSnapshots.Usage()
	Log("*snapshot cache:: memory_mb", Env.FILE_STORE_VPS_SNAPSHOT_CACHE_MB, "disk_mb", Env.FILE_STORE_VPS_SNAPSHOT_DISK_MB,
		"dir", directory, "| kept from the last run:", files, "files,", diskBytes>>20, "MB")
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
	lock, err := AcquireLockWithOptions(context.Background(), ActionCompactDataFrame, int64(nameHash.Sum64()),
		LockOptions{MaxWaiters: dataFrameLockMaxWaiters, Wait: wait, Lease: lease})
	if errors.Is(err, fareward.ErrLockBusy) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return lock.Release, true, nil
}
