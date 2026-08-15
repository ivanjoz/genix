package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"app/agent/discovery"
	"app/agent/embedding"
	"app/agent/knowledge"
	"app/agent/llm"
	"app/agent/routing"
	"app/agent/webpage"
	"app/core"
)

const (
	featureDiscoveryTimeout = 45 * time.Second
	toolDiscoveryTimeout    = 5 * time.Second
)

var (
	discoveryPlannerOnce sync.Once
	discoveryPlanner     *discovery.Planner
	discoveryPlannerErr  error

	documentationRetrieverOnce sync.Once
	documentationRetriever     *knowledge.Retriever
	documentationRetrieverErr  error

	agentToolCatalog = discovery.NewToolCatalog()
)

func getDiscoveryPlanner() (*discovery.Planner, error) {
	discoveryPlannerOnce.Do(func() {
		chatClient, err := llm.NewClientForProvider(core.Env.CLASSIFIER_PROVIDER, core.Env.CLASSIFIER_MODEL_ID)
		if err != nil {
			discoveryPlannerErr = err
			return
		}
		discoveryPlanner, discoveryPlannerErr = discovery.NewPlanner(chatClient, core.Env.CLASSIFIER_MODEL_ID)
	})
	return discoveryPlanner, discoveryPlannerErr
}

func getDocumentationRetriever() (*knowledge.Retriever, error) {
	documentationRetrieverOnce.Do(func() {
		embeddingClient, err := embedding.NewClientFromEnv()
		if err != nil {
			documentationRetrieverErr = err
			return
		}
		qdrantStore, err := knowledge.NewStoreFromEnv()
		if err != nil {
			documentationRetrieverErr = err
			return
		}
		if err := qdrantStore.ValidateExistingCollection(context.Background()); err != nil {
			documentationRetrieverErr = err
			_ = qdrantStore.Close()
			return
		}
		documentationRetriever, documentationRetrieverErr = knowledge.NewRetriever(embeddingClient, qdrantStore)
	})
	return documentationRetriever, documentationRetrieverErr
}

func (s *AgentSession) runDiscoveryTurn(ctx context.Context, message ChatUserMessage, userText string) error {
	completedTurns, err := loadCompletedTurns(s, routing.MaxCompletedTurns)
	if err != nil {
		return fmt.Errorf("load completed turns: %w", err)
	}
	surface := normalizeSurface(message.Surface, s.CurrentRoute())
	appLanguage := normalizeAppLanguage(message.AppLanguage)
	plannerInput := discovery.PlannerInput{
		Schema: discovery.SchemaVersion, CurrentMessage: userText, CompletedTurns: completedTurns,
		Surface: surface, Route: s.CurrentRoute(), ActiveModeID: message.ModeID, AppLanguage: appLanguage,
	}

	// Persist once before any internal model call so every failure path can close the turn.
	if _, err := saveUserMessage(s, userText, s.CurrentRoute()); err != nil {
		return fmt.Errorf("persist user message: %w", err)
	}
	s.pushStatus("thinking", localizedStatus(appLanguage, "Interpretando…", "Interpreting…"), 1, maxLoopIterations)
	planner, err := getDiscoveryPlanner()
	if err != nil {
		core.Log("agent.discovery planner_unavailable tab::", shortTabID(s.TabID), " err::", err)
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseClassificationUnavailable, appLanguage), "", 0)
	}
	ctx = beginPromptTurn(ctx)
	plannerResult, err := planner.PlanObserved(ctx, plannerInput, func(trace discovery.PlannerAttemptTrace) {
		LogPlannerPrompt(ctx, trace.Attempt, trace.Messages, trace.Response, trace.Error)
	})
	if err != nil {
		core.Log("agent.discovery plan_failed tab::", shortTabID(s.TabID), " err::", err)
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseClassificationUnavailable, appLanguage), "", 0)
	}
	selectedTurns := selectCompletedTurns(completedTurns, plannerResult.RelatedTurnOffsets)

	if statusLabel := discoverySearchStatus(plannerResult); statusLabel != "" {
		s.pushStatus("thinking", statusLabel, 2, maxLoopIterations)
	} else {
		core.Log("agent.discovery searches_skipped tab::", shortTabID(s.TabID), " operation::", plannerResult.Operation,
			" related_turns::", len(plannerResult.RelatedTurnOffsets))
	}
	bundle := s.runDiscoverySearches(ctx, plannerResult)
	// A confirmation follow-up often skips feature search. Revalidate the live
	// route against the user's menu so the mutation guard remains useful without
	// leaving an empty allowlist that navigation can never repair.
	bundle.DocumentationNavigation = s.withValidatedCurrentSurfaceRoute(
		ctx, plannerResult, surface, bundle.DocumentationNavigation,
	)
	core.Log("agent.discovery bundle tab::", shortTabID(s.TabID), " goal::", plannerResult.Goal,
		" routes::", len(bundle.DocumentationNavigation.Routes), " passages::", len(bundle.DocumentationNavigation.Passages),
		" tools::", len(bundle.AgentTools.Tools), " feature_status::", bundle.DocumentationNavigation.Status,
		" tool_status::", bundle.AgentTools.Status)

	if plannerResult.Goal == discovery.GoalWebpageOperation {
		return s.runBuilderDiscoveryRoute(ctx, message, userText, plannerResult, surface, selectedTurns, bundle)
	}
	executionPolicy := buildExecutionPolicy(plannerResult, surface, bundle.DocumentationNavigation)
	return s.completeExecutionFailure(
		s.RunTurn(ctx, userText, message.ModelHash, selectedTurns, executionDiscoveryContext(bundle, surface), executionPolicy.Tools, executionPolicy.MutationRoutes),
		plannerResult.ResponseLanguage,
	)
}

