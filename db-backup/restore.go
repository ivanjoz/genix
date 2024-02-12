package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func restore(backupName string) {

	backupDirectory := Env.BACKUP_MAIN_DIR + backupName
	backupFileName := backupName + ".tar.zst"
	backupTarFile := Env.BACKUP_MAIN_DIR + backupFileName

	if _, err := os.Stat(backupTarFile); !os.IsNotExist(err) {

		cfg, err := makeAwsConfig()
		if err != nil {
			panic(err)
		}

		// Open the file to upload
		file, err := os.Create(backupTarFile)
		if err != nil {
			fmt.Println("Error opening file to download:", err)
			return
		}
		defer file.Close()

		fmt.Println("Downloading backup file.", backupFileName)
		downloader := manager.NewDownloader(s3.NewFromConfig(cfg))

		_, err = downloader.Download(context.TODO(), file, &s3.GetObjectInput{
			Bucket: &Env.S3_BUCKET,
			Key:    aws.String("_backups/" + backupFileName),
		})

		if err != nil {
			fmt.Println("Error downloading backup from s3:", err)
			return
		}
	} else {
		fmt.Println("Backup file already exists.")
	}

	if _, err := os.Stat(backupDirectory); !os.IsNotExist(err) {
		if err := os.RemoveAll(backupDirectory); err != nil {
			fmt.Println("Error cleaning backup directory:", err)
			return
		}
	}

	if err := os.Mkdir(backupDirectory, os.ModePerm); err != nil {
		log.Fatal("Error creating backup directory:", backupDirectory, "|", err)
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

	if Env.IS_PRODUCTION {
		fmt.Println("Stoping Scylla-Server...")
		cmd = exec.Command("sudo", "systemctl", "stop", "scylla-server")
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Replacing backup files...")

	for _, dir := range backupDirEntries {
		name := dir.Name()
		path := backupDirectory + "/" + name
		backupToTablePathMap[name] = path
		tableName := strings.Split(name, "-")[0]
		if _, ok := backupToTablePathMap[tableName]; !ok {
			backupToTablePathMap[tableName] = path
		}
	}

	DATA_DIR := Env.SCYLLA_DATA + Env.KEYSPACE

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

	if Env.IS_PRODUCTION {
		fmt.Println("Updating data owner to scylla...")
		cmd := exec.Command("sudo", "chown", "-R", "scylla", "/var/lib/scylla/data/"+Env.KEYSPACE)
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Starting Scylla-Server...")
	cmd = exec.Command("sudo", "systemctl", "start", "scylla-server")
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
