<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { PRODUCT_SHARED_LIST_CATEGORIA_ID } from '#core/products-lists.ts'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { SharedListsService } from '#services/business/shared-lists.svelte.ts'
  import { ProductsService } from '#services/production/products.svelte.ts'
  import { PACK_PRICE_TYPE, PACK_ROUNDING, type IProductPack } from '#services/production/product-packs.ts'
  import { ProductPacksService, postProductPack } from '#services/production/product-packs.svelte.ts'
  import {
    describePackComponents, describePackPrice, findPackError, makeEmptyPack, packComponentRowsOf, packComponentsOf,
    type IPackComponentRow,
  } from './packs.ts'

  // Packs (docs/PACKS_PLAN.md): bundles of products and/or categories priced as a whole — a menu is
  // "1 × Entrada + 1 × Plato Fondo". The sales screens apply them automatically.
  const ui = useUI()
  const PACK_LAYER_ID = 1
  const packs = new ProductPacksService(true)
  const products = new ProductsService(true)
  const sharedLists = new SharedListsService([PRODUCT_SHARED_LIST_CATEGORIA_ID], true)

  let packForm = $state(makeEmptyPack())
  let componentRows = $state<IPackComponentRow[]>([])

  const categories = $derived(sharedLists.ListaRecordsMap.get(PRODUCT_SHARED_LIST_CATEGORIA_ID) || [])
  const categoryNameOf = (categoryID: number) => sharedLists.recordsMap.get(categoryID)?.Name || `#${categoryID}`
  const productNameOf = (productID: number) => products.recordsMap.get(productID)?.Name || `#${productID}`

  // One picker for both kinds: categories first (the menu case), then products.
  const packItemOptions = $derived([
    ...categories.map((category) => ({ Key: `c${category.ID}`, Name: `${tr('Category|Categoría')}: ${category.Name}` })),
    ...products.records.filter((product) => product.ss > 0).map((product) => ({ Key: `p${product.ID}`, Name: product.Name })),
  ])

  const priceTypeOptions = [
    { ID: PACK_PRICE_TYPE.FIXED, Name: tr('Fixed pack price|Precio fijo del pack') },
    { ID: PACK_PRICE_TYPE.PERCENT_OFF, Name: tr('Percentage off|Descuento en porcentaje') },
    { ID: PACK_PRICE_TYPE.AMOUNT_OFF, Name: tr('Amount off|Descuento en monto') },
  ]
  const roundingOptions = [
    { ID: PACK_ROUNDING.NONE, Name: tr('No rounding|Sin redondeo') },
    { ID: PACK_ROUNDING.UP, Name: tr('Round up to the sol|Al sol superior') },
    { ID: PACK_ROUNDING.NEAREST, Name: tr('Round to the nearest sol|Al sol más cercano') },
    { ID: PACK_ROUNDING.DOWN, Name: tr('Round down to the sol|Al sol inferior') },
  ]
  const priceValueLabel = $derived(
    packForm.PriceType === PACK_PRICE_TYPE.FIXED ? 'Pack price (S/)|Precio del pack (S/)'
      : packForm.PriceType === PACK_PRICE_TYPE.PERCENT_OFF ? 'Discount (%)|Descuento (%)' : 'Discount (S/)|Descuento (S/)',
  )

  const openPack = (pack: IProductPack) => {
    packForm = structuredClone($state.snapshot(pack))
    componentRows = packComponentRowsOf(packForm)
    ui.openSideLayer(PACK_LAYER_ID)
  }

  const closePack = () => {
    packForm = makeEmptyPack()
    componentRows = []
  }

  const savePack = async (isDelete = false) => {
    const packToSave: IProductPack = {
      ...$state.snapshot(packForm), Components: packComponentsOf($state.snapshot(componentRows)), ss: isDelete ? 0 : 1,
    }
    if (isDelete) {
      if (!(await confirmWarn({
        title: tr('Delete|Eliminar'),
        message: tr(`Delete "${packForm.Name}"? Sales already made keep their prices.|¿Eliminar "${packForm.Name}"? Las ventas ya hechas conservan sus precios.`),
      }))) return
    } else {
      const packError = findPackError(packForm, componentRows)
      if (packError) {
        notifyFailure(tr(packError))
        return
      }
    }
    showLoading(tr('Saving pack...|Guardando pack...'))
    try {
      await postProductPack(packs, packToSave)
      ui.openSideLayer(0)
      closePack()
    } catch (saveError) {
      console.error('[product-packs] save error', { packToSave, saveError })
      notifyFailure(saveError)
    }
    hideLoading()
  }

  const packColumns: ITableColumn<IProductPack>[] = [
    { header: 'Pack', headerCss: 'w-220', css: 'px-6', highlight: true, getValue: (pack) => pack.Name },
    {
      header: 'Components|Componentes', css: 'px-6', useLineClamp: true,
      getValue: (pack) => describePackComponents(pack.Components, productNameOf, categoryNameOf),
    },
    { header: 'Price|Precio', headerCss: 'w-300', css: 'px-6 ff-mono', align: 'right', getValue: (pack) => describePackPrice(pack, tr) },
  ]

  const componentColumns: ITableColumn<IPackComponentRow>[] = [
    {
      id: 'item', header: 'Product or category|Producto o categoría', css: 'px-4', field: 'ItemKey',
      cellOptions: [], getCellOptions: () => packItemOptions, cellOptionsKeyId: 'Key', cellOptionsKeyName: 'Name',
      onCellSelect: (componentRow, value) => { componentRow.ItemKey = String(value || '') },
    },
    {
      id: 'quantity', header: 'Quantity|Cantidad', width: '110px', align: 'right', cellInputType: 'number', field: 'Quantity',
      getValue: (componentRow) => componentRow.Quantity || '',
      onCellEdit: (componentRow, value) => { componentRow.Quantity = parseInt(String(value || '0')) || 0 },
    },
    { id: 'actions', header: '', width: '52px', getValue: () => '' },
  ]
