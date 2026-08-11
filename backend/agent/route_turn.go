package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"app/agent/embedding"
	"app/agent/knowledge"
	"app/agent/llm"
	"app/agent/routing"
	"app/agent/webpage"
	"app/core"
)

const documentationEvidenceMaxBytes = 24_000

var (
	requestClassifierOnce sync.Once
	requestClassifier     *routing.Classifier
	requestClassifierErr  error

	documentationRetrieverOnce sync.Once
	documentationRetriever     *knowledge.Retriever
	documentationRetrieverErr  error
)

func getRequestClassifier() (*routing.Classifier, error) {
	requestClassifierOnce.Do(func() {
		chatClient, err := llm.NewClientForProvider(core.Env.CLASSIFIER_PROVIDER, core.Env.CLASSIFIER_MODEL_ID)
		if err != nil {
			requestClassifierErr = err
			return
		}
		requestClassifier, requestClassifierErr = routing.NewClassifier(chatClient, core.Env.CLASSIFIER_MODEL_ID)
	})
	return requestClassifier, requestClassifierErr
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

func (s *AgentSession) runClassifiedTurn(ctx context.Context, message ChatUserMessage, userText string) error {
	completedTurns, err := loadCompletedTurns(s, routing.MaxCompletedTurns)
	if err != nil {
		return fmt.Errorf("load completed turns: %w", err)
	}
	surface := normalizeSurface(message.Surface, s.CurrentRoute())
	appLanguage := normalizeAppLanguage(message.AppLanguage)
	classifierInput := routing.ClassifierInput{
		Schema: routing.SchemaVersion, CurrentMessage: userText, CompletedTurns: completedTurns,
		Surface: surface, Route: s.CurrentRoute(), ActiveModeID: message.ModeID,
		Capabilities: routing.CapabilitySnapshot(), AppLanguage: appLanguage,
	}

	// Load history first, then persist the current user exactly once before any route runs.
	if _, err := saveUserMessage(s, userText, s.CurrentRoute()); err != nil {
		return fmt.Errorf("persist user message: %w", err)
	}
	s.pushStatus("thinking", localizedStatus(appLanguage, "Interpretando…", "Interpreting…"), 1, maxLoopIterations)
	classifier, err := getRequestClassifier()
	if err != nil {
		core.Log("agent.router classifier_unavailable tab::", shortTabID(s.TabID), " err::", err)
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseClassificationUnavailable, appLanguage), "", 0)
	}
	verdict, err := classifier.Classify(ctx, classifierInput)
	if err != nil {
		core.Log("agent.router classification_failed tab::", shortTabID(s.TabID), " err::", err)
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseClassificationUnavailable, appLanguage), "", 0)
	}
	selectedTurns := selectCompletedTurns(completedTurns, verdict.RelatedTurnOffsets)
	core.Log("agent.router route tab::", shortTabID(s.TabID), " intent::", verdict.PrimaryIntent,
		" language::", verdict.ResponseLanguage, " selected_turns::", len(selectedTurns),
		" surface::", surface.Kind, " capabilities::", len(verdict.RequiredCapabilities))

	switch verdict.PrimaryIntent {
	case routing.IntentSocial:
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseSocial, verdict.ResponseLanguage), "", 0)
	case routing.IntentOutOfScope:
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseOutOfScope, verdict.ResponseLanguage), "", 0)
	case routing.IntentAmbiguous:
		return s.completeTurn(verdict.ClarificationQuestion, "", 0)
	case routing.IntentOperationalData:
		if unavailableCapability(verdict.RequiredCapabilities) != "" {
			return s.completeTurn(routing.LocalizedResponse(routing.ResponseOperationalUnavailable, verdict.ResponseLanguage), "", 0)
		}
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseOperationalUnavailable, verdict.ResponseLanguage), "", 0)
	case routing.IntentProductKnowledge:
		evidenceContext, retrievalError := s.retrieveDocumentationContext(ctx, verdict.StandaloneRequest)
		if retrievalError != nil {
			core.Log("agent.router documentation_unavailable tab::", shortTabID(s.TabID), " err::", retrievalError)
			return s.completeTurn(routing.LocalizedResponse(routing.ResponseDocumentationUnavailable, verdict.ResponseLanguage), "", 0)
		}
		if evidenceContext == "" {
			return s.completeTurn(routing.LocalizedResponse(routing.ResponseDocumentationMissing, verdict.ResponseLanguage), "", 0)
		}
		return s.completeExecutionFailure(
			s.RunTurn(ctx, userText, message.ModelHash, selectedTurns, routingPromptContext(verdict, surface)+evidenceContext, toolsForVerdict(verdict)),
			verdict.ResponseLanguage,
		)
	case routing.IntentNavigation:
		routingContext := routingPromptContext(verdict, surface)
		if hasCapability(verdict.RequiredCapabilities, routing.CapabilityDocumentationSearch) {
			evidenceContext, retrievalError := s.retrieveDocumentationContext(ctx, verdict.StandaloneRequest)
			switch {
			case retrievalError != nil:
				core.Log("agent.router navigation_documentation_unavailable tab::", shortTabID(s.TabID), " err::", retrievalError)
				routingContext += "\n[Documentation status] Documentation is unavailable. Use only the live accessible menu for navigation; do not claim detailed business rules."
			case evidenceContext == "":
				routingContext += "\n[Documentation status] No verified passage matched. Use only the live accessible menu for navigation; do not invent feature behavior."
			default:
				routingContext += evidenceContext
			}
		}
		return s.completeExecutionFailure(
			s.RunTurn(ctx, userText, message.ModelHash, selectedTurns, routingContext, toolsForVerdict(verdict)),
			verdict.ResponseLanguage,
		)
	case routing.IntentWebpageBuild, routing.IntentWebpageAddSection, routing.IntentWebpageEditSection,
		routing.IntentWebpageRemoveSection, routing.IntentWebpageReorder, routing.IntentWebpageInspect:
		return s.runBuilderRoute(ctx, message, userText, verdict, surface, selectedTurns)
	default:
		return s.completeExecutionFailure(
			s.RunTurn(ctx, userText, message.ModelHash, selectedTurns, routingPromptContext(verdict, surface), toolsForVerdict(verdict)),
			verdict.ResponseLanguage,
		)
	}
}

