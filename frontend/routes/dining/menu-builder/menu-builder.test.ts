import { describe, expect, it } from 'vitest'
import { MENU_BLOCK, type IDiningMenuBlock } from '#core/dining-menu.ts'
import {
  dropCategory, dropProduct, incompleteBlockMessage, insertBlock, insertIndexOver, moveBlock, newBlock, removeBlock,
} from './menu-builder.ts'

const titleBlock = (Text: string): IDiningMenuBlock => ({ Kind: MENU_BLOCK.TITLE, Text })
const texts = (blocks: IDiningMenuBlock[]) => blocks.map((block) => block.Text)

describe('moving blocks', () => {
  const blocks = [titleBlock('a'), titleBlock('b'), titleBlock('c')]

  it('moves a block down to an insertion point of the old list', () => {
    const moved = moveBlock(blocks, 0, 3)
    expect(texts(moved.blocks)).toEqual(['b', 'c', 'a'])
    expect(moved.selectedIndex).toBe(2)
  })

  it('moves a block up', () => {
    expect(texts(moveBlock(blocks, 2, 0).blocks)).toEqual(['c', 'a', 'b'])
  })

  it('dropping a block beside itself changes nothing', () => {
    expect(texts(moveBlock(blocks, 1, 1).blocks)).toEqual(['a', 'b', 'c'])
    expect(texts(moveBlock(blocks, 1, 2).blocks)).toEqual(['a', 'b', 'c'])
  })
})

describe('dropping a product', () => {
  const pickedBlock: IDiningMenuBlock = { Kind: MENU_BLOCK.PRODUCTS, ProductIDs: [7] }

  it('appends to a products block it lands on, once', () => {
    const appended = dropProduct([pickedBlock], 8, 0, 0)
    expect(appended.blocks[0].ProductIDs).toEqual([7, 8])
    expect(dropProduct(appended.blocks, 8, 0, 0).blocks[0].ProductIDs).toEqual([7, 8])
  })

  it('makes a new block of one product elsewhere', () => {
    const dropped = dropProduct([titleBlock('a')], 9, 1, 0)
    expect(dropped.blocks[1]).toMatchObject({ Kind: MENU_BLOCK.PRODUCTS, ProductIDs: [9] })
    expect(dropped.selectedIndex).toBe(1)
  })
})

describe('other edits', () => {
  it('drops a category as a new block', () => {
    expect(dropCategory([], 5, 0).blocks[0]).toMatchObject({ Kind: MENU_BLOCK.CATEGORY, CategoryID: 5 })
  })

  it('clamps an insertion point and removes a block', () => {
    expect(insertBlock([titleBlock('a')], titleBlock('b'), 99).selectedIndex).toBe(1)
    expect(texts(removeBlock([titleBlock('a'), titleBlock('b')], 0))).toEqual(['b'])
  })

  it('inserts before or after the block under the pointer', () => {
    expect(insertIndexOver(2, 110, 100, 40)).toBe(2)
    expect(insertIndexOver(2, 130, 100, 40)).toBe(3)
  })

  it('names the first incomplete block', () => {
    expect(incompleteBlockMessage({ Blocks: [titleBlock('a'), newBlock(MENU_BLOCK.CATEGORY)] })).toContain('2')
    expect(incompleteBlockMessage({ Blocks: [titleBlock('a'), newBlock(MENU_BLOCK.DIVIDER)] })).toBe('')
  })
})
