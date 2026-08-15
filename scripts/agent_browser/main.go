package main

// agent_browser — a headless Chrome that logs itself in, so an external AI agent can drive and
// *see* the running app with no human in the loop.
//
// The app already exposes everything an agent needs to act (POST /agent) and to read the page
// (its agentic HTML + component registry). What it could not do was exist without a human: every
// one of those endpoints needs a browser tab somebody opened and a session somebody logged into.
// This command supplies both, and adds the one view the agentic HTML cannot give — a real pixel
// screenshot to contrast against it.
//
// Lifecycle: `start` is resident. It launches Chrome, holds a CDP session open to collect console
// output, and stays alive until interrupted (run it in the background). Every other subcommand is
// short-lived and finds the running browser through the state file; Chrome accepts concurrent CDP
// clients, so they connect alongside the collector.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"app/agent_browser/browser"

	"github.com/pelletier/go-toml/v2"
)

// Everything lands in the OS temp dir, next to the screenshot the backend already writes there
// (backend/agent/http.go:writeScreenshot).
var (
	statePath      = filepath.Join(os.TempDir(), "genix-agent-browser.json")
	consoleLogPath = filepath.Join(os.TempDir(), "genix-agent-console.jsonl")
	profileDir     = filepath.Join(os.TempDir(), "genix-agent-profile")
	shotPath       = filepath.Join(os.TempDir(), "genix-agent-shot.png")
	pageHTMLPath   = filepath.Join(os.TempDir(), "genix-agent-page.html")
)

// browserState is what a short-lived subcommand needs to find the browser `start` launched.
type browserState struct {
	ChromePID  int
	DebugPort  int
	AppURL     string
	BackendURL string
	CompanyID  int
	UserID     int
	StartedAt  string
}

func readBrowserState() (browserState, error) {
	state := browserState{}
	raw, err := os.ReadFile(statePath)
	if err != nil {
		return state, fmt.Errorf("no hay un navegador corriendo (falta %s). Ejecuta primero: go run . agent_browser start", statePath)
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return state, fmt.Errorf("el archivo de estado %s está corrupto: %w", statePath, err)
	}
	return state, nil
}

func writeBrowserState(state browserState) error {
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, raw, 0o600)
}

