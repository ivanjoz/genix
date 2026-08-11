package agent

import (
	"strings"
	"testing"

	"app/agent/knowledge"
	"app/agent/llm"
	"app/agent/routing"
)

func TestSelectCompletedTurnsPreservesChronologicalNonContiguousOrder(t *testing.T) {
	turns := []routing.CompletedTurn{{Offset: -3}, {Offset: -2}, {Offset: -1}}
	selected := selectCompletedTurns(turns, []int{-1, -3})
	if len(selected) != 2 || selected[0].Offset != -3 || selected[1].Offset != -1 {
		t.Fatalf("unexpected selected turns: %+v", selected)
	}
}

func TestHasCapabilityRequiresExactCapability(t *testing.T) {
	capabilities := []routing.CapabilityName{routing.CapabilityMenu, routing.CapabilityDocumentationSearch}
	if !hasCapability(capabilities, routing.CapabilityDocumentationSearch) {
		t.Fatal("documentation capability was not detected")
	}
	if hasCapability(capabilities, routing.CapabilitySalesSearch) {
		t.Fatal("unavailable sales capability was incorrectly detected")
	}
}

func TestToolsForVerdictRestrictsReadAndActionRoutes(t *testing.T) {
	productTools := toolsForVerdict(routing.Verdict{PrimaryIntent: routing.IntentProductKnowledge, RequestedOperation: routing.OperationRead})
	if got := toolNames(productTools); len(got) != 1 || got[0] != "finish" {
		t.Fatalf("product knowledge received extra tools: %v", got)
	}

	navigationVerdict := routing.Verdict{
		PrimaryIntent: routing.IntentProductKnowledge, RequestedOperation: routing.OperationNavigate,
	}
	if got := toolNames(toolsForVerdict(navigationVerdict)); strings.Join(got, ",") != "get_menu,navigate,get_page,finish" {
		t.Fatalf("unexpected navigation tools: %v", got)
	}

	actionTools := toolNames(toolsForVerdict(routing.Verdict{PrimaryIntent: routing.IntentPageAction}))
	if !containsString(actionTools, "invoke_batch") {
		t.Fatalf("page action lacks mutation tool: %v", actionTools)
	}
}

func toolNames(tools []llm.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Function.Name)
	}
	return names
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func TestMenuRoutesDeduplicatesAccessibleNavigationTargets(t *testing.T) {
	menu := []AgentMenuGroup{{Options: []AgentMenuOption{{Route: "/finance/cash-banks"}, {Route: " /finance/cash-banks "}, {Route: "/sales/orders"}}}}
	routes := menuRoutes(menu)
	if len(routes) != 2 || routes[0] != "/finance/cash-banks" || routes[1] != "/sales/orders" {
		t.Fatalf("unexpected menu routes: %+v", routes)
	}
}

func TestDocumentationEvidenceExposesNavigationButNotDiagnostics(t *testing.T) {
	evidence := buildDocumentationEvidence([]knowledge.DocumentationResult{{
		CitationID: "finance.cash-banks#capability.reconcile", PageTitle: "Cash & Banks",
		Route: "/finance/cash-banks", SectionTitle: "Reconcile cash", Content: "Compare observed cash.",
		PointID: "private-point", DocumentationHash: "private-hash",
	}})
	if !strings.Contains(evidence, "/finance/cash-banks") || !strings.Contains(evidence, "finance.cash-banks#capability.reconcile") {
		t.Fatalf("navigation evidence missing: %s", evidence)
	}
	if strings.Contains(evidence, "private-point") || strings.Contains(evidence, "private-hash") {
		t.Fatalf("diagnostic identifiers leaked into prompt: %s", evidence)
	}
}

func TestLiveContextMatchesExactBuilderIdentity(t *testing.T) {
	surface := routing.SurfaceContext{
		Kind: routing.SurfaceWebpageEditor, Route: "/webpage-builder/42", PageID: "42",
		HasSelectedSection: true, SelectedSectionID: "hero",
	}
	result := AgentContextResult{
		SurfaceKind: string(surface.Kind), Route: surface.Route, PageID: surface.PageID,
		Scope: string(routing.BuilderScopeSelectedSection), SelectedSectionID: "hero", Content: "html",
	}
	if !liveContextMatches(surface, routing.BuilderScopeSelectedSection, result) {
		t.Fatal("matching builder state was rejected")
	}
	result.SelectedSectionID = "different"
	if liveContextMatches(surface, routing.BuilderScopeSelectedSection, result) {
		t.Fatal("stale selected section was accepted")
	}
}
