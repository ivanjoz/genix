package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

const SCYLLA_DATA = "/var/lib/scylla/data"
const BACKUP_MAIN_DIR = "/home/ubuntu"

type EnvStruct struct {
	IS_PRODUCTION   bool
	AWS_PROFILE     string
	AWS_REGION      string
	S3_BUCKET       string
	SCYLLA_DATA     string
	KEYSPACE        string
	BACKUP_MAIN_DIR string
}

var Env EnvStruct

func main() {
	populateVariables()

	fmt.Println(Env)

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

func populateVariables() {

	IS_PRODUCTION := false
	if _, err := os.Stat(SCYLLA_DATA); !os.IsNotExist(err) {
		IS_PRODUCTION = true
	}

	fmt.Println("IS_PRODUCTION =", IS_PRODUCTION)

	wd, _ := os.Getwd()
	dirname := strings.Split(wd, "/")
	if IS_PRODUCTION {
		dirname = append(dirname, "credentials.json")
	} else {
		dirname[len(dirname)-1] = "credentials.json"
	}

	credentialsJson := strings.Join(dirname, "/")
	file, err := os.Open(credentialsJson)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Read the content of the file
	variablesBytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading credentials.json:", err)
		return
	}

	err = json.Unmarshal(variablesBytes, &Env)
	if err != nil {
		fmt.Println("Error parsing credentials.json:", err)
		return
	}

	Env.KEYSPACE = "genix"
	Env.IS_PRODUCTION = IS_PRODUCTION
	// Check if is a Scylla instalation
	if Env.IS_PRODUCTION {
		Env.SCYLLA_DATA = SCYLLA_DATA + "/"
		Env.BACKUP_MAIN_DIR = BACKUP_MAIN_DIR + "/"
	} else {
		Env.SCYLLA_DATA = "/home/ivanjoz/Documents/"
		Env.BACKUP_MAIN_DIR = "/home/ivanjoz/Documents/backup_demo/"
	}
}

func makeAwsConfig() (aws.Config, error) {
	var cfg aws.Config
	var err error

	setConfig := func(lo *config.LoadOptions) error {
		lo.Region = Env.AWS_REGION
		return nil
	}

	accessKeyEnv := os.Getenv("AWS_ACCESS_KEY_ID")
	if len(accessKeyEnv) > 0 {
		cfg, err = config.LoadDefaultConfig(context.TODO(), setConfig)
	} else {
		cfg, err = config.LoadDefaultConfig(
			context.TODO(), config.WithSharedConfigProfile(Env.AWS_PROFILE), setConfig)
	}
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}
