<script lang="ts">
import SearchSelect from '#components/form/SearchSelect.svelte'
import Info from '#components/misc/Info.svelte'
import T from '#components/misc/T.svelte'
import PCGEAccountSelect from '#domain/PCGEAccountSelect.svelte'
import { pcgeAccountLabel } from '#core/pcge.ts'
import { formatN, formatTime } from '#libs/helpers.ts'
import {
  operationKindLabels, OPERATION_STATUS_INCOMPLETE, type IAccountingOperation,
} from './accounts-consolidation'
import { cashClassificationOptions, type IEditableTarget } from './accounts-consolidation.svelte'

// The body of the entry side layer: the operation's journal entry, and one selector per thing the
// user may classify in it. The page owns the Layer and its Save; it reads the edited targets back.

interface Props {
  operation: IAccountingOperation
  targets: IEditableTarget[]
}

const { operation, targets }: Props = $props()

const formatMoney = (cents: number) => cents ? formatN(cents / 100, 2) : ""
</script>

<div class="flex flex-wrap items-center gap-x-16 gap-y-4 mt-8 mb-12 text-sm">
  <div><T text={operationKindLabels[operation.Kind]} /></div>
  <div>{formatTime(operation.Date, "d-m-Y")}</div>
  <div class="ff-mono">{operation.DocumentLabel || "—"}</div>
  {#if !operation.InOfficialBook}
    <span class="px-6 rounded bg-yellow-bg text-yellow-fg">
      <T text="Out of the official Diario|Fuera del Diario oficial" />
    </span>
  {/if}
</div>

{#if operation.Status === OPERATION_STATUS_INCOMPLETE}
  <Info color="yellow" css="mb-12" text={operation.Problem} />
{/if}

<table class="w-full text-sm mb-16" aria-label="Journal entry lines">
  <thead>
    <tr class="border-b border-line text-left">
      <th class="py-4 font-bold"><T text="Account|Cuenta" /></th>
      <th class="py-4 font-bold text-right w-110"><T text="Debit|Debe" /></th>
      <th class="py-4 font-bold text-right w-110"><T text="Credit|Haber" /></th>
    </tr>
  </thead>
  <tbody>
    {#each operation.Lines as line, lineIndex (lineIndex)}
      <tr class="border-b border-line-soft">
        <td class="py-4 pr-8" class:text-red-fg={!line.AccountCode}>
          {#if line.AccountCode}{pcgeAccountLabel(line.AccountCode)}{:else}<T text="— Unclassified|— Sin clasificar" />{/if}
          {#if !line.IsNature}<i class="icon-[fa--lock] text-fg-subtle ml-4"></i>{/if}
        </td>
        <td class="py-4 text-right ff-mono">{formatMoney(line.Debit)}</td>
        <td class="py-4 text-right ff-mono">{formatMoney(line.Credit)}</td>
      </tr>
    {/each}
  </tbody>
</table>

{#if targets.length === 0}
  <Info text="Every account of this operation is derived from its type; nothing to classify.|Todas las cuentas de esta operación se derivan de su tipo; no hay nada que clasificar." />
{:else}
  <div class="h4 ff-bold mb-6"><T text="Classification|Clasificación contable" /></div>
  <div class="grid grid-cols-24 gap-10">
    {#each targets as target (`${target.Kind}-${target.ID}-${target.Line}`)}
      {#if target.PCGEKind}
        <PCGEAccountSelect saveOn={target} save="AccountCode" kind={target.PCGEKind}
          label={target.Label || "Accounting account|Cuenta contable"} css="col-span-24" required />
      {:else}
        <SearchSelect saveOn={target} save="CashOptionKey" css="col-span-24" required
          label="Type and account|Tipo y cuenta" keyId="key" keyName="name"
          options={cashClassificationOptions(target)} />
      {/if}
    {/each}
  </div>
{/if}