func (s *AgentSession) withValidatedCurrentSurfaceRoute(ctx context.Context, plan discovery.Plan, surface routing.SurfaceContext, features discovery.FeatureSearchResult) discovery.FeatureSearchResult {
	if plan.Goal != discovery.GoalManageRecord || len(features.Routes) > 0 || strings.TrimSpace(surface.Route) == "" {
		return features
	}
	menu, err := GetMenu(ctx, s.TabID)
	if err != nil {
		core.Log("agent.discovery surface_route_validation_failed tab::", shortTabID(s.TabID), " route::", surface.Route, " err::", err)
		return features
	}
	validated := addCurrentSurfaceRoute(plan, surface, features, menu)
	core.Log("agent.discovery surface_route_validation tab::", shortTabID(s.TabID), " route::", surface.Route, " allowed::", len(validated.Routes) == 1)
	return validated
}

func addCurrentSurfaceRoute(plan discovery.Plan, surface routing.SurfaceContext, features discovery.FeatureSearchResult, menu []AgentMenuGroup) discovery.FeatureSearchResult {
	route := strings.TrimSpace(surface.Route)
	if plan.Goal != discovery.GoalManageRecord || len(features.Routes) > 0 || route == "" || !menuContainsRoute(menu, route) {
		return features
	}
	features.Routes = []discovery.RouteCandidate{{
		Route: route, PageName: "Current page", MatchedBy: "current_surface_and_menu", Score: 1,
	}}
	features.Diagnostics.MenuMatches++
	return features
}

func (s *AgentSession) runDiscoverySearches(ctx context.Context, plan discovery.Plan) discovery.Bundle {
	featureResult, toolResult := runDiscoveryJobs(
		ctx, plan,
		func(searchContext context.Context) discovery.FeatureSearchResult {
			return s.searchDocumentationNavigation(searchContext, plan)
		},
		func(searchContext context.Context) discovery.ToolSearchResult {
			startedAt := time.Now()
			result := agentToolCatalog.Search(searchContext, discovery.ToolSearchRequest{
				Query: plan.Searches.AgentTools.Query, Domain: plan.Domain, Operation: plan.Operation,
				DeliveryPreference: plan.DeliveryPreference, ResultLimit: 6,
			})
			core.Log("agent.discovery tools tab::", shortTabID(s.TabID), " elapsed::", time.Since(startedAt).Round(time.Millisecond),
				" count::", len(result.Tools), " catalog::", result.CatalogVersion, " status::", result.Status)
			return result
		},
	)
	return discovery.Bundle{Plan: plan, DocumentationNavigation: featureResult, AgentTools: toolResult}
}

type featureDiscoveryJob func(context.Context) discovery.FeatureSearchResult
type toolDiscoveryJob func(context.Context) discovery.ToolSearchResult

