<script lang="ts">
// Always-present header button + layer: the global notifications/process accumulator.
// The circular trigger grows an animated rotating border while any process is in
// progress, and a badge shows how many. Clicking it lists every row (in-progress
// processes pinned to the top); sendUserNotification pops it open with only the
// recent rows and a link to the rest.
import ButtonLayer from '#components/buttons/ButtonLayer.svelte'
import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
import { tr } from '#core/store.svelte.ts'
import { formatTime } from '#libs/helpers.ts'
import {
  notifications, notificationsLayer, hydrateNotifications, inProgressProcessCount,
  NOTIFICATIONS_TAB_MESSAGES, NOTIFICATIONS_TAB_INFORMATION,
} from '#core/notifications.svelte.ts'
import { sortNotificationRows, selectVisibleNotifications, notificationAppearance } from '#core/notifications.ts'

// Load persisted rows once so the list survives reloads.
hydrateNotifications()

// Count of running processes — drives the spinning border + badge.
const activeCount = $derived(inProgressProcessCount())

const selection = $derived(
  selectVisibleNotifications(sortNotificationRows([...notifications.values()]), notificationsLayer.recentSinceTime)
)

// Every shade of a card (background, border, icon, title) derives from this accent.
const accentByColor = {
  blue: '#1f6fd6', green: '#1f9a4f', yellow: '#c18a06', red: '#d64545',
}

const iconClassByName = {
  info: 'icon-[fa--info-circle]',
  success: 'icon-[fa--check-circle]',
  warning: 'icon-[fa--exclamation-triangle]',
  error: 'icon-[fa--times-circle]',
}
</script>

<ButtonLayer layerClass="md:w-560"
  contentCss="p-8 max-h-[70vh] overflow-y-auto"
  label="Opens the notifications and running processes panel."
  bind:isOpen={notificationsLayer.isOpen}
  onClose={() => { notificationsLayer.recentSinceTime = 0 }}
>
  {#snippet button(isOpen)}
    <!-- Circular trigger matching the gear/reload siblings; ring spins while active. -->
    <span class="nf-btn flex items-center justify-center" class:nf-active={activeCount > 0}>
      <i class="icon-[fa--info-circle] text-white text-lg"></i>
      {#if activeCount > 0}
        <span class="nf-badge">{activeCount}</span>
      {/if}
    </span>
  {/snippet}

  <OptionsStrip css="mb-8"
    options={[[NOTIFICATIONS_TAB_MESSAGES, 'Messages|Mensajes'], [NOTIFICATIONS_TAB_INFORMATION, 'Information|Información']]}
    selected={notificationsLayer.selectedTab}
    onSelect={(opt) => { notificationsLayer.selectedTab = opt[0] as number }}
  />

  {#if notificationsLayer.selectedTab === NOTIFICATIONS_TAB_INFORMATION}
    <div class="px-12 py-16 c-gray h5 text-center">
      {tr('No information|Sin información')}
    </div>
  {:else if selection.visibleRows.length === 0}
    <div class="px-12 py-16 c-gray h5 text-center">
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
            <div class="text-sm ff-mono c-gray shrink-0">{formatTime(row.createdAt, 'M-d h:n')}</div>
          </div>
          {#if row.subtitle}
            <div class="text-sm ff-bold c-gray">{row.subtitle}</div>
          {/if}
          {#if row.message}
            <div class="text-sm nf-text whitespace-pre-line">{row.message}</div>
          {/if}
        </div>
      </div>
    {/each}
  {/if}

  {#if notificationsLayer.selectedTab === NOTIFICATIONS_TAB_MESSAGES && selection.hiddenCount > 0}
    <button type="button" class="nf-more w-full py-6 mt-4 text-center"
      onclick={() => { notificationsLayer.recentSinceTime = 0 }}
    >
      {selection.hiddenCount === 1
        ? tr('See 1 more message...|Ver 1 mensaje más...')
        : tr(`See ${selection.hiddenCount} more messages...|Ver ${selection.hiddenCount} mensajes más...`)}
    </button>
  {/if}
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

  /* In-progress count badge. */
  .nf-badge {
    position: absolute;
    top: -3px;
    right: -3px;
    min-width: 16px;
    height: 16px;
    padding: 0 4px;
    border-radius: 9999px;
    background-color: #e75c5c;
    color: white;
    font-family: 'bold';
    line-height: 16px;
    text-align: center;
  }

  /* Card tinted by its color scheme (--nf-accent, set inline per row). */
  .nf-card {
    background-color: color-mix(in srgb, var(--nf-accent) 7%, white);
    border: 1px solid color-mix(in srgb, var(--nf-accent) 28%, white);
    border-radius: 8px;
    margin-bottom: 6px;
  }
  .nf-card:last-of-type {
    margin-bottom: 0;
  }
  :global(.dark) .nf-card {
    background-color: color-mix(in srgb, var(--nf-accent) 14%, transparent);
    border-color: color-mix(in srgb, var(--nf-accent) 40%, transparent);
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
    color: #4343ad;
    border-radius: 6px;
    cursor: pointer;
  }
  .nf-more:hover {
    background-color: rgba(67, 67, 173, 0.08);
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
