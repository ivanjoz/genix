package main

// The page-facing half of agent_browser: read the page, act on it, and capture what it really
// looks like.
//
// Reading and acting go through the backend's POST /agent — the same endpoint the LLM agent uses,
// so this CLI never becomes a second, divergent way to drive the app. Only the screenshot goes
// straight to Chrome, because a true pixel render is precisely what the app cannot produce about
// itself.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"app/agent_browser/browser"
)

// Mirrors backend/agent/protocol.go — only the fields this CLI prints.
type agentComponent struct {
	ID      int
	Type    string
	Label   string
	Methods []string
}

type agentPage struct {
	Components []agentComponent
	HTML       string
}

type agentResponse struct {
	Results []json.RawMessage
	Page    agentPage
}

type agentAction struct {
	ID                string `json:",omitempty"`
	Method            string
	Args              []any `json:",omitempty"`
	ReturnPageContent bool  `json:",omitempty"`
}

func newFlagSet(name string) *flag.FlagSet { return flag.NewFlagSet(name, flag.ExitOnError) }

// readAllLimited reads an error body without letting a misbehaving response fill memory.
func readAllLimited(body io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(body, 8*1024))
	return string(raw), err
}

// postAgentActions runs a batch through the backend and returns the post-action page snapshot.
// An empty batch is the documented "what does the page look like now?" call.
func postAgentActions(ctx context.Context, backendURL string, actions []agentAction) (agentResponse, error) {
	result := agentResponse{}
	body, err := json.Marshal(map[string]any{"Actions": actions})
	if err != nil {
		return result, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, backendURL+"/agent", bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return result, fmt.Errorf("no se pudo contactar al backend en %s: %w", backendURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		detail, _ := readAllLimited(response.Body)
		return result, fmt.Errorf("el backend respondió %d: %s", response.StatusCode, strings.TrimSpace(detail))
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("respuesta ilegible del backend: %w", err)
	}
	return result, nil
}

// printComponents renders the registry the agent navigates by: one line per handle, with the
// methods it accepts. This is the map; the screenshot is the territory.
func printComponents(page agentPage) {
	fmt.Printf("\n%d componentes registrados:\n", len(page.Components))
	for _, component := range page.Components {
		label := component.Label
		if label == "" {
			label = "—"
		}
		fmt.Printf("  %-6d %-16s %-32s %s\n", component.ID, component.Type, label, strings.Join(component.Methods, ","))
	}
}

func agentBrowserGoto(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: agent_browser goto <ruta>")
	}
	state, err := readBrowserState()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// ReturnPageContent makes the browser wait for the route change and the DOM to settle before
	// answering, so the snapshot below belongs to the new page and not the old one.
	response, err := postAgentActions(ctx, state.BackendURL, []agentAction{
		{Method: "navigate", Args: []any{args[0]}, ReturnPageContent: true},
	})
	if err != nil {
		return err
	}
	fmt.Printf("navegado a %s\n", args[0])
	printComponents(response.Page)
	return nil
}

func agentBrowserHTML(args []string) error {
	flags := newFlagSet("html")
	outPath := flags.String("out", pageHTMLPath, "dónde escribir el HTML agéntico")
	if err := flags.Parse(args); err != nil {
		return err
	}
	state, err := readBrowserState()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := postAgentActions(ctx, state.BackendURL, []agentAction{})
	if err != nil {
		return err
	}
	if err := os.WriteFile(*outPath, []byte(response.Page.HTML), 0o600); err != nil {
		return err
	}
	fmt.Printf("html       %s (%d bytes)\n", *outPath, len(response.Page.HTML))
	printComponents(response.Page)
	return nil
}

