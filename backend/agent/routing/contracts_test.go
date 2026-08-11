package routing

import "testing"

func validInput() ClassifierInput {
	return ClassifierInput{
		Schema: SchemaVersion, CurrentMessage: "¿Puedo anularla después?", AppLanguage: LanguageSpanish,
		Surface:        SurfaceContext{Kind: SurfaceERPPage, Route: "/logistics/purchase-orders"},
		CompletedTurns: []CompletedTurn{{Offset: -1, UserMessage: "¿Cómo confirmo una OC?", AssistantMessage: "Abre la orden y confirma.", Route: "/logistics/purchase-orders"}},
		Capabilities:   CapabilitySnapshot(),
	}
}

func validDocumentationVerdict() Verdict {
	return Verdict{
		Schema: SchemaVersion, Language: LanguageSpanish, ResponseLanguage: LanguageSpanish,
		Scope: ScopeGenix, PrimaryIntent: IntentProductKnowledge, RequestedOperation: OperationRead,
		RelatedTurnOffsets: []int{-1}, StandaloneRequest: "¿Puedo anular una orden de compra después de confirmarla?",
		RequiredCapabilities: []CapabilityName{CapabilityDocumentationSearch},
		Builder:              BuilderDecision{Operation: BuilderNone, ContextScope: BuilderScopeNone},
	}
}

func TestVerdictAcceptsProductKnowledgeFollowUp(t *testing.T) {
	if err := validDocumentationVerdict().Validate(validInput()); err != nil {
		t.Fatal(err)
	}
}

func TestVerdictRejectsUnrelatedOffset(t *testing.T) {
	verdict := validDocumentationVerdict()
	verdict.RelatedTurnOffsets = []int{-2}
	if err := verdict.Validate(validInput()); err == nil {
		t.Fatal("offset not supplied to the classifier must be rejected")
	}
}

func TestVerdictRejectsProductKnowledgeWithoutSearch(t *testing.T) {
	verdict := validDocumentationVerdict()
	verdict.RequiredCapabilities = nil
	if err := verdict.Validate(validInput()); err == nil {
		t.Fatal("product knowledge without documentation search must be rejected")
	}
}

func TestVerdictRejectsDocumentationSearchWithoutStandaloneRequest(t *testing.T) {
	verdict := validDocumentationVerdict()
	verdict.PrimaryIntent = IntentNavigation
	verdict.StandaloneRequest = ""
	if err := verdict.Validate(validInput()); err == nil {
		t.Fatal("documentation search without a standalone query must be rejected")
	}
}

func TestVerdictAcceptsBuilderAddSectionOnlyWithFullLiveState(t *testing.T) {
	input := validInput()
	input.CurrentMessage = "Add a product section"
	input.Surface = SurfaceContext{Kind: SurfaceWebpageEditor, Route: "/webpage-builder/42", AvailableContexts: []BuilderContextScope{BuilderScopeFullPage}}
	verdict := Verdict{
		Schema: SchemaVersion, Language: LanguageEnglish, ResponseLanguage: LanguageEnglish,
		Scope: ScopeGenix, PrimaryIntent: IntentWebpageAddSection, RequestedOperation: OperationCreate,
		StandaloneRequest: input.CurrentMessage, RequiredCapabilities: []CapabilityName{CapabilityWebpageBuilder},
		Builder: BuilderDecision{Operation: BuilderAddSection, ContextScope: BuilderScopeFullPage, RequiresLiveState: true},
	}
	if err := verdict.Validate(input); err != nil {
		t.Fatal(err)
	}
	verdict.Builder.ContextScope = BuilderScopeSelectedSection
	if err := verdict.Validate(input); err == nil {
		t.Fatal("add_section must reject selected-section state")
	}
}

func TestOperationalDataRequiresFutureDomainCapability(t *testing.T) {
	input := validInput()
	verdict := Verdict{
		Schema: SchemaVersion, Language: LanguageSpanish, ResponseLanguage: LanguageSpanish,
		Scope: ScopeGenix, PrimaryIntent: IntentOperationalData, RequestedOperation: OperationRead,
		StandaloneRequest: "Dame la última venta", Builder: BuilderDecision{Operation: BuilderNone, ContextScope: BuilderScopeNone},
	}
	if err := verdict.Validate(input); err == nil {
		t.Fatal("operational data without a domain search capability must be rejected")
	}
	verdict.RequiredCapabilities = []CapabilityName{CapabilitySalesSearch}
	if err := verdict.Validate(input); err != nil {
		t.Fatal(err)
	}
}

func TestLocalizedResponseFallsBackToSpanish(t *testing.T) {
	if got := LocalizedResponse(ResponseOutOfScope, LanguageMixed); got != localizedResponses[ResponseOutOfScope][LanguageSpanish] {
		t.Fatalf("unexpected fallback response: %q", got)
	}
}
