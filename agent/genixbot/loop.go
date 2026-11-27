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
//	assistant : read()                      → full outline
//	assistant : act("74 click")             → full outline (the page changed)
//	assistant : act("79 setValue Pedro")    → Delta (only the changed lines)
//	assistant : Listo, llené Nombre. ¿Guardo el cliente?    (plain text ends the turn)
//
// A data question ends without a last model call: the function's answer is the reply.
//
//	user      : Ruta: /sales/sale_orders\n¿cuánto vendí esta semana?
//	assistant : search("", "ventas totales de un periodo") → Funciones: signature lines
//	assistant : call("sales_total", {"period":"S"})          → the Summary; the turn ends
//
// The last completed turn is replayed whole (user, tool calls, tool results, answer); older turns keep
// their calls and only the first line of each tool result (see Messages).

const (
	// MaxSteps caps the model calls of one turn.
	MaxSteps = 12
	// repeatedFailureLimit: failing calls in a row after which the turn must answer without tools.
	repeatedFailureLimit = 3
	// HistoryTurns is how many completed turns are replayed: enough for the model to tell which earlier
	// thread a short reply ("ok", "sí") continues after the user changed topic.
	HistoryTurns = 5
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

// Search answers `search`: a "Hechos" section of "Fact <id> | <tipo> | <texto>" lines and a
// "Funciones" section of signature lines, each only when its query is not empty. route is the page
// the user is on: the turn's route, then the last one `open` reached (facts of that page rank first).
// The implementation adds the user's access, so only functions the user may call are listed.
type Search interface {
	Search(ctx context.Context, facts, functions, route string) (string, error)
}

// Functions runs the data functions `call` names. An error goes back to the model ("Error | …").
type Functions interface {
	Call(ctx context.Context, name string, arguments map[string]any) (FunctionResult, error)
}

// FunctionResult is a call that ends the turn: an Answer, or a Choice the user must make first.
type FunctionResult struct {
	Answer *FunctionAnswer
	Choice *FunctionChoice
}

// FunctionAnswer is the reply of a data function. Summary is the tool message later turns replay
// (they never see the table); Display holds the tables and charts, opaque here: the adapter that
// made it reads it back.
type FunctionAnswer struct {
	Message string
	Summary string
	Display any
}

// FunctionChoice asks which record an ambiguous name stands for. Question is the reply, Options the
// chat's buttons, Summary the tool message ("Pregunta | ¿Qué producto es «x»? | #12 …; #13 …"), so a
// typed reply can name the record.
type FunctionChoice struct {
	Question string
	Summary  string
	Options  []ChoiceOption
}

type ChoiceOption struct {
	ID    string
	Label string
}

// ErrorSearchFirst: `call` of a function no `search` of this turn listed.
const ErrorSearchFirst = "busca la función primero"

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
// A data function that ended the turn leaves its tables and charts in Display, or its question in
// Choice (Answer is then the question).
type TurnResult struct {
	Answer  string
	Display any
	Choice  *FunctionChoice
	Steps   []llm.Message
	Actions []string
}

// Messages builds the prompt of a turn: the system prompt, the replayed history and the user message.
// Only the last completed turn keeps its tool results whole. Older turns keep each result's first line
// ("Page /crm/customers | Clientes", an "Error | …" line): what the model did stays readable, the page
// it read then does not — the current page comes from `read`, never from an old outline.
func Messages(history []CompletedTurn, route, text string) []llm.Message {
	if len(history) > HistoryTurns {
		history = history[len(history)-HistoryTurns:]
	}
	messages := []llm.Message{{Role: "system", Content: SystemPrompt}}
	for index, completedTurn := range history {
		messages = append(messages, llm.Message{Role: "user", Content: UserMessage(completedTurn.Route, completedTurn.UserText)})
		isOlderTurn := index < len(history)-1
		for _, step := range completedTurn.Steps {
			if isOlderTurn && step.Role == "tool" {
				step.Content, _, _ = strings.Cut(step.Content, "\n")
			}
			messages = append(messages, step)
		}
		messages = append(messages, llm.Message{Role: "assistant", Content: completedTurn.Answer})
	}
	return append(messages, llm.Message{Role: "user", Content: UserMessage(route, text)})
}

// RunTurn runs one user message until the model answers in plain text or a data function answers.
// onToolCall sees every call before it runs (Genix shows it as the turn's progress).
func RunTurn(ctx context.Context, model Model, page Page, search Search, functions Functions, history []CompletedTurn,
	route, text string, onToolCall func(llm.ToolCall)) (TurnResult, error) {

	messages := Messages(history, route, text)
	// A "!" action is valid when the previous answer asked, or the user asked for the final action now.
	previousAnswerAsked := len(history) > 0 && strings.HasSuffix(strings.TrimSpace(history[len(history)-1].Answer), "?")
	turn := &turnState{page: page, search: search, functions: functions, route: route, searchedFunctions: map[string]bool{},
		mayConfirm: previousAnswerAsked || AsksForFinalAction(text)}
	result := TurnResult{}
	// A turn that keeps failing stops calling tools: the same call failing twice in a row, or three
	// failing calls in a row, ends with one more model call without tools, which must answer.
	failedCall, failures, lastError := "", 0, ""
	tools := Tools
	for step := 0; step < MaxSteps; step++ {
		reply, err := model.Next(ctx, messages, tools)
		if err != nil {
			return result, err
		}
		if len(reply.ToolCalls) == 0 || tools == nil {
			result.Answer = strings.TrimSpace(reply.Content)
			if result.Answer == "" && tools == nil {
				result.Answer = "No pude hacerlo: " + lastError + "."
			}
			if result.Answer == "" {
				return result, errors.New("genix-bot: empty answer")
			}
			return result, nil
		}
		// The step's reasoning stays for the rest of the turn: the first step's reasoning is the turn's
		// plan (finish the view, then answer). LFM3's template drops it once a new user message comes.
		assistantMessage := llm.Message{Role: "assistant", ToolCalls: reply.ToolCalls, ReasoningContent: reply.ReasoningContent}
		result.Steps = append(result.Steps, assistantMessage)
		messages = append(messages, assistantMessage)
		for index, call := range reply.ToolCalls {
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
			if functionResult := turn.functionResult; functionResult != nil {
				// The function's answer is the reply. The calls after it never ran: the stored step
				// keeps only what ran, so the history replays a call per tool message.
				result.Steps[len(result.Steps)-index-2].ToolCalls = reply.ToolCalls[:index+1]
				if functionResult.Choice != nil {
					result.Answer, result.Choice = functionResult.Choice.Question, functionResult.Choice
				} else {
					result.Answer, result.Display = functionResult.Answer.Message, functionResult.Answer.Display
				}
				return result, nil
			}
			signature := call.Function.Name + " " + call.Function.Arguments
			switch {
			case !strings.HasPrefix(content, "Error |"):
				failedCall, failures = "", 0
			case signature == failedCall:
				failures = repeatedFailureLimit
			default:
				failedCall, failures = signature, failures+1
			}
			if failures > 0 {
				firstLine, _, _ := strings.Cut(strings.TrimPrefix(content, "Error | "), "\n")
				lastError = strings.TrimSuffix(firstLine, ".")
			}
		}
		if failures >= repeatedFailureLimit {
			tools = nil
		}
	}
	return result, ErrStepLimit
}

type turnState struct {
	page      Page
	search    Search
	functions Functions
	// route is the page the user is on: the turn's route, then the last one `open` reached.
	route string
	// searchedFunctions are the function names a `search` of this turn listed: the only ones `call` runs.
	searchedFunctions map[string]bool
	// functionResult is set by a `call` that ends the turn.
	functionResult *FunctionResult
	// mayConfirm: a "!" action is the user's confirmation (AsksForFinalAction or an answer that asked).
	mayConfirm bool
	// shownOutline is the last full outline this turn showed the model: `act` answers with the
	// delta against it. A new turn starts without one, so its first `act` answers in full.
	shownOutline string
}

// toolArguments are every tool's arguments; each tool reads its own.
type toolArguments struct {
	Route      string         `json:"route"`
	Actions    string         `json:"actions"`
	Facts      string         `json:"facts"`
	Functions  string         `json:"functions"`
	Function   string         `json:"function"`
	Arguments  map[string]any `json:"arguments"`
	Expression string         `json:"expression"`
}

// signatureNamePattern reads the function name of a signature line in the "Funciones" section.
var signatureNamePattern = regexp.MustCompile(`(?m)^\s*([a-z][a-z0-9_]*)\(`)

// runTool runs one call and returns the tool message and the action-log entry ("" for reads).
func (turn *turnState) runTool(ctx context.Context, call llm.ToolCall) (string, string) {
	arguments := toolArguments{}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &arguments); err != nil && call.Function.Arguments != "" {
		return "Error | argumentos no válidos: " + err.Error(), ""
	}
	switch call.Function.Name {
	case ToolRead:
		pageOutline, err := turn.page.Outline(ctx)
		if err != nil {
			return "Error | read: " + err.Error(), ""
		}
		turn.shownOutline = pageOutline
		return strings.TrimSpace(pageOutline), ""

	case ToolOpen:
		route := strings.TrimSpace(arguments.Route)
		pageOutline, err := turn.page.Navigate(ctx, route)
		if err != nil {
			return "Error | " + route + ": " + err.Error(), ""
		}
		turn.shownOutline, turn.route = pageOutline, route
		return strings.TrimSpace(pageOutline), "open " + route

	case ToolAct:
		return turn.runActions(ctx, arguments.Actions)

	case ToolSearch:
		if strings.TrimSpace(arguments.Facts) == "" && strings.TrimSpace(arguments.Functions) == "" {
			return "Error | search: facts y functions están vacíos", ""
		}
		found, err := turn.search.Search(ctx, strings.TrimSpace(arguments.Facts), strings.TrimSpace(arguments.Functions), turn.route)
		if err != nil {
			return "Error | search: " + err.Error(), ""
		}
		for _, match := range signatureNamePattern.FindAllStringSubmatch(found, -1) {
			turn.searchedFunctions[match[1]] = true
		}
		return strings.TrimSpace(found), ""

	case ToolCall:
		if !turn.searchedFunctions[arguments.Function] {
			return "Error | " + arguments.Function + ": " + ErrorSearchFirst, ""
		}
		functionResult, err := turn.functions.Call(ctx, arguments.Function, arguments.Arguments)
		if err != nil {
			return "Error | " + err.Error(), ""
		}
		turn.functionResult = &functionResult
		if functionResult.Choice != nil {
			return functionResult.Choice.Summary, "call " + arguments.Function
		}
		return functionResult.Answer.Summary, "call " + arguments.Function

	case ToolCalc:
		return CalculateResult(arguments.Expression), ""
	}
	return "Error | herramienta desconocida: " + call.Function.Name, ""
}

