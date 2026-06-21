package webpage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"app/agent/llm"
	"app/core"
)

// Loop logging is a local-dev aid for the page-builder agent. Every RunTurn
// gets its own "code" and a folder named after it; inside, one numbered file
// is written for each context-send to the model — the main loop iterations AND
// the generate_svg / find_image subagents — plus a file for each tool result
// and the final apply_sections payload. The numeric prefix preserves the exact
// chronological order, so the folder reads top-to-bottom as "what happened" in
// the turn. Disabled (every method a no-op) in any non-local environment.
//
// Layout:
//   /tmp/promps/webpage/                                  (created once, lazily)
//   /tmp/promps/webpage/YYYY_MM_DD/<code>/                (one folder per turn)
//   /tmp/promps/webpage/YYYY_MM_DD/<code>/00_meta.txt
//   /tmp/promps/webpage/YYYY_MM_DD/<code>/01_main_iter1.txt
//   /tmp/promps/webpage/YYYY_MM_DD/<code>/02_tool_find_image.txt
//   /tmp/promps/webpage/YYYY_MM_DD/<code>/03_subagent_find_image_select.txt
//   …

const builderLogRoot = "/tmp/promps/webpage"

var (
	builderLogInitOnce sync.Once
	builderLogReady    bool
	// builderLogSeq disambiguates two turns that start in the same second.
	builderLogSeq atomic.Uint64
)

func initBuilderLog() {
	builderLogInitOnce.Do(func() {
		if !core.Env.IS_LOCAL {
			return
		}
		if err := os.MkdirAll(builderLogRoot, 0o755); err != nil {
			core.Log("agent.webpage loop-log mkdir root failed:: root::", builderLogRoot, " err::", err)
			return
		}
		builderLogReady = true
	})
}

// turnLog is the per-RunTurn logger. A nil *turnLog is a valid no-op logger
// (returned when local-dev logging is disabled), so every method must tolerate
// a nil receiver — callers never branch on whether logging is on.
type turnLog struct {
	dir   string // the loop folder
	code  string // the loop code (also the folder name)
	model string // resolved active model, for files where req.Model is blank
	seq   int    // file counter; the chronological prefix on each write
}

// newTurnLog allocates a turn code, creates its folder, and returns the logger.
// Returns nil (no-op) outside local dev or if the folder can't be created.
func newTurnLog(modeID int, activeModel string) *turnLog {
	if !core.Env.IS_LOCAL {
		return nil
	}
	initBuilderLog()
	if !builderLogReady {
		return nil
	}
	now := time.Now()
	// code = HHMMSS_<mode>_<seq%1000>: short, sortable, unique within a second.
	n := builderLogSeq.Add(1)
	code := fmt.Sprintf("%s_%s_%03d", now.Format("150405"), modeName(modeID), n%1000)
	dir := filepath.Join(builderLogRoot, now.Format("2006_01_02"), code)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		core.Log("agent.webpage loop-log mkdir failed:: dir::", dir, " err::", err)
		return nil
	}
	core.Log("agent.webpage loop-log started code::", code, " dir::", dir)
	return &turnLog{dir: dir, code: code, model: activeModel}
}

// meta writes the 00_meta.txt header describing the turn's inputs. Uses a fixed
// "00_" prefix so it always sorts first, without consuming the seq counter.
func (l *turnLog) meta(modeID int, userText, pageContext string) {
	if l == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "loop code : %s\n", l.code)
	fmt.Fprintf(&b, "mode      : %d (%s)\n", modeID, modeName(modeID))
	fmt.Fprintf(&b, "model     : %s\n", l.model)
	fmt.Fprintf(&b, "started   : %s\n\n", l.code)
	fmt.Fprintf(&b, "----- USER PROMPT (%d bytes) -----\n%s\n\n", len(userText), userText)
	fmt.Fprintf(&b, "----- PAGE CONTEXT (%d bytes) -----\n%s\n", len(pageContext), pageContext)
	l.writeFixed("00_meta.txt", b.String())
}

// exchange logs one full LLM round-trip: the messages array sent to the model
// (the "context send") together with the model's reply. label names the call,
// e.g. "main_iter1" or "subagent_generate_svg".
func (l *turnLog) exchange(label string, req llm.ChatRequest, resp *llm.ChatResponse, err error) {
	if l == nil {
		return
	}
	model := req.Model
	if model == "" {
		model = l.model
	}
	var b strings.Builder
	fmt.Fprintf(&b, "=== EXCHANGE: %s ===\nmodel : %s\ntools : %s\n\n", label, model, toolNames(req.Tools))
	b.WriteString("----- REQUEST (context sent to the agent) -----\n\n")
	b.WriteString(formatMessages(req.Messages))
	b.WriteString("\n----- RESPONSE -----\n")
	switch {
	case err != nil:
		fmt.Fprintf(&b, "ERROR: %v\n", err)
	case resp != nil && len(resp.Choices) > 0:
		ch := resp.Choices[0]
		fmt.Fprintf(&b, "finish_reason: %s  tokens(in/out/total): %d/%d/%d\n\n",
			ch.FinishReason, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		b.WriteString(formatMessages([]llm.Message{ch.Message}))
	default:
		b.WriteString("(empty response)\n")
	}
	l.write(label, b.String())
}

// tool logs one dispatched tool call's args and the result fed back into the
// conversation. apply_sections is logged separately (it terminates the turn).
func (l *turnLog) tool(name, args, result string) {
	if l == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "=== TOOL RESULT: %s ===\n\nargs:\n%s\n\nresult:\n%s\n", name, args, result)
	l.write("tool_"+name, b.String())
}

// note writes an arbitrary labeled file (used for the final apply_sections
// payload and any other one-off record).
func (l *turnLog) note(label, content string) {
	if l == nil {
		return
	}
	l.write(label, content)
}

// write persists content under the next chronological prefix. A turn is driven
// by a single goroutine, so seq needs no locking.
func (l *turnLog) write(label, content string) {
	l.seq++
	l.writeFixed(fmt.Sprintf("%02d_%s.txt", l.seq, sanitizeLabel(label)), content)
}

func (l *turnLog) writeFixed(name, content string) {
	path := filepath.Join(l.dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		core.Log("agent.webpage loop-log write failed:: path::", path, " err::", err)
	}
}

// formatMessages renders an LLM messages slice as a readable transcript: one
// section per message, with tool-call expansion, so the file can be skimmed
// without a JSON viewer. Mirrors the chat loop's formatPromptMessages.
func formatMessages(messages []llm.Message) string {
	var b strings.Builder
	for i, m := range messages {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "--- [%d] role=%s", i, m.Role)
		if m.ToolCallID != "" {
			fmt.Fprintf(&b, " tool_call_id=%s", m.ToolCallID)
		}
		b.WriteString(" ---\n")
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

// modeName maps a builder mode id to a short folder-safe label.
func modeName(modeID int) string {
	switch modeID {
	case ModeBuildPage:
		return "build"
	case ModeEditSection:
		return "edit"
	default:
		return fmt.Sprintf("mode%d", modeID)
	}
}

// toolNames joins the registered tool names for the request header line.
func toolNames(tools []llm.Tool) string {
	if len(tools) == 0 {
		return "(none)"
	}
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Function.Name
	}
	return strings.Join(names, ", ")
}

// sanitizeLabel keeps filenames safe: lowercase alnum, others collapse to '_'.
func sanitizeLabel(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "entry"
	}
	return out
}
