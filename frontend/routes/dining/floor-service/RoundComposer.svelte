<script lang="ts">
  import { onMount } from 'svelte'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import TableGrid from '#components/vTable/TableGrid.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import type { IProductPack } from '#core/product-packs.ts'
  import { QUANTITY_LINE_SCALE } from '#core/quantity.ts'
  import { packGroupRows, packGroupRowsTotal, type IPackGroupRow } from '#core/sale-packs.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import type { ITableOrder } from '../dining.ts'
  import { makeRoundRequest, removeRoundRowUnits, roundPackLines, setRoundRowUnits } from './floor-service.ts'
  import { postTableOrderRound, type RoundDraftState } from './floor-service.svelte.ts'
  import PackHeaderRow, { PACK_HEADER_ROW_HEIGHT } from './PackHeaderRow.svelte'

  // A new round for one table: the products come from the menu picker beside the layer (MenuPicker,
  // only what can be ordered now); here the waiter sets units and notes and sends it to the kitchen.
  // The first round opens the table and asks for guests. The packs this round fills show as a header
  // row over their dishes (round only: the charge also packs across rounds).
  let { round, layoutID, tableShapeID, isOpening, productCategoryIDs, packs, onSent, onCancel }: {
    round: RoundDraftState
    layoutID: number
    tableShapeID: number
    isOpening: boolean
    productCategoryIDs: (productID: number) => number[]
    packs: IProductPack[]
    onSent: (order: ITableOrder) => void
    onCancel: () => void
  } = $props()

  type IRoundRow = IPackGroupRow<ReturnType<typeof roundPackLines>[number]>

  let opening = $state({ Guests: 0 })
  let isSending = $state(false)
  const formatAmount = (cents: number) => `S/ ${formatN(cents / 100, 2)}`
  const rows = $derived(packGroupRows(roundPackLines(round.draft), productCategoryIDs, packs))
  const rowUnits = (row: IRoundRow) => row.Quantity / QUANTITY_LINE_SCALE

  // Each visit starts an empty round. The page re-syncs the menu's stock when composing starts.
  onMount(() => {
    round.draft = []
    round.isComposing = true
    return () => { round.isComposing = false }
  })

  async function sendRound() {
    if (!round.draft.some((line) => line.Units > 0) || isSending) return
    isSending = true
    try {
      onSent(await postTableOrderRound(makeRoundRequest(layoutID, tableShapeID, Number(opening.Guests) || 0, round.draft)))
    } catch (roundError) {
      console.error('[floor-service] round error', { layoutID, tableShapeID, roundError })
    } finally {
      isSending = false
    }
  }

  // A packed row is read-only (units and note): only its delete works, so a dish a pack took can still
  // be taken off the round. Every edit replaces the draft, and the packs are found again.
  const columns: ITableColumn<IRoundRow>[] = [
    { header: 'Product|Producto', css: 'px-6', getValue: (row) => row.line?.Name || '' },
    { header: 'Units|Cant.', field: 'Units', width: '70px', css: 'px-6 text-right', cellInputType: 'number',
      getValue: (row) => rowUnits(row), disableCellInteractions: (row) => row.isPacked,
      onCellEdit: (row, value) => {
        round.draft = setRoundRowUnits(round.draft, row.line!.key, rowUnits(row), Math.round(Number(value)) || 0)
      } },
    { header: 'Note|Nota', field: 'Note', css: 'px-6', getValue: (row) => row.line?.Note || '',
      disableCellInteractions: (row) => row.isPacked,
      onCellEdit: (row, value) => {
        const note = String(value || '').slice(0, 120)
        round.draft = round.draft.map((line) => line.key === row.line!.key ? { ...line, Note: note } : line)
      } },
    { header: 'Subtotal', width: '96px', css: 'px-6 text-right ff-mono', getValue: (row) => formatAmount(row.Amount) },
    { id: 'actions', header: '', width: '36px', css: 'text-center px-4',
      buttonDeleteHandler: (row) => { round.draft = removeRoundRowUnits(round.draft, row.line!.key, rowUnits(row)) } },
  ]
</script>

<div class="flex flex-col gap-10" aria-label="New round for the table: products, units and notes">
  <!-- Above the lines: guests (first round only), the total and the actions, so they stay in reach as the round grows. -->
  <div class="flex items-center gap-12">
    {#if isOpening}
      <Input saveOn={opening} save="Guests" type="number" css="w-120" label="Guests|Comensales" />
    {/if}
    <span class="ff-bold">{tr('Total|Total')} {formatAmount(packGroupRowsTotal(rows))}</span>
    <div class="ml-auto flex items-center gap-8">
      <Button name="Cancel|Cancelar" label="Discards this round." onClick={onCancel} />
      <Button color="green" icon="icon-[fa--paper-plane]" name="Send to kitchen|Enviar a cocina"
        label="Sends the round to the kitchen." disabled={isSending || !round.draft.some((line) => line.Units > 0)} onClick={sendRound} />
    </div>
  </div>
  <TableGrid {columns} data={rows} rowHeight={34} getRowId={(row) => row.key}
    getRowHeight={(row) => row.pack ? PACK_HEADER_ROW_HEIGHT : undefined}
    useRowRenderer={(row) => !!row.pack} rowRenderer={packHeaderRow}
    emptyMessage="Pick products from the menu.|Elija los productos de la carta." />
</div>

{#snippet packHeaderRow(row: IRoundRow)}
  <PackHeaderRow packName={row.pack?.Name || ''} packAmount={formatAmount(row.Amount)} />
{/snippet}
