package main

import "os"

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
	mode := "r"
	backupName := ""
	if len(args) > 0 {
		mode = args[0]
	}
	if len(args) > 1 {
		mode = args[1]
	}

	if mode == "r" {
		restore(backupName)
	} else {
		backup()
	}
}
