<script lang="ts">
  // In-app agent chat widget, in two places that share one conversation (agentChat in
  // agent-chat.svelte.ts):
  //   - Desktop: the top header's pill-shaped 2-row textarea; clicking/focusing it opens a
  //     floating panel below with the message history.
  //   - Phones (inLayer): the Info layer's Agent tab renders the input and the history inline —
  //     the header has no room for the pill there.
  // While the input holds a short query the panel also shows menu shortcuts.

  import { onMount, tick, untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { highlString } from '@genix/ui/utilities';
  import DOMPurify from 'dompurify';
  import { Marked } from 'marked';
  import { ensureAgentStream } from './sse';
  import { AGENT_ROLE_STATUS, AGENT_ROLE_USER } from './chat_history.idb';
  import {
    agentChat, answerAgentPause, chatLog, ensureAgentHistoryLoaded, expandAgentPanelForUserInteraction, sendAgentTurn,
  } from './agent-chat.svelte';
  import AgentAnswer from './AgentAnswer.svelte';
  import { agentModes } from './agent.svelte';
  import { highlightWords, searchMenuOptions, type MenuSearchOption } from './menu_search';
  import { Core, tr } from '#core/store.svelte.ts';

  const { inLayer = false, onShowLayer, onCloseLayer }: {
    /** Renders inside the phone Info layer, which owns the open state, instead of the header. */
    inLayer?: boolean;
    onShowLayer?: () => void;
    onCloseLayer?: () => void;
  } = $props();

  const INTERNAL_ROUTE_PATTERN = /^\/[A-Za-z0-9_-]+(?:\/[A-Za-z0-9_-]+)*\/?$/;
  const INTERNAL_ROUTE_START_PATTERN = /(?<![:/A-Za-z0-9_-])\/[A-Za-z0-9_-]+(?:\/[A-Za-z0-9_-]+)*\/?/;
  const escapeHTML = (value: string): string => value.replace(/[&<>"']/g, (character) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  })[character] || character);
  const renderInternalRoute = (route: string, asCode = false): string => {
    const safeRoute = escapeHTML(route);
    return `<a href="${safeRoute}">${asCode ? `<code>${safeRoute}</code>` : safeRoute}</a>`;
  };
  const markdownRenderer = new Marked({
    renderer: {
      // Routes commonly arrive wrapped in backticks; preserve the code style
      // while turning only strict application paths into links.
      codespan(token) {
        const route = token.text.trim();
        return INTERNAL_ROUTE_PATTERN.test(route) ? renderInternalRoute(route, true) : false;
      },
    },
    extensions: [{
      name: 'internalRoute',
      level: 'inline',
      start(source) { return source.search(INTERNAL_ROUTE_START_PATTERN); },
      tokenizer(source) {
        const match = INTERNAL_ROUTE_START_PATTERN.exec(source);
        if (!match || match.index !== 0) { return; }
        return { type: 'internalRoute', raw: match[0], route: match[0] };
      },
      renderer(token) { return renderInternalRoute(String(token.route || '')); },
    }],
  });
  const AGENT_MARKDOWN_TAGS = ['p', 'br', 'strong', 'em', 'ul', 'ol', 'li', 'code', 'pre', 'a'];
  const AGENT_MARKDOWN_ATTRIBUTES = ['href', 'target', 'rel'];

  // --- State ------------------------------------------------------------------

  // The header panel's own open state; inLayer the panel is open whenever it is mounted.
  let isHeaderPanelOpen = $state(false);
  let inputText = $state('');
  let hostElement: HTMLElement | undefined = $state();
  let textareaElement: HTMLTextAreaElement | undefined = $state();
  let scrollElement: HTMLDivElement | undefined = $state();
  let menuGridHeight = $state(0);

  const isOpen = $derived(inLayer || isHeaderPanelOpen);
  // Menu shortcuts shown above the history while the input holds a short query.
  const menuMatches = $derived(searchMenuOptions(inputText));
  const menuHighlights = $derived(highlightWords(inputText));
  const isInteractionPanelCollapsed = $derived(
    isOpen && agentChat.isInteractingWithPage && !agentChat.userExpandedDuringInteraction,
  );
  const isShowingHeaderStatus = $derived(agentChat.isBusy && agentChat.headerStatusItems.length > 0);
  // On a phone width is scarce: bubbles keep only a small offset to tell the speakers apart.
  const bubbleMaxWidthClass = $derived(inLayer ? 'max-w-[92%]' : 'max-w-[80%]');

  // --- Helpers ----------------------------------------------------------------

  const renderAgentMarkdown = (markdown: string): string => {
    if (typeof window === 'undefined') { return ''; }
    // Marked accepts raw HTML, so sanitize its final output immediately before
    // Svelte inserts it with {@html}; the allowlist keeps chat markup minimal.
    const renderedHTML = markdownRenderer.parse(markdown, { async: false, breaks: true });
    return DOMPurify.sanitize(renderedHTML, {
      ALLOWED_TAGS: AGENT_MARKDOWN_TAGS,
      ALLOWED_ATTR: AGENT_MARKDOWN_ATTRIBUTES,
    });
  };

  // Keeps the newest row in view whenever the conversation grows, the panel opens, or it comes back
  // from the collapsed strip.
  $effect(() => {
    void [agentChat.messages.length, agentChat.statusLabel, agentChat.isBusy, isInteractionPanelCollapsed, isOpen, scrollElement];
    untrack(async () => {
      await tick();
      if (scrollElement) { scrollElement.scrollTop = scrollElement.scrollHeight; }
    });
  });

  // --- Open / close lifecycle -------------------------------------------------

  // Warm up the stream while the user types: the first turn would open it
  // anyway, and doing it here hides the connect latency.
  const warmUpAgent = () => {
    void ensureAgentStream().catch((streamError) => {
      chatLog('warn', 'stream warmup failed', { error: String(streamError) });
    });
    void ensureAgentHistoryLoaded();
  };

  // The phone layer mounts this only while its Agent tab shows.
  onMount(() => { if (inLayer) { warmUpAgent(); } });

  const openPanel = () => {
    if (inLayer) { onShowLayer?.(); return; }
    if (isHeaderPanelOpen) { return; }
    isHeaderPanelOpen = true;
    warmUpAgent();
  };

  const closePanel = () => {
    if (inLayer) { onCloseLayer?.(); } else { isHeaderPanelOpen = false; }
  };

  const sendMessage = () => {
    const text = inputText.trim();
    if (!text || agentChat.isBusy) { return; }
    inputText = '';
    void sendAgentTurn(text, openPanel);
  };

  const openMenuOption = (option: MenuSearchOption) => {
    inputText = '';
    closePanel();
    textareaElement?.blur();
    void goto(option.route);
  };

  // The phone layer closes itself on an outside click.
  const handleDocumentClick = (event: MouseEvent) => {
    if (inLayer || !isOpen || !hostElement) { return; }
    if (!(event.target instanceof Node)) { return; }
    if (!hostElement.contains(event.target)) { closePanel(); }
  };

  const handleRouteLinkClick = (event: MouseEvent) => {
    if (!isOpen || !hostElement || !(event.target instanceof Element)) { return; }
    const anchor = event.target.closest<HTMLAnchorElement>('a[href]');
    const route = anchor?.getAttribute('href') || '';
    if (!anchor || !hostElement.contains(anchor) || !INTERNAL_ROUTE_PATTERN.test(route)) { return; }
    // Let the anchor/SvelteKit perform navigation, then remove the chat overlay
    // so the selected destination is immediately visible.
    closePanel();
    textareaElement?.blur();
    chatLog('info', 'agent route link closed panel', { route });
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
    document.addEventListener('click', handleRouteLinkClick);
    document.addEventListener('keydown', handleKeydown);
    return () => {
      document.removeEventListener('mousedown', handleDocumentClick);
      document.removeEventListener('click', handleRouteLinkClick);
      document.removeEventListener('keydown', handleKeydown);
    };
  });

  const onTextareaKey = (event: KeyboardEvent) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      sendMessage();
    }
  };
