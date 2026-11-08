package filestore

// The ORM's file store on a VPS: a local directory.
//
// On a VPS the backend, the database and every cache file live on the same machine, so the delta logs,
// the by-IDs slot files and the cache-group files are plain files under one directory. S3 is the
// Lambda path only (store.go).
//
// The semantics are S3's, which the ORM is written against:
//   - the ETag is a hash of the content, so a conditional write fails whenever the file changed;
//   - a write replaces the file atomically (temporary file + rename), a reader never sees half of it;
//   - concurrent appends all land.
//
// Every operation on a key holds that key's stripe lock: a mutex inside the process and an flock on
// the stripe's lock file across processes, because the genix-agent binary on the same VPS writes the
// same directory. A read takes it too, so it never sees an append half written.

import (
	"context"
	"errors"
	"fmt"
	"hash/crc64"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/ivanjoz/genix-orm/utils"
)

// localDiskStripes bounds the lock files: two keys share a lock only when they hash to one stripe.
const localDiskStripes = 64

// lockDirectoryName holds the stripe lock files; no ORM key starts with a dot.
const lockDirectoryName = ".locks"

var crc64Table = crc64.MakeTable(crc64.ECMA)

// LocalDiskStore is a utils.FileStore in a local directory. Safe for concurrent use, also across
// processes on the same machine.
type LocalDiskStore struct {
	root    string
	stripes [localDiskStripes]localDiskStripe
}

type localDiskStripe struct {
	mutex    sync.Mutex
	lockFile *os.File
}

// NewLocalDisk opens (creating it if needed) the store rooted at directory.
func NewLocalDisk(directory string) (*LocalDiskStore, error) {
	store := &LocalDiskStore{root: directory}
	lockDirectory := filepath.Join(directory, lockDirectoryName)
	if err := os.MkdirAll(lockDirectory, 0o700); err != nil {
		return nil, fmt.Errorf("local file store: creating %s: %w", lockDirectory, err)
	}
	for stripeIndex := range store.stripes {
		lockFile, err := os.OpenFile(filepath.Join(lockDirectory, strconv.Itoa(stripeIndex)), os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, fmt.Errorf("local file store: opening lock %d: %w", stripeIndex, err)
		}
		store.stripes[stripeIndex].lockFile = lockFile
	}
	return store, nil
}

// lockKey takes the key's stripe in this process, then across processes. The returned func releases both.
func (store *LocalDiskStore) lockKey(key string) func() {
	keyHash := fnv.New32a()
	keyHash.Write([]byte(key))
	stripe := &store.stripes[keyHash.Sum32()%localDiskStripes]
	stripe.mutex.Lock()
	syscall.Flock(int(stripe.lockFile.Fd()), syscall.LOCK_EX)
	return func() {
		syscall.Flock(int(stripe.lockFile.Fd()), syscall.LOCK_UN)
		stripe.mutex.Unlock()
	}
}

// pathOf maps a key to its file. Keys are relative slash paths; one that would leave the root is refused.
func (store *LocalDiskStore) pathOf(key string) (string, error) {
	cleanKey := filepath.Clean(filepath.FromSlash(key))
	if key == "" || filepath.IsAbs(cleanKey) || cleanKey == ".." || strings.HasPrefix(cleanKey, ".."+string(filepath.Separator)) ||
		strings.HasPrefix(cleanKey, lockDirectoryName) {
		return "", fmt.Errorf("local file store: invalid key %q", key)
	}
	return filepath.Join(store.root, cleanKey), nil
}

// contentETag is S3's notion of an ETag: equal content, equal tag. The length makes a collision of
// two different files need the same size and the same CRC-64.
func contentETag(content []byte) string {
	return strconv.FormatUint(crc64.Checksum(content, crc64Table), 16) + "-" + strconv.Itoa(len(content))
}

// readLocked reads the file under its lock. A missing file wraps utils.ErrFileNotFound.
func readLocked(key, path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", key, utils.ErrFileNotFound)
	}
	return content, err
}

func (store *LocalDiskStore) GetFile(key string) ([]byte, string, error) {
	path, err := store.pathOf(key)
	if err != nil {
		return nil, "", err
	}
	unlock := store.lockKey(key)
	defer unlock()
	content, err := readLocked(key, path)
	if err != nil {
		return nil, "", err
	}
	return content, contentETag(content), nil
}

func (store *LocalDiskStore) PutFileIfMatch(ctx context.Context, key string, content []byte, etag string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := store.pathOf(key)
	if err != nil {
		return "", err
	}
	unlock := store.lockKey(key)
	defer unlock()

	currentETag := ""
	currentContent, err := readLocked(key, path)
	switch {
	case err == nil:
		currentETag = contentETag(currentContent)
	case !errors.Is(err, utils.ErrFileNotFound):
		return "", err
	}
	if currentETag != etag {
		return "", fmt.Errorf("%s: %w", key, utils.ErrFilePreconditionFailed)
	}

	// Temporary file + rename: a crash or a reader in another process sees the old file or the new one.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("local file store: %s: %w", key, err)
	}
	temporaryPath := path + temporarySuffix
	if err := os.WriteFile(temporaryPath, content, 0o600); err != nil {
		os.Remove(temporaryPath)
		return "", fmt.Errorf("local file store: writing %s: %w", key, err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		os.Remove(temporaryPath)
		return "", fmt.Errorf("local file store: replacing %s: %w", key, err)
	}
	return contentETag(content), nil
}

func (store *LocalDiskStore) AppendBytesToFile(ctx context.Context, key string, content []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := store.pathOf(key)
	if err != nil {
		return err
	}
	unlock := store.lockKey(key)
	defer unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("local file store: %s: %w", key, err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("local file store: opening %s: %w", key, err)
	}
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("local file store: appending to %s: %w", key, writeErr)
	}
	return closeErr
}

// ListFiles walks the deepest directory the prefix names and keeps the keys that start with it.
func (store *LocalDiskStore) ListFiles(prefix string) ([]string, error) {
	walkRoot := store.root
	if slash := strings.LastIndexByte(prefix, '/'); slash >= 0 {
		prefixDirectory, err := store.pathOf(prefix[:slash])
		if err != nil {
			return nil, err
		}
		walkRoot = prefixDirectory
	}
	keys := []string{}
	err := filepath.WalkDir(walkRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == lockDirectoryName {
				return filepath.SkipDir
			}
			return nil
		}
		relativePath, _ := filepath.Rel(store.root, path)
		key := filepath.ToSlash(relativePath)
		if strings.HasPrefix(key, prefix) && !strings.HasSuffix(key, temporarySuffix) {
			keys = append(keys, key)
		}
		return nil
	})
	return keys, err
}

func (store *LocalDiskStore) DeleteFiles(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return err
		}
		path, err := store.pathOf(key)
		if err != nil {
			return err
		}
		unlock := store.lockKey(key)
		removeErr := os.Remove(path)
		unlock()
		if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return fmt.Errorf("local file store: deleting %s: %w", key, removeErr)
		}
	}
	return nil
}
