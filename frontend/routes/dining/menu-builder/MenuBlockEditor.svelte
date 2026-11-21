<script lang="ts">
  import { useUI } from '@genix/ui'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import ColorPicker from '#components/form/ColorPicker.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import SelectBox from '#components/form/SelectBox.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import ImagePickerModal from '#domain/ImagePickerModal.svelte'
  import { formatN } from '#libs/helpers.ts'
  import {
    MENU_ALIGN, MENU_BLOCK, MENU_FONT, MENU_PRODUCTS_VARIANT, type IDiningMenu, type IMenuCatalogProduct, type IMenuCategory,
  } from '#core/dining-menu.ts'
  import { MENU_BLOCK_TILES, removeBlock } from './menu-builder.ts'
  import type { MenuBuilderState } from './menu-builder.svelte.ts'

  // The builder's right panel: the selected block's fields, or — with no block selected — the carta's
  // own name and theme. Edits land straight on the draft; saving is the page's.
  let { builder, menu, products, categories }: {
    builder: MenuBuilderState, menu: IDiningMenu, products: IMenuCatalogProduct[], categories: IMenuCategory[]
  } = $props()

  const ui = useUI()
  const imagePickerModalId = ui.nextComponentId()

  const block = $derived(builder.selectedIndex >= 0 ? menu.Blocks[builder.selectedIndex] : undefined)
  const blockLabel = $derived(MENU_BLOCK_TILES.find((tile) => tile.kind === block?.Kind)?.label || '')
  const productByID = $derived(new Map(products.map((product) => [product.ID, product])))

  const variantOptions = [
    { id: MENU_PRODUCTS_VARIANT.COMPACT, name: 'Compact list|Lista compacta' },
    { id: MENU_PRODUCTS_VARIANT.WITH_PHOTO, name: 'List with photo|Lista con foto' },
    { id: MENU_PRODUCTS_VARIANT.CARDS, name: 'Cards (2 columns)|Tarjetas (2 columnas)' },
  ]
  const alignOptions = [{ id: MENU_ALIGN.LEFT, name: 'Left|Izquierda' }, { id: MENU_ALIGN.CENTER, name: 'Center|Centro' }]
  const fontOptions = [
    { id: MENU_FONT.SANS, name: 'Sans|Sans' }, { id: MENU_FONT.SERIF, name: 'Serif|Serif' }, { id: MENU_FONT.ROUNDED, name: 'Rounded|Redondeada' },
  ]

  // "Productos puntuales": the picked products as an editable table, plus a picker that adds one.
  const productPicker = $state({ ProductID: 0 })
  const pickedProducts = $derived((block?.ProductIDs || []).flatMap((productID) => productByID.get(productID) || []))

  function addPickedProduct() {
    if (!block || !productPicker.ProductID) return
    if (!(block.ProductIDs || []).includes(productPicker.ProductID)) block.ProductIDs = [...(block.ProductIDs || []), productPicker.ProductID]
    productPicker.ProductID = 0
  }

  function movePickedProduct(productID: number, step: -1 | 1) {
    if (!block?.ProductIDs) return
    const fromPosition = block.ProductIDs.indexOf(productID)
    const toPosition = fromPosition + step
    if (toPosition < 0 || toPosition >= block.ProductIDs.length) return
    const reordered = [...block.ProductIDs]
    ;[reordered[fromPosition], reordered[toPosition]] = [reordered[toPosition], reordered[fromPosition]]
    block.ProductIDs = reordered
  }

  const pickedProductColumns: ITableColumn<IMenuCatalogProduct>[] = [
    { header: 'Product|Producto', css: 'px-6', getValue: (product) => product.Name },
    { header: 'Price|Precio', width: '80px', css: 'px-6 text-right', getValue: (product) => formatN(product.Price / 100, 2) },
    { id: 'up', header: '', width: '32px', css: 'text-center cursor-pointer', getValue: () => '↑',
      onCellClick: (product, _index, rerender) => { movePickedProduct(product.ID, -1); rerender() } },
    { id: 'down', header: '', width: '32px', css: 'text-center cursor-pointer', getValue: () => '↓',
      onCellClick: (product, _index, rerender) => { movePickedProduct(product.ID, 1); rerender() } },
    { id: 'actions', header: '', width: '36px', css: 'text-center px-4',
      buttonDeleteHandler: (product) => { if (block) block.ProductIDs = (block.ProductIDs || []).filter((productID) => productID !== product.ID) } },
  ]

  const isProductBlock = $derived(block?.Kind === MENU_BLOCK.ALL_PRODUCTS || block?.Kind === MENU_BLOCK.CATEGORY || block?.Kind === MENU_BLOCK.PRODUCTS)
  const isTextBlock = $derived(block?.Kind === MENU_BLOCK.TITLE || block?.Kind === MENU_BLOCK.SUBTITLE || block?.Kind === MENU_BLOCK.TEXT)
