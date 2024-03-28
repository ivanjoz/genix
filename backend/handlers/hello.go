package handlers

import (
	"app/core"
	"app/types"
	"os"
	"strings"

	"app/cbor"
)

func HelloWorld(req *core.HandlerArgs) core.HandlerResponse {

	body1 := `
		Todo está funcionando bien!
		ENVIROMENT = $2
		DB_HOST = $3
		APP_CODE = $4
		EXEC_ARGS = $5
	`
	body1 = strings.Replace(body1, "$2", core.Env.ENVIROMENT, -1)
	body1 = strings.Replace(body1, "$3", core.Env.DB_HOST, -1)
	body1 = strings.Replace(body1, "$4", os.Getenv("APP_CODE"), -1)
	body1 = strings.Replace(body1, "$5", strings.Join(os.Args, " | "), -1)

	return req.MakeResponsePlain(&body1)
}

type DemoStruct4 struct {
	types.TAGS `table:"demo_structs"`
	CompanyID  int32    `cbor:"1,keyasint,omitempty" json:"companyID,omitempty" db:"company_id,pk"`
	ID         int32    `cbor:"2,keyasint,omitempty" json:"id,omitempty" db:"id,pk"`
	Edad       int32    `cbor:"3,keyasint,omitempty" json:"edad,omitempty" db:"edad,zx1,zx2"`
	Nombre     string   `cbor:"4,keyasint,omitempty" json:"nombre,omitempty" db:"nombre,zx1"`
	Palabras   []string `cbor:"5,keyasint,omitempty" json:"palabras,omitempty" db:"palabras"`
	Rangos     []int32  `cbor:"6,keyasint,omitempty" json:"rangos,omitempty" db:"rangos"`
	Smallint   int16    `cbor:"7,keyasint,omitempty" json:"small_int,omitempty" db:"small_int,zx2"`
	Peso       float32  `cbor:"8,keyasint,omitempty" json:"peso,omitempty" db:"peso"`
	Peso64     float64  `cbor:"9,keyasint,omitempty" json:"peso64,omitempty" db:"peso_64"`
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

	bytes1, err := cbor.Marshal(array1)
	if err != nil {
		panic(err)
	}

	//convert bytes1 to base64c
	base64c := core.BytesToBase64(bytes1)
	result := map[string]string{"base64": base64c}

	return req.MakeResponse(&result)
}
