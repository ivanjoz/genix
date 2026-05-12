// Command handlers invoked by the WS bridge in response to backend messages.
// Kept separate from ws.ts so the socket layer stays a thin transport.
//
// All wire types use capitalized field names to match the Go side (which omits
// json tags). Internal registry types stay lowercase.

import { goto } from "$app/navigation";
import { Core } from "$core/store.svelte";
import { canUserAccessRoute } from "$core/security";
import { Agent, agentHandles, type AgentHandle, type AgentListFilter, type AgentMethodName } from "./registry";

// --- Wire envelopes -----------------------------------------------------------

export interface WsMessage {
  ID: number;
  Type: string;
  Payload?: unknown;
}

interface MenuOption {
  Name: string;
  Route: string;
}

interface MenuGroup {
  ID: number;
  Name: string;
  Options: MenuOption[];
}

interface ScreenshotResult {
  MIME: string;
  Base64: string;
  Width: number;
  Height: number;
}

interface InvokePayload {
  HandleID: number;
  Method: AgentMethodName;
  Args?: unknown[];
}

interface NavigatePayload {
  Route: string;
}

// Mirrors the Go agent.InvocationResult: one entry per executed invocation in
// a batch reply. Value/Error are mutually exclusive (Value on OK, Error
// otherwise) and omitted from the JSON when absent.
interface InvocationResult {
  OK: boolean;
  Value?: unknown;
  Error?: string;
}

type CommandHandler = (payload: any) => Promise<unknown> | unknown;

// --- Pacing -------------------------------------------------------------------

// Minimum delay between page-mutating actions. Pacing applies *per action* —
// for `agent.invoke`, that means between each item in the batch, not just
// between WS messages. Read-only queries skip the gap entirely.
const ACTION_GAP_MS = 250;

let lastActionAt = 0;

// pacedRun gates `fn` so it starts no sooner than ACTION_GAP_MS after the
// previous paced action completed, and stamps the timestamp on completion
// (success or failure) so the next action's gap is measured from now.
const pacedRun = async <T>(fn: () => Promise<T> | T): Promise<T> => {
  const wait = ACTION_GAP_MS - (Date.now() - lastActionAt);
  if (wait > 0) { await new Promise((resolve) => setTimeout(resolve, wait)); }
  try {
    return await fn();
  } finally {
    lastActionAt = Date.now();
  }
};

// --- Screenshot ---------------------------------------------------------------

interface ScreenshotState {
  stream: MediaStream | null;
  videoEl: HTMLVideoElement | null;
}

// getDisplayMedia stream + offscreen <video> reused across screenshot calls
// so the user is only prompted once per session.
const screenshotState: ScreenshotState = { stream: null, videoEl: null };

// Captures a frame from the active screen-share. First invocation triggers
// the user's "share screen" permission prompt; subsequent calls reuse the stream.
const captureScreenshot = async (): Promise<ScreenshotResult> => {
  let stream = screenshotState.stream;
  if (!stream || !stream.active) {
    stream = await navigator.mediaDevices.getDisplayMedia({ video: true, audio: false });
    screenshotState.stream = stream;
    // If the user clicks "Stop sharing" in the browser bar, drop the cached stream.
    for (const track of stream.getVideoTracks()) {
      track.addEventListener("ended", () => {
        if (screenshotState.stream === stream) { screenshotState.stream = null; }
      });
    }
  }
  const track = stream.getVideoTracks()[0];
  const settings = track.getSettings();

  let video = screenshotState.videoEl;
  if (!video) {
    video = document.createElement("video");
    video.muted = true;
    (video as any).playsInline = true;
    video.style.cssText = "position:fixed;left:-9999px;top:-9999px;width:1px;height:1px;opacity:0;pointer-events:none";
    document.body.appendChild(video);
    screenshotState.videoEl = video;
  }
  if (video.srcObject !== stream) {
    video.srcObject = stream;
    await video.play();
  }
  // Wait for the first frame to be ready when video metadata isn't loaded yet.
  if (!video.videoWidth) {
    await new Promise<void>((resolve) => {
      const onLoaded = () => { video!.removeEventListener("loadedmetadata", onLoaded); resolve(); };
      video!.addEventListener("loadedmetadata", onLoaded);
    });
  }

  const width = settings.width || video.videoWidth;
  const height = settings.height || video.videoHeight;
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) { throw new Error("2d context unavailable"); }
  ctx.drawImage(video, 0, 0, width, height);
  const dataUrl = canvas.toDataURL("image/png");
  const base64 = dataUrl.split(",")[1] || "";
  return { MIME: "image/png", Base64: base64, Width: width, Height: height };
};

