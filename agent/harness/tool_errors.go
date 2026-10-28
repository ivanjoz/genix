package harness

import (
	"context"
	"errors"
	"strings"

	"app/core"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
)

// ToolErrorsAsText returns a failing tool's error to the model as its result, instead of letting
// it end the turn: by default an Eino tool error rises through the graph and kills the run. A model
// that reads "route not in the menu" picks another route; one that never sees it cannot.
//
// Control signals are not swallowed: an interrupt (how a tool asks the user and Eino checkpoints)
// and a cancellation (the turn's clock) must stop the run. It also turns an empty argument string
// into `{}`, which InferTool cannot parse when every parameter is optional.
func ToolErrorsAsText() compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
				if strings.TrimSpace(input.Arguments) == "" {
					input.Arguments = "{}"
				}
				output, err := next(ctx, input)
				if err == nil || isControlSignal(err) || core.IsCreditRateLimitError(err) {
					return output, err
				}
				core.Log("agent.harness tool failed, returned to the model::", input.Name, " err::", err)
				return &compose.ToolOutput{Result: toolFailureText(input.Name, err)}, nil
			}
		},
	}
}

func isControlSignal(err error) bool {
	if _, isInterrupt := compose.IsInterruptRerunError(err); isInterrupt {
		return true
	}
	if _, isInterrupt := compose.ExtractInterruptInfo(err); isInterrupt {
		return true
	}
	var cancelError *adk.CancelError
	if errors.As(err, &cancelError) {
		return true
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// toolFailureText drops the graph's wrapper ("[LocalFunc] failed to invoke tool, toolName=…"),
// which tells the model nothing, and keeps the error in the JSON shape the page tools already use.
func toolFailureText(toolName string, err error) string {
	reason := err.Error()
	if cut := strings.LastIndex(reason, "err="); cut >= 0 {
		reason = reason[cut+len("err="):]
	}
	if cut := strings.Index(reason, "\n------------------------"); cut >= 0 {
		reason = reason[:cut]
	}
	return core.ToJsonNoErr(map[string]string{
		"error": strings.TrimSpace(reason),
		"hint":  "The " + toolName + " call was not made. Do not repeat it unchanged: fix what the error says or try another way.",
	})
}