// finalActionVerbPattern finds a verb of a final action — guardar, grabar, enviar, eliminar, borrar,
// anular, emitir, pagar, cobrar, entregar — in any person, tense or enclitic form ("guárdalo",
// "envíala", "págale", "entrégaselo"), over text lowercased and without accents. The endings are
// listed so most nouns that share a stem do not count ("página", "pago", "cobro", "envío",
// "borrador").
var finalActionVerbPattern = regexp.MustCompile(`\b(guard|grab|envi|elimin|borr|anul|emit|pagu?|cobr|entregu?)` +
	`(a|as|an|ar|ad|amos|ando|e|es|en|emos|ir|id|imos|iendo)((me|se|nos)?(lo|la|los|las|le|les)|me|nos|se)?\b`)

// nounLikeVerbForms are verb forms written exactly like a noun ("la entrega pendiente"): they never
// count, so "entrega el pedido" still asks "¿Lo entrego?" (the user's choice, 2026-11-27: stricter).
var nounLikeVerbForms = map[string]bool{"entrega": true, "entregas": true}

var accentFolder = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u")

// AsksForFinalAction reports whether the user's message asks explicitly for a save, send, delete,
// cancel, issue, payment or delivery: then a "!" action needs no question first. A fixed verb list,
// no reading of intent: "crea el cliente" or "regístralo" alone still asks "¿Lo guardo?".
func AsksForFinalAction(text string) bool {
	for _, verbForm := range finalActionVerbPattern.FindAllString(accentFolder.Replace(strings.ToLower(text)), -1) {
		if !nounLikeVerbForms[verbForm] {
			return true
		}
	}
	return false
}

