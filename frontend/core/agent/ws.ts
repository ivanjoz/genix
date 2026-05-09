// WebSocket bridge: backend -> browser command channel.
// Connects to the local Go backend so it can drive the page as an agent
// (list components, invoke methods, capture HTML, screenshots).

import { Agent, agentHandles, isAgentEnabled, type AgentMethodName, type AgentOption } from "./registry";

// Wire envelope shared with the Go backend (capitalized field names since the
// Go side avoids json tags).
type WsMessage = { ID: number; Type: string; Payload?: any };

const bridgeState = {
  socket: null as WebSocket | null,
  reconnectDelayMs: 1000,
  // getDisplayMedia stream + offscreen <video> reused across screenshot calls
  // so the user is only prompted once per session.
  screenStream: null as MediaStream | null,
  videoEl: null as HTMLVideoElement | null,
  started: false,
};

const remapAgentList = (items: Array<{ id: number; type: string; label: string }>) =>
  items.map((item) => ({ ID: item.id, Type: item.type, Label: item.label }));

const remapAgentDescribe = (items: Array<{ id: number; type: string; label: string; methods: string[] }>) =>
  items.map((item) => ({ ID: item.id, Type: item.type, Label: item.label, Methods: item.methods }));

const remapOptions = (options: AgentOption[] | undefined) =>
  (options || []).map((option) => ({ ID: option.id, Value: option.value }));

// Captures a frame from the active screen-share. First invocation triggers
// the user's "share screen" permission prompt; subsequent calls reuse the stream.
const captureScreenshot = async () => {
  let stream = bridgeState.screenStream;
  if (!stream || !stream.active) {
    stream = await navigator.mediaDevices.getDisplayMedia({ video: true, audio: false });
    bridgeState.screenStream = stream;
    // If the user clicks "Stop sharing" in the browser bar, drop the cached stream.
    for (const track of stream.getVideoTracks()) {
      track.addEventListener("ended", () => {
        if (bridgeState.screenStream === stream) { bridgeState.screenStream = null; }
      });
    }
  }
  const track = stream.getVideoTracks()[0];
  const settings = track.getSettings();

  let video = bridgeState.videoEl;
  if (!video) {
    video = document.createElement("video");
    video.muted = true;
    (video as any).playsInline = true;
    video.style.cssText = "position:fixed;left:-9999px;top:-9999px;width:1px;height:1px;opacity:0;pointer-events:none";
    document.body.appendChild(video);
    bridgeState.videoEl = video;
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

// Single generic dispatcher: backend sends { HandleID, Method, Args }, we look
// up the handle, find the named method, call it.
const invokeMethod = (payload: { HandleID: number; Method: AgentMethodName; Args?: unknown[] }) => {
  const handle = agentHandles.get(payload.HandleID);
  if (!handle) { throw new Error(`unknown handle id: ${payload.HandleID}`); }
  const fn = handle[payload.Method] as ((...args: unknown[]) => unknown) | undefined;
  if (typeof fn !== "function") {
    throw new Error(`handle ${payload.HandleID} (${handle.type}) does not implement ${payload.Method}`);
  }
  const args = Array.isArray(payload.Args) ? payload.Args : [];
  const result = fn.apply(handle, args);
  if (result === undefined) { return null; }
  if (payload.Method === "getOptions") { return remapOptions(result as AgentOption[]); }
  return result;
};

const commandHandlers: Record<string, (payload: any) => Promise<any> | any> = {
  getPageContent: () => Agent.getPageContent(),
  agentList: (payload) => remapAgentList(Agent.list(payload || undefined)),
  agentDescribe: () => remapAgentDescribe(Agent.describe()),
  screenshot: () => captureScreenshot(),
  "agent.invoke": invokeMethod,
};

const releaseScreenStream = () => {
  if (!bridgeState.screenStream) { return; }
  for (const track of bridgeState.screenStream.getTracks()) { track.stop(); }
  bridgeState.screenStream = null;
};

const scheduleReconnect = () => {
  const delay = Math.min(bridgeState.reconnectDelayMs, 10_000);
  bridgeState.reconnectDelayMs = Math.min(delay * 2, 10_000);
  console.info("[Agent] retry in", delay, "ms");
  setTimeout(connectAgentSocket, delay);
};

const connectAgentSocket = () => {
  if (typeof window === "undefined") { return; }
  if (bridgeState.socket && bridgeState.socket.readyState <= WebSocket.OPEN) { return; }

  // Local-only: the backend always listens on this port during dev.
  const url = "ws://localhost:3589/ws/agent";
  console.info("[Agent] connecting", url);
  let socket: WebSocket;
  try {
    socket = new WebSocket(url);
  } catch (error) {
    console.warn("[Agent] socket constructor failed:", error);
    scheduleReconnect();
    return;
  }
  bridgeState.socket = socket;

  socket.addEventListener("open", () => {
    console.info("[Agent] connected");
    bridgeState.reconnectDelayMs = 1000;
    socket.send(JSON.stringify({ Type: "ready" }));
  });

  socket.addEventListener("message", async (event) => {
    let message: WsMessage;
    try {
      message = JSON.parse(typeof event.data === "string" ? event.data : "");
    } catch (parseError) {
      console.warn("[Agent] bad message json:", parseError);
      return;
    }
    const handler = commandHandlers[message.Type];
    if (!handler) {
      socket.send(JSON.stringify({ ID: message.ID, Type: "error", Payload: { Message: `unknown command: ${message.Type}` } }));
      return;
    }
    try {
      const result = await handler(message.Payload);
      socket.send(JSON.stringify({ ID: message.ID, Type: "result", Payload: result }));
    } catch (commandError: any) {
      const errorMessage = String(commandError?.message || commandError);
      console.warn("[Agent] command error:", message.Type, errorMessage);
      socket.send(JSON.stringify({ ID: message.ID, Type: "error", Payload: { Message: errorMessage } }));
    }
  });

  const onClose = () => {
    if (bridgeState.socket !== socket) { return; }
    bridgeState.socket = null;
    releaseScreenStream();
    scheduleReconnect();
  };
  socket.addEventListener("close", onClose);
  socket.addEventListener("error", () => {
    try { socket.close(); } catch { /* ignore */ }
  });
};

export const startAgentBridge = () => {
  if (bridgeState.started) { return; }
  if (typeof window === "undefined") { return; }
  if (!isAgentEnabled()) { return; }
  bridgeState.started = true;
  connectAgentSocket();
};

// Test helper: push the current page content to the backend without waiting
// for a request. Backend just logs it.
export const sendPageContent = async () => {
  const socket = bridgeState.socket;
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    console.warn("[Agent] sendPageContent: socket not open");
    return;
  }
  const payload = await Agent.getPageContent();
  console.log("payload", payload)
  socket.send(JSON.stringify({ Type: "pageContent", Payload: payload }));
  console.info("[Agent] sendPageContent: pushed", payload.HTML.length, "bytes");
};

if (typeof window !== "undefined") {
  // Expose on the existing devtools handle so you can call __agent.sendPageContent().
  (window as any).__agent = Object.assign((window as any).__agent || {}, { sendPageContent });
}
