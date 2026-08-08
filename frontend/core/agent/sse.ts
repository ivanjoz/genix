// SSE + POST channel between the backend (the agent) and this browser tab.
//
// One long-lived stream carries every server→browser message — page-driving
// commands AND chat events — and every browser→backend message (command
// replies, the pageContent test push) is a short POST. A chat turn is a
// separate plain POST that blocks until the turn ends; its events arrive on the
// stream, not in its response.
//
// The stream has two possible hosts, and the client code is identical against
// either: the backend itself (`/agent/stream` + `/agent/in`) when it runs
// locally or on the VPS, or the SSE bridge (`/sse` + `/in`, see server_utils/)
// when the backend runs in Lambda and cannot hold a connection open. Env
// resolves which one applies (AGENT_STREAM_BASE).
//
// The stream is lazy: nothing connects on app boot. It opens the first time the
// chat is used and stays open from then on, reconnecting on drops. Command
// execution lives in commands.ts.

import { Env } from "$core/env";
import { security } from "$libs/ui-runtime.svelte";
import { Agent, isAgentEnabled } from "$components/agent/registry";
import { encodeChannelToken, isValidTabID, mintTabID } from "./channel";
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
    // Logging must never break the agent channel.
  }
  const logger = level === "warn" ? console.warn : console.info;
  logger(`[Agent] ${message}`, detail || "");
};

// agentStreamEndpoint resolves where the stream and the inbound POST live for
// the currently selected API endpoint. The bridge exposes them at its root; the
// backend serves them under /agent/. Env.AGENT_STREAM_BASE already applied the
// selection rule, so a differing base is exactly what identifies the bridge.
const agentStreamEndpoint = () => {
  const streamBase = String(Env.AGENT_STREAM_BASE || "").replace(/\/+$/, "");
  const selectedApiBase = String(Env.selectedApiEndpointRoute || "").replace(/\/+$/, "");
  const usesBridge = !!streamBase && streamBase !== selectedApiBase;
  return {
    streamUrl: `${streamBase}${usesBridge ? "/sse" : "/agent/stream"}`,
    inboundUrl: `${streamBase}${usesBridge ? "/in" : "/agent/in"}`,
    usesBridge,
  };
};

// The bridge authenticates every client call with the session token, and it is
// how it derives the company/user half of the channel identity. The backend's
// own endpoints ignore the header, so it is sent unconditionally.
const authorizationHeaders = (): Record<string, string> => {
  const sessionToken = security.getToken(true);
  return sessionToken ? { Authorization: `Bearer ${sessionToken}` } : {};
};

// getAgentTabID returns the per-tab id (8 chars) the backend routes page
// commands and chat events to. Stored in sessionStorage so it survives
// navigations within the tab but is unique per tab. It is also the key of this
// tab's local chat history (chat_history.idb), which is why it stays separate
// from the channel token: the token embeds the company and would change when
// the user switches, splitting the history.
const AGENT_TAB_KEY = "__agent_tab_id";
const AGENT_SESSION_KEY = "__agent_session_id";

export const getAgentTabID = (): string => {
  if (typeof window === "undefined") { return ""; }
  let id = sessionStorage.getItem(AGENT_TAB_KEY) || "";
  // A value from an older build (a 36-char UUID) can't go into a channel token.
  if (!isValidTabID(id)) {
    id = mintTabID();
    sessionStorage.setItem(AGENT_TAB_KEY, id);
  }
  return id;
};

// getAgentChannelToken names this tab's stream on the wire. Recomputed on every
// use rather than stored: the company can change mid-session and the channel
// must follow it.
export const getAgentChannelToken = (): string =>
  encodeChannelToken(Env.getCompanyID() || 0, getAgentUserID(), getAgentTabID());

// getAgentSessionID scopes the conversation history the backend persists. It is
// minted here, not on the server: under Lambda two turns of the same
// conversation can be served by different execution environments, and a
// server-minted id would restart the LLM's history on every instance change.
export const getAgentSessionID = (): number => {
  if (typeof window === "undefined") { return 0; }
  const stored = Number(sessionStorage.getItem(AGENT_SESSION_KEY));
  if (stored > 0) { return stored; }
  const sessionID = Math.floor(Date.now() / 1000);
  sessionStorage.setItem(AGENT_SESSION_KEY, String(sessionID));
  return sessionID;
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

// requireChannelToken fails loudly instead of connecting to a channel the
// backend will reject: without a company or a session there is no identity to
// name, and the stream would 401 on every retry.
const requireChannelToken = (): string => {
  const channelToken = getAgentChannelToken();
  if (!channelToken) {
    throw new Error("no hay sesión activa para abrir el canal del agente");
  }
  return channelToken;
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

// --- Message plumbing ---------------------------------------------------------

// postIn sends one browser→backend message. Fire-and-forget for the caller's
// purposes — the backend acknowledges immediately and any real response arrives
// asynchronously down the stream.
const postIn = async (body: object): Promise<void> => {
  try {
    await fetch(`${agentStreamEndpoint().inboundUrl}?ch=${requireChannelToken()}`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authorizationHeaders() },
      body: JSON.stringify(body),
    });
  } catch (error) {
    agentLog("warn", "postIn failed", { error: String(error), type: (body as { Type?: string }).Type });
  }
};

