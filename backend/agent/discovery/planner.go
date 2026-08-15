package discovery

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
	plannerAttempts  = 2
	plannerMaxTokens = 1_400
)

var plannerReasoningDisabled = false

type ChatCompleter interface {
	Chat(context.Context, llm.ChatRequest) (*llm.ChatResponse, error)
}

type Planner struct {
	chatClient ChatCompleter
	modelID    string
}

// PlannerAttemptTrace exposes the exact request and raw response strictly for
// local observability; it never influences discovery validation or routing.
type PlannerAttemptTrace struct {
	Attempt  int
	Messages []llm.Message
	Response string
	Error    string
}

type PlannerAttemptObserver func(PlannerAttemptTrace)

func NewPlanner(chatClient ChatCompleter, modelID string) (*Planner, error) {
	if chatClient == nil {
		return nil, errors.New("discovery planner chat client is required")
	}
	if modelID = strings.TrimSpace(modelID); modelID == "" {
		return nil, errors.New("agent.classifier_model is required for discovery planning")
	}
	return &Planner{chatClient: chatClient, modelID: modelID}, nil
}

func (planner *Planner) Plan(ctx context.Context, input PlannerInput) (Plan, error) {
	return planner.PlanObserved(ctx, input, nil)
}

// PlanObserved runs the same strict planner while reporting every provider
// attempt to the caller-provided local logger.
func (planner *Planner) PlanObserved(ctx context.Context, input PlannerInput, observer PlannerAttemptObserver) (Plan, error) {
	if err := input.Validate(); err != nil {
		return Plan{}, fmt.Errorf("validate discovery input: %w", err)
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return Plan{}, fmt.Errorf("marshal discovery input: %w", err)
	}

	var lastError error
	for attempt := 1; attempt <= plannerAttempts; attempt++ {
		startedAt := time.Now()
		messages := plannerMessages(string(inputJSON), attempt, lastError)
		response, chatError := planner.chatClient.Chat(ctx, llm.ChatRequest{
			Model:       planner.modelID,
			Messages:    messages,
			Temperature: float32Pointer(0),
			MaxTokens:   plannerMaxTokens,
			Reasoning:   &llm.ReasoningOptions{Enabled: &plannerReasoningDisabled},
		})
		if chatError != nil {
			lastError = fmt.Errorf("provider call: %w", chatError)
			observePlannerAttempt(observer, PlannerAttemptTrace{Attempt: attempt, Messages: messages, Error: lastError.Error()})
			core.Log("agent.discovery plan_failed model::", planner.modelID, " attempt::", attempt,
				" elapsed::", time.Since(startedAt).Round(time.Millisecond), " stage::provider")
			continue
		}
		if response == nil || len(response.Choices) == 0 {
			lastError = errors.New("provider returned no discovery choice")
			observePlannerAttempt(observer, PlannerAttemptTrace{Attempt: attempt, Messages: messages, Error: lastError.Error()})
			continue
		}

		rawResponse := response.Choices[0].Message.Content
		plan, decodeError := decodePlan(rawResponse)
		if decodeError == nil {
			plan = normalizeDocumentationSearch(plan)
			decodeError = plan.Validate(input)
		}
		trace := PlannerAttemptTrace{Attempt: attempt, Messages: messages, Response: rawResponse}
		if decodeError != nil {
			lastError = decodeError
			trace.Error = decodeError.Error()
			observePlannerAttempt(observer, trace)
			core.Log("agent.discovery plan_failed model::", planner.modelID, " attempt::", attempt,
				" elapsed::", time.Since(startedAt).Round(time.Millisecond), " stage::validation")
			continue
		}
		observePlannerAttempt(observer, trace)
		core.Log("agent.discovery plan_ok model::", planner.modelID, " attempt::", attempt,
			" elapsed::", time.Since(startedAt).Round(time.Millisecond), " goal::", plan.Goal,
			" operation::", plan.Operation, " features::", plan.Searches.DocumentationNavigation.Needed,
			" tools::", plan.Searches.AgentTools.Needed, " documentation_query::", plan.Searches.DocumentationNavigation.Query,
			" related_turns::", len(plan.RelatedTurnOffsets))
		return plan, nil
	}
	return Plan{}, fmt.Errorf("discovery planner failed after %d attempts: %w", plannerAttempts, lastError)
}

func observePlannerAttempt(observer PlannerAttemptObserver, trace PlannerAttemptTrace) {
	if observer != nil {
		observer(trace)
	}
}

func plannerMessages(inputJSON string, attempt int, previousError error) []llm.Message {
	systemPrompt := PlannerSystemPrompt
	if attempt > 1 {
		systemPrompt += "\nYour previous response was invalid. Return one corrected JSON object only."
		if previousError != nil {
			// Concrete contract feedback prevents the repair attempt from blindly
			// repeating the same invalid search or operation combination.
			systemPrompt += " Validation error: " + previousError.Error()
		}
	}
	return []llm.Message{{Role: "system", Content: systemPrompt}, {Role: "user", Content: inputJSON}}
}

