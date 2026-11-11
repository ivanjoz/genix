package cloud

import (
	config "app/config/types"
	coreTypes "app/core/types"
	"app/db"
	security "app/security/types"
	"reflect"
	"testing"
)

// TestMirroredSchemasCompile exercises the schema compiler on every mirrored table.
// buildTableMeta only reads the declaration; compiling it is what catches an index the
// primary database cannot build (a local index on a partitionless table, a leading view
// column with an explicit width).
func TestMirroredSchemasCompile(t *testing.T) {
	if columnCount := len(db.MakeTable[coreTypes.User]().GetColumns()); columnCount == 0 {
		t.Error("users compiled with no columns")
	}
	if columnCount := len(db.MakeTable[security.Profile]().GetColumns()); columnCount == 0 {
		t.Error("profiles compiled with no columns")
	}
	if columnCount := len(db.MakeTable[config.Company]().GetColumns()); columnCount == 0 {
		t.Error("companies compiled with no columns")
	}
}

// The composite index value is the mirror's only lookup key on DynamoDB, so its exact
// format is a storage contract: a change here silently invalidates every index value
// already written. These tests pin the format and the slot assignment.

func userMeta(t *testing.T) *TableMeta {
	t.Helper()
	meta, err := buildTableMeta[coreTypes.User]()
	if err != nil {
		t.Fatalf("buildTableMeta(User): %v", err)
	}
	return meta
}

func columnNames(meta *TableMeta) []string {
	names := make([]string, 0, len(meta.Columns))
	for _, column := range meta.Columns {
		names = append(names, column.ColumnName)
	}
	return names
}

func hasColumn(meta *TableMeta, columnName string) bool {
	for _, column := range meta.Columns {
		if column.ColumnName == columnName {
			return true
		}
	}
	return false
}

func TestUserMetaMirrorsTheTableStructColumns(t *testing.T) {
	meta := userMeta(t)

	if meta.TableName != "users" {
		t.Errorf("table name = %q, want %q", meta.TableName, "users")
	}
	if meta.PartitionColumn != "company_id" {
		t.Errorf("partition column = %q, want %q", meta.PartitionColumn, "company_id")
	}

	// password_hash is the column whose absence broke self-hosted login.
	if !hasColumn(meta, "password_hash") {
		t.Errorf("password_hash is not mirrored; columns = %v", columnNames(meta))
	}
	// Password is a write-only record field with no column, so it must never reach a database.
	if hasColumn(meta, "password") {
		t.Errorf("plaintext password column leaked into the mirror; columns = %v", columnNames(meta))
	}
}

func TestUserIndexSlotsFollowDeclarationOrder(t *testing.T) {
	meta := userMeta(t)

	expectedSlots := []struct {
		dynamoIndex string
		keyColumns  []string
		keyDigits   []int8
	}{
		{"ix1", []string{"user"}, []int8{0}},
		{"ix2", []string{"email"}, []int8{0}},
		// The users' TypeDelta has no mirror slot.
	}

	if len(meta.Indexes) != len(expectedSlots) {
		t.Fatalf("index count = %d, want %d", len(meta.Indexes), len(expectedSlots))
	}

	for slotIndex, expected := range expectedSlots {
		index := meta.Indexes[slotIndex]
		if index.DynamoIndex != expected.dynamoIndex {
			t.Errorf("slot %d = %q, want %q", slotIndex, index.DynamoIndex, expected.dynamoIndex)
		}
		keyColumns := []string{}
		keyDigits := []int8{}
		for _, key := range index.Keys {
			keyColumns = append(keyColumns, key.ColumnName)
			keyDigits = append(keyDigits, key.Digits)
		}
		if !reflect.DeepEqual(keyColumns, expected.keyColumns) {
			t.Errorf("slot %s keys = %v, want %v", index.DynamoIndex, keyColumns, expected.keyColumns)
		}
		if !reflect.DeepEqual(keyDigits, expected.keyDigits) {
			t.Errorf("slot %s digits = %v, want %v", index.DynamoIndex, keyDigits, expected.keyDigits)
		}
	}
}

func TestBuildIndexValuePrefixesWithThePartition(t *testing.T) {
	user := coreTypes.User{CompanyID: 7, ID: 1, User: "admin"}
	if got, want := userMeta(t).Indexes[0].buildIndexValue(reflect.ValueOf(user)), "7_admin"; got != want {
		t.Errorf("user index value = %q, want %q", got, want)
	}
}

func TestBuildIndexKeyRangeIsExactForAnEquality(t *testing.T) {
	meta := userMeta(t)
	condition := queryCondition{ColumnName: "user", Operator: "=", Value: "admin"}

	index, err := matchIndex(meta.Indexes, []queryCondition{condition})
	if err != nil {
		t.Fatalf("matchIndex: %v", err)
	}

	indexRange, err := buildIndexKeyRange(index, "7", condition)
	if err != nil {
		t.Fatalf("buildIndexKeyRange: %v", err)
	}
	if !indexRange.IsExact || indexRange.Lower != "7_admin" {
		t.Errorf("range = %+v, want exact 7_admin", indexRange)
	}
}

func TestMatchIndexRejectsUnindexedAndMultiColumnQueries(t *testing.T) {
	meta := userMeta(t)

	if _, err := matchIndex(meta.Indexes, []queryCondition{
		{ColumnName: "job_title", Operator: "=", Value: "x"},
	}); err == nil {
		t.Error("querying an unindexed column must fail rather than scan")
	}

	if _, err := matchIndex(meta.Indexes, []queryCondition{
		{ColumnName: "user", Operator: "=", Value: "admin"},
		{ColumnName: "email", Operator: "=", Value: "a@b.c"},
	}); err == nil {
		t.Error("a query on two columns must be rejected: the mirror has no composite indexes")
	}
}

func TestCompanyHasNoPartitionPrefix(t *testing.T) {
	meta, err := buildTableMeta[config.Company]()
	if err != nil {
		t.Fatalf("buildTableMeta(Company): %v", err)
	}

	if meta.PartitionColumn != "" {
		t.Errorf("partition column = %q, want empty: companies are global", meta.PartitionColumn)
	}

	// GET.empresas on the mirror: companies updated since a watermark.
	condition := queryCondition{ColumnName: "updated", Operator: ">=", Value: 1234}
	index, err := matchIndex(meta.Indexes, []queryCondition{condition})
	if err != nil {
		t.Fatalf("matchIndex: %v", err)
	}

	indexRange, err := buildIndexKeyRange(index, "", condition)
	if err != nil {
		t.Fatalf("buildIndexKeyRange: %v", err)
	}
	if indexRange.IsExact {
		t.Error("a >= query must not produce an exact match")
	}
	if got, want := indexRange.Lower, "0000001234"; got != want {
		t.Errorf("lower = %q, want %q (no tenant prefix)", got, want)
	}
	if got, want := indexRange.Upper, "9999999999"; got != want {
		t.Errorf("upper = %q, want %q", got, want)
	}
}
