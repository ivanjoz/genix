<script lang="ts">
  import { confirmWarn, notifyFailure, notifySuccess } from '@genix/ui/notify'
  import { untrack } from 'svelte'
  import Button from '#components/buttons/Button.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import LoadingBar from '#components/misc/LoadingBar.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import { WarehousesService } from '#routes/business/branches-warehouses/branches-warehouses.svelte.ts'
  import { ProductsService } from '#services/production/products.svelte.ts'
  import type { IProductionOrder } from '#services/production/production-orders.ts'
  import DailyCloseSummary from './DailyCloseSummary.svelte'
  import {
    canRunClose, closeOrderProductUnits, closeOrderTotalValue, oldestPendingDay,
    type IDailyClosePendingDay, type IDailyClosePreview,
  } from './daily-close.ts'
  import { getDailyClosePending, getDailyClosePreview, postDailyClose, ProductionClosesService } from './daily-close.svelte.ts'

  const warehousesService = new WarehousesService()
  const productsService = new ProductsService()

  // W: where the sales left and the products enter. S: where the supplies are consumed (default W).
  let closeForm = $state({ WarehouseID: 0, SuppliesWarehouseID: 0 })
  let pendingDays = $state<IDailyClosePendingDay[]>([])
  let preview = $state<IDailyClosePreview>()
  let closesService = $state<ProductionClosesService>()
  let isLoading = $state(false)
  let isClosing = $state(false)
  let loadRequestID = 0

  const closableDay = $derived(oldestPendingDay(pendingDays))
  const formatAmount = (cents: number) => `S/ ${formatN((cents || 0) / 100, 2)}`

  // Reads the pending days, then previews the oldest: only that one can be closed. A late answer
  // for a warehouse left behind is dropped.
  async function loadWarehouseClose(warehouseID: number, suppliesWarehouseID: number) {
    const requestID = ++loadRequestID
    preview = undefined
    if (!(warehouseID > 0)) { pendingDays = []; return }
    isLoading = true
    try {
      const nextPendingDays = await getDailyClosePending(warehouseID)
      if (requestID !== loadRequestID) return
      pendingDays = nextPendingDays
      const dayToClose = oldestPendingDay(nextPendingDays)
      if (!dayToClose) return

      const nextPreview = await getDailyClosePreview(warehouseID, suppliesWarehouseID || warehouseID, dayToClose.Date)
      // The product and supply names, before the summary renders them.
      const productIDs = [...(nextPreview.Products || []).map((product) => product.ProductID),
        ...(nextPreview.Supplies || []).map((supply) => supply.SupplyID)]
      await productsService.syncIDs(productIDs)
      if (requestID !== loadRequestID) return
      preview = nextPreview
    } catch (loadError) {
      console.error('[daily-close] load error', { warehouseID, suppliesWarehouseID, loadError })
      notifyFailure(String(loadError) || tr('Could not load the close.|No se pudo cargar el cierre.'))
    } finally {
      if (requestID === loadRequestID) isLoading = false
    }
  }

  // The history of the warehouse; a late answer for a warehouse left behind is dropped.
  let closesRequestID = 0
  async function loadCloses(warehouseID: number) {
    const requestID = ++closesRequestID
    closesService = undefined
    if (!(warehouseID > 0)) return
    try {
      const nextClosesService = new ProductionClosesService(warehouseID)
      await nextClosesService.fetchOnline()
      const productIDs = nextClosesService.records.flatMap((closeOrder) =>
        (closeOrder.Lines || []).map((line) => line.ProductID))
      await productsService.syncIDs(productIDs)
      if (requestID === closesRequestID) closesService = nextClosesService
    } catch (loadError) {
      console.error('[daily-close] closes load error', { warehouseID, loadError })
      notifyFailure(String(loadError) || tr('Could not load the past closes.|No se pudieron cargar los cierres anteriores.'))
    }
  }

  // A new W resets S to it; a new S only re-previews.
  $effect(() => {
    const warehouseID = closeForm.WarehouseID
    untrack(() => {
      closeForm.SuppliesWarehouseID = warehouseID
      void loadWarehouseClose(warehouseID, warehouseID)
      void loadCloses(warehouseID)
    })
  })

  async function runClose() {
    if (!preview || !canRunClose(preview) || isClosing) return
    const closePreview = preview
    const confirmed = await confirmWarn({
      title: tr('Daily Close|Cierre del Día'),
      message: tr(`Close ${formatTime(closePreview.Date, 'd-M-Y')}? The supplies are consumed and the products enter and leave the warehouse. It cannot be undone.`
        + `|¿Cerrar el ${formatTime(closePreview.Date, 'd-M-Y')}? Se consumen los insumos y los productos ingresan y salen del almacén. No se puede deshacer.`),
    })
    if (!confirmed) return

    isClosing = true
    try {
      const savedClose = await postDailyClose(closePreview)
      notifySuccess(tr(`Close No. ${savedClose.ID} registered.|Cierre N° ${savedClose.ID} registrado.`))
    } catch (closeError) {
      console.error('[daily-close] close error', { closePreview, closeError })
      notifyFailure(String(closeError) || tr('Could not run the close.|No se pudo realizar el cierre.'))
    } finally {
      isClosing = false
    }
    // Either way the day changed: the next pending day, or the same one with a fresh preview.
    await loadWarehouseClose(closeForm.WarehouseID, closeForm.SuppliesWarehouseID)
    await loadCloses(closeForm.WarehouseID)
  }

  // Days after the oldest wait their turn, so they are shown muted.
  const laterDayCss = (day: IDailyClosePendingDay) => day.Date !== closableDay?.Date ? 'text-fg-muted' : 'ff-bold'

  const pendingDayColumns: ITableColumn<IDailyClosePendingDay>[] = [
    { header: 'Day|Día', css: 'px-6', setCellCss: laterDayCss, getValue: (day) => String(formatTime(day.Date, 'd-M-Y')) },
    { header: 'Sales|Ventas', headerCss: 'w-80', css: 'px-6 ff-mono', align: 'right', setCellCss: laterDayCss, getValue: (day) => day.SaleCount },
    { header: 'Units|Unidades', headerCss: 'w-90', css: 'px-6 ff-mono', align: 'right', setCellCss: laterDayCss, getValue: (day) => day.Units },
  ]

  const closeColumns: ITableColumn<IProductionOrder>[] = [
    { header: 'No.|N°', headerCss: 'w-70', css: 'text-center ff-bold', getValue: (closeOrder) => closeOrder.ID },
    { header: 'Day|Día', headerCss: 'w-110', css: 'px-6', getValue: (closeOrder) => String(formatTime(closeOrder.Date, 'd-M-Y')) },
    { header: 'Products|Productos', css: 'px-6', getValue: (closeOrder) => closeOrderProductUnits(closeOrder)
      .map((product) => `${product.Units}x ${productsService.recordsMap.get(product.ProductID)?.Name || `#${product.ProductID}`}`).join(', ') },
    { header: 'Sales|Ventas', headerCss: 'w-80', css: 'px-6 ff-mono', align: 'right',
      getValue: (closeOrder) => (closeOrder.SaleOrderIDs || []).length },
    { header: 'Value|Valor', headerCss: 'w-120', css: 'px-6 ff-mono', align: 'right',
      getValue: (closeOrder) => formatAmount(closeOrderTotalValue(closeOrder)) },
    { header: 'Supplies wh.|Alm. insumos', headerCss: 'w-150', css: 'px-6',
      getValue: (closeOrder) => warehousesService.AlmacenesMap.get(closeOrder.SuppliesWarehouseID)?.Name || '' },
  ]
</script>

<Page title="Daily Close|Cierre del Día">
  <Layer type="content">
    <div class="flex flex-wrap items-end gap-10 mb-12" aria-label="Warehouse of the sales and warehouse of the supplies">
      <SearchSelect saveOn={closeForm} save="WarehouseID" css="w-full md:w-280"
        options={warehousesService.Almacenes || []} keyId="ID" keyName="Name"
        label="Sales warehouse|Almacén de ventas" placeholder=":: seleccione ::" />
      <SearchSelect saveOn={closeForm} save="SuppliesWarehouseID" css="w-full md:w-280"
        options={warehousesService.Almacenes || []} keyId="ID" keyName="Name"
        label="Supplies warehouse|Almacén de insumos" placeholder=":: seleccione ::"
        onChange={() => { void loadWarehouseClose(closeForm.WarehouseID, closeForm.SuppliesWarehouseID) }} />
    </div>

    {#if !closeForm.WarehouseID}
      <div class="p-12 rounded-md bg-surface-soft text-fg-muted">
        {tr('Select the warehouse whose sales you close.|Seleccione el almacén cuyas ventas va a cerrar.')}
      </div>
    {:else}
      <div class="grid grid-cols-24 gap-16">
        <div class="col-span-24 md:col-span-7" aria-label="Days with sales pending the close, oldest first">
          <div class="h3 mb-6">{tr('Pending days|Días pendientes')}</div>
          <VTable columns={pendingDayColumns} data={pendingDays} disableVirtualizer={true} estimateSize={36}
            emptyMessage="No sales pending the close.|No hay ventas pendientes de cierre." />
        </div>
        <div class="col-span-24 md:col-span-17">
          {#if isLoading || isClosing}
            <div class="p-8 min-h-240 w-full fx-c rounded-md bg-surface-soft">
              <LoadingBar label={tr(isClosing ? 'Closing...|Cerrando...' : 'Loading...|Cargando...')} />
            </div>
          {:else if preview}
            <div class="flex items-center mb-10">
              <div class="h2">{tr(`Close of ${formatTime(preview.Date, 'd-M-Y')}|Cierre del ${formatTime(preview.Date, 'd-M-Y')}`)}</div>
              <Button color="green" icon="icon-[fa--check]" name="Accept and close|Aceptar y cerrar" css="ml-auto"
                label="Runs the close of this day: consumes the supplies and records the products' entry and exit."
                disabled={!canRunClose(preview)} onClick={runClose} />
            </div>
            <DailyCloseSummary {preview} {productsService} />
          {/if}
        </div>
      </div>

      <div class="mt-20" aria-label="Past closes of this warehouse">
        <div class="h3 mb-6">{tr('Past closes|Cierres anteriores')}</div>
        <VTable columns={closeColumns} data={closesService?.records || []} disableVirtualizer={true} estimateSize={36}
          emptyMessage="No closes yet.|Aún no hay cierres." />
      </div>
    {/if}
  </Layer>
</Page>
