package harness

import (
	"context"
	"errors"
	"fmt"
	"time"

	"app/core"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// Every agent turn goes through RunTurn: it is the one place that creates Eino's Runner, charges
// the tokens, pauses before the deadline and turns the event stream into a TurnResult. Callers
// never touch adk.Runner themselves.

// TurnRequest is one turn, new or resumed.
type TurnRequest struct {
	// CheckPointID names this turn's checkpoint. A resumed turn passes the ID it paused under.
	CheckPointID string
	// Store is CheckPointStore in the app; tests pass an in-memory one.
	Store adk.CheckPointStore
	Model model.ToolCallingChatModel
	// Instruction is the system prompt, sent literally: Eino formats an instruction as an f-string
	// once session values exist, so a `{` in a prompt (JSON examples) would break the turn.
	Instruction string
	// Messages is the conversation to answer: the related history plus the user's message. Ignored
	// on a resume, whose messages are in the checkpoint.
	Messages []*schema.Message
	Tools    []tool.BaseTool
	// EndsTurn decides, per result, whether a tool's result ends the turn with no further model
	// call: finish always does, the sales tool only when it answered (its errors go back to the
	// model, which fixes its call). Eino's ReturnDirectly decides by name before the tool runs,
	// so it can't tell the two apart.
	EndsTurn func(toolName, toolResult string) bool
	// Handlers rewrite the model's input (tool-round pruning) or wrap tool calls.
	Handlers      []adk.ChatModelAgentMiddleware
	MaxIterations int
	// Resume continues the paused turn CheckPointID instead of starting one.
	Resume *TurnResume
	// OnModelMessage sees every assistant message, before its tool calls run: progress labels.
	OnModelMessage func(message *schema.Message)
	// OnToolResult sees every tool result.
	OnToolResult func(toolName, result string)
}

// TurnResume is the answer to a pause. An empty InterruptID resumes a turn the clock paused.
type TurnResume struct {
	InterruptID string
	Answer      any
}

// TurnResult is how the turn ended. Exactly one of FinalMessage, EndToolName or Interrupt is the
// outcome when Err is nil.
type TurnResult struct {
	FinalMessage string
	// EndToolName and EndToolResult are the tool result EndsTurn accepted.
	EndToolName   string
	EndToolResult string
	Interrupt     *TurnInterrupt
	Usage         TokenUsage
	Err           error
}

// TurnInterrupt is a paused turn: a tool asked the user something (Info), or the clock paused it
// (ByDeadline). Either way the checkpoint is stored and the next request resumes CheckPointID.
type TurnInterrupt struct {
	CheckPointID string
	InterruptID  string
	Info         any
	ByDeadline   bool
}

// TokenUsage sums the provider's token counts over the turn's model calls.
type TokenUsage struct {
	Prompt     int
	Completion int
	Calls      int
}

const (
	// pauseMargin is reserved before the deadline to stop at a safe point: the model call in flight
	// finishes, Eino serializes the graph and the checkpoint reaches S3.
	pauseMargin = 40 * time.Second
	// safePointWait is how long the pause waits for the tool calls in flight before stopping at
	// once, which also checkpoints but loses the model call in progress.
	safePointWait = 15 * time.Second
)

// RunTurn runs one turn to its end, pause or failure.
func RunTurn(ctx context.Context, request TurnRequest) (result TurnResult) {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "genix",
		Description: "Genix in-app assistant",
		Instruction: request.Instruction,
		Model:       request.Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               request.Tools,
				ToolCallMiddlewares: []compose.ToolMiddleware{ToolErrorsAsText()},
			},
		},
		GenModelInput: func(_ context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			return append([]adk.Message{schema.SystemMessage(instruction)}, input.Messages...), nil
		},
		// The end check runs first, so no other handler rewrites a prompt that is never sent.
		Handlers:      append([]adk.ChatModelAgentMiddleware{&turnEndCheck{endsTurn: request.EndsTurn}}, request.Handlers...),
		MaxIterations: request.MaxIterations,
	})
	if err != nil {
		return TurnResult{Err: fmt.Errorf("build agent: %w", err)}
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, CheckPointStore: request.Store})
	cancelOption, cancelTurn := adk.WithCancel()
	stopPauseTimer := schedulePause(ctx, cancelTurn, request.CheckPointID)
	defer stopPauseTimer()
	runOptions := []adk.AgentRunOption{adk.WithCheckPointID(request.CheckPointID), cancelOption}

	var events *adk.AsyncIterator[*adk.AgentEvent]
	switch {
	case request.Resume == nil:
		events = runner.Run(ctx, request.Messages, runOptions...)
	case request.Resume.InterruptID == "":
		events, err = runner.Resume(ctx, request.CheckPointID, runOptions...)
	default:
		events, err = runner.ResumeWithParams(ctx, request.CheckPointID, &adk.ResumeParams{
			Targets: map[string]any{request.Resume.InterruptID: request.Resume.Answer},
		}, runOptions...)
	}
	if err != nil {
		return TurnResult{Err: fmt.Errorf("resume turn %s: %w", request.CheckPointID, err)}
	}

	consumeTurnEvents(ctx, request, events, cancelTurn, &result)
	if result.Err == nil && result.Interrupt == nil && ctx.Err() != nil {
		result.Err = fmt.Errorf("turn ended by its context: %w", ctx.Err())
	}
	core.Log("agent.harness turn end checkpoint::", request.CheckPointID, " model_calls::", result.Usage.Calls,
		" tokens_in::", result.Usage.Prompt, " tokens_out::", result.Usage.Completion,
		" interrupted::", result.Interrupt != nil, " err::", result.Err)
	return result
}

