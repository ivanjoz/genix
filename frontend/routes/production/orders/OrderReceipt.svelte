<script lang="ts">
  import { useUI } from '@genix/ui'
  import { untrack } from 'svelte'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Checkbox from '#components/form/Checkbox.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { postProductionOrderReceipt } from '#services/production/production-orders.svelte.ts'
  import type { IProductionOrder } from '#services/production/production-orders.ts'
  import {
    findReceiptError, makeReceiptEntryLines, proposeReceiptConsumptions, type IReceiptEntryConsumption, type IReceiptEntryLine,
  } from './production-order.ts'
  import type { ProductionCatalogs } from './production-order.svelte.ts'

  // A receipt of the order (docs/MANUFACTURING_PLAN.md §6.2): the good and scrapped units per size, then
  // the actual consumption of each supply, proposed from the plan and edited by the user. "Completar"
  // makes it the last one: it proposes what is left of each supply and finishes the order.
  let { order, catalogs, modalID, onSaved }: {
    order: IProductionOrder, catalogs: ProductionCatalogs, modalID: number, onSaved: (savedOrder: IProductionOrder) => void
  } = $props()

  const ui = useUI()
  // Mounted per order version ({#key order.ver}): the lines start from the order as it was opened.
  let receiptLines = $state<IReceiptEntryLine[]>(untrack(() => makeReceiptEntryLines(order.Lines || [])))
  let isCompleting = $state(false)
  // A typed consumption is kept until the units or the "Completar" box change the proposal again.
  let consumptions = $state<IReceiptEntryConsumption[]>([])
  $effect(() => {
    consumptions = proposeReceiptConsumptions(order.Supplies || [], order.Lines || [], receiptLines, isCompleting)
  })
  const integerOf = (value: string | number) => Math.max(parseInt(String(value || '0')) || 0, 0)

  const lineColumns: ITableColumn<IReceiptEntryLine>[] = [
    { header: 'Product|Producto', css: 'px-6', getValue: (line) => catalogs.productNameOf(line.ProductID) },
    { header: 'Size|Talla', width: '90px', css: 'px-6', getValue: (line) => catalogs.presentationNameOf(line.ProductID, line.PresentationID) },
    {
      header: 'Pending|Pendiente', width: '100px', align: 'right',
      getValue: (line) => {
        const orderLine = order.Lines.find((candidate) => candidate.ProductID === line.ProductID && candidate.PresentationID === line.PresentationID)
        return orderLine ? Math.max(orderLine.PlannedUnits - (orderLine.ProducedUnits || 0), 0) : 0
      },
    },
    {
      id: 'good', header: 'Good|Buenas', width: '100px', align: 'right', cellInputType: 'number',
      getValue: (line) => line.Units, onCellEdit: (line, value) => { line.Units = integerOf(value); receiptLines = [...receiptLines] },
    },
    {
      id: 'scrapped', header: 'Scrapped|Falladas', width: '100px', align: 'right', cellInputType: 'number',
      getValue: (line) => line.Scrapped, onCellEdit: (line, value) => { line.Scrapped = integerOf(value); receiptLines = [...receiptLines] },
    },
  ]

  // A supply's quantity is typed as units and sub-units of its divisor, like a recipe line.
  const consumptionColumns: ITableColumn<IReceiptEntryConsumption>[] = [
    { header: 'Supply|Insumo', css: 'px-6', getValue: (consumption) => catalogs.supplies.recordsMap.get(consumption.SupplyID)?.Name || `#${consumption.SupplyID}` },
    {
      id: 'units', header: 'Units|Unidades', width: '100px', align: 'right', cellInputType: 'number',
      getValue: (consumption) => Math.floor(consumption.SubUnits / consumption.SubDivisor),
      onCellEdit: (consumption, value) => {
        consumption.SubUnits = integerOf(value) * consumption.SubDivisor + consumption.SubUnits % consumption.SubDivisor
      },
    },
    {
      id: 'sub', header: 'Sub-units|Sub-unidades', width: '110px', align: 'right', cellInputType: 'number',
      getValue: (consumption) => consumption.SubUnits % consumption.SubDivisor,
      disableCellInteractions: (consumption) => consumption.SubDivisor <= 1,
      onCellEdit: (consumption, value) => {
        consumption.SubUnits = Math.floor(consumption.SubUnits / consumption.SubDivisor) * consumption.SubDivisor + integerOf(value)
      },
    },
    {
      header: 'Proposed|Propuesto', width: '150px', align: 'right',
      getValue: (consumption) => {
        const proposal = proposeReceiptConsumptions(order.Supplies || [], order.Lines || [], receiptLines, isCompleting)
          .find((proposed) => proposed.SupplyID === consumption.SupplyID)
        return catalogs.formatSupplyQuantity(consumption.SupplyID, proposal?.SubUnits || 0, consumption.SubDivisor)
      },
    },
  ]

  const saveReceipt = async () => {
    const receiptError = findReceiptError(receiptLines, consumptions)
    if (receiptError) {
      notifyFailure(tr(receiptError))
      return
    }
    const hasMaterialAtWorkshop = isCompleting && (order.Supplies || []).some((orderSupply) => (orderSupply.AtWorkshop || 0) > 0)
    const confirmed = await confirmWarn({
      title: tr('Receipt|Ingreso'),
      message: tr(isCompleting
        ? `The products enter stock and the order is finished.${hasMaterialAtWorkshop ? ' Material is still at the workshop.' : ''}|Los productos ingresan al stock y la orden se termina.${hasMaterialAtWorkshop ? ' Aún hay material en el taller.' : ''}`
        : 'The products enter stock and the supplies are consumed. It cannot be undone.|Los productos ingresan al stock y se consumen los insumos. No se puede deshacer.'),
    })
    if (!confirmed || !catalogs.orders) return
    showLoading(tr('Registering receipt...|Registrando ingreso...'))
    try {
      const savedOrder = await postProductionOrderReceipt(catalogs.orders, {
        OrderID: order.ID, Version: order.ver, Complete: isCompleting,
        Lines: $state.snapshot(receiptLines).filter((line) => line.Units > 0 || line.Scrapped > 0),
        Consumptions: $state.snapshot(consumptions).map((consumption) => ({ SupplyID: consumption.SupplyID, SubUnits: consumption.SubUnits })),
      })
      ui.closeModal(modalID)
      onSaved(savedOrder)
    } catch (receiptError) {
      console.error('[production-orders] receipt error', { orderID: order.ID, receiptError })
      notifyFailure(receiptError)
    }
    hideLoading()
  }
</script>

<Modal id={modalID} size={8} title={tr(`Receipt of order N° ${order.ID}|Ingreso de la orden N° ${order.ID}`)}
  bodyCss="px-16 py-14" onSave={saveReceipt} saveButtonLabel={tr('Register|Registrar')}>
  <div aria-label="Units received per product and size">
    <VTable columns={lineColumns} data={receiptLines} disableVirtualizer={true} estimateSize={40} />
  </div>
  <Checkbox css="mt-12" label="Complete: last receipt, finishes the order|Completar: último ingreso, termina la orden"
    checked={isCompleting} onToggle={(isChecked: boolean) => { isCompleting = isChecked }} />
  <h3 class="font-bold mt-16 mb-6">{tr('Actual consumption|Consumo real')}</h3>
  <div aria-label="Actual consumption per supply">
    <VTable columns={consumptionColumns} data={consumptions} disableVirtualizer={true} estimateSize={40}
      emptyMessage="The order has no supplies.|La orden no tiene insumos.">
      {#snippet cellRenderer(_consumption, _column, defaultContent)}
        {defaultContent}
      {/snippet}
    </VTable>
  </div>
</Modal>