</script>

{#snippet conversationPanel()}
  {#if isInteractionPanelCollapsed}
    <!-- Keep the driven page visible while retaining a clear activity cue. -->
    <div class="min-h-30 flex items-center px-12 text-sm text-red-fg" role="status" aria-live="polite">
      Interactuando con la pantalla...
    </div>
  {:else}
    {#if agentModes.all.length > 1}
      <!-- Mode bar — only when the page registered extra modes. Uses the same
           indigo as the header/textarea so it reads as a strip between the
           pill above and the chat below. Active mode is opaque + tinted;
           the rest are dimmed. -->
      <div class="_modeBar flex items-center gap-8 px-12 py-8" class:_modeBarLayer={inLayer}>
        {#each agentModes.all as mode (mode.ID)}
          {@const isActive = mode.ID === agentModes.activeID}
          <button type="button"
            class="_modeBtn flex items-center gap-4 px-10 py-4 text-xs text-fg"
            class:_modeActive={isActive}
            onclick={() => agentModes.select(mode.ID)}
          >
            <span class="{mode.Icon} text-[13px]"></span>
            {tr(mode.Name, Core.languaje)}
          </button>
        {/each}
      </div>
    {/if}
    <div class="relative">
      {#if menuMatches.length > 0}
        <!-- Overlays the top of the history instead of pushing it down, so typing
             a query never shifts or resizes the panel. -->
        <div bind:offsetHeight={menuGridHeight}
          class="_menuGrid absolute top-0 left-0 right-0 z-10 grid {inLayer ? 'grid-cols-2' : 'grid-cols-3'} gap-8 px-12 py-10 bg-surface"
        >
          {#each menuMatches as option (option.route)}
            <button type="button" class="_menuCard flex items-center gap-6 px-6 py-6 text-left"
              onclick={() => openMenuOption(option)}
            >
              <span class="_menuIcon shrink-0 flex ml-2 text-[16px] items-center justify-center text-accent-fg">
                <span class={option.icon}></span>
              </span>
              <span class="_menuName text-xs leading-tight text-fg-soft">
                {#each highlString(tr(option.name, Core.languaje), menuHighlights) as w}
                  <span class={w.highl ? '_highl' : ''}>{w.text}</span>
                {/each}
              </span>
            </button>
          {/each}
        </div>
      {/if}
      <!-- The panel grows only when the history area is shorter than the overlay. The phone layer
           already pads its content: the conversation adds no side gap of its own there. -->
      <div bind:this={scrollElement}
        class="overflow-y-auto flex flex-col gap-8 {inLayer ? '_scrollLayer px-2 pt-24 pb-10' : 'max-h-[60vh] px-12 py-10'}"
        style:min-height="max({inLayer ? 200 : 260}px, {menuMatches.length > 0 ? menuGridHeight : 0}px)"
      >
        {#if agentChat.messages.length === 0}
          <div class="text-center text-fg-subtle text-sm py-20">
            Empieza una conversación con el agente.
          </div>
        {:else}
          {#each agentChat.messages as msg (msg.id ?? msg.timestamp)}
            {#if msg.role === AGENT_ROLE_STATUS}
              <div class="_status px-4 text-xs leading-snug
                {msg.statusKind === 'error' ? 'text-red-fg not-italic' : 'text-fg-subtle italic'}">
                {msg.statusKind === 'error' ? '⚠' : '·'} {msg.message}
              </div>
            {:else}
              {@const isUser = msg.role === AGENT_ROLE_USER}
              <div class="_row flex {isUser ? 'justify-end' : 'justify-start'}">
                <!-- A reply with charts or tables is wider than a text bubble: they need the width more than text does. -->
                <div class="_bubble px-10 py-8 rounded-2xl text-sm leading-snug
                  {msg.tables?.length || msg.charts?.length ? 'w-[calc(94%-24px)]' : bubbleMaxWidthClass}
                  {isUser
                    ? 'bg-accent-solid text-white rounded-br-md'
                    : '_markdown bg-surface-muted text-fg rounded-bl-md'}"
                >
                  {#if isUser}
                    <span class="whitespace-pre-wrap">{msg.message}</span>
                  {:else}
                    {@html renderAgentMarkdown(msg.message)}
                  {/if}
                  {#if msg.pending}<span class="opacity-60 ml-4">…</span>{/if}
                  {#if msg.tables?.length || msg.charts?.length}
                    <AgentAnswer tables={msg.tables} charts={msg.charts} />
                  {/if}
                </div>
              </div>
              <!-- Only the last reply's question can still be answered: a later message started another turn. -->
              {#if msg.pause && msg === agentChat.messages[agentChat.messages.length - 1] && !agentChat.isBusy}
                {@const pause = msg.pause}
                <div class="flex flex-wrap gap-6 pl-4">
                  {#each pause.Choices?.length ? pause.Choices : [{ ID: '', Label: Core.languaje === 2 ? 'Continue' : 'Continuar' }] as choice (choice.ID)}
                    <button type="button" class="_choice px-10 py-4 rounded-full text-sm border border-accent-solid text-fg"
                      onclick={() => void answerAgentPause(pause, choice.ID, choice.Label, openPanel)}
                    >{choice.Label}</button>
                  {/each}
                </div>
              {/if}
            {/if}
          {/each}
        {/if}
        {#if agentChat.isBusy}
          <div class="_row flex justify-start">
            <div class="_bubble px-10 py-8 rounded-2xl text-sm bg-surface-muted text-fg-muted italic flex items-center gap-6">
              <span class="_dot inline-block w-6 h-6 rounded-full bg-accent-solid animate-pulse"></span>
              {agentChat.statusLabel || 'Pensando…'}
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}
{/snippet}

{#if inLayer}
  <div bind:this={hostElement} data-agent-hidden="true" class="_layerHost">
    <!-- The header pill's shape on the white layer: a light inset with the accent border. -->
    <div class="_layerInput relative flex items-start rounded-xl">
      <textarea bind:this={textareaElement} bind:value={inputText}
        rows={2}
        aria-label="Pregúntale a Genix"
        placeholder={tr(agentModes.active.Placeholder, Core.languaje)}
        class="_input p-6 pl-12 flex-1 bg-transparent outline-none resize-none text-[15px]"
        onfocus={expandAgentPanelForUserInteraction}
        oninput={expandAgentPanelForUserInteraction}
        onkeydown={onTextareaKey}
      ></textarea>
      <button type="button"
        class="_send _sendLayer shrink-0 bg-accent-solid hover:bg-primary disabled:opacity-40 text-white transition-colors"
        onclick={sendMessage}
        disabled={agentChat.isBusy || !inputText.trim()}
        aria-label="Enviar mensaje"
      >
        {#if agentChat.isBusy}<span class="_spinner"></span>{:else}➤{/if}
      </button>
    </div>
    {@render conversationPanel()}
  </div>
{:else}
<!-- Desktop only: on a phone the agent is the Info layer's Agent tab (NotificationsButton). -->
<div bind:this={hostElement} data-agent-hidden="true" class="_host relative hidden md:block w-full max-w-[36rem]">
  <div class="_pill absolute top-0 h-full w-full flex items-start gap-6 px-10 py-6 bg-white/15 hover:bg-white/20 focus-within:bg-white/25 border border-white/20 rounded-2xl transition-colors cursor-text"
    onclick={() => {
      expandAgentPanelForUserInteraction();
      textareaElement?.focus();
    }}
    class:h-full={!isOpen}
    class:rounded-b-none={isOpen}
    class:h-44={isOpen}
    onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { textareaElement?.focus(); } }}
    role="button"
    tabindex="-1"
  >
    {#if !inputText}
      <!-- Placeholder follows the active agent mode (search / build page / …). -->
      <div class="_placeholder absolute left-12 top-16 -translate-y-1/2 flex items-center gap-6 text-white/60 text-sm pointer-events-none">
        <span class="{agentModes.active.Icon} text-[15px]"></span>
        {tr(agentModes.active.Placeholder, Core.languaje)}
      </div>
    {/if}
    <textarea bind:this={textareaElement} bind:value={inputText}
      rows={2}
      aria-label="Pregúntale a Genix"
      class="_input p-4 pl-12 flex-1 bg-transparent text-white outline-none resize-none text-sm leading-tight"
      class:_inputStatus={isShowingHeaderStatus}
      onfocus={() => { openPanel(); expandAgentPanelForUserInteraction(); }}
      oninput={expandAgentPanelForUserInteraction}
      onkeydown={onTextareaKey}
    ></textarea>
    {#if isShowingHeaderStatus}
      <div class="_statusLane" aria-live="polite">
        {#each agentChat.headerStatusItems as item, index (item.id)}
          <div class="_statusItem text-xs italic leading-tight"
            class:_statusLeaving={index < agentChat.headerStatusItems.length - 1}
          >
            {item.label}
          </div>
        {/each}
      </div>
    {/if}
    <button type="button"
      class="_send shrink-0 self-stretch px-10 rounded-xl bg-white/15 hover:bg-white/25 disabled:opacity-40
        text-white text-sm font-medium transition-colors"
      onclick={sendMessage}
      disabled={agentChat.isBusy || !inputText.trim()}
      aria-label="Enviar mensaje"
    >
      {#if agentChat.isBusy}
        <span class="_spinner"></span>
      {:else}
        ➤
      {/if}
    </button>
  </div>

  {#if isOpen}
    <div class="_panel absolute left-0 right-0 top-[calc(100%+4px)] z-300
      bg-surface rounded-b-2xl rounded-t-none shadow-2xl border border-line overflow-hidden"
      class:_panelCollapsed={isInteractionPanelCollapsed}
    >
      {@render conversationPanel()}
    </div>
  {/if}
</div>
{/if}

<style>
  ._host {
    color-scheme: light;
    height: calc(var(--header-height) - 10px);
  }
  ._pill {
	  background-color: #00000030;
	  padding: 0;
	  border: 0;
		box-shadow: rgb(255 255 255 / 8%) 0px 1px 4px 0px, rgb(255 255 255 / 12%) 0px 1px 10px 0px;
		margin-bottom: 1px;
  }
  ._input {
    /* Prevent the browser-supplied min-height from forcing the pill taller
       than rows=2 needs. */
    min-height: 0;
    line-height: 1.3;
    padding-right: 54px;
  }
  ._placeholder {
    /* The icon and copy are one synthetic placeholder so they move together. */
    display: flex;
    align-items: center;
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
  /* Phone layer: the input's own accent inset and a send button that fills its height. */
  ._layerInput {
    border: 1px solid var(--accent-solid);
    background-color: var(--surface-soft);
  }
  ._sendLayer {
    top: 4px;
    bottom: 4px;
    height: auto;
    width: 36px;
    border-radius: 6px 9px 9px 6px;
  }
  /* Fits under the Info layer's tabs and input, inside the layer's 70vh content. The messages fade
     out as they scroll up to the input: a mask, so it fades into whatever the layer's background is. */
  ._scrollLayer {
    max-height: calc(70vh - 120px);
    mask-image: linear-gradient(to bottom, transparent 0, transparent 6px, #000 28px);
  }
  ._spinner {
    width: 14px;
    height: 14px;
    border: 2px solid rgb(255 255 255 / 30%);
    border-top-color: rgb(255 255 255 / 90%);
    border-radius: 999px;
    animation: spin 700ms linear infinite;
  }
  ._menuGrid {
    border-bottom: 1px solid var(--line);
    box-shadow: 0 6px 12px -6px rgb(15 23 42 / 18%);
  }
  ._menuCard {
    border: 1px solid color-mix(in oklab, var(--accent-border) 50%, var(--surface));
    border-radius: 8px;
    background: var(--accent-bg);
    min-height: 42px;
    transition: background-color 0.15s ease, border-color 0.15s ease;
  }
  ._menuCard:hover {
    background: var(--accent-bg-strong);
    border-color: var(--accent-border);
  }
  /* Fixed box so every card's text starts at the same offset — the icon glyphs
     have different intrinsic widths. The mask lives on the inner span so it
     keeps its own aspect ratio instead of stretching to the box. */
  ._menuIcon {
    width: 16px;
  }
  ._menuName {
    /* Long names wrap to a second line, then clip. */
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  ._highl {
    color: var(--red-fg);
    text-decoration: underline;
  }
  ._panel {
  	border: 1px solid var(--accent-solid);
  	/* No top border so the mode bar sits flush against the search pill, with no
  	   light line between them. */
  	border-top: 0;
  }
  ._panelCollapsed {
    background: color-mix(in oklab, var(--surface) 82%, transparent);
    border-color: color-mix(in oklab, var(--red-solid) 45%, transparent);
    backdrop-filter: blur(8px);
    box-shadow: 0 8px 24px rgb(15 23 42 / 12%);
  }
  ._markdown :global(p) {
    margin: 0;
  }
  ._markdown :global(p + p),
  ._markdown :global(p + ul),
  ._markdown :global(p + ol),
  ._markdown :global(ul + p),
  ._markdown :global(ol + p),
  ._markdown :global(pre + p) {
    margin-top: 8px;
  }
  ._markdown :global(ul),
  ._markdown :global(ol) {
    margin: 8px 0;
    padding-left: 18px;
  }
  ._markdown :global(ul) {
    list-style: disc;
  }
  ._markdown :global(ol) {
    list-style: decimal;
  }
  ._markdown :global(li + li) {
    margin-top: 2px;
  }
  ._markdown :global(code) {
    padding: 1px 4px;
    border-radius: 4px;
    background: color-mix(in oklab, var(--fg) 8%, transparent);
  }
  ._markdown :global(pre) {
    margin: 8px 0;
    padding: 8px;
    overflow-wrap: anywhere;
    white-space: pre-wrap;
    border-radius: 6px;
    background: color-mix(in oklab, var(--fg) 8%, transparent);
  }
  ._markdown :global(pre code) {
    padding: 0;
    background: transparent;
  }
  ._markdown :global(a) {
    color: var(--accent-fg);
    text-decoration: underline;
  }
  /* Mode bar: a light lavender strip between the search pill and the white chat. */
  ._modeBar {
  	background: var(--accent-bg-strong);
  }
  /* In the phone layer it sits under the input, with a gap and rounded like it. */
  ._modeBarLayer {
    margin-top: 8px;
    border-radius: 10px;
  }
  /* Mode bar buttons: dimmed when inactive; the active one is an opaque tab with
     a purple underline (margin-bottom pulls it onto the bar's bottom edge). */
  ._modeBtn {
    opacity: 0.55;
    line-height: 1;
  }
  ._modeBtn:hover {
    opacity: 0.85;
  }
  ._modeActive {
    opacity: 1;
    border-bottom: 2px solid var(--accent-border);
    border-radius: 0;
    padding-bottom: 6px;
    margin-bottom: -2px;
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
  ._choice:hover {
    background-color: var(--accent-solid);
    color: white;
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
