<script lang="ts">
  import PhoneFrame from '#domain/PhoneFrame.svelte'
  import {
    MENU_BLOCK, resolveDiningMenu, type IDiningMenu, type IMenuCatalogProduct, type IMenuCategory,
  } from '#core/dining-menu.ts'
  import DiningMenuView, { type IMenuBuilderHooks } from '#ecommerce/components/dining-menu/DiningMenuView.svelte'
  import {
    MENU_DRAG, dropCategory, dropProduct, insertBlock, insertIndexOver, moveBlock, newBlock, removeBlock,
  } from './menu-builder.ts'
  import type { MenuBuilderState } from './menu-builder.svelte.ts'

  // The phone preview: the carta drawn by the same renderer the customer gets, with drag-and-drop on
  // top. Every drop is one pure edit of menu-builder.ts applied to the draft.
  let { builder, menu, products, categories }: {
    builder: MenuBuilderState, menu: IDiningMenu, products: IMenuCatalogProduct[], categories: IMenuCategory[]
  } = $props()

  const resolvedBlocks = $derived(resolveDiningMenu(menu.Blocks, products, categories, 'builder'))

  let dropIndex = $state<number | null>(null)
  let appendTargetIndex = $state<number | null>(null)
  const clearDropState = () => { dropIndex = null; appendTargetIndex = null }

  const hooks: IMenuBuilderHooks = {
    get selectedIndex() { return builder.selectedIndex },
    get dropIndex() { return dropIndex },
    get appendTargetIndex() { return appendTargetIndex },
    onSelect: (blockIndex) => { builder.selectedIndex = blockIndex },
    onDragStart: (event, blockIndex) => {
      event.dataTransfer?.setData(MENU_DRAG.MOVE_BLOCK, String(blockIndex))
      if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move'
    },
    // Over a block: the top half drops before it, the bottom half after; a product over a
    // "Productos puntuales" block would join it instead, so that block lights up and no line shows.
    onDragOver: (event, blockIndex) => {
      event.preventDefault()
      event.stopPropagation()
      const isProductDrag = event.dataTransfer?.types.includes(MENU_DRAG.PRODUCT)
      if (isProductDrag && menu.Blocks[blockIndex].Kind === MENU_BLOCK.PRODUCTS) {
        appendTargetIndex = blockIndex
        dropIndex = null
        return
      }
      const blockBox = (event.currentTarget as HTMLElement).getBoundingClientRect()
      appendTargetIndex = null
      dropIndex = insertIndexOver(blockIndex, event.clientY, blockBox.top, blockBox.height)
    },
    onDragEnd: clearDropState,
  }

  // Below the last block (or on an empty carta) a drop goes to the end.
  function dragOverEnd(event: DragEvent) {
    event.preventDefault()
    appendTargetIndex = null
    dropIndex = menu.Blocks.length
  }

  function drop(event: DragEvent) {
    event.preventDefault()
    const transfer = event.dataTransfer
    const insertIndex = dropIndex ?? menu.Blocks.length
    const overBlockIndex = appendTargetIndex ?? undefined
    clearDropState()
    if (!transfer) return

    const movedIndex = transfer.getData(MENU_DRAG.MOVE_BLOCK)
    const newKind = transfer.getData(MENU_DRAG.NEW_BLOCK)
    const productID = transfer.getData(MENU_DRAG.PRODUCT)
    const categoryID = transfer.getData(MENU_DRAG.CATEGORY)
    if (movedIndex) {
      const moved = moveBlock(menu.Blocks, Number(movedIndex), insertIndex)
      builder.applyEdit(moved.blocks, moved.selectedIndex)
    } else if (newKind) {
      const inserted = insertBlock(menu.Blocks, newBlock(Number(newKind)), insertIndex)
      builder.applyEdit(inserted.blocks, inserted.selectedIndex)
    } else if (productID) {
      const dropped = dropProduct(menu.Blocks, Number(productID), insertIndex, overBlockIndex)
      builder.applyEdit(dropped.blocks, dropped.selectedIndex)
    } else if (categoryID) {
      const dropped = dropCategory(menu.Blocks, Number(categoryID), insertIndex)
      builder.applyEdit(dropped.blocks, dropped.selectedIndex)
    }
  }

  function removeSelectedOnDelete(event: KeyboardEvent) {
    if (event.key !== 'Delete' || builder.selectedIndex < 0) return
    builder.applyEdit(removeBlock(menu.Blocks, builder.selectedIndex), -1)
  }
</script>

<PhoneFrame scrollable>
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div class="min-h-full" style:background={menu.Theme.BackgroundColor} role="region" aria-label="Carta preview"
    ondragover={dragOverEnd} ondrop={drop} onkeydown={removeSelectedOnDelete}
    ondragleave={(event) => { if (event.target === event.currentTarget) clearDropState() }}>
    <DiningMenuView {resolvedBlocks} theme={menu.Theme} builder={hooks} />
  </div>
</PhoneFrame>
