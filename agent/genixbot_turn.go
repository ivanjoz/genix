package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"genix-agent/datafunctions"
	"genix-agent/genixbot"
	"genix-agent/genixbot/outline"
	"genix-agent/harness"
	"genix-agent/llm"
	"genix-agent/routing"
)

// A turn of a [[models]] entry with contract = "genix-bot": the fine-tuned model carries Genix's
// rules, pages and vocabulary in its weights, so the turn skips the discovery planner and the
// execution prompt and runs genixbot.RunTurn against this tab. Pages reach the model as compact
// outlines, never as the snapshot XML.

const searchRequestTimeout = 10 * time.Second

// runGenixBotTurn persists the user message, runs the turn and persists the answer with the turn's
// steps, which the next turn replays.
func (s *AgentSession) runGenixBotTurn(ctx context.Context, modelConfig llm.ModelConfig, userText string) error {
	history, err := loadGenixBotHistory(s)
	if err != nil {
		return fmt.Errorf("load genix-bot history: %w", err)
	}
	route := s.CurrentRoute()
	if _, err := saveUserMessage(s, userText, route); err != nil {
		return fmt.Errorf("persist user message: %w", err)
	}
	client, err := llm.NewClientForProvider(modelConfig.Provider, modelConfig.ID)
	if err != nil {
		return fmt.Errorf("genix-bot client: %w", err)
	}
	model := &genixBotModel{client: client}
	s.pushStatus("thinking", "Pensando…", 1, genixbot.MaxSteps)
	step := 1
	page := &genixBotPage{session: s}
	if turn := llm.LocalTurnFrom(ctx); turn != nil {
		page.turnID = turn.ID
	}
	accesses := genixBotAccesses()
	functions := genixBotFunctions{deps: datafunctions.DatabaseDeps(s.CompanyID), accesses: accesses}
	result, err := genixbot.RunTurn(llm.WithLocalStage(ctx, "genix-bot"), model, page, genixBotSearch{accesses: accesses}, functions,
		history, route, userText, func(call llm.ToolCall) {
			step++
			s.pushStatus("acting", genixBotStatusLabel(call), step, genixbot.MaxSteps)
		})
	if err != nil {
		core.Log("agent.genix-bot turn failed tab::", shortTabID(s.TabID), " steps::", len(result.Steps), " err::", err)
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseTurnFailed, routing.LanguageSpanish), "", model.tokens)
	}

	stepsJSON, err := json.Marshal(result.Steps)
	if err != nil {
		return fmt.Errorf("encode genix-bot steps: %w", err)
	}
	summary := strings.Join(result.Actions, " · ")
	timestamp, err := saveMessage(s, RoleAgent, result.Answer, summary, string(stepsJSON), model.tokens)
	if err != nil {
		return fmt.Errorf("persist agent message: %w", err)
	}
	s.sendJSON(ChatTypeAgentReply, genixBotReply(result.Answer, summary, timestamp, result.Display, result.Choice))
	return nil
}

// genixBotReply is a turn's reply: a data answer's tables and charts, or a choice card. The card has
// no checkpoint: its Resume comes back with an empty CheckPointID (resumeGenixBotChoice).
func genixBotReply(message, summary string, timestamp int64, display any, choice *genixbot.FunctionChoice) ChatAgentReply {
	reply := ChatAgentReply{Message: message, Summary: summary, Timestamp: timestamp}
	if answer, isDataAnswer := display.(datafunctions.Answer); isDataAnswer {
		reply.Tables, reply.Charts = answer.Tables, answer.Charts
	}
	if choice != nil {
		reply.Pause = &ChatTurnPause{}
		for _, option := range choice.Options {
			reply.Pause.Choices = append(reply.Pause.Choices, harness.Choice(option))
		}
	}
	return reply
}

