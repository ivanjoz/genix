package discovery

import "testing"

func TestNormalizeDocumentationSearchRemovesCustomerNameAndNumbers(t *testing.T) {
	plan := Plan{
		Goal: GoalManageRecord, Operation: OperationCreate, Domain: "customers",
		Entities: []EntityHint{{Type: "customer", Name: "Pedro Pascal"}},
		Searches: SearchPlan{
			DocumentationNavigation: SearchDecision{Needed: true, Query: `Agregar un cliente llamado "Pedro Pascal" 123`},
			AgentTools:              SearchDecision{Needed: true, Query: `Buscar ventas de Pedro Pascal en 2026`},
		},
	}

	normalized := normalizeDocumentationSearch(plan)
	if normalized.Searches.DocumentationNavigation.Query != "Agregar un cliente" {
		t.Fatalf("instance values leaked into documentation query: %q", normalized.Searches.DocumentationNavigation.Query)
	}
	if normalized.Searches.AgentTools.Query != plan.Searches.AgentTools.Query {
		t.Fatalf("agent-tool query lost record filters: %q", normalized.Searches.AgentTools.Query)
	}
}

func TestNormalizeDocumentationSearchRemovesProductNameAndAmount(t *testing.T) {
	plan := Plan{
		Goal: GoalManageRecord, Operation: OperationCreate, Domain: "products",
		Entities: []EntityHint{{Type: "product", Name: "Aceite Tondero"}},
		Searches: SearchPlan{DocumentationNavigation: SearchDecision{
			Needed: true, Query: "Crear un producto llamado Aceite Tondero a 12 soles",
		}},
	}

	query := normalizeDocumentationSearch(plan).Searches.DocumentationNavigation.Query
	if query != "Crear un producto" {
		t.Fatalf("product instance values leaked into documentation query: %q", query)
	}
}
