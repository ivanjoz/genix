// Package bench prepares the local stack the backend benchmarks run against: a ScyllaDB container,
// a fareward instance and a seeded keyspace, all named by bench.config.toml. It holds no benchmark
// itself — the per-stage ones live in this folder's _test.go files and the router one in the main
// package (router_bench_test.go), which is the only place mainHandler can be called from.
package bench

import (
	"app/core"
	"app/db"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	fareward "github.com/ivanjoz/fareward/go"
	orm "github.com/ivanjoz/genix-orm/db"
	"github.com/ivanjoz/genix-orm/utils"
)

// EnabledEnv gates every benchmark: without a running stack they would fail on the first query,
// so a plain `go test ./...` skips them instead.
const EnabledEnv = "GENIX_BENCH"

var setupOnce sync.Once

// NoSnapshotsEnv turns the VPS snapshot cache off, to measure first syncs read from ScyllaDB.
const NoSnapshotsEnv = "GENIX_BENCH_NO_SNAPSHOTS"

// Setup wires the process the way main() does for the VPS server, minus the two S3-backed pieces.
//
// The file store is the VPS one — the local snapshot cache — over an in-memory store instead of
// S3: on the Scylla driver only the snapshots use it. The text search backend is replaced by a no-op
// because genix-s3 writes its index to S3 on every product write — only the seeder writes, and its
// search index is not what is being measured.
func Setup() {
	setupOnce.Do(func() {
		if os.Getenv("GENIX_CONFIG_FILE") == "" {
			os.Setenv("GENIX_CONFIG_FILE", benchConfigPath())
		}
		core.PopulateVariables()

		fareward.SetLogger(core.Log)
		if err := core.ConfigureFareward(core.Env.FAREWARD_ADDRESS, core.Env.INTERNAL_APIKEY); err != nil {
			panic("bench: invalid fareward configuration: " + err.Error())
		}
		db.SetTextSearchBackend(noopTextSearch{})
		if os.Getenv(NoSnapshotsEnv) != "" {
			core.Env.FILE_STORE_VPS_SNAPSHOT_CACHE_MB = 0
		}
		core.ConfigureVPSSnapshotCache(utils.NewMemoryFileStore())

		db.Connect(db.ConnectParams{
			Keyspace:    core.Env.DB_NAME,
			FDBSocket:   core.Env.FDB_SOCKET,
			YDBEndpoint: core.Env.YDB_ENDPOINT,
			YDBDatabase: core.Env.YDB_DATABASE,
			OceanBase: db.OceanBaseParams{
				Host: core.Env.OCEANBASE_HOST, Port: core.Env.OCEANBASE_PORT,
				User: core.Env.OCEANBASE_USER, Password: core.Env.OCEANBASE_PASSWORD,
				Partitions: core.Env.OCEANBASE_PARTITIONS, MaxConns: core.Env.OCEANBASE_MAX_CONNS,
				Engine: core.Env.OCEANBASE_ENGINE,
			},
		})
	})
}

// benchConfigPath resolves bench.config.toml next to this source file, so the benchmarks find it
// whichever package directory `go test` runs them from.
func benchConfigPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "bench.config.toml")
}

type noopTextSearch struct{}

func (noopTextSearch) Apply(context.Context, string, string, []orm.TextDoc, []int32) error {
	return nil
}
func (noopTextSearch) Search(context.Context, string, string, string, int, int) ([]orm.TextHit, error) {
	return nil, nil
}
func (noopTextSearch) Compact(context.Context, string, string) error                 { return nil }
func (noopTextSearch) Rebuild(context.Context, string, string, orm.TextSource) error { return nil }
func (noopTextSearch) Drop(context.Context, string, string) error                    { return nil }