// loadGenixBotHistory pairs the session's last user and agent rows into replayable turns: the user
// row keeps the route of its turn, the agent row the turn's steps (AttachedContent).
func loadGenixBotHistory(s *AgentSession) ([]genixbot.CompletedTurn, error) {
	rows, err := loadLastN(s, genixbot.HistoryTurns*4)
	if err != nil {
		return nil, err
	}
	history := []genixbot.CompletedTurn{}
	var pendingUser *genixbot.CompletedTurn
	for _, row := range rows {
		switch row.Role {
		case RoleUser:
			pendingUser = &genixbot.CompletedTurn{Route: strings.TrimSpace(row.AttachedContent), UserText: strings.TrimSpace(row.Message)}
		case RoleAgent:
			if pendingUser == nil {
				continue
			}
			pendingUser.Answer = strings.TrimSpace(row.Message)
			if row.AttachedContent != "" {
				if err := json.Unmarshal([]byte(row.AttachedContent), &pendingUser.Steps); err != nil {
					return nil, fmt.Errorf("decode the steps of the agent row %d: %w", row.Timestamp, err)
				}
			}
			history = append(history, *pendingUser)
			pendingUser = nil
		}
	}
	return history, nil
}

// genixBotModel calls the model greedily, as it was evaluated, and counts the turn's tokens.
type genixBotModel struct {
	client *llm.Client
	tokens int32
}

func (model *genixBotModel) Next(ctx context.Context, messages []llm.Message, tools []llm.Tool) (llm.Message, error) {
	response, err := model.client.Chat(ctx, llm.ChatRequest{
		Messages: messages, Tools: tools, ToolChoice: "auto",
		MaxTokens: 600, TopK: 1, Samplers: []string{"top_k"},
	})
	if err != nil {
		return llm.Message{}, err
	}
	model.tokens += response.Usage.TotalTokens
	if len(response.Choices) == 0 {
		return llm.Message{}, errors.New("genix-bot: response without choices")
	}
	return response.Choices[0].Message, nil
}

// genixBotPage is the tab's page as outlines.
type genixBotPage struct {
	session *AgentSession
	// turnID names the turn's page log files, the same id the model log records (X-Genix-Turn).
	turnID    string
	pagesRead int
}

// logPage keeps, in local development, every page a turn read: route, HTML and components, the input
// of ParsePageHTML. genix-local-ai turns them into simulator states, so evaluation and training can
// replay the pages real sessions saw. Never fails the turn.
func (page *genixBotPage) logPage(content PageContent, route string) {
	if !core.Env.IS_DEV_ARG || page.turnID == "" {
		return
	}
	page.pagesRead++
	directory := filepath.Join(core.ProjectTmpDir(), "genixbot-pages", time.Now().Format("2006_01_02"))
	encoded, err := json.Marshal(map[string]any{"Route": route, "SessionID": page.session.SessionID, "HTML": content.HTML, "Components": content.Components})
	if err == nil {
		err = os.MkdirAll(directory, 0o755)
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(directory, fmt.Sprintf("%s_%02d.json", page.turnID, page.pagesRead)), encoded, 0o644)
	}
	if err != nil {
		core.Log("agent.genix-bot page log failed::", err)
	}
}

func (page *genixBotPage) Outline(ctx context.Context) (string, error) {
	content, err := GetPageContent(ctx, page.session.TabID)
	if err != nil {
		return "", err
	}
	page.logPage(content, page.session.CurrentRoute())
	return genixBotOutline(content, page.session.CurrentRoute())
}

func (page *genixBotPage) Navigate(ctx context.Context, route string) (string, error) {
	menu, err := GetMenu(ctx, page.session.TabID)
	if err != nil {
		return "", err
	}
	if !menuContainsRoute(menu, route) {
		return "", errors.New("la ruta no está en el menú")
	}
	navigation, err := NavigateWithPage(ctx, page.session.TabID, route, true)
	if err != nil {
		return "", err
	}
	page.session.setCurrentRoute(route)
	if navigation.Page == nil {
		return "", errors.New("la página no devolvió su contenido")
	}
	page.logPage(*navigation.Page, route)
	return genixBotOutline(*navigation.Page, route)
}

