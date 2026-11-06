package filestore

// Local snapshot files for a long-running server (VPS).
//
// The ORM's snapshot files (delta and cache-group) hold a database read so the next first sync of
// the same prefix starts from the file and reads only what changed since it. On Lambda they live in
// S3, which is what a cold instance needs; a long-running server reads S3 at about the cost of
// reading ScyllaDB, so it keeps them here instead: in memory up to a byte cap, the least recently
// used spilled to a local directory up to a second cap, and the oldest of those deleted.
//
// A snapshot is a cache, never the only copy of anything: the ORM validates every one it serves
// against the database, and a missing file only means the next read goes to ScyllaDB. That is what
// lets this store drop files freely, and lose the memory tier on a restart. Only snapshot keys may
// come here — WithLocalSnapshots routes everything else (delta logs, by-IDs files) to the durable
// store.

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ivanjoz/genix-orm/utils"
)

// LocalSnapshots is a utils.FileStore for snapshot files: memory first, then a local directory.
// Safe for concurrent use. GetFile hands out the stored bytes without copying them (a snapshot is
// megabytes, read on every first sync); callers only decode them.
type LocalSnapshots struct {
	mutex sync.Mutex
	// entries by key; recency orders them, most recently used at the front.
	entries map[string]*localSnapshot
	recency *list.List
	// memoryBytes and diskBytes are what each tier holds; the caps bound them.
	memoryBytes, memoryCapBytes int64
	diskBytes, diskCapBytes     int64
	// directory holds the spilled files, one per key. "" keeps everything in memory.
	directory string
	// lastETag numbers every write, so a PutFileIfMatch can tell whether the file changed.
	lastETag uint64
}

type localSnapshot struct {
	key  string
	etag string
	// content is the file while it is in memory, nil once spilled.
	content []byte
	size    int64
	// isOnDisk: the file is in directory. isSpilling: it is being written there, still served from
	// content until that finishes.
	isOnDisk, isSpilling bool
	element              *list.Element
}

// NewLocalSnapshots holds up to memoryCapBytes in memory and diskCapBytes in directory (0 or ""
// for no disk tier). Files a previous run left in directory are kept, oldest first in line for
// deletion: they are as good as they were, and the ORM checks their age before serving them.
func NewLocalSnapshots(memoryCapBytes, diskCapBytes int64, directory string) (*LocalSnapshots, error) {
	store := &LocalSnapshots{
		entries:        map[string]*localSnapshot{},
		recency:        list.New(),
		memoryCapBytes: memoryCapBytes,
		diskCapBytes:   diskCapBytes,
	}
	if diskCapBytes <= 0 || directory == "" {
		return store, nil
	}
	store.directory = directory
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("local snapshots: creating %s: %w", directory, err)
	}
	if err := store.indexSpilledFiles(); err != nil {
		return nil, err
	}
	return store, nil
}

// indexSpilledFiles registers the files a previous run left, newest at the front. A half-written
// temporary file (a crash mid-spill) is deleted.
func (store *LocalSnapshots) indexSpilledFiles() error {
	dirEntries, err := os.ReadDir(store.directory)
	if err != nil {
		return fmt.Errorf("local snapshots: reading %s: %w", store.directory, err)
	}
	type spilledFile struct {
		name, key string
		size      int64
		modified  time.Time
	}
	spilledFiles := []spilledFile{}
	for _, dirEntry := range dirEntries {
		info, infoErr := dirEntry.Info()
		if dirEntry.IsDir() || infoErr != nil {
			continue
		}
		escapedKey, _, hasETag := strings.Cut(dirEntry.Name(), etagSeparator)
		key, unescapeErr := url.PathUnescape(escapedKey)
		if strings.HasSuffix(dirEntry.Name(), temporarySuffix) || !hasETag || unescapeErr != nil {
			os.Remove(filepath.Join(store.directory, dirEntry.Name()))
			continue
		}
		spilledFiles = append(spilledFiles, spilledFile{name: dirEntry.Name(), key: key, size: info.Size(), modified: info.ModTime()})
	}
	// Oldest first, so the newest ends at the front of recency, and a key written twice before a
	// crash keeps its newest file.
	slices.SortFunc(spilledFiles, func(left, right spilledFile) int { return left.modified.Compare(right.modified) })
	for _, spilled := range spilledFiles {
		if older, exists := store.entries[spilled.key]; exists {
			store.forget(older)
		}
		entry := &localSnapshot{key: spilled.key, etag: store.nextETag(), size: spilled.size, isOnDisk: true}
		// ETags restart with the process, so the file is renamed to the one it gets now.
		if err := os.Rename(filepath.Join(store.directory, spilled.name), store.filePath(entry)); err != nil {
			continue
		}
		entry.element = store.recency.PushFront(entry)
		store.entries[spilled.key] = entry
		store.diskBytes += spilled.size
	}
	store.deleteOverDiskCap()
	return nil
}

