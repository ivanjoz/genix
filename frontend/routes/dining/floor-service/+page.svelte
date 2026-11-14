<script lang="ts">
  import { useUI } from '@genix/ui'
  import { onMount, untrack } from 'svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { nowSUnixTime } from '#libs/helpers.ts'
  import { ProductsService } from '#services/production/products.svelte.ts'
  import LayoutCanvas from '../LayoutCanvas.svelte'
  import { DiningLayoutsService, TableOrdersService } from '../dining.svelte.ts'
  import {
    TABLE_STATE, shapeAt, shapeRect, tableState, tableStateLabels, tableWaitTicks, waitMinutes,
    type IGridPoint, type ILayoutShape, type ITableOrder,
  } from '../dining.ts'
  import ReadyToPickupStrip from './ReadyToPickupStrip.svelte'
  import TableOrderPanel from './TableOrderPanel.svelte'
  import { isTableShape, tableLabelOf } from './floor-service.ts'

  // The live floor map (docs/DINING_PLAN.md §10): each table colored by its order's state (§9), the
  // dishes waiting at the pickup points, and a side panel to order and follow one table. Until the
  // live bridge lands (step 4) the open orders are pulled every 15 seconds while the page is visible.
  const ui = useUI()
  const TABLE_LAYER_ID = 1
  const POLL_MILLISECONDS = 15_000
  const CELL_SIZE = 16

  const layoutsService = new DiningLayoutsService()
  const productsService = new ProductsService(true)
  let layoutPicker = $state({ LayoutID: 0 })
  let ordersService = $state<TableOrdersService>()
  let selectedTableShapeID = $state(0)
  // The 2-second clock the waits are measured with; it re-renders, it recomputes no data.
  let nowTime = $state(nowSUnixTime())

  const layout = $derived(layoutsService.recordsMap.get(layoutPicker.LayoutID))
  const orderByTable = $derived(new Map((ordersService?.records || []).map((order) => [order.TableShapeID, order])))
  const selectedOrder = $derived(orderByTable.get(selectedTableShapeID))

  // The only layout opens by itself.
  $effect(() => {
    if (layoutsService.records.length === 1 && !untrack(() => layoutPicker.LayoutID)) {
      layoutPicker.LayoutID = layoutsService.records[0].ID
    }
  })
  // Each layout syncs its own open orders.
  $effect(() => {
    const layoutID = layoutPicker.LayoutID
    untrack(() => { ordersService = layoutID ? new TableOrdersService(layoutID) : undefined })
  })

  onMount(() => {
    const clockTimer = setInterval(() => { nowTime = nowSUnixTime() }, 2000)
    const pollTimer = setInterval(() => {
      if (document.visibilityState === 'visible') ordersService?.fetchOnline()
    }, POLL_MILLISECONDS)
    return () => { clearInterval(clockTimer); clearInterval(pollTimer) }
  })

  // A write answers the whole order: it replaces the cached one at once, and the sync follows.
  function onOrderChanged(order: ITableOrder) {
    if (!ordersService) return
    const others = ordersService.records.filter((candidate) => candidate.ID !== order.ID)
    ordersService.records = order.ss === 1 ? [...others, order] : others
    ordersService.fetchOnline()
  }

  function onPointerDown(_event: PointerEvent, point: IGridPoint) {
    const shape = layout && shapeAt(layout.Shapes || [], point)
    if (!shape || !isTableShape(shape.Kind)) return
    selectedTableShapeID = shape.ShapeID
    ui.openSideLayer(TABLE_LAYER_ID)
  }

  // The mid tones: the -bg ones are too pale to tell the states apart across a room.
  const STATE_FILL: Record<number, string> = {
    [TABLE_STATE.FREE]: 'fill-surface-raised',
    [TABLE_STATE.EATING]: 'fill-green-border',
    [TABLE_STATE.ORDERED]: 'fill-blue-border',
    [TABLE_STATE.WAITING]: 'fill-amber-border',
    [TABLE_STATE.TO_CONFIRM]: 'fill-purple-border',
    [TABLE_STATE.READY]: 'fill-teal-border',
    [TABLE_STATE.LATE]: 'fill-red-border',
  }
  const shapeFill = (shape: ILayoutShape) =>
    isTableShape(shape.Kind) && layout ? STATE_FILL[tableState(orderByTable.get(shape.ShapeID), layout, nowTime)] : undefined
  const productName = (productID: number) => productsService.recordsMap.get(productID)?.Name || `#${productID}`
</script>

<Page title="Table service|Atención de mesas">
  <Layer type="content">
    <div class="flex flex-wrap items-end gap-10 mb-10" aria-label="Pick the layout to serve">
      <SearchSelect saveOn={layoutPicker} save="LayoutID" css="w-full md:w-280" label="Layout|Salón"
        options={layoutsService.records} keyId="ID" keyName="Name" placeholder=":: seleccione ::" />
      <div class="flex flex-wrap gap-6" aria-label="Table colors">
        {#each Object.values(TABLE_STATE) as state (state)}
          <span class="flex items-center gap-4 text-fg-muted">
            <svg width="14" height="14"><rect width="14" height="14" rx="3" class="{STATE_FILL[state]} stroke-line-strong" /></svg>
            {tr(tableStateLabels[state])}
          </span>
        {/each}
      </div>
    </div>

    {#if layout}
      <ReadyToPickupStrip {layout} orders={ordersService?.records || []} {nowTime} {productName} {onOrderChanged} />
      <div class="overflow-auto rounded-md border border-line mt-10">
        <LayoutCanvas {layout} cellSize={CELL_SIZE} showGrid={false} {shapeFill} {onPointerDown}
          selectedShapeIDs={ui.state.sideLayerId === TABLE_LAYER_ID ? [selectedTableShapeID] : []}>
          {#snippet overlay()}
            <!-- How long each table has waited for its kitchen, just above its top-right corner (inside it would cover the label). -->
            {#each (layout.Shapes || []).filter((shape) => isTableShape(shape.Kind)) as table (table.ShapeID)}
              {@const waitedTicks = tableWaitTicks(orderByTable.get(table.ShapeID), nowTime)}
              {#if waitMinutes(waitedTicks)}
                {@const rect = shapeRect(table)}
                <text x={(rect.x + rect.w) * CELL_SIZE} y={rect.y * CELL_SIZE - 4} text-anchor="end"
                  class="fill-fg text-[14px] ff-bold pointer-events-none">{waitMinutes(waitedTicks)}'</text>
              {/if}
            {/each}
          {/snippet}
        </LayoutCanvas>
      </div>
    {:else}
      <div class="p-12 rounded-md bg-surface-soft text-fg-muted">{tr('Pick a layout.|Elija un salón.')}</div>
    {/if}
  </Layer>

  <Layer type="side" id={TABLE_LAYER_ID} sideLayerSize={640} titleCss="h2 mb-6"
    title={layout && selectedTableShapeID ? tr(`Table ${tableLabelOf(layout, selectedTableShapeID)}|Mesa ${tableLabelOf(layout, selectedTableShapeID)}`) : ''}>
    {#if layout && selectedTableShapeID}
      {#key selectedTableShapeID}
        <TableOrderPanel {layout} tableShapeID={selectedTableShapeID} order={selectedOrder} {nowTime} {productName} {onOrderChanged} />
      {/key}
    {/if}
  </Layer>
</Page>
