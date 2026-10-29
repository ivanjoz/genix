// One-off migration for the columns that used to carry a `db:"..."` name the ORM would not have
// derived on its own (Spanish names, `_ids` suffixes). Their tags were deleted so the column name
// is ToSnakeCase(field name); this copies the live data over to the new names.
//
// Run it BEFORE deploying the backend that no longer has the tags, then run fn-homologate so the
// views and indexes it drops are recreated:
//
//	cd backend
//	go run . fn-rename-columns          # report only
//	go run . fn-rename-columns apply
//	go run . fn-homologate
//
// It is state-based, so re-running after a failure picks up where it stopped.

package exec

import (
	"app/core"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gocql/gocql"
	"github.com/ivanjoz/genix-orm/scylla"
)

type columnRename struct {
	OldName string
	NewName string
}

type tableColumnRenames struct {
	Table   string
	Renames []columnRename
}

// columnRenamePlan lists every table whose tags were removed. NewName is what ToSnakeCase gives the
// Go field (see TestColumnRenamesMatchSnakeCase), so the typed ORM finds the data after the swap.
var columnRenamePlan = []tableColumnRenames{
	{Table: "profiles", Renames: []columnRename{
		{"empresa_id", "company_id"}, {"nombre", "name"}, {"descripcion", "description"}, {"modulos_ids", "modules"},
	}},
	{Table: "products", Renames: []columnRename{
		{"nombre", "name"}, {"params_ids", "params"},
	}},
	{Table: "sites", Renames: []columnRename{
		{"empresa_id", "company_id"}, {"nombre", "name"}, {"descripcion", "description"},
		{"direccion", "address"}, {"pais_ciudad_id", "city_id"},
	}},
	{Table: "system_parameters", Renames: []columnRename{
		{"empresa_id", "company_id"}, {"parameter_id", "id"},
	}},
	{Table: "companies", Renames: []columnRename{{"culqui_config", "culqi_config"}}},
	{Table: "sale_order", Renames: []columnRename{{"caja_id_", "last_payment_caja_id"}}},
	{Table: "gallery_images", Renames: []columnRename{{"empresa_id", "company_id"}}},
}

type liveColumn struct {
	Kind string // partition_key, clustering, regular
	Type string
}

type renameStep struct {
	columnRename
	IsKey   bool
	Type    string
	Pending bool // false = already migrated
}

// planTableRenames decides, per rename, whether it still has to run, from the live catalog alone.
// A rename whose old and new column are both absent is an error: the plan does not describe this table.
func planTableRenames(table string, liveColumns map[string]liveColumn, renames []columnRename) ([]renameStep, error) {
	steps := []renameStep{}
	for _, rename := range renames {
		oldColumn, hasOld := liveColumns[rename.OldName]
		_, hasNew := liveColumns[rename.NewName]
		switch {
		case hasOld:
			steps = append(steps, renameStep{rename, oldColumn.Kind != "regular", oldColumn.Type, true})
		case hasNew:
			steps = append(steps, renameStep{columnRename: rename})
		default:
			return nil, fmt.Errorf("table %s has neither %q nor %q", table, rename.OldName, rename.NewName)
		}
	}
	return steps, nil
}

// RenameColumns copies the data of the de-tagged columns to their derived names.
//
// fn-rename-columns [apply]
func RenameColumns(args *core.ExecArgs) core.FuncResponse {
	shouldWrite := strings.Contains(args.Message, "apply")
	keyspace := core.Env.DB_NAME
	session := scylla.MakeScyllaConnection(makeConnParams())

	for _, tablePlan := range columnRenamePlan {
		liveColumns := map[string]liveColumn{}
		columnIterator := session.Query(
			"SELECT column_name, kind, type FROM system_schema.columns WHERE keyspace_name = ? AND table_name = ?",
			keyspace, tablePlan.Table).Iter()
		var columnName string
		var live liveColumn
		for columnIterator.Scan(&columnName, &live.Kind, &live.Type) {
			liveColumns[columnName] = live
		}
		if closeError := columnIterator.Close(); closeError != nil {
			return args.MakeErr("could not read the columns of", tablePlan.Table, closeError)
		}

		steps, planError := planTableRenames(tablePlan.Table, liveColumns, tablePlan.Renames)
		if planError != nil {
			return args.MakeErr(planError)
		}
		pendingSteps := []renameStep{}
		for _, step := range steps {
			if step.Pending {
				pendingSteps = append(pendingSteps, step)
			}
		}
		fmt.Printf("%s: %d of %d renames pending\n", tablePlan.Table, len(pendingSteps), len(steps))
		if len(pendingSteps) == 0 || !shouldWrite {
			continue
		}
		if migrateError := migrateTableColumns(session, keyspace, tablePlan.Table, liveColumns, pendingSteps); migrateError != nil {
			return args.MakeErr("table", tablePlan.Table, "failed:", migrateError)
		}
	}
	if !shouldWrite {
		fmt.Println("report only; pass `apply` to write. Run fn-homologate afterwards.")
	}
	return core.FuncResponse{Message: "ok"}
}

