<script lang="ts">
  // In-app agent chat widget. Lives in the top header as a pill-shaped 2-row
  // textarea; clicking/focusing opens a floating panel below that shows the
  // message history. The textarea stays in the header at the top of the
  // visual chat layout; the floating panel only renders the history list.
  //
  // WS connection (`/ws/agent-chat`) is opened lazily on first user
  // interaction so the cost is paid only by users who actually engage. The
  // page-driving WS (`/ws/agent` in ws.ts) is a separate connection that
  // shares the same TabID — that's how the backend will (in step 5) route
  // tool calls to the same tab the chat is mounted in.
  //
  // History persistence is local-only via Dexie (chat_history.idb.ts). The
  // backend persists its own copy in ScyllaDB; this cache exists so the
  // widget can re-render past messages instantly on open.

  import { onDestroy, tick } from 'svelte';
  import { Env } from '$core/env';
  import { agentWsBase, getAgentTabID } from './ws';
  import { getSelectedAgentModelHash } from './models.svelte';
  import {
    AGENT_ROLE_AGENT,
    AGENT_ROLE_STATUS,
    AGENT_ROLE_USER,
    appendAgentChatMessage,
    loadAgentChatHistory,
    updateAgentChatMessage,
    type AgentChatRow,
  } from './chat_history.idb';

  // --- Wire types (mirror backend/agent/chat_ws.go) ---------------------------

  interface ChatEnvelope {
    Type: string;
    Payload?: unknown;
  }

  interface AgentReplyPayload {
    Message: string;
    Summary: string;
    Timestamp: number;
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

  const CHAT_TYPE_USER_MESSAGE = 'userMessage';
  const CHAT_TYPE_AGENT_REPLY = 'agentReply';
  const CHAT_TYPE_AGENT_ERROR = 'agentError';
  const CHAT_TYPE_AGENT_STATUS = 'agentStatus';
  const CHAT_LOG_KEY = '__agent_chat_debug_log';
  const CHAT_LOG_LIMIT = 120;

  // --- State ------------------------------------------------------------------

  let isOpen = $state(false);
  let inputText = $state('');
  let messages = $state<AgentChatRow[]>([]);
  let socket = $state<WebSocket | null>(null);
  let wsReady = $state(false);
  let isBusy = $state(false);
  let statusLabel = $state(''); // transient progress text shown while a turn is in flight
  let headerStatusItems = $state<{ id: number; label: string }[]>([]);
  let hostElement: HTMLElement | undefined = $state();
  let textareaElement: HTMLTextAreaElement | undefined = $state();
  let scrollElement: HTMLDivElement | undefined = $state();
  let historyLoaded = false;
  let nextHeaderStatusID = 1;

  // --- Helpers ----------------------------------------------------------------

  const chatLog = (level: 'info' | 'warn', message: string, detail?: unknown) => {
    const entry = { at: new Date().toISOString(), level, message, detail };
    // Store the recent trail because production reloads can erase console-only
    // clues before we inspect a websocket failure.
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

  const readUserID = (): number => {
    if (typeof window === 'undefined') { return 0; }
    try {
      const raw = localStorage.getItem(Env.appId + 'UserInfo');
      if (!raw) { return 0; }
      const info = JSON.parse(raw) as { ID?: number };
      return Number(info?.ID) || 0;
    } catch {
      return 0;
    }
  };

  const scrollToBottom = async () => {
    await tick();
    if (scrollElement) {
      scrollElement.scrollTop = scrollElement.scrollHeight;
    }
  };

  const pushHeaderStatus = (label: string) => {
    if (!label) { return; }
    // Keep the previous trace only long enough to animate it out when the next
    // status takes the fixed header slot.
    headerStatusItems = [...headerStatusItems, { id: nextHeaderStatusID++, label }].slice(-2);
  };

  const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

  const ensureHistoryLoaded = async () => {
    if (historyLoaded) { return; }
    historyLoaded = true;
    const rows = await loadAgentChatHistory(getAgentTabID());
    if (rows.length > 0) {
      messages = rows;
      await scrollToBottom();
    }
  };

  const ensureWsOpen = () => {
    if (socket && socket.readyState <= WebSocket.OPEN) { return; }
    const tab = getAgentTabID();
    const company = Env.getEmpresaID() || 0;
    const user = readUserID();
    // Seed the backend with the current SPA path so the first get_page
    // status label can name the route. Subsequent navigations the agent
    // performs update this server-side.
    const path = encodeURIComponent(window.location.pathname || '');
    const url = `${agentWsBase()}/ws/agent-chat?tab=${encodeURIComponent(tab)}&company=${company}&user=${user}&path=${path}`;

    chatLog('info', 'connecting chat ws', { url, tab, company, user, apiMain: Env.API_ROUTES.MAIN });
    const ws = new WebSocket(url);
    socket = ws;
    wsReady = false;

    ws.addEventListener('open', () => {
      if (socket !== ws) { return; }
      wsReady = true;
      chatLog('info', 'chat ws open', { url, tab });
    });
    ws.addEventListener('message', (event) => {
      void handleWsMessage(event.data);
    });
    ws.addEventListener('close', () => {
      if (socket !== ws) { return; }
      socket = null;
      wsReady = false;
      isBusy = false;
      chatLog('warn', 'chat ws closed', { url, tab });
    });
    ws.addEventListener('error', (err) => {
      chatLog('warn', 'chat ws error', { url, error: String(err) });
      try { ws.close(); } catch { /* ignore */ }
    });
  };

  const handleWsMessage = async (raw: unknown) => {
    let env: ChatEnvelope;
    try {
      env = JSON.parse(typeof raw === 'string' ? raw : '');
    } catch {
      chatLog('warn', 'bad chat message json', { rawType: typeof raw });
      return;
    }
    chatLog('info', 'chat message received', { type: env.Type });
    switch (env.Type) {
      case CHAT_TYPE_AGENT_REPLY: {
        pushHeaderStatus('Respondiendo...');
        await wait(350);
        const payload = env.Payload as AgentReplyPayload;
        const row: AgentChatRow = {
          tabID: getAgentTabID(),
          role: AGENT_ROLE_AGENT,
          message: payload.Message,
          summary: payload.Summary,
          timestamp: payload.Timestamp || Date.now(),
        };
        const id = await appendAgentChatMessage(row);
        if (id) { row.id = id; }
        // Clear the pending flag on the most recent user row.
        const lastUser = [...messages].reverse().find((m) => m.role === AGENT_ROLE_USER && m.pending);
        if (lastUser?.id) {
          await updateAgentChatMessage(lastUser.id, { pending: false });
          lastUser.pending = false;
        }
        messages = [...messages, row];
        isBusy = false;
        statusLabel = '';
        headerStatusItems = [];
        await scrollToBottom();
        break;
      }
      case CHAT_TYPE_AGENT_ERROR: {
        const payload = env.Payload as AgentErrorPayload;
        chatLog('warn', 'agent error received', { message: payload?.Message });
        const row: AgentChatRow = {
          tabID: getAgentTabID(),
          role: AGENT_ROLE_AGENT,
          message: `⚠ ${payload.Message}`,
          timestamp: Date.now(),
        };
        const id = await appendAgentChatMessage(row);
        if (id) { row.id = id; }
        messages = [...messages, row];
        isBusy = false;
        statusLabel = '';
        headerStatusItems = [];
        await scrollToBottom();
        break;
      }
      case CHAT_TYPE_AGENT_STATUS: {
        // Two flavors of status:
        //   - "acting" → a concrete action (Haciendo click…, Navegando…).
        //     Persisted as an inline gray trace row so the user can later
        //     review every step the agent took for a turn.
        //   - "thinking" → transient "Pensando…" between iterations.
        //     Rendered only as the bottom bubble while busy; not persisted
        //     (would just clutter history with duplicates).
        const payload = env.Payload as AgentStatusPayload;
        const label = payload?.Label || '';
        chatLog('info', 'agent status received', { state: payload?.State, label, step: payload?.Step, maxSteps: payload?.MaxSteps });
        if (payload?.State === 'acting' && label) {
          pushHeaderStatus(label);
          const row: AgentChatRow = {
            tabID: getAgentTabID(),
            role: AGENT_ROLE_STATUS,
            message: label,
            timestamp: Date.now(),
          };
          const id = await appendAgentChatMessage(row);
          if (id) { row.id = id; }
          messages = [...messages, row];
          statusLabel = ''; // action is shown in-line; transient bubble idle
        } else {
          statusLabel = label;
        }
        await scrollToBottom();
        break;
      }
    }
  };

  const sendMessage = async () => {
    const text = inputText.trim();
    if (!text || isBusy) { return; }
    ensureWsOpen();
    // The user can hit send before the socket finishes its open handshake;
    // queue the optimistic row anyway and wait briefly for `wsReady`.
    const tab = getAgentTabID();
    const optimistic: AgentChatRow = {
      tabID: tab,
      role: AGENT_ROLE_USER,
      message: text,
      timestamp: Date.now(),
      pending: true,
    };
    const id = await appendAgentChatMessage(optimistic);
    if (id) { optimistic.id = id; }
    messages = [...messages, optimistic];
    inputText = '';
    isBusy = true;
    pushHeaderStatus('Pensando...');
    await scrollToBottom();

    const waitOpen = async () => {
      const start = Date.now();
      while (!wsReady && Date.now() - start < 5_000) {
        await new Promise((r) => setTimeout(r, 50));
      }
    };
    await waitOpen();
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      chatLog('warn', 'send failed: chat ws not open', { readyState: socket?.readyState });
      isBusy = false;
      headerStatusItems = [];
      if (optimistic.id) { await updateAgentChatMessage(optimistic.id, { pending: false }); }
      optimistic.pending = false;
      messages = [...messages, {
        tabID: tab,
        role: AGENT_ROLE_AGENT,
        message: '⚠ No se pudo conectar con el agente.',
        timestamp: Date.now(),
      }];
      return;
    }
    chatLog('info', 'sending user message', { tab, bytes: text.length, modelHash: getSelectedAgentModelHash() });
    socket.send(JSON.stringify({
      Type: CHAT_TYPE_USER_MESSAGE,
      Payload: {
        Message: text,
        ModelHash: getSelectedAgentModelHash(),
        Timestamp: optimistic.timestamp,
      },
    } satisfies ChatEnvelope));
  };

  // --- Open / close lifecycle -------------------------------------------------

  const openPanel = () => {
    if (isOpen) { return; }
    isOpen = true;
    void ensureHistoryLoaded();
    ensureWsOpen();
    void scrollToBottom();
  };

  const closePanel = () => {
    isOpen = false;
  };

  const handleDocumentClick = (event: MouseEvent) => {
    if (!isOpen || !hostElement) { return; }
    if (!(event.target instanceof Node)) { return; }
    if (!hostElement.contains(event.target)) { closePanel(); }
  };

  const handleKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && isOpen) {
      closePanel();
      textareaElement?.blur();
    }
  };

  $effect(() => {
    if (typeof window === 'undefined') { return; }
    document.addEventListener('mousedown', handleDocumentClick);
    document.addEventListener('keydown', handleKeydown);
    return () => {
      document.removeEventListener('mousedown', handleDocumentClick);
      document.removeEventListener('keydown', handleKeydown);
    };
  });

  onDestroy(() => {
    if (socket) {
      try { socket.close(); } catch { /* ignore */ }
      socket = null;
    }
  });

  const onTextareaKey = (event: KeyboardEvent) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      void sendMessage();
    }
  };

  if (typeof window !== 'undefined') {
    (window as any).__agentChat = Object.assign((window as any).__agentChat || {}, {
      debugLog: () => JSON.parse(localStorage.getItem(CHAT_LOG_KEY) || '[]'),
    });
  }
