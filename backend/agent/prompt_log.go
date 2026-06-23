package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"app/agent/llm"
	"app/core"
)

// Prompt logging is a local-dev aid: every LLM request the chat loop sends
// gets dumped to a flat .txt file so the developer can inspect exactly what
// reached the model on each iteration. Disabled in any non-local environment.
//
// Layout:
//   <project>/tmp/promps/                 (created once at app startup)
//   <project>/tmp/promps/YYYY_MM_DD/      (created lazily the first write of the day)
//   <project>/tmp/promps/YYYY_MM_DD/<userID>_promp_<unixNano>.txt
//
// The parent tmp/promps directory is created in InitPromptLog so the per-
// write path doesn't pay for that check. The daily subdirectory is cached
// behind a mutex and only re-created when the date rolls over.

var (
	promptLogMu       sync.Mutex
	promptLogTodayDir string // cached "<project>/tmp/promps/YYYY_MM_DD" for the current date
)

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

// LogPrompt persists the message slice the chat loop is about to send to
// the LLM. No-op outside local dev. Failures are logged but never propagate
// — prompt logging is observability, not correctness.
func LogPrompt(userID int32, messages []llm.Message) {
	if !core.Env.IS_LOCAL {
		return
	}
	now := time.Now()
	dir, err := ensurePromptLogDailyDir(now)
	if err != nil {
		core.Log("agent.prompt-log ensure-daily-dir failed::", err)
		return
	}
	name := fmt.Sprintf("%d_promp_%d.txt", userID, now.UnixNano())
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(formatPromptMessages(messages)), 0o644); err != nil {
		core.Log("agent.prompt-log write failed:: path::", path, " err::", err)
		return
	}
	core.Log("agent.prompt-log promp", name, "saved in", dir)
}

// ensurePromptLogDailyDir returns the cached YYYY_MM_DD subdir, or creates
// it (and updates the cache) on the first write of a new day. The mutex
// keeps concurrent turns from racing on MkdirAll.
func ensurePromptLogDailyDir(now time.Time) (string, error) {
	today := now.Format("2006_01_02")
	promptLogMu.Lock()
	defer promptLogMu.Unlock()
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

func promptLogRoot() string {
	return filepath.Join(core.ProjectTmpDir(), "promps")
}

// formatPromptMessages renders the messages array as a readable transcript:
// one section per message with role + tool-call expansion, so the file can
// be skimmed without a JSON viewer.
func formatPromptMessages(messages []llm.Message) string {
	var b strings.Builder
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
	return b.String()
}
