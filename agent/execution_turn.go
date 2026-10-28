package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"app/core"
	"genix-agent/harness"
	"genix-agent/llm"
	"genix-agent/routing"
	"genix-agent/salestools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
)

// The execution stage of a discovered request, on the Eino harness. Discovery decides the tools
// and the routing context; this builds the agent from them, runs it, and closes the turn with the
// finish tool's message, a plain-text answer, or a pause the next request resumes.

const (
	// maxLoopIterations caps the model calls of one turn: the typical
	// "get_page → navigate → invoke_batch → finish" arc with a few retries.
	maxLoopIterations = 12
	// toolResultMaxBytes truncates a tool result before it goes back into the conversation. Page
	// snapshots are already reduced to 30 KB; this wider guard leaves room for their grammar.
	toolResultMaxBytes = 40_000
	// keepRecentToolRounds caps the tool rounds (assistant tool calls + their results) kept in the
	// prompt. A get_page snapshot can be ~10K tokens; the model keeps the latest page state and can
	// call get_page again for anything older.
	keepRecentToolRounds = 2
)

// executionSpec is what a turn's agent is built from. A paused turn stores it next to its
// checkpoint, because a resume must rebuild the very same agent and Eino's checkpoint holds only
// the conversation state.
type executionSpec struct {
	ModelID        string
	Instruction    string
	ToolNames      []string
	MutationRoutes []string
	Language       routing.Language
}

// RunTurn runs one discovered request with the backend-approved tools. The caller has already
// persisted the user message.
func (s *AgentSession) RunTurn(ctx context.Context, userText, modelHash string, language routing.Language,
	completedTurns []routing.CompletedTurn, routingContext string, availableTools []llm.Tool, mutationRoutes []string) error {

	modelID, err := harness.ResolveModelID(modelHash)
	if err != nil {
		return err
	}
	spec := executionSpec{
		ModelID:        modelID,
		Instruction:    turnInstruction(routingContext),
		MutationRoutes: mutationRoutes,
		Language:       language,
	}
	for _, availableTool := range availableTools {
		spec.ToolNames = append(spec.ToolNames, availableTool.Function.Name)
	}
	// D2: every turn with tools may answer sales figures. Its gate is GET.sale-summary's, which
	// access.toml leaves unmapped: any signed-in user reads it (TestSalesToolGateMatchesTheRoute).
	spec.ToolNames = append(spec.ToolNames, salestools.ToolName)
	core.Log("agent.turn execution model::", modelID, " tab::", shortTabID(s.TabID), " tools::", strings.Join(spec.ToolNames, ","),
		" mutation_routes::", strings.Join(mutationRoutes, ","), " page_connected::", IsConnected(s.TabID))
	return s.runExecution(ctx, spec, newCheckPointID(), turnMessages(completedTurns, userText), nil)
}

// RunReadOnlyTurn answers from supplied live context with no tools. It is used for builder
// inspection, never for builder edits.
func (s *AgentSession) RunReadOnlyTurn(ctx context.Context, userText, modelHash string, completedTurns []routing.CompletedTurn, routingContext string) error {
	modelID, err := harness.ResolveModelID(modelHash)
	if err != nil {
		return err
	}
	routingContext += "\nThis is read-only. No tools are available. Return the final answer as plain text and do not claim that an action was executed."
	spec := executionSpec{ModelID: modelID, Instruction: turnInstruction(routingContext)}
	return s.runExecution(ctx, spec, newCheckPointID(), turnMessages(completedTurns, userText), nil)
}

// TurnResumeRequest is the `Resume` of a p-agent-turn body: the answer to a paused turn.
type TurnResumeRequest struct {
	CheckPointID string
	// InterruptID is the question answered; empty continues a turn the clock paused.
	InterruptID string
	// Answer is the picked choice's ID. The tool that asked validates it against its own state.
	Answer string
}

// checkPointIDPattern keeps an ID echoed by the browser from addressing anything but a checkpoint
// file: it becomes part of a file store key.
var checkPointIDPattern = regexp.MustCompile(`^[a-z0-9]{6,20}$`)

