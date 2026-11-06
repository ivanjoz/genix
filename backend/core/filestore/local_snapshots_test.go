package filestore

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/ivanjoz/genix-orm/utils"
)

func snapshotBytes(fill byte, size int) []byte { return bytes.Repeat([]byte{fill}, size) }

func mustPut(t *testing.T, store utils.FileStore, key string, content []byte, etag string) string {
	t.Helper()
	newETag, err := store.PutFileIfMatch(context.Background(), key, content, etag)
	if err != nil {
		t.Fatalf("put %s: %v", key, err)
	}
	return newETag
}

func mustGet(t *testing.T, store utils.FileStore, key string) []byte {
	t.Helper()
	content, _, err := store.GetFile(key)
	if err != nil {
		t.Fatalf("get %s: %v", key, err)
	}
	return content
}

// The ORM's write rule: a put succeeds only over the version it read ("" = none).
func TestLocalSnapshotsHonoursETags(t *testing.T) {
	store, _ := NewLocalSnapshots(1<<20, 0, "")
	if _, _, err := store.GetFile("delta-snapshot/a"); !errors.Is(err, utils.ErrFileNotFound) {
		t.Fatalf("missing file: %v", err)
	}
	firstETag := mustPut(t, store, "delta-snapshot/a", []byte("one"), "")
	if _, err := store.PutFileIfMatch(context.Background(), "delta-snapshot/a", []byte("two"), ""); !errors.Is(err, utils.ErrFilePreconditionFailed) {
		t.Fatalf("a put over an existing file without its ETag: %v", err)
	}
	mustPut(t, store, "delta-snapshot/a", []byte("two"), firstETag)
	if content := mustGet(t, store, "delta-snapshot/a"); string(content) != "two" {
		t.Fatalf("got %q", content)
	}
}

