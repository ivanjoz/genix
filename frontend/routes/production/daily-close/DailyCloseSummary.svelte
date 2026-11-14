<script lang="ts">
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import type { ProductsService } from '#services/production/products.svelte.ts'
  import { COST_SOURCE } from '#services/production/production-orders.ts'
  import {
    closeTotalValue, formatSupplySubUnits, isMovementDateShifted, supplyShortage,
    type IDailyClosePreview, type IDailyCloseProduct, type IDailyCloseSupply,
  } from './daily-close.ts'

  // The preview of a close: what it produces, what it consumes and what blocks it.
  let { preview, productsService }: { preview: IDailyClosePreview, productsService: ProductsService } = $props()

  const formatAmount = (cents: number) => `S/ ${formatN((cents || 0) / 100, 2)}`
  const productName = (productID: number) => productsService.recordsMap.get(productID)?.Name || `#${productID}`
  const supplyAmount = (supply: IDailyCloseSupply, subUnits: number) =>
    formatSupplySubUnits(subUnits, supply.SubDivisor, productsService.recordsMap.get(supply.SupplyID)?.SbuUnit)

  // A product or supply short of stock is tinted: it consumes less than its recipe asks.
  const shortProductCss = (product: IDailyCloseProduct) => product.ShortSupplies ? 'bg-amber-bg' : undefined
  const shortSupplyCss = (supply: IDailyCloseSupply) => supplyShortage(supply) > 0 ? 'bg-amber-bg' : undefined

  const productColumns: ITableColumn<IDailyCloseProduct>[] = [
    { header: 'Product|Producto', css: 'px-6', setCellCss: shortProductCss, getValue: (product) => productName(product.ProductID) },
    { header: 'Units|Unidades', headerCss: 'w-100', css: 'px-6 ff-mono', align: 'right', setCellCss: shortProductCss,
      getValue: (product) => product.Units },
    { header: 'Cost from|Costo según', headerCss: 'w-140', css: 'px-6', setCellCss: shortProductCss,
      getValue: (product) => tr(product.CostSource === COST_SOURCE.STANDARD ? 'Standard cost|Costo estándar' : 'Recipe|Receta') },
    { header: 'Est. value|Valor est.', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right', setCellCss: shortProductCss,
      getValue: (product) => formatAmount(product.Value) },
  ]

  const supplyColumns: ITableColumn<IDailyCloseSupply>[] = [
    { header: 'Supply|Insumo', css: 'px-6', setCellCss: shortSupplyCss, getValue: (supply) => productName(supply.SupplyID) },
    { header: 'Needed|Necesario', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right', setCellCss: shortSupplyCss,
      getValue: (supply) => supplyAmount(supply, supply.Needed) },
    { header: 'Available|Disponible', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right', setCellCss: shortSupplyCss,
      getValue: (supply) => supplyAmount(supply, supply.Available) },
    { header: 'Consumed|Se consume', headerCss: 'w-130', css: 'px-6 ff-mono', align: 'right', setCellCss: shortSupplyCss,
      getValue: (supply) => supplyAmount(supply, supply.Consumed) },
  ]
</script>

<div class="flex flex-col gap-12" aria-label="Daily close summary: products produced, supplies consumed and blocking reasons">
  {#if (preview.BlockingReasons || []).length}
    <div class="p-10 rounded-md bg-red-bg" aria-label="Reasons the close cannot run">
      <div class="ff-bold mb-4"><i class="icon-[fa--ban] mr-6"></i>{tr('The close cannot run:|El cierre no se puede realizar:')}</div>
      <ul class="list-disc pl-20">
        {#each preview.BlockingReasons || [] as reason}<li>{reason}</li>{/each}
      </ul>
    </div>
  {/if}
  {#if isMovementDateShifted(preview)}
    <div class="p-10 rounded-md bg-amber-bg text-amber-fg">
      <i class="icon-[fa--exclamation-triangle] mr-6"></i>
      {tr(`This close will be recorded on ${formatTime(preview.MovementDate, 'd-M-Y')}, not on ${formatTime(preview.Date, 'd-M-Y')}.`
        + `|Este cierre se registrará con fecha ${formatTime(preview.MovementDate, 'd-M-Y')}, no ${formatTime(preview.Date, 'd-M-Y')}.`)}
    </div>
  {/if}
  {#if preview.IsStrict}
    <div class="text-fg-muted">
      {tr('Strict mode: a short supply blocks the close.|Modo estricto: un insumo faltante bloquea el cierre.')}
    </div>
  {/if}

  <div>
    <div class="flex items-center mb-6">
      <div class="h3">{tr('Products produced|Productos producidos')}</div>
      <div class="ml-auto">
        {tr(`${(preview.SaleOrderIDs || []).length} sales|${(preview.SaleOrderIDs || []).length} ventas`)}
        · <span class="ff-mono ff-bold">{formatAmount(closeTotalValue(preview.Products))}</span>
      </div>
    </div>
    <VTable columns={productColumns} data={preview.Products || []} disableVirtualizer={true} estimateSize={36}
      emptyMessage="No products to produce.|No hay productos por producir." />
  </div>

  <div>
    <div class="h3 mb-6">{tr('Supplies consumed|Insumos consumidos')}</div>
    <VTable columns={supplyColumns} data={preview.Supplies || []} disableVirtualizer={true} estimateSize={36}
      emptyMessage="No supplies: every product uses its standard cost.|Sin insumos: todos los productos usan su costo estándar." />
  </div>
</div>
