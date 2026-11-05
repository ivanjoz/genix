package exec

import (
	"app/cloud"
	"app/core"
	"app/core/filestore"
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ivanjoz/genix-orm/scylla"
)

// fn-wipe-company-data empties the ERP data of one company so the sample generators can fill it
// again (fn-generate-demo-data). It keeps the company, its users, profiles, configuration, secrets,
// SaaS metering and webpage, and every table not partitioned by company (cities, legal parameters,
// image assets, logs).
//
// Dry run by default: it prints what it would delete. Only --apply deletes. Per wiped table:
//   - the company's partition of the base table (Scylla drops its materialized views itself);
//   - its delta-moves rows and cache-group marks, its by-IDs slots and its text index;
//   - its S3 delta and cache-group snapshots; its DataFrames are rebuilt empty;
//   - its autoincrement counters, back to 0, and the company's invoice correlativos (cpe_<id>_*).
//
// Then it seeds the site, warehouse 1 and cash register 1 fn-init would, and republishes the
// company config blob. Browsers keep their delta caches: users must clear site data.
//
//	go run . fn-wipe-company-data              # dry run, company 1
//	go run . fn-wipe-company-data --apply
//	go run . fn-wipe-company-data --only=hr_    # only the tables whose name starts with hr_

// wipeKeptTables are company-partitioned tables that are not ERP data.
var wipeKeptTables = []string{
	"users", "profiles", "parameters", "system_parameters", "company_secrets",
	"credit_usage_company", "credit_usage_user", "credit_history", "usage_log",
	"webpages", "ecommerce_page_content", "gallery_images",
}

type wipeTable struct {
	name    string
	id      int16
	column  string // the partition column
	counter *regexp.Regexp
}

