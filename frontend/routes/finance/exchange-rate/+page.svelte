<script lang="ts">
  import { onMount, untrack } from 'svelte'
  import Page from '$domain/Page.svelte'
  import TableGrid from '$components/vTable/TableGrid.svelte'
  import type { ITableColumn } from '$components/vTable/types'
  import Button from '$components/buttons/Button.svelte'
  import T from '$components/misc/T.svelte'
  import { tr } from '$core/store.svelte'
  import { Loading, Notify, formatN } from '$libs/helpers'
  import { BcrpDefaultRates, ExchangeRatesService } from './exchange-rate.svelte'
  import {
    buildCalendarRows,
    buildMonthDrafts,
    calendarMonthKeys,
    dirtyDraftsToRecords,
    EXCHANGE_RATE_SCALE,
    mergeFetchedRates,
    MONTH_NAMES,
    MONTHS_OFFERED,
    monthIndexOfMonthKey,
    parseRateInput,
    WEEKDAY_NAMES,
    yearOfMonthKey,
    type ICalendarRow,
    type RateKind,
  } from './exchange-rate'

  const exchangeRates = new ExchangeRatesService(true)
  const bcrpDefaults = new BcrpDefaultRates()

  const monthKeys = calendarMonthKeys(new Date(), MONTHS_OFFERED)
  const calendarRows = buildCalendarRows(monthKeys)

  let monthDrafts = $state(buildMonthDrafts([], monthKeys))

  // The BCRP series lives on another origin and is fetched by the browser, so it is asked for
  // once the page is mounted and never during prerender.
  onMount(() => { bcrpDefaults.load(monthKeys) })

  $effect(() => {
    const fetchedRecords = exchangeRates.records
    untrack(() => {
      // A delta sync refreshes every month the user is not editing; dirty ones keep what was typed.
      mergeFetchedRates(monthDrafts, fetchedRecords)
    })
  })

  const draftByMonthKey = $derived(new Map(monthDrafts.map((draft) => [draft.ID, draft])))
  const dirtyMonthsCount = $derived(monthDrafts.filter((draft) => draft.isDirty).length)

  const monthTitle = (monthKey: number) =>
    `${tr(MONTH_NAMES[monthIndexOfMonthKey(monthKey)])} ${yearOfMonthKey(monthKey)}`

  const rateOfDay = (row: ICalendarRow, weekdayIndex: number, rateKind: RateKind): number => {
    const day = row.Days[weekdayIndex]
    if (!day) return 0
    return draftByMonthKey.get(row.MonthKey)?.[rateKind][day - 1] || 0
  }

  /** What the BCRP published that day, which is what a cell the company has not loaded shows. */
  const bcrpRateOfDay = (row: ICalendarRow, weekdayIndex: number, rateKind: RateKind): number => {
    const day = row.Days[weekdayIndex]
    if (!day) return 0
    return bcrpDefaults.ratesByMonthKey.get(row.MonthKey)?.[rateKind][day - 1] || 0
  }

  /** The rate the cell shows: the company's own if it has one, the BCRP default otherwise. */
  const shownRateOfDay = (row: ICalendarRow, weekdayIndex: number, rateKind: RateKind): number =>
    rateOfDay(row, weekdayIndex, rateKind) || bcrpRateOfDay(row, weekdayIndex, rateKind)

  const editRateOfDay = (
    row: ICalendarRow, weekdayIndex: number, rateKind: RateKind, typedValue: string | number,
  ) => {
    const day = row.Days[weekdayIndex]
    const draft = draftByMonthKey.get(row.MonthKey)
    if (!day || !draft) return
    draft[rateKind][day - 1] = parseRateInput(typedValue)
    draft.isDirty = true
  }

  // Each weekday is a group of three tracks: the day number it lands on, and its two rates.
  const rateSubcolumn = (
    weekdayIndex: number, rateKind: RateKind, header: string,
  ): ITableColumn<ICalendarRow> => ({
    id: `${rateKind}${weekdayIndex}`,
    header,
    width: 'minmax(64px, 1fr)',
    align: 'right',
    cellInputType: 'number',
    css: 'ff-mono text-[14px]',
    inputCss: 'ff-mono text-right',
    disableCellInteractions: (row) => !row.Days[weekdayIndex],
    // Editing opens on the rate that is on screen, defaults included: accepting a BCRP value
    // unchanged writes nothing, and typing over it is what makes it the company's own.
    getValue: (row) => {
      const rate = shownRateOfDay(row, weekdayIndex, rateKind)
      return rate ? rate / EXCHANGE_RATE_SCALE : ''
    },
    // Gray is the BCRP default and blue is what the company loaded: the colour is the only
    // thing that tells one from the other, since neither carries a mark of its own.
    setCellCss: (row) =>
      rateOfDay(row, weekdayIndex, rateKind) ? '_rate-own' : '_rate-bcrp',
    render: (row) => {
      const rate = shownRateOfDay(row, weekdayIndex, rateKind)
      return rate ? (formatN(rate / EXCHANGE_RATE_SCALE, 3) as string) : ''
    },
    onCellEdit: (row, typedValue) => editRateOfDay(row, weekdayIndex, rateKind, typedValue),
  })

  const columns: ITableColumn<ICalendarRow>[] = WEEKDAY_NAMES.map((weekdayName, weekdayIndex) => ({
    id: `weekday${weekdayIndex}`,
    header: weekdayName,
    subcols: [
      {
        // No label: the column under a weekday that holds a number is the day itself.
        id: `day${weekdayIndex}`,
        header: '',
        width: '38px',
        align: 'center',
        css: 'ff-mono text-[14px] _day-cell',
        setCellCss: (row) => (row.Days[weekdayIndex] ? '_day-filled' : ''),
        getValue: (row) => row.Days[weekdayIndex] || '',
      },
      rateSubcolumn(weekdayIndex, 'Buy', 'Buy|Compra'),
      rateSubcolumn(weekdayIndex, 'Sell', 'Sell|Venta'),
    ],
  }))

  const saveEditedMonths = async () => {
    const recordsToSave = dirtyDraftsToRecords(monthDrafts)
    if (recordsToSave.length === 0) {
      Notify.failure(tr('There are no changes to save|No hay cambios por guardar'))
      return
    }

    Loading.standard(tr('Saving|Guardando') + '...')
    await exchangeRates.postAndSync(recordsToSave)
    Loading.remove()

    for (const draft of monthDrafts) draft.isDirty = false
    Notify.success(tr('Exchange rates saved|Tipos de cambio guardados'))
  }
