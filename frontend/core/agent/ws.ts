// WebSocket bridge: backend -> browser command channel.
// Connects to the local Go backend so it can drive the page as an agent. All
// command execution lives in commands.ts; this file just owns the socket
// lifecycle and the message-to-reply plumbing.

import { Agent, isAgentEnabled } from "./registry";
import { releaseScreenStream, runCommand, type WsMessage } from "./commands";

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
  socket.send(JSON.stringify({ ID: id, Type: type, Payload: payload }));
};

const handleMessage = async (socket: WebSocket, raw: unknown) => {
  let message: WsMessage;
  try {
    message = JSON.parse(typeof raw === "string" ? raw : "");
  } catch (parseError) {
    console.warn("[Agent] bad message json:", parseError);
    return;
  }
  try {
    const result = await runCommand(message.Type, message.Payload);
    sendReply(socket, message.ID, "result", result);
  } catch (commandError: any) {
    const errorMessage = String(commandError?.message || commandError);
    console.warn("[Agent] command error:", message.Type, errorMessage);
    sendReply(socket, message.ID, "error", { Message: errorMessage });
  }
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

  socket.addEventListener("message", (event) => {
    void handleMessage(socket, event.data);
  });

  socket.addEventListener("close", () => {
    if (bridgeState.socket !== socket) { return; }
    bridgeState.socket = null;
    releaseScreenStream();
    scheduleReconnect();
  });

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
  socket.send(JSON.stringify({ Type: "pageContent", Payload: payload }));
  console.info("[Agent] sendPageContent: pushed", payload.HTML.length, "bytes");
};

if (typeof window !== "undefined") {
  // Expose on the existing devtools handle so you can call __agent.sendPageContent().
  (window as any).__agent = Object.assign((window as any).__agent || {}, { sendPageContent });
}
