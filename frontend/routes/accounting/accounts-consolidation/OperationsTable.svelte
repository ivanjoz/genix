<script lang="ts">
import { confirmWarn, hideLoading, notifyFailure, notifySuccess, showLoading } from '@genix/ui/notify'
import { useUI } from '@genix/ui'
import { untrack } from 'svelte'
import { SvelteSet } from 'svelte/reactivity'
import Button from '#components/buttons/Button.svelte'
import Layer from '#components/layers/Layer.svelte'
import T from '#components/misc/T.svelte'
import VTable from '#components/vTable/VTable.svelte'
import type { ITableColumn } from '#components/vTable/types.ts'
import PCGEAccountSelect from '#domain/PCGEAccountSelect.svelte'
import { tr } from '#core/store.svelte.ts'
import { pcgeAccountLabel } from '#core/pcge.ts'
import { extractError } from '#libs/ui-runtime.svelte.ts'
import { formatN, formatTime } from '#libs/helpers.ts'
import {
  bulkClassificationTargets, classificationTargetsOf, groupOperations, isInStatusView, operationKindLabels,
  type IAccountingOperation, type IAccountingSources, type IClassificationTarget, type IOperationGroup,
} from './accounts-consolidation'
import { accountLinesCell, amountLinesCell, ENTRY_TABLE_CELL_CSS } from './accounts-consolidation.cells'
import {
  putAccountClassifications, type AccountingOperationsService, type IAccountClassification, type IEditableTarget,
} from './accounts-consolidation.svelte'
import OperationEntryLayer from './OperationEntryLayer.svelte'

// Tab Movimientos: every operation of the period, its status, and where the user classifies it —
// one by one in the side layer, or many at once through the row selection. The status view and
// the search text are set in the page's top row.
interface Props {
  operationsService: AccountingOperationsService
  sources: IAccountingSources | null
  operations: IAccountingOperation[]
  statusView: number
  filterText: string
}

const { operationsService, sources, operations, statusView, filterText }: Props = $props()
const ui = useUI()

const visibleOperations = $derived(operations.filter(operation => isInStatusView(operation, statusView)))
// Operations that share a day, kind, glosa, accounts and status read as one row.
const visibleGroups = $derived(groupOperations(visibleOperations))

// Bulk classification: the operations ticked in the table, keyed by their CUO. A new read of
// the period (another month, or after a save) starts with nothing ticked.
const selectedKeys = new SvelteSet<string | number>()
$effect(() => {
  operationsService.response
  untrack(() => selectedKeys.clear())
})
const bulk = $derived(sources
  ? bulkClassificationTargets(operations.filter(operation => selectedKeys.has(operation.Key)), sources)
  : { pcgeKind: 0, targets: [] as IClassificationTarget[], skippedCount: 0 })
let bulkForm = $state({ AccountCode: 0 })

// The group open in the side layer — a row of several operations lists them first — and the
// operation open in it, with editable copies of its classification targets.
let openGroup = $state(null as IOperationGroup | null)
let openOperation = $state(null as IAccountingOperation | null)
let editTargets = $state([] as IEditableTarget[])

const openGroupRow = (group: IOperationGroup) => {
  openGroup = group
  if (group.Operations.length === 1) { openEntry(group.Operations[0]); return }
  openOperation = null
  editTargets = []
  ui.openSideLayer(1)
}

const openEntry = (operation: IAccountingOperation) => {
  if (!sources) { return }
  openOperation = operation
  editTargets = classificationTargetsOf(operation, sources).map(target => ({
    ...target,
    CashOptionKey: target.PCGEKind || !target.AccountCode ? "" : `${target.CashMovementType}|${target.AccountCode}`,
  }))
  ui.openSideLayer(1)
}

// saveClassifications writes the accounts and re-reads the period, so every derived entry and
// the header are rebuilt from what the server stored.
const saveClassifications = async (classifications: IAccountClassification[]): Promise<boolean> => {
  showLoading(tr("Saving...|Guardando..."))
  try {
    await putAccountClassifications(classifications)
  } catch (error) {
    notifyFailure(extractError(error) || tr("The classification could not be saved.|No se pudo guardar la clasificación."))
    return false
  } finally {
    hideLoading()
  }
  await operationsService.fetch(operationsService.loadedPeriod)
  return true
}

