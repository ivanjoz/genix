<script lang="ts">
  import { notifyFailure } from '@genix/ui/notify'
  import Kanban from '#components/layers/Kanban.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { PRODUCTION_ORDER_STATUS, type IProductionOrder } from '#services/production/production-orders.ts'
  import { postProductionOrderStage } from '#services/production/production-orders.svelte.ts'
  import { ORDER_STATUS_LABELS, orderProgressPercent, unitsByProduct } from './production-order.ts'
  import type { ProductionCatalogs } from './production-order.svelte.ts'

  // The status board (docs/MANUFACTURING_PLAN.md §9.4): one column per stage, the released and
  // in-progress orders as cards. Moving a card only changes its stage: no stock, no cost.
  let { catalogs, onOpen }: { catalogs: ProductionCatalogs, onOpen: (order: IProductionOrder) => void } = $props()

  // Kanban ignores a drop on column 0, so the orders with no stage sit in a column of their own id.
  const NO_STAGE_COLUMN_ID = -1

  const columns = $derived([
    { id: NO_STAGE_COLUMN_ID, name: tr('No stage|Sin etapa') },
    ...catalogs.stages.records.map((stage) => ({ id: stage.ID, name: stage.Name })),
  ])
  const cards = $derived((catalogs.orders?.records || []).filter((order) =>
    order.ss === PRODUCTION_ORDER_STATUS.RELEASED || order.ss === PRODUCTION_ORDER_STATUS.IN_PROGRESS))
  // An order whose stage was deleted shows under "No stage".
  const columnIDOf = (order: IProductionOrder) => catalogs.stages.recordsMap.has(order.StageID) ? order.StageID : NO_STAGE_COLUMN_ID

  // The card moves at once and goes back if the server refuses.
  const moveOrder = async (orderID: number, toColumnID: number) => {
    const movedOrder = cards.find((order) => order.ID === orderID)
    if (!movedOrder || !catalogs.orders) return
    const previousStageID = movedOrder.StageID
    movedOrder.StageID = toColumnID === NO_STAGE_COLUMN_ID ? 0 : toColumnID
    try {
      await postProductionOrderStage(catalogs.orders, orderID, movedOrder.StageID)
    } catch (moveError) {
      console.error('[production-orders] stage move error', { orderID, toColumnID, moveError })
      movedOrder.StageID = previousStageID
      notifyFailure(moveError)
    }
  }
</script>

<Kanban {columns} {cards} getCardID={(order) => order.ID} getColumnID={columnIDOf} onmove={moveOrder} css="grow min-h-0 pb-12">
  {#snippet card(order)}
    {@const progressPercent = orderProgressPercent(order.Lines || [])}
    <button type="button" class="w-full text-left p-8 flex flex-col gap-4 cursor-pointer rounded-md text-fg hover:bg-surface-muted"
      onclick={() => onOpen(order)} aria-label="Open production order {order.ID}">
      <div class="flex items-center justify-between gap-6">
        <span class="ff-bold">{tr('Order|Orden')} N° {order.ID}</span>
        <span class="text-fg-muted">{tr(ORDER_STATUS_LABELS[order.ss])}</span>
      </div>
      {#each unitsByProduct(order.Lines || []) as productUnits (productUnits.ProductID)}
        <div class="text-fg-soft">{productUnits.Units}x {catalogs.productNameOf(productUnits.ProductID)}</div>
      {/each}
      {#if order.SubcontractorID > 0}
        <div class="text-fg-muted"><i class="icon-[fa--truck] mr-4"></i>{catalogs.workshopNameOf(order.SubcontractorID)}</div>
      {/if}
      <div class="flex items-center gap-8" aria-label="Received {progressPercent}%">
        <div class="grow h-6 rounded-full bg-surface-muted overflow-hidden">
          <div class="h-full bg-green-solid" style:width="{Math.min(progressPercent, 100)}%"></div>
        </div>
        <span class="ff-mono text-fg-muted">{progressPercent}%</span>
      </div>
    </button>
  {/snippet}
</Kanban>
