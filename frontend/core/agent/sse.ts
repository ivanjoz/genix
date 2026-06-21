// SSE + POST bridge between the backend (the agent) and this browser tab.
// One EventSource (`/agent/stream`) carries every server→browser message —
// page-driving commands AND chat events. Every browser→backend message
// (command replies, the connect "ready" proof, the pageContent test push, and
// chat user messages) is a POST to `/agent/in`. Command execution lives in
// commands.ts; this file owns the stream lifecycle and the message plumbing.

import { Env } from "$core/env";
import { Agent, isAgentEnabled } from "$components/agent/registry";
import { releaseScreenStream, runCommand, type WsMessage } from "./commands";

const AGENT_LOG_KEY = "__agent_debug_log";
const AGENT_LOG_LIMIT = 120;

const agentLog = (level: "info" | "warn", message: string, detail?: unknown) => {
  const entry = { at: new Date().toISOString(), level, message, detail };
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

// agentHttpBase derives the http scheme + host from the selected API endpoint.
// Env.API_ROUTES.MAIN ends with `/api/` (it's the HTTP API root) — the
// `/agent/*` handlers are mounted at the server root, not under `/api`, so we
// strip that suffix.
export const agentHttpBase = (): string => {
  const main = Env.API_ROUTES.MAIN || "";
  return main.replace(/\/+$/, "").replace(/\/api$/, "");
};

// getAgentTabID returns the per-tab id sent on the stream URL so the backend
// can route page commands + chat events to the correct tab. Stored in
// sessionStorage so it survives navigations within the tab but is unique per
// tab. The chat widget reads the same id so chat and page traffic share it.
const AGENT_TAB_KEY = "__agent_tab_id";

const mintTabID = (): string =>
  (window.crypto?.randomUUID?.() as string | undefined)
    || `tab-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;

export const getAgentTabID = (): string => {
  if (typeof window === "undefined") { return ""; }
  let id = sessionStorage.getItem(AGENT_TAB_KEY);
  if (!id) {
    id = mintTabID();
    sessionStorage.setItem(AGENT_TAB_KEY, id);
  }
  return id;
};

// rotateAgentTabID forces a brand-new tab id. Used when the backend tells us
// our stream was "replaced": within a live page there is only ever one stream,
// so a replace can only mean another browsing context (e.g. a duplicated tab,
// which clones sessionStorage) grabbed our id. Minting a fresh id lets both
// tabs coexist instead of endlessly kicking each other off the shared id.
const rotateAgentTabID = (): string => {
  const id = mintTabID();
  sessionStorage.setItem(AGENT_TAB_KEY, id);
  return id;
};

// Reads the cached user id from localStorage so the backend can stamp it on
// the stream. Best-effort: missing/invalid → 0 (backend treats this as anon).
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

// --- Chat event pub/sub -------------------------------------------------------
// The chat widget subscribes here for agentReply/agentStatus/agentError pushed
// down the shared stream, so it doesn't need its own connection.

export interface ChatStreamEvent {
  Type: string;
  Payload?: unknown;
}

type ChatStreamListener = (event: ChatStreamEvent) => void;
const chatListeners = new Set<ChatStreamListener>();

export const subscribeAgentChat = (fn: ChatStreamListener): (() => void) => {
  chatListeners.add(fn);
  return () => { chatListeners.delete(fn); };
};

// --- Stream lifecycle ---------------------------------------------------------

interface BridgeState {
  source: EventSource | null;
  reconnectDelayMs: number;
  started: boolean;
  connected: boolean;
}

const bridgeState: BridgeState = {
  source: null,
  reconnectDelayMs: 1000,
  started: false,
  connected: false,
};

// isStreamConnected reports whether the shared SSE stream is open, so the chat
// widget can warn the user before a POST that would have no return path.
export const isStreamConnected = (): boolean => bridgeState.connected;

// postIn sends one browser→backend message. Fire-and-forget for the caller's
// purposes — the backend acknowledges with `{}` and any real response arrives
// asynchronously down the stream.
const postIn = async (body: object): Promise<void> => {
  const tab = getAgentTabID();
  try {
    await fetch(`${agentHttpBase()}/agent/in?tab=${encodeURIComponent(tab)}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch (error) {
    agentLog("warn", "postIn failed", { error: String(error), type: (body as { Type?: string }).Type });
  }
};

// postChatMessage is the chat widget's send path: POST a userMessage; the
// reply/status/error events come back via subscribeAgentChat. `context` carries
// mode-specific context (e.g. the builder's sections serialized to HTML); empty
// when the active mode needs none.
export const postChatMessage = (message: string, modelHash: string, timestamp: number, modeID: number, context: string): Promise<void> =>
  postIn({ Type: "userMessage", Payload: { Message: message, ModelHash: modelHash, Timestamp: timestamp, ModeID: modeID, Context: context } });

