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
	"strconv"
	"strings"
	"time"

	"app/core"
	"genix-agent/genixbot"
	"genix-agent/genixbot/outline"
	"genix-agent/llm"
	"genix-agent/routing"
)

// A turn of a [[models]] entry with contract = "genix-bot": the fine-tuned model carries Genix's
// rules, pages and vocabulary in its weights, so the turn skips the discovery planner and the
// execution prompt and runs genixbot.RunTurn against this tab. Pages reach the model as compact
// outlines, never as the snapshot XML.

const factsRequestTimeout = 10 * time.Second

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
	result, err := genixbot.RunTurn(llm.WithLocalStage(ctx, "genix-bot"), model, page, genixBotFacts{},
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
	s.sendJSON(ChatTypeAgentReply, ChatAgentReply{Message: result.Answer, Summary: summary, Timestamp: timestamp})
	return nil
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

// genixBotArguments turns a `hacer` argument into the component method's arguments: option ids are
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

// genixBotFacts calls the `hechos` search of agent.local_facts_url.
type genixBotFacts struct{}

func (genixBotFacts) Search(ctx context.Context, consulta, ruta string) (string, error) {
	if core.Env == nil || core.Env.LOCAL_FACTS_URL == "" {
		return "", errors.New("agent.local_facts_url not set in config.toml")
	}
	body, _ := json.Marshal(map[string]string{"consulta": consulta, "ruta": ruta})
	requestContext, cancel := context.WithTimeout(ctx, factsRequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, core.Env.LOCAL_FACTS_URL, bytes.NewReader(body))
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
		return "", fmt.Errorf("hechos HTTP %d: %s", response.StatusCode, core.StrCut(string(text), 200))
	}
	return string(text), nil
}

// genixBotStatusLabel is the chat's progress line for one genix-bot tool call.
func genixBotStatusLabel(call llm.ToolCall) string {
	arguments := map[string]string{}
	_ = json.Unmarshal([]byte(call.Function.Arguments), &arguments)
	switch call.Function.Name {
	case genixbot.ToolVer:
		return "Leyendo la página…"
	case genixbot.ToolIr:
		return "Navegando hacia '" + arguments["ruta"] + "'…"
	case genixbot.ToolHacer:
		if actions, err := genixbot.ParseActions(arguments["acciones"]); err == nil && len(actions) == 1 {
			return methodLabel(actions[0].Method)
		} else if err == nil {
			return fmt.Sprintf("Ejecutando %d acciones…", len(actions))
		}
		return "Ejecutando acciones…"
	case genixbot.ToolHechos:
		return "Consultando la documentación…"
	case genixbot.ToolCalcular:
		return "Calculando…"
	}
	return "Trabajando…"
}