</script>

<Page title="Exchange Rate|Tipo de Cambio">
  <div class="flex flex-wrap items-center gap-10 mb-8">
    <div class="c-gray-500 text-[14px]">
      <T text="USD to PEN, 3 decimals|Dólar a Soles, 3 decimales" />
    </div>
    <div class="flex items-center gap-8 text-[14px]">
      <span class="_rate-bcrp">
        <T text="BCRP interbank (default)|Interbancario BCRP (por defecto)" />
      </span>
      <span class="_rate-own">
        <T text="Loaded by you|Cargado por usted" />
      </span>
    </div>
    <Button
      css="ml-auto"
      color="blue"
      icon="icon-[fa--save]"
      name={dirtyMonthsCount > 0
        ? tr('Save|Guardar') + ` (${dirtyMonthsCount})`
        : tr('Save|Guardar')}
      disabled={dirtyMonthsCount === 0}
      onClick={saveEditedMonths}
    />
  </div>

  <TableGrid
    data={calendarRows}
    columns={columns}
    height="calc(100vh - 150px)"
    rowHeight={30}
    getRowId={(row, rowIndex) => (row.IsMonthHeader ? `month${row.MonthKey}` : `week${rowIndex}`)}
    useRowRenderer={(row) => row.IsMonthHeader}
    {rowRenderer}
  />
</Page>

{#snippet rowRenderer(row: ICalendarRow, _rowIndex: number)}
  <div class="_month-title ff-bold text-[15px]">{monthTitle(row.MonthKey).toUpperCase()}</div>
{/snippet}

<style>
  /* The month opens on a rule, not a filled band: the calendar keeps one ground colour and the
     underline is what separates one month from the previous one. */
  ._month-title {
    width: 100%;
    /* Fill the cell: the row centers its content, so an auto-height title would leave the rule
       floating a couple of pixels above the row edge, out of line with every other grid line. */
    height: 100%;
    display: flex;
    align-items: center;
    padding-left: 10px;
    letter-spacing: 0.04em;
    color: #4b4b6b;
    border-bottom: 2px solid #9c91df;
    border-left: 1px solid #b4b4d0;
  }

  /* The day number opens each weekday group, so its left edge is the group separator. */
  :global(._day-cell) {
    border-left: 1px solid #b4b4d0;
  }

  /* Only the slots that carry a day are painted: the ones spilling into the neighbouring
     month stay blank, which is what marks where the month starts and ends. */
  :global(._day-filled) {
    background-color: #e0ddef;
    color: #2c2b37;
  }

  /* The two origins a rate can have. Nothing else separates them — the default is not stored
     and the loaded one carries no mark — so the colour is the whole distinction. */
  ._rate-bcrp,
  :global(._rate-bcrp) {
    color: #6b6b7a;
  }

  ._rate-own,
  :global(._rate-own) {
    color: #4160bb;
  }
</style>
