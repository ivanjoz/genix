package main

import "testing"

func TestViolationRejectsModuleToModuleImports(t *testing.T) {
	// The six edges this rule was written to remove (MODULE_BOUNDARIES_PLAN.md V1-V5).
	illegal := [][2]string{
		{"app/accounting", "app/finance"},
		{"app/accounting", "app/logistics"},
		{"app/logistics", "app/finance"},
		{"app/sales", "app/business"},
		{"app/sales", "app/finance"},
		{"app/sales", "app/logistics"},
		{"app/agent/pagebuilder", "app/business"},
	}
	for _, edge := range illegal {
		if reason := violation(edge[0], edge[1]); reason == "" {
			t.Errorf("%s -> %s should be a violation", edge[0], edge[1])
		}
	}
}

func TestViolationAllowsTheLegalEdges(t *testing.T) {
	legal := [][2]string{
		// every module may reach any */types, core, db and libs
		{"app/accounting", "app/finance/types"},
		{"app/sales", "app/business/types"},
		{"app/invoicing", "app/sales/types"},
		{"app/finance", "app/core"},
		{"app/logistics", "app/db"},
		{"app/config", "app/libs/servermetrics"},
		// infrastructure is below the modules
		{"app/business", "app/cloud"},
		// a module's own subpackages are internal to it
		{"app/agent", "app/agent/llm"},
		{"app/agent/discovery", "app/agent/knowledge"},
		{"app/finance", "app/finance/types"},
		// composition roots wire everything together — that is their job
		{"app/exec", "app/sales"},
		{"app/tests/sample_records", "app/finance"},
		{"main", "app/accounting"},
		// types may reach core/db and sibling types
		{"app/logistics/types", "app/core"},
		{"app/accounting/types", "app/business/types"},
	}
	for _, edge := range legal {
		if reason := violation(edge[0], edge[1]); reason != "" {
			t.Errorf("%s -> %s should be legal, got: %s", edge[0], edge[1], reason)
		}
	}
}

// The types layer is only load-bearing if it stays a leaf. If a */types package could reach
// a module body, every module could route around the rule through it.
func TestViolationKeepsTypesPackagesLeaf(t *testing.T) {
	if reason := violation("app/finance/types", "app/logistics"); reason == "" {
		t.Error("a types package reaching into a module body must be a violation")
	}
	if reason := violation("app/finance/types", "app/cloud"); reason == "" {
		t.Error("a types package reaching into infrastructure must be a violation")
	}
}

func TestModuleOf(t *testing.T) {
	cases := map[string]string{
		"app/agent/pagebuilder": "app/agent",
		"app/finance/types":     "app/finance",
		"app/finance":           "app/finance",
		"main":                  "main",
	}
	for in, want := range cases {
		if got := moduleOf(in); got != want {
			t.Errorf("moduleOf(%q) = %q, want %q", in, got, want)
		}
	}
}