// defaultBackendURL resolves where the local backend listens by reading the same config.toml the
// backend itself reads. Hardcoding a port would break on every machine whose config differs from
// the author's — the documented 3589 is only the fallback when `server.port` is 0 or absent.
func defaultBackendURL() string {
	const fallbackPort = 3589

	root, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("http://localhost:%d", fallbackPort)
	}
	// Walk up: the command is normally run from scripts/, the config lives at the repo root.
	for {
		configPath := filepath.Join(root, "config.toml")
		if raw, err := os.ReadFile(configPath); err == nil {
			var config struct {
				Server struct {
					Port int `toml:"port"`
				} `toml:"server"`
			}
			if toml.Unmarshal(raw, &config) == nil && config.Server.Port > 0 {
				return fmt.Sprintf("http://localhost:%d", config.Server.Port)
			}
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	return fmt.Sprintf("http://localhost:%d", fallbackPort)
}

// findChromeBinary picks the first installed Chrome/Chromium. Overridable with -chrome for a
// machine that keeps it somewhere unusual.
func findChromeBinary(override string) (string, error) {
	if override != "" {
		return exec.LookPath(override)
	}
	for _, candidate := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no se encontró chrome ni chromium en el PATH; pasa -chrome <ruta>")
}

// main dispatches every `agent_browser` subcommand.
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printAgentBrowserUsage()
		os.Exit(1)
	}

	var err error
	switch args[0] {
	case "start":
		err = agentBrowserStart(args[1:])
	case "stop":
		err = agentBrowserStop()
	case "status":
		err = agentBrowserStatus()
	case "goto":
		err = agentBrowserGoto(args[1:])
	case "shot":
		err = agentBrowserShot(args[1:])
	case "html":
		err = agentBrowserHTML(args[1:])
	case "act":
		err = agentBrowserAct(args[1:])
	case "compare":
		err = agentBrowserCompare(args[1:])
	case "console":
		err = agentBrowserConsole(args[1:])
	default:
		printAgentBrowserUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func printAgentBrowserUsage() {
	fmt.Println(`agent_browser — headless dev browser for the agentic API

  start [-company 1] [-user 1] [-route /] [-app URL] [-backend URL]
        [-width 1440] [-height 900] [-port 9222] [-headed] [-chrome PATH]
                            launch Chrome, log in, hold it open (run in background)
  stop                      kill the browser and drop its state
  status                    is Chrome alive, is the tab registered, where is it
  goto <route>              navigate inside the SPA
  shot [-out PATH] [-full]  real browser screenshot (PNG)
  html [-out PATH]          agentic HTML + component registry
  act '<json actions>'      run actions: [{"ID":"60","Method":"setValue","Args":["x"]}]
  compare                   screenshot + in-page render + agentic HTML, all at once
  console [-n 50] [-level error] [-follow]
                            page console, exceptions and failed requests`)
}

func agentBrowserStart(args []string) error {
	flags := flag.NewFlagSet("start", flag.ExitOnError)
	companyID := flags.Int("company", 1, "CompanyID de la sesión")
	userID := flags.Int("user", 1, "UserID de la sesión")
	route := flags.String("route", "/", "ruta inicial de la app")
	appURL := flags.String("app", "http://localhost:3570", "origen del frontend en desarrollo")
	backendURL := flags.String("backend", defaultBackendURL(), "origen del backend (por defecto, server.port de config.toml)")
	width := flags.Int("width", 1440, "ancho de la ventana")
	height := flags.Int("height", 900, "alto de la ventana")
	debugPort := flags.Int("port", 9222, "puerto de depuración de chrome")
	headed := flags.Bool("headed", false, "abrir con ventana visible (para depurar el propio launcher)")
	chromePath := flags.String("chrome", "", "ruta al binario de chrome")
	if err := flags.Parse(args); err != nil {
		return err
	}

	// A second Chrome on the same port would silently fail to bind, and a second tab would break
	// agent.ResolveTab, which only auto-resolves when exactly one tab is connected.
	if existing, err := readBrowserState(); err == nil {
		if processIsAlive(existing.ChromePID) {
			return fmt.Errorf("ya hay un navegador corriendo (pid %d). Ejecuta `agent_browser stop` primero", existing.ChromePID)
		}
	}

	chromeBinary, err := findChromeBinary(*chromePath)
	if err != nil {
		return err
	}

	// ?agent=1 makes the tab register itself as the external driver's tab (sse.ts:maybeAutoConnect);
	// ?devlogin mints the session before the layout renders (frontend/routes/+layout.ts).
	separator := "?"
	if strings.Contains(*route, "?") {
		separator = "&"
	}
	startURL := fmt.Sprintf("%s%s%sagent=1&devlogin=%d:%d", *appURL, *route, separator, *companyID, *userID)

	chromeArgs := []string{
		fmt.Sprintf("--remote-debugging-port=%d", *debugPort),
		"--user-data-dir=" + profileDir,
		fmt.Sprintf("--window-size=%d,%d", *width, *height),
		"--no-first-run",
		"--no-default-browser-check",
		// /dev/shm is small on many Linux setups and Chrome crashes when it fills.
		"--disable-dev-shm-usage",
		// The tab must keep its SSE stream and timers running while nothing is looking at it.
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding",
	}
	if !*headed {
		chromeArgs = append(chromeArgs, "--headless=new")
	}
	chromeArgs = append(chromeArgs, startURL)

	chrome := exec.Command(chromeBinary, chromeArgs...)
	// Own process group: killing the launcher must not leave a half-dead Chrome behind, and
	// `stop` can reap the whole group by pid.
	chrome.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := chrome.Start(); err != nil {
		return fmt.Errorf("no se pudo lanzar chrome: %w", err)
	}
	fmt.Printf("chrome     pid %d, puerto de depuración %d\n", chrome.Process.Pid, *debugPort)
	fmt.Printf("url        %s\n", startURL)

	state := browserState{
		ChromePID:  chrome.Process.Pid,
		DebugPort:  *debugPort,
		AppURL:     *appURL,
		BackendURL: *backendURL,
		CompanyID:  *companyID,
		UserID:     *userID,
		StartedAt:  time.Now().Format(time.RFC3339),
	}
	if err := writeBrowserState(state); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// From here on Chrome is running, so every failure path has to take it down with us.
	shutdown := func() {
		killProcessGroup(state.ChromePID)
		_ = os.Remove(statePath)
	}

	if err := browser.WaitForDebugPort(ctx, *debugPort, 20*time.Second); err != nil {
		shutdown()
		return err
	}
	client, target, err := browser.ConnectToPage(ctx, *debugPort)
	if err != nil {
		shutdown()
		return err
	}
	defer client.Close()

	events := client.Subscribe()
	// Runtime carries console calls and uncaught exceptions; Log adds what the page itself never
	// sees — failed requests, CSP violations, deprecation warnings.
	for _, domain := range []string{"Runtime.enable", "Log.enable", "Page.enable"} {
		if _, err := client.Call(ctx, domain, nil); err != nil {
			shutdown()
			return fmt.Errorf("no se pudo habilitar %s: %w", domain, err)
		}
	}
	fmt.Printf("target     %s\n", target.ID)

	consoleLog, err := os.Create(consoleLogPath)
	if err != nil {
		shutdown()
		return err
	}
	defer consoleLog.Close()

	// Collect from now on, in the background: the login round-trip below takes seconds and the
	// page's boot errors are exactly what we must not miss.
	go collectConsoleEvents(events, consoleLog)

	if err := waitForRegisteredTab(ctx, *backendURL, 40*time.Second); err != nil {
		fmt.Fprintln(os.Stderr, "aviso:", err)
		fmt.Fprintf(os.Stderr, "revisa %s para ver por qué la página no llegó a registrarse\n", consoleLogPath)
	} else {
		fmt.Printf("tab        registrada en el backend — el agente ya puede usar POST %s/agent\n", *backendURL)
	}
	fmt.Printf("console    %s\n", consoleLogPath)
	fmt.Println("listo. Ctrl-C (o `agent_browser stop`) para cerrar.")

	<-ctx.Done()
	fmt.Println("\ncerrando chrome…")
	shutdown()
	return nil
}

// waitForRegisteredTab blocks until the backend can reach the tab. GET /agent?get=menu answers 503
// while no browser holds an SSE stream, which makes it the readiness probe for the whole chain:
// page loaded → session minted → stream opened → tab registered.
func waitForRegisteredTab(ctx context.Context, backendURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	lastStatus := 0
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, backendURL+"/agent?get=menu", nil)
		if err != nil {
			return err
		}
		response, err := http.DefaultClient.Do(request)
		if err == nil {
			lastStatus = response.StatusCode
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("la pestaña no se registró en %s (último estado del backend: %d)", timeout, lastStatus)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

func agentBrowserStop() error {
	state, err := readBrowserState()
	if err != nil {
		return err
	}
	killProcessGroup(state.ChromePID)
	_ = os.Remove(statePath)
	fmt.Printf("navegador detenido (pid %d)\n", state.ChromePID)
	return nil
}

func agentBrowserStatus() error {
	state, err := readBrowserState()
	if err != nil {
		return err
	}
	fmt.Printf("chrome     pid %d — %s\n", state.ChromePID, aliveLabel(processIsAlive(state.ChromePID)))
	fmt.Printf("sesión     company %d, user %d (desde %s)\n", state.CompanyID, state.UserID, state.StartedAt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if target, err := browser.FirstPageTarget(ctx, state.DebugPort); err == nil {
		fmt.Printf("url        %s\n", target.URL)
	} else {
		fmt.Printf("url        no disponible: %v\n", err)
	}

	if err := waitForRegisteredTab(ctx, state.BackendURL, time.Second); err != nil {
		fmt.Printf("tab        NO registrada en el backend: %v\n", err)
	} else {
		fmt.Println("tab        registrada en el backend")
	}
	return nil
}

func aliveLabel(alive bool) string {
	if alive {
		return "vivo"
	}
	return "muerto"
}

// processIsAlive probes the pid with signal 0, which checks existence without touching it.
func processIsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

// killProcessGroup takes down Chrome and every renderer it forked. The negative pid targets the
// group `start` created with Setpgid, so no zombie renderer keeps the debug port bound.
func killProcessGroup(pid int) {
	if pid <= 0 {
		return
	}
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
}
