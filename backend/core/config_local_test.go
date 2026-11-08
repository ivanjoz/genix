package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestLocalConfigOverrideMergesTablesAndReplacesArrays(t *testing.T) {
	configDirectory := t.TempDir()
	baseConfigPath := filepath.Join(configDirectory, "config.toml")
	baseConfigBytes := []byte(`app_name = "genix"
[server]
port = 14010
domain = "api.example.com"
[file_store]
bucket = "orm-bucket"
vps_dir = "/var/lib/genix/files"
[[models]]
id = "base-a"
[[models]]
id = "base-b"
`)
	localConfigBytes := []byte(`[file_store]
vps_snapshot_cache_mb = 512
[[models]]
id = "local-only"
`)

	// Without the local file the base bytes come back untouched.
	if unchangedBytes := applyLocalConfigOverride(baseConfigBytes, baseConfigPath); string(unchangedBytes) != string(baseConfigBytes) {
		t.Fatalf("no local file must leave the config untouched, got:\n%s", unchangedBytes)
	}

	if err := os.WriteFile(filepath.Join(configDirectory, LocalConfigFileName), localConfigBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	mergedConfig := fileConfig{}
	if err := toml.Unmarshal(applyLocalConfigOverride(baseConfigBytes, baseConfigPath), &mergedConfig); err != nil {
		t.Fatal(err)
	}

	if mergedConfig.AppName != "genix" || mergedConfig.Server.Port != 14010 {
		t.Fatalf("keys absent from the override must keep the base values, got %+v", mergedConfig.Server)
	}
	fileStore := mergedConfig.FileStore
	if fileStore.Bucket != "orm-bucket" || fileStore.VpsDir != "/var/lib/genix/files" ||
		fileStore.VpsSnapshotCacheMB == nil || *fileStore.VpsSnapshotCacheMB != 512 {
		t.Fatalf("[file_store] must merge key by key, got %+v", mergedConfig.FileStore)
	}
	if len(mergedConfig.Models) != 1 || mergedConfig.Models[0].ID != "local-only" {
		t.Fatalf("an override [[models]] must replace the whole base list, got %+v", mergedConfig.Models)
	}
}
