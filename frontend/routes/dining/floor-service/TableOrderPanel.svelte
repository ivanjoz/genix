<script lang="ts">
  import { useUI } from '@genix/ui'
  import { untrack } from 'svelte'
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
  import type { IProductPack } from '#core/product-packs.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import { security } from '#libs/ui-runtime.svelte.ts'
  import OrderLineCards from '../OrderLineCards.svelte'
  import { postTableOrderLines } from '../dining.svelte.ts'
  import { lineActions, type IDiningLayout, type ITableOrder } from '../dining.ts'
  import { orderTotal } from './floor-service.ts'
  import type { RoundDraftState } from './floor-service.svelte.ts'
  import RoundComposer from './RoundComposer.svelte'
  import TableChargeForm from './TableChargeForm.svelte'
  import TableOrderHistory from './TableOrderHistory.svelte'
  import { CHARGE_TABLE_SUB_ACCESS_ID, TABLE_SERVICE_ACCESS_ID } from './table-charge.ts'

  // One table, in four tabs: "Pedidos" (the open order's rounds and their line actions), "Pagar"
  // (the bill, charged or annulled), "Nuevo pedido" (a new round; the first one opens the order) and
  // "Historial" (the table's orders of today and yesterday). Charging the bill frees the table.
  let { round, layout, tableShapeID, order, nowTime, productName, productCategoryIDs, packs, onOrderChanged }: {
    round: RoundDraftState
    layout: IDiningLayout
    tableShapeID: number
    order: ITableOrder | undefined
    nowTime: number
    productName: (productID: number) => string
    // What "Nuevo pedido" and "Pagar" find packs with.
    productCategoryIDs: (productID: number) => number[]
    packs: IProductPack[]
    onOrderChanged: (order: ITableOrder) => void
  } = $props()

  const TAB_ORDERS = 1
  const TAB_PAY = 2
  const TAB_NEW_ROUND = 3
  const TAB_HISTORY = 4

  const ui = useUI()
  let isMovingLines = $state(false)
  let selectedLineIDs = $state<number[]>([])
  const canCloseBill = security.checkSubAcceso(TABLE_SERVICE_ACCESS_ID, CHARGE_TABLE_SUB_ACCESS_ID)
  // A table with an order opens on its rounds; a free one on its first round. Read once: the panel
  // is re-created per table, and the waiter's pick wins afterwards.
  let selectedTab = $state(untrack(() => order) ? TAB_ORDERS : TAB_NEW_ROUND)

  // Every tab shows on every table, a free one too ("Pedidos" and "Pagar" then say there is nothing);
  // only "Pagar" needs the "Cobrar mesa" sub-access.
  const tabs = $derived([
    [TAB_ORDERS, tr('Orders|Pedidos')],
    ...(canCloseBill ? [[TAB_PAY, tr('Pay|Pagar')]] : []),
    [TAB_NEW_ROUND, tr('New round|Nuevo pedido')],
    [TAB_HISTORY, tr('History|Historial')],
  ] as [number, string][])

  // A closed bill leaves the floor map, and the table is free again.
  function onBillClosed(closedOrder: ITableOrder) {
    onOrderChanged(closedOrder)
    ui.openSideLayer(0)
  }
  const formatAmount = (cents: number) => `S/ ${formatN(cents / 100, 2)}`

  // One request moves every selected line the action fits; the selection is spent with it.
  async function moveLines(status: number, lineIDs: number[]) {
    if (!order || isMovingLines) return
    isMovingLines = true
    try {
      onOrderChanged(await postTableOrderLines({ OrderID: order.ID, LineIDs: lineIDs, Status: status }))
      selectedLineIDs = []
    } catch (linesError) {
      console.error('[floor-service] lines error', { orderID: order.ID, lineIDs, status, linesError })
    } finally {
      isMovingLines = false
    }
  }
</script>

<!-- As tall as the layer: in "Pedidos" only the rounds scroll (OrderLineCards), under a fixed toolbar.
     The other tabs outgrow it and the layer scrolls them as a whole. -->
<div class="h-full flex flex-col gap-12 pt-8" aria-label="The table: its order, its bill, a new round and its history">
  <OptionsStrip options={tabs} selected={selectedTab} buttonCss="min-w-0 flex-1"
    onSelect={(option) => { selectedTab = option[0] }} />

  {#if (selectedTab === TAB_ORDERS || selectedTab === TAB_PAY) && !order}
    <!-- A free table, or its order closed elsewhere (another waiter charged it). -->
    <div class="p-12 rounded-md bg-purple-bg text-purple-fg" aria-label="The table has no open order">
      {selectedTab === TAB_ORDERS ? tr('No open orders.|No hay pedidos abiertos.') : tr('Nothing to pay.|No hay nada que pagar.')}
    </div>
  {:else if selectedTab === TAB_ORDERS && order}
    <OrderLineCards lines={order.Lines} {productName} actionsOf={(line) => lineActions(line, nowTime)} showPrice columns={3}
      bind:selectedLineIDs isBusy={isMovingLines} onApply={moveLines}>
      {#snippet toolbar()}
        <!-- The order's running total, left of the selection's buttons. -->
        <span class="flex items-center gap-6 ff-bold ff-mono" aria-label="The order's total">
          <i class="icon-[mdi--receipt-text-outline] text-[20px] text-fg-muted"></i>
          {formatAmount(orderTotal(order))}
        </span>
      {/snippet}
    </OrderLineCards>
  {:else if selectedTab === TAB_PAY && order}
    <TableChargeForm {order} {productName} {productCategoryIDs} {packs} onDone={onBillClosed} />
  {:else if selectedTab === TAB_NEW_ROUND}
    <RoundComposer {round} layoutID={layout.ID} {tableShapeID} isOpening={!order} {productCategoryIDs} {packs}
      onSent={(savedOrder) => { onOrderChanged(savedOrder); selectedTab = TAB_ORDERS }}
      onCancel={() => { if (order) selectedTab = TAB_ORDERS; else ui.openSideLayer(0) }} />
  {:else if selectedTab === TAB_HISTORY}
    <TableOrderHistory layoutID={layout.ID} {tableShapeID} {productName} />
  {/if}
</div>