</script>

<div class="flex flex-col gap-12 p-12">
  {#if block}
    <!-- Re-created per selected block so each field starts from that block's values. -->
    {#key builder.selectedIndex}
      <div class="flex items-center justify-between gap-8">
        <div class="h3">{tr(blockLabel)}</div>
        <Button color="red" mode="subtle" icon="icon-[fa--trash]" name="Remove|Quitar" label="Removes this block from the carta."
          onClick={() => builder.applyEdit(removeBlock(menu.Blocks, builder.selectedIndex), -1)} />
      </div>

      {#if isTextBlock}
        <Input saveOn={block} save="Text" label="Text|Texto" required useTextArea={block.Kind === MENU_BLOCK.TEXT} rows={5} />
        <SelectBox saveOn={block} save="Align" label="Alignment|Alineación" options={alignOptions} keyId="id" keyName="name" />
      {/if}

      {#if block.Kind === MENU_BLOCK.CATEGORY}
        <SearchSelect saveOn={block} save="CategoryID" label="Category|Categoría" options={categories} keyId="ID" keyName="Name"
          placeholder=":: seleccione ::" />
        <Input saveOn={block} save="Text" label="Heading (optional)|Encabezado (opcional)" />
      {/if}

      {#if block.Kind === MENU_BLOCK.PRODUCTS}
        <div class="flex items-end gap-8">
          <SearchSelect saveOn={productPicker} save="ProductID" css="flex-1 min-w-0" label="Add a product|Añadir un producto"
            options={products} keyId="ID" keyName="Name" placeholder=":: seleccione ::" onChange={addPickedProduct} />
        </div>
        <VTable columns={pickedProductColumns} data={pickedProducts} disableVirtualizer={true} estimateSize={36}
          emptyMessage="No products picked.|No hay productos elegidos." />
      {/if}

      {#if block.Kind === MENU_BLOCK.ALL_PRODUCTS}
        <Checkbox checked={!!block.RestOnly} onToggle={(isChecked) => { block.RestOnly = isChecked }}
          label="Only products not listed by another block|Solo los productos que no están en otro bloque" />
      {/if}

      {#if isProductBlock}
        <SelectBox saveOn={block} save="Variant" label="Style|Estilo" options={variantOptions} keyId="id" keyName="name" />
      {/if}

      {#if block.Kind === MENU_BLOCK.IMAGE}
        {#if block.ImageURL}<img class="w-full rounded-md" src={block.ImageURL} alt="" />{/if}
        <Button color="blue" icon="icon-[fa--image]" name="Pick an image|Elegir imagen" label="Opens the image gallery."
          onClick={() => ui.openModal(imagePickerModalId)} />
        <Input saveOn={block} save="Text" label="Caption (optional)|Pie de imagen (opcional)" />
        <ImagePickerModal modalId={imagePickerModalId} onSelect={(imageUrl) => { block.ImageURL = imageUrl }} />
      {/if}

      {#if block.Kind === MENU_BLOCK.DIVIDER}
        <div class="text-fg-muted">{tr('A line in the accent color between blocks.|Una línea del color de acento entre bloques.')}</div>
      {/if}
    {/key}
  {:else}
    <div class="h3">{tr('Menu|Carta')}</div>
    <Input saveOn={menu} save="Name" label="Name|Nombre" required />
    <div class="grid grid-cols-2 gap-8">
      <ColorPicker saveOn={menu.Theme} save="AccentColor" label="Accent color|Color de acento" />
      <ColorPicker saveOn={menu.Theme} save="BackgroundColor" label="Background color|Color de fondo" />
    </div>
    <SelectBox saveOn={menu.Theme} save="Font" label="Font|Fuente" options={fontOptions} keyId="id" keyName="name" />
    <div class="text-fg-muted">{tr('Click a block on the phone to edit it.|Haz clic en un bloque del celular para editarlo.')}</div>
  {/if}
</div>
