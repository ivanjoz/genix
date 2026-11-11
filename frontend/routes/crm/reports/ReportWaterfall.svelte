<script lang="ts">
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime, todayUnixDay } from '#libs/helpers.ts'
  import type { CrmReportsState } from './reports.svelte.ts'
  import {
    afterStateOf, beforeStateOf, periodFirstDay, waterfallByPeriod, waterfallSummary, WATERFALL_STEPS,
    type WaterfallStep, type WaterfallTotals,
  } from './reports.ts'

  // R3: how the open pipeline moved from the opening to the closing of the range, step by step
  // (opening + new + increased − decreased − won − lost + reopened ± transferred = closing).

  interface Props { report: CrmReportsState }
  const { report }: Props = $props()

  const STEP_LABELS: Record<WaterfallStep, string> = {
    new: 'New|Nuevas', increased: 'Increased|Aumentos', decreased: 'Decreased|Reducciones', won: 'Won|Ganadas',
    lost: 'Lost|Perdidas', reopened: 'Reopened|Reabiertas', transferredIn: 'Transferred in|Transferidas (entrada)',
    transferredOut: 'Transferred out|Transferidas (salida)',
  }
  // Won and lost are what leaves the pipeline for good; the other steps are neutral up/down moves.
  const STEP_BAR_CSS: Partial<Record<WaterfallStep, string>> = { won: 'bg-green-500', lost: 'bg-red-500' }

  const today = todayUnixDay()
  // Amount changes move no count, so in count mode their steps are always 0 and are hidden.
  const shownSteps = $derived(WATERFALL_STEPS.filter((step) => report.showAmounts || (step !== 'increased' && step !== 'decreased')))
  const measure = (total: { count: number, amount: number } | undefined) =>
    report.showAmounts ? (total?.amount ?? 0) / 100 : (total?.count ?? 0)
  const formatValue = (value: number) => report.showAmounts ? formatN(value, 2) : String(value)

  const byPeriod = $derived(waterfallByPeriod(
    report.rowsToToday, report.selectedPipelineID, report.openNow, report.periodFrom, report.periodTo, today, report.ownerIDs,
  ))
  const summary = $derived(waterfallSummary(byPeriod))

  /** The summary as waterfall bars: each step floats from the running total before it to the one after. */
  const bars = $derived.by(() => {
    if (!summary) return []
    const opening = measure(summary.opening)
    const closing = measure(summary.closing)
    const steps = shownSteps.map((step) => ({ step, value: measure(summary.steps.get(step)) }))
    // The scale spans every running total, so no bar leaves the track.
    let running = opening
    const runningTotals = [opening]
    for (const { value } of steps) runningTotals.push(running += value)
    const scaleMin = Math.min(0, ...runningTotals)
    const scaleSpan = (Math.max(0, ...runningTotals) - scaleMin) || 1
    const barOf = (label: string, from: number, to: number, value: number, barCss: string) => ({
      label, value, barCss,
      left: (Math.min(from, to) - scaleMin) * 100 / scaleSpan,
      width: Math.abs(to - from) * 100 / scaleSpan,
    })
    running = opening
    return [
      barOf(tr('Opening|Apertura'), 0, opening, opening, 'bg-slate-500'),
      ...steps.map(({ step, value }) => {
        const from = running
        running += value
        return barOf(tr(STEP_LABELS[step]), from, running, value, STEP_BAR_CSS[step] ?? (value < 0 ? 'bg-amber-500' : 'bg-sky-500'))
      }),
      barOf(tr('Closing|Cierre'), 0, closing, closing, 'bg-slate-500'),
    ]
  })

  interface WaterfallRow {
    key: number
    label: string
    totals: WaterfallTotals
  }

  // A range spanning several years labels each month with its year.
  const monthLabelLayout = $derived(Math.floor(report.periodFrom / 100) === Math.floor(report.periodTo / 100) ? 'M' : 'M y')
  const monthRows = $derived([...byPeriod].sort(([left], [right]) => left - right)
    .map(([period, totals]): WaterfallRow => ({ key: period, label: formatTime(periodFirstDay(period), monthLabelLayout) as string, totals })))

  // One row per seller who had an open deal or a movement in the range, each walked on its own.
  const sellerRows = $derived.by((): WaterfallRow[] => {
    const ownerIDs = new Set([
      ...report.openNow.map((opportunity) => opportunity.OwnerUserID),
      ...report.rowsInRange.flatMap((row) => [beforeStateOf(row)?.ownerID ?? 0, afterStateOf(row).ownerID]),
    ])
    return [...ownerIDs].filter((ownerID) => ownerID && (!report.ownerIDs.length || report.ownerIDs.includes(ownerID)))
      .map((ownerID) => ({
        key: ownerID, label: report.ownerName(ownerID),
        totals: waterfallSummary(waterfallByPeriod(report.rowsToToday, report.selectedPipelineID, report.openNow,
          report.periodFrom, report.periodTo, today, [ownerID])),
      }))
      .filter((row): row is WaterfallRow => !!row.totals && (measure(row.totals.opening) !== 0 || measure(row.totals.closing) !== 0
        || [...row.totals.steps.values()].some((total) => measure(total) !== 0)))
      .sort((left, right) => left.label.localeCompare(right.label))
  })

  const signedColumn = (header: string, valueOf: (row: WaterfallRow) => number): ITableColumn<WaterfallRow> => ({
    header, css: 'px-6 text-right ff-mono',
    getValue: (row) => formatValue(valueOf(row)),
    setCellCss: (row) => valueOf(row) < 0 ? 'text-red-500' : undefined,
  })
  const tableColumns = (firstHeader: string): ITableColumn<WaterfallRow>[] => [
    { header: firstHeader, css: 'px-6', highlight: true, getValue: (row) => row.label },
    signedColumn('Opening|Apertura', (row) => measure(row.totals.opening)),
    ...shownSteps.map((step) => signedColumn(STEP_LABELS[step], (row) => measure(row.totals.steps.get(step)))),
    signedColumn('Closing|Cierre', (row) => measure(row.totals.closing)),
  ]
</script>

<div class="flex flex-col gap-16" aria-label="Pipeline waterfall">
  {#if !summary}
    <div class="text-fg-muted p-16">{tr('The selected months are in the future.|Los meses elegidos están en el futuro.')}</div>
  {:else}
    <div class="bg-white/80 rounded-lg p-12 flex flex-col gap-6" aria-label="Waterfall chart">
      {#each bars as bar (bar.label)}
        <div class="grid grid-cols-[200px_1fr_120px] items-center gap-8 group">
          <span class="truncate">{bar.label}</span>
          <div class="relative h-18">
            <div class="absolute top-0 h-full rounded-[4px] {bar.barCss} group-hover:opacity-80"
              style="left: {bar.left}%; width: max({bar.width}%, 2px)" title="{bar.label}: {formatValue(bar.value)}"></div>
          </div>
          <span class="text-right ff-mono" class:text-red-500={bar.value < 0}>{formatValue(bar.value)}</span>
        </div>
      {/each}
    </div>

    {#key shownSteps.join()}
      <VTable columns={tableColumns(tr('Month|Mes'))} data={monthRows} css="w-full" />
      {#if sellerRows.length}
        <VTable columns={tableColumns(tr('Seller|Vendedor'))} data={sellerRows} css="w-full" />
      {/if}
    {/key}
  {/if}
</div>
