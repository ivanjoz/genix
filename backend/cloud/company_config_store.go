//go:build !js

package cloud

import (
	"app/core"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Where the sealed company configuration lives: S3 on Lambda, the VPS's own disk otherwise, beside
// the ORM's files (core.VPSFileStoreDir). A VPS keeps every cache file on the machine that serves
// it; S3 is the Lambda path.

func companyConfigLocalPath(objectName string) string {
	return filepath.Join(core.VPSFileStoreDir(), companyConfigPath, objectName)
}

func storeCompanyConfigBlob(objectName string, sealed []byte) error {
	if core.Env.IS_SERVERLESS {
		return SaveFile(SaveFileArgs{
			Name:        objectName,
			Path:        companyConfigPath,
			FileContent: sealed,
			ContentType: "application/octet-stream",
			// The blob changes whenever the company does, and every reader wants the
			// current one — a cached copy at a CDN edge would serve retired credentials.
			CacheControl: "no-store",
		})
	}
	localPath := companyConfigLocalPath(objectName)
	if err := os.MkdirAll(filepath.Dir(localPath), 0o700); err != nil {
		return err
	}
	// Temporary file + rename: a reader in this or another process (the agent) sees the old blob
	// or the new one, never half of it. Two writers racing both write a complete, current blob.
	temporaryFile, err := os.CreateTemp(filepath.Dir(localPath), objectName+".*.tmp")
	if err != nil {
		return err
	}
	_, writeErr := temporaryFile.Write(sealed)
	closeErr := temporaryFile.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr == nil {
		writeErr = os.Rename(temporaryFile.Name(), localPath)
	}
	if writeErr != nil {
		os.Remove(temporaryFile.Name())
	}
	return writeErr
}

// loadCompanyConfigBlob returns nil, nil when there is no blob yet: the caller rebuilds it.
func loadCompanyConfigBlob(objectName string) ([]byte, error) {
	if core.Env.IS_SERVERLESS {
		return GetFile(SaveFileArgs{Name: objectName, Path: companyConfigPath})
	}
	sealed, err := os.ReadFile(companyConfigLocalPath(objectName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return sealed, err
}
