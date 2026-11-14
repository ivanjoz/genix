<script lang="ts">
  import { notifyFailure } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import type { IProductionTemplateLine } from '#services/production/production-orders.ts'
  import {
    presentationIDAtColumn, runTotalUnits, runUnitsOf, setRunUnits, sizeGridColumns,
  } from './production-order.ts'
  import type { ProductionCatalogs } from './production-order.svelte.ts'

  // The size grid of a template or an order draft (docs/MANUFACTURING_PLAN.md §4, M3, M16): one row per
  // product, one column per size. Products share a column by size name; a product without that size
  // has a disabled cell, and products without sizes are planned in the "Unidades" column.
  let { productRuns = $bindable(), catalogs, isEditable }: {
    productRuns: IProductionTemplateLine[]
    catalogs: ProductionCatalogs
    isEditable: boolean
  } = $props()

  const sizeColumns = $derived(sizeGridColumns(productRuns, catalogs.recipeProductsByID))
  const presentationIDOf = (productRun: IProductionTemplateLine, columnKey: string) =>
    presentationIDAtColumn(catalogs.recipeProductsByID.get(productRun.ProductID), columnKey)

  // Picking another product drops the sizes: they were the previous product's presentations.
  const pickProduct = (productRun: IProductionTemplateLine, productID: number) => {
    if (productRuns.some((otherRun) => otherRun !== productRun && otherRun.ProductID === productID)) {
      productRun.ProductID = 0
      notifyFailure(tr('The product is already in the grid.|El producto ya está en la grilla.'))
      return
    }
    productRun.ProductID = productID
    productRun.Run = []
  }

  const columns = $derived<ITableColumn<IProductionTemplateLine>[]>([
    isEditable
      ? {
        id: 'product', header: 'Product|Producto', css: 'px-4', field: 'ProductID',
        cellOptions: catalogs.recipeProductOptions, cellOptionsKeyId: 'ID', cellOptionsKeyName: 'Name',
        onCellSelect: (productRun, value) => pickProduct(productRun, Number(value)),
      }
      : { header: 'Product|Producto', css: 'px-6', getValue: (productRun) => catalogs.productNameOf(productRun.ProductID) },
    ...sizeColumns.map((sizeColumn): ITableColumn<IProductionTemplateLine> => ({
      id: `size-${sizeColumn.key}`, header: sizeColumn.label, width: '76px', align: 'right', cellInputType: 'number',
      getValue: (productRun) => {
        const presentationID = presentationIDOf(productRun, sizeColumn.key)
        return presentationID === undefined ? '' : runUnitsOf(productRun, presentationID) || ''
      },
      setCellCss: (productRun) => presentationIDOf(productRun, sizeColumn.key) === undefined ? 'bg-surface-muted' : '',
      disableCellInteractions: (productRun) => !isEditable || presentationIDOf(productRun, sizeColumn.key) === undefined,
      onCellEdit: (productRun, value) => {
        const presentationID = presentationIDOf(productRun, sizeColumn.key)
        if (presentationID !== undefined) setRunUnits(productRun, presentationID, parseInt(String(value || '0')) || 0)
      },
    })),
    { id: 'total', header: 'Total', width: '80px', align: 'right', css: 'ff-bold', getValue: (productRun) => runTotalUnits(productRun) },
    ...(isEditable ? [{ id: 'actions', header: '', width: '52px', getValue: () => '' }] : []),
  ])
</script>

{#if isEditable}
  <div class="flex items-center justify-between mb-6">
    <div class="text-fg-muted">{tr('Units per size. Grey cells are sizes the product does not have.|Unidades por talla. Las celdas grises son tallas que el producto no tiene.')}</div>
    <Button color="green" icon="icon-[fa--plus]" name="Product|Producto" label="Adds an empty product row to the size grid."
      onClick={() => { productRuns = [...productRuns, { ProductID: 0, Run: [] }] }} />
  </div>
{/if}
<div aria-label="Size grid: units per product and size">
  <VTable {columns} data={productRuns} disableVirtualizer={true} estimateSize={40}
    emptyMessage="No products yet.|Aún no hay productos.">
    {#snippet cellRenderer(productRun, column, defaultContent)}
      {#if column.id === 'actions'}
        <div class="flex justify-center">
          <Button color="red" mode="subtle" useCircle height="tiny" icon="icon-[mdi--delete]" label="Removes the product from the grid."
            onClick={() => { productRuns = productRuns.filter((otherRun) => otherRun !== productRun) }} />
        </div>
      {:else}
        <!-- VTable routes every column with an id here once a cell is disabled. -->
        {defaultContent}
      {/if}
    {/snippet}
  </VTable>
</div>
