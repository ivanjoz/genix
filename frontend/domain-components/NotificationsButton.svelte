<script lang="ts">
// Always-present header button + layer: the global notifications/process accumulator.
// The circular trigger grows an animated rotating border while any process is in
// progress, and a badge shows how many. The layer lists items newest-first with
// in-progress processes pinned to the top.
import ButtonLayer from '$components/buttons/ButtonLayer.svelte'
import { tr } from '$core/store.svelte'
import { formatTime } from '$libs/helpers'
import {
  notifications, hydrateNotifications, inProgressProcessCount,
} from '$core/notifications.svelte'
import {
  PROCESS_KIND, PROCESS_STATUS_IN_PROGRESS, PROCESS_STATUS_DONE, PROCESS_STATUS_CANCELED,
  NOTIFICATION_TYPE_WARN, NOTIFICATION_TYPE_ERROR, type NotificationRow,
} from '$core/notifications.idb'

// Load persisted rows once so the list survives reloads.
hydrateNotifications()

// Count of running processes — drives the spinning border + badge.
const activeCount = $derived(inProgressProcessCount())

// In-progress processes float to the top, everything else is newest-first.
const items = $derived(
  [...notifications.values()].sort((a, b) => {
    const aActive = a.kind === PROCESS_KIND && a.status === PROCESS_STATUS_IN_PROGRESS ? 0 : 1
    const bActive = b.kind === PROCESS_KIND && b.status === PROCESS_STATUS_IN_PROGRESS ? 0 : 1
    if (aActive !== bActive) return aActive - bActive
    return b.createdAt - a.createdAt
  })
)

// Severity/status accent color per row (left bar + icon tint).
const rowAccent = (row: NotificationRow): string => {
  if (row.kind === PROCESS_KIND) {
    if (row.status === PROCESS_STATUS_CANCELED) return '#e75c5c'
    if (row.status === PROCESS_STATUS_DONE) return '#29b15d'
    return '#1277f5'
  }
  if (row.type === NOTIFICATION_TYPE_ERROR) return '#e75c5c'
  if (row.type === NOTIFICATION_TYPE_WARN) return '#e0a317'
  return '#1277f5'
}
</script>

<ButtonLayer layerClass="md:w-560"
  contentCss="p-8 max-h-[70vh] overflow-y-auto"
  label="Opens the notifications and running processes panel."
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

  {#if items.length === 0}
    <div class="px-12 py-16 c-gray h5 text-center">
      {tr('No notifications|Sin notificaciones')}
    </div>
  {:else}
    {#each items as row (row.id)}
      {@const accent = rowAccent(row)}
      <div class="nf-row flex items-start gap-6 px-10 py-6">
        <div class="nf-icon flex items-center justify-center" style="color: {accent}">
          {#if row.kind === PROCESS_KIND && row.status === PROCESS_STATUS_IN_PROGRESS}
            <span class="nf-spinner"></span>
          {:else if row.kind === PROCESS_KIND && row.status === PROCESS_STATUS_DONE}
            <i class="icon-[fa--check]"></i>
          {:else if row.kind === PROCESS_KIND && row.status === PROCESS_STATUS_CANCELED}
            <i class="icon-[fa--close]"></i>
          {:else}
            <i class="icon-[fa--info-circle]"></i>
          {/if}
        </div>
        <div class="flex-1 min-w-0">
          <div class="flex items-center justify-between gap-8">
            <div class="h5 ff-bold truncate">{row.name}</div>
            <div class="text-xs ff-mono c-gray shrink-0">{formatTime(row.createdAt, 'M-d h:n')}</div>
          </div>
          {#if row.text}
            <div class="fs13 c-gray nf-text">{row.text}</div>
          {/if}
        </div>
      </div>
    {/each}
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

  /* Light gray card per row — no status border, the icon alone carries the state. */
  .nf-row {
    background-color: #f3f4f6;
    border-radius: 8px;
    margin-bottom: 6px;
  }
  .nf-row:last-child {
    margin-bottom: 0;
  }
  :global(.dark) .nf-row {
    background-color: rgba(255, 255, 255, 0.06);
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

  /* Per-row spinner for in-progress processes. */
  .nf-spinner {
    width: 16px;
    height: 16px;
    border: 2px solid rgba(18, 119, 245, 0.3);
    border-top-color: #1277f5;
    border-radius: 50%;
    animation: nf-ring-spin 0.8s linear infinite;
  }
</style>
