//go:build js

package cloud

// In the browser the sealed company configuration goes to the installed object store (IndexedDB).

func storeCompanyConfigBlob(objectName string, sealed []byte) error {
	return SaveFile(SaveFileArgs{
		Name: objectName, Path: companyConfigPath, FileContent: sealed,
		ContentType: "application/octet-stream", CacheControl: "no-store",
	})
}

func loadCompanyConfigBlob(objectName string) ([]byte, error) {
	return GetFile(SaveFileArgs{Name: objectName, Path: companyConfigPath})
}
