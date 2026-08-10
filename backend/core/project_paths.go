package core

import (
	"os"
	"path/filepath"
	"strings"
	"errors"
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

// FindProjectRoot locates the monorepo root. It lives in core because three packages need it and
// they cannot share it any other way: `cloud` needs it to run the Node renderer from disk, and it
// cannot import `exec` — `exec` already imports `cloud`.
//
// GENIX_REPOSITORY_ROOT wins when set. On a VPS the service runs from a systemd unit whose working
// directory is not the clone, so walking up from there would never find the markers; main.go prints
// this variable at boot precisely because deployments depend on it.
func FindProjectRoot() (string, error) {
	if repositoryRoot := strings.TrimSpace(os.Getenv("GENIX_REPOSITORY_ROOT")); repositoryRoot != "" {
		if isProjectRoot(repositoryRoot) {
			return repositoryRoot, nil
		}
		return "", errors.New("GENIX_REPOSITORY_ROOT no apunta a la raíz del proyecto Genix: " + repositoryRoot)
	}

	currentDirectory, workingDirectoryError := os.Getwd()
	if workingDirectoryError != nil {
		return "", workingDirectoryError
	}

	for {
		if isProjectRoot(currentDirectory) {
			return currentDirectory, nil
		}

		parentDirectory := filepath.Dir(currentDirectory)
		if parentDirectory == currentDirectory {
			return "", errors.New("no se encontró la raíz del proyecto Genix")
		}
		currentDirectory = parentDirectory
	}
}

// isProjectRoot checks two markers of the monorepo, not one: `backend` runs from its own
// subdirectory and either marker on its own is far too common a filename.
func isProjectRoot(directory string) bool {
	return isRegularFile(filepath.Join(directory, "deploy.sh")) &&
		isRegularFile(filepath.Join(directory, "AGENTS.md"))
}

func isRegularFile(filePath string) bool {
	fileInfo, fileError := os.Stat(filePath)
	return fileError == nil && !fileInfo.IsDir()
}
