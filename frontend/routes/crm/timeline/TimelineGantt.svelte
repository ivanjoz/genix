<script lang="ts">
  import { tr } from '#core/store.svelte.ts'
  import { formatTime } from '#libs/helpers.ts'
  import type { ICrmPipelineStage } from '#services/crm/crm-pipeline.svelte.ts'
  import {
    endMarkCounts, OUTCOME_LEFT, screenPixelSize, snapToScreenPixel, type SellerLanes, type TimelineCard, type TimelinePoint,
  } from './timeline.ts'

  // The Gantt itself: a sticky seller column, one column per day, and each seller's lanes of cards.
  // A card is its title plus a line through the stages it entered: a dot per stage (its colour),
  // ★ won, ✕ lost, ↦ left for another pipeline, » still open. A dashed stretch is a time it spent
  // closed (lost, won or away) before reopening or coming back.

  interface Props {
    sellers: SellerLanes[]
    stages: ICrmPipelineStage[]
    fromDay: number
    toDay: number
    ownerName: (userID: number) => string
  }
  const { sellers, stages, fromDay, toDay, ownerName }: Props = $props()

  const DAY_PX = 32
  const LANE_PX = 60
  const NAME_PX = 190
  const FALLBACK_COLOR = '#94a3b8'
  const MONTH_NAMES = ['Jan|Ene', 'Feb|Feb', 'Mar|Mar', 'Apr|Abr', 'May|May', 'Jun|Jun', 'Jul|Jul', 'Aug|Ago', 'Sep|Set', 'Oct|Oct', 'Nov|Nov', 'Dec|Dic']

  // The line row: 18px tall, the line centred on y = 9. Sizes are in CSS px before snapping.
  const LINE_ROW_PX = 18
  const LINE_CENTER_PX = 9
  const TITLE_PX = 22

  // Positions and sizes are snapped to screen pixels (see snapToScreenPixel), so at 110% or 125%
  // zoom every tick and line has the same thickness. The ratio follows the zoom live.
  let devicePixelRatio = $state(typeof window === 'undefined' ? 1 : window.devicePixelRatio)
  $effect(() => {
    const zoomQuery = window.matchMedia(`(resolution: ${devicePixelRatio}dppx)`)
    const onZoomChange = () => { devicePixelRatio = window.devicePixelRatio }
    zoomQuery.addEventListener('change', onZoomChange)
    return () => zoomQuery.removeEventListener('change', onZoomChange)
  })
  const at = (cssPx: number) => snapToScreenPixel(cssPx, devicePixelRatio)
  const sizeOf = (cssPx: number) => screenPixelSize(cssPx, devicePixelRatio)
  /** A horizontal span [x1, x2) with both edges on screen pixels. */
  const spanStyle = (x1: number, x2: number) => `left: ${at(x1)}px; width: ${at(x2) - at(x1)}px`
  /** Top of something of a given height centred on the line. */
  const centeredTop = (height: number) => at(LINE_CENTER_PX - height / 2)
  const lineThickness = $derived(sizeOf(3))
  const tickWidth = $derived(sizeOf(2))
  const tickHeight = $derived(sizeOf(9))
  const dotSize = $derived(sizeOf(14))

  const stageByID = $derived(new Map(stages.map((stage) => [stage.ID, stage])))
  const colorOf = (stageID: number) => stageByID.get(stageID)?.Color || FALLBACK_COLOR
  const days = $derived(Array.from({ length: Math.max(0, toDay - fromDay + 1) }, (_, index) => fromDay + index))
  const dateOf = (unixDay: number) => new Date(unixDay * 86400000)
  const isWeekend = (unixDay: number) => [0, 6].includes(dateOf(unixDay).getUTCDay())

  /** The card's box inside the window, in px. Cards that began before the window start at its edge. */
  const boxOf = (card: TimelineCard) => {
    const start = Math.max(card.startDay, fromDay)
    const end = Math.min(card.endDay, toDay)
    return { start, left: (start - fromDay) * DAY_PX + 2, right: (end - fromDay + 1) * DAY_PX - 2, width: (end - start + 1) * DAY_PX - 4 }
  }
  /** x of a point inside its card (negative when it happened before the window: clipped). */
  const xOf = (point: TimelinePoint, boxStart: number) => (point.day - boxStart + point.offset) * DAY_PX - 2

  /** Colour of the line on a day: the stage it sat in, grey while closed or away; '' past its end. */
  const lineColorOnDay = (card: TimelineCard, unixDay: number) => {
    const current = card.points.findLast((point) => point.day <= unixDay)
    if (!current) return ''
    const isLast = current === card.points.at(-1)
    if (current.outcome) return isLast ? '' : FALLBACK_COLOR
    return colorOf(current.stageID)
  }
  /** One tick per visible day of the card, at the middle of the day, coloured by that day's stage. */
  const ticksOf = (card: TimelineCard, boxStart: number) => {
    const lastDay = Math.min(card.endDay, toDay)
    const ticks: { x: number, color: string }[] = []
    for (let unixDay = boxStart; unixDay <= lastDay; unixDay++) {
      const color = lineColorOnDay(card, unixDay)
      if (color) ticks.push({ x: (unixDay - boxStart + 0.5) * DAY_PX - 2, color })
    }
    return ticks
  }

  const describe = (card: TimelineCard) => [card.title, ...card.points.map((point) => {
    const stageName = tr(stageByID.get(point.stageID)?.Name ?? `#${point.stageID}`)
    const outcome = point.outcome === 1 ? ` ★ ${tr('won|ganada')}` : point.outcome === 2 ? ` ✕ ${tr('lost|perdida')}`
      : point.outcome === OUTCOME_LEFT ? ` ↦ ${tr('left the pipeline|salió del pipeline')}` : ''
    return `${formatTime(point.day, 'd-M-Y')} · ${stageName}${outcome}`
  })].join('\n')
