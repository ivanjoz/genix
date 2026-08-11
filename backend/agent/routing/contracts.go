// Package routing owns the untrusted global-classifier contract and its validation.
// Execution stays in package agent so a model verdict can never authorize an action.
package routing

import (
	"fmt"
	"strings"
)

const (
	SchemaVersion         = 1
	MaxCompletedTurns     = 5
	MaxMessageBytes       = 4_000
	MaxStandaloneBytes    = 4_000
	MaxClarificationBytes = 500
)

type Language string

const (
	LanguageSpanish Language = "es"
	LanguageEnglish Language = "en"
	LanguageMixed   Language = "mixed"
	LanguageUnknown Language = "unknown"
)

type Scope string

const (
	ScopeGenix      Scope = "genix"
	ScopeOutOfScope Scope = "out_of_scope"
	ScopeUnclear    Scope = "unclear"
)

type Intent string

const (
	IntentSocial               Intent = "social"
	IntentProductKnowledge     Intent = "product_knowledge"
	IntentOperationalData      Intent = "operational_data"
	IntentCurrentPage          Intent = "current_page"
	IntentNavigation           Intent = "navigation"
	IntentPageAction           Intent = "page_action"
	IntentConfirmation         Intent = "confirmation"
	IntentWebpageBuild         Intent = "webpage_build"
	IntentWebpageAddSection    Intent = "webpage_add_section"
	IntentWebpageEditSection   Intent = "webpage_edit_section"
	IntentWebpageRemoveSection Intent = "webpage_remove_section"
	IntentWebpageReorder       Intent = "webpage_reorder_section"
	IntentWebpageInspect       Intent = "webpage_inspect"
	IntentOutOfScope           Intent = "out_of_scope"
	IntentAmbiguous            Intent = "ambiguous"
)

type RequestedOperation string

const (
	OperationRead     RequestedOperation = "read"
	OperationNavigate RequestedOperation = "navigate"
	OperationCreate   RequestedOperation = "create"
	OperationUpdate   RequestedOperation = "update"
	OperationDelete   RequestedOperation = "delete"
	OperationConfirm  RequestedOperation = "confirm"
	OperationReject   RequestedOperation = "reject"
	OperationNone     RequestedOperation = "none"
)

type BuilderOperation string

const (
	BuilderNone           BuilderOperation = "none"
	BuilderBuildPage      BuilderOperation = "build_page"
	BuilderAddSection     BuilderOperation = "add_section"
	BuilderEditSection    BuilderOperation = "edit_section"
	BuilderRemoveSection  BuilderOperation = "remove_section"
	BuilderReorderSection BuilderOperation = "reorder_section"
	BuilderInspectPage    BuilderOperation = "inspect_page"
)

type BuilderContextScope string

const (
	BuilderScopeNone            BuilderContextScope = "none"
	BuilderScopeFullPage        BuilderContextScope = "full_page"
	BuilderScopeSelectedSection BuilderContextScope = "selected_section"
)

type SurfaceKind string

const (
	SurfaceERPPage             SurfaceKind = "erp_page"
	SurfaceWebpageBuilderPages SurfaceKind = "webpage_builder_pages"
	SurfaceWebpageEditor       SurfaceKind = "webpage_builder_editor"
	SurfaceStorefrontPreview   SurfaceKind = "ecommerce_storefront_preview"
	SurfaceUnknown             SurfaceKind = "unknown"
)

type CapabilityName string

const (
	CapabilityDocumentationSearch CapabilityName = "documentation_search"
	CapabilityCurrentPage         CapabilityName = "current_page"
	CapabilityMenu                CapabilityName = "menu"
	CapabilityBrowserAction       CapabilityName = "browser_action"
	CapabilityWebpageBuilder      CapabilityName = "webpage_builder"
	CapabilitySalesSearch         CapabilityName = "sales_search"
	CapabilityPurchaseSearch      CapabilityName = "purchase_search"
	CapabilityInventorySearch     CapabilityName = "inventory_search"
	CapabilityFinanceSearch       CapabilityName = "finance_search"
	CapabilityCustomerSearch      CapabilityName = "customer_search"
	CapabilitySupplierSearch      CapabilityName = "supplier_search"
)

// CompletedTurn is one complete user/assistant pair. Offset -1 is newest.
type CompletedTurn struct {
	Offset           int    `json:"offset"`
	UserMessage      string `json:"user_message"`
	AssistantMessage string `json:"assistant_message"`
	ActionSummary    string `json:"action_summary,omitempty"`
	Route            string `json:"route,omitempty"`
}

