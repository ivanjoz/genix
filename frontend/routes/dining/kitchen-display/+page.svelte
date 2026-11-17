<script lang="ts">
  import { onMount, untrack } from 'svelte'
  import Button from '#components/buttons/Button.svelte'
  import Card from '#components/cards/Card.svelte'
  import CardsStrip from '#components/navigation/CardsStrip.svelte'
  import SelectBox from '#components/form/SelectBox.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatQuantity, unpackQuantityLine } from '#core/quantity.ts'
  import Page from '#domain/Page.svelte'
  import { nowSUnixTime } from '#libs/helpers.ts'
  import { ProductsService } from '#services/production/products.svelte.ts'
  import OrderLineCards from '../OrderLineCards.svelte'
  import { DiningLayoutsService, TableOrdersService, postTableOrderLines, readServedLayoutID, saveServedLayoutID } from '../dining.svelte.ts'
  import { LINE_STATUS, waitMinutes } from '../dining.ts'
  import {
    gridStep, kitchenLineActions, kitchenTickets, linesToMarkReady, pendingLineAfter, resolveSelectedTicketKey,
    ticketWaitLevel, type IKitchenTicket, type TWaitLevel,
  } from './kitchen-display.ts'

  // The kitchen display (docs/DINING_PLAN.md §5): the tickets still to prepare on the left, the
  // longest-waiting first; the selected one's products on the right, as cards. Driven by keyboard:
  // ↑/↓ walk the tickets, → enters one, the arrows walk its cards (← on the first column goes
  // back), Space selects a card, Enter marks ready the selected cards or the one under the keyboard.
  // Until the live bridge lands the open orders are pulled every 15 seconds while the page is visible.
  const POLL_MILLISECONDS = 15_000
  const STATIONS_ALL = 0

  const layoutsService = new DiningLayoutsService()
  const productsService = new ProductsService(true)
  let layoutID = $state(readServedLayoutID())
  let stationID = $state(STATIONS_ALL)
  let ordersService = $state<TableOrdersService>()
  let nowTime = $state(nowSUnixTime())

  let focusArea = $state<'list' | 'detail'>('list')
  let selectedTicketKey = $state('')
  let focusedLineID = $state(0)
  let selectedLineIDs = $state<number[]>([])
  let isMovingLines = $state(false)
  // The row the selected ticket was last on: when it leaves, the one that moves into it is selected.
  let selectedTicketIndex = 0

  const layout = $derived(layoutsService.recordsMap.get(layoutID))
  const tickets = $derived(layout ? kitchenTickets(ordersService?.records || [], layout, stationID, nowTime) : [])
  const selectedTicket = $derived(tickets.find((ticket) => ticket.key === selectedTicketKey))
  const stationOptions = $derived([{ ID: STATIONS_ALL, Name: 'All|Todas' }, ...(layout?.Stations || [])])

  // The only layout opens by itself, also over a remembered one that no longer exists.
  $effect(() => {
    if (layoutsService.records.length === 1 && !untrack(() => layoutsService.recordsMap.has(layoutID))) {
      layoutID = layoutsService.records[0].ID
    }
  })
  // Each layout syncs its own open orders, and starts on every station. The pick is remembered for the next reload.
  $effect(() => {
    const pickedLayoutID = layoutID
    untrack(() => {
      ordersService = pickedLayoutID ? new TableOrdersService(pickedLayoutID) : undefined
      if (pickedLayoutID) saveServedLayoutID(pickedLayoutID)
      stationID = STATIONS_ALL
    })
  })
  // The list changes under the selection (a ticket prepared here or elsewhere, a new round).
  $effect(() => {
    const currentTickets = tickets
    untrack(() => {
      const resolvedKey = resolveSelectedTicketKey(currentTickets, selectedTicketKey, selectedTicketIndex)
      if (resolvedKey !== selectedTicketKey) selectTicket(resolvedKey)
      selectedTicketIndex = Math.max(0, currentTickets.findIndex((ticket) => ticket.key === resolvedKey))
    })
  })
  // What the keyboard is on stays on screen.
  $effect(() => {
    const ticketKey = selectedTicketKey, lineID = focusedLineID
    untrack(() => {
      document.querySelector(`[data-ticket-key="${ticketKey}"]`)?.scrollIntoView({ block: 'nearest' })
      if (lineID) document.querySelector(`[data-line-id="${lineID}"]`)?.scrollIntoView({ block: 'nearest' })
    })
  })

  onMount(() => {
    const clockTimer = setInterval(() => { nowTime = nowSUnixTime() }, 2000)
    const pollTimer = setInterval(() => {
      if (document.visibilityState === 'visible') ordersService?.fetchOnline()
    }, POLL_MILLISECONDS)
    return () => { clearInterval(clockTimer); clearInterval(pollTimer) }
  })

  function selectTicket(ticketKey: string) {
    selectedTicketKey = ticketKey
    selectedLineIDs = []
    focusedLineID = 0
    focusArea = 'list'
  }

  function enterTicket(ticket: IKitchenTicket) {
    focusArea = 'detail'
    focusedLineID = ticket.pendingLineIDs[0]
  }

  // One request per action; after "ready" the keyboard moves on to the next card still in the
  // kitchen. A ticket left with nothing to prepare leaves the list, and the next one takes its row.
  async function moveTicketLines(status: number, lineIDs: number[]) {
    const ticket = selectedTicket
    if (!ticket || !lineIDs.length || isMovingLines) return
    isMovingLines = true
    try {
      const nextFocusedLineID = status === LINE_STATUS.READY ? pendingLineAfter(ticket, lineIDs) : focusedLineID
      ordersService?.applyChangedOrder(await postTableOrderLines({ OrderID: ticket.orderID, LineIDs: lineIDs, Status: status }))
      selectedLineIDs = []
      if (focusArea === 'detail') focusedLineID = nextFocusedLineID
    } catch (linesError) {
      console.error('[kitchen-display] lines error', { orderID: ticket.orderID, lineIDs, status, linesError })
    } finally {
      isMovingLines = false
    }
  }

  function onListKey(key: string) {
    const index = tickets.findIndex((ticket) => ticket.key === selectedTicketKey)
    if (key === 'ArrowUp' || key === 'ArrowDown') {
      const nextTicket = tickets[Math.min(Math.max(index + (key === 'ArrowDown' ? 1 : -1), 0), tickets.length - 1)]
      if (nextTicket) selectTicket(nextTicket.key)
    } else if ((key === 'ArrowRight' || key === 'Enter') && selectedTicket) {
      enterTicket(selectedTicket)
    }
  }

  function onDetailKey(key: string, ticket: IKitchenTicket) {
    const focusedLine = ticket.lines.find((line) => line.LineID === focusedLineID)
    if (key.startsWith('Arrow')) {
      const nextIndex = gridStep(Math.max(0, ticket.lines.indexOf(focusedLine!)), key, ticket.lines.length)
      if (nextIndex === -1) { focusArea = 'list'; focusedLineID = 0 } else focusedLineID = ticket.lines[nextIndex].LineID
    } else if (key === ' ' && focusedLine && kitchenLineActions(focusedLine, nowTime).length) {
      selectedLineIDs = selectedLineIDs.includes(focusedLineID)
        ? selectedLineIDs.filter((lineID) => lineID !== focusedLineID)
        : [...selectedLineIDs, focusedLineID]
    } else if (key === 'Enter') {
      moveTicketLines(LINE_STATUS.READY, linesToMarkReady(ticket, selectedLineIDs, focusedLineID))
    } else if (key === 'Escape') {
      if (selectedLineIDs.length) selectedLineIDs = []
      else { focusArea = 'list'; focusedLineID = 0 }
    }
  }

  const KEYBOARD_KEYS = new Set(['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Enter', ' ', 'Escape'])

  function onKeyDown(event: KeyboardEvent) {
    const target = event.target as HTMLElement
    if (!KEYBOARD_KEYS.has(event.key) || target.closest('input, textarea, select, [contenteditable="true"]')) return
    const isArrow = event.key.startsWith('Arrow')
    // A focused button or card keeps its own Enter / Space; an arrow takes the keyboard back to the page.
    if (!isArrow && event.key !== 'Escape' && target.closest('button, [role="button"]')) return
    if (isArrow) (document.activeElement as HTMLElement | null)?.blur()
    event.preventDefault()
    if (focusArea === 'detail' && selectedTicket) onDetailKey(event.key, selectedTicket)
    else onListKey(event.key)
  }

  const WAIT_CSS: Record<TWaitLevel, string> = {
    ok: 'bg-surface-soft text-fg',
    warn: 'bg-amber-bg text-amber-fg',
    late: 'bg-red-bg text-red-fg',
  }
  const productName = (productID: number) => productsService.recordsMap.get(productID)?.Name || `#${productID}`
  const pendingSummary = (ticket: IKitchenTicket) => ticket.lines
    .filter((line) => line.Status === LINE_STATUS.ORDERED)
    .map((line) => `${formatQuantity(unpackQuantityLine(line.Quantity), line.SubDivisor || 1)}x ${productName(line.ProductID)}`)
    .join(', ')
