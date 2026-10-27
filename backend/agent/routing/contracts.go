// Package routing retains the stable chat wire types shared by discovery,
// execution, persistence, and frontend surface providers.
package routing

const (
	MaxCompletedTurns     = 5
	MaxClarificationBytes = 500
)

type Language string

const (
	LanguageSpanish Language = "es"
	LanguageEnglish Language = "en"
	LanguageMixed   Language = "mixed"
	LanguageUnknown Language = "unknown"
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

// CompletedTurn is one complete persisted user/assistant pair. Offset -1 is newest.
type CompletedTurn struct {
	Offset           int    `json:"offset" cb:"offset,1"`
	UserMessage      string `json:"user_message" cb:"user_message,2"`
	AssistantMessage string `json:"assistant_message" cb:"assistant_message,3"`
	ActionSummary    string `json:"action_summary,omitempty" cb:"action_summary,4"`
	Route            string `json:"route,omitempty" cb:"route,5"`
}

// SurfaceContext is compact routing metadata supplied by the active frontend page.
type SurfaceContext struct {
	Kind                SurfaceKind           `json:"kind" cb:"kind,1"`
	Route               string                `json:"route,omitempty" cb:"route,2"`
	PageID              string                `json:"page_id,omitempty" cb:"page_id,3"`
	ActiveAgentMode     string                `json:"active_agent_mode,omitempty" cb:"active_agent_mode,4"`
	HasSelectedSection  bool                  `json:"has_selected_section" cb:"has_selected_section,5"`
	SelectedSectionID   string                `json:"selected_section_id,omitempty" cb:"selected_section_id,6"`
	SelectedSectionType string                `json:"selected_section_type,omitempty" cb:"selected_section_type,7"`
	AvailableContexts   []BuilderContextScope `json:"available_contexts,omitempty" cb:"available_contexts,8"`
}
