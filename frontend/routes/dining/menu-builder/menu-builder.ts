import {
  MENU_ALIGN, MENU_BLOCK, MENU_PRODUCTS_VARIANT, type IDiningMenu, type IDiningMenuBlock,
} from '#core/dining-menu.ts'

// The carta builder's edits (docs/DINING_MENU_PLAN.md §4.3). Pure and immutable: each returns the
// new block list, and where a block was added or moved, the index the selection should follow.

// The kinds a user can drag from the "Bloques" tab, with the label their tile shows.
export const MENU_BLOCK_TILES: { kind: number, label: string, icon: string }[] = [
  { kind: MENU_BLOCK.ALL_PRODUCTS, label: 'All products|Todos los productos', icon: 'icon-[fa--th-list]' },
  { kind: MENU_BLOCK.CATEGORY, label: 'Category|Categoría', icon: 'icon-[fa--tag]' },
  { kind: MENU_BLOCK.PRODUCTS, label: 'Picked products|Productos puntuales', icon: 'icon-[fa--cutlery]' },
  { kind: MENU_BLOCK.TITLE, label: 'Title|Título', icon: 'icon-[fa--header]' },
  { kind: MENU_BLOCK.SUBTITLE, label: 'Subtitle|Subtítulo', icon: 'icon-[fa--font]' },
  { kind: MENU_BLOCK.TEXT, label: 'Text|Texto', icon: 'icon-[fa--align-left]' },
  { kind: MENU_BLOCK.IMAGE, label: 'Image|Imagen', icon: 'icon-[fa--image]' },
  { kind: MENU_BLOCK.DIVIDER, label: 'Divider|Divisor', icon: 'icon-[fa--minus]' },
]

// A new block with the fields its kind needs already valid, so a dropped block saves as is
// (a category block still needs its category picked).
export function newBlock(kind: number): IDiningMenuBlock {
  switch (kind) {
    case MENU_BLOCK.ALL_PRODUCTS: return { Kind: kind, Variant: MENU_PRODUCTS_VARIANT.WITH_PHOTO }
    case MENU_BLOCK.CATEGORY: return { Kind: kind, CategoryID: 0, Variant: MENU_PRODUCTS_VARIANT.WITH_PHOTO }
    case MENU_BLOCK.PRODUCTS: return { Kind: kind, ProductIDs: [], Variant: MENU_PRODUCTS_VARIANT.WITH_PHOTO }
    case MENU_BLOCK.TITLE: return { Kind: kind, Text: 'Nuevo título', Align: MENU_ALIGN.LEFT }
    case MENU_BLOCK.SUBTITLE: return { Kind: kind, Text: 'Nuevo subtítulo', Align: MENU_ALIGN.LEFT }
    case MENU_BLOCK.TEXT: return { Kind: kind, Text: 'Escribe aquí tu texto', Align: MENU_ALIGN.LEFT }
    default: return { Kind: kind }
  }
}

export function insertBlock(blocks: IDiningMenuBlock[], block: IDiningMenuBlock, insertIndex: number) {
  const position = Math.min(Math.max(insertIndex, 0), blocks.length)
  return { blocks: [...blocks.slice(0, position), block, ...blocks.slice(position)], selectedIndex: position }
}

// Moves a block to an insertion point measured in the list as it was before the move (what the
// drop placeholder shows), so dropping a block next to itself changes nothing.
export function moveBlock(blocks: IDiningMenuBlock[], fromIndex: number, insertIndex: number) {
  const movedBlock = blocks[fromIndex]
  const targetIndex = insertIndex > fromIndex ? insertIndex - 1 : insertIndex
  const withoutMoved = blocks.filter((_, position) => position !== fromIndex)
  return insertBlock(withoutMoved, movedBlock, targetIndex)
}

export const removeBlock = (blocks: IDiningMenuBlock[], blockIndex: number) =>
  blocks.filter((_, position) => position !== blockIndex)

// A product dropped on a "Productos puntuales" block joins it (unless it is already there); dropped
// anywhere else it becomes a new block of one product at the insertion point.
export function dropProduct(blocks: IDiningMenuBlock[], productID: number, insertIndex: number, overBlockIndex?: number) {
  const overBlock = overBlockIndex === undefined ? undefined : blocks[overBlockIndex]
  if (overBlock?.Kind === MENU_BLOCK.PRODUCTS) {
    const productIDs = overBlock.ProductIDs || []
    if (productIDs.includes(productID)) return { blocks, selectedIndex: overBlockIndex! }
    const updatedBlocks = blocks.map((block, position) =>
      position === overBlockIndex ? { ...block, ProductIDs: [...productIDs, productID] } : block)
    return { blocks: updatedBlocks, selectedIndex: overBlockIndex! }
  }
  return insertBlock(blocks, { ...newBlock(MENU_BLOCK.PRODUCTS), ProductIDs: [productID] }, insertIndex)
}

export const dropCategory = (blocks: IDiningMenuBlock[], categoryID: number, insertIndex: number) =>
  insertBlock(blocks, { ...newBlock(MENU_BLOCK.CATEGORY), CategoryID: categoryID }, insertIndex)

// The index a block dragged over another would land at: before it in the top half, after in the bottom.
export const insertIndexOver = (blockIndex: number, pointerY: number, blockTop: number, blockHeight: number) =>
  pointerY < blockTop + blockHeight / 2 ? blockIndex : blockIndex + 1

// What the "Guardar" button sends: the carta as edited. Blocks the user left incomplete (a category
// block with no category, a products block with no product) are what the server refuses, so they
// are named here first for a message the user can act on.
export function incompleteBlockMessage(menu: Pick<IDiningMenu, 'Blocks'>): string {
  for (const [position, block] of menu.Blocks.entries()) {
    const blockNumber = position + 1
    if (block.Kind === MENU_BLOCK.CATEGORY && !block.CategoryID) return `El bloque ${blockNumber} necesita una categoría.`
    if (block.Kind === MENU_BLOCK.PRODUCTS && !(block.ProductIDs || []).length) return `El bloque ${blockNumber} necesita al menos un producto.`
    if (block.Kind === MENU_BLOCK.IMAGE && !block.ImageURL) return `El bloque ${blockNumber} necesita una imagen.`
  }
  return ''
}

// The drag payloads of the builder, one MIME type each so a drop knows what it received.
export const MENU_DRAG = {
  NEW_BLOCK: 'application/x-genix-menu-block',
  PRODUCT: 'application/x-genix-menu-product',
  CATEGORY: 'application/x-genix-menu-category',
  MOVE_BLOCK: 'application/x-genix-menu-move',
} as const
