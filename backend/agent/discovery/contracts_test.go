package discovery

import (
	"testing"

	"app/agent/routing"
)

func validPlannerInput() PlannerInput {
	return PlannerInput{
		Schema: SchemaVersion, CurrentMessage: "Quiero el reporte de ventas", AppLanguage: routing.LanguageSpanish,
		Surface:        routing.SurfaceContext{Kind: routing.SurfaceERPPage, Route: "/finance/cash-banks"},
		CompletedTurns: []routing.CompletedTurn{{Offset: -1, UserMessage: "Hola", AssistantMessage: "Hola"}},
	}
}

func validPlan() Plan {
	return Plan{
		Schema: SchemaVersion, Language: routing.LanguageSpanish, ResponseLanguage: routing.LanguageSpanish,
		Scope: ScopeGenix, Goal: GoalViewReport, Operation: OperationRead, Domain: "sales",
		DeliveryPreference: DeliveryUnspecified, StandaloneRequest: "Quiero el reporte de ventas",
		Searches: SearchPlan{
			DocumentationNavigation: SearchDecision{Needed: true, Query: "Quiero el reporte de ventas"},
			AgentTools:              SearchDecision{Needed: true, Query: "Quiero el reporte de ventas"},
		},
		Builder: BuilderDecision{Operation: routing.BuilderNone, ContextScope: routing.BuilderScopeNone},
	}
}

func TestPlanAcceptsBothDiscoverySearches(t *testing.T) {
	if err := validPlan().Validate(validPlannerInput()); err != nil {
		t.Fatal(err)
	}
}

func TestPlanRejectsInconsistentSearchDecision(t *testing.T) {
	plan := validPlan()
	plan.Searches.AgentTools.Query = ""
	if err := plan.Validate(validPlannerInput()); err == nil {
		t.Fatal("enabled search without a query must be rejected")
	}
	plan = validPlan()
	plan.Searches.AgentTools.Needed = false
	if err := plan.Validate(validPlannerInput()); err == nil {
		t.Fatal("disabled search with a query must be rejected")
	}
}

func TestPlanRejectsUnavailableRelatedTurn(t *testing.T) {
	plan := validPlan()
	plan.RelatedTurnOffsets = []int{-2}
	if err := plan.Validate(validPlannerInput()); err == nil {
		t.Fatal("planner must not select a turn that was not supplied")
	}
}

func TestReportGoalRequiresReadOperation(t *testing.T) {
	plan := validPlan()
	plan.Operation = OperationNone
	if err := plan.Validate(validPlannerInput()); err == nil {
		t.Fatal("report goal without read operation must be rejected")
	}
}

func TestManageRecordConfirmationRequiresContextAndSkipsDiscovery(t *testing.T) {
	input := validPlannerInput()
	plan := validPlan()
	plan.Goal = GoalManageRecord
	plan.Operation = OperationConfirm
	plan.RelatedTurnOffsets = []int{-1}
	plan.Searches = SearchPlan{}
	if err := plan.Validate(input); err != nil {
		t.Fatalf("valid confirmation was rejected: %v", err)
	}

	plan.RelatedTurnOffsets = nil
	if err := plan.Validate(input); err == nil {
		t.Fatal("confirmation without a related turn must be rejected")
	}

	plan.RelatedTurnOffsets = []int{-1}
	plan.Searches.DocumentationNavigation = SearchDecision{Needed: true, Query: "Guardar stock"}
	if err := plan.Validate(input); err == nil {
		t.Fatal("confirmation with discovery must be rejected")
	}
}