</script>

<div bind:this={hostElement} data-agent-hidden="true" class="_host relative w-full max-w-[36rem]">
  <div class="_pill flex items-start gap-6 px-10 py-6 bg-white/15 hover:bg-white/20 focus-within:bg-white/25
    border border-white/20 rounded-2xl transition-colors cursor-text"
    onclick={() => textareaElement?.focus()}
    onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { textareaElement?.focus(); } }}
    role="button"
    tabindex="-1"
  >
    <span class="icon-message text-white/80 text-base mt-2"></span>
    <textarea bind:this={textareaElement} bind:value={inputText}
      rows={2}
      placeholder="Pregúntale a Genix…"
      class="_input p-4 flex-1 bg-transparent text-white placeholder-white/60 outline-none resize-none text-sm leading-tight"
      class:_inputStatus={isBusy && headerStatusItems.length > 0}
      onfocus={openPanel}
      onkeydown={onTextareaKey}
    ></textarea>
    {#if isBusy && headerStatusItems.length > 0}
      <div class="_statusLane" aria-live="polite">
        {#each headerStatusItems as item, index (item.id)}
          <div class="_statusItem text-xs italic leading-tight"
            class:_statusLeaving={index < headerStatusItems.length - 1}
          >
            {item.label}
          </div>
        {/each}
      </div>
    {/if}
    <button type="button"
      class="_send shrink-0 self-stretch px-10 rounded-xl bg-white/15 hover:bg-white/25 disabled:opacity-40
        text-white text-sm font-medium transition-colors"
      onclick={() => void sendMessage()}
      disabled={isBusy || !inputText.trim()}
      aria-label="Enviar mensaje"
    >
      {#if isBusy}
        <span class="_spinner"></span>
      {:else}
        ➤
      {/if}
    </button>
  </div>

  {#if isOpen}
    <div class="_panel absolute left-0 right-0 top-full mt-2 z-300
      bg-white rounded-2xl shadow-2xl border border-gray-200 overflow-hidden"
    >
      <div bind:this={scrollElement}
        class="_scroll max-h-[60vh] min-h-[260px] overflow-y-auto px-12 py-10 flex flex-col gap-8"
      >
        {#if messages.length === 0}
          <div class="text-center text-gray-400 text-sm py-20">
            Empieza una conversación con el agente.
          </div>
        {:else}
          {#each messages as msg (msg.id ?? msg.timestamp)}
            {#if msg.role === AGENT_ROLE_STATUS}
              <div class="_status px-4 text-xs text-gray-400 italic leading-snug">
                · {msg.message}
              </div>
            {:else}
              {@const isUser = msg.role === AGENT_ROLE_USER}
              <div class="_row flex {isUser ? 'justify-end' : 'justify-start'}">
                <div class="_bubble max-w-[80%] px-10 py-8 rounded-2xl text-sm leading-snug whitespace-pre-wrap
                  {isUser
                    ? 'bg-indigo-600 text-white rounded-br-md'
                    : 'bg-gray-100 text-gray-800 rounded-bl-md'}"
                >
                  {msg.message}{#if msg.pending}<span class="opacity-60 ml-4">…</span>{/if}
                </div>
              </div>
            {/if}
          {/each}
        {/if}
        {#if isBusy}
          <div class="_row flex justify-start">
            <div class="_bubble px-10 py-8 rounded-2xl text-sm bg-gray-100 text-gray-500 italic flex items-center gap-6">
              <span class="_dot inline-block w-6 h-6 rounded-full bg-indigo-400 animate-pulse"></span>
              {statusLabel || 'Pensando…'}
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  ._host {
    color-scheme: light;
  }
  ._pill {
	  background-color: #00000030;
	  padding: 0;
	  border: 0;
	  height: calc(var(--header-height) - 9px);
	  box-shadow: rgb(255 255 255 / 8%) 0px 1px 7px 0px, rgb(255 255 255 / 14%) 0px 1px 14px 0px;
		margin-bottom: 1px;
  }
  ._input {
    /* Prevent the browser-supplied min-height from forcing the pill taller
       than rows=2 needs. */
    min-height: 0;
    line-height: 1.3;
    padding-right: 54px;
  }
  ._inputStatus {
    padding-right: 360px;
  }
  ._statusLane {
    position: absolute;
    right: 44px;
    top: 0;
    bottom: 0;
    width: 340px;
    overflow: hidden;
    pointer-events: none;
    mask-image: linear-gradient(to bottom, transparent 0%, #000 28%, #000 72%, transparent 100%);
  }
  ._statusItem {
    position: absolute;
    left: 0;
    right: 0;
    top: 50%;
    color: rgb(255 255 255 / 70%);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: right;
    animation: status-enter 240ms ease-out both;
  }
  ._statusLeaving {
    animation: status-leave 520ms ease-out both;
  }
  ._send {
	  position: absolute;
	  right: 4px;
	  height: calc(100% - 8px);
	  top: 3px;
	  width: 32px;
	  border-radius: 6px 14px 14px 6px;
	  font-size: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  ._spinner {
    width: 14px;
    height: 14px;
    border: 2px solid rgb(255 255 255 / 30%);
    border-top-color: rgb(255 255 255 / 90%);
    border-radius: 999px;
    animation: spin 700ms linear infinite;
  }
  @keyframes status-enter {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
    to {
      opacity: 1;
      transform: translateY(-50%);
    }
  }
  @keyframes status-leave {
    from {
      opacity: 0.75;
      transform: translateY(-50%);
    }
    to {
      opacity: 0;
      transform: translateY(calc(-50% - 18px));
    }
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
  @media (max-width: 900px) {
    ._inputStatus {
      padding-right: 54px;
    }
    ._statusLane {
      display: none;
    }
  }
</style>
