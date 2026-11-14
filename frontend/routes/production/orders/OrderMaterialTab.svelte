<script lang="ts">
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import { PRODUCTION_ORDER_STATUS, type IProductionOrder, type IProductionOrderSupply } from '#services/production/production-orders.ts'
  import { postProductionOrderMaterial } from '#services/production/production-orders.svelte.ts'
  import { materialToSend } from './production-order.ts'
  import type { ProductionCatalogs } from './production-order.svelte.ts'

  // The "Maquila" tab (docs/MANUFACTURING_PLAN.md §8, M35). A draft picks the workshop and its price per
  // unit (saved with the order). Once released, the supplies reserved in the company warehouse are sent
  // to the workshop's warehouse and the leftovers come back; receipts consume at the workshop.
  let { order = $bindable(), catalogs }: { order: IProductionOrder, catalogs: ProductionCatalogs } = $props()

  const isDraft = $derived(order.ss === PRODUCTION_ORDER_STATUS.DRAFT)
  const canMoveMaterial = $derived(order.SubcontractorID > 0 &&
    (order.ss === PRODUCTION_ORDER_STATUS.RELEASED || order.ss === PRODUCTION_ORDER_STATUS.IN_PROGRESS))
  // Only a provider with its own warehouse can be a workshop: the material goes there.
  const workshopOptions = $derived((catalogs.workshops?.records || []).filter((provider) =>
    (catalogs.warehouses.Almacenes || []).some((warehouse) => warehouse.ss > 0 && warehouse.SubcontractorID === provider.ID)))

  // What to move per supply, in sub-units; prefilled with what is still to send.
  let toMoveBySupplyID = $state(new Map<number, number>())
  $effect(() => {
    const supplies = order.Supplies
    toMoveBySupplyID = new Map((supplies || []).map((orderSupply) => [orderSupply.SupplyID, materialToSend(orderSupply)]))
  })
  const formatCents = (cents: number) => formatN((cents || 0) / 100, 2)
  const agreedTotal = $derived((order.Lines || []).reduce((total, line) => total + line.PlannedUnits, 0) * (order.SubcontractPricePerUnit || 0))

  const moveMaterial = async (isReturn: boolean) => {
    const lines = (order.Supplies || [])
      .map((orderSupply) => ({ SupplyID: orderSupply.SupplyID, SubUnits: toMoveBySupplyID.get(orderSupply.SupplyID) || 0 }))
      .filter((line) => line.SubUnits > 0)
    if (!lines.length) {
      notifyFailure(tr('Enter the quantities to move.|Ingrese las cantidades a trasladar.'))
      return
    }
    const confirmed = await confirmWarn({
      title: tr(isReturn ? 'Return leftovers|Devolver sobrante' : 'Send materials|Enviar materiales'),
      message: tr(isReturn
        ? 'The material comes back from the workshop to the supplies warehouse.|El material vuelve del taller al almacén de insumos.'
        : `The material goes to the workshop "${catalogs.workshopNameOf(order.SubcontractorID)}".|El material va al taller "${catalogs.workshopNameOf(order.SubcontractorID)}".`),
    })
    if (!confirmed || !catalogs.orders) return
    showLoading(tr('Moving material...|Trasladando material...'))
    try {
      order = await postProductionOrderMaterial(catalogs.orders, order.ID, isReturn, lines)
    } catch (moveError) {
      console.error('[production-orders] material error', { orderID: order.ID, isReturn, lines, moveError })
      notifyFailure(moveError)
    }
    hideLoading()
  }

  const supplyAmount = (orderSupply: IProductionOrderSupply, subUnits: number) =>
    catalogs.formatSupplyQuantity(orderSupply.SupplyID, subUnits || 0, orderSupply.SubDivisor)

  const columns: ITableColumn<IProductionOrderSupply>[] = [
    { header: 'Supply|Insumo', css: 'px-6', getValue: (orderSupply) => catalogs.supplies.recordsMap.get(orderSupply.SupplyID)?.Name || `#${orderSupply.SupplyID}` },
    { header: 'Planned|Planificado', width: '130px', align: 'right', getValue: (orderSupply) => supplyAmount(orderSupply, orderSupply.Planned) },
    { header: 'Reserved|Reservado', width: '130px', align: 'right', getValue: (orderSupply) => supplyAmount(orderSupply, orderSupply.Reserved) },
    { header: 'At workshop|En taller', width: '130px', align: 'right', getValue: (orderSupply) => supplyAmount(orderSupply, orderSupply.AtWorkshop || 0) },
    { header: 'Consumed|Consumido', width: '130px', align: 'right', getValue: (orderSupply) => supplyAmount(orderSupply, orderSupply.Consumed) },
    {
      id: 'toMove', header: 'To move (sub-units)|A trasladar (sub-unid.)', width: '150px', align: 'right', cellInputType: 'number',
      getValue: (orderSupply) => toMoveBySupplyID.get(orderSupply.SupplyID) || 0,
      disableCellInteractions: () => !canMoveMaterial,
      onCellEdit: (orderSupply, value) => {
        toMoveBySupplyID.set(orderSupply.SupplyID, Math.max(parseInt(String(value || '0')) || 0, 0))
        toMoveBySupplyID = new Map(toMoveBySupplyID)
      },
    },
  ]