func decodePlan(content string) (Plan, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(strings.TrimSpace(content)))
	decoder.DisallowUnknownFields()
	plan := Plan{}
	if err := decoder.Decode(&plan); err != nil {
		return Plan{}, fmt.Errorf("decode strict discovery JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Plan{}, errors.New("discovery response contains more than one JSON value")
		}
		return Plan{}, fmt.Errorf("decode trailing discovery content: %w", err)
	}
	return plan, nil
}

func float32Pointer(value float32) *float32 { return &value }

const PlannerSystemPrompt = `You are the discovery planner for Genix.
Genix is a web-based ERP and ecommerce application for small businesses. Users manage products, inventory, sales, purchases, customers, suppliers, finance, reports, and storefronts primarily through application pages, while some business data can also be retrieved through specialized agent tools.

You prepare evidence searches. Never answer the user, call application tools, invent a route, invent a data tool, or decide that a capability is unavailable. Return exactly one JSON object matching schema version 1.

Enum values:
- language: es, en, mixed, unknown; response_language: es or en.
- scope: genix, out_of_scope, unclear.
- goal: social, explain_product, manage_record, view_report, query_company_data, inspect_current_page, operate_current_page, webpage_operation, out_of_scope, unclear.
- operation: read, create, update, delete, confirm, reject, none.
- delivery_preference: ui, inline, explanation, unspecified.
- builder.operation: none, build_page, add_section, edit_section, remove_section, reorder_section, inspect_page.
- builder.context_scope: none, full_page, selected_section.

Required shape:
{"schema":1,"language":"es","response_language":"es","scope":"genix","goal":"manage_record","operation":"create","domain":"products","entities":[{"type":"product","name":"Aceite Tondero"}],"delivery_preference":"unspecified","related_turn_offsets":[],"standalone_request":"Crear un producto llamado Aceite Tondero con precio de 12 soles.","searches":{"documentation_navigation":{"needed":true,"query":"Crear un producto"},"agent_tools":{"needed":false,"query":""}},"builder":{"operation":"none","context_scope":"none","target_section_type":"","target_section_reference":"","requires_live_state":false},"needs_clarification":false,"clarification_question":""}

Rules:
1. goal describes the business goal, never the delivery mechanism. Creating, updating, or deleting an ERP record is manage_record, not product explanation or company-data query.
2. explain_product, view_report, query_company_data, and inspect_current_page always use operation=read. social, out_of_scope, and unclear use operation=none. manage_record uses create, update, delete, confirm, or reject.
3. Product behavior, procedures, prerequisites, rules, limitations, rationale, and “how do I” questions use explain_product and delivery_preference=explanation.
4. Requests for actual records, aggregates, balances, stock, metrics, or inline reports use query_company_data. Requests for an existing report without explicit inline wording use view_report.
5. Explicit open/go/take-me wording uses delivery_preference=ui. Explicit show-here/summarize/calculate wording uses inline. Otherwise use unspecified; execution defaults to the Genix UI.
6. Request documentation_navigation when an existing Genix page, documented workflow, report, explanation, or builder entry point may help. A webpage operation already inside webpage_builder_editor or webpage_builder_pages needs neither discovery search; outside those surfaces, search documentation/navigation to locate the builder.
7. Request agent_tools for actual company records, aggregates, balances, stock, metrics, or inline reports. Request BOTH searches when either an existing page or inline data tool could satisfy the request, such as “I want the sales report.”
8. Current visible values and controls normally need neither search; call 2 can inspect the current page. A short follow-up accepting or rejecting a pending record action (for example "si guarda", "sí guárdalo", "no cancel", or "yes, save it") is manage_record with operation=confirm or reject, selects the required completed-turn offset, and MUST set both searches to needed=false.
9. documentation_navigation.query searches software features, not record instances. Keep only the generic user action, record/report type, workflow, and relevant feature vocabulary. Exclude every proper or instance value: product/customer/supplier names, brands, SKUs, IDs, quoted names, and all numbers, amounts, and dates. Example: “Agregar un cliente llamado Pedro Pascal” becomes “Agregar un cliente”; “Crear Aceite Tondero a 12 soles” becomes “Crear un producto”. Always list those omitted instance names in entities.
10. Preserve names, dates, amounts, filters, status, negation, and operation in standalone_request and agent_tools.query, because execution and company-data tool discovery may need them. Select only supplied completed-turn offsets needed to resolve references.
11. The frontend surface disambiguates webpage-builder wording. Builder operations preserve the existing scope rules: add/remove/reorder/inspect use full_page; edit uses selected_section when selected, otherwise full_page; a page-list build uses no live state.
12. Use unclear with one focused clarification only when discovery cannot safely search the possible meanings. Output JSON only.`