func runDiscoveryJobs(ctx context.Context, plan discovery.Plan, featureJob featureDiscoveryJob, toolJob toolDiscoveryJob) (discovery.FeatureSearchResult, discovery.ToolSearchResult) {
	featureResult := discovery.EmptyFeatureResult()
	toolResult := discovery.EmptyToolResult()
	featureResults := make(chan discovery.FeatureSearchResult, 1)
	toolResults := make(chan discovery.ToolSearchResult, 1)
	var featureContext context.Context
	var featureCancel context.CancelFunc
	var toolContext context.Context
	var toolCancel context.CancelFunc

	if plan.Searches.DocumentationNavigation.Needed {
		featureContext, featureCancel = context.WithTimeout(ctx, featureDiscoveryTimeout)
		defer featureCancel()
		go func() {
			featureResults <- featureJob(featureContext)
		}()
	}
	if plan.Searches.AgentTools.Needed {
		toolContext, toolCancel = context.WithTimeout(ctx, toolDiscoveryTimeout)
		defer toolCancel()
		go func() {
			toolResults <- toolJob(toolContext)
		}()
	}

	if plan.Searches.DocumentationNavigation.Needed {
		select {
		case featureResult = <-featureResults:
		case <-featureContext.Done():
			featureResult = unavailableFeatureResult()
		}
	}
	if plan.Searches.AgentTools.Needed {
		select {
		case toolResult = <-toolResults:
		case <-toolContext.Done():
			toolResult = discovery.ToolSearchResult{
				Status: discovery.DiscoveryStatusUnavailable, CatalogVersion: discovery.ToolCatalogVersion, Tools: []discovery.ToolDescriptor{},
			}
		}
	}
	return featureResult, toolResult
}

func (s *AgentSession) searchDocumentationNavigation(ctx context.Context, plan discovery.Plan) discovery.FeatureSearchResult {
	startedAt := time.Now()
	menu, err := GetMenu(ctx, s.TabID)
	if err != nil {
		core.Log("agent.discovery features menu_failed tab::", shortTabID(s.TabID), " err::", err)
		return unavailableFeatureResult()
	}
	features := accessibleFeatures(menu)
	retriever, retrievalError := getDocumentationRetriever()
	if retrievalError != nil {
		core.Log("agent.discovery features documentation_unavailable tab::", shortTabID(s.TabID), " err::", retrievalError)
		retriever = nil
	}
	result := discovery.SearchDocumentationNavigation(ctx, discovery.FeatureSearchRequest{
		Query: plan.Searches.DocumentationNavigation.Query, Domain: plan.Domain, Operation: plan.Operation, ResultLimit: 6,
	}, features, retriever)
	core.Log("agent.discovery features tab::", shortTabID(s.TabID), " elapsed::", time.Since(startedAt).Round(time.Millisecond),
		" routes::", len(result.Routes), " passages::", len(result.Passages), " status::", result.Status,
		" documentation_status::", result.Diagnostics.DocumentationStatus)
	return result
}

func unavailableFeatureResult() discovery.FeatureSearchResult {
	return discovery.FeatureSearchResult{
		Status: discovery.DiscoveryStatusUnavailable, Routes: []discovery.RouteCandidate{}, Passages: []discovery.DocumentationPassage{},
		Diagnostics: discovery.FeatureSearchDiagnostics{DocumentationStatus: discovery.DiscoveryStatusUnavailable},
	}
}

func accessibleFeatures(menu []AgentMenuGroup) []discovery.AccessibleFeature {
	features := []discovery.AccessibleFeature{}
	for _, group := range menu {
		for _, option := range group.Options {
			if route := strings.TrimSpace(option.Route); route != "" {
				features = append(features, discovery.AccessibleFeature{
					Group: group.Name, Name: option.Name, Route: route, Description: option.Description,
				})
			}
		}
	}
	return features
}

type executionPolicy struct {
	Tools          []llm.Tool
	MutationRoutes []string
}