// ResumeTurn continues a paused turn of this user with the user's answer.
func (s *AgentSession) ResumeTurn(ctx context.Context, resume TurnResumeRequest, userText string) error {
	if !checkPointIDPattern.MatchString(resume.CheckPointID) {
		return fmt.Errorf("invalid checkpoint id %q", resume.CheckPointID)
	}
	store := harness.CheckPointStore{CompanyID: s.CompanyID, UserID: s.UserID}
	specContent, err := store.GetTurnSpec(resume.CheckPointID)
	if err != nil {
		core.Log("agent.turn resume spec missing tab::", shortTabID(s.TabID), " checkpoint::", resume.CheckPointID, " err::", err)
		return s.completeTurn(routing.LocalizedResponse(routing.ResponseTurnFailed, routing.LanguageSpanish), "", 0)
	}
	spec := executionSpec{}
	if err := json.Unmarshal(specContent, &spec); err != nil {
		return fmt.Errorf("decode turn spec %s: %w", resume.CheckPointID, err)
	}
	if _, err := saveUserMessage(s, userText, s.CurrentRoute()); err != nil {
		return fmt.Errorf("persist user message: %w", err)
	}
	core.Log("agent.turn resume tab::", shortTabID(s.TabID), " checkpoint::", resume.CheckPointID, " interrupt::", resume.InterruptID)
	return s.completeExecutionFailure(
		s.runExecution(beginPromptTurn(ctx), spec, resume.CheckPointID, nil, &harness.TurnResume{InterruptID: resume.InterruptID, Answer: resume.Answer}),
		spec.Language,
	)
}

// runExecution builds the turn's agent from spec, runs or resumes it, and closes the turn.
func (s *AgentSession) runExecution(ctx context.Context, spec executionSpec, checkPointID string,
	messages []*schema.Message, resume *harness.TurnResume) error {

	chatModel, err := harness.ChatModel(ctx, spec.ModelID)
	if err != nil {
		return fmt.Errorf("chat model: %w", err)
	}
	tools, err := s.executionTools(spec)
	if err != nil {
		return err
	}
	store := harness.CheckPointStore{CompanyID: s.CompanyID, UserID: s.UserID}
	s.pushStatus("thinking", "Pensando…", 1, maxLoopIterations)

	modelCalls := 0
	result := harness.RunTurn(ctx, harness.TurnRequest{
		CheckPointID:  checkPointID,
		Store:         store,
		Model:         chatModel,
		Instruction:   spec.Instruction,
		Messages:      messages,
		Tools:         tools,
		EndsTurn:      endsTurn,
		Handlers:      []adk.ChatModelAgentMiddleware{&pageToolRoundsPruner{}},
		MaxIterations: maxLoopIterations,
		Resume:        resume,
		OnModelMessage: func(message *schema.Message) {
			modelCalls++
			for _, call := range message.ToolCalls {
				if call.Function.Name != llm.FinishToolName {
					s.pushStatus("acting", s.statusLabelFor(llm.ToolCall{Function: llm.ToolCallFunction{
						Name: call.Function.Name, Arguments: call.Function.Arguments,
					}}), modelCalls, maxLoopIterations)
				}
			}
		},
		OnToolResult: func(toolName, toolResult string) {
			core.Log("agent.turn tool result tab::", shortTabID(s.TabID), " name::", toolName, " bytes::", len(toolResult),
				" preview::", core.StrCut(toolResult, 500))
			if summary, failed := toolResultErrorSummary(toolResult); failed {
				// Recoverable failures stay in the visible action trail; the model still sees the
				// result and can retry.
				s.pushStatus("error", "Error: "+summary, modelCalls, maxLoopIterations)
			} else if toolName != llm.FinishToolName {
				s.pushStatus("thinking", "Pensando…", modelCalls+1, maxLoopIterations)
			}
		},
	})
	tokens := int32(result.Usage.Prompt + result.Usage.Completion)

	switch {
	case result.Err != nil:
		return result.Err
	case result.Interrupt != nil:
		return s.pauseTurn(ctx, store, spec, result.Interrupt, tokens)
	case result.EndToolName == llm.FinishToolName:
		message, summary, err := parseFinishArgs(result.EndToolResult)
		if err != nil {
			return fmt.Errorf("parse finish args: %w (raw=%s)", err, result.EndToolResult)
		}
		return s.completeTurn(message, summary, tokens)
	case result.EndToolName == salestools.ToolName:
		answer := salestools.Answer{}
		if err := json.Unmarshal([]byte(result.EndToolResult), &answer); err != nil {
			return fmt.Errorf("decode the sales answer: %w", err)
		}
		return s.completeAnswer(answer, tokens)
	case strings.TrimSpace(result.FinalMessage) != "":
		// The prompt asks for `finish`, but a plain answer is still the answer.
		return s.completeTurn(strings.TrimSpace(result.FinalMessage), "", tokens)
	default:
		return errors.New("the turn ended with no finish call and no answer")
	}
}