func (page *genixBotPage) Invoke(ctx context.Context, actions []genixbot.Action) ([]genixbot.ActionResult, string, error) {
	invocations := make([]InvokePayload, 0, len(actions))
	for _, action := range actions {
		handleID, method, arguments, err := resolveTarget(action.ID, action.Method, genixBotArguments(action))
		if err != nil {
			return nil, "", err
		}
		invocations = append(invocations, InvokePayload{HandleID: handleID, Method: method, Args: arguments})
	}
	batch, err := InvokeBatch(ctx, page.session.TabID, invocations, true)
	if err != nil {
		return nil, "", err
	}
	results := make([]genixbot.ActionResult, len(batch.Results))
	for index, invocation := range batch.Results {
		results[index] = genixbot.ActionResult{Error: actionError(invocation.Error), Value: optionsText(invocation.Value)}
		if !invocation.OK && invocation.Error == "" {
			results[index].Error = "la acción falló"
		}
	}
	if batch.Page == nil {
		return nil, "", errors.New("la página no devolvió su contenido")
	}
	page.logPage(*batch.Page, page.session.CurrentRoute())
	pageOutline, err := genixBotOutline(*batch.Page, page.session.CurrentRoute())
	return results, pageOutline, err
}

var (
	unknownHandlePattern = regexp.MustCompile(`^unknown handle id: `)
	missingMethodPattern = regexp.MustCompile(`^handle \S+ \((\w+)\) does not implement (\w+)$`)
)

// actionError words the browser's generic failures the way genix-bot was trained to read them;
// component-specific errors pass through.
func actionError(browserError string) string {
	if unknownHandlePattern.MatchString(browserError) {
		return genixbot.ErrorUnknownID
	}
	if match := missingMethodPattern.FindStringSubmatch(browserError); match != nil {
		return genixbot.ErrorNoMethod(match[1], match[2])
	}
	return browserError
}

// genixBotArguments turns an `act` argument into the component method's arguments: option ids are
// numbers, a row is its composite id, a value is text.
func genixBotArguments(action genixbot.Action) []any {
	if action.Argument == "" {
		return []any{}
	}
	if action.Method == "select" || action.Method == "remove" {
		if optionID, err := strconv.Atoi(action.Argument); err == nil {
			return []any{optionID}
		}
	}
	return []any{action.Argument}
}

// optionsText renders the options a search returned the way outlines list them: "[id] label, …".
func optionsText(value json.RawMessage) string {
	options := []AgentOption{}
	if len(value) == 0 || json.Unmarshal(value, &options) != nil || options == nil {
		return ""
	}
	labels := make([]string, len(options))
	for index, option := range options {
		labels[index] = fmt.Sprintf("[%v] %v", option.ID, option.Value)
	}
	return genixbot.SearchMatches(labels)
}

// genixBotOutline parses the browser's DOM with the full snapshot (no row or byte limits: the
// outline caps rows itself) and renders it as the outline genix-bot was trained on.
func genixBotOutline(content PageContent, route string) (string, error) {
	snapshot, err := ParsePageHTML(content.HTML, content.Components)
	if err != nil {
		return "", fmt.Errorf("parse page: %w", err)
	}
	components := make([]outline.Component, len(content.Components))
	for index, component := range content.Components {
		components[index].ID = component.ID
		for _, option := range component.Options {
			components[index].Options = append(components[index].Options, outline.ComponentOption{ID: option.ID, Value: option.Value})
		}
	}
	return outline.Render(snapshot, content.HTML, outline.Options{Route: route, OptionLabels: outline.OptionLabelsFromComponents(components)})
}

// genixBotSearch calls the docindex `/search` of agent.local_search_url, listing only the functions
// the user's accesses allow.
type genixBotSearch struct{ accesses []string }

func (search genixBotSearch) Search(ctx context.Context, facts, functions, route string) (string, error) {
	if core.Env == nil || core.Env.LOCAL_SEARCH_URL == "" {
		return "", errors.New("agent.local_search_url not set in config.toml")
	}
	body, _ := json.Marshal(map[string]any{"facts": facts, "functions": functions, "route": route, "access": search.accesses})
	requestContext, cancel := context.WithTimeout(ctx, searchRequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, core.Env.LOCAL_SEARCH_URL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	text, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search HTTP %d: %s", response.StatusCode, core.StrCut(string(text), 200))
	}
	return string(text), nil
}

// genixBotAccesses are the API permissions of the data functions the user may call. Every
// function's access is unmapped in access.toml, so any signed-in user holds them all
// (TestDataFunctionAccessesAreUnmapped); a function behind a mapped access needs the user's grants
// read here first.
func genixBotAccesses() []string {
	accesses := []string{}
	for _, function := range datafunctions.Functions {
		if !slices.Contains(accesses, function.Access) {
			accesses = append(accesses, function.Access)
		}
	}
	return accesses
}

