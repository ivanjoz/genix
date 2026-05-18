// WebSocket bridge: backend -> browser command channel.
// Connects to the local Go backend so it can drive the page as an agent. All
// command execution lives in commands.ts; this file just owns the socket
// lifecycle and the message-to-reply plumbing.

import { Env } from "$core/env";
import { Agent, isAgentEnabled } from "./registry";
import { releaseScreenStream, runCommand, type WsMessage } from "./commands";

const AGENT_LOG_KEY = "__agent_debug_log";
const AGENT_LOG_LIMIT = 120;

const agentLog = (level: "info" | "warn", message: string, detail?: unknown) => {
  const entry = {
    at: new Date().toISOString(),
    level,
    message,
    detail,
  };
  // Keep a small browser-side ring buffer because production console output is
  // often lost after reloads; __agent.debugLog exposes the same trail.
  try {
    const previous = JSON.parse(localStorage.getItem(AGENT_LOG_KEY) || "[]");
    const next = Array.isArray(previous) ? [...previous, entry].slice(-AGENT_LOG_LIMIT) : [entry];
    localStorage.setItem(AGENT_LOG_KEY, JSON.stringify(next));
  } catch {
    // Logging must never break the agent bridge.
  }
  const logger = level === "warn" ? console.warn : console.info;
  logger(`[Agent] ${message}`, detail || "");
};

// agentWsBase derives the WS scheme + host from the selected API endpoint.
// Env.API_ROUTES.MAIN ends with `/api/` (it's the HTTP API root) — the
// `/ws/agent` and `/ws/agent-chat` handlers are mounted at the server root,
// not under `/api`, so we strip that suffix before swapping http→ws.
export const agentWsBase = (): string => {
  const main = Env.API_ROUTES.MAIN || "";
  // Drop trailing slash, then the optional `/api` segment.
  const root = main.replace(/\/+$/, "").replace(/\/api$/, "");
  return root.replace(/^http/, "ws");
};

