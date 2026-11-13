<script lang="ts">
// Always-present header button + layer: the global notifications/process accumulator.
// The circular trigger grows an animated rotating border while any process is in
// progress, and a badge shows how many. Clicking it lists every row (in-progress
// processes pinned to the top); sendUserNotification pops it open with only the
// recent rows and a link to the rest. Its Information tab shows the open route's
// GUIDE.md. On phones it also holds the agent chat tab.
import { untrack } from 'svelte'
import { MediaQuery } from 'svelte/reactivity'
import { page } from '$app/state'
import ButtonLayer from '#components/buttons/ButtonLayer.svelte'
import MarkdownView from '#components/markdown/MarkdownView.svelte'
import { routeGuide, showRouteGuide } from '#components/markdown/route-guide.svelte.ts'
import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
import AgentChat from '#core/agent/AgentChat.svelte'
import { tr } from '#core/store.svelte.ts'
import { formatTime } from '#libs/helpers.ts'
import { security } from '#libs/ui-runtime.svelte.ts'
import {
  notifications, notificationsLayer, hydrateNotifications, inProgressProcessCount,
  NOTIFICATIONS_TAB_MESSAGES, NOTIFICATIONS_TAB_INFORMATION, NOTIFICATIONS_TAB_AGENT,
} from '#core/notifications.svelte.ts'
import { sortNotificationRows, selectVisibleNotifications, notificationAppearance } from '#core/notifications.ts'

// Load persisted rows once so the list survives reloads.
hydrateNotifications()

// Below Tailwind's md (749px) the header hides its agent pill, so the agent moves into this layer, first.
const isMobileViewport = new MediaQuery('max-width: 748px')
const isAgentTabOffered = $derived(isMobileViewport.current && security.isLogged())
const INFORMATION_ICON = 'icon-[fa--info-circle]'
const tabOptions = $derived([
  ...(isAgentTabOffered ? [{ id: NOTIFICATIONS_TAB_AGENT, name: 'Agent|Agente', icon: 'icon-[mdi--robot]' }] : []),
  // Short name on a phone, where three tabs with icons share the layer's width.
  { id: NOTIFICATIONS_TAB_INFORMATION, name: isMobileViewport.current ? 'Info|Info' : 'Information|Información', icon: INFORMATION_ICON },
  { id: NOTIFICATIONS_TAB_MESSAGES, name: 'Messages|Mensajes', icon: 'icon-[mdi--message-text]' },
])
// The Agent tab only exists on phones: left selected there and resized to desktop, the layer falls
// back to the Information tab.
const shownTab = $derived(notificationsLayer.selectedTab === NOTIFICATIONS_TAB_AGENT && !isAgentTabOffered
  ? NOTIFICATIONS_TAB_INFORMATION : notificationsLayer.selectedTab)
// The header trigger wears the icon of the last selected tab.
const triggerIcon = $derived(tabOptions.find((tabOption) => tabOption.id === shownTab)?.icon || INFORMATION_ICON)
const isAgentTabShown = $derived(shownTab === NOTIFICATIONS_TAB_AGENT)

// The agent's events reopen the layer on its tab, also after the user closed it mid-turn.
const showAgentTab = () => {
  notificationsLayer.selectedTab = NOTIFICATIONS_TAB_AGENT
  notificationsLayer.isOpen = true
}

// Closing through isOpen skips the layer's onClose, so its popup window is reset here too.
const closeLayer = () => {
  notificationsLayer.isOpen = false
  notificationsLayer.recentSinceTime = 0
}

// The Information tab shows the open route's user guide (GUIDE.md), each file its own lazy chunk.
const guideFiles = import.meta.glob<string>('../routes/**/GUIDE.md', { query: '?raw', import: 'default' })
$effect(() => {
  const routeId = page.route.id || ''
  untrack(() => { void showRouteGuide(guideFiles, routeId) })
})

// Entering a route with a guide flashes the trigger's border once, a quiet hint that there is something to read.
const GUIDE_PULSE_MS = 2000
let isGuidePulsing = $state(false)
$effect(() => {
  if (!routeGuide.markdown) { isGuidePulsing = false; return }
  isGuidePulsing = true
  const pulseTimer = setTimeout(() => { isGuidePulsing = false }, GUIDE_PULSE_MS)
  return () => { clearTimeout(pulseTimer) }
})