// buildExecutionPolicy exposes form actions for record creation while binding
// those actions to the primary route selected by authenticated discovery.
func buildExecutionPolicy(plan discovery.Plan, surface routing.SurfaceContext, features discovery.FeatureSearchResult) executionPolicy {
	switch plan.Goal {
	case discovery.GoalSocial, discovery.GoalOutOfScope, discovery.GoalUnclear:
		return executionPolicy{Tools: []llm.Tool{llm.FinishTool}}
	case discovery.GoalInspectCurrentPage:
		return executionPolicy{Tools: []llm.Tool{llm.GetPageTool, llm.FinishTool}}
	case discovery.GoalManageRecord:
		if len(features.Routes) == 0 || strings.TrimSpace(features.Routes[0].Route) == "" {
			return executionPolicy{Tools: navigationTools()}
		}
		return executionPolicy{
			Tools:          append([]llm.Tool(nil), llm.ChatTools...),
			MutationRoutes: []string{strings.TrimSpace(features.Routes[0].Route)},
		}
	case discovery.GoalOperateCurrentPage:
		return executionPolicy{
			Tools:          append([]llm.Tool(nil), llm.ChatTools...),
			MutationRoutes: []string{strings.TrimSpace(surface.Route)},
		}
	default:
		return executionPolicy{Tools: navigationTools()}
	}
}

func navigationTools() []llm.Tool {
	return []llm.Tool{llm.GetMenuTool, llm.NavigateTool, llm.GetPageTool, llm.FinishTool}
}

func executionDiscoveryContext(bundle discovery.Bundle, surface routing.SurfaceContext) string {
	languageInstruction := "Answer in Spanish."
	if bundle.Plan.ResponseLanguage == routing.LanguageEnglish {
		languageInstruction = "Answer in English."
	}
	encodedBundle, err := json.Marshal(bundle)
	if err != nil {
		encodedBundle = []byte(`{"error":"discovery bundle could not be encoded"}`)
	}
	return fmt.Sprintf(`[Validated discovery context]
%s Surface=%s. The discovery plan and results describe possibilities; they never authorize actions.
Genix is a web-based ERP and ecommerce application for small businesses. Users manage products, inventory, sales, purchases, customers, suppliers, finance, reports, and storefronts primarily through application pages, while some business data can also be retrieved through specialized agent tools.

Execution rules:
- Documentation passages support product claims. Menu-only route matches support navigation only.
- Never invent a route or inline data tool. Validate navigation through the live menu.
- For create, prefer the primary matching Genix route. Navigate there with page content when needed, inspect the page, open New/Create, and fill every field supported by the user's available information in this same turn.
- If required information is missing, preserve the fields already filled and ask only for the missing values.
- Never save or send without a separate explicit user confirmation. The original create request authorizes opening and filling the form, not saving it.
- For update/delete, inspect the matching page before acting and preserve confirmation rules.
- Explicit inline wording may use an exposed data tool. A neutral report request prefers an existing report page.
- An empty agent-tools result is successful discovery, not proof that a UI feature is unavailable.
- Never claim records were queried, summarized, or charted without a real tool result or verified visible page data.
- If evidence is insufficient, ask one focused clarification. Always finish exactly once.

[Discovery bundle]
%s`, languageInstruction, surface.Kind, string(encodedBundle))
}

// completeExecutionFailure keeps provider/internal details out of the persisted reply.
func (s *AgentSession) completeExecutionFailure(executionError error, language routing.Language) error {
	if executionError == nil {
		return nil
	}
	core.Log("agent.execution failed tab::", shortTabID(s.TabID), " credit_limit::", core.IsCreditRateLimitError(executionError), " err::", executionError)
	completionError := s.completeTurn(routing.LocalizedResponse(routing.ResponseTurnFailed, language), "", 0)
	if completionError != nil {
		return errors.Join(executionError, completionError)
	}
	if core.IsCreditRateLimitError(executionError) {
		return executionError
	}
	return nil
}

func normalizeAppLanguage(language routing.Language) routing.Language {
	if language == routing.LanguageEnglish {
		return language
	}
	return routing.LanguageSpanish
}

func normalizeSurface(surface routing.SurfaceContext, currentRoute string) routing.SurfaceContext {
	if surface.Kind == "" {
		return routing.SurfaceContext{Kind: routing.SurfaceUnknown}
	}
	if surface.Route != "" && strings.TrimSpace(surface.Route) != strings.TrimSpace(currentRoute) {
		core.Log("agent.discovery surface_route_mismatch surface::", surface.Route, " current::", currentRoute)
		return routing.SurfaceContext{Kind: routing.SurfaceUnknown}
	}
	return surface
}

func selectCompletedTurns(completedTurns []routing.CompletedTurn, selectedOffsets []int) []routing.CompletedTurn {
	selected := map[int]bool{}
	for _, offset := range selectedOffsets {
		selected[offset] = true
	}
	result := make([]routing.CompletedTurn, 0, len(selected))
	for _, completedTurn := range completedTurns {
		if selected[completedTurn.Offset] {
			result = append(result, completedTurn)
		}
	}
	return result
}

