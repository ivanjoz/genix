<script lang="ts">
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { productoUnidadOptions } from '#core/products-lists.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import { ProductStockSimpleService } from '#routes/logistics/products-stock/stock-movement.ts'
  import { SupplyMaterialService } from '#routes/production/supplies-materials/supply-material.svelte.ts'
  import type { IProductionLabor, IProductionSupply } from '#services/production/products.svelte.ts'
  import { ProductionLaborTypesService } from '#services/production/production-settings.svelte.ts'
  import {
    estimateProductionUnitCost, findProductionError, makeEmptyRecipeLabor, makeEmptyRecipeSupply, pickRecipeSupply, recipeLaborCost,
    recipeSupplyCost, recipeSupplyQuantity, setRecipeSupplyQuantity, setWastePercent, supplyAverageCosts, wastePercentOf,
    type IProductionFields,
  } from './product-production.ts'

  // The "Producción" tab of a made product or a semi-finished supply (docs/MANUFACTURING_PLAN.md §2):
  // the recipe production consumes (supplies with their waste, labor minutes) and what a unit costs.
  // Edits the form's Production* fields in place; the form's Save sends them. excludeSupplyID keeps a
  // semi-finished supply out of its own recipe.
  let { product, excludeSupplyID = 0 }: { product: IProductionFields, excludeSupplyID?: number } = $props()

  const supplies = new SupplyMaterialService(true)
  const laborTypes = new ProductionLaborTypesService(true)
  // Every warehouse's stock rows, for the supplies' average unit cost.
  const stockRows = new ProductStockSimpleService(true)
  const averageCostBySupplyID = $derived(supplyAverageCosts(stockRows.records))
  const costPerHourByLaborTypeID = $derived(new Map(laborTypes.records.map((laborType) => [laborType.ID, laborType.CostPerHour])))
  const costEstimate = $derived(estimateProductionUnitCost(product, averageCostBySupplyID, costPerHourByLaborTypeID))
  const productionError = $derived(findProductionError(product))

  const unitNameByID = new Map(productoUnidadOptions.map((unitOption) => [unitOption.i, unitOption.v]))
  const numberOf = (value: string | number) => {
    const parsed = parseInt(String(value || '0'))
    return isNaN(parsed) ? 0 : parsed
  }
  const formatCents = (cents: number) => formatN(cents / 100, 2)

  const recipeColumns: ITableColumn<IProductionSupply>[] = [
    {
      id: 'supply', header: 'Supply|Insumo', css: 'px-4', field: 'SupplyID',
      getCellOptions: () => supplies.records.filter((supply) => supply.ID !== excludeSupplyID), cellOptionsKeyId: 'ID', cellOptionsKeyName: 'Name',
      onCellSelect: (recipeSupply, value) => {
        pickRecipeSupply(recipeSupply, Number(value), supplies.recordsMap.get(Number(value))?.SbuQuantity)
      },
    },
    {
      id: 'units', header: 'Units|Unidades', width: '110px', align: 'right', cellInputType: 'number',
      getValue: (recipeSupply) => recipeSupplyQuantity(recipeSupply).units,
      onCellEdit: (recipeSupply, value) => {
        setRecipeSupplyQuantity(recipeSupply, { ...recipeSupplyQuantity(recipeSupply), units: numberOf(value) })
      },
    },
    {
      // Only a supply with a sub-unit ("kg" with 1000 "g") takes a fraction of a unit.
      id: 'sub', header: 'Sub-units|Sub-unidades', width: '120px', align: 'right', cellInputType: 'number',
      getValue: (recipeSupply) => recipeSupplyQuantity(recipeSupply).sub,
      disableCellInteractions: (recipeSupply) => recipeSupply.SubDivisor <= 1,
      onCellEdit: (recipeSupply, value) => {
        setRecipeSupplyQuantity(recipeSupply, { ...recipeSupplyQuantity(recipeSupply), sub: numberOf(value) })
      },
    },
    {
      id: 'unit', header: 'Unit|Unidad', width: '140px', css: 'px-4',
      getValue: (recipeSupply) => {
        const supply = supplies.recordsMap.get(recipeSupply.SupplyID)
        if (!supply) return ''
        const unitName = unitNameByID.get(supply.UnitID) || tr('unit|unidad')
        return recipeSupply.SubDivisor > 1 ? `${unitName} · ${recipeSupply.SubDivisor} ${supply.SbuUnit}` : unitName
      },
    },
    {
      // Expected waste on top of the quantity: leather cut for shoes loses ~15%.
      id: 'waste', header: 'Waste %|Merma %', width: '96px', align: 'right', cellInputType: 'number',
      getValue: (recipeSupply) => wastePercentOf(recipeSupply),
      onCellEdit: (recipeSupply, value) => { setWastePercent(recipeSupply, parseFloat(String(value || '0'))) },
    },
    {
      id: 'cost', header: 'Est. cost|Costo est.', width: '120px', align: 'right',
      getValue: (recipeSupply) => {
        if (!recipeSupply.SupplyID) return ''
        const lineCost = recipeSupplyCost(recipeSupply, averageCostBySupplyID.get(recipeSupply.SupplyID))
        return lineCost === undefined ? tr('no cost|sin costo') : formatCents(lineCost)
      },
    },
    { id: 'actions', header: '', width: '52px', getValue: () => '' },
  ]

  const laborColumns: ITableColumn<IProductionLabor>[] = [
    {
      id: 'laborType', header: 'Labor type|Tipo de trabajo', css: 'px-4', field: 'LaborTypeID',
      getCellOptions: () => laborTypes.records, cellOptionsKeyId: 'ID', cellOptionsKeyName: 'Name',
      onCellSelect: (recipeLabor, value) => { recipeLabor.LaborTypeID = Number(value) },
    },
    {
      id: 'minutes', header: 'Minutes|Minutos', width: '110px', align: 'right', cellInputType: 'number',
      getValue: (recipeLabor) => recipeLabor.Minutes,
      onCellEdit: (recipeLabor, value) => { recipeLabor.Minutes = Math.max(numberOf(value), 0) },
    },
    {
      id: 'rate', header: 'Cost / hour|Costo / hora', width: '120px', align: 'right',
      getValue: (recipeLabor) => {
        const laborType = laborTypes.recordsMap.get(recipeLabor.LaborTypeID)
        return laborType ? formatCents(laborType.CostPerHour) : ''
      },
    },
    {
      id: 'cost', header: 'Est. cost|Costo est.', width: '120px', align: 'right',
      getValue: (recipeLabor) => {
        const laborType = laborTypes.recordsMap.get(recipeLabor.LaborTypeID)
        return laborType ? formatCents(recipeLaborCost(recipeLabor, laborType.CostPerHour)) : ''
      },
    },
    { id: 'actions', header: '', width: '52px', getValue: () => '' },
  ]