// genixBotFunctions runs the datafunctions catalog over one company's data.
type genixBotFunctions struct {
	deps     datafunctions.Deps
	accesses []string
}

func (functions genixBotFunctions) Call(ctx context.Context, name string, arguments map[string]any) (genixbot.FunctionResult, error) {
	result, err := functions.run(ctx, name, arguments)
	return genixBotFunctionResult(result), err
}

// genixBotFunctionResult is a datafunctions result as genix-bot reads it; the answer travels whole as
// the Display, so the reply gets its tables and charts back.
func genixBotFunctionResult(result datafunctions.Result) genixbot.FunctionResult {
	switch {
	case result.Choice != nil:
		choice := &genixbot.FunctionChoice{Question: result.Choice.Question, Summary: result.Choice.Summary()}
		for _, option := range result.Choice.Options {
			choice.Options = append(choice.Options, genixbot.ChoiceOption(option))
		}
		return genixbot.FunctionResult{Choice: choice}
	case result.Answer != nil:
		return genixbot.FunctionResult{Answer: &genixbot.FunctionAnswer{Message: result.Answer.Message, Summary: result.Answer.Summary, Display: *result.Answer}}
	}
	return genixbot.FunctionResult{}
}

func (functions genixBotFunctions) run(ctx context.Context, name string, arguments map[string]any) (datafunctions.Result, error) {
	index := slices.IndexFunc(datafunctions.Functions, func(function datafunctions.Function) bool { return function.Name == name })
	if index >= 0 && !slices.Contains(functions.accesses, datafunctions.Functions[index].Access) {
		return datafunctions.Result{}, fmt.Errorf("el usuario no tiene acceso a %s", name)
	}
	return datafunctions.Functions.Run(ctx, functions.deps, name, arguments)
}

// errNoPendingChoice: the session's last agent row is not a choice card of a `call`.
var errNoPendingChoice = errors.New("no pending genix-bot choice")

// resumeGenixBotChoice answers a choice card with no model call (genix-local-ai plan 13 §4.4): the
// last agent row's pending `call` runs again with the ambiguous name replaced by the pick, and the
// row is rewritten as if the model had called with the pick — the call with "#id" and its Summary,
// the answer as the reply — so the history reads as one trajectory. A second ambiguous name asks
// again the same way. The click is not stored as a user row: the rewritten turn already holds it.
func (s *AgentSession) resumeGenixBotChoice(ctx context.Context, answer string) error {
	rows, err := loadLastN(s, 1)
	if err != nil {
		return fmt.Errorf("load the pending choice: %w", err)
	}
	steps := []llm.Message{}
	if len(rows) == 0 || rows[0].Role != RoleAgent || json.Unmarshal([]byte(rows[0].AttachedContent), &steps) != nil {
		core.Log("agent.genix-bot resume without a pending choice tab::", shortTabID(s.TabID))
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseTurnFailed, routing.LanguageSpanish), "", 0)
	}
	functions := genixBotFunctions{deps: datafunctions.DatabaseDeps(s.CompanyID), accesses: genixBotAccesses()}
	result, steps, err := resumeFunctionCall(ctx, functions, steps, answer)
	if err != nil {
		core.Log("agent.genix-bot resume failed tab::", shortTabID(s.TabID), " err::", err)
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseTurnFailed, routing.LanguageSpanish), "", 0)
	}
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return fmt.Errorf("encode genix-bot steps: %w", err)
	}
	row := rows[0]
	row.AttachedContent, row.Updated = string(stepsJSON), core.SUnixTime()
	var display any
	if result.Choice != nil {
		row.Message = result.Choice.Question
	} else {
		row.Message, display = result.Answer.Message, result.Answer.Display
	}
	if err := db.Upsert(&[]coreTypes.AgentMessage{row}); err != nil {
		return fmt.Errorf("rewrite the agent message: %w", err)
	}
	// The browser orders its rows by timestamp: the reply comes after the click it answers.
	s.sendJSON(ChatTypeAgentReply, genixBotReply(row.Message, row.Summary, time.Now().UnixMilli(), display, result.Choice))
	return nil
}

