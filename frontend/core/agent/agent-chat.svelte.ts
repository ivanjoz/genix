// Conversation state and turn runner of the agent chat. One conversation is shared by the header
// pill (desktop) and the Info layer's Agent tab (phones), so it lives here and not in
// AgentChat.svelte: a turn keeps running, and its events keep landing, while the phone layer has the
// tab unmounted.
//
// A turn is one runAgentTurn POST that resolves when the backend finishes; the reply, the status rows
// and any page command arrive meanwhile on the tab's SSE stream (sse.ts) through subscribeAgentChat.
// History persistence is local-only via Dexie (chat_history.idb.ts), so past messages render
// instantly; the backend keeps its own copy.

import { browser } from '$app/env';
import { Core } from '#core/store.svelte.ts';
import {
  AgentTurnRejectedError,
  getAgentTabID,
  runAgentTurn,
  subscribeAgentChat,
  type ChatStreamEvent,
} from './sse';
import { getSelectedAgentModelHash } from './models.svelte';
import {
  AGENT_ROLE_AGENT,
  AGENT_ROLE_STATUS,
  AGENT_ROLE_USER,
  appendAgentChatMessage,
  loadAgentChatHistory,
  updateAgentChatMessage,
  type AgentChatRow,
  type AgentTurnPause,
} from './chat_history.idb';
import type { AgentChart, AgentTable } from './agent-answer';
import { agentModes, type AgentSectionsPayload } from './agent.svelte';

// --- Wire types (mirror agent/chat_ws.go) ---------------------------

interface AgentReplyPayload {
  Message: string;
  Summary: string;
  Timestamp: number;
  Pause?: AgentTurnPause;
  Tables?: AgentTable[];
  Charts?: AgentChart[];
}

interface AgentErrorPayload {
  Message: string;
}

interface AgentStatusPayload {
  State: string;
  Label: string;
  Step: number;
  MaxSteps: number;
}

const CHAT_TYPE_AGENT_REPLY = 'agentReply';
const CHAT_TYPE_AGENT_ERROR = 'agentError';
const CHAT_TYPE_AGENT_STATUS = 'agentStatus';
const CHAT_TYPE_AGENT_SECTIONS = 'agentSections';
const CHAT_TYPE_TURN_END = 'turnEnd';
const CHAT_LOG_KEY = '__agent_chat_debug_log';
const CHAT_LOG_LIMIT = 120;
// After an action the driven page stays visible this long before the conversation comes back.
const POST_ACTION_RESTORE_DELAY_MS = 500;

export const agentChat = $state({
  messages: [] as AgentChatRow[],
  isBusy: false,
  // While the agent acts on the page, the panel collapses to a strip so the page stays visible.
  isInteractingWithPage: false,
  userExpandedDuringInteraction: false,
  // Transient progress text of the bottom bubble while a turn is in flight.
  statusLabel: '',
  // The header lane keeps the previous label just long enough to animate it out.
  headerStatusItems: [] as { id: number; label: string }[],
});

let historyLoaded = false;
let nextHeaderStatusID = 1;
let interactionRestoreTimer: ReturnType<typeof setTimeout> | undefined;
// Opens the panel the running turn was sent from (header pill or phone layer), so the turn's events
// bring that panel back even after the user closed it.
let showTurnPanel = () => {};

export const chatLog = (level: 'info' | 'warn', message: string, detail?: unknown) => {
  const entry = { at: new Date().toISOString(), level, message, detail };
  // Store the recent trail because production reloads can erase console-only
  // clues before we inspect a stream failure.
  try {
    const previous = JSON.parse(localStorage.getItem(CHAT_LOG_KEY) || '[]');
    const next = Array.isArray(previous) ? [...previous, entry].slice(-CHAT_LOG_LIMIT) : [entry];
    localStorage.setItem(CHAT_LOG_KEY, JSON.stringify(next));
  } catch {
    // Logging must never affect the chat widget.
  }
  const logger = level === 'warn' ? console.warn : console.info;
  logger(`[AgentChat] ${message}`, detail || '');
};

export const ensureAgentHistoryLoaded = async () => {
  if (historyLoaded) { return; }
  historyLoaded = true;
  const rows = await loadAgentChatHistory(getAgentTabID());
  if (rows.length > 0) { agentChat.messages = rows; }
};

const pushHeaderStatus = (label: string) => {
  if (!label) { return; }
  agentChat.headerStatusItems = [...agentChat.headerStatusItems, { id: nextHeaderStatusID++, label }].slice(-2);
};

const clearTurnState = () => {
  agentChat.isBusy = false;
  agentChat.statusLabel = '';
  agentChat.headerStatusItems = [];
};

const addChatRow = async (row: AgentChatRow) => {
  const id = await appendAgentChatMessage(row);
  if (id) { row.id = id; }
  agentChat.messages = [...agentChat.messages, row];
};