func agentBrowserAct(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf(`uso: agent_browser act '[{"ID":"60","Method":"setValue","Args":["Hola"]}]'`)
	}
	state, err := readBrowserState()
	if err != nil {
		return err
	}

	actions := []agentAction{}
	if err := json.Unmarshal([]byte(args[0]), &actions); err != nil {
		return fmt.Errorf("el argumento debe ser un array JSON de acciones: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	response, err := postAgentActions(ctx, state.BackendURL, actions)
	if err != nil {
		return err
	}

	// Results is shorter than Actions when one fails: the batch stops there.
	fmt.Printf("%d de %d acciones ejecutadas\n", len(response.Results), len(actions))
	for index, raw := range response.Results {
		fmt.Printf("  [%d] %s\n", index, summarizeActionResult(raw))
	}
	if len(response.Results) < len(actions) {
		fmt.Printf("  la acción %d falló y detuvo el lote\n", len(response.Results)-1)
	}
	printComponents(response.Page)
	return nil
}

// summarizeActionResult keeps a result line readable. An action with ReturnPageContent embeds a
// whole page snapshot in its Value — tens of KB of HTML that would drown the caller's context for
// nothing, since printComponents already renders the refreshed registry right below.
func summarizeActionResult(raw json.RawMessage) string {
	var result struct {
		OK    bool
		Error string
		Value json.RawMessage
	}
	if json.Unmarshal(raw, &result) != nil {
		return string(raw)
	}
	if !result.OK {
		return "ERROR: " + result.Error
	}

	value := strings.TrimSpace(string(result.Value))
	switch {
	case value == "" || value == "null":
		return "ok"
	case len(value) > 240:
		return fmt.Sprintf("ok (%d bytes de respuesta) %s…", len(value), value[:240])
	}
	return "ok " + value
}

// agentBrowserShot captures what the browser actually painted, straight from Chrome.
func agentBrowserShot(args []string) error {
	flags := newFlagSet("shot")
	outPath := flags.String("out", shotPath, "dónde escribir el PNG")
	fullPage := flags.Bool("full", false, "capturar la página completa, no sólo el viewport")
	if err := flags.Parse(args); err != nil {
		return err
	}
	state, err := readBrowserState()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, _, err := browser.ConnectToPage(ctx, state.DebugPort)
	if err != nil {
		return err
	}
	defer client.Close()

	path, width, height, err := captureBrowserScreenshot(ctx, client, *outPath, *fullPage)
	if err != nil {
		return err
	}
	fmt.Printf("captura    %s (%dx%d)\n", path, width, height)
	return nil
}

// captureBrowserScreenshot writes Chrome's own render to disk and reports the captured size.
func captureBrowserScreenshot(ctx context.Context, client *browser.Client, outPath string, fullPage bool) (string, int, int, error) {
	params := map[string]any{"format": "png"}
	width, height := 0, 0

	if fullPage {
		// A full-page shot needs the content box, which is taller than the viewport; without the
		// explicit clip Chrome captures only what is on screen.
		var metrics struct {
			CSSContentSize struct {
				Width  float64 `json:"width"`
				Height float64 `json:"height"`
			} `json:"cssContentSize"`
		}
		if err := client.CallInto(ctx, "Page.getLayoutMetrics", nil, &metrics); err != nil {
			return "", 0, 0, err
		}
		width = int(metrics.CSSContentSize.Width)
		height = int(metrics.CSSContentSize.Height)
		params["captureBeyondViewport"] = true
		params["clip"] = map[string]any{"x": 0, "y": 0, "width": metrics.CSSContentSize.Width, "height": metrics.CSSContentSize.Height, "scale": 1}
	}

	var shot struct {
		Data string `json:"data"`
	}
	if err := client.CallInto(ctx, "Page.captureScreenshot", params, &shot); err != nil {
		return "", 0, 0, err
	}
	raw, err := base64.StdEncoding.DecodeString(shot.Data)
	if err != nil {
		return "", 0, 0, fmt.Errorf("la captura no es base64 válido: %w", err)
	}
	if err := os.WriteFile(outPath, raw, 0o600); err != nil {
		return "", 0, 0, err
	}

	if !fullPage {
		// Viewport shots have no clip to report, so ask the page for its own size.
		var viewport struct {
			Result struct {
				Value struct {
					Width  int `json:"width"`
					Height int `json:"height"`
				} `json:"value"`
			} `json:"result"`
		}
		_ = client.CallInto(ctx, "Runtime.evaluate", map[string]any{
			"expression":    "({width: window.innerWidth, height: window.innerHeight})",
			"returnByValue": true,
		}, &viewport)
		width, height = viewport.Result.Value.Width, viewport.Result.Value.Height
	}
	return outPath, width, height, nil
}

// agentBrowserCompare produces the three views of the same moment side by side. This is the point
// of the whole command: the agent navigates by the HTML, but only the screenshot tells it whether
// the page it thinks it is on is the page a user would see.
//
// The two screenshots differ on purpose. Chrome's is ground truth. The in-page one is what
// frontend/core/agent/screenshot.ts produces with modern-screenshot, which is what the *product's*
// agent gets — when the two diverge, that renderer is dropping something (tainted canvas, a
// stripped @font-face), and that is a bug in the app, not in the navigation.
func agentBrowserCompare(args []string) error {
	flags := newFlagSet("compare")
	fullPage := flags.Bool("full", false, "capturar la página completa, no sólo el viewport")
	if err := flags.Parse(args); err != nil {
		return err
	}
	state, err := readBrowserState()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client, target, err := browser.ConnectToPage(ctx, state.DebugPort)
	if err != nil {
		return err
	}
	defer client.Close()

	fmt.Printf("url        %s\n", target.URL)

	shotFile, width, height, err := captureBrowserScreenshot(ctx, client, shotPath, *fullPage)
	if err != nil {
		return err
	}
	fmt.Printf("navegador  %s (%dx%d)  ← lo que el usuario vería\n", shotFile, width, height)

	// The in-page render is fetched through the backend so it goes down the same path the product
	// agent uses; the handler writes the PNG and answers with its location.
	if inPagePath, err := fetchInPageScreenshot(ctx, state.BackendURL); err != nil {
		fmt.Printf("in-page    falló: %v\n", err)
	} else {
		fmt.Printf("in-page    %s  ← lo que modern-screenshot produce\n", inPagePath)
	}

	response, err := postAgentActions(ctx, state.BackendURL, []agentAction{})
	if err != nil {
		return err
	}
	if err := os.WriteFile(pageHTMLPath, []byte(response.Page.HTML), 0o600); err != nil {
		return err
	}
	fmt.Printf("html       %s (%d bytes)  ← por lo que el agente navega\n", pageHTMLPath, len(response.Page.HTML))
	printComponents(response.Page)
	return nil
}

// fetchInPageScreenshot triggers the app's own DOM renderer through GET /agent?get=screenshot and
// returns the path the backend wrote.
func fetchInPageScreenshot(ctx context.Context, backendURL string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, backendURL+"/agent?get=screenshot", nil)
	if err != nil {
		return "", err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		detail, _ := readAllLimited(response.Body)
		return "", fmt.Errorf("estado %d: %s", response.StatusCode, strings.TrimSpace(detail))
	}
	var result struct {
		Path   string
		Width  int
		Height int
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s (%dx%d)", result.Path, result.Width, result.Height), nil
}