// Count of running processes — drives the spinning border + badge.
const activeCount = $derived(inProgressProcessCount())

const selection = $derived(
  selectVisibleNotifications(sortNotificationRows([...notifications.values()]), notificationsLayer.recentSinceTime)
)

// Every shade of a card (background, border, icon, title) derives from this accent.
const accentByColor = {
  blue: 'var(--blue-solid)', green: 'var(--green-solid)', yellow: 'var(--yellow-solid)', red: 'var(--red-solid)',
}

const iconClassByName = {
  info: 'icon-[fa--info-circle]',
  success: 'icon-[fa--check-circle]',
  warning: 'icon-[fa--exclamation-triangle]',
  error: 'icon-[fa--times-circle]',
}
</script>

<ButtonLayer layerClass="md:w-680 nf-layer"
  contentCss="p-8 max-h-[70vh] flex flex-col"
  label="Opens the notifications and running processes panel."
  bind:isOpen={notificationsLayer.isOpen}
  onClose={() => { notificationsLayer.recentSinceTime = 0 }}
>
  {#snippet button(isOpen)}
    <!-- Circular trigger matching the gear/reload siblings; ring spins while active. -->
    <span class="nf-btn flex items-center justify-center" class:nf-active={activeCount > 0}
      class:nf-guide-pulse={isGuidePulsing && activeCount === 0}>
      <i class="{triggerIcon} text-white text-lg max-md:text-[24px]"></i>
      {#if activeCount > 0}
        <span class="nf-badge">{activeCount}</span>
      {/if}
    </span>
  {/snippet}

  <!-- On a phone the tabs share the full width; on desktop they keep their natural width. -->
  <OptionsStrip css="mb-8 max-md:w-full" buttonCss="min-w-156 max-md:min-w-86 max-md:flex-1"
    options={tabOptions} keyId="id" keyName="name" keyIcon="icon"
    selected={shownTab}
    onSelect={(tabOption) => { notificationsLayer.selectedTab = tabOption.id }}
  />

  <!-- Only the tab's content scrolls: the tabs stay pinned at the top of the layer. -->
  <div class="min-h-0 overflow-y-auto">
  {#if isAgentTabShown}
    <AgentChat inLayer onShowLayer={showAgentTab} onCloseLayer={closeLayer} />
  {:else if shownTab === NOTIFICATIONS_TAB_INFORMATION}
    <MarkdownView css="pl-8 pr-6 pb-4" content={routeGuide.markdown}
      emptyText={tr('This page has no guide yet.|Esta página aún no tiene guía.')} />
  {:else if selection.visibleRows.length === 0}
    <div class="px-12 py-16 text-fg-muted h5 text-center">
      {tr('No notifications|Sin notificaciones')}
    </div>
  {:else}
    {#each selection.visibleRows as row (row.id)}
      {@const appearance = notificationAppearance(row)}
      <div class="nf-card flex items-start gap-8 px-10 py-8" style:--nf-accent={accentByColor[appearance.color]}>
        <div class="nf-icon flex items-center justify-center">
          {#if appearance.icon === 'spinner'}
            <span class="nf-spinner"></span>
          {:else}
            <i class={iconClassByName[appearance.icon]}></i>
          {/if}
        </div>
        <div class="flex-1 min-w-0">
          <div class="flex items-start justify-between gap-8">
            <div class="nf-title h5 ff-bold min-w-0 break-words">{row.title}</div>
            <div class="text-sm ff-mono text-fg-muted shrink-0">{formatTime(row.createdAt, 'M-d h:n')}</div>
          </div>
          {#if row.subtitle}
            <div class="text-sm ff-bold text-fg-muted">{row.subtitle}</div>
          {/if}
          {#if row.message}
            <div class="text-sm nf-text whitespace-pre-line">{row.message}</div>
          {/if}
        </div>
      </div>
    {/each}
  {/if}

  {#if shownTab === NOTIFICATIONS_TAB_MESSAGES && selection.hiddenCount > 0}
    <button type="button" class="nf-more w-full py-6 mt-4 text-center"
      onclick={() => { notificationsLayer.recentSinceTime = 0 }}
    >
      {selection.hiddenCount === 1
        ? tr('See 1 more message...|Ver 1 mensaje más...')
        : tr(`See ${selection.hiddenCount} more messages...|Ver ${selection.hiddenCount} mensajes más...`)}
    </button>
  {/if}
  </div>
</ButtonLayer>

<style>
  /* Circular button, same footprint as the header's gear/reload buttons. */
  .nf-btn {
    position: relative;
    width: 40px;
    height: 40px;
    border-radius: 9999px;
    background-color: rgba(255, 255, 255, 0.1);
    transition: background-color 0.15s;
  }
  .nf-btn:hover {
    background-color: rgba(255, 255, 255, 0.2);
  }

  /* Rotating gradient ring on the button edge while a process runs. The conic
     gradient is masked to a thin ring so only the border appears to spin. */
  .nf-active::before {
    content: '';
    position: absolute;
    inset: -2px;
    border-radius: 9999px;
    background: conic-gradient(from 0deg, transparent 0deg, #ffffff 90deg, transparent 200deg);
    -webkit-mask: radial-gradient(farthest-side, transparent calc(100% - 3px), #000 calc(100% - 3px));
    mask: radial-gradient(farthest-side, transparent calc(100% - 3px), #000 calc(100% - 3px));
    animation: nf-ring-spin 1s linear infinite;
  }

  @keyframes nf-ring-spin {
    to { transform: rotate(360deg); }
  }

  /* The same masked ring, a soft light-blue arc running the border twice when a route with a guide
     opens. A wide arc: a narrow one is barely visible while it spins. */
  .nf-guide-pulse::before {
    content: '';
    position: absolute;
    inset: -2px;
    border-radius: 9999px;
    pointer-events: none;
    background: conic-gradient(from 0deg, transparent 0deg, transparent 200deg, rgb(167 224 255 / 0.5) 285deg, #e2f5ff 345deg, transparent 360deg);
    -webkit-mask: radial-gradient(farthest-side, transparent calc(100% - 3px), #000 calc(100% - 3px));
    mask: radial-gradient(farthest-side, transparent calc(100% - 3px), #000 calc(100% - 3px));
    animation: nf-ring-spin 1s linear 2;
  }
  /* Reduced motion: the ring just lights up and fades. */
  @media (prefers-reduced-motion: reduce) {
    .nf-guide-pulse::before {
      background: #e2f5ff;
      animation: nf-guide-fade 2s ease-out 1;
    }
    @keyframes nf-guide-fade {
      0%, 70% { opacity: 1; }
      100% { opacity: 0; }
    }
  }

  @media only screen and (max-width: 749px) {
    .nf-btn {
      width: 44px;
      height: 44px;
    }
    /* Full-width over the page on a phone (it holds the agent chat): a halo sets it apart from the
       content behind. The element selector outranks ButtonLayer's own scoped shadow. */
    :global(div.button-layer.nf-layer) {
      outline: 4px solid #00000030;
      box-shadow: rgb(60 64 67 / 30%) 0 1px 2px 0, rgb(60 64 67 / 15%) 0 1px 4px 1px;
    }
  }

  /* In-progress count badge. */
  .nf-badge {
    position: absolute;
    top: -3px;
    right: -3px;
    min-width: 16px;
    height: 16px;
    padding: 0 4px;
    border-radius: 9999px;
    background-color: var(--red-solid);
    color: white;
    font-family: 'bold';
    line-height: 16px;
    text-align: center;
  }

  /* Card tinted by its color scheme (--nf-accent, set inline per row). */
  .nf-card {
    background-color: color-mix(in oklab, var(--nf-accent) 8%, var(--surface));
    border: 1px solid color-mix(in oklab, var(--nf-accent) 30%, var(--surface));
    border-radius: 8px;
    margin-bottom: 6px;
  }
  .nf-card:last-of-type {
    margin-bottom: 0;
  }
  .nf-icon, .nf-title {
    color: var(--nf-accent);
  }

  .nf-icon {
    width: 20px;
    height: 20px;
    margin-top: 2px;
    flex-shrink: 0;
  }

  .nf-text {
    line-height: 1.3;
    word-break: break-word;
  }

  .nf-more {
    color: var(--accent-fg);
    border-radius: 6px;
    cursor: pointer;
  }
  .nf-more:hover {
    background-color: var(--accent-bg);
  }

  /* Per-row spinner for in-progress processes. */
  .nf-spinner {
    width: 16px;
    height: 16px;
    border: 2px solid color-mix(in srgb, currentColor 30%, transparent);
    border-top-color: currentColor;
    border-radius: 50%;
    animation: nf-ring-spin 0.8s linear infinite;
  }
</style>
