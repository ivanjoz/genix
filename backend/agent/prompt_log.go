package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"app/agent/llm"
	"app/core"
)

// Prompt logging is a local-dev aid: every planner exchange and executor
// request is dumped to a readable file. Disabled outside local development.
//
// Layout:
//   <project>/tmp/promps/                 (created once at app startup)
//   <project>/tmp/promps/YYYY_MM_DD/      (created lazily the first write of the day)
//   <project>/tmp/promps/YYYY_MM_DD/.index
//   <project>/tmp/promps/YYYY_MM_DD/<turn>_promp_<call>_<stage>.txt
//
// The parent tmp/promps directory is created in InitPromptLog so the per-
// write path doesn't pay for that check. The daily subdirectory is cached
// behind a mutex and only re-created when the date rolls over.

var (
	promptLogMu       sync.Mutex
	promptLogTodayDir string // cached "<project>/tmp/promps/YYYY_MM_DD" for the current date
)

const promptLogIndexName = ".index"

type promptTurnLogContextKey struct{}

type promptTurnLog struct {
	directory  string
	turn       int
	nextPrompt int
}

// InitPromptLog ensures tmp/promps exists. Called once at app startup so
// the per-write path can skip the parent-dir check. No-op outside local dev.
func InitPromptLog() {
	if !core.Env.IS_LOCAL {
		core.Log("agent.prompt-log disabled (IS_LOCAL=false)")
		return
	}
	promptLogRoot := promptLogRoot()
	if err := os.MkdirAll(promptLogRoot, 0o755); err != nil {
		core.Log("agent.prompt-log mkdir root failed::", " root::", promptLogRoot, " err::", err)
		return
	}
	core.Log("agent.prompt-log ready root::", promptLogRoot)
}

// beginPromptTurn reserves one persistent daily turn number. All planner and
// executor calls derived from the same user message share this context value.
func beginPromptTurn(ctx context.Context) context.Context {
	if !core.Env.IS_LOCAL {
		return ctx
	}
	promptLogMu.Lock()
	defer promptLogMu.Unlock()

	directory, err := ensurePromptLogDailyDirLocked(time.Now())
	if err != nil {
		core.Log("agent.prompt-log ensure-daily-dir failed::", err)
		return ctx
	}
	turn, err := allocateNextPromptTurn(directory)
	if err != nil {
		core.Log("agent.prompt-log allocate-turn failed::", err)
		return ctx
	}
	core.Log("agent.prompt-log turn started::", turn, " dir::", directory)
	return context.WithValue(ctx, promptTurnLogContextKey{}, &promptTurnLog{directory: directory, turn: turn})
}

// LogPlannerPrompt stores both the planner request and its raw response so
// discovery decisions remain auditable before validation and execution.
func LogPlannerPrompt(ctx context.Context, attempt int, messages []llm.Message, response, plannerError string) {
	stage := "planner"
	if attempt > 1 {
		stage = "planner_retry"
	}
	writePromptLog(ctx, stage, formatPromptExchange(messages, nil, response, plannerError))
}

// LogExecutorPrompt stores the exact messages and tool schemas sent on one
// executor iteration; later iterations include prior tool calls and results.
func LogExecutorPrompt(ctx context.Context, messages []llm.Message, tools []llm.Tool) {
	writePromptLog(ctx, "executor", formatPromptExchange(messages, tools, "", ""))
}

// writePromptLog increments the call number inside one reserved user turn.
// Failures remain observable but never affect the user-facing agent turn.
func writePromptLog(ctx context.Context, stage, content string) {
	if !core.Env.IS_LOCAL {
		return
	}
	turnLog, ok := ctx.Value(promptTurnLogContextKey{}).(*promptTurnLog)
	if !ok {
		core.Log("agent.prompt-log missing turn context stage::", stage)
		return
	}
	promptLogMu.Lock()
	defer promptLogMu.Unlock()

	turnLog.nextPrompt++
	name := promptLogFileName(turnLog.turn, turnLog.nextPrompt, stage)
	path := filepath.Join(turnLog.directory, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		core.Log("agent.prompt-log write failed:: path::", path, " err::", err)
		return
	}
	core.Log("agent.prompt-log promp", name, "saved in", turnLog.directory)
}