</script>

<div class="grid grid-cols-24 gap-x-10 gap-y-10 mt-6 md:mt-16" aria-label="Product production">
  <Input label="Yield (units)|Rinde (unidades)" saveOn={product} save="ProductionYieldPortions" type="number"
    css="col-span-24 md:col-span-8" />
  <!-- Overhead: what neither the supplies nor the labor lines carry (energy, rent share). -->
  <Input label="Overhead per unit|Costos indirectos por unidad" saveOn={product} save="ProductionExtraCostPerUnit" type="number"
    baseDecimals={2} css="col-span-12 md:col-span-8" />
  <!-- When set, the standard cost is the unit cost whatever the supplies cost; they are still consumed. -->
  <Input label="Standard cost|Costo estándar" saveOn={product} save="ProductionStandardCost" type="number"
    baseDecimals={2} css="col-span-12 md:col-span-8" />
</div>

<div class="flex items-center justify-between mt-16 mb-6">
  <div>
    <h3 class="font-bold">{tr('Recipe|Receta')}</h3>
    <div class="text-fg-muted">{tr('Supplies for the units above.|Insumos para las unidades indicadas.')}</div>
  </div>
  <Button color="green" icon="icon-[fa--plus]" name="Supply|Insumo" label="Adds an empty supply line to the recipe."
    onClick={() => { product.ProductionSupplies = [...product.ProductionSupplies, makeEmptyRecipeSupply()] }} />
