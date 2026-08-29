package exec

import (
	business "app/business/types"
	"app/cloud"
	config "app/config/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	finance "app/finance/types"
	security "app/security/types"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/ivanjoz/genix-orm/scylla"
	"github.com/ivanjoz/genix-orm/scylla/text_search"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// configureTextSearchGenixSearch resolves the GenixSearch endpoint
// from credentials (falling back to 127.0.0.1:14446) and pushes it
// into the text_search package. GENIXSEARCH_PASSWORD must be set in
// prod or writes will fail at handshake; we log a warning when it's
// missing.
// reserveSeededAutoincrementIDs moves an autoincrement counter past the IDs written by hand below.
//
// The ORM only reserves counter values for records whose key is still 0 (insert-update.go:278),
// so seeding a row with a literal ID leaves the counter untouched. With the counter at zero, the
// next record created through the normal path draws ID 1 and — because a Scylla INSERT is an
// upsert — silently overwrites the seed. That is exactly how public sign-up replaced companies 1
// and 2 instead of creating new ones.
//
// The reservation is unconditional: re-running fn-init skips a few IDs, which is harmless, and the
// counter can only be read through the ORM by also incrementing it.
func reserveSeededAutoincrementIDs(partitionValue any, tableName string, seededCount int) {
	// Same key the ORM builds for the counter: x{partition}_{table}_{autoincrementPartition}.
	counterName := fmt.Sprintf("x%v_%v_0", partitionValue, tableName)
	if _, err := db.GetAutoincrementID(counterName, seededCount); err != nil {
		panic("No se pudo reservar el rango de IDs sembrados de " + counterName + ". " + err.Error())
	}
	core.Log("Se reservó el rango de IDs sembrados:", counterName, "x", seededCount)
}

func configureTextSearchGenixSearch() {
	host, port := core.ParseGenixSearchURL(core.Env.GENIXSEARCH_URL)
	password := strings.TrimSpace(core.Env.GENIXSEARCH_PASSWORD)
	if password == "" && core.Env.IS_PROD {
		core.Log("text_search: GENIXSEARCH_PASSWORD empty in prod; writes will fail at handshake")
	}
	text_search.Configure(host, port, password)
}

// seedCompanyOperatingRecords gives each seeded company the Site, Warehouse and CashBank it needs
// to be operational, mirroring what business.PostInitialData creates for a company registered
// through the sign-up wizard.
//
// Without them, login reports InitialDataPending and the company can do nothing until those rows
// exist. The wizard covers the companies it creates; the ones fn-init writes by hand were left to
// a separate bootstrap screen, which is the only reason that screen had to exist.
//
// IDs are left at zero so the ORM autoincrements them. That is deliberate and unlike the companies
// and users above: those carry literal IDs and therefore need reserveSeededAutoincrementIDs to
// push the counter past them, while an autoincremented insert advances its own counter.
func seedCompanyOperatingRecords(companyIDs []int32, seedTimestamp int32) {
	cityID := firstSeededCityID()

	for _, companyID := range companyIDs {
		sites := []business.Site{{
			CompanyID: companyID,
			Name:      "Principal",
			Address:   "Principal",
			CityID:    cityID,
			Status:    1,
			Created:   seedTimestamp,
			Updated:   seedTimestamp,
		}}
		if err := db.Insert(&sites); err != nil {
			panic(fmt.Sprintf("Error al crear la sede inicial de la empresa %v. %v", companyID, err))
		}
		siteID := sites[0].ID

		warehouses := []business.Warehouse{{
			CompanyID: companyID,
			SiteID:    siteID,
			Name:      "Central",
			Status:    1,
			Created:   seedTimestamp,
			Updated:   seedTimestamp,
		}}
		if err := db.Insert(&warehouses); err != nil {
			panic(fmt.Sprintf("Error al crear el almacén inicial de la empresa %v. %v", companyID, err))
		}

		cashBanks := []finance.CashBank{{
			CompanyID: companyID,
			SiteID:    siteID,
			Name:      "Caja Principal",
			// Type 1 = "Caja" and CurrencyType 1 = PEN, the same defaults PostInitialData uses.
			Type:         1,
			CurrencyType: 1,
			Status:       1,
			Created:      seedTimestamp,
			Updated:      seedTimestamp,
		}}
		if err := db.Insert(&cashBanks); err != nil {
			panic(fmt.Sprintf("Error al crear la caja inicial de la empresa %v. %v", companyID, err))
		}

		core.Log("Datos iniciales sembrados para la empresa", companyID,
			":: sede", siteID, "almacén", warehouses[0].ID, "caja", cashBanks[0].ID)
	}
}

// firstSeededCityID picks a district for the seeded sites so the row is complete. It is best
// effort: a site with no city is still a usable site, and failing the whole bootstrap over a
// cosmetic field would be worse than leaving it at zero.
func firstSeededCityID() int32 {
	cities := []business.CityLocation{}
	query := db.Query(&cities)
	query.Select(query.ID, query.Hierarchy).CountryID.Equals(604).Limit(1)
	if err := query.Exec(); err != nil {
		core.Log("No se pudo resolver una ciudad para las sedes iniciales:", err)
		return 0
	}
	if len(cities) == 0 {
		return 0
	}
	return cities[0].ID
}

func ConfigInit(args *core.ExecArgs) core.FuncResponse {

	if len(core.Env.ADMIN_PASSWORD) == 0 || len(core.Env.SECRET_PHRASE) == 0 {
		panic("No se especificado el admin_password y el secret_phrase en config.toml")
	}

	// Wire the GenixSearch endpoint before any seed write hits a
	// TextSearchColumn-backed table. The text_search package can't
	// import core (cycle: core -> core/types -> db -> text_search), so
	// the resolved config is pushed in here.
	configureTextSearchGenixSearch()

	// Bootstrap ORM internal tables before any ScyllaDB seed writes depend on them.
	if err := scylla.Init(); err != nil {
		panic("Error al inicializar las tablas internas del ORM. " + err.Error())
	}

	DeployDatabaseSchemas(args)

	if err := cloud.Init[config.Company](); err != nil {
		panic("Error al inicializar la tabla cloud de empresas. " + err.Error())
	}
	if err := cloud.Init[coreTypes.User](); err != nil {
		panic("Error al inicializar la tabla cloud de usuarios. " + err.Error())
	}
	if err := cloud.Init[security.Profile](); err != nil {
		panic("Error al inicializar la tabla cloud de perfiles. " + err.Error())
	}

	seedTimestamp := core.SUnixTime()
	password := core.Env.SECRET_PHRASE + core.Env.ADMIN_PASSWORD
	passwordHash := core.FnvHashString64(password, -1, 20)
	empresas := []config.Company{
		{
			ID:        1,
			Name:      "Principal",
			LegalName: "Principal",
			RUC:       "11000000000",
			Updated:   seedTimestamp,
		},
		{
			ID:        2,
			Name:      "Test",
			LegalName: "Test",
			RUC:       "12000000000",
			Updated:   seedTimestamp,
		},
	}
	usuarios := []coreTypes.User{
		{
			ID:           1,
			CompanyID:    1,
			User:         "admin",
			FirstName:    "admin",
			LastName:     "root",
			PasswordHash: passwordHash,
			Status:       1,
			Created:      seedTimestamp,
			Updated:      seedTimestamp,
		},
		{
			ID:           1,
			CompanyID:    2,
			User:         "system",
			FirstName:    "system",
			LastName:     "demo",
			PasswordHash: passwordHash,
			Status:       1,
			Created:      seedTimestamp,
			Updated:      seedTimestamp,
		},
	}

	// Seed companies in ScyllaDB first so the relational source of truth is ready.
	if err := db.Insert(&empresas); err != nil {
		panic("Error al crear las empresas iniciales en ScyllaDB. " + err.Error())
	}
	if err := cloud.Insert(empresas); err != nil {
		panic("Error al crear las empresas iniciales en cloud. " + err.Error())
	}
	// Companies have no partition, so the counter key is x0_companies_0.
	reserveSeededAutoincrementIDs(0, "companies", len(empresas))
	core.Log("Se crearon/actualizaron las empresas iniciales en ScyllaDB y cloud.")

	// Seed admin/system users in ScyllaDB so delta-cache and ID-based reads stay consistent.
	if err := db.Insert(&usuarios); err != nil {
		panic("Error al crear los usuarios iniciales en ScyllaDB. " + err.Error())
	}
	if err := cloud.Insert(usuarios); err != nil {
		panic("Error al crear los usuarios iniciales en cloud. " + err.Error())
	}
	// Users are partitioned by CompanyID, so each seeded company has its own counter. Without this
	// the first user created from the users page would draw ID 1 and overwrite the admin account.
	for _, seededUser := range usuarios {
		reserveSeededAutoincrementIDs(seededUser.CompanyID, "users", 1)
	}
	core.Log("Se crearon/actualizaron los usuarios iniciales en ScyllaDB y cloud.")

	// Run the city import as part of the base bootstrap to leave the environment operational after init.
	ImportCiudades(args)
	core.Log("Se importaron las ciudades iniciales.")

	// After the cities, because a seeded site references one.
	seededCompanyIDs := make([]int32, 0, len(empresas))
	for _, seededCompany := range empresas {
		seededCompanyIDs = append(seededCompanyIDs, seededCompany.ID)
	}
	seedCompanyOperatingRecords(seededCompanyIDs, seedTimestamp)

	core.Print(usuarios)

	return core.FuncResponse{}
}

func ImportCiudades(args *core.ExecArgs) core.FuncResponse {

	wd, _ := os.Getwd()
	filePath := wd + "/assets/ubigeo_distrito.csv"
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

	recordsMap := map[int32]business.CityLocation{}

	addRecords := func(id, padreID, nombre string, jerarquia int8) {
		cityID, err := strconv.ParseInt(id, 10, 32)
		if err != nil {
			panic("ubigeo inválido: " + id)
		}
		parentID := int64(0)
		if padreID != "" {
			parentID, err = strconv.ParseInt(padreID, 10, 32)
			if err != nil {
				panic("ubigeo padre inválido: " + padreID)
			}
		}
		cityID32 := int32(cityID)
		if _, ok := recordsMap[cityID32]; !ok {
			recordsMap[cityID32] = business.CityLocation{
				ID:        cityID32,
				CountryID: 604,
				ParentID:  int32(parentID),
				Name:      nombre,
				Hierarchy: jerarquia,
				Updated:   core.SUnixTime(),
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
	ciudades := []business.CityLocation{}
	q1 := db.Query(&ciudades)
	err := q1.Select(q1.ID, q1.Name, q1.ParentID, q1.Hierarchy).
		CountryID.Equals(604).Exec()

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
func DeployDatabaseSchemas(args *core.ExecArgs) core.FuncResponse {

	if len(core.Env.DB_NAME) == 0 {
		panic("No se ha especificado el db.name en config.toml")
	}

	scylla.MakeScyllaConnection(makeConnParams())

	if err := scylla.CreateKeyspaceIfNotExists(); err != nil {
		panic(fmt.Sprintf("Error al crear el keyspace '%v': %v", core.Env.DB_NAME, err))
	}

	scylla.DeployScylla(0, MakeScyllaControllers()...)

	return core.FuncResponse{}
}

// fn-recalc
func RecalcVirtualColumnsValues(args *core.ExecArgs) core.FuncResponse {

	scylla.MakeScyllaConnection(makeConnParams())

	scylla.QueryExec("DELETE FROM genix.almacen_producto where empresa_id = 1")
	scylla.QueryExec("DELETE FROM genix.almacen_movimiento where empresa_id = 1")

	return core.FuncResponse{}
}
