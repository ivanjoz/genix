package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"app/agent/discovery"
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

func TestExecutionToolsKeepNavigationForReadGoals(t *testing.T) {
	for _, goal := range []discovery.Goal{discovery.GoalExplainProduct, discovery.GoalViewReport, discovery.GoalQueryCompanyData} {
		names := toolNames(buildExecutionPolicy(discovery.Plan{Goal: goal}, routing.SurfaceContext{}, discovery.EmptyFeatureResult()).Tools)
		if strings.Join(names, ",") != "get_menu,navigate,get_page,finish" {
			t.Fatalf("goal %s lost safe navigation tools: %v", goal, names)
		}
	}
}

func TestExecutionPolicyAllowsRecordFormActionsOnlyOnPrimaryDiscoveredRoute(t *testing.T) {
	matchingFeatures := discovery.FeatureSearchResult{Routes: []discovery.RouteCandidate{{Route: "/business/products"}}}
	policy := buildExecutionPolicy(
		discovery.Plan{Goal: discovery.GoalManageRecord},
		routing.SurfaceContext{Route: "/finance/cash-banks"}, matchingFeatures,
	)
	if !containsString(toolNames(policy.Tools), llm.InvokeBatchToolName) {
		t.Fatalf("record creation cannot fill the destination form: %v", toolNames(policy.Tools))
	}
	if len(policy.MutationRoutes) != 1 || policy.MutationRoutes[0] != "/business/products" {
		t.Fatalf("record actions were not bound to the primary route: %+v", policy)
	}
	if routeAllowed("/finance/cash-banks", policy.MutationRoutes) || !routeAllowed("/business/products", policy.MutationRoutes) {
		t.Fatalf("record action route guard is incorrect: %+v", policy.MutationRoutes)
	}
	inspectNames := toolNames(buildExecutionPolicy(discovery.Plan{Goal: discovery.GoalInspectCurrentPage}, routing.SurfaceContext{}, discovery.EmptyFeatureResult()).Tools)
	if containsString(inspectNames, llm.InvokeBatchToolName) || strings.Join(inspectNames, ",") != "get_page,finish" {
		t.Fatalf("page inspection received mutation tools: %v", inspectNames)
	}
}

func TestExecutionPolicyRevalidatesCurrentSurfaceRouteForFollowUp(t *testing.T) {
	plan := discovery.Plan{Goal: discovery.GoalManageRecord, Operation: discovery.OperationUpdate}
	surface := routing.SurfaceContext{Kind: routing.SurfaceERPPage, Route: "/logistics/products-stock"}
	menu := []AgentMenuGroup{{Options: []AgentMenuOption{{Route: "/logistics/products-stock"}}}}

	validated := addCurrentSurfaceRoute(plan, surface, discovery.EmptyFeatureResult(), menu)
	policy := buildExecutionPolicy(plan, surface, validated)
	if !routeAllowed(surface.Route, policy.MutationRoutes) || !containsString(toolNames(policy.Tools), llm.InvokeBatchToolName) {
		t.Fatalf("validated follow-up route cannot mutate current page: routes=%v tools=%v", policy.MutationRoutes, toolNames(policy.Tools))
	}

	rejected := addCurrentSurfaceRoute(plan, surface, discovery.EmptyFeatureResult(), []AgentMenuGroup{})
	if policy := buildExecutionPolicy(plan, surface, rejected); len(policy.MutationRoutes) != 0 || containsString(toolNames(policy.Tools), llm.InvokeBatchToolName) {
		t.Fatalf("route absent from accessible menu received mutation access: %+v", policy)
	}
}

func TestExecutionContextRequiresCreateFormFillBeforeConfirmation(t *testing.T) {
	bundle := discovery.Bundle{
		Plan: discovery.Plan{ResponseLanguage: routing.LanguageSpanish, Goal: discovery.GoalManageRecord},
	}
	context := executionDiscoveryContext(bundle, routing.SurfaceContext{Kind: routing.SurfaceERPPage})
	for _, instruction := range []string{"open New/Create", "fill every field", "separate explicit user confirmation"} {
		if !strings.Contains(context, instruction) {
			t.Fatalf("create-form instruction %q missing from execution context: %s", instruction, context)
		}
	}
}

