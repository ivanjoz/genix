<script lang="ts">
  import { onMount } from 'svelte'
  import Card from '#components/cards/Card.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import OrderLineCards from '../OrderLineCards.svelte'
  import { TABLE_ORDER_STATUS, type ITableOrder } from '../dining.ts'
  import { getTableOrderHistory } from './floor-service.svelte.ts'
  import { orderTotal, tableOrderStatusLabels } from './floor-service.ts'

  // "Historial": the orders the table received today and yesterday, as an accordion. Each row is
  // the time it opened, its status and its amount; tapping it unfolds its rounds, read-only.
  let { layoutID, tableShapeID, productName }: {
    layoutID: number
    tableShapeID: number
    productName: (productID: number) => string
  } = $props()

  let orders = $state<ITableOrder[]>([])
  let isLoading = $state(true)
  let expandedOrderID = $state(0)
  const formatAmount = (cents: number) => `S/ ${formatN(cents / 100, 2)}`

  onMount(async () => {
    try {
      orders = await getTableOrderHistory(layoutID, tableShapeID)
    } catch (historyError) {
      console.error('[floor-service] history error', { layoutID, tableShapeID, historyError })
    } finally {
      isLoading = false
    }
  })

  const STATUS_CSS: Record<number, string> = {
    [TABLE_ORDER_STATUS.ANNULLED]: 'bg-red-bg text-red-fg',
    [TABLE_ORDER_STATUS.OPEN]: 'bg-amber-bg text-amber-fg',
    [TABLE_ORDER_STATUS.CHARGED]: 'bg-green-bg text-green-fg',
  }
</script>

<div class="flex flex-col gap-8" aria-label="The table's orders of today and yesterday, newest first">
  {#each orders as order (order.ID)}
    {@const isExpanded = expandedOrderID === order.ID}
    <div class="rounded-md border border-line bg-surface">
      <Card label="Shows or hides this order's products." onClick={() => { expandedOrderID = isExpanded ? 0 : order.ID }}
        css="flex items-center gap-12 px-10 py-8 cursor-pointer hover:bg-surface-soft">
        <i class="{isExpanded ? 'icon-[fa--chevron-down]' : 'icon-[fa--chevron-right]'} text-fg-muted"></i>
        <span class="ff-mono">{formatTime(order.Created || 0, 'd-M h:n')}</span>
        <span class="px-6 rounded {STATUS_CSS[order.ss]}">{tr(tableOrderStatusLabels[order.ss])}</span>
        <span class="ml-auto ff-bold ff-mono">{formatAmount(orderTotal(order))}</span>
      </Card>
      {#if isExpanded}
        <div class="flex flex-col gap-8 px-10 pb-10" aria-label="The order's details and products">
          <div class="flex flex-wrap gap-12 text-fg-muted">
            {#if order.Guests}<span>{order.Guests} {tr('guests|comensales')}</span>{/if}
            {#if order.SaleOrderID}<span>{tr('Sale|Venta')} {order.SaleOrderID}</span>{/if}
            {#if order.AnnulReason}<span>{tr('Reason|Motivo')}: {order.AnnulReason}</span>{/if}
          </div>
          <OrderLineCards lines={order.Lines} {productName} showPrice columns={3} />
        </div>
      {/if}
    </div>
  {:else}
    <div class="p-12 rounded-md bg-surface-soft text-fg-muted">
      {isLoading ? tr('Loading...|Cargando...') : tr('No orders today or yesterday.|Sin pedidos hoy ni ayer.')}
    </div>
  {/each}
</div>