// SurfaceContext is compact routing metadata supplied by the active frontend page.
type SurfaceContext struct {
	Kind                SurfaceKind           `json:"kind"`
	Route               string                `json:"route,omitempty"`
	PageID              string                `json:"page_id,omitempty"`
	ActiveAgentMode     string                `json:"active_agent_mode,omitempty"`
	HasSelectedSection  bool                  `json:"has_selected_section"`
	SelectedSectionID   string                `json:"selected_section_id,omitempty"`
	SelectedSectionType string                `json:"selected_section_type,omitempty"`
	AvailableContexts   []BuilderContextScope `json:"available_contexts,omitempty"`
}

type CapabilityState struct {
	Name      CapabilityName `json:"name"`
	Available bool           `json:"available"`
}

// ClassifierInput contains no documents, page HTML, access lists, or records.
type ClassifierInput struct {
	Schema         int               `json:"schema"`
	CurrentMessage string            `json:"current_message"`
	CompletedTurns []CompletedTurn   `json:"completed_turns,omitempty"`
	Surface        SurfaceContext    `json:"surface"`
	Route          string            `json:"route,omitempty"`
	ActiveModeID   int               `json:"active_mode_id"`
	Capabilities   []CapabilityState `json:"capabilities"`
	AppLanguage    Language          `json:"app_language"`
}

type EntityHint struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type BuilderDecision struct {
	Operation              BuilderOperation    `json:"operation"`
	ContextScope           BuilderContextScope `json:"context_scope"`
	TargetSectionType      string              `json:"target_section_type,omitempty"`
	TargetSectionReference string              `json:"target_section_reference,omitempty"`
	RequiresLiveState      bool                `json:"requires_live_state"`
}

// Verdict is routing data only; it never contains a user-facing answer.
type Verdict struct {
	Schema                int                `json:"schema"`
	Language              Language           `json:"language"`
	ResponseLanguage      Language           `json:"response_language"`
	Scope                 Scope              `json:"scope"`
	PrimaryIntent         Intent             `json:"primary_intent"`
	SecondaryIntents      []Intent           `json:"secondary_intents,omitempty"`
	RequestedOperation    RequestedOperation `json:"requested_operation"`
	RelatedTurnOffsets    []int              `json:"related_turn_offsets,omitempty"`
	StandaloneRequest     string             `json:"standalone_request"`
	RequiredCapabilities  []CapabilityName   `json:"required_capabilities,omitempty"`
	BusinessDomain        string             `json:"business_domain,omitempty"`
	Entities              []EntityHint       `json:"entities,omitempty"`
	Builder               BuilderDecision    `json:"builder"`
	NeedsClarification    bool               `json:"needs_clarification"`
	ClarificationQuestion string             `json:"clarification_question,omitempty"`
}

func (input ClassifierInput) Validate() error {
	if input.Schema != SchemaVersion {
		return fmt.Errorf("unsupported classifier input schema %d", input.Schema)
	}
	if err := boundedRequired("current_message", input.CurrentMessage, MaxMessageBytes); err != nil {
		return err
	}
	if len(input.CompletedTurns) > MaxCompletedTurns {
		return fmt.Errorf("completed_turns exceeds %d", MaxCompletedTurns)
	}
	seenOffsets := map[int]bool{}
	for _, completedTurn := range input.CompletedTurns {
		if completedTurn.Offset < -MaxCompletedTurns || completedTurn.Offset > -1 || seenOffsets[completedTurn.Offset] {
			return fmt.Errorf("invalid or duplicate completed-turn offset %d", completedTurn.Offset)
		}
		seenOffsets[completedTurn.Offset] = true
		if err := boundedRequired("completed_turn.user_message", completedTurn.UserMessage, MaxMessageBytes); err != nil {
			return err
		}
		if err := boundedRequired("completed_turn.assistant_message", completedTurn.AssistantMessage, MaxMessageBytes*2); err != nil {
			return err
		}
	}
	if !validLanguage(input.AppLanguage) {
		return fmt.Errorf("invalid app_language %q", input.AppLanguage)
	}
	if !validSurfaceKind(input.Surface.Kind) {
		return fmt.Errorf("invalid surface kind %q", input.Surface.Kind)
	}
	if err := validateSurface(input.Surface); err != nil {
		return err
	}
	for _, capability := range input.Capabilities {
		if !KnownCapability(capability.Name) {
			return fmt.Errorf("unknown capability %q", capability.Name)
		}
	}
	return nil
}

