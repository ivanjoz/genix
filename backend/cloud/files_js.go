//go:build js

package cloud

import (
	"errors"
	"strings"
	"syscall/js"
	"time"

	"github.com/ivanjoz/genix-orm/idb/indexeddb"
)

// defaultObjectStore has no default in the browser: the store needs the open database, so the
// browser entry point installs it with SetObjectStore before serving a request.
func defaultObjectStore() ObjectStore {
	panic("cloud: the browser build must install its object store with SetObjectStore at boot")
}

// indexedDBObjectStore is the ObjectStore of the browser build: one IndexedDB object store keyed by
// Path + "/" + Name, each object a Blob carrying its content type, plus the metadata a listing
// returns. Buckets do not exist locally, so SaveFileArgs.Bucket is ignored.
type indexedDBObjectStore struct {
	database  *indexeddb.Database
	storeName string
}

// NewIndexedDBObjectStore creates the object store when it is missing.
func NewIndexedDBObjectStore(database *indexeddb.Database, storeName string) (ObjectStore, error) {
	if err := database.EnsureStores(indexeddb.StoreSpec{Name: storeName}); err != nil {
		return nil, err
	}
	return &indexedDBObjectStore{database: database, storeName: storeName}, nil
}

func (store *indexedDBObjectStore) PutObject(args SaveFileArgs) error {
	if len(args.FileContent) == 0 {
		// LocalFilePath has no meaning in a browser tab: there is no filesystem to read it from.
		return errors.New("no hay acciones a realizar")
	}
	object := js.ValueOf(map[string]any{
		"content":      indexeddb.NewBlob(args.ContentType, args.FileContent),
		"contentType":  args.ContentType,
		"cacheControl": args.CacheControl,
		"size":         len(args.FileContent),
		"modified":     time.Now().UnixMilli(),
	})
	return store.database.Put(store.storeName, indexeddb.Entry{Key: args.Path + "/" + args.Name, Value: object})
}

func (store *indexedDBObjectStore) GetObject(args SaveFileArgs) ([]byte, error) {
	key := args.Path + "/" + args.Name
	object, exists, err := store.database.Get(store.storeName, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("object not found: " + key)
	}
	return indexeddb.BlobBytes(object.Get("content"))
}

// ListObjects follows ListObjectsV2: keys under Prefix, after StartAfter, at most MaxKeys.
func (store *indexedDBObjectStore) ListObjects(args SaveFileArgs) ([]FileInfo, error) {
	entries, err := store.database.GetAll(store.storeName, indexeddb.KeyRangePrefix(args.Prefix))
	if err != nil {
		return nil, err
	}
	files := []FileInfo{}
	for _, entry := range entries {
		key := entry.Key.(js.Value).String()
		if args.StartAfter != "" && strings.Compare(key, args.StartAfter) <= 0 {
			continue
		}
		files = append(files, FileInfo{
			Key:          key,
			Size:         int64(entry.Value.Get("size").Int()),
			LastModified: time.UnixMilli(int64(entry.Value.Get("modified").Float())),
		})
		if args.MaxKeys > 0 && len(files) == int(args.MaxKeys) {
			break
		}
	}
	return files, nil
}

func (store *indexedDBObjectStore) DeleteObject(args SaveFileArgs) error {
	return store.database.Delete(store.storeName, args.Path+"/"+args.Name)
}

func (store *indexedDBObjectStore) ObjectExists(args SaveFileArgs) (bool, error) {
	_, exists, err := store.database.Get(store.storeName, args.Path+"/"+args.Name)
	return exists, err
}