// ensurePromptLogDailyDirLocked returns the cached daily directory. Caller
// holds promptLogMu so sequence allocation and file creation stay ordered.
func ensurePromptLogDailyDirLocked(now time.Time) (string, error) {
	today := now.Format("2006_01_02")
	if promptLogTodayDir != "" && filepath.Base(promptLogTodayDir) == today {
		return promptLogTodayDir, nil
	}
	dir := filepath.Join(promptLogRoot(), today)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	promptLogTodayDir = dir
	return dir, nil
}

// allocateNextPromptTurn persists the last allocated daily turn. If .index
// does not exist yet, existing readable logs seed the migration safely.
func allocateNextPromptTurn(directory string) (int, error) {
	indexPath := filepath.Join(directory, promptLogIndexName)
	lastTurn := 0
	indexContent, err := os.ReadFile(indexPath)
	switch {
	case err == nil:
		lastTurn, err = strconv.Atoi(strings.TrimSpace(string(indexContent)))
		if err != nil || lastTurn < 0 {
			return 0, fmt.Errorf("parse %s: invalid turn index", indexPath)
		}
	case os.IsNotExist(err):
		lastTurn, err = maximumExistingPromptTurn(directory)
		if err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("read %s: %w", indexPath, err)
	}

	nextTurn := lastTurn + 1
	if err := os.WriteFile(indexPath, []byte(strconv.Itoa(nextTurn)+"\n"), 0o644); err != nil {
		return 0, fmt.Errorf("write %s: %w", indexPath, err)
	}
	return nextTurn, nil
}

func maximumExistingPromptTurn(directory string) (int, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return 0, err
	}
	maximumTurn := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.Contains(name, "_promp_") || !strings.HasSuffix(name, ".txt") {
			continue
		}
		turnText, _, found := strings.Cut(name, "_promp_")
		if !found {
			continue
		}
		turn, conversionError := strconv.Atoi(turnText)
		if conversionError == nil && turn > maximumTurn {
			maximumTurn = turn
		}
	}
	return maximumTurn, nil
}

func promptLogFileName(turn, call int, stage string) string {
	return fmt.Sprintf("%d_promp_%d_%s.txt", turn, call, stage)
}

func promptLogRoot() string {
	return filepath.Join(core.ProjectTmpDir(), "promps")
}

// formatPromptExchange renders messages, actual tool schemas, and the raw
// planner result in one file that can be inspected without a JSON viewer.
func formatPromptExchange(messages []llm.Message, tools []llm.Tool, response, exchangeError string) string {
	var b strings.Builder
	b.WriteString("=== REQUEST MESSAGES ===\n")
	for i, m := range messages {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "=== [%d] role=%s", i, m.Role)
		if m.ToolCallID != "" {
			fmt.Fprintf(&b, " tool_call_id=%s", m.ToolCallID)
		}
		b.WriteString(" ===\n")
		if m.Content != "" {
			b.WriteString(m.Content)
			if !strings.HasSuffix(m.Content, "\n") {
				b.WriteByte('\n')
			}
		}
		for _, tc := range m.ToolCalls {
			fmt.Fprintf(&b, "[tool_call id=%s name=%s args=%s]\n", tc.ID, tc.Function.Name, tc.Function.Arguments)
		}
	}
	if tools != nil {
		b.WriteString("\n=== REQUEST TOOLS ===\n")
		encodedTools, err := json.MarshalIndent(tools, "", "  ")
		if err != nil {
			fmt.Fprintf(&b, "<tool serialization failed: %v>\n", err)
		} else {
			b.Write(encodedTools)
			b.WriteByte('\n')
		}
	}
	if response != "" || exchangeError != "" {
		b.WriteString("\n=== RAW RESPONSE ===\n")
		if response != "" {
			b.WriteString(response)
			if !strings.HasSuffix(response, "\n") {
				b.WriteByte('\n')
			}
		}
		if exchangeError != "" {
			fmt.Fprintf(&b, "ERROR: %s\n", exchangeError)
		}
	}
	return b.String()
}