// pauseTurn stores the spec next to the checkpoint and closes the turn with the question (or the
// "it took too long" notice) and what the browser needs to resume it.
func (s *AgentSession) pauseTurn(ctx context.Context, store harness.CheckPointStore, spec executionSpec,
	interrupt *harness.TurnInterrupt, tokens int32) error {

	specContent, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("encode turn spec: %w", err)
	}
	if err := store.SetTurnSpec(ctx, interrupt.CheckPointID, specContent); err != nil {
		return fmt.Errorf("store turn spec: %w", err)
	}
	pause := &ChatTurnPause{CheckPointID: interrupt.CheckPointID, InterruptID: interrupt.InterruptID}
	message := localizedStatus(spec.Language, "La consulta tomó más tiempo del previsto. ¿Continúo?",
		"This is taking longer than expected. Shall I continue?")
	if question, isChoice := interrupt.Info.(harness.ChoiceQuestion); isChoice {
		message, pause.Choices = question.Question, question.Choices
	} else if !interrupt.ByDeadline {
		return fmt.Errorf("unsupported interrupt %T", interrupt.Info)
	}
	timestamp, err := saveAgentMessage(s, message, "", tokens)
	if err != nil {
		return fmt.Errorf("persist agent message: %w", err)
	}
	s.sendJSON(ChatTypeAgentReply, ChatAgentReply{Message: message, Timestamp: timestamp, Pause: pause})
	return nil
}

// endsTurn: finish always ends the turn; the sales tool only when it answered, since its errors go
// back to the model to fix the statement.
func endsTurn(toolName, toolResult string) bool {
	switch toolName {
	case llm.FinishToolName:
		return true
	case salestools.ToolName:
		_, failed := toolResultErrorSummary(toolResult)
		return !failed
	}
	return false
}

// executionTools builds the spec's tools: the sales tool, and the page tools from their llm
// declarations.
func (s *AgentSession) executionTools(spec executionSpec) ([]tool.BaseTool, error) {
	tools := make([]tool.BaseTool, 0, len(spec.ToolNames))
	for _, toolName := range spec.ToolNames {
		if toolName == salestools.ToolName {
			tools = append(tools, salestools.FrameSQLTool{CompanyID: s.CompanyID, Language: string(spec.Language)})
			continue
		}
		declaration, found := pageToolDeclarations[toolName]
		if !found {
			return nil, fmt.Errorf("unknown tool %q in the turn spec", toolName)
		}
		toolInfo, err := toolInfoFromDeclaration(declaration)
		if err != nil {
			return nil, err
		}
		tools = append(tools, pageTool{session: s, info: toolInfo, mutationRoutes: spec.MutationRoutes})
	}
	return tools, nil
}

var pageToolDeclarations = map[string]llm.Tool{
	llm.FinishToolName:      llm.FinishTool,
	llm.GetPageToolName:     llm.GetPageTool,
	llm.GetMenuToolName:     llm.GetMenuTool,
	llm.NavigateToolName:    llm.NavigateTool,
	llm.InvokeBatchToolName: llm.InvokeBatchTool,
}

// toolInfoFromDeclaration keeps the JSON schemas the prompts were tuned against (llm/prompts.go)
// instead of re-deriving them from Go structs.
func toolInfoFromDeclaration(declaration llm.Tool) (*schema.ToolInfo, error) {
	parametersJSON, err := json.Marshal(declaration.Function.Parameters)
	if err != nil {
		return nil, err
	}
	parameters := &jsonschema.Schema{}
	if err := json.Unmarshal(parametersJSON, parameters); err != nil {
		return nil, fmt.Errorf("tool %s schema: %w", declaration.Function.Name, err)
	}
	return &schema.ToolInfo{
		Name:        declaration.Function.Name,
		Desc:        declaration.Function.Description,
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(parameters),
	}, nil
}

