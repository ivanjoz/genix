<script lang="ts">
  import { onMount } from 'svelte'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import type { ITableOrder } from '../dining.ts'
  import {
    addToRoundDraft, makeRoundRequest, menuOptions, roundDraftTotal, type IDiningMenuProduct, type IRoundDraftLine,
  } from './floor-service.ts'
  import { getDiningMenu, postTableOrderRound } from './floor-service.svelte.ts'

  // A new round for one table: pick products from the layout's menu (only what can be ordered now),
  // set units and notes, send it to the kitchen. The first round opens the table and asks for guests.
  let { layoutID, tableShapeID, isOpening, onSent, onCancel }: {
    layoutID: number
    tableShapeID: number
    isOpening: boolean
    onSent: (order: ITableOrder) => void
    onCancel: () => void
  } = $props()

  let menu = $state<IDiningMenuProduct[]>([])
  let draft = $state<IRoundDraftLine[]>([])
  let opening = $state({ Guests: 0 })
  let isSending = $state(false)
  const options = $derived(menuOptions(menu))
  const formatAmount = (cents: number) => `S/ ${formatN(cents / 100, 2)}`

  onMount(async () => {
    try {
      menu = await getDiningMenu(layoutID)
    } catch (menuError) {
      console.error('[floor-service] menu error', { layoutID, menuError })
    }
  })

  async function sendRound() {
    if (!draft.some((line) => line.Units > 0) || isSending) return
    isSending = true
    try {
      onSent(await postTableOrderRound(makeRoundRequest(layoutID, tableShapeID, Number(opening.Guests) || 0, draft)))
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
      buttonDeleteHandler: (line) => { draft = draft.filter((candidate) => candidate.key !== line.key) } },
  ]
</script>

<div class="flex flex-col gap-10" aria-label="New round for the table: products, units and notes">
  <div class="flex flex-wrap items-end gap-8">
    <SearchSelect css="flex-1 min-w-240" label="Add a product|Agregar producto" options={options} keyId="ID" keyName="Name"
      placeholder={menu.length ? ':: buscar ::' : tr('Loading the menu...|Cargando la carta...')} clearOnSelect
      onChange={(option: { ID: number }) => { draft = addToRoundDraft(draft, menu, option.ID) }} />
    {#if isOpening}
      <Input saveOn={opening} save="Guests" type="number" css="w-120" label="Guests|Comensales" />
    {/if}
  </div>
  <VTable {columns} data={draft} disableVirtualizer={true} estimateSize={40}
    emptyMessage="Pick products to order.|Elija los productos del pedido." />
  <div class="flex items-center gap-8">
    <span class="ff-bold">{tr('Total|Total')} {formatAmount(roundDraftTotal(draft))}</span>
    <div class="ml-auto flex gap-8">
      <Button name="Cancel|Cancelar" label="Discards this round." onClick={onCancel} />
      <Button color="green" icon="icon-[fa--paper-plane]" name="Send to kitchen|Enviar a cocina"
        label="Sends the round to the kitchen." disabled={isSending || !draft.some((line) => line.Units > 0)} onClick={sendRound} />
    </div>
  </div>
</div>
