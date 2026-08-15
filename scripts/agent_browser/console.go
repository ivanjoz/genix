package main

// Console, exception and failed-request collection for agent_browser.
//
// Why it matters: the agentic HTML shows what the page *rendered*. When a page comes back empty
// the HTML says nothing about why, and the agent's next move is a guess. The console says the
// component threw, or a request 404'd, which turns a guess into a diagnosis.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"app/agent_browser/browser"
)

// consoleRecord is one JSONL line of the log. Flat on purpose: the agent greps it.
type consoleRecord struct {
	Time   string `json:"Time"`
	Level  string `json:"Level"`
	Source string `json:"Source"`
	Text   string `json:"Text"`
	URL    string `json:"URL,omitempty"`
	Line   int    `json:"Line,omitempty"`
}

// severityRank orders levels so `-level warning` can mean "warning and worse".
var severityRank = map[string]int{"debug": 0, "verbose": 0, "log": 1, "info": 2, "warning": 3, "error": 4}

// collectConsoleEvents drains the CDP event stream into the log file until the session closes.
func collectConsoleEvents(events <-chan browser.Event, out io.Writer) {
	encoder := json.NewEncoder(out)
	for event := range events {
		record, ok := decodeConsoleEvent(event)
		if !ok {
			continue
		}
		record.Time = time.Now().Format(time.RFC3339)
		_ = encoder.Encode(record)
		// Mirrored to stdout so a developer watching the resident `start` sees errors live.
		fmt.Printf("[%s] %s %s\n", record.Level, record.Source, record.Text)
	}
}

// decodeConsoleEvent maps the three CDP event shapes we listen to onto one record. Anything else
// (frame navigations, execution-context lifecycle) is dropped.
func decodeConsoleEvent(event browser.Event) (consoleRecord, bool) {
	switch event.Method {
	case "Runtime.consoleAPICalled":
		var params struct {
			Type string `json:"type"`
			Args []struct {
				Type        string          `json:"type"`
				Value       json.RawMessage `json:"value"`
				Description string          `json:"description"`
			} `json:"args"`
			StackTrace struct {
				CallFrames []struct {
					URL        string `json:"url"`
					LineNumber int    `json:"lineNumber"`
				} `json:"callFrames"`
			} `json:"stackTrace"`
		}
		if json.Unmarshal(event.Params, &params) != nil {
			return consoleRecord{}, false
		}

		parts := []string{}
		for _, arg := range params.Args {
			// A primitive arrives as `value`; an object only as `description`, since CDP does not
			// serialize object graphs unless asked to.
			if len(arg.Value) > 0 {
				parts = append(parts, strings.Trim(string(arg.Value), `"`))
			} else if arg.Description != "" {
				parts = append(parts, arg.Description)
			} else {
				parts = append(parts, arg.Type)
			}
		}

		record := consoleRecord{Level: normalizeLevel(params.Type), Source: "console", Text: strings.Join(parts, " ")}
		if len(params.StackTrace.CallFrames) > 0 {
			record.URL = params.StackTrace.CallFrames[0].URL
			record.Line = params.StackTrace.CallFrames[0].LineNumber
		}
		return record, true

	case "Runtime.exceptionThrown":
		var params struct {
			ExceptionDetails struct {
				Text       string `json:"text"`
				URL        string `json:"url"`
				LineNumber int    `json:"lineNumber"`
				Exception  struct {
					Description string `json:"description"`
				} `json:"exception"`
			} `json:"exceptionDetails"`
		}
		if json.Unmarshal(event.Params, &params) != nil {
			return consoleRecord{}, false
		}
		details := params.ExceptionDetails
		// The description carries the stack; the text is just "Uncaught".
		text := details.Exception.Description
		if text == "" {
			text = details.Text
		}
		return consoleRecord{Level: "error", Source: "exception", Text: text, URL: details.URL, Line: details.LineNumber}, true

	case "Log.entryAdded":
		var params struct {
			Entry struct {
				Level      string `json:"level"`
				Source     string `json:"source"`
				Text       string `json:"text"`
				URL        string `json:"url"`
				LineNumber int    `json:"lineNumber"`
			} `json:"entry"`
		}
		if json.Unmarshal(event.Params, &params) != nil {
			return consoleRecord{}, false
		}
		entry := params.Entry
		return consoleRecord{Level: normalizeLevel(entry.Level), Source: entry.Source, Text: entry.Text, URL: entry.URL, Line: entry.LineNumber}, true
	}
	return consoleRecord{}, false
}

func normalizeLevel(level string) string {
	switch level {
	case "error", "assert":
		return "error"
	case "warning", "warn":
		return "warning"
	case "debug", "verbose", "trace":
		return "debug"
	case "info":
		return "info"
	}
	return "log"
}

func agentBrowserConsole(args []string) error {
	flags := newFlagSet("console")
	lineCount := flags.Int("n", 50, "cuántos registros mostrar")
	minLevel := flags.String("level", "log", "nivel mínimo: debug|log|info|warning|error")
	follow := flags.Bool("follow", false, "seguir el archivo en vivo")
	if err := flags.Parse(args); err != nil {
		return err
	}

	minRank, known := severityRank[*minLevel]
	if !known {
		return fmt.Errorf("nivel desconocido %q; usa debug|log|info|warning|error", *minLevel)
	}

	file, err := os.Open(consoleLogPath)
	if err != nil {
		return fmt.Errorf("no hay log de consola en %s; ¿está corriendo `agent_browser start`?", consoleLogPath)
	}
	defer file.Close()

	matching := []consoleRecord{}
	scanner := bufio.NewScanner(file)
	// Console lines carry stack traces, well past bufio's 64KB default.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		record := consoleRecord{}
		if json.Unmarshal(scanner.Bytes(), &record) != nil {
			continue
		}
		if severityRank[record.Level] >= minRank {
			matching = append(matching, record)
		}
	}

	if len(matching) > *lineCount {
		matching = matching[len(matching)-*lineCount:]
	}
	for _, record := range matching {
		printConsoleRecord(record)
	}
	if len(matching) == 0 {
		fmt.Printf("(sin registros de nivel %s o superior en %s)\n", *minLevel, consoleLogPath)
	}

	if !*follow {
		return nil
	}
	// Follow from the current end of file, appending as the resident collector writes.
	for {
		time.Sleep(500 * time.Millisecond)
		for scanner.Scan() {
			record := consoleRecord{}
			if json.Unmarshal(scanner.Bytes(), &record) != nil {
				continue
			}
			if severityRank[record.Level] >= minRank {
				printConsoleRecord(record)
			}
		}
		// A finished scanner stays finished; re-arm it against the same handle to pick up appends.
		scanner = bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	}
}

func printConsoleRecord(record consoleRecord) {
	location := ""
	if record.URL != "" {
		location = fmt.Sprintf("  (%s:%d)", record.URL, record.Line)
	}
	fmt.Printf("%s  %-7s %-10s %s%s\n", record.Time, record.Level, record.Source, record.Text, location)
}
