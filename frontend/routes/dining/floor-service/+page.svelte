<script lang="ts">
  import { useUI } from '@genix/ui'
  import { onMount, untrack } from 'svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import LayerStatic from '#components/layers/LayerStatic.svelte'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { nowSUnixTime } from '#libs/helpers.ts'
  import { ProductPacksService } from '#services/production/product-packs.svelte.ts'
  import { ProductsService } from '#services/production/products.svelte.ts'
  import LayoutCanvas from '../LayoutCanvas.svelte'
  import { DiningLayoutsService, TableOrdersService, readServedLayoutID, saveServedLayoutID } from '../dining.svelte.ts'
  import MenuPicker from './MenuPicker.svelte'
  import {
    TABLE_STATE, shapeAt, shapeRect, tableLabelOf, tableState, tableStateLabels, tableWaitTicks, waitMinutes,
    type IGridPoint, type ILayoutShape, type ITableOrder,
  } from '../dining.ts'
  import TablesToServeList from './TablesToServeList.svelte'
  import TableOrderPanel from './TableOrderPanel.svelte'
  import { diningMenu, isTableShape, tablesToServe } from './floor-service.ts'
  import { DiningMenuSources, RoundDraftState } from './floor-service.svelte.ts'

  // The waiter's page (docs/DINING_PLAN.md §10). "Mapa": each table colored by its order's state
  // (§9) and a side panel to order, follow and charge one table. "Por entregar": the page split in
  // two, the tables still owed a dish on the left (the longest-waiting first) and the picked one
  // fixed on the right, with the same panel as the map's. Until the live bridge lands (step 4)
  // the open orders are pulled every 15 seconds while the page is visible.
  const ui = useUI()
  const TAB_MAP = 1
  const TAB_TO_DELIVER = 2
  const TABLE_LAYER_ID = 1
  const POLL_MILLISECONDS = 15_000
  const CELL_SIZE = 16

  const layoutsService = new DiningLayoutsService()
  const productsService = new ProductsService(true)
  // The packs the tables' rounds and bills preview; the charge applies them for real.
  const packsService = new ProductPacksService(true)
  let layoutPicker = $state({ LayoutID: readServedLayoutID() })
  let ordersService = $state<TableOrdersService>()
  let selectedTableShapeID = $state(0)
  // The selected table's round in the making: "Nuevo pedido" edits it, the menu picker beside the layer adds to it.
  const round = new RoundDraftState()
  const menuSources = new DiningMenuSources()
  // Re-creates the table's panel, which opens on "Pedidos" when the table has an order: how a tap on
  // "Por entregar"'s dimmed list leaves "Nuevo pedido" (every listed table has one).
  let tablePanelMountCount = $state(0)
  // The 2-second clock the waits are measured with; it re-renders, it recomputes no data.
  let nowTime = $state(nowSUnixTime())

  const layout = $derived(layoutsService.recordsMap.get(layoutPicker.LayoutID))
  const menu = $derived(layout
    ? diningMenu(productsService.records, menuSources.stockService.records, layout.WarehouseID, menuSources.penRate) : [])
  const isMenuLoading = $derived(!productsService.isReady || !menuSources.stockService.isReady)
  const orderByTable = $derived(new Map((ordersService?.records || []).map((order) => [order.TableShapeID, order])))
  const selectedOrder = $derived(orderByTable.get(selectedTableShapeID))
  const selectedWaitMinutes = $derived(waitMinutes(tableWaitTicks(selectedOrder, nowTime)))
  const tablesToServeList = $derived(layout ? tablesToServe(ordersService?.records || [], layout, nowTime) : [])
  // The "Por entregar" tab carries how many tables are owed a dish, so the waiter sees it from the map.
  const pageTabs = $derived([
    { id: TAB_MAP, name: 'Map|Mapa' },
    {
      id: TAB_TO_DELIVER,
      name: tablesToServeList.length ? `To deliver (${tablesToServeList.length})|Por entregar (${tablesToServeList.length})` : 'To deliver|Por entregar',
    },
  ])
  const selectedTableTitle = $derived(layout && selectedTableShapeID
    ? tr(`Table ${tableLabelOf(layout, selectedTableShapeID)}|Mesa ${tableLabelOf(layout, selectedTableShapeID)}`) : '')

  // The only layout opens by itself, also over a remembered one that no longer exists.
  $effect(() => {
    if (layoutsService.records.length === 1 && !untrack(() => layoutsService.recordsMap.has(layoutPicker.LayoutID))) {
      layoutPicker.LayoutID = layoutsService.records[0].ID
    }
  })
  // Each layout syncs its own open orders. The pick is remembered for the next reload.
  $effect(() => {
    const layoutID = layoutPicker.LayoutID
    untrack(() => {
      ordersService = layoutID ? new TableOrdersService(layoutID) : undefined
      if (layoutID) saveServedLayoutID(layoutID)
    })
  })

  // Units left change by the minute: each "Nuevo pedido" syncs the stock again (the cached rows paint first).
  $effect(() => {
    if (round.isComposing) untrack(() => menuSources.stockService.fetch())
  })

  onMount(() => {
    const clockTimer = setInterval(() => { nowTime = nowSUnixTime() }, 2000)
    const pollTimer = setInterval(() => {
      if (document.visibilityState === 'visible') ordersService?.fetchOnline()
    }, POLL_MILLISECONDS)
    return () => { clearInterval(clockTimer); clearInterval(pollTimer) }
  })

  // "Por entregar" always shows one table on its right half: the one the waiter tapped while it is
  // still owed a dish, else the longest-waiting one. The map's side layer closes on entering the tab.
  $effect(() => {
    if (ui.state.pageOptionSelected !== TAB_TO_DELIVER) return
    const listedTableShapeIDs = tablesToServeList.map((table) => table.order.TableShapeID)
    untrack(() => {
      if (ui.state.sideLayerId === TABLE_LAYER_ID) ui.openSideLayer(0)
      if (!listedTableShapeIDs.includes(selectedTableShapeID)) selectedTableShapeID = listedTableShapeIDs[0] || 0
    })
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
  const productCategoryIDs = (productID: number) => productsService.recordsMap.get(productID)?.CategoryIDs || []
</script>

<Page title="Tables|Mesas" options={pageTabs}>
  {#if ui.state.pageOptionSelected === TAB_TO_DELIVER}
    <!-- Split in two: the tables owed dishes on the left, the picked one fixed on the right (LayerStatic).
         The panel takes half of what is right of the menu, the width the menu picker's overlay stops at. -->
    <div class="flex h-full gap-20 [--deliver-panel-width:calc((100vw-var(--menu-min-width))/2)]">
      <div class="flex-1 min-w-0 flex flex-col h-[calc(100vh-var(--header-height)-24px)]">
        {@render layoutPickerBar()}
        <div class="flex-1 min-h-0 overflow-y-auto p-4 -m-4">
          {#if layout}
            <TablesToServeList tables={tablesToServeList} {selectedTableShapeID} {productName}
              onSelect={(tableShapeID) => { selectedTableShapeID = tableShapeID }} />
          {:else}
            <div class="p-12 rounded-md bg-surface-soft text-fg-muted">{tr('Pick a layout.|Elija un salón.')}</div>
          {/if}
        </div>
        <!-- While "Nuevo pedido" is open the menu picker sits flush left of the panel, over the dimmed list, as on the map. -->
        {#if round.isComposing && ui.state.deviceType !== 3}
          <!-- Fixed like the map Layer's backdrop: the page's padding would clip an absolute one. Where the map's
               backdrop closes the layer, this one drops the round and goes back to "Pedidos". -->
          <div class="fixed top-[var(--header-height)] left-[var(--menu-min-width)] bottom-0 right-[var(--deliver-panel-width)]
            z-[var(--layer-zindex)] flex justify-end">
            <button type="button" aria-label={tr('Back to the orders|Volver a pedidos')} onclick={() => { tablePanelMountCount++ }}
              class="absolute inset-0 bg-black/35 backdrop-blur-[4px] cursor-default"></button>
            <div class="relative flex flex-col w-380 px-10 py-10 bg-surface shadow-lg">{@render menuPicker()}</div>
          </div>
        {/if}
      </div>
      <LayerStatic css="w-[var(--deliver-panel-width)] shrink-0 bg-surface border-l border-line flex flex-col h-[calc(100vh-var(--header-height))] shadow-lg md:-m-10"
        mobileLayerTitle={selectedTableTitle} useMobileLayerVertical={124}>
        {#if selectedTableShapeID}
          <div class="flex flex-col px-12 pt-10" aria-label="The picked table">
            <div class="flex items-center gap-8">
              <span class="h2">{selectedTableTitle}</span>
              {@render tableStateBadge()}
            </div>
            {@render tableGuestsLine()}
          </div>
          {#if round.isComposing && ui.state.deviceType === 3}
            <div class="h-[42vh] shrink-0 mt-8 px-12">{@render menuPicker()}</div>
          {/if}
          <div class="flex-1 min-h-0 overflow-y-auto px-12 pb-10">{@render tablePanel()}</div>
        {:else}
          <div class="m-12 p-12 rounded-md bg-surface-soft text-fg-muted">{tr('No table is owed a dish.|Ninguna mesa espera platos.')}</div>
        {/if}
      </LayerStatic>
    </div>
  {:else}
    <Layer type="content">
      {@render layoutPickerBar()}
      {#if layout}
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
  {/if}

  <!-- While "Nuevo pedido" is open the menu picker sits left of the layer (leftPanel), over the dimmed map. -->
  <Layer type="side" id={TABLE_LAYER_ID} width={770} titleCss="h2" title={selectedTableTitle}
    leftPanel={round.isComposing ? menuPicker : undefined} leftPanelWidth={380} useLeftPanelBackdrop
    titleSide={tableStateBadge} subtitle={tableGuestsLine}>
    {@render tablePanel()}
  </Layer>
</Page>

{#snippet layoutPickerBar()}
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
{/snippet}

<!-- Beside the table's name: its state, as a label in its map color, and how long the kitchen keeps it waiting. -->
{#snippet tableStateBadge()}
  {#if layout && selectedOrder}
    {@const selectedState = tableState(selectedOrder, layout, nowTime)}
    <span class="inline-block mb-6 px-6 py-1 rounded-md text-[15px] ff-bold text-nowrap {STATE_BADGE[selectedState]}"
      aria-label="The table's state and wait">
      {tr(tableStateLabels[selectedState]).toUpperCase()}{#if selectedWaitMinutes}&nbsp;· {selectedWaitMinutes} min{/if}
    </span>
  {/if}
{/snippet}

<!-- The second line of the header: whether the table waits for the kitchen, and its guests. -->
{#snippet tableGuestsLine()}
  {#if selectedOrder && (selectedWaitMinutes || selectedOrder.Guests)}
    <div class="-mt-4 text-fg-muted text-nowrap" aria-label="The table's wait and guests">
      {[
        selectedWaitMinutes ? tr('Waiting|Esperando') : '',
        selectedOrder.Guests ? `${selectedOrder.Guests} ${tr(selectedOrder.Guests === 1 ? 'guest|comensal' : 'guests|comensales')}` : '',
      ].filter(Boolean).join('. ')}
    </div>
  {/if}
{/snippet}

<!-- The table's tabs, the same in the map's side layer and in "Por entregar"'s right half. -->
{#snippet tablePanel()}
  {#if layout && selectedTableShapeID}
    {#key `${selectedTableShapeID}_${tablePanelMountCount}`}
      <TableOrderPanel {round} {layout} tableShapeID={selectedTableShapeID} order={selectedOrder} {nowTime} {productName}
        {productCategoryIDs} packs={packsService.records} {onOrderChanged} />
    {/key}
  {/if}
{/snippet}

{#snippet menuPicker()}
  <MenuPicker {round} {menu} isLoading={isMenuLoading} />
{/snippet}
