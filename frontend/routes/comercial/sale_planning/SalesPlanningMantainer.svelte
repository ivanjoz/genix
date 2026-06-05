<script lang="ts">
  import OptionsStrip from '$components/navigation/OptionsStrip.svelte'
  import Layer from '$components/layers/Layer.svelte'
  import VTable from '$components/vTable/VTable.svelte'
  import type { ITableColumn } from '$components/vTable/types'
  import Button from '$components/buttons/Button.svelte'
  import FilterInput from '$components/form/FilterInput.svelte'
  import Input from '$components/form/Input.svelte'
  import SearchSelect from '$components/form/SearchSelect.svelte'
  import T from '$components/misc/T.svelte'
  import { Core, tr } from '$core/store.svelte'
  import { Loading, Notify, ConfirmWarn } from '$libs/helpers'
  import { ProductosService, type IProduct } from '$routes/negocio/productos/productos.svelte'
  import {
    SalesPlanningService,
    SeasonalityCurveService,
    resolveCurve,
    weekMonthGroups,
    WEEKS_PER_YEAR,
    type ISalesPlanning,
    type ISeasonalityCurve,
  } from './sale_planning.svelte'

  const monthGroups = weekMonthGroups()

  const productos = new ProductosService(true)
  const planning = new SalesPlanningService(true)
  const curves = new SeasonalityCurveService(true)

  let view = $state(1)
  let productFilter = $state('')
  let curveFilter = $state('')

  // Working week rows keep the value optional so empty weeks render blank (not 0).
  type WeekInput = { Week: number; value?: number }

  // Product plan side layer (id=1)
  let planForm = $state({} as ISalesPlanning)
  let planWeeks = $state([] as WeekInput[])
  let planProductName = $state('')

  // Seasonality curve side layer (id=2)
  let curveForm = $state({} as ISeasonalityCurve)
  let curveWeeks = $state([] as WeekInput[])

  const planByProduct = $derived.by(() => {
    const map = new Map<number, ISalesPlanning>()
    for (const plan of planning.records) map.set(plan.ProductID, plan)
    return map
  })

  const curveNameByID = $derived.by(() => {
    const map = new Map<number, string>()
    for (const curve of curves.records) map.set(curve.ID, curve.Name)
    return map
  })

  /* ---------- Product plan ---------- */

  const openProductPlan = (product: IProduct) => {
    planProductName = product.Name
    const existing = planByProduct.get(product.ID)
    planForm = existing
      ? { ...existing }
      : ({ ID: 0, ProductID: product.ID, BaseQuantity: 0, SeasonalityCurveID: 0, WeeklyQuantity: [], ss: 1 } as ISalesPlanning)

    const byWeek = new Map<number, number>()
    for (const week of existing?.WeeklyQuantity || []) byWeek.set(week.Week, week.Quantity)
    planWeeks = Array.from({ length: WEEKS_PER_YEAR }, (_, i) => ({ Week: i + 1, value: byWeek.get(i + 1) }))
    Core.openSideLayer(1)
  }

  const savePlan = async () => {
    if ((planForm.BaseQuantity || 0) < 0) {
      Notify.failure(tr('Base quantity cannot be negative|La cantidad base no puede ser negativa'))
      return
    }
    // Persist only the weeks the owner actually filled; empty weeks fall back to the base.
    planForm.WeeklyQuantity = planWeeks
      .filter((w) => (w.value || 0) > 0)
      .map((w) => ({ Week: w.Week, Quantity: w.value as number }))

    Loading.standard(tr('Saving|Guardando') + '...')
    await planning.postAndSync([planForm])
    Loading.remove()
    Core.openSideLayer(0)
  }

  /* ---------- Seasonality curve ---------- */

  const newCurve = () => {
    curveForm = { ID: 0, Name: '', Curve: [], ss: 1 } as ISeasonalityCurve
    curveWeeks = Array.from({ length: WEEKS_PER_YEAR }, (_, i) => ({ Week: i + 1 }))
    Core.openSideLayer(2)
  }

  const openCurve = (curve: ISeasonalityCurve) => {
    curveForm = { ...curve }
    const byWeek = new Map<number, number>()
    for (const week of curve.Curve || []) byWeek.set(week.Week, week.Percent)
    curveWeeks = Array.from({ length: WEEKS_PER_YEAR }, (_, i) => ({ Week: i + 1, value: byWeek.get(i + 1) }))
    Core.openSideLayer(2)
  }

  const saveCurve = async () => {
    if ((curveForm.Name || '').trim().length < 2) {
      Notify.failure(tr('The curve needs a name|La curva necesita un nombre'))
      return
    }
    // Sparse storage: keep only filled weeks; gaps inherit the previous filled week on read.
    curveForm.Curve = curveWeeks
      .filter((w) => (w.value || 0) > 0)
      .map((w) => ({ Week: w.Week, Percent: w.value as number }))

    if (curveForm.Curve.length === 0) {
      Notify.failure(tr('Fill at least one week|Complete al menos una semana'))
      return
    }

    Loading.standard(tr('Saving|Guardando') + '...')
    await curves.postAndSync([curveForm])
    Loading.remove()
    Core.openSideLayer(0)
  }

  const deleteCurve = () => {
    ConfirmWarn(
      tr('Delete curve|Eliminar curva'),
      tr('Are you sure you want to delete|¿Está seguro que desea eliminar') + ` "${curveForm.Name}"?`,
      tr('YES|SI'),
      tr('NO|NO'),
      async () => {
        Loading.standard(tr('Deleting|Eliminando') + '...')
        await curves.postAndSync([{ ...curveForm, ss: 0 }])
        Loading.remove()
        Core.openSideLayer(0)
      },
    )
  }

  /* Resolved (forward-filled) preview of the curve being edited. */
  const curvePreview = $derived(
    resolveCurve(
      curveWeeks
        .filter((w) => (w.value || 0) > 0)
        .map((w) => ({ Week: w.Week, Percent: w.value as number })),
    ),
  )

  /* ---------- Columns ---------- */

  const productColumns: ITableColumn<IProduct>[] = [
    { header: 'ID', field: 'ID', width: '60px', getValue: (e) => e.ID || '' },
    { header: 'Product|Producto', field: 'Name', useLineClamp: true, getValue: (e) => e.Name },
    {
      header: 'Base Qty|Cant. Base',
      width: '110px',
      getValue: (e) => planByProduct.get(e.ID)?.BaseQuantity || '',
    },
    {
      header: 'Seasonality|Estacionalidad',
      width: '160px',
      getValue: (e) => {
        const curveID = planByProduct.get(e.ID)?.SeasonalityCurveID
        return curveID ? curveNameByID.get(curveID) || '' : ''
      },
    },
  ]

  const curveColumns: ITableColumn<ISeasonalityCurve>[] = [
    { header: 'ID', field: 'ID', width: '60px', getValue: (e) => e.ID || '' },
    { header: 'Name|Nombre', field: 'Name', getValue: (e) => e.Name },
    {
      header: 'Filled weeks|Semanas',
      width: '120px',
      getValue: (e) => (e.Curve || []).length,
    },
  ]

  const curveOptions = $derived([
    { ID: 0, Name: tr('None|Ninguna') },
    ...curves.records.map((c) => ({ ID: c.ID, Name: c.Name })),
  ])