// getAgentTabID returns the per-tab id sent on the WS upgrade so the backend
// can route page commands to the correct tab. Stored in sessionStorage so it
// survives navigations within the tab but is unique per tab (sessionStorage
// is naturally tab-scoped). The chat widget (in step 4+) reads the same id so
// chat and page connections share it.
export const getAgentTabID = (): string => {
  if (typeof window === "undefined") { return ""; }
  const key = "__agent_tab_id";
  let id = sessionStorage.getItem(key);
  if (!id) {
    id = (window.crypto?.randomUUID?.() as string | undefined)
      || `tab-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
    sessionStorage.setItem(key, id);
  }
  return id;
};

// Reads the cached user id from localStorage so the backend can stamp it on
// the page connection. Best-effort: missing/invalid → 0 (backend treats this
// as anonymous).
const getAgentUserID = (): number => {
  if (typeof window === "undefined") { return 0; }
  try {
    const raw = localStorage.getItem(Env.appId + "UserInfo");
    if (!raw) { return 0; }
    const info = JSON.parse(raw) as { ID?: number };
    return Number(info?.ID) || 0;
  } catch {
    return 0;
  }
};

interface BridgeState {
  socket: WebSocket | null;
  reconnectDelayMs: number;
  started: boolean;
}

const bridgeState: BridgeState = {
  socket: null,
  reconnectDelayMs: 1000,
  started: false,
};

const sendReply = (socket: WebSocket, id: number, type: "result" | "error", payload: unknown) => {
  agentLog(type === "error" ? "warn" : "info", "sending reply", { id, type, payloadBytes: JSON.stringify(payload ?? null).length });
  socket.send(JSON.stringify({ ID: id, Type: type, Payload: payload }));
};

const handleMessage = async (socket: WebSocket, raw: unknown) => {
  let message: WsMessage;
  try {
    message = JSON.parse(typeof raw === "string" ? raw : "");
  } catch (parseError) {
    agentLog("warn", "bad message json", { error: String(parseError), rawType: typeof raw });
    return;
  }
  agentLog("info", "command received", { id: message.ID, type: message.Type, payloadBytes: JSON.stringify(message.Payload ?? null).length });
  try {
    const result = await runCommand(message.Type, message.Payload);
    sendReply(socket, message.ID, "result", result);
  } catch (commandError: any) {
    const errorMessage = String(commandError?.message || commandError);
    agentLog("warn", "command error", { id: message.ID, type: message.Type, error: errorMessage });
    sendReply(socket, message.ID, "error", { Message: errorMessage });
  }
};

const scheduleReconnect = () => {
  const delay = Math.min(bridgeState.reconnectDelayMs, 10_000);
  bridgeState.reconnectDelayMs = Math.min(delay * 2, 10_000);
  agentLog("info", "retry scheduled", { delayMs: delay });
  setTimeout(connectAgentSocket, delay);
};

const connectAgentSocket = () => {
  if (typeof window === "undefined") { return; }
  if (bridgeState.socket && bridgeState.socket.readyState <= WebSocket.OPEN) { return; }

  // Backend address comes from the selected API endpoint so dev/prod/VPS
  // targets all work without hardcoding a port. The backend routes commands
  // by TabID — include it plus company/user for bookkeeping on the upgrade
  // URL.
  const tab = getAgentTabID();
  const company = Env.getCompanyID() || 0;
  const user = getAgentUserID();
  const url = `${agentWsBase()}/ws/agent?tab=${encodeURIComponent(tab)}&company=${company}&user=${user}`;
  agentLog("info", "connecting page bridge", { url, tab, company, user, apiMain: Env.API_ROUTES.MAIN, local: globalThis._isLocal, enabled: isAgentEnabled() });
  let socket: WebSocket;
  try {
    socket = new WebSocket(url);
  } catch (error) {
    agentLog("warn", "socket constructor failed", { error: String(error), url });
    scheduleReconnect();
    return;
  }
  bridgeState.socket = socket;

  socket.addEventListener("open", () => {
    agentLog("info", "page bridge connected", { url, tab });
    bridgeState.reconnectDelayMs = 1000;
    socket.send(JSON.stringify({
      Type: "ready",
      Payload: {
        URL: window.location.href,
        Path: window.location.pathname,
        AgentEnabled: isAgentEnabled(),
        Handles: Agent.describe().length,
        UserAgent: navigator.userAgent,
      },
    }));
  });

  socket.addEventListener("message", (event) => {
    void handleMessage(socket, event.data);
  });

  socket.addEventListener("close", () => {
    if (bridgeState.socket !== socket) { return; }
    agentLog("warn", "page bridge closed", { url, tab });
    bridgeState.socket = null;
    releaseScreenStream();
    scheduleReconnect();
  });

  socket.addEventListener("error", (error) => {
    agentLog("warn", "page bridge socket error", { url, error: String(error) });
    try { socket.close(); } catch { /* ignore */ }
  });
};

export const startAgentBridge = () => {
  if (bridgeState.started) {
    agentLog("info", "start skipped: already started");
    return;
  }
  if (typeof window === "undefined") { return; }
  if (!isAgentEnabled()) {
    // Production needs the page-driving bridge for the in-app chat agent; log
    // the disabled condition but still connect so backend diagnostics can tell
    // whether the route/proxy/websocket path is alive.
    agentLog("warn", "agent registry disabled; starting bridge for diagnostics", { local: globalThis._isLocal, enableFlag: window.ENABLE_UI_AGENT });
  }
  bridgeState.started = true;
  connectAgentSocket();
};

// Test helper: push the current page content to the backend without waiting
// for a request. Backend just logs it.
export const sendPageContent = async () => {
  const socket = bridgeState.socket;
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    agentLog("warn", "sendPageContent skipped: socket not open", { readyState: socket?.readyState });
    return;
  }
  const payload = await Agent.getPageContent();
  socket.send(JSON.stringify({ Type: "pageContent", Payload: payload }));
  agentLog("info", "sendPageContent pushed", { htmlBytes: payload.HTML.length, components: payload.Components.length });
};

if (typeof window !== "undefined") {
  // Expose on the existing devtools handle so you can call __agent.sendPageContent().
  (window as any).__agent = Object.assign((window as any).__agent || {}, {
    sendPageContent,
    debugLog: () => JSON.parse(localStorage.getItem(AGENT_LOG_KEY) || "[]"),
  });
}
