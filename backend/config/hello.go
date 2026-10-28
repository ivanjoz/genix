package config

import (
	"app/config/types"
	"app/core"
	"os"
	"strings"

	"github.com/ivanjoz/colbin"
)

func HelloWorld(req *core.HandlerArgs) core.HandlerResponse {

	body1 := `
		GENIX VERSION 0.11
		--------------------------------
		Todo está funcionando bien!
		BUILD_DATE = $1
		ENVIROMENT = $2
		DB_HOST = $3
		APP_CODE = $4
		EXEC_ARGS = $5
	`
	body1 = strings.Replace(body1, "$1", core.BuildDate, -1)
	body1 = strings.Replace(body1, "$2", core.Env.ENVIROMENT, -1)
	body1 = strings.Replace(body1, "$3", core.Env.DB_HOST, -1)
	body1 = strings.Replace(body1, "$4", os.Getenv("APP_CODE"), -1)
	body1 = strings.Replace(body1, "$5", strings.Join(os.Args, " | "), -1)

	bodyBytes := []byte(body1)
	response := req.MakeResponsePlain(&bodyBytes)
	// Keep this health response readable without compression support.
	response.DisableCompression = true
	return response
}

type DemoStruct4 struct {
	types.TAGS `table:"demo_structs"`
	CompanyID  int32    `json:"companyID" db:"company_id,pk" cb:"companyID,1"`
	ID         int32    `json:"id" db:"id,pk" cb:"id,2"`
	Edad       int32    `json:"edad" db:"edad,zx1,zx2" cb:"edad,3"`
	Nombre     string   `json:"nombre" db:"nombre,zx1" cb:"nombre,4"`
	Palabras   []string `json:"palabras" db:"palabras" cb:"palabras,5"`
	Rangos     []int32  `json:"rangos" db:"rangos" cb:"rangos,6"`
	Smallint   int16    `json:"small_int" db:"small_int,zx2" cb:"small_int,7"`
	Peso       float32  `json:"peso" db:"peso" cb:"peso,8"`
	Peso64     float64  `json:"peso64" db:"peso_64" cb:"peso64,9"`
}

func Demo1(req *core.HandlerArgs) core.HandlerResponse {

	array1 := []DemoStruct4{
		{Nombre: "hola123",
			Edad:      123,
			CompanyID: 12,
			Smallint:  1,
		},
		{Nombre: "ho1dasd",
			Edad:      13,
			CompanyID: 1,
			Smallint:  11,
		},
	}

	bytes1, err := colbin.Marshal(array1)
	if err != nil {
		panic(err)
	}

	//convert bytes1 to base64c
	base64c := core.BytesToBase64(bytes1)
	result := map[string]string{"base64": base64c}

	return req.MakeResponse(&result)
}
