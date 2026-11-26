package genixbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"genix-agent/genixbot/outline"
	"genix-agent/llm"
)

// The turn loop, shared by Genix (agent/genixbot_turn.go: the browser tab) and the evaluation
// (genix-local-ai cmd/eval-loop: captured page states). It reproduces what the training data shows:
//
//	user      : Ruta: /crm/customers\ncrea el cliente Pedro Pascal
//	assistant : ver()                       → full outline
//	assistant : hacer("74 click")           → full outline (the page changed)
//	assistant : hacer("79 setValue Pedro")  → Delta (only the changed lines)
//	assistant : Listo, llené Nombre. ¿Guardo el cliente?    (plain text ends the turn)
//
// Past turns are replayed whole (user, tool calls, tool results, answer), the way the two-turn
// samples (save or cancel after the confirmation question) are written.

const (
	// MaxSteps caps the model calls of one turn.
	MaxSteps = 12
	// HistoryTurns is how many completed turns are replayed. The training data has at most one
	// before the current message; a few more keep short follow-ups answerable.
	HistoryTurns = 3
	// maxAnswerTokens: LFM3 may open a <think> block before answering.
	maxAnswerTokens = 600
)

// ErrStepLimit: the model kept calling tools without answering.
var ErrStepLimit = errors.New("genix-bot: step limit reached without an answer")

// Page is the page a turn reads and acts on. Every method returns the page's outline after it ran.
type Page interface {
	Outline(ctx context.Context) (string, error)
	// Navigate opens a route of the user's menu; an unknown route is an error.
	Navigate(ctx context.Context, route string) (string, error)
	// Invoke runs the actions in order and stops at the first failure: one result per action run.
	Invoke(ctx context.Context, actions []Action) ([]ActionResult, string, error)
}

// ActionResult is what one action returned: an error, or a value worth showing (search matches).
type ActionResult struct {
	Error string
	Value string
}

// Facts answers `hechos`: "Fact <id> | <tipo> | <texto>" lines.
type Facts interface {
	Search(ctx context.Context, consulta, ruta string) (string, error)
}

// Model is one chat call: messages and tools in, the assistant message out.
type Model interface {
	Next(ctx context.Context, messages []llm.Message, tools []llm.Tool) (llm.Message, error)
}

// CompletedTurn is a past turn as it is replayed: Steps are its assistant tool calls and tool results.
type CompletedTurn struct {
	Route    string
	UserText string
	Answer   string
	Steps    []llm.Message
}

// TurnResult is a finished turn: the answer, the steps to replay next turn and the action log.
type TurnResult struct {
	Answer  string
	Steps   []llm.Message
	Actions []string
}

// Messages builds the prompt of a turn: the system prompt, the replayed history and the user message.
func Messages(history []CompletedTurn, route, text string) []llm.Message {
	if len(history) > HistoryTurns {
		history = history[len(history)-HistoryTurns:]
	}
	messages := []llm.Message{{Role: "system", Content: SystemPrompt}}
	for _, completedTurn := range history {
		messages = append(messages, llm.Message{Role: "user", Content: UserMessage(completedTurn.Route, completedTurn.UserText)})
		messages = append(messages, completedTurn.Steps...)
		messages = append(messages, llm.Message{Role: "assistant", Content: completedTurn.Answer})
	}
	return append(messages, llm.Message{Role: "user", Content: UserMessage(route, text)})
}

// RunTurn runs one user message until the model answers in plain text. onToolCall sees every call
// before it runs (Genix shows it as the turn's progress).
func RunTurn(ctx context.Context, model Model, page Page, facts Facts, history []CompletedTurn, route, text string,
	onToolCall func(llm.ToolCall)) (TurnResult, error) {

	messages := Messages(history, route, text)
	turn := &turnState{page: page, facts: facts}
	// A "!" action is the user's confirmation: valid only when the previous answer asked for it.
	if len(history) > 0 {
		turn.confirmationAsked = strings.HasSuffix(strings.TrimSpace(history[len(history)-1].Answer), "?")
	}
	result := TurnResult{}
	for step := 0; step < MaxSteps; step++ {
		reply, err := model.Next(ctx, messages, Tools)
		if err != nil {
			return result, err
		}
		if len(reply.ToolCalls) == 0 {
			result.Answer = strings.TrimSpace(reply.Content)
			if result.Answer == "" {
				return result, errors.New("genix-bot: empty answer")
			}
			return result, nil
		}
		assistantMessage := llm.Message{Role: "assistant", ToolCalls: reply.ToolCalls}
		result.Steps = append(result.Steps, assistantMessage)
		messages = append(messages, assistantMessage)
		for _, call := range reply.ToolCalls {
			if onToolCall != nil {
				onToolCall(call)
			}
			content, action := turn.runTool(ctx, call)
			if action != "" {
				result.Actions = append(result.Actions, action)
			}
			toolMessage := llm.Message{Role: "tool", Content: content, ToolCallID: call.ID}
			result.Steps = append(result.Steps, toolMessage)
			messages = append(messages, toolMessage)
		}
	}
	return result, ErrStepLimit
}

