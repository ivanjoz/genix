package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"app/agent/llm"
	"app/core"
)

const (
	classifierAttempts  = 2
	classifierMaxTokens = 1_200
)

var classifierReasoningDisabled = false

// ChatCompleter lets tests exercise retry and validation without a live provider.
type ChatCompleter interface {
	Chat(context.Context, llm.ChatRequest) (*llm.ChatResponse, error)
}

type Classifier struct {
	chatClient ChatCompleter
	modelID    string
}

func NewClassifier(chatClient ChatCompleter, modelID string) (*Classifier, error) {
	if chatClient == nil {
		return nil, errors.New("classifier chat client is required")
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, errors.New("agent.classifier_model is required")
	}
	return &Classifier{chatClient: chatClient, modelID: modelID}, nil
}

// Classify retries once when the provider or strict verdict validation fails.
func (classifier *Classifier) Classify(ctx context.Context, input ClassifierInput) (Verdict, error) {
	if err := input.Validate(); err != nil {
		return Verdict{}, fmt.Errorf("validate classifier input: %w", err)
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return Verdict{}, fmt.Errorf("marshal classifier input: %w", err)
	}

	var lastError error
	for attempt := 1; attempt <= classifierAttempts; attempt++ {
		startedAt := time.Now()
		response, chatError := classifier.chatClient.Chat(ctx, llm.ChatRequest{
			Model:       classifier.modelID,
			Messages:    classifierMessages(string(inputJSON), attempt, lastError),
			Temperature: float32Pointer(0),
			MaxTokens:   classifierMaxTokens,
			Reasoning:   &llm.ReasoningOptions{Enabled: &classifierReasoningDisabled},
		})
		if chatError != nil {
			lastError = fmt.Errorf("provider call: %w", chatError)
			core.Log("agent.router classifier_failed model::", classifier.modelID, " attempt::", attempt,
				" elapsed::", time.Since(startedAt).Round(time.Millisecond), " stage::provider")
			continue
		}

		rawContent := response.Choices[0].Message.Content
		verdict, decodeError := decodeVerdict(rawContent)
		if decodeError == nil {
			decodeError = verdict.Validate(input)
		}
		if decodeError != nil {
			lastError = decodeError
			core.Log("agent.router classifier_failed model::", classifier.modelID, " attempt::", attempt,
				" elapsed::", time.Since(startedAt).Round(time.Millisecond), " stage::validation response_bytes::", len(rawContent))
			continue
		}

		core.Log("agent.router classifier_ok model::", classifier.modelID, " attempt::", attempt,
			" elapsed::", time.Since(startedAt).Round(time.Millisecond), " response_bytes::", len(rawContent),
			" language::", verdict.Language, " scope::", verdict.Scope, " intent::", verdict.PrimaryIntent,
			" operation::", verdict.RequestedOperation, " related_turns::", len(verdict.RelatedTurnOffsets))
		return verdict, nil
	}
	return Verdict{}, fmt.Errorf("classifier failed after %d attempts: %w", classifierAttempts, lastError)
}

func classifierMessages(inputJSON string, attempt int, previousError error) []llm.Message {
	systemPrompt := classifierSystemPrompt
	if attempt > 1 {
		// Return only the error category; raw model output and user text stay out of logs and repair prose.
		systemPrompt += "\nYour previous response was invalid. Return one corrected JSON object only."
		if previousError != nil {
			systemPrompt += " Validation category: " + validationCategory(previousError) + "."
		}
	}
	return []llm.Message{{Role: "system", Content: systemPrompt}, {Role: "user", Content: inputJSON}}
}

func decodeVerdict(content string) (Verdict, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(strings.TrimSpace(content)))
	decoder.DisallowUnknownFields()
	verdict := Verdict{}
	if err := decoder.Decode(&verdict); err != nil {
		return Verdict{}, fmt.Errorf("decode strict verdict JSON: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return Verdict{}, err
	}
	return verdict, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing classifier content: %w", err)
	}
	return errors.New("classifier response contains more than one JSON value")
}

func validationCategory(err error) string {
	message := err.Error()
	for _, category := range []string{"schema", "language", "scope", "intent", "operation", "offset", "capability", "builder", "clarification", "JSON"} {
		if strings.Contains(strings.ToLower(message), strings.ToLower(category)) {
			return strings.ToLower(category)
		}
	}
	return "contract"
}

func float32Pointer(value float32) *float32 { return &value }

const classifierSystemPrompt = `You are the global request classifier for Genix, an ERP and ecommerce builder.
Return exactly one JSON object matching schema version 1. Never answer the user and never call tools.

Use only these enum values:
- language: es, en, mixed, unknown; response_language: es or en.
- scope: genix, out_of_scope, unclear.
- primary_intent and secondary_intents: social, product_knowledge, operational_data, current_page, navigation, page_action, confirmation, webpage_build, webpage_add_section, webpage_edit_section, webpage_remove_section, webpage_reorder_section, webpage_inspect, out_of_scope, ambiguous.
- requested_operation: read, navigate, create, update, delete, confirm, reject, none.
- builder.operation: none, build_page, add_section, edit_section, remove_section, reorder_section, inspect_page.
- builder.context_scope: none, full_page, selected_section.

Required JSON fields: schema, language, response_language, scope, primary_intent, secondary_intents,
requested_operation, related_turn_offsets, standalone_request, required_capabilities, business_domain,
entities, builder, needs_clarification, clarification_question.
Exact shape:
{"schema":1,"language":"es","response_language":"es","scope":"genix","primary_intent":"product_knowledge","secondary_intents":[],"requested_operation":"read","related_turn_offsets":[],"standalone_request":"...","required_capabilities":["documentation_search"],"business_domain":"","entities":[{"type":"","name":""}],"builder":{"operation":"none","context_scope":"none","target_section_type":"","target_section_reference":"","requires_live_state":false},"needs_clarification":false,"clarification_question":""}

Rules:
1. Product behavior, procedures, prerequisites, rules, limitations, rationale, or where a feature is located use product_knowledge and documentation_search.
2. Real company records, sales, balances, stock, customers, suppliers, or reports use operational_data and the matching search capability even when unavailable.
3. Current visible values and controls use current_page. Non-builder UI changes use page_action.
4. The frontend surface disambiguates implicit builder wording, but explicit ERP wording wins.
5. In webpage_builder_editor, “add a product section” means webpage_add_section with full_page live state. Asking how to create an inventory product remains product_knowledge.
6. Select only supplied completed-turn offsets that are necessary to resolve this request. Use none for an independent topic.
7. standalone_request preserves the user language, negation, dates, amounts, statuses, and operation. Resolve omitted references only from selected turns; never invent values.
8. Use ambiguous, scope unclear, and one focused clarification question when safe routing is impossible.
9. A builder operation other than none requires webpage_builder. Add/remove/reorder/inspect use full_page. Edit uses selected_section when selected, otherwise full_page. A page-list build uses no live state until a target is chosen.
10. social and out_of_scope need no capabilities. Output JSON only.`
