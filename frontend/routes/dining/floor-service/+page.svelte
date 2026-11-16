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
  import MenuPicker from './MenuPicker.svelte'
  import {
    LINE_STATUS, TABLE_STATE, shapeAt, shapeRect, tableLabelOf, tableState, tableStateLabels, tableWaitTicks, waitMinutes,
    type IGridPoint, type ILayoutShape, type ITableOrder,
  } from '../dining.ts'
  import ReadyToDeliverList from './ReadyToDeliverList.svelte'
  import TableOrderPanel from './TableOrderPanel.svelte'
  import { isTableShape } from './floor-service.ts'
  import { RoundDraftState } from './floor-service.svelte.ts'

  // The waiter's page (docs/DINING_PLAN.md §10). "Mapa": each table colored by its order's state
  // (§9) and a side panel to order, follow and charge one table. "Por entregar": the dishes the
  // kitchen ("Pedidos", dining/kitchen-display) marked ready. Until the live bridge lands (step 4)
  // the open orders are pulled every 15 seconds while the page is visible.
  const ui = useUI()
  const TAB_MAP = 1
  const TAB_TO_DELIVER = 2
  const TABLE_LAYER_ID = 1
  const POLL_MILLISECONDS = 15_000
  const CELL_SIZE = 16

  const layoutsService = new DiningLayoutsService()
  const productsService = new ProductsService(true)
  let layoutPicker = $state({ LayoutID: 0 })
  let ordersService = $state<TableOrdersService>()
  let selectedTableShapeID = $state(0)
  // The selected table's round in the making: "Nuevo pedido" edits it, the menu picker beside the layer adds to it.
  const round = new RoundDraftState()
  // The 2-second clock the waits are measured with; it re-renders, it recomputes no data.
  let nowTime = $state(nowSUnixTime())

  const layout = $derived(layoutsService.recordsMap.get(layoutPicker.LayoutID))
  const orderByTable = $derived(new Map((ordersService?.records || []).map((order) => [order.TableShapeID, order])))
  const selectedOrder = $derived(orderByTable.get(selectedTableShapeID))
  const selectedWaitMinutes = $derived(waitMinutes(tableWaitTicks(selectedOrder, nowTime)))
  // The "Por entregar" tab carries how many dishes wait, so the waiter sees it from the map.
  const readyLineCount = $derived((ordersService?.records || [])
    .reduce((count, order) => count + order.Lines.filter((line) => line.Status === LINE_STATUS.READY).length, 0))
  const pageTabs = $derived([
    { id: TAB_MAP, name: 'Map|Mapa' },
    { id: TAB_TO_DELIVER, name: readyLineCount ? `To deliver (${readyLineCount})|Por entregar (${readyLineCount})` : 'To deliver|Por entregar' },
  ])

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

  const onOrderChanged = (order: ITableOrder) => ordersService?.applyChangedOrder(order)

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
  // The same hue as the table on the map, as a label beside the table's name in its side panel.
  const STATE_BADGE: Record<number, string> = {
    [TABLE_STATE.FREE]: 'bg-surface-soft text-fg-muted',
    [TABLE_STATE.EATING]: 'bg-green-bg text-green-fg',
    [TABLE_STATE.ORDERED]: 'bg-blue-bg text-blue-fg',
    [TABLE_STATE.WAITING]: 'bg-amber-bg text-amber-fg',
    [TABLE_STATE.TO_CONFIRM]: 'bg-purple-bg text-purple-fg',
    [TABLE_STATE.READY]: 'bg-teal-bg text-teal-fg',
    [TABLE_STATE.LATE]: 'bg-red-bg text-red-fg',
  }
  const shapeFill = (shape: ILayoutShape) =>
    isTableShape(shape.Kind) && layout ? STATE_FILL[tableState(orderByTable.get(shape.ShapeID), layout, nowTime)] : undefined
  const productName = (productID: number) => productsService.recordsMap.get(productID)?.Name || `#${productID}`
