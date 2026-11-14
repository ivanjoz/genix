<script lang="ts">
  import { page } from '$app/state'
  import { confirmWarn, notifyFailure, notifySuccess } from '@genix/ui/notify'
  import { untrack } from 'svelte'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import { unitsByProduct } from '#routes/production/orders/production-order.ts'
  import type { IProductionOrder } from '#services/production/production-orders.ts'
  import { findPurchaseSuggestionsError, lastSubcontractOf, providerPriceOf, type IPurchaseSuggestion, type ISupplyRequirement } from './requirements.ts'
  import { RequirementsState } from './requirements.svelte.ts'

  const requirementsState = new RequirementsState()
  const { catalogs } = requirementsState

  const formatAmount = (cents: number) => `S/ ${formatN((cents || 0) / 100, 2)}`
  const supplyNameOf = (supplyID: number) => catalogs.supplies.recordsMap.get(supplyID)?.Name || `#${supplyID}`
  const formatSubUnits = (requirement: ISupplyRequirement, subUnits: number) =>
    subUnits ? catalogs.formatSupplyQuantity(requirement.SupplyID, Math.round(subUnits), requirement.SubDivisor) : ''
  const formatWholeUnits = (supplyID: number, units: number) => {
    const subDivisor = Math.max(catalogs.supplies.recordsMap.get(supplyID)?.SbuQuantity || 1, 1)
    return catalogs.formatSupplyQuantity(supplyID, units * subDivisor, subDivisor)
  }

  // "?order-id=" comes from a draft whose release was refused for short supplies: it is picked, and
  // its supplies warehouse is the one netted, once the drafts arrive.
  const requestedOrderID = Number(page.url.searchParams.get('order-id')) || 0
  let hasAppliedRequestedOrder = false
  $effect(() => {
    const requestedOrder = requirementsState.draftOrders.find((order) => order.ID === requestedOrderID)
    if (!requestedOrder || hasAppliedRequestedOrder) return
    untrack(() => {
      hasAppliedRequestedOrder = true
      requirementsState.selectedOrderIDs.add(requestedOrder.ID)
      if (!requirementsState.form.WarehouseID) requirementsState.form.WarehouseID = requestedOrder.SuppliesWarehouseID
    })
  })

  async function createPurchaseOrders() {
    const suggestionsError = findPurchaseSuggestionsError(requirementsState.purchaseSuggestions)
    if (suggestionsError) { notifyFailure(tr(suggestionsError)); return }
    const providerCount = new Set(requirementsState.purchaseSuggestions.filter((suggestion) => suggestion.Units > 0)
      .map((suggestion) => suggestion.ProviderID)).size
    const confirmed = await confirmWarn({
      title: tr('Create purchase orders|Crear órdenes de compra'),
      message: tr(`${providerCount} pending purchase order(s) will be created, one per provider.|Se crearán ${providerCount} orden(es) de compra pendientes, una por proveedor.`),
    })
    if (!confirmed) return
    try {
      const createdOrderIDs = await requirementsState.createPurchaseOrders()
      notifySuccess(tr(`Purchase orders created: ${createdOrderIDs.join(', ')}|Órdenes de compra creadas: ${createdOrderIDs.join(', ')}`))
    } catch (createError) {
      notifyFailure(String(createError))
    }
  }

  async function createSemiFinishedDrafts() {
    const draftCount = requirementsState.semiFinishedShortfalls.length
    const confirmed = await confirmWarn({
      title: tr('Create production orders|Crear órdenes de producción'),
      message: tr(`${draftCount} draft production order(s) will be created.|Se crearán ${draftCount} orden(es) de producción en borrador.`),
    })
    if (!confirmed) return
    try {
      const createdOrderIDs = await requirementsState.createSemiFinishedDrafts()
      notifySuccess(tr(`Draft orders created: ${createdOrderIDs.join(', ')}|Órdenes en borrador creadas: ${createdOrderIDs.join(', ')}`))
    } catch (createError) {
      notifyFailure(String(createError))
    }
  }

  const draftOrderColumns: ITableColumn<IProductionOrder>[] = [
    { header: 'No.|N°', headerCss: 'w-70', css: 'text-center ff-bold', getValue: (order) => order.ID },
    { header: 'Date|Fecha', headerCss: 'w-110', css: 'px-6', getValue: (order) => String(formatTime(order.Date, 'd-M-Y')) },
    { header: 'Products|Productos', css: 'px-6', useLineClamp: true, getValue: (order) => unitsByProduct(order.Lines || [])
      .map((product) => `${product.Units}x ${catalogs.productNameOf(product.ProductID)}`).join(', ') },
    { header: 'Supplies wh.|Alm. insumos', headerCss: 'w-150', css: 'px-6', getValue: (order) => catalogs.warehouseNameOf(order.SuppliesWarehouseID) },
  ]

  const requirementColumns: ITableColumn<ISupplyRequirement>[] = [
    { header: 'Supply|Insumo', css: 'px-6', getValue: (requirement) => supplyNameOf(requirement.SupplyID),
      renderPrefix: (requirement) => requirement.IsSemiFinished ? '<i class="icon-[fa--cogs] text-fg-muted mr-6"></i>' : false },
    { header: 'Level|Nivel', headerCss: 'w-60', css: 'text-center', getValue: (requirement) => requirement.Level },
    { header: 'Need|Necesidad', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right', getValue: (requirement) => formatSubUnits(requirement, requirement.Need) },
    { header: 'Minimum|Mínimo', headerCss: 'w-110', css: 'px-6 ff-mono', align: 'right', getValue: (requirement) => formatSubUnits(requirement, requirement.MinimumStock) },
    { header: 'Stock', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right', getValue: (requirement) => formatSubUnits(requirement, requirement.Stock) },
    { header: 'Reserved|Reservado', headerCss: 'w-120', css: 'px-6 ff-mono', align: 'right', getValue: (requirement) => formatSubUnits(requirement, requirement.Reserved) },
    { header: 'Blocked|Bloqueado', headerCss: 'w-120', css: 'px-6 ff-mono', align: 'right', getValue: (requirement) => formatSubUnits(requirement, requirement.Blocked) },
    { header: 'On order|Por llegar', headerCss: 'w-120', css: 'px-6 ff-mono', align: 'right', getValue: (requirement) => formatSubUnits(requirement, requirement.OnOrder) },
    { header: 'Shortfall|Faltante', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right',
      setCellCss: (requirement) => requirement.Shortfall > 0 ? 'text-red-solid ff-bold' : 'text-fg-muted',
      getValue: (requirement) => formatSubUnits(requirement, requirement.Shortfall) || '0' },
  ]

  const purchaseColumns: ITableColumn<IPurchaseSuggestion>[] = [
    { header: 'Supply|Insumo', css: 'px-6', getValue: (suggestion) => supplyNameOf(suggestion.SupplyID) },
    { header: 'Units|Unidades', headerCss: 'w-110', css: 'px-6 ff-mono', align: 'right', cellInputType: 'number',
      getValue: (suggestion) => suggestion.Units,
      onCellEdit: (suggestion, value, rerender) => { suggestion.Units = Math.max(Math.floor(Number(value) || 0), 0); rerender() } },
    { header: 'Provider|Proveedor', headerCss: 'w-240', css: 'px-6', field: 'ProviderID',
      getCellOptions: () => catalogs.workshops?.records || [], cellOptionsKeyId: 'ID', cellOptionsKeyName: 'Name',
      getValue: (suggestion) => suggestion.ProviderID ? catalogs.workshopNameOf(suggestion.ProviderID) : '',
      // Another provider brings its own price for the supply, when it has one on file.
      onCellSelect: (suggestion, value, rerender) => {
        suggestion.ProviderID = Number(value) || 0
        suggestion.Price = providerPriceOf(requirementsState.providerOffersBySupplyID.get(suggestion.SupplyID), suggestion.ProviderID) || suggestion.Price
        rerender()
      } },
    { header: 'Price|Precio', headerCss: 'w-110', css: 'px-6 ff-mono', align: 'right', cellInputType: 'number',
      getValue: (suggestion) => (suggestion.Price || 0) / 100,
      render: (suggestion) => formatN((suggestion.Price || 0) / 100, 2),
      onCellEdit: (suggestion, value, rerender) => { suggestion.Price = Math.max(Math.round((Number(value) || 0) * 100), 0); rerender() } },
    { header: 'Total', headerCss: 'w-120', css: 'px-6 ff-mono', align: 'right',
      getValue: (suggestion) => formatAmount(suggestion.Units * suggestion.Price) },
  ]

  const makeColumns: ITableColumn<ISupplyRequirement>[] = [
    { header: 'Semi-finished|Semielaborado', css: 'px-6', getValue: (requirement) => supplyNameOf(requirement.SupplyID) },
    { header: 'Units|Unidades', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right',
      getValue: (requirement) => formatWholeUnits(requirement.SupplyID, requirement.ShortfallUnits) },
    { header: 'Workshop|Taller', headerCss: 'w-240', css: 'px-6', getValue: (requirement) => {
      const lastSubcontract = lastSubcontractOf(requirement.SupplyID, catalogs.orders?.records || [])
      return lastSubcontract ? catalogs.workshopNameOf(lastSubcontract.SubcontractorID) : tr('In-house|Propio')
    } },
  ]
</script>

<Page title="Purchase requirements|Necesidades de compra">
  <Layer type="content">
    <div class="flex flex-wrap items-end gap-10 mb-12" aria-label="Supplies warehouse, template and series to plan">
      <SearchSelect saveOn={requirementsState.form} save="WarehouseID" css="w-full md:w-260"
        options={catalogs.warehouses.Almacenes || []} keyId="ID" keyName="Name"
        label="Supplies warehouse|Almacén de insumos" placeholder=":: seleccione ::" />
      <SearchSelect saveOn={requirementsState.form} save="TemplateID" css="w-full md:w-260"
        options={catalogs.templates.records} keyId="ID" keyName="Name"
        label="Template (optional)|Plantilla (opcional)" placeholder=":: ninguna ::" />
      <Input saveOn={requirementsState.form} save="Series" type="number" label="Series" css="w-120" />
      <Checkbox saveOn={requirementsState.form} save="ReplenishToMinimum" css="mb-6"
        label="Replenish to the minimum|Reponer al mínimo" />
    </div>

    <div class="grid grid-cols-24 gap-16">
      <div class="col-span-24 lg:col-span-9" aria-label="Draft production orders to plan">
        <div class="h3 mb-6">{tr('Draft orders|Órdenes en borrador')}</div>
        <VTable columns={draftOrderColumns} data={requirementsState.draftOrders} disableVirtualizer={true} estimateSize={36}
          selectedIDs={requirementsState.selectedOrderIDs} getRowID={(order) => order.ID}
          emptyMessage="No draft orders.|No hay órdenes en borrador." />
      </div>
      <div class="col-span-24 lg:col-span-15" aria-label="Every supply the plan needs, netted against the warehouse">
        <div class="h3 mb-6">{tr('Supplies needed|Insumos necesarios')}</div>
        {#if !requirementsState.form.WarehouseID}
          <div class="p-12 rounded-md bg-surface-soft text-fg-muted">
            {tr('Select the supplies warehouse, then pick draft orders or a template and its series.|Seleccione el almacén de insumos y luego órdenes en borrador o una plantilla y sus series.')}
          </div>
        {:else}
          <VTable columns={requirementColumns} data={requirementsState.requirements} disableVirtualizer={true} estimateSize={36}
            emptyMessage="Nothing planned yet.|Aún no hay nada planificado." />
        {/if}
      </div>
    </div>

    {#if requirementsState.form.WarehouseID}
      <div class="mt-20" aria-label="Supplies to buy, one purchase order per provider">
        <div class="flex items-center mb-6">
          <div class="h3">{tr('To buy|Por comprar')}</div>
          <Button color="green" icon="icon-[fa--shopping-cart]" name="Create purchase orders|Crear órdenes de compra" css="ml-auto"
            label="Creates one pending purchase order per provider with these lines."
            disabled={requirementsState.isSaving || !requirementsState.purchaseSuggestions.length
              || requirementsState.purchasedPlan === requirementsState.requirements}
            onClick={createPurchaseOrders} />
        </div>
        <VTable columns={purchaseColumns} data={requirementsState.purchaseSuggestions} disableVirtualizer={true} estimateSize={36}
          emptyMessage="No supply is short.|Ningún insumo falta." />
      </div>

      <div class="mt-20" aria-label="Semi-finished supplies to make, as draft production orders">
        <div class="flex items-center mb-6">
          <div class="h3">{tr('To make|Por fabricar')}</div>
          <Button color="blue" icon="icon-[fa--cogs]" name="Create orders|Crear órdenes" css="ml-auto"
            label="Creates one draft production order per semi-finished supply."
            disabled={requirementsState.isSaving || !requirementsState.semiFinishedShortfalls.length
              || requirementsState.draftedPlan === requirementsState.requirements}
            onClick={createSemiFinishedDrafts} />
        </div>
        <VTable columns={makeColumns} data={requirementsState.semiFinishedShortfalls} disableVirtualizer={true} estimateSize={36}
          emptyMessage="No semi-finished supply is short.|Ningún semielaborado falta." />
      </div>
    {/if}
  </Layer>
</Page>