export const releaseScreenStream = () => {
  if (!screenshotState.stream) { return; }
  for (const track of screenshotState.stream.getTracks()) { track.stop(); }
  screenshotState.stream = null;
};

// --- Pulse overlay ------------------------------------------------------------

// Visual radar-style ping anchored at the center of the targeted component
// every time the agent invokes a write method. Purely cosmetic: helps the
// human watching the page see which element the agent is acting on.

const PULSE_STYLE_ID = "__agent_pulse_style";
const PULSE_CLASS = "__agent-pulse";
const PULSE_LIFETIME_MS = 900;

// Read-only / query methods that should NOT trigger a pulse — they don't
// mutate the page and would just add noise during option lookups.
const PULSE_SKIP_METHODS = new Set<AgentMethodName>([
  "getOptions",
  "getOptionsChild",
  "getValue",
]);

const ensurePulseStyle = () => {
  if (document.getElementById(PULSE_STYLE_ID)) { return; }
  const style = document.createElement("style");
  style.id = PULSE_STYLE_ID;
  style.textContent = `
.${PULSE_CLASS} {
  position: fixed;
  pointer-events: none;
  width: 56px;
  height: 56px;
  margin: -28px 0 0 -28px;
  border: 2px solid #ff3b30;
  border-radius: 50%;
  box-sizing: border-box;
  z-index: 2147483647;
  opacity: 0;
  animation: __agent-pulse-fade 800ms ease-out forwards;
}
.${PULSE_CLASS}::before,
.${PULSE_CLASS}::after {
  content: "";
  position: absolute;
  inset: -2px;
  border: 2px solid #ff3b30;
  border-radius: 50%;
  box-sizing: border-box;
  opacity: 0;
}
.${PULSE_CLASS}::before { animation: __agent-pulse-wave 800ms ease-out forwards; }
.${PULSE_CLASS}::after  { animation: __agent-pulse-wave 800ms ease-out 180ms forwards; }
@keyframes __agent-pulse-fade {
  0%, 70% { opacity: 1; }
  100%    { opacity: 0; }
}
@keyframes __agent-pulse-wave {
  0%   { transform: scale(1);   opacity: 1; }
  70%  { opacity: 0.2; }
  100% { transform: scale(2);   opacity: 0; }
}
`;
  document.head.appendChild(style);
};

// findActionElement picks the most specific DOM target for the invocation:
// cell methods (`*Child`) point at the cell, `remove` on a SearchCard points
// at the option chip, `select` with a composite arg points at the table row.
// Anything else falls back to the handle's own root element.
const findActionElement = (
  handle: AgentHandle,
  method: AgentMethodName,
  args: unknown[],
): HTMLElement | null => {
  const root = document.querySelector<HTMLElement>(`[data-id="${handle.type}:${handle.id}"]`);
  if (method.endsWith("Child") && args.length > 0) {
    const childID = args[0];
    const cell = document.querySelector<HTMLElement>(`[data-id="${handle.id}:${childID}"]`)
      || document.querySelector<HTMLElement>(`[data-id="TableRow:${handle.id}:${childID}"]`);
    if (cell) { return cell; }
  }
  if (method === "remove" && root && typeof args[0] === "string") {
    const colon = args[0].indexOf(":");
    if (colon >= 0) {
      const option = root.querySelector<HTMLElement>(`[data-id="Option:${args[0].slice(colon + 1)}"]`);
      if (option) { return option; }
    }
  }
  if (method === "select" && typeof args[0] === "string" && args[0].includes(":")) {
    const row = document.querySelector<HTMLElement>(`[data-id="TableRow:${args[0]}"]`);
    if (row) { return row; }
  }
  return root;
};