// dispatchMessage routes one decoded frame: `ID > 0` is a command awaiting a
// reply, anything else is a chat event for the widget.
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

// --- Stream lifecycle ---------------------------------------------------------

// Server frames that drive the connection itself instead of the conversation.
const FRAME_BRIDGE_READY = "bridgeReady";
const FRAME_REPLACED = "replaced";

const STREAM_READY_TIMEOUT_MS = 10_000;
const STREAM_MAX_RECONNECT_DELAY_MS = 10_000;

interface StreamState {
  started: boolean;
  ready: boolean;
  abortController: AbortController | null;
  reconnectDelayMs: number;
  readyWaiters: Array<() => void>;
}

const streamState: StreamState = {
  started: false,
  ready: false,
  abortController: null,
  reconnectDelayMs: 1000,
  readyWaiters: [],
};

// isStreamConnected reports whether the tab currently has a live, handshaken
// stream. Exposed for devtools and for the chat widget's diagnostics.
export const isStreamConnected = (): boolean => streamState.ready;

// markStreamReady fires on the handshake frame, which the server only sends
// once the tab is registered — i.e. once the backend can actually reach us.
const markStreamReady = () => {
  streamState.ready = true;
  streamState.reconnectDelayMs = 1000;
  const waiters = streamState.readyWaiters.splice(0);
  waiters.forEach((resolve) => resolve());
};

const markStreamDisconnected = () => {
  streamState.ready = false;
  // A turn may have started a getDisplayMedia capture for screenshotReal; drop
  // it so the browser's "sharing your screen" banner doesn't outlive the stream.
  releaseScreenStream();
};

