<script lang="ts">
  import { useUI } from '@genix/ui'
  import { untrack } from 'svelte'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import {
    postProductionOrder, postProductionOrderAnnul, postProductionOrderComplete, postProductionOrderRelease, postProductionTemplate,
  } from '#services/production/production-orders.svelte.ts'
  import {
    PRODUCTION_ORDER_STATUS, type IProductionOrder, type IProductionOrderSupply, type IProductionReceipt,
  } from '#services/production/production-orders.ts'
  import OrderLaborTab from './OrderLaborTab.svelte'
  import OrderMaterialTab from './OrderMaterialTab.svelte'
  import OrderReceipt from './OrderReceipt.svelte'
  import ProductSizeGrid from './ProductSizeGrid.svelte'
  import {
    ORDER_STATUS_LABELS, findOrderDraftError, orderLinesToProductRuns, orderLinesToTemplateLines, productRunsToOrderLines,
    summarizeOrderLabor, summarizeOrderProducts, summarizeOrderSupplies,
  } from './production-order.ts'
  import type { ProductionCatalogs } from './production-order.svelte.ts'

  // The detail of a production order (docs/MANUFACTURING_PLAN.md §12): a draft is edited and released;
  // a released order takes receipts, labor and workshop shipments; the Resumen tab compares the plan
  // with what happened.
  let { order = $bindable(), catalogs, layerID }: { order: IProductionOrder, catalogs: ProductionCatalogs, layerID: number } = $props()

  const ui = useUI()
  const RECEIPT_MODAL_ID = 41
  let layerView = $state(1)
  let isReceiptOpen = $state(false)
  // The size grid edits runs; the order keeps lines. The page mounts this layer anew on every open
  // ({#key}), so the grid starts from the order being opened, a new one from a template included.
  let productRuns = $state(untrack(() => orderLinesToProductRuns(order.Lines || [])))

  const isDraft = $derived(order.ss === PRODUCTION_ORDER_STATUS.DRAFT)
  const isOpen = $derived(order.ss === PRODUCTION_ORDER_STATUS.RELEASED || order.ss === PRODUCTION_ORDER_STATUS.IN_PROGRESS)
  const formatCents = (cents: number | undefined) => cents === undefined ? '—' : formatN(cents / 100, 2)
  const formatPercent = (percent: number | undefined) => percent === undefined ? '—' : `${formatN(percent, 0)}%`
  const supplyNameOf = (supplyID: number) => catalogs.supplies.recordsMap.get(supplyID)?.Name || `#${supplyID}`
  const supplyAmount = (supplyID: number, subUnits: number, subDivisor: number) =>
    catalogs.formatSupplyQuantity(supplyID, subUnits || 0, subDivisor)

  // The supply adjustment is typed as a percentage; the order keeps basis points.
  const adjustmentForm = $derived({ percent: (order.SupplyAdjustmentBasisPoints || 0) / 100 })

  // A finished request answers the saved order: it replaces the form, so `ver` stays current.
  const runOrderAction = async (loadingText: string, action: () => Promise<IProductionOrder>) => {
    showLoading(tr(loadingText))
    try {
      order = await action()
      productRuns = orderLinesToProductRuns(order.Lines || [])
    } catch (actionError) {
      console.error('[production-orders] action error', { orderID: order.ID, actionError })
      notifyFailure(actionError)
      hideLoading()
      return false
    }
    hideLoading()
    return true
  }

  const saveDraft = async () => {
    const lines = productRunsToOrderLines($state.snapshot(productRuns)).filter((line) => line.PlannedUnits > 0)
    const draftError = findOrderDraftError(order, lines)
    if (draftError) {
      notifyFailure(tr(draftError))
      return false
    }
    if (!catalogs.orders) return false
    const orders = catalogs.orders
    return runOrderAction('Saving order...|Guardando orden...', () => postProductionOrder(orders, { ...$state.snapshot(order), Lines: lines }))
  }

  const releaseOrder = async () => {
    if (!(await confirmWarn({
      title: tr('Release|Liberar'),
      message: tr('The recipes are frozen on the order and its supplies are reserved.|Las recetas se congelan en la orden y se reservan sus insumos.'),
    }))) return
    if (!(await saveDraft()) || !catalogs.orders) return
    const orders = catalogs.orders
    const isReleased = await runOrderAction('Releasing order...|Liberando orden...', () => postProductionOrderRelease(orders, order.ID))
    if (!isReleased) {
      // A shortage refuses the release: the requirements page says what to buy.
      notifyFailure(tr(`See what to buy in Purchase requirements (/production/requirements?order-id=${order.ID}).|Revise qué comprar en Necesidades de compra (/production/requirements?order-id=${order.ID}).`))
    }
  }

  const completeOrder = async () => {
    const hasMaterialAtWorkshop = (order.Supplies || []).some((orderSupply) => (orderSupply.AtWorkshop || 0) > 0)
    if (!(await confirmWarn({
      title: tr('Complete|Completar'),
      message: tr(`The order is finished and what it still reserves is released.${hasMaterialAtWorkshop ? ' Material is still at the workshop.' : ''}|La orden se termina y se libera lo que aún reserva.${hasMaterialAtWorkshop ? ' Aún hay material en el taller.' : ''}`),
    })) || !catalogs.orders) return
    const orders = catalogs.orders
    await runOrderAction('Completing order...|Completando orden...', () => postProductionOrderComplete(orders, order.ID))
  }

  const annulOrder = async () => {
    if (!(await confirmWarn({ title: tr('Annul|Anular'), message: tr('Annul the order? Its reservations are released.|¿Anular la orden? Se liberan sus reservas.') }))
      || !catalogs.orders) return
    const orders = catalogs.orders
    if (await runOrderAction('Annulling order...|Anulando orden...', () => postProductionOrderAnnul(orders, order.ID))) ui.openSideLayer(0)
  }

  // "Guardar como plantilla": the order's units ÷ its series become one series of a new template.
  const saveAsTemplate = async () => {
    const lines = productRunsToOrderLines($state.snapshot(productRuns)).filter((line) => line.PlannedUnits > 0)
    const series = Math.max(order.Series || 1, 1)
    showLoading(tr('Saving template...|Guardando plantilla...'))
    try {
      await postProductionTemplate(catalogs.templates, {
        ID: 0, Name: tr(`From order N° ${order.ID}|Desde orden N° ${order.ID}`), Lines: orderLinesToTemplateLines(lines, series),
        DefaultSeries: series, WarehouseID: order.WarehouseID, SuppliesWarehouseID: order.SuppliesWarehouseID, Notes: order.Notes, ss: 1, upd: 0,
      })
    } catch (templateError) {
      notifyFailure(templateError)
    }
    hideLoading()
  }

  const supplyColumns: ITableColumn<IProductionOrderSupply>[] = [
    { header: 'Supply|Insumo', css: 'px-6', getValue: (orderSupply) => supplyNameOf(orderSupply.SupplyID) },
    { header: 'Planned|Planificado', width: '140px', align: 'right', getValue: (orderSupply) => supplyAmount(orderSupply.SupplyID, orderSupply.Planned, orderSupply.SubDivisor) },
    { header: 'Available|Disponible', width: '140px', align: 'right', getValue: (orderSupply) => supplyAmount(orderSupply.SupplyID, orderSupply.Available, orderSupply.SubDivisor) },
    { header: 'Consumed|Consumido', width: '140px', align: 'right', getValue: (orderSupply) => supplyAmount(orderSupply.SupplyID, orderSupply.Consumed, orderSupply.SubDivisor) },
    { header: 'Reserved|Reservado', width: '140px', align: 'right', getValue: (orderSupply) => supplyAmount(orderSupply.SupplyID, orderSupply.Reserved, orderSupply.SubDivisor) },
  ]

  const receiptColumns: ITableColumn<IProductionReceipt>[] = [
    { header: 'N°', width: '50px', align: 'center', getValue: (receipt) => receipt.N },
    { header: 'Day|Día', width: '110px', getValue: (receipt) => String(formatTime(receipt.Date, 'd-M-Y')) },
    { header: 'Good|Buenas', width: '80px', align: 'right', getValue: (receipt) => (receipt.Lines || []).reduce((total, line) => total + line.Units, 0) },
    { header: 'Scrapped|Falladas', width: '90px', align: 'right', getValue: (receipt) => (receipt.Lines || []).reduce((total, line) => total + (line.Scrapped || 0), 0) },
    { header: 'Supplies|Insumos', width: '110px', align: 'right', getValue: (receipt) => formatCents(receipt.SuppliesValue || 0) },
    { header: 'Labor|M. de obra', width: '110px', align: 'right', getValue: (receipt) => formatCents(receipt.LaborValue || 0) },
    { header: 'Workshop|Maquila', width: '110px', align: 'right', getValue: (receipt) => formatCents(receipt.SubcontractValue || 0) },
    { header: 'Overhead|Indirectos', width: '110px', align: 'right', getValue: (receipt) => formatCents(receipt.OverheadValue || 0) },
    {
      header: 'Value|Valor', width: '120px', align: 'right', css: 'ff-bold',
      getValue: (receipt) => formatCents((receipt.Lines || []).reduce((total, line) => total + (line.Value || 0), 0)),
    },
  ]

  const estimatedUnitCostByProductID = $derived(new Map((order.Lines || []).map((line) =>
    [line.ProductID, catalogs.estimatedUnitCostOf(line.ProductID) ?? 0])))
  const productSummaryColumns: ITableColumn<ReturnType<typeof summarizeOrderProducts>[number]>[] = [
    { header: 'Product|Producto', css: 'px-6', getValue: (row) => catalogs.productNameOf(row.ProductID) },
    { header: 'Planned|Plan.', width: '80px', align: 'right', getValue: (row) => row.PlannedUnits },
    { header: 'Good|Buenas', width: '80px', align: 'right', getValue: (row) => row.ProducedUnits },
    { header: 'Scrapped|Falladas', width: '90px', align: 'right', getValue: (row) => row.ScrappedUnits },
    { header: 'Yield|Rendim.', width: '80px', align: 'right', getValue: (row) => formatPercent(row.yieldPercent) },
    { header: 'Est. unit cost|Costo est. unit.', width: '130px', align: 'right', getValue: (row) => formatCents(row.estimatedUnitCost) },
    { header: 'Booked unit cost|Costo real unit.', width: '130px', align: 'right', getValue: (row) => formatCents(row.bookedUnitCost) },
  ]
  const supplySummaryColumns: ITableColumn<ReturnType<typeof summarizeOrderSupplies>[number]>[] = [
    { header: 'Supply|Insumo', css: 'px-6', getValue: (row) => supplyNameOf(row.SupplyID) },
    { header: 'Planned|Planificado', width: '140px', align: 'right', getValue: (row) => supplyAmount(row.SupplyID, row.Planned, row.SubDivisor) },
    { header: 'For what was made|Para lo fabricado', width: '160px', align: 'right', getValue: (row) => supplyAmount(row.SupplyID, row.expectedForOutput, row.SubDivisor) },
    { header: 'Consumed|Consumido', width: '140px', align: 'right', getValue: (row) => supplyAmount(row.SupplyID, row.Consumed, row.SubDivisor) },
    { header: 'Usage|Uso', width: '80px', align: 'right', getValue: (row) => formatPercent(row.usagePercent) },
  ]
  const laborSummaryColumns: ITableColumn<ReturnType<typeof summarizeOrderLabor>[number]>[] = [
    { header: 'Labor type|Tipo de trabajo', css: 'px-6', getValue: (row) => catalogs.laborTypes.recordsMap.get(row.LaborTypeID)?.Name || `#${row.LaborTypeID}` },
    { header: 'Planned min|Min. plan.', width: '110px', align: 'right', getValue: (row) => row.PlannedMinutes },
    { header: 'Logged min|Min. reg.', width: '110px', align: 'right', getValue: (row) => row.RecordedMinutes || 0 },
    { header: 'Minutes|Minutos', width: '90px', align: 'right', getValue: (row) => formatPercent(row.minutesPercent) },
    { header: 'Planned cost|Costo plan.', width: '120px', align: 'right', getValue: (row) => formatCents(row.PlannedCost) },
    { header: 'Booked cost|Costo cargado', width: '120px', align: 'right', getValue: (row) => formatCents(row.RecordedCost || 0) },
  ]
