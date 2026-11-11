<script lang="ts">
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import type { IClientWeekBar } from '#domain/client-week-sales.ts'

  // One bar per week, the oldest first. Blue is the paid part, red the unpaid part above it.
  const { bars }: { bars: IClientWeekBar[] } = $props()

  const BAR_AREA_HEIGHT = 26
  // The client's own biggest week fills the bar area, so each row reads as the client's trend.
  const tallestPurchases = $derived(Math.max(1, ...bars.map((bar) => bar.purchases)))

  function barTitle(bar: IClientWeekBar): string {
    const weekLabel = `${tr('Week|Semana')} ${bar.week % 100} (20${Math.floor(bar.week / 100)})`
    return `${weekLabel} · ${tr('Purchases|Compras')} S/ ${formatN(bar.purchases / 100, 2)}`
      + ` · ${tr('Unpaid|Pendiente')} S/ ${formatN(bar.unpaid / 100, 2)}`
  }
</script>

<div class="flex h-28 items-end gap-1" aria-label="Purchases of the client in the last weeks, unpaid part in red">
  {#each bars as bar (bar.week)}
    {@const barHeight = bar.purchases > 0 ? Math.max(2, Math.round((bar.purchases / tallestPurchases) * BAR_AREA_HEIGHT)) : 1}
    {@const unpaidHeight = bar.purchases > 0 ? Math.round((bar.unpaid / bar.purchases) * barHeight) : 0}
    <div class="flex w-4 flex-col" style="height: {barHeight}px" title={barTitle(bar)}>
      {#if bar.purchases > 0}
        <div class="w-full bg-red-500" style="height: {unpaidHeight}px"></div>
        <div class="w-full grow bg-blue-500"></div>
      {:else}
        <div class="w-full grow bg-gray-300"></div>
      {/if}
    </div>
  {/each}
</div>
