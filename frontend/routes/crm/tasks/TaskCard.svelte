<script lang="ts">
  import Badge from '#components/misc/Badge.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import type { ICrmTask } from '#services/crm/crm-task.svelte.ts'

  // One task: a badge (the stage of an open task, the result of a closed one) and the action, then
  // the client, then the opportunity and its amount (the server attaches all three to the task). The
  // whole card opens the task's side layer. With a seller name (a leader reading the whole team), the
  // badge and the seller share the top line and the action drops to its own line. A closed task shows
  // the time it was closed at the end of the first line.
  let { task, badgeText, badgeCss, sellerName = '', timeLabel = '', isSelected, onOpen }: {
    task: ICrmTask
    badgeText: string
    badgeCss: string
    sellerName?: string
    timeLabel?: string
    isSelected: boolean
    onOpen: () => void
  } = $props()
</script>

{#snippet stageBadge(extraCss = '')}
  {#if badgeText}<Badge text={badgeText} size="small" css="{badgeCss} {extraCss}" />{/if}
{/snippet}

<button type="button" aria-label="Open task"
  class="flex flex-col gap-4 min-w-0 text-left rounded border bg-surface px-12 py-10 cursor-pointer
    hover:border-blue-500 hover:shadow-sm {isSelected ? 'border-blue-500' : 'border-line'}"
  onclick={onOpen}>
  <!-- The action wraps instead of truncating: it is what the user has to do. -->
  {#if sellerName}
    <!-- Pulled up toward the card's top border so the header line reads as a header. -->
    <div class="flex items-center justify-between gap-8 min-w-0 mb-2">
      {@render stageBadge('-mt-2')}
      <!-- At most 24 characters of the seller's name. -->
      <span class="flex items-center gap-6 min-w-0 text-fg-muted -mt-5" title={sellerName}>
        <i class="icon-[fa--briefcase] shrink-0"></i>
        <span class="truncate max-w-[24ch]">{sellerName}</span>
      </span>
    </div>
    <span class="ff-bold leading-tight text-[15px]">{task.Title}</span>
  {:else}
    <div class="flex items-start gap-8 min-w-0">
      {@render stageBadge()}
      <span class="ff-bold leading-tight text-[15px] grow">{task.Title}</span>
      {#if timeLabel}<span class="text-fg-muted shrink-0">{timeLabel}</span>{/if}
    </div>
  {/if}
  <div class="flex items-center gap-6 min-w-0" class:text-fg-muted={!task.ClientName}>
    <i class="icon-[fa--user] shrink-0 text-fg-muted"></i>
    <span class="truncate">{task.ClientName || tr('No client|Sin cliente')}</span>
  </div>
  <!-- The opportunity, with its amount (whole units only) at the right end of the line. -->
  <div class="flex items-center justify-between gap-8 min-w-0">
    <span class="text-fg-muted truncate">{task.OpportunityTitle}</span>
    <span class="ff-mono shrink-0 text-[14px]">{formatN(task.OpportunityAmount / 100, 0)}</span>
  </div>
</button>
