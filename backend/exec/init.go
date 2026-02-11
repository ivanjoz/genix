package exec

import (
	"app/core"
	"app/db"
	"app/handlers"
	s "app/types"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	// core.Log(recordsImported)

	err = db.Insert(&recordsImported)
	if err != nil {
		panic(err)
	}

	return core.FuncResponse{}
}

func ExportCiudades(args *core.ExecArgs) core.FuncResponse {

	// ciudades de Peru
	ciudades := []s.PaisCiudad{}
	q1 := db.Query(&ciudades)
	err := q1.Select(q1.Nombre, q1.CiudadID, q1.PadreID).
		PaisID.Equals(604).Exec()

	if err != nil {
		panic(err)
	}

	core.Log("ciudades obtenidas::", len(ciudades))

	cwd, _ := os.Getwd()
	parentDir := filepath.Dir(cwd)
	filePath := filepath.Join(parentDir, "frontend", "public", "assets", "peru_ciudades.json")

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return core.FuncResponse{}
	}
	defer file.Close()

	// Encode the struct as JSON and write it to the file
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(ciudades); err != nil {
		fmt.Println("Error encoding JSON:", err)
	}

	return core.FuncResponse{}
}

// fn-homologate
func Homologate(args *core.ExecArgs) core.FuncResponse {
	/*
		// Conexión a la base de datos
		db.MakeScyllaConnection(db.ConnParams{
			Host:     core.Env.DB_HOST,
			Port:     int(core.Env.DB_PORT),
			User:     core.Env.DB_USER,
			Password: core.Env.DB_PASSWORD,
			Keyspace: core.Env.DB_NAME,
		})

		fmt.Println("----------")

		structTypes := []any{}
		for _, cn := range MakeScyllaControllers() {
			structTypes = append(structTypes, cn.StructType)
		}

		db.DeployScylla(0, structTypes...)
	*/

	db.MakeScyllaConnection(db.ConnParams{
		Host:     core.Env.DB_HOST,
		Port:     int(core.Env.DB_PORT),
		User:     core.Env.DB_USER,
		Password: core.Env.DB_PASSWORD,
		Keyspace: core.Env.DB_NAME,
	})

	db.DeployScylla(0, MakeScyllaControllers()...)

	return core.FuncResponse{}
}

// fn-recalc
func RecalcVirtualColumnsValues(args *core.ExecArgs) core.FuncResponse {
	/*
		for _, cn := range MakeScyllaControllers() {
			cn.RecalcVirtualColumns()
		}
	*/
	/*
		db.RecalcVirtualColumns[types.AlmacenProducto]()
	*/

	db.MakeScyllaConnection(db.ConnParams{
		Host:     core.Env.DB_HOST,
		Port:     int(core.Env.DB_PORT),
		User:     core.Env.DB_USER,
		Password: core.Env.DB_PASSWORD,
		Keyspace: core.Env.DB_NAME,
	})

	db.QueryExec("DELETE FROM genix.almacen_producto where empresa_id = 1")
	db.QueryExec("DELETE FROM genix.almacen_movimiento where empresa_id = 1")

	return core.FuncResponse{}
}

func RecalcSequences(partValue any) {

	db.MakeScyllaConnection(db.ConnParams{
		Host:     core.Env.DB_HOST,
		Port:     int(core.Env.DB_PORT),
		User:     core.Env.DB_USER,
		Password: core.Env.DB_PASSWORD,
		Keyspace: core.Env.DB_NAME,
	})

	fmt.Println("Recalculando Counter de Tablas...")
	/* 
	for _, sc := range MakeScyllaControllers() {
		sc.ResetCounter(partValue)
	}
	*/
}
