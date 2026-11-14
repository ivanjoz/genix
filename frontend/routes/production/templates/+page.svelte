<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { formatN } from '#libs/helpers.ts'
  import ProductSizeGrid from '#routes/production/orders/ProductSizeGrid.svelte'
  import { explodeProductionLines, productRunsToOrderLines, runTotalUnits, type IExplodedSupply } from '#routes/production/orders/production-order.ts'
  import { ProductionCatalogs } from '#routes/production/orders/production-order.svelte.ts'
  import { type IProductionTemplate } from '#services/production/production-orders.ts'
  import { postProductionTemplate } from '#services/production/production-orders.svelte.ts'
  import { estimateSeriesCost, findTemplateError, makeEmptyTemplate } from './production-template.ts'

  // Production templates (docs/MANUFACTURING_PLAN.md §4): reusable lists of products in their size
  // runs. An order is created from one for a number of series and is independent of it afterwards.
  const ui = useUI()
  const TEMPLATE_LAYER_ID = 1
  const catalogs = new ProductionCatalogs(false)

  let templateForm = $state(makeEmptyTemplate())

  // One series, exploded with the current recipes: what it costs and which supplies it needs.
  const seriesLines = $derived(productRunsToOrderLines(templateForm.Lines))
  const seriesCost = $derived(estimateSeriesCost(templateForm.Lines, catalogs.recipeProductsByID,
    catalogs.averageCostBySupplyID, catalogs.costPerHourByLaborTypeID))
  const seriesSupplies = $derived(explodeProductionLines(seriesLines, 0, catalogs.recipeProductsByID).supplies)
  const templateError = $derived(findTemplateError(templateForm))
  const formatCents = (cents: number) => formatN(cents / 100, 2)

  const openTemplate = (template: IProductionTemplate) => {
    templateForm = structuredClone($state.snapshot(template))
    templateForm.Lines = templateForm.Lines || []
    ui.openSideLayer(TEMPLATE_LAYER_ID)
  }

  const saveTemplate = async (isDelete = false) => {
    if (isDelete) {
      if (!(await confirmWarn({
        title: tr('Delete|Eliminar'),
        message: tr(`Delete "${templateForm.Name}"? Orders made from it keep their lines.|¿Eliminar "${templateForm.Name}"? Las órdenes creadas con ella conservan sus líneas.`),
      }))) return
    } else if (templateError) {
      notifyFailure(tr(templateError))
      return
    }
    showLoading(tr('Saving template...|Guardando plantilla...'))
    try {
      await postProductionTemplate(catalogs.templates, { ...$state.snapshot(templateForm), ss: isDelete ? 0 : 1 })
      ui.openSideLayer(0)
      templateForm = makeEmptyTemplate()
    } catch (saveError) {
      console.error('[production-templates] save error', { templateForm: $state.snapshot(templateForm), saveError })
      notifyFailure(saveError)
    }
    hideLoading()
  }

  const templateColumns: ITableColumn<IProductionTemplate>[] = [
    { header: 'Template|Plantilla', css: 'px-6', highlight: true, getValue: (template) => template.Name },
    { header: 'Products|Productos', headerCss: 'w-100', css: 'px-6 ff-mono', align: 'right', getValue: (template) => (template.Lines || []).length },
    {
      header: 'Units / series|Unidades / serie', headerCss: 'w-140', css: 'px-6 ff-mono', align: 'right',
      getValue: (template) => (template.Lines || []).reduce((total, templateLine) => total + runTotalUnits(templateLine), 0),
    },
    { header: 'Default series|Series por defecto', headerCss: 'w-150', css: 'px-6 ff-mono', align: 'right', getValue: (template) => template.DefaultSeries || 0 },
    { header: 'Warehouse|Almacén', headerCss: 'w-180', css: 'px-6', getValue: (template) => catalogs.warehouseNameOf(template.WarehouseID) },
  ]

  const supplyColumns: ITableColumn<IExplodedSupply>[] = [
    { header: 'Supply|Insumo', css: 'px-6', getValue: (supply) => catalogs.supplies.recordsMap.get(supply.SupplyID)?.Name || `#${supply.SupplyID}` },
    {
      header: 'Quantity|Cantidad', headerCss: 'w-160', css: 'px-6 ff-mono', align: 'right',
      getValue: (supply) => catalogs.formatSupplyQuantity(supply.SupplyID, supply.Planned, supply.SubDivisor),
    },
    {
      header: 'Est. cost|Costo est.', headerCss: 'w-120', css: 'px-6 ff-mono', align: 'right',
      getValue: (supply) => {
        const averageUnitCost = catalogs.averageCostBySupplyID.get(supply.SupplyID)
        return averageUnitCost === undefined ? tr('no cost|sin costo') : formatCents(averageUnitCost * supply.Planned / supply.SubDivisor)
      },
    },
  ]
