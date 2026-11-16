<script lang="ts">
  import Button from '#components/buttons/Button.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatQuantity, unpackQuantityLine } from '#core/quantity.ts'
  import { postTableOrderLines } from '../dining.svelte.ts'
  import { LINE_STATUS, waitMinutes, type IDiningLayout, type ITableOrder } from '../dining.ts'
  import { readyToPickupGroups, type IReadyItem } from './floor-service.ts'

  // "Por entregar" (docs/DINING_PLAN.md §5): the dishes the kitchen marked ready, waiting at each
  // station's pickup point to be carried to their table, the longest-waiting first. One tap marks a
  // dish served.
  let { layout, orders, nowTime, productName, onOrderChanged }: {
    layout: IDiningLayout
    orders: ITableOrder[]
    nowTime: number
    productName: (productID: number) => string
    onOrderChanged: (order: ITableOrder) => void
  } = $props()

  const groups = $derived(readyToPickupGroups(orders, layout))
  let pendingKey = $state('')

  async function markServed(item: IReadyItem) {
    const itemKey = `${item.orderID}-${item.line.LineID}`
    if (pendingKey) return
    pendingKey = itemKey
    try {
      onOrderChanged(await postTableOrderLines({ OrderID: item.orderID, LineIDs: [item.line.LineID], Status: LINE_STATUS.SERVED }))
    } catch (servedError) {
      console.error('[floor-service] served error', { orderID: item.orderID, lineID: item.line.LineID, servedError })
    } finally {
      pendingKey = ''
    }
  }
</script>

<div class="flex flex-col gap-12" aria-label="Dishes ready to deliver, by station">
  {#each groups as group (group.stationID)}
    <div class="flex flex-col rounded-md border border-line bg-surface">
      <div class="px-12 py-8 ff-bold text-teal-fg bg-teal-bg rounded-t-md">{group.stationName}</div>
      {#each group.items as item (`${item.orderID}-${item.line.LineID}`)}
        <div class="flex flex-wrap items-center gap-10 px-12 py-6 border-t border-line">
          <span class="ff-bold w-60">{item.tableLabel}</span>
          <span class="ff-mono w-40 text-right">{formatQuantity(unpackQuantityLine(item.line.Quantity), item.line.SubDivisor || 1)}</span>
          <div class="flex-1 min-w-160">
            {productName(item.line.ProductID)}
            {#if item.line.Note}<span class="text-fg-muted"> · {item.line.Note}</span>{/if}
          </div>
          <span class="text-fg-muted">{waitMinutes(nowTime - (item.line.ReadyAt || nowTime))} min</span>
          <Button name="Served|Servido" label="Marks this dish served." height="compact" color="green"
            disabled={pendingKey === `${item.orderID}-${item.line.LineID}`} onClick={() => markServed(item)} />
        </div>
      {/each}
    </div>
  {:else}
    <!-- min-h-240 leaves room for the page's Salón dropdown above, which the content layer would clip. -->
    <div class="p-12 min-h-240 rounded-md bg-surface-soft text-fg-muted">{tr('Nothing ready to deliver.|Nada listo para entregar.')}</div>
  {/each}
</div>