</script>

<div class="grid grid-cols-12 md:flex md:flex-row items-center mb-8">
  <OptionsStrip
    selected={view}
    css="col-span-12 mb-6 md:mb-0"
    options={[
      [1, 'Products|Productos'],
      [2, 'Seasonality|Estacionalidad'],
    ]}
    useMobileGrid={true}
    onSelect={(e) => {
      Core.openSideLayer(0)
      view = e[0] as number
    }}
  />

  {#if view === 1}
    <FilterInput label="Filter products|Filtrar productos"
      css="w-full md:w-220 md:ml-12 col-span-12"
      icon="icon-search"
      bind:value={productFilter}
    />
  {:else}
    <FilterInput label="Filter curves|Filtrar curvas"
      css="w-full md:w-220 md:ml-12 col-span-7"
      icon="icon-search"
      bind:value={curveFilter}
    />
    <Button name="New|Nueva"
      color="green"
      icon="icon-plus"
      css="ml-auto col-span-5"
      onClick={newCurve}
    />
  {/if}
</div>

{#if view === 1}
  <Layer type="content">
    <VTable
      columns={productColumns}
      data={productos.records}
      filterText={productFilter}
      getFilterContent={(e) => e.Name}
      selected={planForm?.ProductID}
      isSelected={(e, id) => e.ID === id}
      onRowClick={(e) => openProductPlan(e)}
    />
  </Layer>
{:else}
  <Layer type="content">
    <VTable
      columns={curveColumns}
      data={curves.records}
      filterText={curveFilter}
      getFilterContent={(e) => e.Name}
      selected={curveForm?.ID}
      isSelected={(e, id) => e.ID === id}
      onRowClick={(e) => openCurve(e)}
    />
  </Layer>
{/if}

<!-- Product plan editor -->
<Layer type="side" id={1} sideLayerSize={860}
  css="px-8 py-8 md:px-16 md:py-12"
  title={planProductName}
  titleCss="h2 mb-6"
  onSave={savePlan}
  onClose={() => { planForm = {} as ISalesPlanning; planWeeks = [] }}
>
  <div class="grid grid-cols-24 gap-10 mt-12">
    <Input label="Base weekly quantity|Cantidad base semanal"
      saveOn={planForm} save="BaseQuantity" type="number"
      css="col-span-24 md:col-span-8"
    />
    <SearchSelect label="Seasonality curve|Curva de estacionalidad"
      saveOn={planForm} save="SeasonalityCurveID"
      keyId="ID" keyName="Name"
      options={curveOptions}
      css="col-span-24 md:col-span-12"
    />
  </div>

  <div class="mt-16 mb-6 c-gray-600 text-13">
    <T text="Weekly quantity (leave empty to use the base)|Cantidad por semana (vacío = usa la base)" />
  </div>
  <div class="flex flex-col gap-6">
    {#each monthGroups as group (group.month)}
      <div class="flex items-start gap-8">
        <div class="w-54 pt-6 shrink-0 ff-bold c-gray-600 text-13">
          {tr(group.monthLabel)} {group.startDay}
        </div>
        <div class="flex-1 grid grid-cols-3 sm:grid-cols-5 gap-4">
          {#each group.weeks as w (w)}
            <Input label={`W${w}`} saveOn={planWeeks[w - 1]} save="value" type="number" inputCss="text-center" />
          {/each}
        </div>
      </div>
    {/each}
  </div>
</Layer>

<!-- Seasonality curve editor -->
<Layer type="side" id={2} sideLayerSize={860}
  css="px-8 py-8 md:px-16 md:py-12"
  title={curveForm?.Name || tr('New curve|Nueva curva')}
  titleCss="h2 mb-6"
  onSave={saveCurve}
  onDelete={curveForm?.ID ? deleteCurve : undefined}
  onClose={() => { curveForm = {} as ISeasonalityCurve; curveWeeks = [] }}
>
  <div class="grid grid-cols-24 gap-10 mt-12">
    <Input label="Name|Nombre" saveOn={curveForm} save="Name" required
      css="col-span-24 md:col-span-12"
    />
  </div>

  <div class="mt-16 mb-6 c-gray-600 text-13">
    <T text="Weekly percentage (100 = no change, 150 = +50%). Empty weeks inherit the previous filled week.|Porcentaje por semana (100 = sin cambio, 150 = +50%). Las semanas vacías heredan la anterior." />
  </div>
  <div class="flex flex-col gap-6">
    {#each monthGroups as group (group.month)}
      <div class="flex items-start gap-8">
        <div class="w-54 pt-6 shrink-0 ff-bold c-gray-600 text-13 ff-mono">
          {tr(group.monthLabel)}
        </div>
        <div class="flex-1 grid grid-cols-3 sm:grid-cols-5 gap-4">
          {#each group.weeks as w (w)}
            <Input label={`W${w}`} saveOn={curveWeeks[w - 1]} save="value" type="number"
              baseDecimals={1} inputCss="text-center"
              placeholder={String(Math.round((curvePreview[w - 1] || 1) * 100))}
            />
          {/each}
        </div>
      </div>
    {/each}
  </div>
</Layer>
