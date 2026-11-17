<script lang="ts">
  import Card from '#components/cards/Card.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatQuantity, unpackQuantityLine } from '#core/quantity.ts'
  import { LINE_STATUS_CSS } from '../OrderLineCards.svelte'
  import { lineStatusLabels, waitMinutes } from '../dining.ts'
  import type { ITableToServe } from './floor-service.ts'

  // "Por entregar"'s left half: one card per table still owed a dish (tablesToServe), the
  // longest-waiting first, with a preview of its dishes. A tap opens the whole table on the right.
  let { tables, selectedTableShapeID, productName, onSelect }: {
    tables: ITableToServe[]
    selectedTableShapeID: number
    productName: (productID: number) => string
    onSelect: (tableShapeID: number) => void
  } = $props()

  // A card previews this many dishes; the rest are counted.
  const PREVIEW_LINES = 4
</script>

<div class="grid gap-10 grid-cols-[repeat(auto-fill,minmax(260px,1fr))]" aria-label="Tables still owed dishes, the longest-waiting first">
  {#each tables as table (table.order.ID)}
    {@const isSelected = table.order.TableShapeID === selectedTableShapeID}
    {@const waitedMinutes = waitMinutes(table.waitTicks)}
    <Card label="Opens this table." onClick={() => onSelect(table.order.TableShapeID)}
      css="flex flex-col gap-6 p-10 rounded-md border-2 cursor-pointer hover:border-blue-border
        {isSelected ? 'border-blue-solid bg-blue-bg' : 'border-line bg-surface'}">
      <div class="flex items-center justify-between gap-8">
        <span class="ff-bold text-[20px]">{tr(`Table ${table.tableLabel}|Mesa ${table.tableLabel}`)}</span>
        {#if waitedMinutes}
          <!-- Red when the table is late, the same "Retrasado" the panel's badge shows. -->
          <span class="flex items-center gap-4 ff-bold ff-mono {table.isLate ? 'text-red-fg' : ''}" aria-label="Minutes waited">
            <i class="icon-[mdi--clock-outline] text-[18px]"></i>{waitedMinutes} min
          </span>
        {/if}
      </div>
      <div class="flex flex-col gap-2 text-[15px] leading-[20px]" aria-label="The table's pending dishes">
        {#each table.unservedLines.slice(0, PREVIEW_LINES) as line (line.LineID)}
          <div class="flex items-center gap-8">
            <span class="ff-mono w-32 shrink-0 text-right">{formatQuantity(unpackQuantityLine(line.Quantity), line.SubDivisor || 1)}</span>
            <span class="flex-1 min-w-0 truncate">{productName(line.ProductID)}</span>
            <span class="shrink-0 px-6 rounded {LINE_STATUS_CSS[line.Status]}">{tr(lineStatusLabels[line.Status])}</span>
          </div>
        {/each}
        {#if table.unservedLines.length > PREVIEW_LINES}
          {@const moreCount = table.unservedLines.length - PREVIEW_LINES}
          <div class="text-fg-muted">{tr(`+${moreCount} more|+${moreCount} más`)}</div>
        {/if}
      </div>
    </Card>
  {:else}
    <div class="p-12 rounded-md bg-surface-soft text-fg-muted">{tr('Nothing to deliver.|Nada por entregar.')}</div>
  {/each}
</div>
