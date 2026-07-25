package exec

import (
	"app/db"
	"github.com/ivanjoz/genix-orm/scylla"
	"testing"
)

// The generated init() in controllers.generated.go registers every table by name. This asserts the
// three tables that opt into GenericRecord resolve through that registry and that their declared
// columns map to real table columns — a typo in a GenericRecord slot panics here instead of on a
// live request. It needs no database connection: only table metadata is compiled.
func TestGenericRecordSchemasResolveThroughTheNameRegistry(t *testing.T) {
	testCases := []struct {
		tableName          string
		expectedProjection string
	}{
		// products.Name is stored in the legacy "nombre" column — the plan resolves real column names.
		{"products", "id, nombre, sku, final_price, brand_id, status"},
		{"client_provider", "id, name, registry_number, type, status"},
		{"users", "id, user, first_name, last_name, status"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.tableName, func(t *testing.T) {
			resolvedTable, err := db.ResolveTableByName(testCase.tableName)
			if err != nil {
				t.Fatalf("table %q is not registered: %v", testCase.tableName, err)
			}
			scyllaTable, ok := resolvedTable.(scylla.ScyllaTable)
			if !ok {
				t.Fatalf("table %q was not compiled by the scylla driver", testCase.tableName)
			}
			if projection := scyllaTable.GenericRecordProjection(); projection != testCase.expectedProjection {
				t.Fatalf("projection mismatch for %q\n got: %v\nwant: %v",
					testCase.tableName, projection, testCase.expectedProjection)
			}
		})
	}
}

// Tables that never declared GenericRecord must stay unexposed even though they are registered.
func TestTablesWithoutGenericRecordAreNotExposed(t *testing.T) {
	for _, tableName := range []string{"warehouses", "sale_order", "expenses"} {
		resolvedTable, err := db.ResolveTableByName(tableName)
		if err != nil {
			t.Fatalf("table %q is not registered: %v", tableName, err)
		}
		scyllaTable, ok := resolvedTable.(scylla.ScyllaTable)
		if !ok {
			t.Fatalf("table %q was not compiled by the scylla driver", tableName)
		}
		if projection := scyllaTable.GenericRecordProjection(); projection != "" {
			t.Fatalf("table %q must not expose generic records, got projection: %v", tableName, projection)
		}
	}
}
