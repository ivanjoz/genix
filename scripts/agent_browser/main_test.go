package main

import (
	"encoding/json"
	"strings"
	"testing"

	"app/agent_browser/browser"
)

// The three CDP event shapes must all collapse onto one record, because the console log is the
// agent's only window into why a page failed — an event shape that silently decodes to nothing
// would read as "the page had no errors".
func TestDecodeConsoleEventCoversEveryShape(t *testing.T) {
	checks := []struct {
		name       string
		event      browser.Event
		wantLevel  string
		wantSource string
		wantText   string
	}{
		{
			name: "console.error with primitive args",
			event: browser.Event{Method: "Runtime.consoleAPICalled", Params: json.RawMessage(
				`{"type":"error","args":[{"type":"string","value":"boom"},{"type":"number","value":42}],
				  "stackTrace":{"callFrames":[{"url":"http://localhost:3570/app.ts","lineNumber":17}]}}`)},
			wantLevel: "error", wantSource: "console", wantText: "boom 42",
		},
		{
			name: "console.log with an object arg, which CDP only describes",
			event: browser.Event{Method: "Runtime.consoleAPICalled", Params: json.RawMessage(
				`{"type":"log","args":[{"type":"object","description":"Object"}]}`)},
			wantLevel: "log", wantSource: "console", wantText: "Object",
		},
		{
			name: "uncaught exception prefers the description, which carries the stack",
			event: browser.Event{Method: "Runtime.exceptionThrown", Params: json.RawMessage(
				`{"exceptionDetails":{"text":"Uncaught","url":"http://localhost:3570/x.ts","lineNumber":3,
				  "exception":{"description":"TypeError: undefined is not a function"}}}`)},
			wantLevel: "error", wantSource: "exception", wantText: "TypeError: undefined is not a function",
		},
		{
			name: "failed request, which only Log.entryAdded reports",
			event: browser.Event{Method: "Log.entryAdded", Params: json.RawMessage(
				`{"entry":{"level":"error","source":"network","text":"Failed to load resource: 404","url":"http://localhost:3570/missing.svg"}}`)},
			wantLevel: "error", wantSource: "network", wantText: "Failed to load resource: 404",
		},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			record, ok := decodeConsoleEvent(check.event)
			if !ok {
				t.Fatalf("el evento no se decodificó")
			}
			if record.Level != check.wantLevel {
				t.Errorf("Level = %q, se esperaba %q", record.Level, check.wantLevel)
			}
			if record.Source != check.wantSource {
				t.Errorf("Source = %q, se esperaba %q", record.Source, check.wantSource)
			}
			if record.Text != check.wantText {
				t.Errorf("Text = %q, se esperaba %q", record.Text, check.wantText)
			}
		})
	}
}

func TestDecodeConsoleEventIgnoresUnrelatedEvents(t *testing.T) {
	event := browser.Event{Method: "Page.frameNavigated", Params: json.RawMessage(`{"frame":{}}`)}
	if _, ok := decodeConsoleEvent(event); ok {
		t.Fatal("Page.frameNavigated no debería producir un registro de consola")
	}
}

// `-level warning` has to mean "warning and worse", so a filter never hides the errors it was
// opened to find.
func TestSeverityRankOrdersLevels(t *testing.T) {
	if severityRank["error"] <= severityRank["warning"] {
		t.Error("error debe ordenar por encima de warning")
	}
	if severityRank["warning"] <= severityRank["log"] {
		t.Error("warning debe ordenar por encima de log")
	}
	if severityRank["debug"] >= severityRank["log"] {
		t.Error("debug debe ordenar por debajo de log")
	}
}

// An action carrying ReturnPageContent answers with a whole page snapshot; printing it raw would
// bury the caller in HTML that printComponents already shows in readable form.
func TestSummarizeActionResultTruncatesPageSnapshots(t *testing.T) {
	snapshot := `{"OK":true,"Value":{"page":{"HTML":"` + strings.Repeat("x", 5000) + `"}}}`
	summary := summarizeActionResult(json.RawMessage(snapshot))

	if len(summary) > 400 {
		t.Errorf("el resumen sigue siendo enorme (%d bytes)", len(summary))
	}
	if !strings.HasPrefix(summary, "ok (") {
		t.Errorf("se esperaba un resumen con el tamaño, se obtuvo %q", summary)
	}
}

func TestSummarizeActionResultReportsFailures(t *testing.T) {
	summary := summarizeActionResult(json.RawMessage(`{"OK":false,"Error":"no method setValue on handle 60"}`))
	if summary != "ERROR: no method setValue on handle 60" {
		t.Errorf("resumen = %q", summary)
	}
}

func TestSummarizeActionResultKeepsVoidResultsShort(t *testing.T) {
	if summary := summarizeActionResult(json.RawMessage(`{"OK":true,"Value":null}`)); summary != "ok" {
		t.Errorf("resumen = %q, se esperaba \"ok\"", summary)
	}
}

// The mint endpoint and the CLI are useless against the wrong port, and this machine's backend
// listens on 14010 — the documented 3589 is only the fallback.
func TestDefaultBackendURLReadsConfiguredPort(t *testing.T) {
	url := defaultBackendURL()
	if !strings.HasPrefix(url, "http://localhost:") {
		t.Fatalf("URL con formato inesperado: %q", url)
	}
}