const saveEntry = async () => {
  const classifications: IAccountClassification[] = []
  for (const target of editTargets) {
    if (target.PCGEKind) {
      if (!target.AccountCode) { notifyFailure(tr("Pick every account.|Seleccione todas las cuentas.")); return }
      classifications.push({ Kind: target.Kind, ID: target.ID, Line: target.Line, AccountCode: target.AccountCode, CashMovementType: 0 })
      continue
    }
    const [movementType, accountCode] = (target.CashOptionKey || "").split("|").map(Number)
    if (!accountCode) { notifyFailure(tr("Pick the type and account.|Seleccione el tipo y la cuenta.")); return }
    classifications.push({ Kind: target.Kind, ID: target.ID, Line: 0, AccountCode: accountCode, CashMovementType: movementType })
  }
  if (classifications.length === 0) { ui.openSideLayer(0); return }
  if (await saveClassifications(classifications)) {
    notifySuccess(tr("Classification saved.|Clasificación guardada."))
    ui.openSideLayer(0)
  }
}

const applyBulk = async () => {
  if (!bulkForm.AccountCode) { notifyFailure(tr("Pick the account to apply.|Seleccione la cuenta a aplicar.")); return }
  const confirmed = await confirmWarn({
    title: tr("Apply account|Aplicar cuenta"),
    message: `${pcgeAccountLabel(bulkForm.AccountCode)} → ${bulk.targets.length} ${tr("records|registros")}`,
  })
  if (!confirmed) { return }
  const classifications = bulk.targets.map(target => ({
    Kind: target.Kind, ID: target.ID, Line: target.Line, AccountCode: bulkForm.AccountCode, CashMovementType: 0,
  }))
  if (await saveClassifications(classifications)) {
    notifySuccess(tr("Account applied.|Cuenta aplicada."))
    bulkForm = { AccountCode: 0 }
  }
}

// Only a row of one operation can be ticked; a row of several opens its list instead.
const isSingleOperation = (group: IOperationGroup) => group.Operations.length === 1

const columns: ITableColumn<IOperationGroup>[] = [
  {
    header: "", width: "36px", css: "text-center",
    render: group => !isSingleOperation(group) ? ""
      : selectedKeys.has(group.Operations[0].Key)
        ? '<i class="icon-[fa--check-square-o] text-green-solid text-[16px]"></i>'
        : '<i class="icon-[fa--square-o] text-fg-subtle text-[16px]"></i>',
    disableCellInteractions: group => !isSingleOperation(group),
    onCellClick: group => {
      if (!isSingleOperation(group)) { openGroupRow(group); return }
      const operationKey = group.Operations[0].Key
      if (selectedKeys.has(operationKey)) { selectedKeys.delete(operationKey) } else { selectedKeys.add(operationKey) }
    },
  },
  { header: "Date|Fecha", width: "70px", getValue: group => formatTime(group.Date, "M-d") as string },
  { header: "Operation|Operación", width: "130px", getValue: group => tr(operationKindLabels[group.Kind]) },
  { header: "Description|Descripción", width: "minmax(160px, 1.4fr)", getValue: group => tr(group.Description) },
  {
    header: "Doc. Nº|Nº Doc", width: "130px", css: "ff-mono",
    render: group => !isSingleOperation(group) ? `(${group.Operations.length})`
      : group.Operations[0].DocumentLabel
        || `<span class="px-6 rounded bg-yellow-bg text-yellow-fg ff-sans">${tr("no comprobante|sin comprobante")}</span>`,
  },
  {
    header: "Debit|Debe", subcols: [
      { header: "Account|Cuenta", render: group => accountLinesCell(group.DebitLines, group.Status, group.Operations) },
      { header: "Amount|Monto", align: "right", css: "ff-mono", render: group => amountLinesCell(group.DebitLines) },
    ],
  },
  {
    header: "Credit|Haber", subcols: [
      { header: "Account|Cuenta", render: group => accountLinesCell(group.CreditLines, group.Status, group.Operations) },
      { header: "Amount|Monto", align: "right", css: "ff-mono", render: group => amountLinesCell(group.CreditLines) },
    ],
  },
]
</script>

