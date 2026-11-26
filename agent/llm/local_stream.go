package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// The local provider is a llama-server (genix-local-ai) reached over the tailnet. Its streamed
// answer carries what the chat's progress bar shows: `prompt_progress` while the prompt is encoded
// and `timings` while tokens are decoded. Callers stay non-streaming — Eino's chat model and
// Client.Chat post a plain request and read a plain chat.completion — because LocalStreamTransport
// asks llama-server for the stream itself, reports the progress to the turn's sink and hands back
// the assembled completion.
//
// Each call also carries X-Genix-* headers naming its turn, call number and stage, so the model log
// on the llama-server side (genix-local-ai's model-proxy) groups the calls of one user message.

// LocalProgress is one progress report of a local model call.
type LocalProgress struct {
	Phase           string // "prefill" (encoding the prompt) | "thinking" | "answer"
	PromptTokens    int
	PromptProcessed int // prompt tokens ready: reused from the cache plus encoded
	PromptMs        float64
	GeneratedTokens int
	TokensPerSecond float64 // decoding speed
}

// localProgressInterval caps the reports per call; a phase change is always reported.
const localProgressInterval = 250 * time.Millisecond

// LocalTurn is what a turn hands its local model calls: who they belong to, for the model log, and
// where their progress goes.
type LocalTurn struct {
	ID        string
	SessionID int64
	CompanyID int32
	UserID    int32
	Report    func(LocalProgress)
	calls     atomic.Int32
}

type localTurnKey struct{}
type localStageKey struct{}

// WithLocalTurn makes every local model call made with ctx belong to turn.
func WithLocalTurn(ctx context.Context, turn *LocalTurn) context.Context {
	return context.WithValue(ctx, localTurnKey{}, turn)
}

// LocalTurnFrom returns the turn ctx belongs to, nil outside a turn.
func LocalTurnFrom(ctx context.Context) *LocalTurn {
	turn, _ := ctx.Value(localTurnKey{}).(*LocalTurn)
	return turn
}

// WithLocalStage names the stage ("planner", "executor", "pagebuilder") of the calls made with ctx.
func WithLocalStage(ctx context.Context, stage string) context.Context {
	return context.WithValue(ctx, localStageKey{}, stage)
}

// LocalStreamTransport is the http.RoundTripper of the local provider.
type LocalStreamTransport struct{}

// localStreamChunk is the part of a llama-server chat.completion.chunk this file reads.
type localStreamChunk struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Timings *struct {
		CacheN             int     `json:"cache_n"`
		PromptN            int     `json:"prompt_n"`
		PromptMs           float64 `json:"prompt_ms"`
		PredictedN         int     `json:"predicted_n"`
		PredictedPerSecond float64 `json:"predicted_per_second"`
	} `json:"timings"`
	PromptProgress *struct {
		Total     int     `json:"total"`
		Cache     int     `json:"cache"`
		Processed int     `json:"processed"`
		TimeMs    float64 `json:"time_ms"`
	} `json:"prompt_progress"`
}

// localToolCall is one tool call being assembled from its streamed pieces.
type localToolCall struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

func (LocalStreamTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	requestBody := map[string]any{}
	if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
		return nil, fmt.Errorf("local model: decode request: %w", err)
	}
	request.Body.Close()
	requestBody["stream"] = true
	requestBody["return_progress"] = true
	requestBody["timings_per_token"] = true
	streamBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("local model: encode request: %w", err)
	}
	streamRequest := request.Clone(request.Context())
	streamRequest.Body = io.NopCloser(bytes.NewReader(streamBody))
	streamRequest.ContentLength = int64(len(streamBody))
	turn, _ := request.Context().Value(localTurnKey{}).(*LocalTurn)
	if turn != nil {
		stage, _ := request.Context().Value(localStageKey{}).(string)
		streamRequest.Header.Set("X-Genix-Turn", turn.ID)
		streamRequest.Header.Set("X-Genix-Call", strconv.Itoa(int(turn.calls.Add(1))))
		streamRequest.Header.Set("X-Genix-Stage", stage)
		streamRequest.Header.Set("X-Genix-Session", strconv.FormatInt(turn.SessionID, 10))
		streamRequest.Header.Set("X-Genix-Company", strconv.Itoa(int(turn.CompanyID)))
		streamRequest.Header.Set("X-Genix-User", strconv.Itoa(int(turn.UserID)))
	}

	response, err := http.DefaultTransport.RoundTrip(streamRequest)
	if err != nil || response.StatusCode != http.StatusOK {
		// An error answer is plain JSON, which the caller already reads.
		return response, err
	}
	defer response.Body.Close()

	var report func(LocalProgress)
	if turn != nil {
		report = turn.Report
	}
	completion, err := assembleLocalStream(response.Body, report)
	if err != nil {
		return nil, err
	}
	completionBody, err := json.Marshal(completion)
	if err != nil {
		return nil, fmt.Errorf("local model: encode completion: %w", err)
	}
	return &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Proto:         response.Proto,
		ProtoMajor:    response.ProtoMajor,
		ProtoMinor:    response.ProtoMinor,
		Header:        http.Header{"Content-Type": {"application/json"}},
		Body:          io.NopCloser(bytes.NewReader(completionBody)),
		ContentLength: int64(len(completionBody)),
		Request:       request,
	}, nil
}

