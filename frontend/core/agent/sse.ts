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

// isStreamConnected reports whether the idle page-bridge stream is open. Not
// meaningful for chat — a turn brings its own stream — but the external HTTP
// driver's opt-in bridge is worth being able to check from devtools.
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

// dispatchMessage routes one decoded frame. Shared by both streams — the
// turn stream and the idle page-bridge speak the identical protocol, so
// `ID > 0` is always a command awaiting a reply and anything else is a chat
// event for the widget.
const dispatchMessage = async (message: WsMessage & ChatStreamEvent) => {
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

const parseFrame = (raw: string): (WsMessage & ChatStreamEvent) | null => {
  try {
    return JSON.parse(raw);
  } catch (parseError) {
    agentLog("warn", "bad stream json", { error: String(parseError), bytes: raw.length });
    return null;
  }
};

const handleStreamMessage = async (raw: unknown) => {
  const message = parseFrame(typeof raw === "string" ? raw : "");
  if (!message) { return; }

  // "replaced": another browsing context took our tab id. Rotate to a fresh id
  // and reconnect so both tabs stay alive instead of ping-ponging the shared id.
  // Only the idle page-bridge can hit this; a turn stream is never replaced.
  if (message.Type === "replaced") {
    const next = rotateAgentTabID();
    agentLog("warn", "stream replaced by another context; rotated", { next });
    teardownStream();
    scheduleReconnect();
    return;
  }

  await dispatchMessage(message);
};

// --- Turn stream --------------------------------------------------------------
// One chat turn is one POST whose response body streams until the turn ends.
// It carries both chat events and the page commands the agent's tools issue;
// command replies go back over the separate short POST to /agent/in, because
// the response body is one-way. Nothing stays connected between turns.

const TURN_END = "turnEnd";

// readTurnStream pulls `data: <json>\n\n` frames out of the response body.
// Comment lines (`: ping` keepalives) carry no data field and are skipped.
// Resolves when the body ends or the backend marks the turn finished.
const readTurnStream = async (body: ReadableStream<Uint8Array>) => {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  try {
    for (;;) {
      const { value, done } = await reader.read();
      if (done) { return; }
      buffer += decoder.decode(value, { stream: true });
      for (let cut = buffer.indexOf("\n\n"); cut >= 0; cut = buffer.indexOf("\n\n")) {
        const frame = buffer.slice(0, cut);
        buffer = buffer.slice(cut + 2);
        if (!frame.startsWith("data:")) { continue; }
        const message = parseFrame(frame.slice(5).trim());
        if (!message) { continue; }
        if (message.Type === TURN_END) {
          agentLog("info", "turn finished");
          return;
        }
        // Deliberately not awaited: a command handler that navigates and
        // re-snapshots the page can take seconds, and blocking the read loop
        // would stall the status events queued behind it.
        void dispatchMessage(message);
      }
    }
  } finally {
    try { await reader.cancel(); } catch { /* body already closed */ }
  }
};

// runAgentTurn is the chat widget's send path. Resolves when the turn is over;
// rejects only if the turn could not be started or the stream broke. Replies,
// status and errors arrive via subscribeAgentChat as they happen. `context`
// carries mode-specific payload (e.g. the builder's sections serialized to
// HTML); empty when the active mode needs none.
export const runAgentTurn = async (
  message: string,
  modelHash: string,
  timestamp: number,
  modeID: number,
  context: string,
): Promise<void> => {
  const tab = getAgentTabID();
  const body = {
    Message: message,
    ModelHash: modelHash,
    Timestamp: timestamp,
    ModeID: modeID,
    Context: context,
    // Sent per-turn rather than once at connect, so switching company or
    // navigating by hand between turns is reflected without a reconnect.
    CompanyID: Env.getCompanyID() || 0,
    UserID: getAgentUserID(),
    Path: window.location.pathname || "",
  };
  agentLog("info", "turn start", { tab, modeID, modelHash, bytes: message.length, contextBytes: context.length });

  const response = await fetch(`${agentHttpBase()}/agent/turn?tab=${encodeURIComponent(tab)}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const detail = await response.text().catch(() => "");
    agentLog("warn", "turn rejected", { status: response.status, detail });
    throw new Error(`agent turn failed (${response.status})`);
  }
  if (!response.body) { throw new Error("agent turn returned no stream"); }

  try {
    await readTurnStream(response.body);
  } finally {
    // A turn may have started a getDisplayMedia capture for screenshotReal;
    // drop it so the browser's "sharing your screen" banner doesn't outlive it.
    releaseScreenStream();
  }
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

// startAgentBridge opens the idle page-bridge stream. The in-app chat does NOT
// need this — a turn carries its own stream (runAgentTurn). It exists only for
// the external HTTP driver (`POST /agent`, `GET /agent?get=…`, used by Claude
// Code / Gemini), whose ResolveTab needs a tab already connected and which
// can't open one itself.
//
// Opt-in only, never on app boot: the stream is long-lived and re-dials on
// every drop, so starting it unconditionally turns every idle tab into a
// permanent /agent/stream + /agent/in loop.
export const startAgentBridge = () => {
  if (bridgeState.started) {
    agentLog("info", "start skipped: already started");
    return;
  }
  if (typeof window === "undefined") { return; }
  if (!isAgentEnabled()) {
    agentLog("warn", "start skipped: agent registry disabled", { local: globalThis._isLocal, enableFlag: window.ENABLE_UI_AGENT });
    return;
  }
  bridgeState.started = true;
  connectAgentStream();
};

const BRIDGE_OPT_IN_KEY = "__agent_bridge";

// maybeStartAgentBridge honours the two opt-ins for the idle bridge: `?agent=1`
// on the URL for a one-off, or the localStorage flag it sets for a sticky one
// that survives reloads (the external driver's usual mode). `?agent=0` clears it.
const maybeStartAgentBridge = () => {
  if (typeof window === "undefined") { return; }
  let enabled = false;
  try {
    const flag = new URLSearchParams(window.location.search).get("agent");
    if (flag === "1") { localStorage.setItem(BRIDGE_OPT_IN_KEY, "1"); }
    if (flag === "0") { localStorage.removeItem(BRIDGE_OPT_IN_KEY); }
    enabled = localStorage.getItem(BRIDGE_OPT_IN_KEY) === "1";
  } catch {
    // Storage blocked (private mode / embedded) — treat as opted out.
  }
  if (enabled) { startAgentBridge(); }
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
  // Expose on the existing devtools handle so you can call __agent.connect().
  (window as any).__agent = Object.assign((window as any).__agent || {}, {
    connect: startAgentBridge,
    sendPageContent,
    debugLog: () => JSON.parse(localStorage.getItem(AGENT_LOG_KEY) || "[]"),
  });
  maybeStartAgentBridge();
}
