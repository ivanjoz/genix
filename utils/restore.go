package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func restore(backupName string) {

	backupDirectory := BACKUP_MAIN_DIR + backupName

	if _, err := os.Stat(backupDirectory); !os.IsNotExist(err) {
		if err := os.RemoveAll(backupDirectory); err != nil {
			log.Fatal("Error removing previous backup directory:", backupDirectory, "|", err)
		}
	}

	if err := os.Mkdir(backupDirectory, os.ModePerm); err != nil {
		log.Fatal("Error creating backup directory:", backupDirectory, "|", err)
	}

	backupTarFile := backupDirectory + ".tar.zst"

	if _, err := os.Stat(backupDirectory); os.IsNotExist(err) {
		log.Fatal("Backup File Don't Exists:", backupTarFile)
	}

	fmt.Println("Uncompressing backup file...")
	cmdString := fmt.Sprintf("--zstd -xvf %v.tar.zst -C %v", backupDirectory, backupDirectory)

	cmd := exec.Command("tar", strings.Split(cmdString, " ")...)
	if err := cmd.Run(); err != nil {
		log.Fatal("Error uncompressing backup:", backupDirectory, "|", err)
	}

	backupDirEntries, err := os.ReadDir(backupDirectory)
	if err != nil {
		fmt.Println("Error reading Backup Directory:", backupDirectory, "|", err)
		return
	}

	backupToTablePathMap := map[string]string{}

	fmt.Println("Reading backup entries...")

	for _, dir := range backupDirEntries {
		name := dir.Name()
		path := backupDirectory + "/" + name
		backupToTablePathMap[name] = path
		tableName := strings.Split(name, "-")[0]
		if _, ok := backupToTablePathMap[tableName]; !ok {
			backupToTablePathMap[tableName] = path
		}
	}

	fmt.Println(backupToTablePathMap)

	const DATA_DIR = SCYLLA_DATA + KEYSPACE

	dirEntries, err := os.ReadDir(DATA_DIR)
	if err != nil {
		fmt.Println("Error reading DATA_DIR:", DATA_DIR, "|", err)
		return
	}

	for _, tableDirectories := range dirEntries {
		if !tableDirectories.IsDir() {
			continue
		}

		tableDirName := tableDirectories.Name()
		tableBackupPath := ""

		if _, ok := backupToTablePathMap[tableDirName]; ok {
			tableBackupPath = backupToTablePathMap[tableDirName]
		} else {
			tablaDirNameSingle := strings.Split(tableDirName, "-")[0]
			if _, ok := backupToTablePathMap[tablaDirNameSingle]; ok {
				tableBackupPath = backupToTablePathMap[tablaDirNameSingle]
			}
		}

		if len(tableBackupPath) == 0 {
			fmt.Println("Backup not found for this table:", tableDirName)
			continue
		}

		tableDirectory := DATA_DIR + "/" + tableDirName
		tableFiles, err := os.ReadDir(tableDirectory)

		if err != nil {
			fmt.Println("Error reading table directory:", tableDirectory, "|", err)
			return
		}

		fmt.Println("Deleting current files from table:", tableDirName)

		for _, tableFile := range tableFiles {
			if tableFile.IsDir() {
				continue
			}
			e := os.Remove(tableDirectory + "/" + tableFile.Name())
			if e != nil {
				log.Fatal(e)
			}
		}

		fmt.Println("Populating backup files to table:", tableDirName)

		tableBackupFiles, err := os.ReadDir(tableBackupPath)
		if err != nil {
			fmt.Println("Error reading table backup directory:", tableBackupPath, "|", err)
			return
		}

		for _, backupFile := range tableBackupFiles {
			backupFilePath := tableBackupPath + "/" + backupFile.Name()
			bytesRead, err := os.ReadFile(backupFilePath)
			if err != nil {
				log.Fatal("Error reading backup file:", backupFilePath, "|", err)
			}
			tableFileDestPath := tableDirectory + "/" + backupFile.Name()

			if err = os.WriteFile(tableFileDestPath, bytesRead, 0644); err != nil {
				log.Fatal("Error writing backup file:", tableFileDestPath, "|", err)
			}
		}
	}
}
