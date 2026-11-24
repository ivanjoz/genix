<script lang="ts">
  import { goto } from '$app/navigation'
  import Button from '#components/buttons/Button.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import { SEVERITY_ORDER, type CashFlowSeverity, type ICashFlowWarning } from './cash-flow-warnings.ts'

  // The warnings layer's body: by severity, each gap with what the forecast assumed, the money at
  // stake, its records and a link to the page where it is fixed (opened as is).
  let { warnings }: { warnings: ICashFlowWarning[] } = $props()

  const severityLabels: Record<CashFlowSeverity, string> = { high: 'High|Alta', medium: 'Medium|Media', info: 'Info|Informativa' }
  const severityCss: Record<CashFlowSeverity, string> = { high: 'text-red-fg', medium: 'text-orange-fg', info: 'text-fg-muted' }
</script>

{#if warnings.length === 0}
  <div class="text-fg-muted mt-12">{tr('No warnings: the forecast assumed nothing.|Sin alertas: el pronóstico no asumió nada.')}</div>
{/if}
{#each SEVERITY_ORDER as severity}
  {@const group = warnings.filter((warning) => warning.severity === severity)}
  {#if group.length > 0}
    <div class="h4 ff-bold mt-14 mb-6 {severityCss[severity]}">{tr(severityLabels[severity])} ({group.length})</div>
    {#each group as warning (warning.Code)}
      <div class="rounded-[10px] border border-line p-10 mb-8" aria-label={`Warning ${warning.Code}`}>
        <div class="flex items-start justify-between gap-10">
          <div>{tr(warning.message)}</div>
          {#if warning.route}
            <Button mode="subtle" color="blue" height="tiny" name="Fix|Corregir" label={`Opens ${warning.route}.`}
              onClick={() => goto(warning.route)} />
          {/if}
        </div>
        <div class="text-fg-muted mt-4">
          {warning.RecordCount} {tr('records|registros')}
          {#if warning.AmountAtStakePEN}· S/ {formatN(warning.AmountAtStakePEN / 100, 2)} {tr('at stake|en juego')}{/if}
        </div>
        {#if warning.Records?.length}
          <div class="mt-4 text-fg-muted">
            {warning.Records.map((record) => record.Label || record.ID).join(' · ')}{warning.RecordCount > warning.Records.length ? ' …' : ''}
          </div>
        {/if}
      </div>
    {/each}
  {/if}
{/each}
