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