func TestExecutionContextTreatsEmptyCatalogAsSuccessful(t *testing.T) {
	bundle := discovery.Bundle{
		Plan:                    discovery.Plan{ResponseLanguage: routing.LanguageSpanish, Goal: discovery.GoalViewReport},
		DocumentationNavigation: discovery.EmptyFeatureResult(), AgentTools: discovery.EmptyToolResult(),
	}
	context := executionDiscoveryContext(bundle, routing.SurfaceContext{Kind: routing.SurfaceERPPage})
	if !strings.Contains(context, `"status":"ok"`) || !strings.Contains(context, `"tools":[]`) {
		t.Fatalf("empty catalog was not preserved as a successful result: %s", context)
	}
	if !strings.Contains(context, "empty agent-tools result is successful") {
		t.Fatalf("UI fallback policy missing from execution context: %s", context)
	}
}

func TestDiscoveryJobsStartInParallel(t *testing.T) {
	featureStarted := make(chan struct{})
	toolStarted := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	plan := discovery.Plan{Searches: discovery.SearchPlan{
		DocumentationNavigation: discovery.SearchDecision{Needed: true, Query: "products"},
		AgentTools:              discovery.SearchDecision{Needed: true, Query: "products"},
	}}
	go func() {
		defer close(finished)
		runDiscoveryJobs(context.Background(), plan,
			func(context.Context) discovery.FeatureSearchResult {
				close(featureStarted)
				<-release
				return discovery.EmptyFeatureResult()
			},
			func(context.Context) discovery.ToolSearchResult {
				close(toolStarted)
				<-release
				return discovery.EmptyToolResult()
			},
		)
	}()
	for _, started := range []<-chan struct{}{featureStarted, toolStarted} {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("selected discovery jobs did not start concurrently")
		}
	}
	close(release)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("parallel discovery did not finish")
	}
}

func TestDiscoverySearchStatusMatchesSelectedSources(t *testing.T) {
	tests := []struct {
		name     string
		searches discovery.SearchPlan
		expected string
	}{
		{name: "none", expected: ""},
		{name: "documentation", searches: discovery.SearchPlan{DocumentationNavigation: discovery.SearchDecision{Needed: true}}, expected: "Consultando documentación y navegación…"},
		{name: "tools", searches: discovery.SearchPlan{AgentTools: discovery.SearchDecision{Needed: true}}, expected: "Buscando herramientas disponibles…"},
		{name: "both", searches: discovery.SearchPlan{DocumentationNavigation: discovery.SearchDecision{Needed: true}, AgentTools: discovery.SearchDecision{Needed: true}}, expected: "Consultando documentación y herramientas…"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := discovery.Plan{ResponseLanguage: routing.LanguageSpanish, Searches: test.searches}
			if status := discoverySearchStatus(plan); status != test.expected {
				t.Fatalf("unexpected status %q, expected %q", status, test.expected)
			}
		})
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

func TestAccessibleFeaturesPreserveMenuRoutes(t *testing.T) {
	menu := []AgentMenuGroup{{Name: "Business", Options: []AgentMenuOption{{Name: "Products", Route: " /business/products ", Description: "Create products"}}}}
	features := accessibleFeatures(menu)
	if len(features) != 1 || features[0].Route != "/business/products" || features[0].Name != "Products" {
		t.Fatalf("unexpected accessible features: %+v", features)
	}
}

func TestMenuContainsOnlyExactAccessibleRoute(t *testing.T) {
	menu := []AgentMenuGroup{{Options: []AgentMenuOption{{Route: "/business/products"}}}}
	if !menuContainsRoute(menu, " /business/products ") {
		t.Fatal("accessible route was rejected")
	}
	if menuContainsRoute(menu, "/business/products/new") {
		t.Fatal("invented child route was accepted")
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
