// Package discovery owns the untrusted first-call plan and the read-only
// evidence bundle supplied to the execution agent.
package discovery

import (
	"fmt"
	"strings"

	"genix-agent/routing"
)

const (
	SchemaVersion       = 1
	MaxMessageBytes     = 4_000
	MaxStandaloneBytes  = 4_000
	MaxSearchQueryBytes = 4_000
)

type Scope string

const (
	ScopeGenix      Scope = "genix"
	ScopeOutOfScope Scope = "out_of_scope"
	ScopeUnclear    Scope = "unclear"
)

type Goal string

const (
	GoalSocial             Goal = "social"
	GoalExplainProduct     Goal = "explain_product"
	GoalManageRecord       Goal = "manage_record"
	GoalViewReport         Goal = "view_report"
	GoalQueryCompanyData   Goal = "query_company_data"
	GoalInspectCurrentPage Goal = "inspect_current_page"
	GoalOperateCurrentPage Goal = "operate_current_page"
	GoalWebpageOperation   Goal = "webpage_operation"
	GoalOutOfScope         Goal = "out_of_scope"
	GoalUnclear            Goal = "unclear"
)

type Operation string

const (
	OperationRead    Operation = "read"
	OperationCreate  Operation = "create"
	OperationUpdate  Operation = "update"
	OperationDelete  Operation = "delete"
	OperationConfirm Operation = "confirm"
	OperationReject  Operation = "reject"
	OperationNone    Operation = "none"
)

type DeliveryPreference string

const (
	DeliveryUI          DeliveryPreference = "ui"
	DeliveryInline      DeliveryPreference = "inline"
	DeliveryExplanation DeliveryPreference = "explanation"
	DeliveryUnspecified DeliveryPreference = "unspecified"
)

type SearchDecision struct {
	Needed bool   `json:"needed" cb:"needed,1"`
	Query  string `json:"query" cb:"query,2"`
}

type SearchPlan struct {
	DocumentationNavigation SearchDecision `json:"documentation_navigation" cb:"documentation_navigation,1"`
	AgentTools              SearchDecision `json:"agent_tools" cb:"agent_tools,2"`
}

type EntityHint struct {
	Type string `json:"type" cb:"type,1"`
	Name string `json:"name" cb:"name,2"`
}

type BuilderDecision struct {
	Operation              routing.BuilderOperation    `json:"operation" cb:"operation,1"`
	ContextScope           routing.BuilderContextScope `json:"context_scope" cb:"context_scope,2"`
	TargetSectionType      string                      `json:"target_section_type" cb:"target_section_type,3"`
	TargetSectionReference string                      `json:"target_section_reference" cb:"target_section_reference,4"`
	RequiresLiveState      bool                        `json:"requires_live_state" cb:"requires_live_state,5"`
}

type PlannerInput struct {
	Schema         int                     `json:"schema" cb:"schema,1"`
	CurrentMessage string                  `json:"current_message" cb:"current_message,2"`
	CompletedTurns []routing.CompletedTurn `json:"completed_turns" cb:"completed_turns,3"`
	Surface        routing.SurfaceContext  `json:"surface" cb:"surface,4"`
	Route          string                  `json:"route" cb:"route,5"`
	ActiveModeID   int                     `json:"active_mode_id" cb:"active_mode_id,6"`
	AppLanguage    routing.Language        `json:"app_language" cb:"app_language,7"`
}

type Plan struct {
	Schema                int                `json:"schema" cb:"schema,1"`
	Language              routing.Language   `json:"language" cb:"language,2"`
	ResponseLanguage      routing.Language   `json:"response_language" cb:"response_language,3"`
	Scope                 Scope              `json:"scope" cb:"scope,4"`
	Goal                  Goal               `json:"goal" cb:"goal,5"`
	Operation             Operation          `json:"operation" cb:"operation,6"`
	Domain                string             `json:"domain" cb:"domain,7"`
	Entities              []EntityHint       `json:"entities" cb:"entities,8"`
	DeliveryPreference    DeliveryPreference `json:"delivery_preference" cb:"delivery_preference,9"`
	RelatedTurnOffsets    []int              `json:"related_turn_offsets" cb:"related_turn_offsets,10"`
	StandaloneRequest     string             `json:"standalone_request" cb:"standalone_request,11"`
	Searches              SearchPlan         `json:"searches" cb:"searches,12"`
	Builder               BuilderDecision    `json:"builder" cb:"builder,13"`
	NeedsClarification    bool               `json:"needs_clarification" cb:"needs_clarification,14"`
	ClarificationQuestion string             `json:"clarification_question" cb:"clarification_question,15"`
}