</script>

<Page title="Packs">
  <Layer type="content">
    <div class="flex items-center justify-between mb-6 gap-12 flex-wrap" aria-label="Packs toolbar with create button">
      <div class="text-fg-muted">{tr('Products or categories sold together at a fixed price or with a discount, applied automatically in sales.|Productos o categorías vendidos juntos a un precio fijo o con descuento, aplicados automáticamente en las ventas.')}</div>
      <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo" label="Opens the side layer to create a pack."
        onClick={() => openPack(makeEmptyPack())} />
    </div>
    <VTable columns={packColumns} data={packs.records} css="w-full" maxHeight="calc(100vh - 8rem - 16px)"
      selected={packForm.ID} isSelected={(pack, selectedID) => pack.ID === selectedID} onRowClick={openPack}
      emptyMessage="No packs yet.|Aún no hay packs." />
  </Layer>
</Page>

<Layer
  id={PACK_LAYER_ID}
  type="side"
  width={720}
  title={packForm.ID > 0 ? packForm.Name : 'New pack|Nuevo pack'}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  onSave={() => savePack()}
  onDelete={packForm.ID > 0 ? () => savePack(true) : undefined}
  onClose={closePack}
>
  <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Pack form">
    <Input bind:saveOn={packForm} save="Name" label="Name|Nombre" required={true} css="col-span-24" />
    <SearchSelect bind:saveOn={packForm} save="PriceType" options={priceTypeOptions} keyId="ID" keyName="Name" notEmpty={true}
      label="Price rule|Regla de precio" css="col-span-24 md:col-span-12" />
    <Input bind:saveOn={packForm} save="PriceValue" label={priceValueLabel} type="number" baseDecimals={2} required={true}
      css="col-span-24 md:col-span-12" />
    {#if packForm.PriceType !== PACK_PRICE_TYPE.FIXED}
      <!-- A fixed price is already the chosen total; only a computed one is rounded. -->
      <SearchSelect bind:saveOn={packForm} save="RoundingMode" options={roundingOptions} keyId="ID" keyName="Name" notEmpty={true}
        label="Rounding of the pack total|Redondeo del total del pack" css="col-span-24 md:col-span-12" />
    {/if}
  </div>

  <div class="flex items-center justify-between mt-16 mb-6">
    <div>
      <h3 class="font-bold">{tr('Components|Componentes')}</h3>
      <div class="text-fg-muted">{tr('A category slot takes any product of that category.|Un componente de categoría acepta cualquier producto de esa categoría.')}</div>
    </div>
    <Button color="green" icon="icon-[fa--plus]" name="Component|Componente" label="Adds an empty component row to the pack."
      onClick={() => { componentRows = [...componentRows, { ItemKey: '', Quantity: 1 }] }} />
  </div>
  <div aria-label="Pack components: quantity of each product or category">
    <VTable columns={componentColumns} data={componentRows} disableVirtualizer={true} estimateSize={40}
      emptyMessage="No components yet.|Aún no hay componentes.">
      {#snippet cellRenderer(componentRow, column, defaultContent)}
        {#if column.id === 'actions'}
          <div class="flex justify-center">
            <Button color="red" mode="subtle" useCircle height="tiny" icon="icon-[mdi--delete]" label="Removes the component from the pack."
              onClick={() => { componentRows = componentRows.filter((otherRow) => otherRow !== componentRow) }} />
          </div>
        {:else}
          {defaultContent}
        {/if}
      {/snippet}
    </VTable>
  </div>
</Layer>
