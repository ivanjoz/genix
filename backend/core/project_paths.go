package core

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectTmpDir returns the repository-local tmp directory during local
// development, so debug artifacts stay beside the project instead of /tmp.
func ProjectTmpDir() string {
	if wd, err := os.Getwd(); err == nil {
		if projectRoot := findProjectRoot(wd); projectRoot != "" {
			return filepath.Join(projectRoot, "tmp")
		}
	}
	if Env != nil && strings.TrimSpace(Env.TMP_DIR) != "" {
		return strings.TrimRight(Env.TMP_DIR, string(os.PathSeparator))
	}
	return filepath.Join(os.TempDir(), "genix")
}

func findProjectRoot(startDir string) string {
	for dir := startDir; ; dir = filepath.Dir(dir) {
		if pathExists(filepath.Join(dir, "AGENTS.md")) && pathExists(filepath.Join(dir, "backend")) {
			return dir
		}
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			return ""
		}
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
