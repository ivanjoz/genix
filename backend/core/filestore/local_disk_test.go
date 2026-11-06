package filestore

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/ivanjoz/genix-orm/utils"
)

func TestLocalDiskConditionalWritesFollowTheETag(t *testing.T) {
	store, err := NewLocalDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, _, err := store.GetFile("delta-log/1/a"); !errors.Is(err, utils.ErrFileNotFound) {
		t.Fatalf("a missing file must be ErrFileNotFound, got %v", err)
	}
	firstETag, err := store.PutFileIfMatch(ctx, "delta-log/1/a", []byte("one"), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.PutFileIfMatch(ctx, "delta-log/1/a", []byte("two"), ""); !errors.Is(err, utils.ErrFilePreconditionFailed) {
		t.Fatalf("creating an existing file must fail the precondition, got %v", err)
	}
	if _, err := store.PutFileIfMatch(ctx, "delta-log/1/a", []byte("two"), firstETag); err != nil {
		t.Fatal(err)
	}
	// The first ETag is stale now: the file changed under it.
	if _, err := store.PutFileIfMatch(ctx, "delta-log/1/a", []byte("three"), firstETag); !errors.Is(err, utils.ErrFilePreconditionFailed) {
		t.Fatalf("a stale ETag must fail the precondition, got %v", err)
	}
	content, etag, err := store.GetFile("delta-log/1/a")
	if err != nil || string(content) != "two" || etag != contentETag([]byte("two")) {
		t.Fatalf("got %q %q %v", content, etag, err)
	}
	if _, err := store.PutFileIfMatch(ctx, "../outside", []byte("x"), ""); err == nil {
		t.Fatal("a key leaving the root must be refused")
	}
}

func TestLocalDiskConcurrentAppendsAllLand(t *testing.T) {
	store, err := NewLocalDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	const appenders = 64
	waitGroup := sync.WaitGroup{}
	for range appenders {
		waitGroup.Go(func() {
			if err := store.AppendBytesToFile(context.Background(), "delta-log/2/log", []byte("0123456789")); err != nil {
				t.Error(err)
			}
		})
	}
	waitGroup.Wait()
	content, _, err := store.GetFile("delta-log/2/log")
	if err != nil || len(content) != appenders*10 || strings.Trim(string(content), "0123456789") != "" {
		t.Fatalf("expected %d bytes of whole appends, got %d (%v)", appenders*10, len(content), err)
	}
}

func TestLocalDiskListsAndDeletesByPrefix(t *testing.T) {
	store, err := NewLocalDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, key := range []string{"ids-cache/5/a", "ids-cache/5/b", "ids-cache/6/a", "delta-log/5/a"} {
		if _, err := store.PutFileIfMatch(ctx, key, []byte(key), ""); err != nil {
			t.Fatal(err)
		}
	}
	keys, err := store.ListFiles("ids-cache/5/")
	slices.Sort(keys)
	if err != nil || !slices.Equal(keys, []string{"ids-cache/5/a", "ids-cache/5/b"}) {
		t.Fatalf("got %v %v", keys, err)
	}
	if keys, _ := store.ListFiles("ids-cache/"); len(keys) != 3 {
		t.Fatalf("a folder prefix must list its subfolders, got %v", keys)
	}
	if keys, _ := store.ListFiles("nothing/here/"); len(keys) != 0 {
		t.Fatalf("a missing folder lists nothing, got %v", keys)
	}
	if err := store.DeleteFiles(ctx, "ids-cache/5/a", "ids-cache/missing"); err != nil {
		t.Fatal(err)
	}
	if keys, _ := store.ListFiles("ids-cache/5/"); !slices.Equal(keys, []string{"ids-cache/5/b"}) {
		t.Fatalf("got %v", keys)
	}
}