// A reply ends the turn's user row as pending.
const clearPendingUserRow = async () => {
  const lastUserRow = [...agentChat.messages].reverse().find((row) => row.role === AGENT_ROLE_USER && row.pending);
  if (!lastUserRow?.id) { return; }
  await updateAgentChatMessage(lastUserRow.id, { pending: false });
  lastUserRow.pending = false;
};

const finishPageInteraction = () => {
  if (interactionRestoreTimer) { clearTimeout(interactionRestoreTimer); }
  interactionRestoreTimer = undefined;
  agentChat.isInteractingWithPage = false;
  agentChat.userExpandedDuringInteraction = false;
};

const beginPageInteraction = (label: string) => {
  if (interactionRestoreTimer) { clearTimeout(interactionRestoreTimer); }
  interactionRestoreTimer = undefined;
  showTurnPanel();
  agentChat.isInteractingWithPage = true;
  agentChat.userExpandedDuringInteraction = false;
  chatLog('info', 'interaction panel collapsed', { label });
};

// A completed action or turn always makes the conversation visible again, also when navigation
// closed or remounted the widget while a tool was running.
const restoreExpandedPanel = (reason: string) => {
  showTurnPanel();
  if (interactionRestoreTimer) {
    chatLog('info', 'interaction panel restore already scheduled', { reason });
    return;
  }
  const completeRestore = () => {
    interactionRestoreTimer = undefined;
    agentChat.isInteractingWithPage = false;
    agentChat.userExpandedDuringInteraction = false;
    void ensureAgentHistoryLoaded();
    chatLog('info', 'interaction panel restored', { reason, delayMs: POST_ACTION_RESTORE_DELAY_MS });
  };
  if (!agentChat.isInteractingWithPage) {
    completeRestore();
    return;
  }
  // Keep the driven page visible briefly after the action completes; later
  // reply/turnEnd events reuse this timer instead of extending the delay.
  interactionRestoreTimer = setTimeout(completeRestore, POST_ACTION_RESTORE_DELAY_MS);
  chatLog('info', 'interaction panel restore scheduled', { reason, delayMs: POST_ACTION_RESTORE_DELAY_MS });
};

// A deliberate input interaction temporarily takes priority over the compact strip so the user can
// inspect or continue the conversation.
export const expandAgentPanelForUserInteraction = () => {
  if (!agentChat.isInteractingWithPage || agentChat.userExpandedDuringInteraction) { return; }
  agentChat.userExpandedDuringInteraction = true;
  chatLog('info', 'interaction panel expanded by user');
};

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

// handleChatEvent processes one agent event pushed down the shared SSE stream. The page-driving
// bridge in sse.ts owns the connection; the chat only consumes chat-typed events.
const handleChatEvent = async (env: ChatStreamEvent) => {
  chatLog('info', 'chat message received', { type: env.Type });
  switch (env.Type) {
    case CHAT_TYPE_AGENT_REPLY: {
      restoreExpandedPanel('agent reply');
      pushHeaderStatus('Respondiendo...');
      await wait(350);
      const payload = env.Payload as AgentReplyPayload;
      await addChatRow({
        tabID: getAgentTabID(),
        role: AGENT_ROLE_AGENT,
        message: payload.Message,
        summary: payload.Summary,
        timestamp: payload.Timestamp || Date.now(),
        tables: payload.Tables,
        charts: payload.Charts,
        pause: payload.Pause,
      });
      await clearPendingUserRow();
      clearTurnState();
      break;
    }
    case CHAT_TYPE_AGENT_SECTIONS: {
      restoreExpandedPanel('agent sections');
      // Page-builder loop returned edited sections. Hand them to the page's
      // applier (parses HTML back into the builder AST); the rest mirrors a
      // normal agentReply so the chat shows the agent's message + clears busy.
      const payload = env.Payload as AgentSectionsPayload;
      try {
        agentModes.applySections(payload);
      } catch (applyError) {
        chatLog('warn', 'applySections failed', { error: String(applyError) });
      }
      await addChatRow({
        tabID: getAgentTabID(),
        role: AGENT_ROLE_AGENT,
        message: payload.Message,
        summary: payload.Summary,
        timestamp: payload.Timestamp || Date.now(),
      });
      await clearPendingUserRow();
      clearTurnState();
      break;
    }
    case CHAT_TYPE_AGENT_ERROR: {
      restoreExpandedPanel('agent error');
      const payload = env.Payload as AgentErrorPayload;
      chatLog('warn', 'agent error received', { message: payload?.Message });
      await addChatRow({
        tabID: getAgentTabID(),
        role: AGENT_ROLE_AGENT,
        message: `⚠ ${payload.Message}`,
        timestamp: Date.now(),
      });
      clearTurnState();
      break;
    }
    case CHAT_TYPE_AGENT_STATUS: {
      // Three flavors of status:
      //   - "acting" → a concrete action (Haciendo click…, Navegando…).
      //     Persisted as an inline gray trace row so the user can later
      //     review every step the agent took for a turn.
      //   - "thinking" → transient "Pensando…" between iterations.
      //     Rendered only as the bottom bubble while busy; not persisted
      //     (would just clutter history with duplicates).
      //   - "error" → a recoverable tool failure. Persisted in red so a
      //     later retry or reply cannot hide what went wrong.
      const payload = env.Payload as AgentStatusPayload;
      const label = payload?.Label || '';
      chatLog('info', 'agent status received', { state: payload?.State, label, step: payload?.Step, maxSteps: payload?.MaxSteps });
      if (payload?.State === 'acting' && label) {
        beginPageInteraction(label);
        pushHeaderStatus(label);
        agentChat.statusLabel = ''; // action is shown in-line; transient bubble idle
        await addChatRow({
          tabID: getAgentTabID(),
          role: AGENT_ROLE_STATUS,
          message: label,
          statusKind: 'action',
          timestamp: Date.now(),
        });
      } else if (payload?.State === 'error' && label) {
        restoreExpandedPanel('agent tool error');
        chatLog('warn', 'agent tool failed', { label, step: payload?.Step });
        agentChat.statusLabel = '';
        await addChatRow({
          tabID: getAgentTabID(),
          role: AGENT_ROLE_STATUS,
          message: label,
          statusKind: 'error',
          timestamp: Date.now(),
        });
      } else {
        restoreExpandedPanel(`agent status: ${payload?.State || 'unknown'}`);
        agentChat.statusLabel = label;
      }
      break;
    }
    case CHAT_TYPE_TURN_END: {
      restoreExpandedPanel('turn end');
      clearTurnState();
      break;
    }
  }
};

