package exec

import (
	"app/core"
	"app/handlers"
	s "app/types"
	"encoding/csv"
	"os"

	"time"
)

func ConfigInit(args *core.ExecArgs) core.FuncResponse {

	if len(core.Env.ADMIN_PASSWORD) == 0 || len(core.Env.SECRET_PHRASE) == 0 {
		panic("No se especificado el ADMIN_PASSWORD y el SECRET_PHRASE en credentials.json")
	}

	empresaTable := handlers.MakeEmpresaTable()
	empresa := s.Empresa{
		ID:          1,
		Nombre:      "Principal",
		RazonSocial: "Principal",
		RUC:         "11000000000",
		Updated:     time.Now().Unix(),
	}

	err := empresaTable.PutItem(&empresa, 1)
	if err != nil {
		panic("Error al crear el usuario admin. " + err.Error())
	}

	core.Log("Se creo/actualizó la empresa.")
	password := core.Env.SECRET_PHRASE + core.Env.ADMIN_PASSWORD
	passwordHash := core.FnvHashString64(password, -1, 20)

	usuario := s.Usuario{
		ID:           1,
		EmpresaID:    empresa.ID,
		Usuario:      "admin",
		Nombres:      "admin",
		Apellidos:    "root",
		PasswordHash: passwordHash,
		Status:       1,
		Created:      time.Now().Unix(),
		Updated:      time.Now().Unix(),
	}

	usuarioTable := handlers.MakeUsuarioTable(empresa.ID)

	err = usuarioTable.PutItem(&usuario, 1)
	if err != nil {
		panic("Error al crear el usuario admin. " + err.Error())
	}

	core.Log("Se creo/actualizó el usuario.")
	core.Print(usuario)

	return core.FuncResponse{}
}

func ImportCiudades(args *core.ExecArgs) core.FuncResponse {

	wd, _ := os.Getwd()
	filePath := wd + "/docs/ubigeo_distrito.csv"
	core.Log("Leyendo archivo .csv: ", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	core.Print(records[0])
	core.Print(records[1])

	recordsMap := map[string]s.PaisCiudad{}

	addRecords := func(id, padreID, nombre string, jerarquia int8) {
		if _, ok := recordsMap[id]; !ok {
			recordsMap[id] = s.PaisCiudad{
				PaisID:    604,
				CiudadID:  id,
				PadreID:   padreID,
				Nombre:    nombre,
				Jerarquia: jerarquia,
				Updated:   time.Now().Unix(),
			}
		}
	}

	for i, e := range records {
		if i == 0 {
			continue
		}
		id := e[0]
		if len(id) < 5 {
			continue
		}
		departamento := e[2]
		provincia := e[3]
		distrito := e[4]
		if len(id) == 5 {
			id = "0" + id
		}
		addRecords(id, id[:4], distrito, 3)
		addRecords(id[:4]+"00", id[:4], "N/A", 3)
		addRecords(id[:4], id[:2], provincia, 2)
		addRecords(id[:2], "", departamento, 1)
	}

	core.Log("Nº de registros:: ", len(recordsMap))
	recordsImported := core.MapToSliceT(recordsMap)

	err = core.DBInsert(&recordsImported)
	if err != nil {
		panic(err)
	}

	return core.FuncResponse{}
}

func Homologate(args *core.ExecArgs) core.FuncResponse {

	for _, controller := range MakeScyllaControllers() {
		controller.InitTable(2)
	}

	return core.FuncResponse{}
}