const pulseInvocation = (handle: AgentHandle, method: AgentMethodName, args: unknown[]) => {
  if (PULSE_SKIP_METHODS.has(method)) { return; }
  if (typeof document === "undefined") { return; }
  const el = findActionElement(handle, method, args);
  if (!el) { return; }
  const rect = el.getBoundingClientRect();
  if (!rect.width || !rect.height) { return; }
  ensurePulseStyle();
  const overlay = document.createElement("div");
  overlay.className = PULSE_CLASS;
  overlay.style.left = `${rect.left + rect.width / 2}px`;
  overlay.style.top = `${rect.top + rect.height / 2}px`;
  document.body.appendChild(overlay);
  setTimeout(() => overlay.remove(), PULSE_LIFETIME_MS);
};

// --- Handlers -----------------------------------------------------------------

// Resolves and invokes one method on a registered handle. The handle's method
// returns the wire shape directly (e.g. getOptions returns AgentOption[] with
// capitalized ID/Value), so no remapping happens here.
const runInvocation = async (invocation: InvokePayload): Promise<unknown> => {
  const handle = agentHandles.get(invocation.HandleID);
  if (!handle) { throw new Error(`unknown handle id: ${invocation.HandleID}`); }
  const fn = handle[invocation.Method] as ((...args: unknown[]) => unknown) | undefined;
  if (typeof fn !== "function") {
    throw new Error(`handle ${invocation.HandleID} (${handle.type}) does not implement ${invocation.Method}`);
  }
  const args = Array.isArray(invocation.Args) ? invocation.Args : [];
  pulseInvocation(handle, invocation.Method, args);
  const result = await fn.apply(handle, args);
  return result === undefined ? null : result;
};

// Executes a batch of invocations sequentially, pacing each by ACTION_GAP_MS.
// Stops on the first failure, returning the truncated result list — the
// caller (backend HTTP layer) checks the last entry's OK flag.
const invokeBatch = async (payload: InvokePayload[] | undefined): Promise<InvocationResult[]> => {
  const invocations = Array.isArray(payload) ? payload : [];
  const results: InvocationResult[] = [];
  for (const invocation of invocations) {
    try {
      const value = await pacedRun(() => runInvocation(invocation));
      results.push({ OK: true, Value: value });
    } catch (err: any) {
      results.push({ OK: false, Error: String(err?.message || err) });
      break;
    }
  }
  return results;
};

// getMenu mirrors the side-menu the user sees: the same access filter as
// SideMenu.svelte, dropping options the user can't reach.
const getMenu = (): MenuGroup[] => {
  const menus = Core.module?.menus || [];
  return menus
    .map((menu) => ({
      ID: menu.id || 0,
      Name: menu.name,
      Options: (menu.options || [])
        .filter((option) => {
          const route = String(option.route || "").trim();
          if (!route) { return false; }
          return canUserAccessRoute(option.route);
        })
        .map((option) => ({ Name: option.name, Route: option.route || "" })),
    }))
    .filter((group) => group.Options.length > 0);
};

const navigate = (payload: NavigatePayload | undefined): Promise<null> =>
  pacedRun(async () => {
    const route = payload?.Route || "";
    if (!route) { throw new Error("navigate: missing route"); }
    await goto(route);
    return null;
  });

const commandHandlers: Record<string, CommandHandler> = {
  getPageContent: () => Agent.getPageContent(),
  agentList: (payload: AgentListFilter | undefined) => Agent.list(payload),
  agentDescribe: () => Agent.describe(),
  screenshot: () => captureScreenshot(),
  "agent.invoke": invokeBatch,
  getMenu,
  navigate,
};

// --- Public dispatcher --------------------------------------------------------

// Looks up the handler for `type` and returns its result. Pacing lives inside
// the handlers that need it (agent.invoke paces between batch items;
// navigate paces against the previous action), so this stays simple.
// Throws on unknown command — the caller maps the throw to a wire error reply.
export const runCommand = async (type: string, payload: unknown): Promise<unknown> => {
  const handler = commandHandlers[type];
  if (!handler) { throw new Error(`unknown command: ${type}`); }
  return handler(payload);
};
