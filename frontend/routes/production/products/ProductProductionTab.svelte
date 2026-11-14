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
  import type { IProduct, IProductionSupply } from '#services/production/products.svelte.ts'
  import {
    estimateProductionUnitCost, findProductionError, makeEmptyRecipeSupply, pickRecipeSupply, recipeSupplyCost, recipeSupplyQuantity,
    setRecipeSupplyQuantity, supplyAverageCosts,
  } from './product-production.ts'

  // The "Producción" tab of a made product (docs/PRODUCTION_PLAN.md §2.5): the recipe the daily close
  // consumes for each unit sold, and what a unit costs. Edits the form's Production* fields in place;
  // the product's Save sends them.
  let { product }: { product: IProduct } = $props()

  const supplies = new SupplyMaterialService(true)
  // Every warehouse's stock rows, for the supplies' average unit cost.
  const stockRows = new ProductStockSimpleService(true)
  const averageCostBySupplyID = $derived(supplyAverageCosts(stockRows.records))
  const costEstimate = $derived(estimateProductionUnitCost(product, averageCostBySupplyID))
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
      getCellOptions: () => supplies.records, cellOptionsKeyId: 'ID', cellOptionsKeyName: 'Name',
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
      id: 'cost', header: 'Est. cost|Costo est.', width: '120px', align: 'right',
      getValue: (recipeSupply) => {
        if (!recipeSupply.SupplyID) return ''
        const lineCost = recipeSupplyCost(recipeSupply, averageCostBySupplyID.get(recipeSupply.SupplyID))
        return lineCost === undefined ? tr('no cost|sin costo') : formatCents(lineCost)
      },
    },
    { id: 'actions', header: '', width: '52px', getValue: () => '' },
  ]
</script>

<div class="grid grid-cols-24 gap-x-10 gap-y-10 mt-6 md:mt-16" aria-label="Product production">
  <Input label="Yield (portions)|Rinde (porciones)" saveOn={product} save="ProductionYieldPortions" type="number"
    css="col-span-24 md:col-span-8" />
  <Input label="Extra cost per unit|Costo extra por unidad" saveOn={product} save="ProductionExtraCostPerUnit" type="number"
    baseDecimals={2} css="col-span-12 md:col-span-8" />
  <!-- When set, the standard cost is the unit cost whatever the supplies cost; they are still consumed. -->
  <Input label="Standard cost|Costo estándar" saveOn={product} save="ProductionStandardCost" type="number"
    baseDecimals={2} css="col-span-12 md:col-span-8" />
</div>

<div class="flex items-center justify-between mt-16 mb-6">
  <div>
    <h3 class="font-bold">{tr('Recipe|Receta')}</h3>
    <div class="text-fg-muted">{tr('Supplies for the portions above.|Insumos para las porciones indicadas.')}</div>
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

<!-- What one unit costs: the standard cost, or the recipe at the supplies' average cost. -->
<div class="flex items-center justify-between mt-16 p-12 rounded-md bg-gray-100" aria-label="Estimated unit cost">
  <div class="font-bold">{tr('Estimated cost per unit|Costo estimado por unidad')}</div>
  {#if productionError}
    <div class="text-red-600">{tr(productionError)}</div>
  {:else if !costEstimate}
    <div class="text-red-600">{tr('No cost source: add supplies or a standard cost.|Sin fuente de costo: agregue insumos o un costo estándar.')}</div>
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