// Validate checks every classifier-controlled field against input the backend supplied.
func (verdict Verdict) Validate(input ClassifierInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	if verdict.Schema != SchemaVersion || !validLanguage(verdict.Language) {
		return fmt.Errorf("invalid verdict schema or language")
	}
	if verdict.ResponseLanguage != LanguageSpanish && verdict.ResponseLanguage != LanguageEnglish {
		return fmt.Errorf("invalid response_language %q", verdict.ResponseLanguage)
	}
	if !validScope(verdict.Scope) || !validIntent(verdict.PrimaryIntent) || !validOperation(verdict.RequestedOperation) {
		return fmt.Errorf("invalid scope, primary_intent, or requested_operation")
	}
	if len(verdict.StandaloneRequest) > MaxStandaloneBytes {
		return fmt.Errorf("standalone_request exceeds %d bytes", MaxStandaloneBytes)
	}
	if (verdict.PrimaryIntent == IntentProductKnowledge || verdict.PrimaryIntent == IntentOperationalData) && strings.TrimSpace(verdict.StandaloneRequest) == "" {
		return fmt.Errorf("standalone_request is required for %s", verdict.PrimaryIntent)
	}
	if verdict.NeedsClarification {
		if err := boundedRequired("clarification_question", verdict.ClarificationQuestion, MaxClarificationBytes); err != nil {
			return err
		}
	} else if strings.TrimSpace(verdict.ClarificationQuestion) != "" {
		return fmt.Errorf("clarification_question requires needs_clarification")
	}
	if err := validateUniqueIntents(verdict); err != nil {
		return err
	}
	if err := validateRelatedOffsets(verdict.RelatedTurnOffsets, input.CompletedTurns); err != nil {
		return err
	}
	if err := validateCapabilities(verdict); err != nil {
		return err
	}
	if err := validateScopeIntent(verdict); err != nil {
		return err
	}
	return validateBuilderDecision(verdict, input.Surface)
}

func validateUniqueIntents(verdict Verdict) error {
	seen := map[Intent]bool{verdict.PrimaryIntent: true}
	for _, intent := range verdict.SecondaryIntents {
		if !validIntent(intent) || seen[intent] {
			return fmt.Errorf("invalid or duplicate secondary intent %q", intent)
		}
		seen[intent] = true
	}
	return nil
}

func validateRelatedOffsets(selected []int, supplied []CompletedTurn) error {
	available := map[int]bool{}
	for _, completedTurn := range supplied {
		available[completedTurn.Offset] = true
	}
	seen := map[int]bool{}
	for _, offset := range selected {
		if !available[offset] || seen[offset] {
			return fmt.Errorf("related turn offset %d was not supplied or is duplicated", offset)
		}
		seen[offset] = true
	}
	return nil
}

func validateCapabilities(verdict Verdict) error {
	seen := map[CapabilityName]bool{}
	for _, capability := range verdict.RequiredCapabilities {
		if !KnownCapability(capability) || seen[capability] {
			return fmt.Errorf("unknown or duplicate required capability %q", capability)
		}
		seen[capability] = true
	}
	if verdict.PrimaryIntent == IntentProductKnowledge && !seen[CapabilityDocumentationSearch] {
		return fmt.Errorf("product_knowledge requires documentation_search")
	}
	if seen[CapabilityDocumentationSearch] && strings.TrimSpace(verdict.StandaloneRequest) == "" {
		return fmt.Errorf("documentation_search requires standalone_request")
	}
	if verdict.PrimaryIntent == IntentOperationalData {
		operationalCapabilities := []CapabilityName{CapabilitySalesSearch, CapabilityPurchaseSearch, CapabilityInventorySearch, CapabilityFinanceSearch, CapabilityCustomerSearch, CapabilitySupplierSearch}
		hasOperationalSearch := false
		for _, capability := range operationalCapabilities {
			hasOperationalSearch = hasOperationalSearch || seen[capability]
		}
		if !hasOperationalSearch {
			return fmt.Errorf("operational_data requires a domain search capability")
		}
	}
	return nil
}

func validateSurface(surface SurfaceContext) error {
	if !surface.HasSelectedSection && (strings.TrimSpace(surface.SelectedSectionID) != "" || strings.TrimSpace(surface.SelectedSectionType) != "") {
		return fmt.Errorf("surface selection metadata requires has_selected_section")
	}
	seenScopes := map[BuilderContextScope]bool{}
	for _, contextScope := range surface.AvailableContexts {
		if contextScope != BuilderScopeFullPage && contextScope != BuilderScopeSelectedSection {
			return fmt.Errorf("invalid surface context scope %q", contextScope)
		}
		if seenScopes[contextScope] {
			return fmt.Errorf("duplicate surface context scope %q", contextScope)
		}
		seenScopes[contextScope] = true
	}
	if seenScopes[BuilderScopeSelectedSection] && !surface.HasSelectedSection {
		return fmt.Errorf("selected_section context requires a selected section")
	}
	return nil
}