</script>

<Layer
  id={layerID}
  type="side"
  sideLayerSize={1000}
  title={order.ID > 0 ? `${tr('Order|Orden')} N° ${order.ID} · ${tr(ORDER_STATUS_LABELS[order.ss] || '')}` : 'New production order|Nueva orden de producción'}
  titleCss="h2 mb-6"
  css="px-8 py-8 md:px-16 md:py-10"
  contentCss="px-0 md:px-0"
  bind:selected={layerView}
  onSave={isDraft ? () => { void saveDraft() } : undefined}
  onClose={() => { layerView = 1 }}
  options={[
    [1, 'Products|Productos'],
    ...(isDraft ? [] : [[2, 'Supplies|Insumos'], [3, 'Labor|Mano de obra']] as [number, string][]),
    ...(isDraft || order.SubcontractorID > 0 ? [[4, 'Subcontracting|Maquila']] as [number, string][] : []),
    ...(isDraft ? [] : [[5, 'Receipts|Ingresos'], [6, 'Summary|Resumen']] as [number, string][]),
  ]}
>
  <div class="flex flex-wrap items-center justify-end gap-8 mb-10" aria-label="Order actions">
    {#if order.ID > 0}
      <Button color="blue" mode="subtle" icon="icon-[fa--clone]" name="Save as template|Guardar como plantilla"
        label="Saves the order's products as a new template." onClick={saveAsTemplate} />
    {/if}
    {#if isDraft && order.ID > 0}
      <Button color="red" mode="subtle" icon="icon-[fa--ban]" name="Annul|Anular" label="Annuls the draft." onClick={annulOrder} />
      <Button color="green" icon="icon-[fa--play]" name="Release|Liberar" label="Freezes the recipes and reserves the supplies." onClick={releaseOrder} />
    {/if}
    {#if order.ss === PRODUCTION_ORDER_STATUS.RELEASED && !(order.Receipts || []).length}
      <Button color="red" mode="subtle" icon="icon-[fa--ban]" name="Annul|Anular" label="Annuls the order and releases its reservations." onClick={annulOrder} />
    {/if}
    {#if isOpen}
      <Button color="green" icon="icon-[fa--sign-in]" name="Receive|Ingresar" label="Registers a receipt of produced units."
        onClick={() => { isReceiptOpen = true; ui.openModal(RECEIPT_MODAL_ID) }} />
    {/if}
    {#if order.ss === PRODUCTION_ORDER_STATUS.IN_PROGRESS}
      <Button color="orange" icon="icon-[fa--flag-checkered]" name="Complete|Completar" label="Finishes the order without another receipt." onClick={completeOrder} />
    {/if}
  </div>

  {#if layerView === 1}
    <div class="grid grid-cols-24 gap-10 mb-12" aria-label="Order warehouses, dates, stage and supply adjustment">
      <SearchSelect bind:saveOn={order} save="WarehouseID" options={catalogs.warehouses.Almacenes || []} keyId="ID" keyName="Name"
        label="Receiving warehouse|Almacén de ingreso" disabled={!isDraft} css="col-span-24 md:col-span-8" />
      <SearchSelect bind:saveOn={order} save="SuppliesWarehouseID" options={catalogs.warehouses.Almacenes || []} keyId="ID" keyName="Name"
        label="Supplies warehouse|Almacén de insumos" disabled={!isDraft} css="col-span-24 md:col-span-8" />
      <SearchSelect bind:saveOn={order} save="StageID" options={catalogs.stages.records} keyId="ID" keyName="Name"
        label="Stage|Etapa" disabled={!isDraft} css="col-span-24 md:col-span-8" />
      <DateInput bind:saveOn={order} save="PlannedStart" label="Planned start|Inicio planificado" disabled={!isDraft} css="col-span-12 md:col-span-6" />
      <DateInput bind:saveOn={order} save="PlannedEnd" label="Planned end|Fin planificado" disabled={!isDraft} css="col-span-12 md:col-span-6" />
      <Input saveOn={adjustmentForm} save="percent" type="number" baseDecimals={2} disabled={!isDraft}
        label="Supply adjustment %|Ajuste de insumos %" css="col-span-12 md:col-span-6"
        onChange={() => { order.SupplyAdjustmentBasisPoints = Math.round((adjustmentForm.percent || 0) * 100) }} />
      <Input bind:saveOn={order} save="Series" type="number" label="Series|Series" disabled={!isDraft} css="col-span-12 md:col-span-6" />
      <Input bind:saveOn={order} save="Notes" label="Notes|Notas" disabled={!isDraft} css="col-span-24" />
    </div>
    <ProductSizeGrid bind:productRuns {catalogs} isEditable={isDraft} />
  {:else if layerView === 2}
    <div aria-label="Supplies frozen on the order">
      <VTable columns={supplyColumns} data={order.Supplies || []} disableVirtualizer={true} estimateSize={36}
        emptyMessage="The order has no supplies.|La orden no tiene insumos." />
    </div>
  {:else if layerView === 3}
    {#key order.ID}
      <OrderLaborTab {order} {catalogs} />
    {/key}
  {:else if layerView === 4}
    <OrderMaterialTab bind:order {catalogs} />
  {:else if layerView === 5}
    <div aria-label="Receipts of the order">
      <VTable columns={receiptColumns} data={order.Receipts || []} disableVirtualizer={true} estimateSize={36}
        emptyMessage="No receipts yet.|Aún no hay ingresos." />
    </div>
  {:else if layerView === 6}
    <h3 class="font-bold mb-6">{tr('Products|Productos')}</h3>
    <VTable columns={productSummaryColumns} data={summarizeOrderProducts(order.Lines || [], estimatedUnitCostByProductID)}
      disableVirtualizer={true} estimateSize={36} />
    <h3 class="font-bold mt-16 mb-6">{tr('Supplies|Insumos')}</h3>
    <VTable columns={supplySummaryColumns} data={summarizeOrderSupplies(order)} disableVirtualizer={true} estimateSize={36}
      emptyMessage="The order has no supplies.|La orden no tiene insumos." />
    <h3 class="font-bold mt-16 mb-6">{tr('Labor|Mano de obra')}</h3>
    <VTable columns={laborSummaryColumns} data={summarizeOrderLabor(order.Labor || [])} disableVirtualizer={true} estimateSize={36}
      emptyMessage="The recipes plan no labor.|Las recetas no planifican mano de obra." />
  {/if}
</Layer>

{#if isReceiptOpen && isOpen}
  {#key order.ver}
    <OrderReceipt {order} {catalogs} modalID={RECEIPT_MODAL_ID}
      onSaved={(savedOrder) => { order = savedOrder; isReceiptOpen = false }} />
  {/key}
{/if}
