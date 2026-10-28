package harness

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// scriptedModel answers each call with the next scripted message, so the runner's wiring (return
// directly, interrupt, checkpoint, resume) is tested without a provider.
type scriptedModel struct {
	mutex   sync.Mutex
	replies []*schema.Message
	calls   int
}

func (scripted *scriptedModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	scripted.mutex.Lock()
	defer scripted.mutex.Unlock()
	reply := scripted.replies[scripted.calls]
	scripted.calls++
	return reply, nil
}

func (scripted *scriptedModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	panic("not used")
}

func (scripted *scriptedModel) WithTools([]*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return scripted, nil
}

type memoryStore struct{ checkPoints map[string][]byte }

func (store *memoryStore) Get(_ context.Context, id string) ([]byte, bool, error) {
	content, found := store.checkPoints[id]
	return content, found, nil
}

func (store *memoryStore) Set(_ context.Context, id string, content []byte) error {
	store.checkPoints[id] = content
	return nil
}

func toolCallMessage(toolName, arguments string) *schema.Message {
	return schema.AssistantMessage("", []schema.ToolCall{{ID: "call-" + toolName, Function: schema.FunctionCall{Name: toolName, Arguments: arguments}}})
}

func withUsage(message *schema.Message) *schema.Message {
	message.ResponseMeta = &schema.ResponseMeta{Usage: &schema.TokenUsage{PromptTokens: 100, CompletionTokens: 10}}
	return message
}

// pickTool asks the user once, then answers with the picked ID.
type pickTool struct{}

func (pickTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "pick", Desc: "asks the user to pick"}, nil
}

func (pickTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	wasInterrupted, _, _ := tool.GetInterruptState[any](ctx)
	if !wasInterrupted {
		return "", tool.StatefulInterrupt(ctx, ChoiceQuestion{Question: "¿Cuál?", Choices: []Choice{{ID: "7", Label: "Siete"}}}, "pending")
	}
	_, _, answer := tool.GetResumeContext[string](ctx)
	return `{"picked":"` + answer + `"}`, nil
}

type finishTool struct{}

func (finishTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "finish", Desc: "ends the turn"}, nil
}

func (finishTool) InvokableRun(_ context.Context, arguments string, _ ...tool.Option) (string, error) {
	return arguments, nil
}

func endsOnFinish(toolName, _ string) bool { return toolName == "finish" }

// reportTool fails its first call and answers the next, like a FrameSQL call the model fixes.
type reportTool struct{ calls *int }

func (reportTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "report", Desc: "answers with a report"}, nil
}

func (report reportTool) InvokableRun(context.Context, string, ...tool.Option) (string, error) {
	*report.calls++
	if *report.calls == 1 {
		return "", errors.New("unknown column")
	}
	return `{"message":"reporte"}`, nil
}

func TestRunTurnSendsAToolErrorBackAndEndsOnItsAnswer(t *testing.T) {
	reportCalls := 0
	scripted := &scriptedModel{replies: []*schema.Message{toolCallMessage("report", `{}`), toolCallMessage("report", `{}`)}}
	result := RunTurn(context.Background(), TurnRequest{
		CheckPointID: "t3", Store: &memoryStore{checkPoints: map[string][]byte{}}, Model: scripted, Instruction: "x",
		Messages: []*schema.Message{schema.UserMessage("ventas")}, Tools: []tool.BaseTool{reportTool{calls: &reportCalls}},
		EndsTurn: func(toolName, toolResult string) bool {
			return toolName == "report" && !strings.HasPrefix(toolResult, `{"error"`)
		},
		MaxIterations: 4,
	})
	if result.Err != nil || result.EndToolName != "report" || result.EndToolResult != `{"message":"reporte"}` {
		t.Fatalf("unexpected end: %+v", result)
	}
	// The failed call went back to the model once; the answer ended the turn with no third call.
	if scripted.calls != 2 {
		t.Fatalf("model calls = %d, want 2", scripted.calls)
	}
}

func TestRunTurnEndsWithTheFinishTool(t *testing.T) {
	scripted := &scriptedModel{replies: []*schema.Message{withUsage(toolCallMessage("finish", `{"message":"listo"}`))}}
	result := RunTurn(context.Background(), TurnRequest{
		CheckPointID: "t1", Store: &memoryStore{checkPoints: map[string][]byte{}}, Model: scripted,
		Instruction: "a literal {brace} must survive", Messages: []*schema.Message{schema.UserMessage("hola")},
		Tools: []tool.BaseTool{finishTool{}}, EndsTurn: endsOnFinish, MaxIterations: 4,
	})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if result.EndToolName != "finish" || result.EndToolResult != `{"message":"listo"}` {
		t.Fatalf("unexpected end: %+v", result)
	}
	if result.Usage.Calls != 1 || result.Usage.Prompt != 100 {
		t.Fatalf("usage not summed: %+v", result.Usage)
	}
}

func TestRunTurnPausesOnAChoiceAndResumesFromTheCheckpoint(t *testing.T) {
	store := &memoryStore{checkPoints: map[string][]byte{}}
	scripted := &scriptedModel{replies: []*schema.Message{
		toolCallMessage("pick", `{}`),
		toolCallMessage("finish", `{"message":"elegiste"}`),
	}}
	request := TurnRequest{
		CheckPointID: "t2", Store: store, Model: scripted, Instruction: "x",
		Messages: []*schema.Message{schema.UserMessage("vende")},
		Tools:    []tool.BaseTool{pickTool{}, finishTool{}}, EndsTurn: endsOnFinish, MaxIterations: 4,
	}
	paused := RunTurn(context.Background(), request)
	if paused.Err != nil || paused.Interrupt == nil {
		t.Fatalf("expected a pause, got %+v", paused)
	}
	question, isChoice := paused.Interrupt.Info.(ChoiceQuestion)
	if !isChoice || question.Choices[0].ID != "7" || paused.Interrupt.InterruptID == "" {
		t.Fatalf("unexpected interrupt: %+v", paused.Interrupt)
	}
	if _, stored := store.checkPoints["t2"]; !stored {
		t.Fatal("no checkpoint stored")
	}

	request.Messages = nil
	request.Resume = &TurnResume{InterruptID: paused.Interrupt.InterruptID, Answer: "7"}
	resumed := RunTurn(context.Background(), request)
	if resumed.Err != nil || resumed.EndToolName != "finish" {
		t.Fatalf("resume did not finish: %+v", resumed)
	}
	// The resume lands inside the tool: no model call was repeated, only the one after the tool.
	if scripted.calls != 2 {
		t.Fatalf("model calls = %d, want 2", scripted.calls)
	}
}

func TestToolFailureTextKeepsTheErrorKey(t *testing.T) {
	decoded := map[string]string{}
	if err := json.Unmarshal([]byte(toolFailureText("navigate", errNavigate)), &decoded); err != nil || decoded["error"] != "route missing" {
		t.Fatalf("unexpected failure text: %v %v", decoded, err)
	}
}

var errNavigate = &navigateError{}

type navigateError struct{}

func (*navigateError) Error() string {
	return "[LocalFunc] failed to invoke tool, toolName=navigate, err=route missing"
}