const (
	temporarySuffix = ".tmp"
	etagSeparator   = "@"
)

func (store *LocalSnapshots) nextETag() string {
	store.lastETag++
	return strconv.FormatUint(store.lastETag, 10)
}

// filePath is one version of a file: the key escaped into one name (the directory stays flat) and
// its ETag, so two versions of a key spilling at once never write the same path.
func (store *LocalSnapshots) filePath(entry *localSnapshot) string {
	return filepath.Join(store.directory, url.PathEscape(entry.key)+etagSeparator+entry.etag)
}

func (store *LocalSnapshots) GetFile(key string) ([]byte, string, error) {
	store.mutex.Lock()
	entry, exists := store.entries[key]
	if !exists {
		store.mutex.Unlock()
		return nil, "", fmt.Errorf("%s: %w", key, utils.ErrFileNotFound)
	}
	store.recency.MoveToFront(entry.element)
	if entry.content != nil {
		content, etag := entry.content, entry.etag
		store.mutex.Unlock()
		return content, etag, nil
	}
	etag, path := entry.etag, store.filePath(entry)
	store.mutex.Unlock()

	// Read outside the lock: a spilled file is megabytes. If it was deleted meanwhile (over the
	// disk cap, or replaced), the read is a miss like any other.
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", key, utils.ErrFileNotFound)
	}
	return content, etag, nil
}

// PutFileIfMatch stores the file in memory when its current ETag is etag ("" = absent), then
// spills what no longer fits. The spill writes happen after the lock is released.
func (store *LocalSnapshots) PutFileIfMatch(_ context.Context, key string, content []byte, etag string) (string, error) {
	store.mutex.Lock()
	entry, exists := store.entries[key]
	currentETag := ""
	if exists {
		currentETag = entry.etag
	}
	if currentETag != etag {
		store.mutex.Unlock()
		return "", fmt.Errorf("%s: %w", key, utils.ErrFilePreconditionFailed)
	}
	if exists {
		store.forget(entry)
	}
	entry = &localSnapshot{key: key, etag: store.nextETag(), content: content, size: int64(len(content))}
	entry.element = store.recency.PushFront(entry)
	store.entries[key] = entry
	store.memoryBytes += entry.size
	newETag := entry.etag
	spills := store.takeSpillsOverMemoryCap()
	store.mutex.Unlock()

	store.writeSpills(spills)
	return newETag, nil
}

// forget removes the entry from both tiers' accounting. Its disk file, if any, is deleted.
func (store *LocalSnapshots) forget(entry *localSnapshot) {
	store.recency.Remove(entry.element)
	delete(store.entries, entry.key)
	if entry.content != nil {
		store.memoryBytes -= entry.size
	}
	if entry.isOnDisk {
		store.diskBytes -= entry.size
		os.Remove(store.filePath(entry))
	}
}

// takeSpillsOverMemoryCap picks the least recently used in-memory files until memory fits. With no
// disk tier they are dropped; otherwise they are marked as spilling and returned to be written.
func (store *LocalSnapshots) takeSpillsOverMemoryCap() []*localSnapshot {
	spills := []*localSnapshot{}
	pendingBytes := int64(0)
	for element := store.recency.Back(); element != nil && store.memoryBytes-pendingBytes > store.memoryCapBytes; {
		entry := element.Value.(*localSnapshot)
		element = element.Prev()
		if entry.content == nil || entry.isSpilling {
			continue
		}
		if store.directory == "" {
			store.forget(entry)
			continue
		}
		entry.isSpilling = true
		pendingBytes += entry.size
		spills = append(spills, entry)
	}
	return spills
}

