<script lang="ts">
  import OptionsStrip from '$components/navigation/OptionsStrip.svelte'
  import Layer from '$components/layers/Layer.svelte'
  import VTable from '$components/vTable/VTable.svelte'
  import type { ITableColumn } from '$components/vTable/types'
  import Button from '$components/buttons/Button.svelte'
  import FilterInput from '$components/form/FilterInput.svelte'
  import Input from '$components/form/Input.svelte'
  import SearchSelect from '$components/form/SearchSelect.svelte'
  import CellSimpleChart from '$components/charts/CellSimpleChart.svelte'
  import T from '$components/misc/T.svelte'
  import { Core, tr } from '$core/store.svelte'
  import { Loading, Notify, ConfirmWarn } from '$libs/helpers'
  import { ProductsService, type IProduct } from '$routes/business/products/products.svelte'
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

  const productos = new ProductsService(true)
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

  const curveByID = $derived.by(() => {
    const map = new Map<number, ISeasonalityCurve>()
    for (const curve of curves.records) map.set(curve.ID, curve)
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

  // Build the row chart: forward-filled values, labels only on weeks the curve actually defines,
  // and a baseline dropped just below the lowest week so even the smallest bar stays visible.
  const buildCurveChart = (curve: ISeasonalityCurve['Curve']) => {
    const resolved = resolveCurve(curve)
    const labels = resolved.map(() => '')
    for (const week of curve || []) {
      if (week.Week >= 1 && week.Week <= WEEKS_PER_YEAR)
        labels[week.Week - 1] = String(Math.round((resolved[week.Week - 1] || 0) * 100))
    }
    // Always label the first week so the curve's starting value is visible.
    if (!labels[0]) labels[0] = String(Math.round((resolved[0] || 0) * 100))
    // Baseline is 20 percentage points below the lowest week (0.20 in multiplier terms),
    // but never negative: a week under 20% just sits on the 0 floor.
    const min = Math.min(...resolved)
    return { values: resolved, labels, minValue: min < 0.2 ? 0 : min - 0.2 }
  }

  // 5-band light green→red scale: low weeks are green, peak weeks are red.
  const seasonalityColorScale = ['#4ade80', '#a3e635', '#fde047', '#fb923c', '#f87171']

  // Per-product weekly quantity: the seasonality multiplier times the week's explicit quantity
  // (or the base when that week wasn't filled). Length/color come from a scale built over these
  // same values. Labels mark only the weeks where the quantity changes (plus week 1).
  const buildProductChart = (plan?: ISalesPlanning): { values: number[]; labels: string[]; minValue: number; colors: string[] } | undefined => {
    if (!plan) return undefined
    const base = plan.BaseQuantity || 0
    const curve = plan.SeasonalityCurveID ? curveByID.get(plan.SeasonalityCurveID) : undefined
    const multipliers = resolveCurve(curve?.Curve || [])
    const explicit = new Map<number, number>()
    for (const week of plan.WeeklyQuantity || []) explicit.set(week.Week, week.Quantity)
    const values = multipliers.map((m, i) => m * (explicit.get(i + 1) || base))
    // Nothing to show if the product has no base quantity and no explicit weeks.
    if (!values.some((v) => v > 0)) return undefined
    const labels = values.map((v, i) =>
      i === 0 || v !== values[i - 1] ? String(Math.round(v)) : '',
    )
    const scale = weekScale(values, base)
    return {
      values,
      labels,
      minValue: scale.minValue,
      colors: values.map((v) => weekBar(v, scale).color),
    }
  }

  // A fixed length/color scale for weekly bars, derived only from base × curve multipliers so it
  // stays put when a single week is overridden. Bars are anchored at the base: a week AT the base
  // is the neutral middle band, weeks below are green, weeks above are red — monotonic in value.
  type WeekScale = { minValue: number; lengthSpan: number; base: number; maxDev: number }
  const weekScale = (baseValues: number[], base: number): WeekScale => {
    const min = Math.min(...baseValues)
    const max = Math.max(...baseValues)
    const minValue = Math.floor(Math.max(0, min - max * 0.15))
    return {
      minValue,
      lengthSpan: Math.max(max - minValue, 1e-9),
      base,
      // Largest distance from the base across the curve sets the symmetric half-range.
      maxDev: Math.max(1e-9, ...baseValues.map((v) => Math.abs(v - base))),
    }
  }

  // Length (% up from the baseline) and color for one week's value, placed on a fixed scale.
  const weekBar = (value: number, s: WeekScale) => {
    const bands = seasonalityColorScale.length
    // 0.5 at the base, →0 far below, →1 far above; clamped so overrides past the curve range cap.
    const ratio = s.base <= 0 ? 0.5 : (value - s.base) / s.maxDev / 2 + 0.5
    return {
      lengthPercent: value > 0 ? Math.max(0, Math.min(100, ((value - s.minValue) / s.lengthSpan) * 100)) : 0,
      color: seasonalityColorScale[Math.max(0, Math.min(bands - 1, Math.floor(ratio * bands)))],
    }
  }

  // Live per-week bars for the product editor: each week's quantity is the seasonality multiplier
  // times the typed value (or the base when empty). Length and color come from a scale built over
  // these same values, so the chart always reflects exactly what's in the inputs.
  // Throttled (250ms) so rapid typing only recomputes/re-animates the bars at most 4×/second.
  let planWeekBars = $state([] as { lengthPercent: number; color: string }[])
  let weekBarsTimer: ReturnType<typeof setTimeout> | null = null
  let weekBarsLastRun = 0

  const computeWeekBars = () => {
    const base = planForm.BaseQuantity || 0
    const curve = planForm.SeasonalityCurveID ? curveByID.get(planForm.SeasonalityCurveID) : undefined
    const multipliers = resolveCurve(curve?.Curve || [])
    // A typed week overrides the base for that week; the seasonality multiplier still applies on top.
    const values = multipliers.map((m, i) => m * (planWeeks[i]?.value || base))
    const scale = weekScale(values, base)
    planWeekBars = values.map((value) => weekBar(value, scale))
  }

  $effect(() => {
    // Touch every input so the effect re-runs whenever the base, curve, or any week changes.
    planForm.BaseQuantity
    planForm.SeasonalityCurveID
    planWeeks.forEach((w) => w.value)

    const elapsed = Date.now() - weekBarsLastRun
    if (elapsed >= 250) {
      // Leading edge: enough time has passed, update immediately.
      weekBarsLastRun = Date.now()
      computeWeekBars()
    } else if (!weekBarsTimer) {
      // Trailing edge: coalesce the burst into one update once the window closes.
      weekBarsTimer = setTimeout(() => {
        weekBarsTimer = null
        weekBarsLastRun = Date.now()
        computeWeekBars()
      }, 250 - elapsed)
    }
  })

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
    {
      // id triggers the cellRenderer snippet so we can draw the weekly quantity as bars.
      id: 'productChart',
      header: 'Weekly quantity|Cantidad semanal',
      useCellRenderer: true,
    },
  ]

  const curveColumns: ITableColumn<ISeasonalityCurve>[] = [
    { header: 'ID', field: 'ID', width: '60px', getValue: (e) => e.ID || '' },
    { header: 'Name|Nombre', field: 'Name', getValue: (e) => e.Name, headerCss: "w-[40%]" },
    {
      // id triggers the cellRenderer snippet so we can draw the resolved curve as bars.
      id: 'curveChart',
      header: 'Filled weeks|Semanas',
      useCellRenderer: true,
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
      icon="icon-[fa--search]"
      bind:value={productFilter}
    />
  {:else}
    <FilterInput label="Filter curves|Filtrar curvas"
      css="w-full md:w-220 md:ml-12 col-span-7"
      icon="icon-[fa--search]"
      bind:value={curveFilter}
    />
    <Button name="New|Nueva"
      color="green"
      icon="icon-[fa--plus]"
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
    >
      {#snippet cellRenderer(record: IProduct, col: ITableColumn<IProduct>)}
        {#if col.id === 'productChart'}
          <!-- Resolved 52-week quantities (base × seasonality, or explicit weeks) as compact bars. -->
          {@const chart = buildProductChart(planByProduct.get(record.ID))}
          {#if chart}
            <div class="h-32 w-full">
              <CellSimpleChart values={chart.values} labels={chart.labels} minValue={chart.minValue}
                barWidth={6} barGap={1} barColors={chart.colors} labelGroup={3}
              />
            </div>
          {/if}
        {/if}
      {/snippet}
    </VTable>
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
    >
      {#snippet cellRenderer(record: ISeasonalityCurve, col: ITableColumn<ISeasonalityCurve>)}
        {#if col.id === 'curveChart'}
          <!-- Forward-filled 52-week curve drawn as compact bars; labels mark the defined weeks. -->
          {@const chart = buildCurveChart(record.Curve)}
          <div class="h-38 w-full">
            <CellSimpleChart values={chart.values} labels={chart.labels} minValue={chart.minValue}
              colorScale={seasonalityColorScale} labelGroup={2}
            />
          </div>
        {/if}
      {/snippet}
    </VTable>
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
        <div class="w-36 pt-8 shrink-0 ff-mono c-gray-700 text-14">
          {tr(group.monthLabel)}
        </div>
        <div class="flex-1 grid grid-cols-3 sm:grid-cols-5 gap-4">
          {#each group.weeks as w (w)}
            <div class="relative">
              <div class="week-badge">{w}</div>
              <Input label={``} saveOn={planWeeks[w - 1]} save="value" type="number" inputCss="text-center pl-30" />
              <!-- Bar shows the resolved weekly quantity (typed value or base × seasonality). -->
              <div class="week-bar"
                style:width={`${planWeekBars[w - 1]?.lengthPercent || 0}%`}
                style:background={planWeekBars[w - 1]?.color}
              ></div>
            </div>
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
            <div class="relative">
              <div class="week-badge">{w}</div>
              <Input label={``} saveOn={curveWeeks[w - 1]} save="value" type="number"
                baseDecimals={1} inputCss="text-center pl-30"
                placeholder={String(Math.round((curvePreview[w - 1] || 1) * 100))}
              />
            </div>
          {/each}
        </div>
      </div>
    {/each}
  </div>
</Layer>

<style>
  .week-badge {
    position: absolute;
    left: 2px;
    top: 2px;
    bottom: 3px;
    z-index: 2;
    padding-bottom: 7px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    border-radius: 5px 0 0 5px;
    background-color: #ececef;
    color: #696994;
    font-size: 14px;
    font-family: 'mono', monospace;
    pointer-events: none;
  }

  .week-bar {
  	position: absolute;
	  left: 2px;
	  bottom: -1px;
	  height: 4px;
	  max-width: calc(100% - 4px);
	  border-radius: 2px;
	  pointer-events: none;
	  transition: width 0.15s ease;
	  z-index: 4;
  }
</style>