</script>

<div class="overflow-auto border border-gray-200 rounded-6 bg-white" style="max-height: calc(100vh - 14rem)" aria-label="Opportunities timeline by seller">
  <div class="relative" style="width: {NAME_PX + days.length * DAY_PX}px">
    <!-- Day header -->
    <div class="flex sticky top-0 z-20 bg-white border-b border-gray-200">
      <div class="sticky left-0 z-30 bg-white shrink-0 px-8 py-4 ff-bold" style="width: {NAME_PX}px">{tr('Seller|Vendedor')}</div>
      {#each days as unixDay, index}
        {@const date = dateOf(unixDay)}
        <div class="shrink-0 text-center py-2 leading-tight {isWeekend(unixDay) ? 'bg-gray-50 text-fg-muted' : ''}" style="width: {DAY_PX}px">
          <div class="text-fg-muted" style="min-height: 18px">{date.getUTCDate() === 1 || index === 0 ? tr(MONTH_NAMES[date.getUTCMonth()]) : ''}</div>
          <div>{date.getUTCDate()}</div>
        </div>
      {/each}
    </div>

    {#each sellers as seller (seller.ownerID)}
      {@const counts = endMarkCounts(seller.lanes)}
      <div class="flex border-b border-gray-200" aria-label={`Timeline of ${ownerName(seller.ownerID)}`}>
        <div class="sticky left-0 z-10 bg-white shrink-0 px-8 py-6 border-r border-gray-200" style="width: {NAME_PX}px">
          <div class="ff-bold text-green-700">{ownerName(seller.ownerID)}</div>
          <div class="text-fg-muted flex gap-8 flex-wrap">
            <span title={tr('Won|Ganadas')}><i class="icon-[fa--star] text-yellow-500"></i> {counts.won}</span>
            <span title={tr('Lost|Perdidas')}><i class="icon-[fa--times] text-red-500"></i> {counts.lost}</span>
            <span title={tr('Still open|Abiertas')}><i class="icon-[fa--angle-double-right] text-blue-500"></i> {counts.open}</span>
            {#if counts.left}<span title={tr('Left the pipeline|Salieron del pipeline')}><i class="icon-[fa--sign-out] text-gray-500"></i> {counts.left}</span>{/if}
          </div>
        </div>
        <!-- Day grid behind the cards; weekends shaded. -->
        <div class="relative shrink-0" style="width: {days.length * DAY_PX}px; height: {seller.lanes.length * LANE_PX}px">
          {#each days as unixDay, index}
            <div class="absolute top-0 bottom-0 border-l border-gray-100 {isWeekend(unixDay) ? 'bg-gray-50' : ''}" style={spanStyle(index * DAY_PX, (index + 1) * DAY_PX)}></div>
          {/each}
          {#each seller.lanes as lane, laneIndex}
            {#each lane as card (card.opportunityID)}
              {@const box = boxOf(card)}
              {@const leadInEnd = card.startDay < fromDay ? 0.5 * DAY_PX - 2 : -Infinity}
              <!-- An open card ends at the middle of its last day (today) when that day is on screen;
                   past the window it runs to the edge. -->
              {@const openEndX = card.endDay <= toDay ? (card.endDay - box.start + 0.5) * DAY_PX - 2 : box.width - 8}
              <div class="absolute overflow-hidden"
                style="{spanStyle(box.left, box.right)}; top: {at(laneIndex * LANE_PX + 4)}px; height: {LANE_PX - 8}px"
                title={describe(card)} aria-label={`Opportunity ${card.title}`}>
                <!-- Fixed height: a line-height in CSS px would start the line row between screen pixels. -->
                <div class="truncate px-2 text-[15px] {card.endMark === 'won' ? 'text-green-600 ff-bold' : ''}" style="height: {TITLE_PX}px; line-height: {TITLE_PX}px">{card.title}</div>
                <div class="relative" style="height: {LINE_ROW_PX}px">
                  <!-- A card that began before the window: the stretch up to the first day's tick is
                       dotted, so it reads as "comes from earlier". -->
                  {#if card.startDay < fromDay && lineColorOnDay(card, fromDay)}
                    <div class="absolute" style="{spanStyle(0, leadInEnd)}; top: {centeredTop(lineThickness)}px; border-top: {lineThickness}px dotted {lineColorOnDay(card, fromDay)}"></div>
                  {/if}
                  {#each card.points as point, pointIndex}
                    {@const next = card.points[pointIndex + 1]}
                    {@const x1 = Math.max(xOf(point, box.start), leadInEnd)}
                    {@const x2 = next ? xOf(next, box.start) : card.endMark === 'open' ? openEndX : x1}
                    {#if x2 > x1}
                      <div class="absolute" style="{spanStyle(x1, x2)}; top: {centeredTop(lineThickness)}px; border-top: {lineThickness}px {point.outcome ? 'dashed' : 'solid'} {point.outcome ? FALLBACK_COLOR : colorOf(point.stageID)}"></div>
                    {/if}
                  {/each}
                  {#each ticksOf(card, box.start) as tick}
                    <div class="absolute" style="left: {at(tick.x - tickWidth / 2)}px; top: {centeredTop(tickHeight)}px; width: {tickWidth}px; height: {tickHeight}px; background: {tick.color}"></div>
                  {/each}
                  {#each card.points as point}
                    {@const x = xOf(point, box.start)}
                    {#if point.outcome === 1}
                      <i class="icon-[fa--star] absolute text-yellow-500" style="left: {at(x - 8)}px; top: {centeredTop(16)}px; width: 16px; height: 16px"></i>
                    {:else if point.outcome === 2}
                      <i class="icon-[fa--times] absolute text-red-500" style="left: {at(x - 8)}px; top: {centeredTop(16)}px; width: 16px; height: 16px"></i>
                    {:else if point.outcome === OUTCOME_LEFT}
                      <span class="absolute bg-white flex" style="left: {at(x - 9)}px; top: {centeredTop(18)}px; padding: 1px">
                        <i class="icon-[fa--sign-out] text-gray-500" style="width: 16px; height: 16px"></i>
                      </span>
                    {:else}
                      <div class="absolute rounded-full border-2 border-white" style="left: {at(x - dotSize / 2)}px; top: {centeredTop(dotSize)}px; width: {dotSize}px; height: {dotSize}px; background: {colorOf(point.stageID)}"></div>
                    {/if}
                  {/each}
                  {#if card.endMark === 'open'}
                    <span class="absolute bg-white flex" style="left: {at(openEndX - 8)}px; top: {centeredTop(18)}px; padding: 1px">
                      <i class="icon-[fa--angle-double-right] text-blue-500" style="width: 14px; height: 16px"></i>
                    </span>
                  {/if}
                </div>
              </div>
            {/each}
          {/each}
        </div>
      </div>
    {:else}
      <div class="text-fg-muted p-16">{tr('No opportunities in the selected range.|No hay oportunidades en el rango elegido.')}</div>
    {/each}
  </div>
</div>