// writeSpills writes each file to the directory and moves it to the disk tier, unless it was
// replaced or dropped while it was being written — then the written version is stale and goes.
func (store *LocalSnapshots) writeSpills(spills []*localSnapshot) {
	for _, entry := range spills {
		path := store.filePath(entry)
		temporaryPath := path + temporarySuffix
		writeErr := os.WriteFile(temporaryPath, entry.content, 0o600)
		if writeErr == nil {
			writeErr = os.Rename(temporaryPath, path)
		}

		store.mutex.Lock()
		switch {
		case store.entries[entry.key] != entry:
			// Replaced or dropped while spilling: this version is nobody's file any more.
			os.Remove(path)
			os.Remove(temporaryPath)
		case writeErr != nil:
			// The disk refused it: drop the file, the next read goes to the database.
			store.forget(entry)
			os.Remove(temporaryPath)
		default:
			entry.content, entry.isSpilling, entry.isOnDisk = nil, false, true
			store.memoryBytes -= entry.size
			store.diskBytes += entry.size
		}
		store.deleteOverDiskCap()
		store.mutex.Unlock()
	}
}

// deleteOverDiskCap deletes the least recently used spilled files until the disk tier fits.
func (store *LocalSnapshots) deleteOverDiskCap() {
	for element := store.recency.Back(); element != nil && store.diskBytes > store.diskCapBytes; {
		entry := element.Value.(*localSnapshot)
		element = element.Prev()
		if entry.isOnDisk {
			store.forget(entry)
		}
	}
}

// Usage reports what each tier holds, for the boot log and tests.
func (store *LocalSnapshots) Usage() (files int, memoryBytes, diskBytes int64) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return len(store.entries), store.memoryBytes, store.diskBytes
}

var errSnapshotsOnly = errors.New("local snapshots: only snapshot files are kept here")

func (store *LocalSnapshots) AppendBytesToFile(_ context.Context, key string, _ []byte) error {
	return fmt.Errorf("%s: %w", key, errSnapshotsOnly)
}

func (store *LocalSnapshots) ListFiles(prefix string) ([]string, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	keys := []string{}
	for key := range store.entries {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (store *LocalSnapshots) DeleteFiles(_ context.Context, keys ...string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	for _, key := range keys {
		if entry, exists := store.entries[key]; exists {
			store.forget(entry)
		}
	}
	return nil
}

// WithLocalSnapshots is the file store a long-running server installs: snapshot keys go to local,
// every other key to durable. isSnapshotKey names which is which (db.IsRecordSnapshotKey).
func WithLocalSnapshots(durable utils.FileStore, local *LocalSnapshots, isSnapshotKey func(string) bool) utils.FileStore {
	return &snapshotRouter{durable: durable, local: local, isSnapshotKey: isSnapshotKey}
}

type snapshotRouter struct {
	durable       utils.FileStore
	local         *LocalSnapshots
	isSnapshotKey func(string) bool
}

func (router *snapshotRouter) storeFor(key string) utils.FileStore {
	if router.isSnapshotKey(key) {
		return router.local
	}
	return router.durable
}

func (router *snapshotRouter) GetFile(key string) ([]byte, string, error) {
	return router.storeFor(key).GetFile(key)
}

func (router *snapshotRouter) PutFileIfMatch(ctx context.Context, key string, content []byte, etag string) (string, error) {
	return router.storeFor(key).PutFileIfMatch(ctx, key, content, etag)
}

func (router *snapshotRouter) AppendBytesToFile(ctx context.Context, key string, content []byte) error {
	return router.storeFor(key).AppendBytesToFile(ctx, key, content)
}

// ListFiles asks both stores: a prefix can span snapshot and protocol keys.
func (router *snapshotRouter) ListFiles(prefix string) ([]string, error) {
	durableKeys, err := router.durable.ListFiles(prefix)
	if err != nil {
		return nil, err
	}
	localKeys, _ := router.local.ListFiles(prefix)
	return append(durableKeys, localKeys...), nil
}

func (router *snapshotRouter) DeleteFiles(ctx context.Context, keys ...string) error {
	localKeys, durableKeys := []string{}, []string{}
	for _, key := range keys {
		if router.isSnapshotKey(key) {
			localKeys = append(localKeys, key)
		} else {
			durableKeys = append(durableKeys, key)
		}
	}
	router.local.DeleteFiles(ctx, localKeys...)
	if len(durableKeys) == 0 {
		return nil
	}
	return router.durable.DeleteFiles(ctx, durableKeys...)
}