</script>

{#if isDraft}
  <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Workshop of a subcontracted order">
    <SearchSelect bind:saveOn={order} save="SubcontractorID" options={workshopOptions} keyId="ID" keyName="Name"
      label="Workshop (empty = made in-house)|Taller (vacío = fabricación propia)" css="col-span-24 md:col-span-16" />
    {#if order.SubcontractorID > 0}
      <Input bind:saveOn={order} save="SubcontractPricePerUnit" type="number" baseDecimals={2}
        label="Price per unit (net of IGV)|Precio por unidad (sin IGV)" css="col-span-24 md:col-span-8" />
    {/if}
    <div class="col-span-24 text-fg-muted">
      {tr('Only providers with a workshop warehouse are offered (Branches & warehouses).|Solo se ofrecen proveedores con un almacén de taller (Sedes y almacenes).')}
    </div>
  </div>
{:else if order.SubcontractorID > 0}
  <div class="flex flex-wrap items-center gap-16 mt-8 p-12 rounded-md bg-surface-soft" aria-label="Workshop and agreed price">
    <div><span class="text-fg-muted">{tr('Workshop|Taller')}:</span> <span class="ff-bold">{catalogs.workshopNameOf(order.SubcontractorID)}</span></div>
    <div><span class="text-fg-muted">{tr('Price / unit|Precio / unidad')}:</span> <span class="ff-mono">{formatCents(order.SubcontractPricePerUnit)}</span></div>
    <div><span class="text-fg-muted">{tr('Agreed for the plan|Acordado por el plan')}:</span> <span class="ff-mono ff-bold">{formatCents(agreedTotal)}</span></div>
  </div>
  <div class="flex items-center justify-end gap-8 mt-12 mb-6">
    {#if canMoveMaterial}
      <Button color="blue" icon="icon-[fa--truck]" name="Send materials|Enviar materiales" label="Sends the quantities to the workshop."
        onClick={() => moveMaterial(false)} />
      <Button color="orange" icon="icon-[fa--undo]" name="Return leftovers|Devolver sobrante" label="Brings the quantities back from the workshop."
        onClick={() => moveMaterial(true)} />
    {/if}
  </div>
  <div aria-label="Supplies at the workshop">
    <VTable {columns} data={order.Supplies || []} disableVirtualizer={true} estimateSize={40}
      emptyMessage="The order has no supplies.|La orden no tiene insumos." />
  </div>
{:else}
  <div class="mt-8 text-fg-muted">{tr('This order is made in-house.|Esta orden se fabrica en la empresa.')}</div>
{/if}