type turnState struct {
	page              Page
	facts             Facts
	confirmationAsked bool
	// shownOutline is the last full outline this turn showed the model: `hacer` answers with the
	// delta against it. A new turn starts without one, so its first `hacer` answers in full.
	shownOutline string
}

// runTool runs one call and returns the tool message and the action-log entry ("" for reads).
func (turn *turnState) runTool(ctx context.Context, call llm.ToolCall) (string, string) {
	arguments := map[string]string{}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &arguments); err != nil && call.Function.Arguments != "" {
		return "Error | argumentos no válidos: " + err.Error(), ""
	}
	switch call.Function.Name {
	case ToolVer:
		pageOutline, err := turn.page.Outline(ctx)
		if err != nil {
			return "Error | ver: " + err.Error(), ""
		}
		turn.shownOutline = pageOutline
		return strings.TrimSpace(pageOutline), ""

	case ToolIr:
		route := strings.TrimSpace(arguments["ruta"])
		pageOutline, err := turn.page.Navigate(ctx, route)
		if err != nil {
			return "Error | " + route + ": " + err.Error(), ""
		}
		turn.shownOutline = pageOutline
		return strings.TrimSpace(pageOutline), "ir " + route

	case ToolHacer:
		return turn.runActions(ctx, arguments["acciones"])

	case ToolHechos:
		facts, err := turn.facts.Search(ctx, arguments["consulta"], arguments["ruta"])
		if err != nil {
			return "Error | hechos: " + err.Error(), ""
		}
		return strings.TrimSpace(facts), ""

	case ToolCalcular:
		return CalculateResult(arguments["expresion"]), ""
	}
	return "Error | herramienta desconocida: " + call.Function.Name, ""
}

// protectedButtonPattern: buttons that save, send, delete, pay or deliver. Clicking one needs "!" (the user's
// confirmation); Genix's buttons carry no role attribute, so the label decides.
var protectedButtonPattern = regexp.MustCompile(`(?i)^\s*Button ([0-9:]+) \| (guardar|grabar|eliminar|borrar|enviar|anular|emitir|pagar|cobrar|entregar|save|delete|send|pay|deliver)\b`)

func (turn *turnState) runActions(ctx context.Context, acciones string) (string, string) {
	actions, err := ParseActions(acciones)
	if err != nil {
		return "Error | " + err.Error(), ""
	}
	// The confirmation guard reads the live page: the model may act on ids from an older outline.
	livePage, err := turn.page.Outline(ctx)
	if err != nil {
		return "Error | hacer: " + err.Error(), ""
	}
	protectedIDs := map[string]bool{}
	for _, line := range strings.Split(livePage, "\n") {
		if match := protectedButtonPattern.FindStringSubmatch(line); match != nil {
			protectedIDs[match[1]] = true
		}
	}
	for _, action := range actions {
		needsConfirmation := action.Method == "click" && protectedIDs[action.ID]
		if (action.Confirmed && !turn.confirmationAsked) || (needsConfirmation && !action.Confirmed) {
			return "Error | " + action.ID + ": requiere confirmación del usuario", ""
		}
	}

	results, nextOutline, err := turn.page.Invoke(ctx, actions)
	if err != nil {
		return "Error | hacer: " + err.Error(), ""
	}
	lines := []string{}
	executed := []string{}
	for index, actionResult := range results {
		action := actions[index]
		switch {
		case actionResult.Error != "":
			lines = append(lines, fmt.Sprintf("Error | %s: %s", action.ID, actionResult.Error))
		case actionResult.Value != "":
			lines = append(lines, fmt.Sprintf("Text | %s %s: %s", action.ID, action.Method, actionResult.Value))
		}
		if actionResult.Error == "" {
			executed = append(executed, action.String())
		}
	}
	pageResult := nextOutline
	if turn.shownOutline != "" {
		pageResult, _ = outline.Delta(turn.shownOutline, nextOutline)
	}
	turn.shownOutline = nextOutline
	lines = append(lines, strings.TrimSpace(pageResult))

	actionLog := ""
	if len(executed) > 0 {
		actionLog = "hacer " + strings.Join(executed, "; ")
	}
	return strings.Join(lines, "\n"), actionLog
}