// resumeFunctionCall runs the pending `call` of a turn's steps with the user's pick: the call runs
// again to get its choice (the browser's answer is checked against those options, never trusted as
// is), the ambiguous name becomes the picked "#id"s, and the call runs with them. It returns the
// result and the steps rewritten with the new arguments and the result's tool message.
func resumeFunctionCall(ctx context.Context, functions genixBotFunctions, steps []llm.Message, answer string) (genixbot.FunctionResult, []llm.Message, error) {
	if len(steps) < 2 || len(steps[len(steps)-2].ToolCalls) == 0 || !strings.HasPrefix(steps[len(steps)-1].Content, "Pregunta |") {
		return genixbot.FunctionResult{}, nil, errNoPendingChoice
	}
	callStep := steps[len(steps)-2]
	pendingCall := callStep.ToolCalls[len(callStep.ToolCalls)-1]
	callArguments := struct {
		Function  string         `json:"function"`
		Arguments map[string]any `json:"arguments"`
	}{}
	if pendingCall.Function.Name != genixbot.ToolCall || json.Unmarshal([]byte(pendingCall.Function.Arguments), &callArguments) != nil {
		return genixbot.FunctionResult{}, nil, errNoPendingChoice
	}
	result, err := functions.run(ctx, callArguments.Function, callArguments.Arguments)
	if err != nil {
		return genixbot.FunctionResult{}, nil, err
	}
	if choice := result.Choice; choice != nil {
		pickedRecords, isOffered := choice.PickedRecords(answer)
		if !isOffered {
			return genixbot.FunctionResult{}, nil, fmt.Errorf("the answer %q is not an option of %q", answer, choice.Question)
		}
		names, _ := callArguments.Arguments[choice.Argument].([]any)
		replaced := []any{}
		for _, name := range names {
			if name == choice.Name {
				for _, record := range pickedRecords {
					replaced = append(replaced, record)
				}
				continue
			}
			replaced = append(replaced, name)
		}
		callArguments.Arguments[choice.Argument] = replaced
		if result, err = functions.run(ctx, callArguments.Function, callArguments.Arguments); err != nil {
			return genixbot.FunctionResult{}, nil, err
		}
	}
	encodedArguments, err := json.Marshal(callArguments)
	if err != nil {
		return genixbot.FunctionResult{}, nil, err
	}
	functionResult := genixBotFunctionResult(result)
	toolMessage := ""
	if functionResult.Choice != nil {
		toolMessage = functionResult.Choice.Summary
	} else {
		toolMessage = functionResult.Answer.Summary
	}
	// Copies: the steps given are left as they were.
	rewritten := slices.Clone(steps)
	rewritten[len(rewritten)-2].ToolCalls = slices.Clone(callStep.ToolCalls)
	rewritten[len(rewritten)-2].ToolCalls[len(callStep.ToolCalls)-1].Function.Arguments = string(encodedArguments)
	rewritten[len(rewritten)-1].Content = toolMessage
	return functionResult, rewritten, nil
}

// genixBotStatusLabel is the chat's progress line for one genix-bot tool call.
func genixBotStatusLabel(call llm.ToolCall) string {
	arguments := struct{ Route, Actions string }{}
	_ = json.Unmarshal([]byte(call.Function.Arguments), &arguments)
	switch call.Function.Name {
	case genixbot.ToolRead:
		return "Leyendo la página…"
	case genixbot.ToolOpen:
		return "Navegando hacia '" + arguments.Route + "'…"
	case genixbot.ToolAct:
		if actions, err := genixbot.ParseActions(arguments.Actions); err == nil && len(actions) == 1 {
			return methodLabel(actions[0].Method)
		} else if err == nil {
			return fmt.Sprintf("Ejecutando %d acciones…", len(actions))
		}
		return "Ejecutando acciones…"
	case genixbot.ToolSearch:
		// One label for both queries: a search of functions alone is not documentation.
		return "Buscando…"
	case genixbot.ToolCall:
		return "Consultando tus datos…"
	case genixbot.ToolCalc:
		return "Calculando…"
	}
	return "Trabajando…"
}
