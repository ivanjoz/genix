//go:build js

package cloud

import (
	"testing"

	"github.com/ivanjoz/genix-orm/idb/indexeddb"
)

// Run under Node with an IndexedDB polyfill, see genix-orm/idb/filestore_test.go.
func TestIndexedDBObjectStore(t *testing.T) {
	database, err := indexeddb.Open("cloud-objects-test")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewIndexedDBObjectStore(database, "objects")
	if err != nil {
		t.Fatal(err)
	}
	SetObjectStore(store)

	for _, name := range []string{"a.json", "b.json", "c.json"} {
		if err := SaveFile(SaveFileArgs{Path: "empresas", Name: name, FileContent: []byte("{}"), ContentType: "application/json"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := SaveFile(SaveFileArgs{Path: "live", Name: "x.db", FileContent: []byte("rows")}); err != nil {
		t.Fatal(err)
	}

	if content, err := GetFile(SaveFileArgs{Path: "live", Name: "x.db"}); err != nil || string(content) != "rows" {
		t.Fatalf("get: %q %v", content, err)
	}
	files, err := ListFiles(SaveFileArgs{Prefix: "empresas/", StartAfter: "empresas/a.json", MaxKeys: 1})
	if err != nil || len(files) != 1 || files[0].Key != "empresas/b.json" || files[0].Size != 2 {
		t.Fatalf("list: %+v %v", files, err)
	}

	if err := DeleteFile(SaveFileArgs{Path: "live", Name: "x.db"}); err != nil {
		t.Fatal(err)
	}
	if exists, err := FileExists(SaveFileArgs{Path: "live", Name: "x.db"}); err != nil || exists {
		t.Fatalf("exists after delete: %v %v", exists, err)
	}
	if _, err := GetFile(SaveFileArgs{Path: "live", Name: "x.db"}); err == nil {
		t.Fatal("get after delete: want an error")
	}
}