</script>

<svelte:window onkeydown={onKeyDown} />

<Page title="Orders|Pedidos" noPadding>
  <div class="flex flex-col md:flex-row md:h-[calc(100vh-var(--header-height)-4px)]">
    <div class="md:w-1/2 flex flex-col min-h-0 bg-page border-b md:border-b-0 md:border-r border-line">
      <div class="flex items-center gap-12 p-12 border-b border-line" aria-label="Pick the layout and the station">
        <CardsStrip options={layoutsService.records} selected={layoutID} keyId="ID" keyName="Name" isGrayBackground
          css="min-w-0" onSelect={(layoutOption) => { layoutID = layoutOption.ID }} />
        {#if (layout?.Stations || []).length > 1}
          <SelectBox options={stationOptions} selected={stationID} keyId="ID" keyName="Name" label="Station|Estación"
            css="ml-auto w-220 shrink-0" onChange={(station) => { stationID = station?.ID ?? STATIONS_ALL }} />
        {/if}
      </div>
      <div class="flex-1 min-h-0 overflow-auto flex flex-col gap-8 p-12" aria-label="Tickets to prepare, the longest-waiting first">
        {#each tickets as ticket (ticket.key)}
          {@const isSelected = ticket.key === selectedTicketKey}
          <div data-ticket-key={ticket.key}>
            <Card label="Opens this ticket." onClick={() => selectTicket(ticket.key)}
              css="flex items-center gap-12 px-12 py-10 rounded-md border-2 cursor-pointer hover:border-blue-border
                {isSelected ? `bg-blue-bg ${focusArea === 'list' ? 'border-blue-solid' : 'border-blue-border'}` : 'border-line bg-surface'}">
              <span class="ff-bold text-[22px] w-64 shrink-0">{ticket.tableLabel}</span>
              <div class="flex-1 min-w-0">
                <div>{tr('Round|Ronda')} {ticket.round}{ticket.stationName ? ` · ${ticket.stationName}` : ''}</div>
                <div class="text-fg-muted truncate">{pendingSummary(ticket)}</div>
              </div>
              <span class="px-8 py-2 rounded ff-bold shrink-0 {WAIT_CSS[ticketWaitLevel(ticket, layout!, nowTime)]}">
                {waitMinutes(nowTime - ticket.waitingSince)} min
              </span>
            </Card>
          </div>
        {:else}
          <div class="p-12 rounded-md bg-surface-soft text-fg-muted">
            {layout ? tr('Nothing to prepare.|Nada por preparar.') : tr('Pick a layout.|Elija un salón.')}
          </div>
        {/each}
      </div>
    </div>

    <div class="md:w-1/2 min-h-0 overflow-auto p-12 flex flex-col gap-12 bg-surface" aria-label="The selected ticket's products">
      {#if selectedTicket && layout}
        <div class="flex flex-wrap items-center gap-12">
          <span class="h2">{tr(`Table ${selectedTicket.tableLabel}|Mesa ${selectedTicket.tableLabel}`)}</span>
          <span class="text-fg-muted">{tr('Round|Ronda')} {selectedTicket.round}{selectedTicket.stationName ? ` · ${selectedTicket.stationName}` : ''}</span>
          <span class="ml-auto px-8 py-2 rounded ff-bold {WAIT_CSS[ticketWaitLevel(selectedTicket, layout, nowTime)]}">
            {waitMinutes(nowTime - selectedTicket.waitingSince)} min
          </span>
        </div>
        <OrderLineCards lines={selectedTicket.lines} {productName} actionsOf={(line) => kitchenLineActions(line, nowTime)}
          bind:selectedLineIDs focusedLineID={focusArea === 'detail' ? focusedLineID : 0} isBusy={isMovingLines}
          onApply={moveTicketLines}>
          {#snippet toolbar()}
            <Button color="blue" icon="icon-[fa--check]" name="All ready|Listo todo" label="Marks every product of this ticket ready."
              disabled={isMovingLines} onClick={() => moveTicketLines(LINE_STATUS.READY, selectedTicket.pendingLineIDs)} />
          {/snippet}
        </OrderLineCards>
        <div class="mt-auto text-fg-muted">
          {tr('↑ ↓ tickets · → enter · arrows move · Space selects · Enter ready · Esc back|↑ ↓ pedidos · → entrar · flechas mueven · Espacio selecciona · Enter listo · Esc volver')}
        </div>
      {:else}
        <div class="p-12 rounded-md bg-surface-soft text-fg-muted">{tr('Pick a ticket.|Elija un pedido.')}</div>
      {/if}
    </div>
  </div>
</Page>
