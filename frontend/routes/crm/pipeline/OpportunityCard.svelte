<script lang="ts">
  import Avatar from '#components/misc/Avatar.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import type { ICrmOpportunity } from '#services/crm/crm-opportunity.svelte.ts'

  // One card of the board. Everything it shows is already resolved by the page.
  let { opportunity, clientName, ownerName, isStale, previousLabel, onOpen }: {
    opportunity: ICrmOpportunity
    clientName: string
    ownerName: string
    isStale: boolean
    /** "<pipeline> · <stage>" the card came from; empty when it never moved. */
    previousLabel: string
    onOpen: () => void
  } = $props()
</script>

<button type="button" class="w-full text-left p-8 flex flex-col gap-2 cursor-pointer rounded-md text-fg hover:bg-surface-muted" onclick={onOpen}
  aria-label="Open opportunity {opportunity.Title}">
  <div class="ff-bold">{opportunity.Title}</div>
  <div class="text-fg-soft">{clientName}</div>
  <div class="flex items-center justify-between gap-6">
    <span class="ff-mono">{formatN(opportunity.Amount / 100, 2)}</span>
    <span class="text-fg-muted">{opportunity.Probability}%</span>
  </div>
  <div class="flex items-center justify-between gap-6 text-fg-muted">
    {#if opportunity.OwnerUserID}
      <Avatar name={ownerName} colorSeed={opportunity.OwnerUserID} />
    {:else}<span></span>{/if}
    {#if isStale}<span class="text-red-fg">{tr('Stale|Estancada')}</span>{/if}
  </div>
  {#if previousLabel}
    <div class="text-fg-muted">{tr('From|Desde')} {previousLabel}</div>
  {/if}
</button>