// schedulePause asks the turn to stop at a safe point shortly before the context's deadline, so a
// turn that runs long leaves a checkpoint to continue from instead of dying with the Lambda.
func schedulePause(ctx context.Context, cancelTurn adk.AgentCancelFunc, checkPointID string) func() {
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline || time.Until(deadline) <= pauseMargin {
		return func() {}
	}
	pauseTimer := time.AfterFunc(time.Until(deadline)-pauseMargin, func() {
		if ctx.Err() != nil {
			return
		}
		core.Log("agent.harness turn near its deadline, pausing with a checkpoint::", checkPointID)
		cancelTurn(adk.WithAgentCancelMode(adk.CancelAfterToolCalls), adk.WithAgentCancelTimeout(safePointWait))
	})
	return func() { pauseTimer.Stop() }
}

// consumeTurnEvents drains the agent's events into the result. Each assistant message's token
// usage is charged as it arrives (D5): a refused charge stops the turn at once.
func consumeTurnEvents(ctx context.Context, request TurnRequest, events *adk.AsyncIterator[*adk.AgentEvent],
	cancelTurn adk.AgentCancelFunc, result *TurnResult) {

	for {
		event, more := events.Next()
		if !more {
			return
		}
		if event == nil {
			continue
		}

		// A pause arrives as an Interrupted action when a tool asked, or as a CancelError when the
		// clock asked. Both left a checkpoint; neither is a failure.
		var cancelError *adk.CancelError
		var turnEnded *turnEndedError
		if event.Err != nil && errors.As(event.Err, &turnEnded) {
			result.EndToolName, result.EndToolResult = turnEnded.toolName, turnEnded.toolResult
			continue
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			result.Interrupt = rootInterrupt(request.CheckPointID, event.Action.Interrupted)
			continue
		}
		if event.Err != nil && errors.As(event.Err, &cancelError) {
			if result.Err == nil && result.Interrupt == nil {
				result.Interrupt = &TurnInterrupt{CheckPointID: request.CheckPointID, ByDeadline: true}
			}
			continue
		}
		if event.Err != nil {
			if result.Interrupt == nil && result.Err == nil {
				result.Err = event.Err
			}
			continue
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		message, err := event.Output.MessageOutput.GetMessage()
		if err != nil || message == nil {
			continue
		}

		switch message.Role {
		case schema.Assistant:
			if message.ResponseMeta != nil && message.ResponseMeta.Usage != nil {
				usage := message.ResponseMeta.Usage
				result.Usage.Prompt += usage.PromptTokens
				result.Usage.Completion += usage.CompletionTokens
				result.Usage.Calls++
				if chargeError := core.ChargeInferenceUsage(ctx, usage.PromptTokens, usage.CompletionTokens); chargeError != nil {
					result.Err = fmt.Errorf("charge inference usage: %w", chargeError)
					cancelTurn(adk.WithAgentCancelMode(adk.CancelImmediate))
					continue
				}
			}
			if request.OnModelMessage != nil {
				request.OnModelMessage(message)
			}
			if message.Content != "" && len(message.ToolCalls) == 0 {
				result.FinalMessage = message.Content
			}
		case schema.Tool:
			toolName := event.Output.MessageOutput.ToolName
			if request.OnToolResult != nil {
				request.OnToolResult(toolName, message.Content)
			}
		}
	}
}

// turnEndedError stops the agent before a model call whose input ends with a tool result that
// EndsTurn accepts. It is the turn's outcome, not a failure.
type turnEndedError struct {
	toolName   string
	toolResult string
}

func (ended *turnEndedError) Error() string { return "turn ended by the tool " + ended.toolName }

// turnEndCheck reads the tool results of the last round before each model call.
type turnEndCheck struct {
	adk.BaseChatModelAgentMiddleware
	endsTurn func(toolName, toolResult string) bool
}

func (check *turnEndCheck) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState,
	_ *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	if check.endsTurn == nil {
		return ctx, state, nil
	}
	for index := len(state.Messages) - 1; index >= 0 && state.Messages[index].Role == schema.Tool; index-- {
		toolMessage := state.Messages[index]
		if check.endsTurn(toolMessage.ToolName, toolMessage.Content) {
			return ctx, state, &turnEndedError{toolName: toolMessage.ToolName, toolResult: toolMessage.Content}
		}
	}
	return ctx, state, nil
}

// rootInterrupt picks the interrupt point that caused the pause: that is the ID a resume targets.
func rootInterrupt(checkPointID string, info *adk.InterruptInfo) *TurnInterrupt {
	for _, interruptContext := range info.InterruptContexts {
		if interruptContext.IsRootCause {
			return &TurnInterrupt{CheckPointID: checkPointID, InterruptID: interruptContext.ID, Info: interruptContext.Info}
		}
	}
	return &TurnInterrupt{CheckPointID: checkPointID, ByDeadline: true}
}