</script>

<Page title="Tables|Mesas" options={pageTabs}>
  <Layer type="content">
    <div class="flex flex-wrap items-end gap-10 mb-10" aria-label="Pick the layout to serve">
      <SearchSelect saveOn={layoutPicker} save="LayoutID" css="w-full md:w-280" label="Layout|Salón"
        options={layoutsService.records} keyId="ID" keyName="Name" placeholder=":: seleccione ::" />
      {#if ui.state.pageOptionSelected === TAB_MAP}
        <div class="flex flex-wrap gap-6" aria-label="Table colors">
          {#each Object.values(TABLE_STATE) as state (state)}
            <span class="flex items-center gap-4 text-fg-muted">
              <svg width="14" height="14"><rect width="14" height="14" rx="3" class="{STATE_FILL[state]} stroke-line-strong" /></svg>
              {tr(tableStateLabels[state])}
            </span>
          {/each}
        </div>
      {/if}
    </div>

    {#if layout && ui.state.pageOptionSelected === TAB_TO_DELIVER}
      <ReadyToDeliverList {layout} orders={ordersService?.records || []} {nowTime} {productName} {onOrderChanged} />
    {:else if layout}
      <div class="overflow-auto rounded-md border border-line">
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
      <!-- min-h-240: the content layer scrolls (overflow: auto), so it must be tall enough for the Salón dropdown. -->
      <div class="p-12 min-h-240 rounded-md bg-surface-soft text-fg-muted">{tr('Pick a layout.|Elija un salón.')}</div>
    {/if}
  </Layer>

  <!-- While "Nuevo pedido" is open the menu picker sits left of the layer (leftPanel), over the dimmed map. -->
  <Layer type="side" id={TABLE_LAYER_ID} width={770} titleCss="h2"
    leftPanel={round.isComposing ? menuPicker : undefined} leftPanelWidth={380} useLeftPanelBackdrop
    title={layout && selectedTableShapeID ? tr(`Table ${tableLabelOf(layout, selectedTableShapeID)}|Mesa ${tableLabelOf(layout, selectedTableShapeID)}`) : ''}
  >
    {#snippet titleSide()}
      <!-- Beside the table's name: its state, as a label in its map color, and how long the kitchen keeps it waiting. -->
      {#if layout && selectedOrder}
        {@const selectedState = tableState(selectedOrder, layout, nowTime)}
        <span class="inline-block mb-6 px-6 py-1 rounded-md text-[15px] ff-bold text-nowrap {STATE_BADGE[selectedState]}"
          aria-label="The table's state and wait">
          {tr(tableStateLabels[selectedState]).toUpperCase()}{#if selectedWaitMinutes}&nbsp;· {selectedWaitMinutes} min{/if}
        </span>
      {/if}
    {/snippet}
    {#snippet subtitle()}
      <!-- The second line of the header: whether the table waits for the kitchen, and its guests. -->
      {#if selectedOrder && (selectedWaitMinutes || selectedOrder.Guests)}
        <div class="-mt-4 text-fg-muted text-nowrap" aria-label="The table's wait and guests">
          {[
            selectedWaitMinutes ? tr('Waiting|Esperando') : '',
            selectedOrder.Guests ? `${selectedOrder.Guests} ${tr(selectedOrder.Guests === 1 ? 'guest|comensal' : 'guests|comensales')}` : '',
          ].filter(Boolean).join('. ')}
        </div>
      {/if}
    {/snippet}
    {#if layout && selectedTableShapeID}
      {#key selectedTableShapeID}
        <TableOrderPanel {round} {layout} tableShapeID={selectedTableShapeID} order={selectedOrder} {nowTime} {productName} {onOrderChanged} />
      {/key}
    {/if}
  </Layer>
</Page>

{#snippet menuPicker()}
  <MenuPicker {round} />
{/snippet}