func toolsForVerdict(verdict routing.Verdict) []llm.Tool {
	if verdict.RequestedOperation == routing.OperationNavigate || hasIntent(verdict.SecondaryIntents, routing.IntentNavigation) {
		return navigationTools()
	}
	switch verdict.PrimaryIntent {
	case routing.IntentNavigation:
		return navigationTools()
	case routing.IntentCurrentPage:
		return []llm.Tool{llm.GetPageTool, llm.FinishTool}
	case routing.IntentPageAction, routing.IntentConfirmation:
		return append([]llm.Tool(nil), llm.ChatTools...)
	default:
		return []llm.Tool{llm.FinishTool}
	}
}

func navigationTools() []llm.Tool {
	return []llm.Tool{llm.GetMenuTool, llm.NavigateTool, llm.GetPageTool, llm.FinishTool}
}

func hasIntent(intents []routing.Intent, expected routing.Intent) bool {
	for _, intent := range intents {
		if intent == expected {
			return true
		}
	}
	return false
}

func hasCapability(capabilities []routing.CapabilityName, expected routing.CapabilityName) bool {
	for _, capability := range capabilities {
		if capability == expected {
			return true
		}
	}
	return false
}

// completeExecutionFailure keeps internal/provider errors out of the chat and
// closes every persisted user turn with one localized assistant response.
func (s *AgentSession) completeExecutionFailure(executionError error, language routing.Language) error {
	if executionError == nil {
		return nil
	}
	core.Log("agent.router downstream_failed tab::", shortTabID(s.TabID), " credit_limit::", core.IsCreditRateLimitError(executionError), " err::", executionError)
	completionError := s.completeTurn(routing.LocalizedResponse(routing.ResponseTurnFailed, language), "", 0)
	if completionError != nil {
		return errors.Join(executionError, completionError)
	}
	// Preserve the HTTP credit-limit contract after the user-safe reply is saved.
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
		core.Log("agent.router surface_route_mismatch surface::", surface.Route, " current::", currentRoute)
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

func unavailableCapability(capabilities []routing.CapabilityName) routing.CapabilityName {
	for _, capability := range capabilities {
		if !routing.CapabilityAvailable(capability) {
			return capability
		}
	}
	return ""
}

func (s *AgentSession) retrieveDocumentationContext(ctx context.Context, standaloneRequest string) (string, error) {
	menu, err := GetMenu(ctx, s.TabID)
	if err != nil {
		return "", fmt.Errorf("retrieve accessible menu: %w", err)
	}
	allowedRoutes := menuRoutes(menu)
	if len(allowedRoutes) == 0 {
		return "", errors.New("accessible menu contains no routes")
	}
	retriever, err := getDocumentationRetriever()
	if err != nil {
		return "", err
	}
	results, err := retriever.SearchDocumentation(ctx, standaloneRequest, knowledge.SearchOptions{
		CandidateLimit: 25, ResultLimit: 6, AllowedRoutes: allowedRoutes,
	})
	if err != nil {
		return "", err
	}
	return buildDocumentationEvidence(results), nil
}

func menuRoutes(menu []AgentMenuGroup) []string {
	routes := []string{}
	seen := map[string]bool{}
	for _, group := range menu {
		for _, option := range group.Options {
			route := strings.TrimSpace(option.Route)
			if route != "" && !seen[route] {
				seen[route] = true
				routes = append(routes, route)
			}
		}
	}
	return routes
}

func buildDocumentationEvidence(results []knowledge.DocumentationResult) string {
	if len(results) == 0 {
		return ""
	}
	var evidence strings.Builder
	evidence.WriteString("\n\n[Verified Genix documentation]\nUse these passages for product claims. Cite the page and route in the answer. Do not infer unsupported rules.\n")
	for _, result := range results {
		passage := fmt.Sprintf("\n[CITATION %s]\nPage: %s\nRoute: %s\nSection: %s\n%s\n",
			result.CitationID, result.PageTitle, result.Route, result.SectionTitle, strings.TrimSpace(result.Content))
		if evidence.Len()+len(passage) > documentationEvidenceMaxBytes {
			break
		}
		evidence.WriteString(passage)
	}
	return evidence.String()
}

func routingPromptContext(verdict routing.Verdict, surface routing.SurfaceContext) string {
	languageInstruction := "Answer in Spanish."
	if verdict.ResponseLanguage == routing.LanguageEnglish {
		languageInstruction = "Answer in English."
	}
	return fmt.Sprintf("[Validated routing context]\n%s Intent=%s. Requested operation=%s. Surface=%s. The classifier does not authorize actions.",
		languageInstruction, verdict.PrimaryIntent, verdict.RequestedOperation, surface.Kind)
}

func localizedStatus(language routing.Language, spanish, english string) string {
	if language == routing.LanguageEnglish {
		return english
	}
	return spanish
}

func (s *AgentSession) runBuilderRoute(ctx context.Context, message ChatUserMessage, userText string, verdict routing.Verdict, surface routing.SurfaceContext, selectedTurns []routing.CompletedTurn) error {
	if surface.Kind == routing.SurfaceWebpageBuilderPages {
		question := "¿Qué página deseas abrir o editar?"
		if verdict.ResponseLanguage == routing.LanguageEnglish {
			question = "Which page would you like to open or edit?"
		}
		return s.completeTurn(question, "", 0)
	}
	if surface.Kind != routing.SurfaceWebpageEditor {
		return s.completeExecutionFailure(
			s.RunTurn(ctx, userText, message.ModelHash, selectedTurns, routingPromptContext(verdict, surface)+" Open the accessible webpage builder before attempting an edit.", navigationTools()),
			verdict.ResponseLanguage,
		)
	}
	liveContext, err := GetAgentContext(ctx, s.TabID, string(verdict.Builder.ContextScope))
	if err != nil {
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseBuilderStateChanged, verdict.ResponseLanguage), "", 0)
	}
	if !liveContextMatches(surface, verdict.Builder.ContextScope, liveContext) {
		core.Log("agent.router builder_state_mismatch tab::", shortTabID(s.TabID), " page::", liveContext.PageID,
			" scope::", liveContext.Scope, " selected::", liveContext.SelectedSectionID)
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseBuilderStateChanged, verdict.ResponseLanguage), "", 0)
	}
	if verdict.PrimaryIntent == routing.IntentWebpageInspect {
		return s.completeExecutionFailure(
			s.RunReadOnlyTurn(ctx, userText, message.ModelHash, selectedTurns,
				routingPromptContext(verdict, surface)+"\n[Current builder state]\n"+liveContext.Content),
			verdict.ResponseLanguage,
		)
	}
	modeID := webpage.ModeBuildPage
	routedOperation := webpage.RoutedOperationBuild
	if verdict.Builder.ContextScope == routing.BuilderScopeSelectedSection {
		modeID = webpage.ModeEditSection
		routedOperation = webpage.RoutedOperationEdit
	} else {
		switch verdict.PrimaryIntent {
		case routing.IntentWebpageEditSection:
			routedOperation = webpage.RoutedOperationEdit
		case routing.IntentWebpageAddSection:
			routedOperation = webpage.RoutedOperationAdd
		case routing.IntentWebpageRemoveSection:
			routedOperation = webpage.RoutedOperationRemove
		case routing.IntentWebpageReorder:
			routedOperation = webpage.RoutedOperationReorder
		}
	}
	return s.completeExecutionFailure(
		webpage.RunTurn(ctx, s, modeID, routedOperation, userText, message.ModelHash, liveContext.Content),
		verdict.ResponseLanguage,
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
