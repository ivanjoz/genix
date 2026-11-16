<script lang="ts">
  import { onMount } from 'svelte'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import type { ITableOrder } from '../dining.ts'
  import { makeRoundRequest, roundDraftTotal, type IRoundDraftLine } from './floor-service.ts'
  import { getDiningMenu, postTableOrderRound, type RoundDraftState } from './floor-service.svelte.ts'

  // A new round for one table: the products come from the menu picker beside the layer (MenuPicker,
  // only what can be ordered now); here the waiter sets units and notes and sends it to the kitchen.
  // The first round opens the table and asks for guests.
  let { round, layoutID, tableShapeID, isOpening, onSent, onCancel }: {
    round: RoundDraftState
    layoutID: number
    tableShapeID: number
    isOpening: boolean
    onSent: (order: ITableOrder) => void
    onCancel: () => void
  } = $props()

  let opening = $state({ Guests: 0 })
  let isSending = $state(false)
  const formatAmount = (cents: number) => `S/ ${formatN(cents / 100, 2)}`

  // Each visit starts an empty round on a fresh menu (its units left change by the minute).
  onMount(() => {
    round.draft = []
    round.menu = []
    round.isMenuLoading = true
    round.isComposing = true
    getDiningMenu(layoutID)
      .then((menu) => { round.menu = menu })
      .catch((menuError) => console.error('[floor-service] menu error', { layoutID, menuError }))
      .finally(() => { round.isMenuLoading = false })
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

  const columns: ITableColumn<IRoundDraftLine>[] = [
    { header: 'Product|Producto', css: 'px-6', getValue: (line) => line.Name },
    { header: 'Units|Cant.', field: 'Units', width: '70px', css: 'px-6 text-right', cellInputType: 'number',
      getValue: (line) => line.Units,
      onCellEdit: (line, value, rerender) => { line.Units = Math.max(0, Math.round(Number(value)) || 0); rerender() } },
    { header: 'Note|Nota', field: 'Note', css: 'px-6', getValue: (line) => line.Note,
      onCellEdit: (line, value) => { line.Note = String(value || '').slice(0, 120) } },
    { header: 'Subtotal', width: '96px', css: 'px-6 text-right ff-mono', getValue: (line) => formatAmount(line.Units * line.Price) },
    { id: 'actions', header: '', width: '36px', css: 'text-center px-4',
      buttonDeleteHandler: (line) => { round.draft = round.draft.filter((candidate) => candidate.key !== line.key) } },
  ]
</script>

<div class="flex flex-col gap-10" aria-label="New round for the table: products, units and notes">
  {#if isOpening}
    <Input saveOn={opening} save="Guests" type="number" css="w-120" label="Guests|Comensales" />
  {/if}
  <VTable {columns} data={round.draft} disableVirtualizer={true} estimateSize={40}
    emptyMessage="Pick products from the menu.|Elija los productos de la carta." />
  <div class="flex items-center gap-8">
    <span class="ff-bold">{tr('Total|Total')} {formatAmount(roundDraftTotal(round.draft))}</span>
    <div class="ml-auto flex gap-8">
      <Button name="Cancel|Cancelar" label="Discards this round." onClick={onCancel} />
      <Button color="green" icon="icon-[fa--paper-plane]" name="Send to kitchen|Enviar a cocina"
        label="Sends the round to the kitchen." disabled={isSending || !round.draft.some((line) => line.Units > 0)} onClick={sendRound} />
    </div>
  </div>
</div>