</div>
<div aria-label="Recipe supplies">
  <VTable columns={recipeColumns} data={product.ProductionSupplies} disableVirtualizer={true} estimateSize={42}
    emptyMessage="No supplies yet.|Aún no hay insumos.">
    {#snippet cellRenderer(recipeSupply, column, defaultContent)}
      {#if column.id === 'actions'}
        <div class="flex justify-center">
          <Button color="red" mode="subtle" useCircle height="tiny" icon="icon-[mdi--delete]" label="Removes the supply from the recipe."
            onClick={() => { product.ProductionSupplies = product.ProductionSupplies.filter((line) => line !== recipeSupply) }} />
        </div>
      {:else}
        <!-- VTable routes every column with an id here once a cell is disabled (the sub-units of a supply without one). -->
        {defaultContent}
      {/if}
    {/snippet}
  </VTable>
</div>

<div class="flex items-center justify-between mt-16 mb-6">
  <div>
    <h3 class="font-bold">{tr('Labor|Mano de obra')}</h3>
    <div class="text-fg-muted">{tr('Minutes per labor type for the units above. An estimate: orders log the real hours.|Minutos por tipo de trabajo para las unidades indicadas. Es una estimación: las órdenes registran las horas reales.')}</div>
  </div>
  <Button color="green" icon="icon-[fa--plus]" name="Labor|Trabajo" label="Adds an empty labor line to the recipe."
    onClick={() => { product.ProductionLabor = [...(product.ProductionLabor || []), makeEmptyRecipeLabor()] }} />
</div>
<div aria-label="Recipe labor">
  <VTable columns={laborColumns} data={product.ProductionLabor || []} disableVirtualizer={true} estimateSize={42}
    emptyMessage="No labor yet.|Aún no hay mano de obra.">
    {#snippet cellRenderer(recipeLabor, column, defaultContent)}
      {#if column.id === 'actions'}
        <div class="flex justify-center">
          <Button color="red" mode="subtle" useCircle height="tiny" icon="icon-[mdi--delete]" label="Removes the labor line from the recipe."
            onClick={() => { product.ProductionLabor = product.ProductionLabor.filter((line) => line !== recipeLabor) }} />
        </div>
      {:else}
        {defaultContent}
      {/if}
    {/snippet}
  </VTable>
</div>

<!-- What one unit costs: the standard cost, or the recipe at the supplies' average cost and the labor types' rates. -->
<div class="flex items-center justify-between mt-16 p-12 rounded-md bg-gray-100" aria-label="Estimated unit cost">
  <div class="font-bold">{tr('Estimated cost per unit|Costo estimado por unidad')}</div>
  {#if productionError}
    <div class="text-red-600">{tr(productionError)}</div>
  {:else if !costEstimate}
    <div class="text-red-600">{tr('No cost source: add supplies, labor or a standard cost.|Sin fuente de costo: agregue insumos, mano de obra o un costo estándar.')}</div>
  {:else}
    <div class="flex items-center gap-8">
      {#if costEstimate.isStandardCost}
        <span class="text-fg-muted">{tr('Standard cost|Costo estándar')}</span>
      {:else if costEstimate.isIncomplete}
        <span class="text-amber-700">{tr('Incomplete: some supply has no cost|Incompleto: algún insumo no tiene costo')}</span>
      {/if}
      <span class="font-bold">{formatCents(costEstimate.unitCost)}</span>
    </div>
  {/if}
</div>