// pageTool is one browser tool. finish only echoes its arguments: it returns directly, and the
// harness hands them back as the turn's result.
type pageTool struct {
	session        *AgentSession
	info           *schema.ToolInfo
	mutationRoutes []string
}

func (pageTool pageTool) Info(_ context.Context) (*schema.ToolInfo, error) { return pageTool.info, nil }

func (pageTool pageTool) InvokableRun(ctx context.Context, argumentsJSON string, _ ...tool.Option) (string, error) {
	if pageTool.info.Name == llm.FinishToolName {
		return argumentsJSON, nil
	}
	toolResult := pageTool.session.dispatchTool(ctx, llm.ToolCall{Function: llm.ToolCallFunction{
		Name: pageTool.info.Name, Arguments: argumentsJSON,
	}}, pageTool.mutationRoutes)
	return clipForLLM(toolResult), nil
}

// pageToolRoundsPruner keeps the prompt from growing with every page snapshot: before each model
// call it drops all but the last keepRecentToolRounds tool rounds and strips the snapshot from every
// tool result but the latest. Both are idempotent, so rewriting the stored state is safe.
type pageToolRoundsPruner struct {
	adk.BaseChatModelAgentMiddleware
}

func (*pageToolRoundsPruner) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState,
	_ *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	state.Messages = prunePageToolRounds(state.Messages, keepRecentToolRounds)
	// Local dev: the exact prompt about to be sent, for post-hoc inspection.
	LogExecutorPrompt(ctx, state.Messages, state.ToolInfos)
	return ctx, state, nil
}

// prunePageToolRounds works on the tail after the first tool round; the system prompt, history and
// user message before it are never touched. A round's tool results always go with its assistant
// message: the protocol needs every tool_call_id answered in the same prompt.
func prunePageToolRounds(messages []*schema.Message, keep int) []*schema.Message {
	roundStarts := []int{}
	for index, message := range messages {
		if message.Role == schema.Assistant && len(message.ToolCalls) > 0 {
			roundStarts = append(roundStarts, index)
		}
	}
	if len(roundStarts) > keep {
		cutFrom := roundStarts[len(roundStarts)-keep]
		baseLen := roundStarts[0]
		pruned := make([]*schema.Message, 0, baseLen+len(messages)-cutFrom)
		pruned = append(pruned, messages[:baseLen]...)
		messages = append(pruned, messages[cutFrom:]...)
	}
	lastToolIndex := -1
	for index, message := range messages {
		if message.Role == schema.Tool {
			lastToolIndex = index
		}
	}
	for index, message := range messages {
		if message.Role == schema.Tool && index != lastToolIndex {
			if stripped := stripPageSnapshotFromToolResult(message.Content); stripped != message.Content {
				strippedMessage := *message
				strippedMessage.Content = stripped
				messages[index] = &strippedMessage
			}
		}
	}
	return messages
}

// turnInstruction keeps retrieved evidence in the system message, where pruning never reaches.
func turnInstruction(routingContext string) string {
	if routingContext = strings.TrimSpace(routingContext); routingContext != "" {
		return llm.SystemPromptChat + "\n\n" + routingContext
	}
	return llm.SystemPromptChat
}

// turnMessages replays the planner-selected turns, each agent reply followed by the note of the
// actions it already took, then the user's message.
func turnMessages(completedTurns []routing.CompletedTurn, currentMessage string) []*schema.Message {
	messages := []*schema.Message{}
	for _, completedTurn := range completedTurns {
		if userMessage := strings.TrimSpace(completedTurn.UserMessage); userMessage != "" {
			messages = append(messages, schema.UserMessage(userMessage))
		}
		if assistantMessage := strings.TrimSpace(completedTurn.AssistantMessage); assistantMessage != "" {
			messages = append(messages, schema.AssistantMessage(assistantMessage, nil))
			if actionNote := actionLogNote(completedTurn.ActionSummary); actionNote != "" {
				messages = append(messages, schema.SystemMessage(actionNote))
			}
		}
	}
	return append(messages, schema.UserMessage(strings.TrimSpace(currentMessage)))
}

func newCheckPointID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