{#if selectedKeys.size > 0}
  <!-- Bulk apply: one account for every record behind the ticked operations. -->
  <div class="flex flex-wrap items-end gap-10 mb-8 p-8 rounded-md bg-surface-soft" aria-label="Bulk classification bar">
    <div class="text-sm self-center">
      {selectedKeys.size} <T text="selected|marcadas" /> · {bulk.targets.length} <T text="records to classify|registros a clasificar" />
      {#if bulk.skippedCount > 0}
        · <span class="text-fg-muted">{bulk.skippedCount} <T text="without a classifiable account|sin cuenta clasificable" /></span>
      {/if}
    </div>
    {#if bulk.pcgeKind}
      <PCGEAccountSelect saveOn={bulkForm} save="AccountCode" kind={bulk.pcgeKind} css="w-320"
        label="Account to apply|Cuenta a aplicar" />
      <Button name="Apply|Aplicar" color="blue" icon="icon-[fa--check]" onClick={applyBulk} />
    {:else if bulk.targets.length > 0}
      <div class="text-sm text-yellow-fg self-center">
        <T text="Select operations of one kind (expenses, supplies or assets) to apply one account.|Marque operaciones de un solo tipo (gastos, insumos o activos) para aplicar una cuenta." />
      </div>
    {/if}
    <Button name="Clear|Limpiar" mode="white" icon="icon-[fa--times]" css="ml-auto"
      onClick={() => selectedKeys.clear()} />
  </div>
{/if}

<Layer type="content">
  <VTable data={visibleGroups} {columns}
    cellCss={ENTRY_TABLE_CELL_CSS}
    {filterText}
    getFilterContent={(group) => `${tr(group.Description)} ${tr(operationKindLabels[group.Kind])} ${
      group.Operations.map(operation => operation.DocumentLabel).join(" ")}`}
    selected={openGroup ?? undefined}
    isSelected={(group, opened) => group.Key === (opened as IOperationGroup).Key}
    onRowClick={(group) => openGroupRow(group)}
    maxHeight="calc(100vh - var(--header-height) - 108px)"
    emptyMessage="No operations in the period.|No hay operaciones en el periodo."
  />
</Layer>

<Layer type="side" id={1} sideLayerSize={620}
  title={openOperation ? `${tr(operationKindLabels[openOperation.Kind])} · ${openOperation.Key}`
    : openGroup ? `${tr(operationKindLabels[openGroup.Kind])} · ${openGroup.Operations.length} ${tr("operations|operaciones")}` : ""}
  titleCss="h2 mb-6"
  onSave={editTargets.length > 0 ? saveEntry : undefined}
  onClose={() => { openGroup = null; openOperation = null; editTargets = [] }}
>
  {#if openOperation}
    {#if openGroup && !isSingleOperation(openGroup)}
      <button class="text-sm text-accent-fg mt-4" onclick={() => { openOperation = null; editTargets = [] }}>
        ← <T text="Back to the list|Volver a la lista" />
      </button>
    {/if}
    <OperationEntryLayer operation={openOperation} targets={editTargets} />
  {:else if openGroup}
    <!-- A row of several operations: each one opens its own entry. -->
    <VTable data={openGroup.Operations} disableVirtualizer={true}
      columns={[
        { header: "CUO", css: "ff-mono", getValue: operation => operation.Key },
        {
          header: "Doc. Nº|Nº Doc", css: "ff-mono",
          render: operation => operation.DocumentLabel
            || `<span class="px-6 rounded bg-yellow-bg text-yellow-fg ff-sans">${tr("no comprobante|sin comprobante")}</span>`,
        },
        { header: "Amount (S/)|Monto (S/)", align: "right", css: "ff-mono", getValue: operation => formatN(operation.Amount / 100, 2) },
      ]}
      onRowClick={(operation) => openEntry(operation)}
      maxHeight="calc(100vh - 160px)"
    />
  {/if}
</Layer>
