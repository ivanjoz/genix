package exec

import (
	"app/core"
	"app/db"
	"slices"
	"strconv"
	"strings"
)

// fn-reload-records reads a company's partition of the named tables and writes it back through the
// ORM insert, which recomputes every derived value. Run it after a deploy adds a derived column to a
// table whose rows are kept across a reseed (users, profiles): the new delta-index column of the old
// rows is empty until they are rewritten, so a Delta() read would never return them.
//
//	go run . fn-reload-records --tables=users,profiles,company_secrets
//	go run . fn-reload-records --tables=users --company=2
func ReloadRecords(args *core.ExecArgs) core.FuncResponse {
	companyID, tableNames := int32(1), []string{}
	for _, argument := range strings.Fields(args.Message) {
		name, rawValue, _ := strings.Cut(strings.TrimPrefix(argument, "--"), "=")
		value, err := strconv.Atoi(rawValue)
		switch {
		case name == "company" && err == nil && value > 0:
			companyID = int32(value)
		case name == "tables" && rawValue != "":
			tableNames = strings.Split(rawValue, ",")
		default:
			return args.MakeErr("Argumento desconocido o inválido:", argument, "(--tables=a,b, --company=N)")
		}
	}
	if len(tableNames) == 0 {
		return args.MakeErr("Debe indicar las tablas: --tables=a,b")
	}
	db.Connect(makeConnParams())

	reloadedTables := []string{}
	for _, controller := range MakeControllers() {
		if !slices.Contains(tableNames, controller.GetTableName()) {
			continue
		}
		if err := controller.ReloadRecords(companyID); err != nil {
			return args.MakeErr("Error al reescribir", controller.GetTableName(), err)
		}
		reloadedTables = append(reloadedTables, controller.GetTableName())
		core.Log("fn-reload-records: rewrote", controller.GetTableName(), "of company", companyID)
	}
	if len(reloadedTables) != len(tableNames) {
		return args.MakeErr("Tablas no encontradas. Reescritas:", reloadedTables, "pedidas:", tableNames)
	}
	return core.FuncResponse{}
}