const handleStreamMessage = async (raw: unknown) => {
  let message: WsMessage & ChatStreamEvent;
  try {
    message = JSON.parse(typeof raw === "string" ? raw : "");
  } catch (parseError) {
    agentLog("warn", "bad stream json", { error: String(parseError), rawType: typeof raw });
    return;
  }

  // "replaced": another browsing context took our tab id. Rotate to a fresh id
  // and reconnect so both tabs stay alive instead of ping-ponging the shared id.
  if (message.Type === "replaced") {
    const next = rotateAgentTabID();
    agentLog("warn", "stream replaced by another context; rotated", { next });
    teardownStream();
    scheduleReconnect();
    return;
  }

  // A non-zero ID means it's a backend-issued page command awaiting a reply.
  // Anything else is a chat event for the widget.
  if (typeof message.ID === "number" && message.ID > 0) {
    agentLog("info", "command received", { id: message.ID, type: message.Type, payloadBytes: JSON.stringify(message.Payload ?? null).length });
    try {
      const result = await runCommand(message.Type, message.Payload);
      await postIn({ ID: message.ID, Type: "result", Payload: result });
    } catch (commandError: any) {
      const errorMessage = String(commandError?.message || commandError);
      agentLog("warn", "command error", { id: message.ID, type: message.Type, error: errorMessage });
      await postIn({ ID: message.ID, Type: "error", Payload: { Message: errorMessage } });
    }
    return;
  }

  agentLog("info", "chat event received", { type: message.Type });
  chatListeners.forEach((fn) => { try { fn(message); } catch { /* listener must not break the stream */ } });
};

const teardownStream = () => {
  bridgeState.connected = false;
  releaseScreenStream();
  if (bridgeState.source) {
    try { bridgeState.source.close(); } catch { /* ignore */ }
    bridgeState.source = null;
  }
};

const scheduleReconnect = () => {
  const delay = Math.min(bridgeState.reconnectDelayMs, 10_000);
  bridgeState.reconnectDelayMs = Math.min(delay * 2, 10_000);
  agentLog("info", "retry scheduled", { delayMs: delay });
  setTimeout(connectAgentStream, delay);
};

const connectAgentStream = () => {
  if (typeof window === "undefined") { return; }
  // EventSource is OPEN or CONNECTING → nothing to do (it self-heals).
  if (bridgeState.source && bridgeState.source.readyState !== EventSource.CLOSED) { return; }

  const tab = getAgentTabID();
  const company = Env.getCompanyID() || 0;
  const user = getAgentUserID();
  const path = encodeURIComponent(window.location.pathname || "");
  const url = `${agentHttpBase()}/agent/stream?tab=${encodeURIComponent(tab)}&company=${company}&user=${user}&path=${path}`;
  agentLog("info", "connecting stream", { url, tab, company, user, apiMain: Env.API_ROUTES.MAIN, local: globalThis._isLocal, enabled: isAgentEnabled() });

  let source: EventSource;
  try {
    source = new EventSource(url);
  } catch (error) {
    agentLog("warn", "EventSource constructor failed", { error: String(error), url });
    scheduleReconnect();
    return;
  }
  bridgeState.source = source;

  source.addEventListener("open", () => {
    if (bridgeState.source !== source) { return; }
    agentLog("info", "stream connected", { url, tab });
    bridgeState.connected = true;
    bridgeState.reconnectDelayMs = 1000;
    // Announce ourselves so backend diagnostics know the bundle opened the stream.
    void postIn({
      Type: "ready",
      Payload: {
        URL: window.location.href,
        Path: window.location.pathname,
        AgentEnabled: isAgentEnabled(),
        Handles: Agent.describe().length,
        UserAgent: navigator.userAgent,
      },
    });
  });

  source.addEventListener("message", (event) => {
    void handleStreamMessage((event as MessageEvent).data);
  });

  source.addEventListener("error", () => {
    if (bridgeState.source !== source) { return; }
    bridgeState.connected = false;
    releaseScreenStream();
    // EventSource auto-reconnects while CONNECTING. Only when it gives up
    // (CLOSED — e.g. the server returned a non-2xx) do we drive a manual retry.
    if (source.readyState === EventSource.CLOSED) {
      agentLog("warn", "stream closed; scheduling manual reconnect", { url, tab });
      bridgeState.source = null;
      scheduleReconnect();
    } else {
      agentLog("info", "stream interrupted; EventSource will retry", { url, tab });
    }
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
    // whether the route/proxy/stream path is alive.
    agentLog("warn", "agent registry disabled; starting bridge for diagnostics", { local: globalThis._isLocal, enableFlag: window.ENABLE_UI_AGENT });
  }
  bridgeState.started = true;
  connectAgentStream();
};

// Test helper: push the current page content to the backend, which just logs
// it. Mirrors the old WS test push.
export const sendPageContent = async () => {
  if (!bridgeState.connected) {
    agentLog("warn", "sendPageContent skipped: stream not connected");
    return;
  }
  const payload = await Agent.getPageContent();
  await postIn({ Type: "pageContent", Payload: payload });
  agentLog("info", "sendPageContent pushed", { htmlBytes: payload.HTML.length, components: payload.Components.length });
};

if (typeof window !== "undefined") {
  // Expose on the existing devtools handle so you can call __agent.sendPageContent().
  (window as any).__agent = Object.assign((window as any).__agent || {}, {
    sendPageContent,
    debugLog: () => JSON.parse(localStorage.getItem(AGENT_LOG_KEY) || "[]"),
  });
}
