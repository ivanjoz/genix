<script lang="ts">
  import { useUI } from '@genix/ui'
  import Button from '#components/buttons/Button.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatQuantity, unpackQuantityLine } from '#core/quantity.ts'
  import { formatN } from '#libs/helpers.ts'
  import { security } from '#libs/ui-runtime.svelte.ts'
  import { postTableOrderLines } from '../dining.svelte.ts'
  import {
    LINE_STATUS, lineStatusLabels, tableState, tableStateLabels, tableWaitTicks, waitMinutes,
    type IDiningLayout, type ITableOrder, type ITableOrderLine,
  } from '../dining.ts'
  import { lineActions, lineAmount, orderRounds, orderTotal } from './floor-service.ts'
  import RoundComposer from './RoundComposer.svelte'
  import TableChargeForm from './TableChargeForm.svelte'
  import { CHARGE_TABLE_SUB_ACCESS_ID, TABLE_SERVICE_ACCESS_ID } from './table-charge.ts'

  // One table's open order: its rounds with each line's state and actions, what it owes so far, and
  // a new round. A table with no order opens one with its first round.
  let { layout, tableShapeID, order, nowTime, productName, onOrderChanged }: {
    layout: IDiningLayout
    tableShapeID: number
    order: ITableOrder | undefined
    nowTime: number
    productName: (productID: number) => string
    onOrderChanged: (order: ITableOrder) => void
  } = $props()

  const ui = useUI()
  let isComposing = $state(false)
  let pendingLineID = $state(0)
  // The bill being closed: charged into a sale or annulled ("Cobrar mesa" sub-access).
  let closingMode = $state<'' | 'charge' | 'annul'>('')
  const canCloseBill = security.checkSubAcceso(TABLE_SERVICE_ACCESS_ID, CHARGE_TABLE_SUB_ACCESS_ID)

  // A closed bill leaves the floor map, and the table is free again.
  function onBillClosed(closedOrder: ITableOrder) {
    onOrderChanged(closedOrder)
    ui.openSideLayer(0)
  }
  const formatAmount = (cents: number) => `S/ ${formatN(cents / 100, 2)}`

  const LINE_STATUS_CSS: Record<number, string> = {
    [LINE_STATUS.CANCELLED]: 'text-fg-muted line-through',
    [LINE_STATUS.TO_CONFIRM]: 'bg-purple-bg text-purple-fg',
    [LINE_STATUS.ORDERED]: 'bg-amber-bg text-amber-fg',
    [LINE_STATUS.READY]: 'bg-teal-bg text-teal-fg',
    [LINE_STATUS.SERVED]: 'bg-green-bg text-green-fg',
  }

  async function moveLine(line: ITableOrderLine, status: number) {
    if (!order || pendingLineID) return
    pendingLineID = line.LineID
    try {
      onOrderChanged(await postTableOrderLines({ OrderID: order.ID, LineIDs: [line.LineID], Status: status }))
    } catch (lineError) {
      console.error('[floor-service] line error', { orderID: order.ID, lineID: line.LineID, status, lineError })
    } finally {
      pendingLineID = 0
    }
  }
</script>

<div class="flex flex-col gap-12 mt-8" aria-label="The table's order: rounds, line states and actions">
  {#if order}
    <div class="flex flex-wrap items-center gap-12">
      <span class="ff-bold">{tr(tableStateLabels[tableState(order, layout, nowTime)])}</span>
      {#if tableWaitTicks(order, nowTime)}
        <span class="text-fg-muted">{tr('Waiting|Esperando')} {waitMinutes(tableWaitTicks(order, nowTime))} min</span>
      {/if}
      {#if order.Guests}<span class="text-fg-muted">{order.Guests} {tr('guests|comensales')}</span>{/if}
      <span class="ml-auto ff-bold ff-mono">{formatAmount(orderTotal(order))}</span>
    </div>
  {/if}

  {#if order && closingMode}
    <TableChargeForm {order} mode={closingMode} onDone={onBillClosed} onCancel={() => { closingMode = '' }} />
  {:else if isComposing || !order}
    <RoundComposer layoutID={layout.ID} {tableShapeID} isOpening={!order}
      onSent={(savedOrder) => { isComposing = false; onOrderChanged(savedOrder) }}
      onCancel={() => { if (order) isComposing = false; else ui.openSideLayer(0) }} />
  {:else}
    <div class="flex flex-wrap items-center gap-8">
      <Button icon="icon-[fa--plus]" name="New round|Nuevo pedido" label="Adds a round to this table."
        onClick={() => { isComposing = true }} />
      {#if canCloseBill}
        <Button color="green" icon="icon-[fa--money]" name="Charge table|Cobrar mesa" css="ml-auto"
          label="Opens the form to close the bill into a sale." onClick={() => { closingMode = 'charge' }} />
        <Button color="red" mode="subtle" icon="icon-[fa--trash]" name="Annul order|Anular pedido"
          label="Opens the form to annul this table order." onClick={() => { closingMode = 'annul' }} />
      {/if}
    </div>
  {/if}

  {#if order}
    {#each orderRounds(order).toReversed() as { round, lines } (round)}
      <div class="rounded-md border border-line p-8 flex flex-col gap-6" aria-label="Round {round}">
        <div class="text-fg-muted">{tr('Round|Ronda')} {round}</div>
        {#each lines as line (line.LineID)}
          <div class="flex flex-wrap items-center gap-8">
            <span class="ff-mono w-40 text-right">{formatQuantity(unpackQuantityLine(line.Quantity), line.SubDivisor || 1)}</span>
            <div class="flex-1 min-w-160 {line.Status === LINE_STATUS.CANCELLED ? 'text-fg-muted line-through' : ''}">
              {productName(line.ProductID)}
              {#if line.Note}<span class="text-fg-muted"> · {line.Note}</span>{/if}
            </div>
            <span class="px-6 rounded {LINE_STATUS_CSS[line.Status]}">{tr(lineStatusLabels[line.Status])}</span>
            <span class="ff-mono w-80 text-right">{formatAmount(lineAmount(line))}</span>
            <div class="flex gap-4">
              {#each lineActions(line, nowTime) as action (action.status)}
                <Button name={action.name} label="Moves this line to another state." height="compact" color={action.color}
                  mode={action.status === LINE_STATUS.CANCELLED ? 'subtle' : undefined}
                  disabled={pendingLineID === line.LineID} onClick={() => moveLine(line, action.status)} />
              {/each}
            </div>
          </div>
        {/each}
      </div>
    {/each}
  {/if}
</div>
