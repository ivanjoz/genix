package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func backup() {
	fmt.Println("Starting Backup...")
	const DATA_DIR = SCYLLA_DATA + KEYSPACE
	BACKUP_DIR := fmt.Sprintf("%vbackup-%v", BACKUP_MAIN_DIR, time.Now().Unix())

	if err := os.Mkdir(BACKUP_DIR, os.ModePerm); err != nil {
		log.Fatal("Error creating backup directory:", BACKUP_DIR, "|", err)
	}

	// Open the directory
	dirEntries, err := os.ReadDir(DATA_DIR)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Iterate over directory entries
	for _, tableDirectories := range dirEntries {
		// Check if it's a directory
		if !tableDirectories.IsDir() {
			continue
		}
		tableBackupPath := BACKUP_DIR + "/" + tableDirectories.Name()

		if err := os.Mkdir(tableBackupPath, os.ModePerm); err != nil {
			log.Fatal("Error al crear:", tableBackupPath, "|", err)
		}

		tableDirectory := DATA_DIR + "/" + tableDirectories.Name()
		tableFiles, err := os.ReadDir(tableDirectory)

		if err != nil {
			fmt.Println("Error reading table directory:", tableDirectory, "|", err)
			return
		}

		fmt.Println("Backup Table:", tableDirectories.Name(), " | Files:", len(tableFiles))

		for _, tableFile := range tableFiles {
			if tableFile.IsDir() {
				continue
			}
			tableFilePath := tableDirectory + "/" + tableFile.Name()
			bytesRead, err := os.ReadFile(tableFilePath)
			if err != nil {
				log.Fatal("Error reading file:", tableFilePath, "|", err)
			}
			tableFileDestPath := tableBackupPath + "/" + tableFile.Name()

			err = os.WriteFile(tableFileDestPath, bytesRead, 0644)

			if err != nil {
				log.Fatal("Error writing file:", tableFileDestPath, "|", err)
			}
		}
	}

	// Create the tar.gzip
	fmt.Println("Creating compressed tar...")
	cmdString := fmt.Sprintf("--zstd -cf %v.tar.zst -C %v .", BACKUP_DIR, BACKUP_DIR)

	cmd := exec.Command("tar", strings.Split(cmdString, " ")...)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	// Delete the backup directory
	fmt.Println("Deleting backup directory...")
	if err := os.RemoveAll(BACKUP_DIR); err != nil {
		log.Fatal(err)
	}

	uploadFile(BACKUP_DIR + ".tar.zst")
	fmt.Println("Finished!")
}

func uploadFile(filePath string) {
	cfg, err := makeAwsConfig()
	if err != nil {
		panic(err)
	}

	uploader := manager.NewUploader(s3.NewFromConfig(cfg), func(u *manager.Uploader) {
		// Define a strategy that will buffer 25 MiB in memory
		u.BufferProvider = manager.NewBufferedReadSeekerWriteToPool(25 * 1024 * 1024)
	})

	fmt.Println("Preparing uploading file...:", filePath)
	// Open the file to upload
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file to send:", err)
		return
	}
	defer file.Close()

	fileNameSlice := strings.Split(filePath, "/")
	fileName := fileNameSlice[len(fileNameSlice)-1]

	fmt.Println("Uploading file...:", filePath)

	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: &Env.S3_BUCKET,
		Key:    aws.String("_backups/" + fileName),
		Body:   file,
	})

	if err != nil {
		panic(err)
	}
}
