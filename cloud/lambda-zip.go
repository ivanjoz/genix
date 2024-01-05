package main

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
)

func WriteExe(writer *zip.Writer, pathInZip string, data []byte) error {
	if pathInZip != "bootstrap" {
		header := &zip.FileHeader{Name: "bootstrap", Method: zip.Deflate}
		header.SetMode(0755 | os.ModeSymlink)
		link, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := link.Write([]byte(pathInZip)); err != nil {
			return err
		}
	}

	exe, err := writer.CreateHeader(&zip.FileHeader{
		CreatorVersion: 3 << 8,     // indicates Unix
		ExternalAttrs:  0777 << 16, // -rwxrwxrwx file permissions
		Name:           pathInZip,
		Method:         zip.Deflate,
	})
	if err != nil {
		return err
	}

	_, err = exe.Write(data)
	return err
}

func CompressExeAndArgs(filePath string) (string, error) {
	outZipPath := filePath + ".zip"
	zipFile, err := os.Create(outZipPath)
	if err != nil {
		return "", err
	}

	defer func() {
		closeErr := zipFile.Close()
		if closeErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to close zip file: %v\n", closeErr)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	err = WriteExe(zipWriter, filepath.Base(filePath), data)
	if err != nil {
		return "", err
	}

	return outZipPath, err
}
