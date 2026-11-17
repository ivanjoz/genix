<script lang="ts" module>
  import { LINE_STATUS } from './dining.ts'

  // A line's status label colors, also used by "Por entregar"'s table cards.
  export const LINE_STATUS_CSS: Record<number, string> = {
    [LINE_STATUS.CANCELLED]: 'text-fg-muted',
    [LINE_STATUS.TO_CONFIRM]: 'bg-purple-bg text-purple-fg',
    [LINE_STATUS.ORDERED]: 'bg-amber-bg text-amber-fg',
    [LINE_STATUS.READY]: 'bg-teal-bg text-teal-fg',
    [LINE_STATUS.SERVED]: 'bg-green-bg text-green-fg',
  }
</script>

<script lang="ts">
  import type { Snippet } from 'svelte'
  import Button from '#components/buttons/Button.svelte'
  import Card from '#components/cards/Card.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatQuantity, unpackQuantityLine } from '#core/quantity.ts'
  import { formatN } from '#libs/helpers.ts'
  import {
    lineAmount, lineStatusLabels, orderRounds, selectionActions,
    type ILineAction, type ITableOrderLine,
  } from './dining.ts'

  // An order's lines as cards, one per product ordered, under a separator per round (newest
  // first). Tapping a card that still has an action selects it; the buttons on top apply to the
  // selected cards they fit. Shared by "Mesas" (every action), "Pedidos" (ready / undo only) and a
  // table's "Historial", which passes no onApply: read-only, without the selection toolbar.
  let {
    lines, productName, actionsOf = () => [], onApply, selectedLineIDs = $bindable([]),
    focusedLineID = 0, isBusy = false, showPrice = false, columns = 2, toolbar,
  }: {
    lines: ITableOrderLine[]
    productName: (productID: number) => string
    actionsOf?: (line: ITableOrderLine) => ILineAction[]
    onApply?: (status: number, lineIDs: number[]) => void
    selectedLineIDs?: number[]
    // The card the keyboard is on (kitchen display).
    focusedLineID?: number
    isBusy?: boolean
    showPrice?: boolean
    // Cards per row. The kitchen display keeps 2: its arrow keys move by DETAIL_COLUMNS.
    columns?: number
    // Buttons that do not depend on the selection, left of the selection's ("Listo todo").
    toolbar?: Snippet
  } = $props()

  const buttons = $derived(selectionActions(lines, selectedLineIDs, actionsOf))
  const isSelectable = (line: ITableOrderLine) => actionsOf(line).length > 0

  function toggleSelected(lineID: number) {
    selectedLineIDs = selectedLineIDs.includes(lineID)
      ? selectedLineIDs.filter((selectedID) => selectedID !== lineID)
      : [...selectedLineIDs, lineID]
  }

  // "Ready|Listo" → "Ready (2)|Listo (2)": the count goes into both languages.
  const withCount = (name: string, count: number) => name.split('|').map((part) => `${part} (${count})`).join('|')
  const formatAmount = (cents: number) => `S/ ${formatN(cents / 100, 2)}`
</script>

<!-- min-h-0 lets a height-bound parent shrink it: the toolbar stays put and only the rounds scroll. -->
<div class="flex flex-col gap-10 min-h-0" aria-label="The order's products: select cards, then apply an action">
  {#if onApply}
    <div class="flex flex-wrap items-center gap-8 min-h-40 pb-8 border-b-2 border-blue-solid">
      {@render toolbar?.()}
      <div class="ml-auto flex flex-wrap items-center gap-8">
        {#each buttons as button (button.name)}
          <Button name={withCount(button.name, button.lineIDs.length)} label={`"${button.name.split('|')[0]}" on the selected products it fits.`}
            color={button.color} mode={button.status === LINE_STATUS.CANCELLED ? 'subtle' : undefined}
            disabled={isBusy} onClick={() => onApply(button.status, button.lineIDs)} />
        {:else}
          <span class="text-fg-muted">{tr('Tap the products to select them.|Toca los productos para seleccionarlos.')}</span>
        {/each}
      </div>
    </div>
  {/if}

  <!-- The 4px padding keeps the keyboard focus ring from being clipped by the scroll box. -->
  <div class="flex flex-col gap-10 min-h-0 overflow-y-auto p-4 -m-4" aria-label="The order's rounds">
    {#each orderRounds(lines).toReversed() as { round, lines: roundLines } (round)}
      <div class="flex items-center gap-8 text-fg-muted" aria-label="Round {round}">
        <span>{tr('Round|Ronda')} {round}</span>
        <div class="flex-1 border-t border-line"></div>
      </div>
      <div class="grid gap-8" style="grid-template-columns: repeat({columns}, minmax(0, 1fr))">
        {#each roundLines as line (line.LineID)}
          {@const isSelected = isSelectable(line) && selectedLineIDs.includes(line.LineID)}
          <div data-line-id={line.LineID}>
            <Card label="Selects this product." onClick={isSelectable(line) ? () => toggleSelected(line.LineID) : undefined}
              css="h-full flex flex-col gap-6 p-10 rounded-md border-2
                {isSelected ? 'border-blue-solid bg-blue-bg' : 'border-line bg-surface'}
                {isSelectable(line) ? 'cursor-pointer hover:border-blue-border' : 'opacity-70'}
                {focusedLineID === line.LineID ? 'ring-2 ring-offset-2 ring-blue-solid' : ''}">
              <div class="flex items-start justify-between gap-6">
                <span class="ff-bold text-[22px] leading-none {isSelected ? 'text-blue-fg' : ''}">
                  {formatQuantity(unpackQuantityLine(line.Quantity), line.SubDivisor || 1)}<span class="text-[15px]">x</span>
                </span>
                <span class="px-6 rounded {LINE_STATUS_CSS[line.Status]}">{tr(lineStatusLabels[line.Status])}</span>
              </div>
              <div class="text-center {line.Status === LINE_STATUS.CANCELLED ? 'line-through text-fg-muted' : ''}">
                {productName(line.ProductID)}
              </div>
              {#if line.Note}<div class="text-center text-fg-muted">{line.Note}</div>{/if}
              {#if showPrice}<div class="mt-auto text-right ff-mono text-fg-muted">{formatAmount(lineAmount(line))}</div>{/if}
            </Card>
          </div>
        {/each}
      </div>
    {/each}
  </div>
</div>