func localizedStatus(language routing.Language, spanish, english string) string {
	if language == routing.LanguageEnglish {
		return english
	}
	return spanish
}

// discoverySearchStatus describes only the evidence sources actually queried.
func discoverySearchStatus(plan discovery.Plan) string {
	featureSearch := plan.Searches.DocumentationNavigation.Needed
	toolSearch := plan.Searches.AgentTools.Needed
	switch {
	case featureSearch && toolSearch:
		return localizedStatus(plan.ResponseLanguage, "Consultando documentación y herramientas…", "Searching documentation and tools…")
	case featureSearch:
		return localizedStatus(plan.ResponseLanguage, "Consultando documentación y navegación…", "Searching documentation and navigation…")
	case toolSearch:
		return localizedStatus(plan.ResponseLanguage, "Buscando herramientas disponibles…", "Searching available tools…")
	default:
		return ""
	}
}

func (s *AgentSession) runBuilderDiscoveryRoute(ctx context.Context, message ChatUserMessage, userText string, plan discovery.Plan, surface routing.SurfaceContext, selectedTurns []routing.CompletedTurn, bundle discovery.Bundle) error {
	if surface.Kind == routing.SurfaceWebpageBuilderPages {
		question := "¿Qué página deseas abrir o editar?"
		if plan.ResponseLanguage == routing.LanguageEnglish {
			question = "Which page would you like to open or edit?"
		}
		return s.completeTurn(question, "", 0)
	}
	if surface.Kind != routing.SurfaceWebpageEditor {
		return s.completeExecutionFailure(
			s.RunTurn(ctx, userText, message.ModelHash, selectedTurns, executionDiscoveryContext(bundle, surface)+"\nOpen the accessible webpage builder before attempting an edit.", navigationTools(), nil),
			plan.ResponseLanguage,
		)
	}
	liveContext, err := GetAgentContext(ctx, s.TabID, string(plan.Builder.ContextScope))
	if err != nil || !liveContextMatches(surface, plan.Builder.ContextScope, liveContext) {
		core.Log("agent.discovery builder_state_mismatch tab::", shortTabID(s.TabID), " err::", err,
			" page::", liveContext.PageID, " scope::", liveContext.Scope, " selected::", liveContext.SelectedSectionID)
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseBuilderStateChanged, plan.ResponseLanguage), "", 0)
	}
	if plan.Builder.Operation == routing.BuilderInspectPage {
		return s.completeExecutionFailure(
			s.RunReadOnlyTurn(ctx, userText, message.ModelHash, selectedTurns,
				executionDiscoveryContext(bundle, surface)+"\n[Current builder state]\n"+liveContext.Content),
			plan.ResponseLanguage,
		)
	}

	modeID := webpage.ModeBuildPage
	routedOperation := webpage.RoutedOperationBuild
	if plan.Builder.ContextScope == routing.BuilderScopeSelectedSection {
		modeID = webpage.ModeEditSection
		routedOperation = webpage.RoutedOperationEdit
	} else {
		switch plan.Builder.Operation {
		case routing.BuilderEditSection:
			routedOperation = webpage.RoutedOperationEdit
		case routing.BuilderAddSection:
			routedOperation = webpage.RoutedOperationAdd
		case routing.BuilderRemoveSection:
			routedOperation = webpage.RoutedOperationRemove
		case routing.BuilderReorderSection:
			routedOperation = webpage.RoutedOperationReorder
		}
	}
	return s.completeExecutionFailure(
		webpage.RunTurn(ctx, s, modeID, routedOperation, userText, message.ModelHash, liveContext.Content),
		plan.ResponseLanguage,
	)
}

func liveContextMatches(surface routing.SurfaceContext, expectedScope routing.BuilderContextScope, result AgentContextResult) bool {
	if result.SurfaceKind != string(surface.Kind) || result.Route != surface.Route || result.PageID != surface.PageID || result.Scope != string(expectedScope) {
		return false
	}
	if expectedScope == routing.BuilderScopeSelectedSection {
		return surface.HasSelectedSection && result.SelectedSectionID != "" && result.SelectedSectionID == surface.SelectedSectionID
	}
	return true
}