</script>

<Page title="Production templates|Plantillas de producción">
  <Layer type="content">
    <div class="flex items-center justify-between mb-6 gap-12 flex-wrap" aria-label="Templates toolbar with create button">
      <div class="text-fg-muted">{tr('Products in their size runs: an order is created from a template for a number of series.|Productos con su serie de tallas: una orden se crea desde una plantilla por un número de series.')}</div>
      <Button color="green" icon="icon-[fa--plus]" name="New|Nueva" label="Opens the side layer to create a template."
        onClick={() => openTemplate(makeEmptyTemplate())} />
    </div>
    <VTable columns={templateColumns} data={catalogs.templates.records} css="w-full" maxHeight="calc(100vh - 8rem - 16px)"
      selected={templateForm.ID} isSelected={(template, selectedID) => template.ID === selectedID} onRowClick={openTemplate}
      emptyMessage="No templates yet.|Aún no hay plantillas." />
  </Layer>
</Page>

<Layer
  id={TEMPLATE_LAYER_ID}
  type="side"
  sideLayerSize={860}
  title={templateForm.ID > 0 ? templateForm.Name : 'New template|Nueva plantilla'}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  onSave={() => saveTemplate()}
  onDelete={templateForm.ID > 0 ? () => saveTemplate(true) : undefined}
  onClose={() => { templateForm = makeEmptyTemplate() }}
>
  <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Template form">
    <Input bind:saveOn={templateForm} save="Name" label="Name|Nombre" required={true} css="col-span-24 md:col-span-16" />
    <Input bind:saveOn={templateForm} save="DefaultSeries" label="Default series|Series por defecto" type="number" css="col-span-24 md:col-span-8" />
    <SearchSelect bind:saveOn={templateForm} save="WarehouseID" options={catalogs.warehouses.Almacenes || []} keyId="ID" keyName="Name"
      label="Receiving warehouse|Almacén de ingreso" placeholder=":: seleccione ::" css="col-span-24 md:col-span-12" />
    <SearchSelect bind:saveOn={templateForm} save="SuppliesWarehouseID" options={catalogs.warehouses.Almacenes || []} keyId="ID" keyName="Name"
      label="Supplies warehouse|Almacén de insumos" placeholder=":: seleccione ::" css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={templateForm} save="Notes" label="Notes|Notas" css="col-span-24" />
  </div>

  <h3 class="font-bold mt-16 mb-6">{tr('One series|Una serie')}</h3>
  <ProductSizeGrid bind:productRuns={templateForm.Lines} {catalogs} isEditable={true} />

  <!-- One series at the current recipes: the supplies' average cost and the labor types' rates. -->
  <div class="flex items-center justify-between mt-16 p-12 rounded-md bg-surface-soft" aria-label="Estimated cost of one series">
    <div class="font-bold">{tr('Estimated cost of one series|Costo estimado de una serie')}</div>
    <div class="flex items-center gap-8">
      {#if seriesCost.isIncomplete}
        <span class="text-amber-700">{tr('Incomplete: some product or supply has no cost|Incompleto: algún producto o insumo no tiene costo')}</span>
      {/if}
      <span class="font-bold ff-mono">{formatCents(seriesCost.totalCost)}</span>
    </div>
  </div>

  <h3 class="font-bold mt-16 mb-6">{tr('Supplies of one series|Insumos de una serie')}</h3>
  <div aria-label="Supplies one series needs, waste included">
    <VTable columns={supplyColumns} data={seriesSupplies} disableVirtualizer={true} estimateSize={36}
      emptyMessage="The products have no recipe supplies.|Los productos no tienen insumos en su receta." />
  </div>
</Layer>
