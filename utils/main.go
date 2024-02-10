package main

import (
	"fmt"
	"os"
)

/*
const SCYLLA_DATA = "/var/lib/scylla/data/"
const KEYSPACE = "genix"
const BACKUP_MAIN_DIR = "/home/ubuntu/"
*/
// /home/ivanjoz/Documents/gerp
const SCYLLA_DATA = "/home/ivanjoz/Documents/"
const KEYSPACE = "gerp"
const BACKUP_MAIN_DIR = "/home/ivanjoz/Documents/backup_demo/"

func main() {
	args := os.Args[1:]
	mode := "b"
	backupName := ""

	for i, e := range args {
		if e == "r" {
			mode = "r"
			if len(args) > (i + 1) {
				backupName = args[i+1]
			} else {
				fmt.Println("Backup name missing in args.")
				return
			}
		}
	}

	if mode == "r" {
		restore(backupName)
	} else {
		backup()
	}
}