func validateScopeIntent(verdict Verdict) error {
	if verdict.PrimaryIntent == IntentOutOfScope {
		if verdict.Scope != ScopeOutOfScope {
			return fmt.Errorf("out_of_scope intent requires out_of_scope scope")
		}
		return nil
	}
	if verdict.PrimaryIntent == IntentAmbiguous {
		if verdict.Scope != ScopeUnclear || !verdict.NeedsClarification {
			return fmt.Errorf("ambiguous intent requires unclear scope and clarification")
		}
		return nil
	}
	if verdict.Scope != ScopeGenix {
		return fmt.Errorf("intent %s requires genix scope", verdict.PrimaryIntent)
	}
	return nil
}

func validateBuilderDecision(verdict Verdict, surface SurfaceContext) error {
	intentOperation := map[Intent]BuilderOperation{
		IntentWebpageBuild: BuilderBuildPage, IntentWebpageAddSection: BuilderAddSection,
		IntentWebpageEditSection: BuilderEditSection, IntentWebpageRemoveSection: BuilderRemoveSection,
		IntentWebpageReorder: BuilderReorderSection, IntentWebpageInspect: BuilderInspectPage,
	}
	expectedOperation, isBuilderIntent := intentOperation[verdict.PrimaryIntent]
	if !isBuilderIntent {
		if verdict.Builder.Operation != BuilderNone || verdict.Builder.ContextScope != BuilderScopeNone || verdict.Builder.RequiresLiveState {
			return fmt.Errorf("non-builder intent contains builder routing")
		}
		return nil
	}
	if verdict.Builder.Operation != expectedOperation {
		return fmt.Errorf("intent %s requires builder operation %s", verdict.PrimaryIntent, expectedOperation)
	}
	if expectedOperation == BuilderBuildPage && surface.Kind == SurfaceWebpageBuilderPages {
		if verdict.Builder.ContextScope != BuilderScopeNone || verdict.Builder.RequiresLiveState {
			return fmt.Errorf("builder page-list build must clarify a target before live state")
		}
		return nil
	}
	if !verdict.Builder.RequiresLiveState {
		return fmt.Errorf("builder operation %s requires live state", expectedOperation)
	}
	if expectedOperation == BuilderEditSection {
		if verdict.Builder.ContextScope != BuilderScopeSelectedSection && verdict.Builder.ContextScope != BuilderScopeFullPage {
			return fmt.Errorf("edit_section requires selected_section or full_page")
		}
		if verdict.Builder.ContextScope == BuilderScopeSelectedSection && (!surface.HasSelectedSection || surface.SelectedSectionID == "") {
			return fmt.Errorf("selected_section edit requires matching surface selection")
		}
		return nil
	}
	if verdict.Builder.ContextScope != BuilderScopeFullPage {
		return fmt.Errorf("builder operation %s requires full_page", expectedOperation)
	}
	return nil
}

func boundedRequired(fieldName, value string, maximumBytes int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	if len(trimmed) > maximumBytes {
		return fmt.Errorf("%s exceeds %d bytes", fieldName, maximumBytes)
	}
	return nil
}

func validLanguage(value Language) bool {
	return value == LanguageSpanish || value == LanguageEnglish || value == LanguageMixed || value == LanguageUnknown
}

func validScope(value Scope) bool {
	return value == ScopeGenix || value == ScopeOutOfScope || value == ScopeUnclear
}

func validIntent(value Intent) bool {
	switch value {
	case IntentSocial, IntentProductKnowledge, IntentOperationalData, IntentCurrentPage,
		IntentNavigation, IntentPageAction, IntentConfirmation, IntentWebpageBuild,
		IntentWebpageAddSection, IntentWebpageEditSection, IntentWebpageRemoveSection,
		IntentWebpageReorder, IntentWebpageInspect, IntentOutOfScope, IntentAmbiguous:
		return true
	default:
		return false
	}
}

func validOperation(value RequestedOperation) bool {
	switch value {
	case OperationRead, OperationNavigate, OperationCreate, OperationUpdate, OperationDelete,
		OperationConfirm, OperationReject, OperationNone:
		return true
	default:
		return false
	}
}

func validSurfaceKind(value SurfaceKind) bool {
	switch value {
	case SurfaceERPPage, SurfaceWebpageBuilderPages, SurfaceWebpageEditor, SurfaceStorefrontPreview, SurfaceUnknown:
		return true
	default:
		return false
	}
}
