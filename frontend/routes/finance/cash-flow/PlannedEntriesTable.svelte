<script lang="ts">
  import Button from '#components/buttons/Button.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { confirmWarn, notifyFailure } from '@genix/ui/notify'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import { dmyToUnixDay, unixDayToDMY } from '#routes/finance/loans/loans.utils.ts'
  import { CashFlowEntriesService, postCashFlowEntry, type ICashFlowEntry } from '#services/finance/cash-flow.svelte.ts'
  import { CASH_FLOW_ENTRY_KINDS } from './cash-flow.ts'

  // The planned one-off movements no record or rule predicts. The amount is typed positive; its sign
  // follows the kind. A row is saved as soon as it is complete, and every later edit saves it again.
  let { entriesService, siteOptions, today }: {
    entriesService: CashFlowEntriesService
    siteOptions: { id: number, name: string }[]
    today: number
  } = $props()

  const CURRENCY_OPTIONS = [{ id: 1, name: 'S/' }, { id: 2, name: 'US$' }]
  const kindOptions = CASH_FLOW_ENTRY_KINDS.map((kind) => ({ ...kind, name: tr(kind.name) }))
  const isInflowKind = (kindID: number) => !!CASH_FLOW_ENTRY_KINDS.find((kind) => kind.id === kindID)?.isInflow

  // Rows not saved yet: they wait for a name and an amount.
  let draftEntries = $state<ICashFlowEntry[]>([])
  const rows = $derived([...entriesService.records.filter((entry) => entry.ss > 0).sort((left, right) => left.Date - right.Date),
    ...draftEntries])

  const saveEntry = async (entry: ICashFlowEntry) => {
    entry.Amount = Math.abs(entry.Amount) * (isInflowKind(entry.Kind) ? 1 : -1)
    if (!entry.Name.trim() || !entry.Amount || !entry.Date) return
    try {
      await postCashFlowEntry(entriesService, entry)
      draftEntries = draftEntries.filter((draft) => draft !== entry)
    } catch (error) {
      notifyFailure(error as string)
    }
  }

  const removeEntry = async (entry: ICashFlowEntry) => {
    if (!entry.ID) {
      draftEntries = draftEntries.filter((draft) => draft !== entry)
      return
    }
    if (!(await confirmWarn({ title: tr('Delete entry|Eliminar entrada'), message: tr(`Delete "${entry.Name}"?|¿Eliminar "${entry.Name}"?`) }))) return
    try {
      await postCashFlowEntry(entriesService, { ...entry, ss: 0 })
    } catch (error) {
      notifyFailure(error as string)
    }
  }

  // Derived: the site options arrive after the sites sync.
  const columns = $derived.by((): ITableColumn<ICashFlowEntry>[] => [
    {
      id: 'date', header: 'Date (dd-mm-yyyy)|Fecha (dd-mm-aaaa)', headerCss: 'w-140', css: 'px-6',
      setCellCss: (entry) => entry.Date < today ? 'text-red-fg' : undefined,
      getValue: (entry) => unixDayToDMY(entry.Date),
      onCellEdit: (entry, value) => { entry.Date = dmyToUnixDay(String(value)) || entry.Date; saveEntry(entry) },
    },
    {
      id: 'name', header: 'Description|Descripción', css: 'px-6', getValue: (entry) => entry.Name,
      onCellEdit: (entry, value) => { entry.Name = String(value); saveEntry(entry) },
    },
    {
      id: 'kind', header: 'Kind|Tipo', headerCss: 'w-190', css: 'px-6', field: 'Kind', cellSelectMode: 'pick',
      cellOptions: kindOptions, cellOptionsKeyId: 'id', cellOptionsKeyName: 'name',
      onCellSelect: (entry, value) => { entry.Kind = Number(value); saveEntry(entry) },
    },
    {
      id: 'site', header: 'Site|Sede', headerCss: 'w-170', css: 'px-6', field: 'SiteID',
      cellOptions: siteOptions, cellOptionsKeyId: 'id', cellOptionsKeyName: 'name',
      onCellSelect: (entry, value) => { entry.SiteID = Number(value); saveEntry(entry) },
    },
    {
      id: 'currency', header: 'Currency|Moneda', headerCss: 'w-90', css: 'px-6', field: 'CurrencyType', cellSelectMode: 'pick',
      cellOptions: CURRENCY_OPTIONS, cellOptionsKeyId: 'id', cellOptionsKeyName: 'name',
      onCellSelect: (entry, value) => { entry.CurrencyType = Number(value); saveEntry(entry) },
    },
    {
      id: 'amount', header: 'Amount|Monto', headerCss: 'w-120', css: 'px-6 text-right', cellInputType: 'number',
      setCellCss: (entry) => entry.Amount < 0 ? 'text-red-fg' : undefined,
      getValue: (entry) => Math.abs(entry.Amount) / 100,
      render: (entry) => formatN(entry.Amount / 100, 2),
      onCellEdit: (entry, value) => { entry.Amount = Math.round((parseFloat(String(value)) || 0) * 100); saveEntry(entry) },
    },
    { id: 'actions', header: '', headerCss: 'w-52', css: 'px-4', getValue: () => '' },
  ])

  const addEntry = () => {
    draftEntries = [...draftEntries, {
      ID: 0, Date: today + 7, SiteID: 0, Amount: 0, CurrencyType: 1, Kind: 5, Name: '', ss: 1, upd: 0,
    }]
  }
</script>

<div class="flex items-center justify-between mt-8 mb-6">
  <div class="text-fg-muted">
    {tr('One-off movements no document or rule predicts: capex, dividends, capital. Past dates stop counting.|Movimientos puntuales que ningún documento ni regla predice: inversiones, dividendos, aportes. Las fechas pasadas dejan de contar.')}
  </div>
  <Button color="green" icon="icon-[fa--plus]" name="Entry|Entrada" label="Adds an empty planned entry a week from today."
    onClick={addEntry} />
</div>
<VTable {columns} data={rows} disableVirtualizer={true} estimateSize={42} css="w-full"
  emptyMessage="No planned entries yet.|Aún no hay entradas planificadas.">
  {#snippet cellRenderer(entry, column, defaultContent)}
    {#if column.id === 'actions'}
      <div class="flex justify-center">
        <Button color="red" mode="subtle" useCircle height="tiny" icon="icon-[mdi--delete]" label="Removes the entry."
          onClick={() => removeEntry(entry)} />
      </div>
    {:else}
      {defaultContent}
    {/if}
  {/snippet}
</VTable>
