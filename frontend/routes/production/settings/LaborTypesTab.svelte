<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import FilterInput from '#components/form/FilterInput.svelte'
  import Input from '#components/form/Input.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import {
    postProductionLaborType, ProductionLaborTypesService, type IProductionLaborType,
  } from '#services/production/production-settings.svelte.ts'

  // Labor types ("Zapatero", "Maquinista"): a recipe plans minutes of each, and their cost per hour
  // estimates the recipe's, a template's and an order's labor. The real cost comes from the hours logged.
  const ui = useUI()
  const LABOR_TYPE_LAYER_ID = 1
  const laborTypes = new ProductionLaborTypesService(true)

  let filterText = $state('')
  let laborTypeForm = $state({} as IProductionLaborType)

  const openLaborType = (laborType: IProductionLaborType) => {
    laborTypeForm = { ...laborType }
    ui.openSideLayer(LABOR_TYPE_LAYER_ID)
  }

  const saveLaborType = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({
      title: tr('Delete|Eliminar'), message: tr(`Delete "${laborTypeForm.Name}"?|¿Eliminar "${laborTypeForm.Name}"?`),
    }))) return
    showLoading(tr('Saving labor type...|Guardando tipo de trabajo...'))
    try {
      await postProductionLaborType({ ...laborTypeForm, ss: isDelete ? 0 : 1 })
      ui.openSideLayer(0)
      laborTypeForm = {} as IProductionLaborType
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<IProductionLaborType>[] = [
    { header: 'ID', headerCss: 'w-54', css: 'text-center ff-bold', getValue: (laborType) => laborType.ID },
    { header: 'Labor type|Tipo de trabajo', css: 'px-6', highlight: true, getValue: (laborType) => laborType.Name },
    { header: 'Cost / hour|Costo / hora', css: 'px-6 text-right', getValue: (laborType) => formatN(laborType.CostPerHour / 100, 2) },
  ]
</script>

<Layer type="content">
  <div class="flex items-center justify-between mb-6 gap-12 flex-wrap" aria-label="Labor types toolbar with filter and create button">
    <FilterInput bind:value={filterText} css="w-256" />
    <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo" label="Opens the side layer to create a labor type."
      onClick={() => openLaborType({ ID: 0, Name: '', CostPerHour: 0, ss: 1, upd: 0 })} />
  </div>
  <VTable
    {columns}
    data={laborTypes.records}
    css="w-full"
    maxHeight="calc(100vh - 8rem - 16px)"
    {filterText}
    getFilterContent={(laborType) => laborType.Name.toLowerCase()}
    selected={laborTypeForm.ID}
    isSelected={(laborType, selectedID) => laborType.ID === selectedID}
    onRowClick={openLaborType}
    emptyMessage="No labor types yet.|Aún no hay tipos de trabajo."
  />
</Layer>

<Layer
  id={LABOR_TYPE_LAYER_ID}
  type="side"
  sideLayerSize={480}
  title={laborTypeForm.ID > 0 ? laborTypeForm.Name : 'New labor type|Nuevo tipo de trabajo'}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  onSave={() => saveLaborType()}
  onDelete={laborTypeForm.ID > 0 ? () => saveLaborType(true) : undefined}
  onClose={() => { laborTypeForm = {} as IProductionLaborType }}
>
  <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Labor type form">
    <Input bind:saveOn={laborTypeForm} save="Name" label="Name|Nombre" required={true} css="col-span-24" />
    <Input bind:saveOn={laborTypeForm} save="CostPerHour" label="Cost per hour (estimate)|Costo por hora (estimado)" type="number"
      baseDecimals={2} css="col-span-24 md:col-span-12" />
    <div class="col-span-24 text-gray-600">
      {tr('Used to estimate the cost of recipes and orders. The labor booked to products uses each worker’s real cost per hour.|Se usa para estimar el costo de recetas y órdenes. La mano de obra cargada a los productos usa el costo real por hora de cada trabajador.')}
    </div>
  </div>
</Layer>