func (input PlannerInput) Validate() error {
	if input.Schema != SchemaVersion {
		return fmt.Errorf("unsupported planner input schema %d", input.Schema)
	}
	if err := boundedRequired("current_message", input.CurrentMessage, MaxMessageBytes); err != nil {
		return err
	}
	if len(input.CompletedTurns) > routing.MaxCompletedTurns {
		return fmt.Errorf("completed_turns exceeds %d", routing.MaxCompletedTurns)
	}
	seenOffsets := map[int]bool{}
	for _, completedTurn := range input.CompletedTurns {
		if completedTurn.Offset < -routing.MaxCompletedTurns || completedTurn.Offset > -1 || seenOffsets[completedTurn.Offset] {
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
	if !validLanguage(input.AppLanguage) || !validSurface(input.Surface) {
		return fmt.Errorf("invalid app language or frontend surface")
	}
	return nil
}

func (plan Plan) Validate(input PlannerInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	if plan.Schema != SchemaVersion || !validLanguage(plan.Language) {
		return fmt.Errorf("invalid plan schema or language")
	}
	if plan.ResponseLanguage != routing.LanguageSpanish && plan.ResponseLanguage != routing.LanguageEnglish {
		return fmt.Errorf("invalid response_language %q", plan.ResponseLanguage)
	}
	if !validScope(plan.Scope) || !validGoal(plan.Goal) || !validOperation(plan.Operation) || !validDelivery(plan.DeliveryPreference) {
		return fmt.Errorf("invalid scope, goal, operation, or delivery preference")
	}
	if len(strings.TrimSpace(plan.StandaloneRequest)) > MaxStandaloneBytes {
		return fmt.Errorf("standalone_request exceeds %d bytes", MaxStandaloneBytes)
	}
	if plan.Goal != GoalSocial && plan.Goal != GoalOutOfScope && strings.TrimSpace(plan.StandaloneRequest) == "" {
		return fmt.Errorf("standalone_request is required for goal %s", plan.Goal)
	}
	if err := validateSearch("documentation_navigation", plan.Searches.DocumentationNavigation); err != nil {
		return err
	}
	if err := validateSearch("agent_tools", plan.Searches.AgentTools); err != nil {
		return err
	}
	if err := validateRelatedOffsets(plan.RelatedTurnOffsets, input.CompletedTurns); err != nil {
		return err
	}
	if plan.NeedsClarification {
		if err := boundedRequired("clarification_question", plan.ClarificationQuestion, routing.MaxClarificationBytes); err != nil {
			return err
		}
	} else if strings.TrimSpace(plan.ClarificationQuestion) != "" {
		return fmt.Errorf("clarification_question requires needs_clarification")
	}
	if err := validateScopeGoal(plan); err != nil {
		return err
	}
	if err := validateGoalOperation(plan); err != nil {
		return err
	}
	return validateBuilder(plan, input.Surface)
}

func validateSearch(name string, decision SearchDecision) error {
	query := strings.TrimSpace(decision.Query)
	if decision.Needed && query == "" {
		return fmt.Errorf("%s query is required when search is needed", name)
	}
	if !decision.Needed && query != "" {
		return fmt.Errorf("%s query requires needed=true", name)
	}
	if len(query) > MaxSearchQueryBytes {
		return fmt.Errorf("%s query exceeds %d bytes", name, MaxSearchQueryBytes)
	}
	return nil
}

func validateRelatedOffsets(selected []int, supplied []routing.CompletedTurn) error {
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

func validateScopeGoal(plan Plan) error {
	switch plan.Goal {
	case GoalOutOfScope:
		if plan.Scope != ScopeOutOfScope {
			return fmt.Errorf("out_of_scope goal requires out_of_scope scope")
		}
	case GoalUnclear:
		if plan.Scope != ScopeUnclear || !plan.NeedsClarification {
			return fmt.Errorf("unclear goal requires unclear scope and clarification")
		}
	default:
		if plan.Scope != ScopeGenix {
			return fmt.Errorf("goal %s requires genix scope", plan.Goal)
		}
	}
	return nil
}

func validateGoalOperation(plan Plan) error {
	switch plan.Goal {
	case GoalExplainProduct, GoalViewReport, GoalQueryCompanyData, GoalInspectCurrentPage:
		if plan.Operation != OperationRead {
			return fmt.Errorf("goal %s requires read operation", plan.Goal)
		}
	case GoalSocial, GoalOutOfScope, GoalUnclear:
		if plan.Operation != OperationNone {
			return fmt.Errorf("goal %s requires none operation", plan.Goal)
		}
	case GoalManageRecord:
		switch plan.Operation {
		case OperationCreate, OperationUpdate, OperationDelete:
			return nil
		case OperationConfirm, OperationReject:
			// A confirmation continues an already prepared action; rediscovery can
			// only add latency and stale evidence before the live page is inspected.
			if len(plan.RelatedTurnOffsets) == 0 {
				return fmt.Errorf("manage_record %s requires a related completed turn", plan.Operation)
			}
			if plan.Searches.DocumentationNavigation.Needed || plan.Searches.AgentTools.Needed {
				return fmt.Errorf("manage_record %s must not request discovery searches", plan.Operation)
			}
			return nil
		default:
			return fmt.Errorf("manage_record requires create, update, delete, confirm, or reject operation")
		}
	}
	return nil
}

func validateBuilder(plan Plan, surface routing.SurfaceContext) error {
	if plan.Goal != GoalWebpageOperation {
		if plan.Builder.Operation != routing.BuilderNone || plan.Builder.ContextScope != routing.BuilderScopeNone || plan.Builder.RequiresLiveState {
			return fmt.Errorf("non-builder goal contains builder routing")
		}
		return nil
	}
	if plan.Builder.Operation == routing.BuilderNone {
		return fmt.Errorf("webpage_operation requires a builder operation")
	}
	if surface.Kind == routing.SurfaceWebpageBuilderPages && plan.Builder.Operation == routing.BuilderBuildPage {
		if plan.Builder.ContextScope != routing.BuilderScopeNone || plan.Builder.RequiresLiveState {
			return fmt.Errorf("builder page-list build must not request live state")
		}
		return nil
	}
	if !plan.Builder.RequiresLiveState {
		return fmt.Errorf("builder operation %s requires live state", plan.Builder.Operation)
	}
	if plan.Builder.Operation == routing.BuilderEditSection {
		if plan.Builder.ContextScope != routing.BuilderScopeSelectedSection && plan.Builder.ContextScope != routing.BuilderScopeFullPage {
			return fmt.Errorf("edit_section requires selected_section or full_page")
		}
		if plan.Builder.ContextScope == routing.BuilderScopeSelectedSection && (!surface.HasSelectedSection || surface.SelectedSectionID == "") {
			return fmt.Errorf("selected_section edit requires a selected section")
		}
		return nil
	}
	if plan.Builder.ContextScope != routing.BuilderScopeFullPage {
		return fmt.Errorf("builder operation %s requires full_page", plan.Builder.Operation)
	}
	return nil
}

func boundedRequired(name, value string, maximum int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", name)
	}
	if len(trimmed) > maximum {
		return fmt.Errorf("%s exceeds %d bytes", name, maximum)
	}
	return nil
}

func validLanguage(language routing.Language) bool {
	return language == routing.LanguageSpanish || language == routing.LanguageEnglish || language == routing.LanguageMixed || language == routing.LanguageUnknown
}

func validSurface(surface routing.SurfaceContext) bool {
	switch surface.Kind {
	case routing.SurfaceERPPage, routing.SurfaceWebpageBuilderPages, routing.SurfaceWebpageEditor, routing.SurfaceStorefrontPreview, routing.SurfaceUnknown:
		return true
	default:
		return false
	}
}

func validScope(scope Scope) bool {
	return scope == ScopeGenix || scope == ScopeOutOfScope || scope == ScopeUnclear
}

func validGoal(goal Goal) bool {
	switch goal {
	case GoalSocial, GoalExplainProduct, GoalManageRecord, GoalViewReport, GoalQueryCompanyData,
		GoalInspectCurrentPage, GoalOperateCurrentPage, GoalWebpageOperation, GoalOutOfScope, GoalUnclear:
		return true
	default:
		return false
	}
}

func validOperation(operation Operation) bool {
	switch operation {
	case OperationRead, OperationCreate, OperationUpdate, OperationDelete, OperationConfirm, OperationReject, OperationNone:
		return true
	default:
		return false
	}
}

func validDelivery(delivery DeliveryPreference) bool {
	return delivery == DeliveryUI || delivery == DeliveryInline || delivery == DeliveryExplanation || delivery == DeliveryUnspecified
}