// WipeCompanyData is the entry point behind `fn-wipe-company-data`.
func WipeCompanyData(args *core.ExecArgs) core.FuncResponse {
	companyID, apply, onlyPrefix := int32(1), false, ""
	for _, argument := range strings.Fields(args.Message) {
		name, rawValue, _ := strings.Cut(strings.TrimPrefix(argument, "--"), "=")
		value, err := strconv.Atoi(rawValue)
		switch {
		case name == "apply":
			apply = true
		case name == "company" && err == nil && value > 0:
			companyID = int32(value)
		case name == "only" && rawValue != "":
			onlyPrefix = rawValue
		default:
			return args.MakeErr("Argumento desconocido o inválido:", argument, "(--company=N, --only=prefijo, --apply)")
		}
	}
	keyspace := core.Env.DB_NAME
	session := scylla.MakeScyllaConnection(makeConnParams())

	controllers := MakeScyllaControllers()
	tables, kept := []wipeTable{}, []string{}
	for _, controller := range controllers {
		schema := controller.DescribeTable()
		isCompanyPartition := schema.Partition == "company_id" || schema.Partition == "empresa_id"
		if !isCompanyPartition || slices.Contains(wipeKeptTables, schema.Name) || !strings.HasPrefix(schema.Name, onlyPrefix) {
			kept = append(kept, schema.Name)
			continue
		}
		tables = append(tables, wipeTable{name: schema.Name, id: schema.ID, column: schema.Partition,
			// Counters are named x<partition>_<table>_<autoincrement part value>.
			counter: regexp.MustCompile(fmt.Sprintf(`^x%d_%s_-?\d+$`, companyID, regexp.QuoteMeta(schema.Name)))})
	}

	// The ORM's side tables that exist in this keyspace.
	existingTables := map[string]bool{}
	tableIter := session.Query(`SELECT table_name FROM system_schema.tables WHERE keyspace_name = ?`, keyspace).Iter()
	for tableName := ""; tableIter.Scan(&tableName); {
		existingTables[tableName] = true
	}
	if err := tableIter.Close(); err != nil {
		return args.MakeErr("No se pudieron listar las tablas:", err)
	}

	counters := []string{}
	counterIter := session.Query(fmt.Sprintf(`SELECT name FROM %s.sequences`, keyspace)).Iter()
	for counterName := ""; counterIter.Scan(&counterName); {
		isInvoiceCorrelativo := onlyPrefix == "" && strings.HasPrefix(counterName, fmt.Sprintf("cpe_%d_", companyID))
		if isInvoiceCorrelativo || slices.ContainsFunc(tables, func(table wipeTable) bool { return table.counter.MatchString(counterName) }) {
			counters = append(counters, counterName)
		}
	}
	if err := counterIter.Close(); err != nil {
		return args.MakeErr("No se pudieron listar los contadores:", err)
	}

	fileStore := filestore.New(s3.NewFromConfig(core.GetAwsConfig()), core.Env.FILE_STORE_BUCKET, core.Env.FILE_STORE_ROOT)
	snapshotFiles := []string{}
	for _, table := range tables {
		for _, prefix := range []string{"delta-snapshot", "cache-group"} {
			files, err := fileStore.ListFiles(fmt.Sprintf("%s/%s/%d/", prefix, table.name, companyID))
			if err != nil {
				return args.MakeErr("No se pudieron listar los snapshots:", err)
			}
			snapshotFiles = append(snapshotFiles, files...)
		}
	}

	tableNames := []string{}
	for _, table := range tables {
		tableNames = append(tableNames, table.name)
	}
	summary := fmt.Sprintf("Empresa %d: %d tablas a vaciar, %d se conservan, %d contadores a 0, %d snapshots S3.\nVaciar: %s\nConservar: %s\nContadores: %s",
		companyID, len(tables), len(kept), len(counters), len(snapshotFiles),
		strings.Join(tableNames, ", "), strings.Join(kept, ", "), strings.Join(counters, ", "))
	if !apply {
		return core.FuncResponse{Message: "SIMULACIÓN (nada se borró; use --apply)\n" + summary}
	}

	for _, table := range tables {
		statements := []string{fmt.Sprintf(`DELETE FROM %s.%s WHERE %s = %d`, keyspace, table.name, table.column, companyID)}
		if existingTables[table.name+"__index_updated"] {
			statements = append(statements, fmt.Sprintf(`DELETE FROM %s.%s__index_updated WHERE partition_id = %d`, keyspace, table.name, companyID))
		}
		statements = append(statements, fmt.Sprintf(`DELETE FROM %s.cache_by_ids_slots WHERE table_id = %d AND partition_id = %d`,
			keyspace, table.id, companyID))
		// Delta moves are partitioned by (partition_id, index_tag, pinned_prefix): list the
		// company's partitions, then delete each one.
		if deltaTable := table.name + "__delta_moves"; existingTables[deltaTable] {
			iter := session.Query(fmt.Sprintf(`SELECT DISTINCT partition_id, index_tag, pinned_prefix FROM %s.%s`, keyspace, deltaTable)).Iter()
			// Typed as deltaMovesCreateScript declares them: gocql cannot scan a bigint into an any.
			var partitionID, pinnedPrefix int64
			var indexTag string
			for iter.Scan(&partitionID, &indexTag, &pinnedPrefix) {
				if partitionID == int64(companyID) {
					statements = append(statements, fmt.Sprintf(`DELETE FROM %s.%s WHERE partition_id = %d AND index_tag = %s AND pinned_prefix = %s`,
						keyspace, deltaTable, companyID, cqlLiteral(indexTag), cqlLiteral(pinnedPrefix)))
				}
			}
			if err := iter.Close(); err != nil {
				return args.MakeErr("No se pudo leer", deltaTable, ":", err)
			}
		}
		for _, statement := range statements {
			if err := scylla.QueryExec(statement); err != nil {
				return args.MakeErr("Falló:", statement, ":", err)
			}
		}
	}
	for _, controller := range controllers {
		if slices.Contains(tableNames, controller.GetTableName()) {
			if err := controller.DropTextSearch(companyID); err != nil {
				return args.MakeErr("No se pudo vaciar el índice de texto de", controller.GetTableName(), ":", err)
			}
			if err := controller.RebuildDataFramesPartition(companyID); err != nil {
				return args.MakeErr("No se pudieron reconstruir los DataFrames de", controller.GetTableName(), ":", err)
			}
		}
	}
	if len(snapshotFiles) > 0 {
		if err := fileStore.DeleteFiles(context.Background(), snapshotFiles...); err != nil {
			return args.MakeErr("No se pudieron borrar los snapshots:", err)
		}
	}
	for _, counterName := range counters {
		if _, err := scylla.SetCounterValue(keyspace, counterName, 0); err != nil {
			return args.MakeErr("No se pudo reiniciar el contador", counterName, ":", err)
		}
	}

	// A partial wipe (--only) left the sites, warehouses and cash registers in place.
	if onlyPrefix != "" {
		return core.FuncResponse{Message: "DATOS BORRADOS\n" + summary}
	}
	seedCompanyOperatingRecords([]int32{companyID}, core.SUnixTime())
	if _, err := cloud.CompactAndStoreCompanyConfig(companyID); err != nil {
		return args.MakeErr("No se pudo publicar la configuración de la empresa:", err)
	}
	return core.FuncResponse{Message: "DATOS BORRADOS\n" + summary}
}

// cqlLiteral writes a delta-moves key value back into a statement.
func cqlLiteral(value any) string {
	switch typed := value.(type) {
	case string:
		return "'" + strings.ReplaceAll(typed, "'", "''") + "'"
	case []byte:
		return fmt.Sprintf("0x%x", typed)
	default:
		return fmt.Sprint(typed)
	}
}
