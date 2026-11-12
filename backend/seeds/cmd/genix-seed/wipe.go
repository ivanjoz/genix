package main

import (
	"app/cloud"
	"app/core"
	"app/db"
	"app/exec"
	"app/seeds/runner"
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// `genix-seed wipe` empties the ERP data of one company so `genix-seed run` can fill it again. It
// keeps the company, its users, profiles, configuration, secrets, SaaS metering and webpage, and
// every table not partitioned by company (cities, legal parameters, image assets, logs).
//
// Dry run by default: it prints what it would delete. Only --apply deletes. Per wiped table:
//   - the company's partition of the base table (its indexes go with the rows);
//   - its delta-moves rows and cache-group marks, its by-IDs slots and its text index;
//   - its delta and cache-group snapshots in the file store; its DataFrames are rebuilt empty;
//   - its autoincrement counters, back to 0, and the company's invoice correlativos (cpe_<id>_*).
//
// Then it seeds the site, warehouse 1 and cash register 1 fn-init would, republishes the company
// config blob and drops the company's seed resume file. Browsers keep their delta caches: users
// must clear site data.
//
//	genix-seed wipe              # dry run, company 1
//	genix-seed wipe --apply
//	genix-seed wipe --only=hr_   # only the tables whose name starts with hr_

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

func wipeCompanyData(arguments []string) error {
	companyID, apply, onlyPrefix := int32(1), false, ""
	for _, argument := range arguments {
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
			return core.Err("Argumento desconocido o inválido:", argument, "(--company=N, --only=prefijo, --apply)")
		}
	}

	controllers := exec.MakeControllers()
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

	counters := []string{}
	allCounters, counterListError := db.ListCounters()
	if counterListError != nil {
		return core.Err("No se pudieron listar los contadores:", counterListError)
	}
	for _, counterName := range allCounters {
		isInvoiceCorrelativo := onlyPrefix == "" && strings.HasPrefix(counterName, fmt.Sprintf("cpe_%d_", companyID))
		if isInvoiceCorrelativo || slices.ContainsFunc(tables, func(table wipeTable) bool { return table.counter.MatchString(counterName) }) {
			counters = append(counters, counterName)
		}
	}

	// The store ConfigureFileStore installed: S3 on Lambda, the VPS disk otherwise.
	fileStore := db.FileStore()
	snapshotFiles := []string{}
	for _, table := range tables {
		for _, prefix := range []string{"delta-snapshot", "cache-group"} {
			files, err := fileStore.ListFiles(fmt.Sprintf("%s/%s/%d/", prefix, table.name, companyID))
			if err != nil {
				return core.Err("No se pudieron listar los snapshots:", err)
			}
			snapshotFiles = append(snapshotFiles, files...)
		}
	}

	tableNames := []string{}
	for _, table := range tables {
		tableNames = append(tableNames, table.name)
	}
	summary := fmt.Sprintf("Empresa %d: %d tablas a vaciar, %d se conservan, %d contadores a 0, %d snapshots.\nVaciar: %s\nConservar: %s\nContadores: %s",
		companyID, len(tables), len(kept), len(counters), len(snapshotFiles),
		strings.Join(tableNames, ", "), strings.Join(kept, ", "), strings.Join(counters, ", "))
	if !apply {
		fmt.Println("SIMULACIÓN (nada se borró; use --apply)\n" + summary)
		return nil
	}

	for _, table := range tables {
		description := db.TableDescription{Name: table.name, ID: table.id, Partition: table.column}
		if err := db.WipePartition(description, companyID); err != nil {
			return core.Err("Falló el vaciado de", table.name, ":", err)
		}
	}
	for _, controller := range controllers {
		if slices.Contains(tableNames, controller.GetTableName()) {
			if err := controller.DropTextSearch(companyID); err != nil {
				return core.Err("No se pudo vaciar el índice de texto de", controller.GetTableName(), ":", err)
			}
			if err := controller.RebuildDataFramesPartition(companyID); err != nil {
				return core.Err("No se pudieron reconstruir los DataFrames de", controller.GetTableName(), ":", err)
			}
		}
	}
	if len(snapshotFiles) > 0 {
		if err := fileStore.DeleteFiles(context.Background(), snapshotFiles...); err != nil {
			return core.Err("No se pudieron borrar los snapshots:", err)
		}
	}
	for _, counterName := range counters {
		if _, err := db.SetCounter(counterName, 0); err != nil {
			return core.Err("No se pudo reiniciar el contador", counterName, ":", err)
		}
	}

	// A partial wipe (--only) left the sites, warehouses and cash registers in place.
	if onlyPrefix != "" {
		fmt.Println("DATOS BORRADOS\n" + summary)
		return nil
	}
	exec.SeedCompanyOperatingRecords([]int32{companyID}, core.SUnixTime())
	if _, err := cloud.CompactAndStoreCompanyConfig(companyID); err != nil {
		return core.Err("No se pudo publicar la configuración de la empresa:", err)
	}
	// A resume point into data that no longer exists would replay the rest of a plan over nothing.
	if err := os.Remove(runner.StatePathFor(companyID)); err != nil && !os.IsNotExist(err) {
		return core.Err("No se pudo borrar el estado de la siembra:", err)
	}
	fmt.Println("DATOS BORRADOS\n" + summary)
	return nil
}