// Subscribed once for the page's life: the events must land even while no chat widget is mounted.
if (browser) {
  subscribeAgentChat((env) => { void handleChatEvent(env); });
  (window as any).__agentChat = Object.assign((window as any).__agentChat || {}, {
    debugLog: () => JSON.parse(localStorage.getItem(CHAT_LOG_KEY) || '[]'),
  });
}

/** Sends one turn. showPanel opens the panel it was sent from; resume answers a paused turn. */
export const sendAgentTurn = async (
  text: string,
  showPanel: () => void,
  resume?: { CheckPointID: string, InterruptID: string, Answer: string },
) => {
  if (!text || agentChat.isBusy) { return; }
  showTurnPanel = showPanel;
  const tab = getAgentTabID();
  const optimistic: AgentChatRow = {
    tabID: tab,
    role: AGENT_ROLE_USER,
    message: text,
    timestamp: Date.now(),
    pending: true,
  };
  await addChatRow(optimistic);
  // The $state proxy of the row, so clearing pending below re-renders it.
  const userRow = agentChat.messages.at(-1)!;
  agentChat.isBusy = true;
  finishPageInteraction();
  pushHeaderStatus('Pensando...');

  // Send only compact surface metadata. Full builder HTML is requested by
  // getAgentContext after the backend classifier validates the required scope.
  const surface = agentModes.getSurface();
  chatLog('info', 'sending user message', { tab, bytes: text.length, modelHash: getSelectedAgentModelHash(), modeID: agentModes.active.ID, surface: surface.kind });
  try {
    // Resolves when the backend finishes the turn. The reply itself already
    // arrived through handleChatEvent — this is just the lifetime.
    await runAgentTurn(text, getSelectedAgentModelHash(), optimistic.timestamp ?? Date.now(), agentModes.active.ID, surface, Core.languaje === 2 ? 'en' : 'es', resume);
  } catch (turnError) {
    chatLog('warn', 'turn failed', { error: String(turnError) });
    agentChat.messages = [...agentChat.messages, {
      tabID: tab,
      role: AGENT_ROLE_AGENT,
      message: turnError instanceof AgentTurnRejectedError
        ? `⚠ ${turnError.message}`
        : '⚠ No se pudo conectar con el agente.',
      timestamp: Date.now(),
    }];
  } finally {
    // Backstop: agentReply/agentError normally clear these. A stream that
    // dies mid-turn produces neither, and the widget must not stay wedged.
    clearTurnState();
    restoreExpandedPanel('turn request finished');
    if (userRow.pending) {
      if (userRow.id) { await updateAgentChatMessage(userRow.id, { pending: false }); }
      userRow.pending = false;
    }
  }
};

// answerAgentPause resumes the paused turn of the last reply with a choice, or continues a turn the
// clock paused. The label is the user's message, the choice's ID the answer the backend checks.
export const answerAgentPause = (pause: AgentTurnPause, answerID: string, label: string, showPanel: () => void) =>
  sendAgentTurn(label, showPanel, { CheckPointID: pause.CheckPointID, InterruptID: pause.InterruptID || '', Answer: answerID });