// readEventStream pulls `data: <json>\n\n` frames out of the response body.
// Comment lines (`: ping` keepalives) carry no data field and are skipped.
// Returns when the body ends; throws only on a transport failure.
const readEventStream = async (body: ReadableStream<Uint8Array>) => {
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

        if (message.Type === FRAME_BRIDGE_READY) {
          agentLog("info", "stream ready");
          markStreamReady();
          continue;
        }
        if (message.Type === FRAME_REPLACED) {
          // Another browsing context took our tab id. Rotate to a fresh one and
          // let the reconnect loop redial, so both tabs stay alive instead of
          // ping-ponging the shared id.
          const next = rotateAgentTabID();
          agentLog("warn", "stream replaced by another context; rotated", { next });
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

// connectAgentStream opens one connection and resolves when it ends. fetch (not
// EventSource) because the bridge authenticates with an Authorization header,
// which EventSource cannot set.
const connectAgentStream = async () => {
  const { streamUrl, usesBridge } = agentStreamEndpoint();
  // company/user no longer travel as separate params: they are inside the token,
  // which is also what the server checks against the session token.
  const path = encodeURIComponent(window.location.pathname || "");
  const url = `${streamUrl}?ch=${requireChannelToken()}&path=${path}`;

  const abortController = new AbortController();
  streamState.abortController = abortController;
  agentLog("info", "connecting stream", { url, tab: getAgentTabID(), bridged: usesBridge });

  const response = await fetch(url, {
    headers: { Accept: "text/event-stream", ...authorizationHeaders() },
    signal: abortController.signal,
  });
  if (!response.ok) {
    throw new Error(`stream HTTP ${response.status}`);
  }
  if (!response.body) {
    throw new Error("stream sin cuerpo");
  }
  await readEventStream(response.body);
};

const delay = (milliseconds: number) => new Promise((resolve) => setTimeout(resolve, milliseconds));

// runStreamLoop keeps the stream up for as long as it is started, backing off
// exponentially between attempts.
const runStreamLoop = async () => {
  while (streamState.started) {
    try {
      await connectAgentStream();
      agentLog("info", "stream closed by server");
    } catch (streamError) {
      if (!streamState.started) { break; }
      agentLog("warn", "stream error", { error: String(streamError) });
    }
    markStreamDisconnected();
    if (!streamState.started) { break; }

    const backoffMs = Math.min(streamState.reconnectDelayMs, STREAM_MAX_RECONNECT_DELAY_MS);
    streamState.reconnectDelayMs = Math.min(backoffMs * 2, STREAM_MAX_RECONNECT_DELAY_MS);
    agentLog("info", "retry scheduled", { delayMs: backoffMs });
    await delay(backoffMs);
  }
  markStreamDisconnected();
};

// ensureAgentStream starts the stream if it isn't running and resolves once the
// handshake confirms the backend can reach this tab. Every caller that is about
// to make the backend push something must await it — that is what guarantees
// there is a channel to push to.
export const ensureAgentStream = (timeoutMs = STREAM_READY_TIMEOUT_MS): Promise<void> => {
  if (typeof window === "undefined") {
    return Promise.reject(new Error("el stream del agente requiere un navegador"));
  }
  if (!streamState.started) {
    streamState.started = true;
    void runStreamLoop();
  }
  if (streamState.ready) { return Promise.resolve(); }

  return new Promise<void>((resolve, reject) => {
    let settled = false;
    const timer = setTimeout(() => {
      if (settled) { return; }
      settled = true;
      reject(new Error("timeout esperando la conexión con el agente"));
    }, timeoutMs);

    streamState.readyWaiters.push(() => {
      if (settled) { return; }
      settled = true;
      clearTimeout(timer);
      resolve();
    });
  });
};

export const stopAgentStream = () => {
  streamState.started = false;
  streamState.abortController?.abort();
  streamState.abortController = null;
  markStreamDisconnected();
};

// --- Turn ---------------------------------------------------------------------

// runAgentTurn is the chat widget's send path. The turn is a plain POST that
// resolves when the backend finishes; replies, status and errors arrive
// meanwhile through subscribeAgentChat. `context` carries mode-specific payload
// (e.g. the builder's sections serialized to HTML); empty when the mode needs
// none.
export const runAgentTurn = async (
  message: string,
  modelHash: string,
  timestamp: number,
  modeID: number,
  context: string,
): Promise<void> => {
  // Lazy start: the stream opens here on the first turn if the chat panel
  // didn't already open it, and the handshake guarantees the backend can push
  // this turn's events before we ask for them.
  await ensureAgentStream();

  const body = {
    Channel: requireChannelToken(),
    SessionID: getAgentSessionID(),
    Message: message,
    ModelHash: modelHash,
    Timestamp: timestamp,
    ModeID: modeID,
    Context: context,
    // Sent per-turn rather than once at connect, so navigating by hand between
    // turns is reflected without a reconnect.
    Path: window.location.pathname || "",
  };
  agentLog("info", "turn start", { channel: body.Channel, modeID, modelHash, bytes: message.length, contextBytes: context.length });

  const response = await fetch(Env.makeRoute("p-agent-turn"), {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authorizationHeaders() },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const detail = await response.text().catch(() => "");
    agentLog("warn", "turn rejected", { status: response.status, detail });
    throw new Error(`agent turn failed (${response.status})`);
  }
  agentLog("info", "turn finished");
};

// Test helper: push the current page content to the backend, which just logs it.
export const sendPageContent = async () => {
  await ensureAgentStream();
  const payload = await Agent.getPageContent();
  await postIn({ Type: "pageContent", Payload: payload });
  agentLog("info", "sendPageContent pushed", { htmlBytes: payload.HTML.length, components: payload.Components.length });
};

const AGENT_AUTOCONNECT_KEY = "__agent_bridge";

// maybeAutoConnect honours the two opt-ins that predate the lazy stream:
// `?agent=1` on the URL for a one-off, or the localStorage flag it sets for a
// sticky one. They exist for the external HTTP driver (Claude Code / Gemini),
// whose ResolveTab needs a tab already connected and which cannot open one
// itself. The in-app chat never needs this — it connects when it is used.
const maybeAutoConnect = () => {
  let enabled = false;
  try {
    const flag = new URLSearchParams(window.location.search).get("agent");
    if (flag === "1") { localStorage.setItem(AGENT_AUTOCONNECT_KEY, "1"); }
    if (flag === "0") { localStorage.removeItem(AGENT_AUTOCONNECT_KEY); }
    enabled = localStorage.getItem(AGENT_AUTOCONNECT_KEY) === "1";
  } catch {
    // Storage blocked (private mode / embedded) — treat as opted out.
  }
  if (enabled) {
    void ensureAgentStream().catch((error) => agentLog("warn", "autoconnect failed", { error: String(error) }));
  }
};

if (typeof window !== "undefined") {
  // Expose on the existing devtools handle so the external HTTP driver's tab can
  // be brought up by hand: __agent.connect().
  (window as any).__agent = Object.assign((window as any).__agent || {}, {
    connect: () => ensureAgentStream().catch((error) => agentLog("warn", "connect failed", { error: String(error) })),
    disconnect: stopAgentStream,
    sendPageContent,
    isConnected: isStreamConnected,
    agentEnabled: isAgentEnabled,
    debugLog: () => JSON.parse(localStorage.getItem(AGENT_LOG_KEY) || "[]"),
  });
  maybeAutoConnect();
}
