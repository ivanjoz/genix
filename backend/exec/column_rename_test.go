package exec

import (
	"testing"

	"github.com/ivanjoz/genix-orm/db"
)

// goFieldByNewColumn names the Go field each migrated column belongs to, so the plan cannot drift
// from what the ORM derives once the tags are gone.
var goFieldByNewColumn = map[string]string{
	"company_id": "CompanyID", "name": "Name", "description": "Description", "modules": "Modules",
	"params": "Params", "address": "Address", "city_id": "CityID", "id": "ID", "culqi_config": "CulqiConfig", "last_payment_caja_id": "LastPaymentCajaID",
}

func TestColumnRenamesMatchSnakeCase(t *testing.T) {
	for _, tablePlan := range columnRenamePlan {
		for _, rename := range tablePlan.Renames {
			goField, known := goFieldByNewColumn[rename.NewName]
			if !known {
				t.Fatalf("%s.%s has no Go field in the test map", tablePlan.Table, rename.NewName)
			}
			if derived := db.ToSnakeCase(goField); derived != rename.NewName {
				t.Errorf("%s: %s derives %q, plan says %q", tablePlan.Table, goField, derived, rename.NewName)
			}
		}
	}
}

func TestPlanTableRenames(t *testing.T) {
	renames := []columnRename{{"empresa_id", "company_id"}, {"nombre", "name"}, {"direccion", "address"}}
	liveColumns := map[string]liveColumn{
		"empresa_id": {"partition_key", "int"},
		"name":       {"regular", "text"}, // already migrated
		"direccion":  {"regular", "text"},
	}
	steps, planError := planTableRenames("sites", liveColumns, renames)
	if planError != nil {
		t.Fatal(planError)
	}
	if !steps[0].Pending || !steps[0].IsKey || steps[1].Pending || !steps[2].Pending || steps[2].IsKey || steps[2].Type != "text" {
		t.Errorf("unexpected plan: %+v", steps)
	}
	if _, planError = planTableRenames("sites", map[string]liveColumn{}, renames); planError == nil {
		t.Error("a table with neither column must be refused")
	}
}