// assembleLocalStream reads the SSE stream into the chat.completion a non-streaming call returns,
// reporting progress on the way. The usage comes from the last timings: llama-server only sends a
// `usage` object to streams that ask for it.
func assembleLocalStream(stream io.Reader, report func(LocalProgress)) (map[string]any, error) {
	progress := LocalProgress{}
	lastReportAt := time.Time{}

	var content, reasoningContent strings.Builder
	toolCalls := []*localToolCall{}
	completion := map[string]any{"object": "chat.completion"}
	finishReason := ""
	promptTokens, completionTokens := 0, 0

	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		data, isData := strings.CutPrefix(scanner.Text(), "data: ")
		if !isData || data == "[DONE]" {
			continue
		}
		chunk := localStreamChunk{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("local model: decode chunk: %w (chunk=%s)", err, truncate(data, 300))
		}
		completion["id"], completion["created"], completion["model"] = chunk.ID, chunk.Created, chunk.Model

		phase := progress.Phase
		if chunk.PromptProgress != nil {
			phase = "prefill"
			progress.PromptTokens = chunk.PromptProgress.Total
			progress.PromptProcessed = chunk.PromptProgress.Cache + chunk.PromptProgress.Processed
			progress.PromptMs = chunk.PromptProgress.TimeMs
		}
		for _, choice := range chunk.Choices {
			content.WriteString(choice.Delta.Content)
			reasoningContent.WriteString(choice.Delta.ReasoningContent)
			if choice.Delta.ReasoningContent != "" {
				phase = "thinking"
			}
			if choice.Delta.Content != "" || len(choice.Delta.ToolCalls) > 0 {
				phase = "answer"
			}
			// A tool call arrives in pieces keyed by index: the first carries its id and name, the
			// rest append to its arguments.
			for _, toolCallDelta := range choice.Delta.ToolCalls {
				for len(toolCalls) <= toolCallDelta.Index {
					toolCalls = append(toolCalls, &localToolCall{})
				}
				toolCall := toolCalls[toolCallDelta.Index]
				if toolCallDelta.ID != "" {
					toolCall.ID = toolCallDelta.ID
				}
				if toolCallDelta.Function.Name != "" {
					toolCall.Name = toolCallDelta.Function.Name
				}
				toolCall.Arguments.WriteString(toolCallDelta.Function.Arguments)
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}
		if chunk.Timings != nil {
			promptTokens = chunk.Timings.CacheN + chunk.Timings.PromptN
			completionTokens = chunk.Timings.PredictedN
			progress.GeneratedTokens = chunk.Timings.PredictedN
			progress.TokensPerSecond = chunk.Timings.PredictedPerSecond
			if chunk.Timings.PromptMs > 0 {
				progress.PromptMs = chunk.Timings.PromptMs
			}
		}

		if report != nil && (phase != progress.Phase || time.Since(lastReportAt) >= localProgressInterval) {
			progress.Phase = phase
			report(progress)
			lastReportAt = time.Now()
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("local model: read stream: %w", err)
	}

	message := map[string]any{"role": "assistant", "content": content.String()}
	if reasoningContent.Len() > 0 {
		message["reasoning_content"] = reasoningContent.String()
	}
	if len(toolCalls) > 0 {
		assembledToolCalls := make([]ToolCall, len(toolCalls))
		for index, toolCall := range toolCalls {
			assembledToolCalls[index] = ToolCall{ID: toolCall.ID, Type: "function",
				Function: ToolCallFunction{Name: toolCall.Name, Arguments: toolCall.Arguments.String()}}
		}
		message["tool_calls"] = assembledToolCalls
	}
	completion["choices"] = []map[string]any{{"index": 0, "message": message, "finish_reason": finishReason}}
	completion["usage"] = map[string]any{
		"prompt_tokens": promptTokens, "completion_tokens": completionTokens, "total_tokens": promptTokens + completionTokens,
	}
	return completion, nil
}
