package main

// The chat half of agent_browser: send a turn, answer a choice card and read the conversation the
// way a user does. It calls the page's own chat (window.__agentChat in
// frontend/core/agent/agent-chat.svelte.ts: the same sendAgentTurn/answerAgentPause the widget calls),
// so the turn runs through the real path — POST p-agent-turn, the SSE reply, the rendered answer —
// and `shot` shows what the user would see.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"app/agent_browser/browser"
)

// chatRow mirrors AgentChatRow (frontend/core/agent/chat_history.idb.ts): only what this CLI prints.
type chatRow struct {
	Role    int    `json:"role"`
	Message string `json:"message"`
	Summary string `json:"summary"`
	Tables  []struct {
		Columns []chatColumn
	} `json:"tables"`
	Charts []struct {
		Columns []chatColumn
	} `json:"charts"`
	Pause *struct {
		Choices []struct{ ID, Label string }
	} `json:"pause"`
}

type chatColumn struct {
	Label  string
	Axis   string
	Values []any
}

// Roles of AgentChatRow (chat_history.idb.ts AGENT_ROLE_*).
var chatRoleNames = map[int]string{1: "user", 2: "agent", 3: "status"}

func agentBrowserChat(args []string) error {
	if len(args) == 0 {
		return errors.New("uso: agent_browser chat send|pick|rows …")
	}
	state, err := readBrowserState()
	if err != nil {
		return err
	}
	switch args[0] {
	case "send":
		flags := newFlagSet("chat send")
		modelID := flags.String("model", "", "id del modelo de config.toml [[models]] (vacío: el elegido en la app)")
		timeout := flags.Duration("timeout", 180*time.Second, "cuánto esperar la respuesta")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		text := strings.TrimSpace(strings.Join(flags.Args(), " "))
		if text == "" {
			return errors.New("uso: agent_browser chat send [-model ID] '<texto>'")
		}
		modelHash := ""
		if *modelID != "" {
			modelHash = modelIDHash(*modelID)
		}
		return runChatCall(state, *timeout, fmt.Sprintf("window.__agentChat.send(%s, %s)", jsString(text), jsString(modelHash)))
	case "pick":
		if len(args) < 2 {
			return errors.New("uso: agent_browser chat pick <ID|etiqueta>")
		}
		return runChatCall(state, 180*time.Second, fmt.Sprintf("window.__agentChat.pick(%s)", jsString(strings.Join(args[1:], " "))))
	case "rows":
		flags := newFlagSet("chat rows")
		count := flags.Int("n", 8, "cuántas filas")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		rows, err := readChatRows(state, 10*time.Second)
		if err != nil {
			return err
		}
		printChatRows(rows[max(0, len(rows)-*count):])
		return nil
	}
	return errors.New("uso: agent_browser chat send|pick|rows …")
}

// runChatCall runs one chat call in the page and prints the rows it added.
func runChatCall(state browserState, timeout time.Duration, call string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client, _, err := browser.ConnectToPage(ctx, state.DebugPort)
	if err != nil {
		return err
	}
	defer client.Close()
	before, err := evaluateInPage[[]chatRow](ctx, client, "window.__agentChat.rows()")
	if err != nil {
		return fmt.Errorf("la página no expone window.__agentChat (¿frontend sin recargar?): %w", err)
	}
	if _, err := evaluateInPage[any](ctx, client, call+".then(() => true)"); err != nil {
		return err
	}
	after, err := evaluateInPage[[]chatRow](ctx, client, "window.__agentChat.rows()")
	if err != nil {
		return err
	}
	printChatRows(after[min(len(before), len(after)):])
	return nil
}

func readChatRows(state browserState, timeout time.Duration) ([]chatRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client, _, err := browser.ConnectToPage(ctx, state.DebugPort)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	return evaluateInPage[[]chatRow](ctx, client, "window.__agentChat.rows()")
}

// printChatRows prints each row with what the user sees: tables and charts by their columns, and a
// choice card by its options (their IDs are what `chat pick` takes).
func printChatRows(rows []chatRow) {
	for _, row := range rows {
		fmt.Printf("[%s] %s\n", chatRoleNames[row.Role], row.Message)
		if row.Summary != "" {
			fmt.Printf("    summary: %s\n", row.Summary)
		}
		for _, table := range row.Tables {
			fmt.Printf("    tabla: %s\n", describeColumns(table.Columns))
		}
		for _, chart := range row.Charts {
			fmt.Printf("    gráfico: %s\n", describeColumns(chart.Columns))
		}
		if row.Pause != nil {
			options := []string{}
			for _, choice := range row.Pause.Choices {
				options = append(options, choice.ID+" = "+choice.Label)
			}
			fmt.Printf("    opciones: %s\n", strings.Join(options, " · "))
		}
	}
}

func describeColumns(columns []chatColumn) string {
	labels := []string{}
	rowCount := 0
	for _, column := range columns {
		label := column.Label
		if column.Axis != "" {
			label += " (" + column.Axis + ")"
		}
		labels = append(labels, label)
		rowCount = max(rowCount, len(column.Values))
	}
	// The first rows, so a later turn can name what this one showed.
	firstRows := []string{}
	for row := range min(rowCount, 3) {
		cells := []string{}
		for _, column := range columns {
			if row < len(column.Values) {
				cells = append(cells, fmt.Sprint(column.Values[row]))
			}
		}
		firstRows = append(firstRows, strings.Join(cells, " | "))
	}
	return fmt.Sprintf("%s — %d filas; %s", strings.Join(labels, " | "), rowCount, strings.Join(firstRows, "; "))
}

// evaluateInPage runs a JS expression in the page, awaiting a promise, and decodes its value.
func evaluateInPage[T any](ctx context.Context, client *browser.Client, expression string) (T, error) {
	var value T
	var response struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	err := client.CallInto(ctx, "Runtime.evaluate", map[string]any{
		"expression": expression, "awaitPromise": true, "returnByValue": true,
	}, &response)
	if err != nil {
		return value, err
	}
	if response.ExceptionDetails != nil {
		return value, fmt.Errorf("la página lanzó: %s", strings.TrimSpace(response.ExceptionDetails.Exception.Description+" "+response.ExceptionDetails.Text))
	}
	if len(response.Result.Value) > 0 {
		err = json.Unmarshal(response.Result.Value, &value)
	}
	return value, err
}

func jsString(text string) string {
	encoded, _ := json.Marshal(text)
	return string(encoded)
}

// modelIDHash is llm.ModelIDHash (agent/llm/models.go): the short key the app sends for a model id.
// The scripts module cannot import the agent's, so it is restated here; they must agree.
func modelIDHash(id string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(id))
	value := int32(hash.Sum32() & 0x7fffffff)
	if value == 0 {
		value = 1
	}
	return strconv.FormatInt(int64(value), 36)
}