// protectedButtonPattern: buttons that save, send, delete, pay or deliver. Clicking one needs "!" (the user's
// confirmation); Genix's buttons carry no role attribute, so the label decides.
var protectedButtonPattern = regexp.MustCompile(`(?i)^\s*Button ([0-9:]+) \| (guardar|grabar|eliminar|borrar|enviar|anular|emitir|pagar|cobrar|entregar|save|delete|send|pay|deliver)\b`)

// discardButtonPattern: close and cancel buttons. On a page with a save button they discard what the form
// holds, so they need "!" too: an "ok" meant for something else must never throw the user's data away.
var discardButtonPattern = regexp.MustCompile(`(?i)^\s*Button ([0-9:]+) \| (cerrar|cancelar|close|cancel)\b`)

func (turn *turnState) runActions(ctx context.Context, actionLines string) (string, string) {
	actions, err := ParseActions(actionLines)
	if err != nil {
		return "Error | " + err.Error(), ""
	}
	// The confirmation guard reads the live page: the model may act on ids from an older outline.
	livePage, err := turn.page.Outline(ctx)
	if err != nil {
		return "Error | act: " + err.Error(), ""
	}
	protectedIDs := map[string]bool{}
	discardIDs := []string{}
	for _, line := range strings.Split(livePage, "\n") {
		if match := protectedButtonPattern.FindStringSubmatch(line); match != nil {
			protectedIDs[match[1]] = true
		}
		if match := discardButtonPattern.FindStringSubmatch(line); match != nil {
			discardIDs = append(discardIDs, match[1])
		}
	}
	if len(protectedIDs) > 0 {
		for _, discardID := range discardIDs {
			protectedIDs[discardID] = true
		}
	}
	for _, action := range actions {
		needsConfirmation := action.Method == "click" && protectedIDs[action.ID]
		if (action.Confirmed && !turn.mayConfirm) || (needsConfirmation && !action.Confirmed) {
			return "Error | " + action.ID + ": requiere confirmación del usuario", ""
		}
	}

	results, nextOutline, err := turn.page.Invoke(ctx, actions)
	if err != nil {
		return "Error | act: " + err.Error(), ""
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
		actionLog = "act " + strings.Join(executed, "; ")
	}
	return strings.Join(lines, "\n"), actionLog
}