func migrateTableColumns(session *gocql.Session, keyspace, table string, liveColumns map[string]liveColumn, steps []renameStep) error {
	fullTableName := keyspace + "." + table

	// A materialized view or index over a column blocks both RENAME and DROP COLUMN. They are derived
	// data: fn-homologate recreates them from the schema once the new names are in place.
	if dropError := dropViewsAndIndexes(session, keyspace, table); dropError != nil {
		return dropError
	}

	// Primary-key columns are the only ones CQL can rename in place, and it keeps their data.
	for _, step := range steps {
		if step.IsKey {
			if execError := session.Query(fmt.Sprintf("ALTER TABLE %s RENAME %s TO %s", fullTableName, step.OldName, step.NewName)).Exec(); execError != nil {
				return execError
			}
		}
	}

	// Every other column is added under its new name and filled row by row from the old one.
	keyColumns := []string{}
	for columnName, column := range liveColumns {
		if column.Kind != "regular" {
			keyColumns = append(keyColumns, columnName)
		}
	}
	for index, keyColumn := range keyColumns {
		for _, step := range steps {
			if step.IsKey && step.OldName == keyColumn {
				keyColumns[index] = step.NewName
			}
		}
	}
	for _, step := range steps {
		if step.IsKey {
			continue
		}
		if addError := session.Query(fmt.Sprintf("ALTER TABLE %s ADD %s %s", fullTableName, step.NewName, step.Type)).Exec(); addError != nil &&
			!strings.Contains(addError.Error(), "conflicts with an existing column") {
			return addError
		}
		copiedRows, copiedRecords, copyError := copyColumnValues(session, fullTableName, keyColumns, step)
		if copyError != nil {
			return copyError
		}
		// The old column is about to be dropped for good, so its rows (with their keys) go to disk first.
		backupPath := fmt.Sprintf("tmp/column-rename/%s.%s.json", table, step.OldName)
		backupBytes, _ := json.Marshal(copiedRecords)
		if writeError := os.MkdirAll("tmp/column-rename", 0o755); writeError != nil {
			return writeError
		}
		if writeError := os.WriteFile(backupPath, backupBytes, 0o644); writeError != nil {
			return writeError
		}
		fmt.Printf("  %s: %s -> %s copied %d rows\n", table, step.OldName, step.NewName, copiedRows)
		if dropError := session.Query(fmt.Sprintf("ALTER TABLE %s DROP %s", fullTableName, step.OldName)).Exec(); dropError != nil {
			return dropError
		}
	}
	return nil
}

// dropViewsAndIndexes drops every live materialized view and secondary index of the table, read
// from the catalog so artifacts the schema no longer declares go too.
func dropViewsAndIndexes(session *gocql.Session, keyspace, table string) error {
	dropStatements := []string{}
	var objectName string

	// A secondary index is backed by a view named <index>_index, which only DROP INDEX can remove.
	indexBackingViews := map[string]bool{}
	indexIterator := session.Query(
		"SELECT index_name FROM system_schema.indexes WHERE keyspace_name = ? AND table_name = ?",
		keyspace, table).Iter()
	for indexIterator.Scan(&objectName) {
		indexBackingViews[objectName+"_index"] = true
		dropStatements = append(dropStatements, fmt.Sprintf("DROP INDEX IF EXISTS %s.%s", keyspace, objectName))
	}
	if closeError := indexIterator.Close(); closeError != nil {
		return closeError
	}

	viewIterator := session.Query(
		"SELECT view_name FROM system_schema.views WHERE keyspace_name = ? AND base_table_name = ? ALLOW FILTERING",
		keyspace, table).Iter()
	for viewIterator.Scan(&objectName) {
		if !indexBackingViews[objectName] {
			dropStatements = append(dropStatements, fmt.Sprintf("DROP MATERIALIZED VIEW IF EXISTS %s.%s", keyspace, objectName))
		}
	}
	if closeError := viewIterator.Close(); closeError != nil {
		return closeError
	}

	for _, statement := range dropStatements {
		fmt.Println("  " + statement)
		if execError := session.Query(statement).Exec(); execError != nil {
			return execError
		}
	}
	return nil
}

// copyColumnValues writes the old column's value into the new one for every row that has one. The
// value travels as the Go type gocql scanned it into, so collections copy as collections.
func copyColumnValues(session *gocql.Session, fullTableName string, keyColumns []string, step renameStep) (int, []map[string]any, error) {
	selectColumns := append(append([]string{}, keyColumns...), step.OldName)
	rowIterator := session.Query(fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectColumns, ", "), fullTableName)).Iter()

	keyConditions := make([]string, len(keyColumns))
	for index, keyColumn := range keyColumns {
		keyConditions[index] = keyColumn + " = ?"
	}
	updateStatement := fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s", fullTableName, step.NewName, strings.Join(keyConditions, " AND "))

	// The updates are independent single-row writes against a remote node, so latency, not load, is
	// the cost: they run concurrently and the first failure is kept.
	var waitGroup sync.WaitGroup
	var firstUpdateError error
	var errorLock sync.Mutex
	inFlight := make(chan struct{}, 32)

	copiedRecords := []map[string]any{}
	rowValues := map[string]any{}
	for rowIterator.MapScan(rowValues) {
		oldValue := rowValues[step.OldName]
		if oldValue != nil {
			updateValues := []any{oldValue}
			for _, keyColumn := range keyColumns {
				updateValues = append(updateValues, rowValues[keyColumn])
			}
			copiedRecords = append(copiedRecords, rowValues)
			inFlight <- struct{}{}
			waitGroup.Add(1)
			go func() {
				defer func() { <-inFlight; waitGroup.Done() }()
				if updateError := session.Query(updateStatement, updateValues...).Exec(); updateError != nil {
					errorLock.Lock()
					firstUpdateError = updateError
					errorLock.Unlock()
				}
			}()
		}
		rowValues = map[string]any{}
	}
	waitGroup.Wait()
	if firstUpdateError != nil {
		rowIterator.Close()
		return 0, nil, firstUpdateError
	}
	return len(copiedRecords), copiedRecords, rowIterator.Close()
}
