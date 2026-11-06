package core

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// LocalConfigFileName is the per-host override that sits next to config.toml. config.toml is the
// shared file pushed to every host as-is; config.local.toml holds what only this host differs in
// (a VPS's own ports, flags, …) and never leaves it. fareward (config.rs) and the configure scripts
// (scripts/configure/toml_config.py) apply the same file with the same rule.
const LocalConfigFileName = "config.local.toml"

// applyLocalConfigOverride returns baseConfigBytes with config.local.toml — when it exists next to
// baseConfigPath — merged over it by mergeConfigOverride. Without the file the bytes are returned
// untouched. A file that exists but does not parse stops the boot: running on the shared values
// while the host's own were silently skipped is the failure this file exists to prevent.
func applyLocalConfigOverride(baseConfigBytes []byte, baseConfigPath string) []byte {
	localConfigPath := filepath.Join(filepath.Dir(baseConfigPath), LocalConfigFileName)
	localConfigBytes, err := os.ReadFile(localConfigPath)
	if errors.Is(err, fs.ErrNotExist) {
		return baseConfigBytes
	}
	if err != nil {
		panic(fmt.Sprintf("no se pudo leer %s: %v", localConfigPath, err))
	}

	baseConfig := map[string]any{}
	if err := toml.Unmarshal(baseConfigBytes, &baseConfig); err != nil {
		panic(fmt.Sprintf("no se pudo parsear %s como TOML: %v", baseConfigPath, err))
	}
	localConfig := map[string]any{}
	if err := toml.Unmarshal(localConfigBytes, &localConfig); err != nil {
		panic(fmt.Sprintf("no se pudo parsear %s como TOML: %v", localConfigPath, err))
	}

	mergedConfigBytes, err := toml.Marshal(mergeConfigOverride(baseConfig, localConfig))
	if err != nil {
		panic(fmt.Sprintf("no se pudo combinar %s con %s: %v", localConfigPath, baseConfigPath, err))
	}
	log.Printf("[core.config] local_override_applied path=%s", localConfigPath)
	return mergedConfigBytes
}

// mergeConfigOverride lays overrideConfig over baseConfig. A table present in both merges key by
// key, recursively; any other value — scalar, array, or array of tables ([[models]], [[servers]]) —
// replaces the base value whole, so an override's [[models]] is the complete list. baseConfig is
// modified and returned.
func mergeConfigOverride(baseConfig, overrideConfig map[string]any) map[string]any {
	for key, overrideValue := range overrideConfig {
		overrideTable, overrideIsTable := overrideValue.(map[string]any)
		baseTable, baseIsTable := baseConfig[key].(map[string]any)
		if overrideIsTable && baseIsTable {
			baseConfig[key] = mergeConfigOverride(baseTable, overrideTable)
			continue
		}
		baseConfig[key] = overrideValue
	}
	return baseConfig
}
