<script lang="ts">
  import { useUI } from '@genix/ui'
  import { notifyFailure } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import FilterInput from '#components/form/FilterInput.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { formatTime } from '#libs/helpers.ts'
  import type { IProductionOrder } from '#services/production/production-orders.ts'
  import OrderBoard from './OrderBoard.svelte'
  import OrderLayer from './OrderLayer.svelte'
  import { ORDER_STATUS_LABELS, makeEmptyOrder, makeOrderFromTemplate, orderProgressPercent, unitsByProduct } from './production-order.ts'
  import { ProductionCatalogs } from './production-order.svelte.ts'

  // Production orders (docs/MANUFACTURING_PLAN.md §1, §12): a list and a board by stage. Page wiring
  // only; the order's detail lives in OrderLayer.
  const ui = useUI()
  const ORDER_LAYER_ID = 1
  const TEMPLATE_MODAL_ID = 2
  const catalogs = new ProductionCatalogs(true)

  const pageOptions = [
    { id: 1, name: 'List|Lista' },
    { id: 2, name: 'Board|Tablero' },
  ]
  let filterText = $state('')
  let orderForm = $state<IProductionOrder>(makeEmptyOrder())
  let templateChoice = $state({ TemplateID: 0, Series: 1 })
  // Bumped on every open: the order layer is mounted anew, so its size grid starts from that order.
  let openCount = $state(0)

  const openOrder = (order: IProductionOrder) => {
    orderForm = structuredClone($state.snapshot(order))
    openCount++
    ui.openSideLayer(ORDER_LAYER_ID)
  }

  // "Nueva orden desde plantilla": the template's runs × the series, as a new draft.
  const createFromTemplate = () => {
    const template = catalogs.templates.recordsMap.get(templateChoice.TemplateID)
    if (!template || !(templateChoice.Series > 0)) {
      notifyFailure(tr('Pick the template and the number of series.|Elija la plantilla y el número de series.'))
      return
    }
    ui.closeModal(TEMPLATE_MODAL_ID)
    openOrder(makeOrderFromTemplate(template, templateChoice.Series))
  }

  const columns: ITableColumn<IProductionOrder>[] = [
    { header: 'N°', headerCss: 'w-60', css: 'text-center ff-bold', getValue: (order) => order.ID },
    { header: 'Day|Día', headerCss: 'w-110', css: 'px-6', getValue: (order) => String(formatTime(order.Date, 'd-M-Y')) },
    { header: 'Status|Estado', headerCss: 'w-110', css: 'px-6', getValue: (order) => tr(ORDER_STATUS_LABELS[order.ss] || '') },
    {
      header: 'Products|Productos', css: 'px-6', highlight: true,
      getValue: (order) => unitsByProduct(order.Lines || []).map((product) => `${product.Units}x ${catalogs.productNameOf(product.ProductID)}`).join(', '),
    },
    { header: 'Stage|Etapa', headerCss: 'w-120', css: 'px-6', getValue: (order) => catalogs.stages.recordsMap.get(order.StageID)?.Name || '' },
    { header: 'Workshop|Taller', headerCss: 'w-150', css: 'px-6', getValue: (order) => order.SubcontractorID > 0 ? catalogs.workshopNameOf(order.SubcontractorID) : '' },
    { header: 'Received|Recibido', headerCss: 'w-100', css: 'px-6 ff-mono', align: 'right', getValue: (order) => `${orderProgressPercent(order.Lines || [])}%` },
  ]
</script>

<Page title="Production orders|Órdenes de producción" options={pageOptions}>
  <div class="flex items-center justify-between mb-8 gap-12 flex-wrap" aria-label="Orders toolbar">
    <FilterInput bind:value={filterText} css="w-256" />
    <div class="flex items-center gap-8">
      <Button color="blue" mode="subtle" icon="icon-[fa--clone]" name="From template|Desde plantilla"
        label="Creates a draft from a template and a number of series."
        onClick={() => { templateChoice = { TemplateID: 0, Series: 1 }; ui.openModal(TEMPLATE_MODAL_ID) }} />
      <Button color="green" icon="icon-[fa--plus]" name="New|Nueva" label="Opens an empty draft order."
        onClick={() => openOrder(makeEmptyOrder())} />
    </div>
  </div>
  {#if ui.state.pageOptionSelected === 2}
    <OrderBoard {catalogs} onOpen={openOrder} />
  {:else}
    <Layer type="content">
      <VTable {columns} data={catalogs.orders?.records || []} css="w-full" maxHeight="calc(100vh - 8rem - 16px)" {filterText}
        getFilterContent={(order) => [order.ID, ...unitsByProduct(order.Lines || []).map((product) => catalogs.productNameOf(product.ProductID))].join(' ').toLowerCase()}
        selected={orderForm.ID} isSelected={(order, selectedID) => order.ID === selectedID} onRowClick={openOrder}
        emptyMessage="No production orders yet.|Aún no hay órdenes de producción." />
    </Layer>
  {/if}
</Page>

{#key openCount}
  <OrderLayer bind:order={orderForm} {catalogs} layerID={ORDER_LAYER_ID} />
{/key}

<Modal id={TEMPLATE_MODAL_ID} size={5} title={tr('New order from template|Nueva orden desde plantilla')} bodyCss="px-16 py-14"
  onSave={createFromTemplate} saveButtonLabel={tr('Create|Crear')}>
  <div class="grid grid-cols-24 gap-10" aria-label="Template and number of series">
    <SearchSelect bind:saveOn={templateChoice} save="TemplateID" options={catalogs.templates.records} keyId="ID" keyName="Name"
      label="Template|Plantilla" css="col-span-24 md:col-span-16"
      onChange={() => { templateChoice.Series = catalogs.templates.recordsMap.get(templateChoice.TemplateID)?.DefaultSeries || 1 }} />
    <Input bind:saveOn={templateChoice} save="Series" type="number" label="Series|Series" css="col-span-24 md:col-span-8" />
  </div>
</Modal>
