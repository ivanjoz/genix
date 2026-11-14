<script lang="ts">
  import Button from '#components/buttons/Button.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatQuantity, unpackQuantityLine } from '#core/quantity.ts'
  import { postTableOrderLines } from '../dining.svelte.ts'
  import { LINE_STATUS, waitMinutes, type IDiningLayout, type ITableOrder } from '../dining.ts'
  import { readyToPickupGroups, type IReadyItem } from './floor-service.ts'

  // "Listos para recoger" (docs/DINING_PLAN.md §5): the dishes waiting at each station's pickup
  // point, the longest-waiting first. One tap marks a dish served.
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

{#if groups.length}
  <div class="flex flex-col gap-6 p-8 rounded-md bg-teal-bg" aria-label="Dishes ready to pick up, by station">
    <div class="ff-bold text-teal-fg">{tr('Ready to pick up|Listos para recoger')}</div>
    {#each groups as group (group.stationID)}
      <div class="flex flex-wrap items-center gap-6">
        <span class="ff-bold w-100">{group.stationName}</span>
        {#each group.items as item (`${item.orderID}-${item.line.LineID}`)}
          <div class="flex items-center gap-6 px-8 py-2 rounded-md bg-surface">
            <span class="ff-bold">{item.tableLabel}</span>
            <span>{formatQuantity(unpackQuantityLine(item.line.Quantity), item.line.SubDivisor || 1)} {productName(item.line.ProductID)}</span>
            <span class="text-fg-muted">{waitMinutes(nowTime - (item.line.ReadyAt || nowTime))} min</span>
            <Button name="Served|Servido" label="Marks this dish served." height="compact" color="green"
              disabled={pendingKey === `${item.orderID}-${item.line.LineID}`} onClick={() => markServed(item)} />
          </div>
        {/each}
      </div>
    {/each}
  </div>
{/if}
