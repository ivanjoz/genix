package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type EnvStruct struct {
	IS_PROD        bool
	IS_LOCAL       bool
	STACK_NAME     string
	APP_CODE       string
	ENVIROMENT     string
	DB_NAME        string
	DB_USER        string
	DB_HOST        string
	DB_PASSWORD    string
	TMP_DIR        string
	REQ_IP         string
	REQ_ID         string
	REQ_ENCODING   string
	REQ_PARAMS     string
	REQ_USER_AGENT string
	HANDLER_PARH   string
	REQ_PATH       string
	AWS_PROFILE    string
	AWS_REGION     string
	S3_BUCKET      string
	DYNAMO_TABLE   string
	REQ_LAMBDA_ID  string
	LOGS_FULL      bool
	LOGS_ONLY_SAVE bool
	DB_DISABLE_SSL bool
	DB_PORT        int32
	USUARIO_ID     int32
	ADMIN_PASSWORD string
	SECRET_PHRASE  string
}

var Env *EnvStruct

func PopulateVariables() {
	fmt.Println("Populando Variables:: ")

	APP_CODE := os.Getenv("APP_CODE")
	IS_LOCAL := len(APP_CODE) == 0

	wd, _ := os.Getwd()

	var variablesBytes []byte

	if IS_LOCAL {
		APP_CODE = "gerp_x"
		dirname := strings.Split(wd, "/")
		dirname[len(dirname)-1] = "credentials.json"
		credentialsJson := strings.Join(dirname, "/")
		file, err := os.Open(credentialsJson)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()

		// Read the content of the file
		variablesBytes, err = io.ReadAll(file)
		if err != nil {
			fmt.Println("Error reading credentials.json:", err)
			return
		}
	} else {
		configJsonBase64 := os.Getenv("CONFIG")
		if len(configJsonBase64) == 0 {
			panic("No se encontraron las variables de entorno.")
		}
		configJsonBase64 = MakeB64UrlDecode(configJsonBase64)
		baseBytes := Base64ToBytes(configJsonBase64)
		variablesBytes = []byte(DecompressZstd(&baseBytes))
	}

	err := json.Unmarshal(variablesBytes, &Env)
	if err != nil {
		fmt.Println("Error parsing credentials.json:", err)
		return
	}

	fmt.Println("Credenciales .json Parseadas:: ")

	if len(Env.DYNAMO_TABLE) == 0 {
		Env.DYNAMO_TABLE = Env.STACK_NAME + "-db"
	}

	Env.APP_CODE = APP_CODE
	Env.IS_LOCAL = IS_LOCAL
	Env.TMP_DIR = If(Env.IS_LOCAL, wd+"/tmp/", "/tmp/")

	Env.IS_PROD = strings.Contains(APP_CODE, "_prd")
	for _, value := range os.Args {
		if value == "prod" {
			Env.IS_PROD = true
			break
		}
	}
}

var REQ_PATHS = []string{}

func makeAwsConfig(mode uint8) (aws_sdk.Config, error) {
	var cfg aws_sdk.Config
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

func GetAwsConfig(acount uint8) aws_sdk.Config {
	cfg, err := makeAwsConfig(acount)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	return cfg
}

type IUsuario struct {
	EmpresaID    int32   `json:"c"`
	ID           int32   `json:"d"`
	PerfilesIDs  []int32 `json:"p"`
	ModulesIDs   []int32 `json:"m"`
	BaseModuleID int32   `json:"b"`
	RolesIDs     []int32 `json:"r"`
	Device       string  `json:"v"`
	IP           string  `json:"i"`
	Usuario      string  `json:"u"`
	AccesosIDs   []int32 `json:"a"`
	Created      int64   `json:"e"`
	ZonaHoraria  int16   `json:"h,omitempty"`
	DeviceID     int     `json:"q,omitempty"`
	Error        string
}

var Usuario IUsuario

func CheckUser(req *HandlerArgs, access int) IUsuario {
	userToken := req.Headers["authorization"]
	if len(userToken) < 8 {
		userToken = req.Headers["Authorization"]
	}

	usuario := IUsuario{}

	if len(userToken) < 8 || !strings.Contains(userToken, "Bearer ") {
		usuario.Error = "No se suministró un Token de usuario"
		return usuario
	}

	encryptedInfo := strings.Split(userToken, " ")[1]
	if len(encryptedInfo) < 8 {
		usuario.Error = "No se encontró la informaación encriptada del usuario"
		return usuario
	}

	encryptedInfoBytes := Base64ToBytes(MakeB64UrlDecode(encryptedInfo))
	if len(encryptedInfoBytes) < 16 {
		usuario.Error = "El tocken de inicio de sesión es inválido"
		return usuario
	}
	decriptedBytes, err := Decrypt(encryptedInfoBytes)

	if err != nil {
		Log("Error desencriptar:", err)
		usuario.Error = "Hubo un error al desencriptar el Token."
		return usuario
	}

	decriptedBytesJson := DecompressZstd(&decriptedBytes)

	if err := json.Unmarshal([]byte(decriptedBytesJson), &usuario); err != nil {
		Log("Error recuperar info:", err)
		usuario.Error = "Error al recuperar la información del usuario."
	}

	Usuario = usuario
	Env.USUARIO_ID = usuario.ID

	return usuario
}
