package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func backup() {
	fmt.Println("Starting Backup...")
	const DATA_DIR = SCYLLA_DATA + KEYSPACE
	BACKUP_DIR := fmt.Sprintf("%v/backup-%v", BACKUP_MAIN_DIR, time.Now().Unix())

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

	fmt.Println("OK")
}