// Past the memory cap the least recently used file moves to disk and is still read; past the disk
// cap the least recently used spilled file is gone, and its read is a miss.
func TestLocalSnapshotsSpillsThenDeletes(t *testing.T) {
	store, err := NewLocalSnapshots(250, 250, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mustPut(t, store, "delta-snapshot/a", snapshotBytes('a', 100), "")
	mustPut(t, store, "delta-snapshot/b", snapshotBytes('b', 100), "")
	mustGet(t, store, "delta-snapshot/a") // a is now the most recently used
	mustPut(t, store, "delta-snapshot/c", snapshotBytes('c', 100), "")

	files, memoryBytes, diskBytes := store.Usage()
	if files != 3 || memoryBytes != 200 || diskBytes != 100 {
		t.Fatalf("after the first spill: files %d memory %d disk %d", files, memoryBytes, diskBytes)
	}
	if content := mustGet(t, store, "delta-snapshot/b"); !bytes.Equal(content, snapshotBytes('b', 100)) {
		t.Fatal("the spilled file did not read back")
	}

	// Two more spills put 300 bytes on a 250-byte disk: the least recently used spilled one goes.
	mustPut(t, store, "delta-snapshot/d", snapshotBytes('d', 100), "")
	mustPut(t, store, "delta-snapshot/e", snapshotBytes('e', 100), "")
	if _, _, diskBytes := store.Usage(); diskBytes > 250 {
		t.Fatalf("disk over its cap: %d", diskBytes)
	}
	missing := 0
	for _, key := range []string{"a", "b", "c", "d", "e"} {
		if _, _, err := store.GetFile("delta-snapshot/" + key); errors.Is(err, utils.ErrFileNotFound) {
			missing++
		}
	}
	if missing != 1 {
		t.Fatalf("expected one file deleted over the disk cap, %d are missing", missing)
	}
}

// With no disk tier, what does not fit in memory is dropped.
func TestLocalSnapshotsMemoryOnlyDrops(t *testing.T) {
	store, _ := NewLocalSnapshots(150, 0, "")
	mustPut(t, store, "delta-snapshot/a", snapshotBytes('a', 100), "")
	mustPut(t, store, "delta-snapshot/b", snapshotBytes('b', 100), "")
	if _, _, err := store.GetFile("delta-snapshot/a"); !errors.Is(err, utils.ErrFileNotFound) {
		t.Fatalf("the least recently used file should be gone: %v", err)
	}
	if files, memoryBytes, _ := store.Usage(); files != 1 || memoryBytes != 100 {
		t.Fatalf("files %d memory %d", files, memoryBytes)
	}
}

// A restart keeps what was spilled: the memory tier is lost, the disk tier is indexed again.
func TestLocalSnapshotsReindexesTheDiskAfterARestart(t *testing.T) {
	directory := t.TempDir()
	store, _ := NewLocalSnapshots(100, 1000, directory)
	mustPut(t, store, "delta-snapshot/products/7/status/all.l0.cb", snapshotBytes('a', 100), "")
	mustPut(t, store, "cache-group/sale_orders/7/2/-5.cb", snapshotBytes('b', 100), "") // spills the first

	restarted, err := NewLocalSnapshots(100, 1000, directory)
	if err != nil {
		t.Fatal(err)
	}
	if content := mustGet(t, restarted, "delta-snapshot/products/7/status/all.l0.cb"); !bytes.Equal(content, snapshotBytes('a', 100)) {
		t.Fatal("the spilled file did not survive the restart")
	}
	if _, _, err := restarted.GetFile("cache-group/sale_orders/7/2/-5.cb"); !errors.Is(err, utils.ErrFileNotFound) {
		t.Fatalf("a file only in memory cannot survive a restart: %v", err)
	}
}

// Concurrent puts and gets of the same keys keep the accounting consistent (run with -race).
func TestLocalSnapshotsConcurrentUse(t *testing.T) {
	store, _ := NewLocalSnapshots(1000, 1000, t.TempDir())
	var waitGroup sync.WaitGroup
	for worker := range 8 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for round := range 200 {
				key := "delta-snapshot/" + string(rune('a'+(worker+round)%6))
				_, etag, _ := store.GetFile(key)
				store.PutFileIfMatch(context.Background(), key, snapshotBytes(byte(round), 150), etag)
			}
		}()
	}
	waitGroup.Wait()
	_, memoryBytes, diskBytes := store.Usage()
	if memoryBytes > 1000+150 || diskBytes > 1000 || memoryBytes < 0 || diskBytes < 0 {
		t.Fatalf("memory %d disk %d", memoryBytes, diskBytes)
	}
}

// Snapshot keys go to the local store; the delta logs and by-IDs files stay on the durable one.
func TestWithLocalSnapshotsRoutesByKey(t *testing.T) {
	durable := utils.NewMemoryFileStore()
	local, _ := NewLocalSnapshots(1<<20, 0, "")
	isSnapshotKey := func(key string) bool { return strings.HasPrefix(key, "delta-snapshot/") }
	router := WithLocalSnapshots(durable, local, isSnapshotKey)

	mustPut(t, router, "delta-snapshot/a", []byte("snapshot"), "")
	mustPut(t, router, "delta-log/a", []byte("log"), "")
	if _, _, err := durable.GetFile("delta-snapshot/a"); !errors.Is(err, utils.ErrFileNotFound) {
		t.Fatal("a snapshot reached the durable store")
	}
	if _, _, err := local.GetFile("delta-log/a"); !errors.Is(err, utils.ErrFileNotFound) {
		t.Fatal("a delta log reached the local store")
	}
	if string(mustGet(t, router, "delta-snapshot/a")) != "snapshot" || string(mustGet(t, router, "delta-log/a")) != "log" {
		t.Fatal("the router did not read each key from its store")
	}
	if err := router.DeleteFiles(context.Background(), "delta-snapshot/a", "delta-log/a"); err != nil {
		t.Fatal(err)
	}
	if keys, _ := router.ListFiles("delta-"); len(keys) != 0 {
		t.Fatalf("left after delete: %v", keys)
	}
}
