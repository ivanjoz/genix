<script lang="ts">
  import T from '$components/misc/T.svelte';
  import { tr } from '$core/store.svelte';
  import { formatN, numberToK } from '$libs/helpers';
  import { creditMeterFillPercent, type ICompanyCreditBudgetMeter } from './company-credit-usage.model';

  let { budget }: { budget?: ICompanyCreditBudgetMeter } = $props();

  // The two windows a charge is refused on, each with the CPU and AI credits still available. A
  // company the report has no budget row for is read as unbudgeted, which is what the limiter does
  // with it too: no budget for the month means every charge is rejected.
  const meters = $derived([
    {
      id: 'daily',
      label: 'Daily|Diario',
      cpu: { remaining: budget?.DailyRemainingCPU || 0, total: budget?.DailyCPU || 0 },
      inference: { remaining: budget?.DailyRemainingInference || 0, total: budget?.DailyInference || 0 },
    },
    {
      id: 'credits',
      label: 'Credits|Créditos',
      cpu: { remaining: budget?.RemainingCPU || 0, total: budget?.MonthlyCPUCeiling || 0 },
      inference: { remaining: budget?.RemainingInference || 0, total: budget?.MonthlyInferenceCeiling || 0 },
    },
  ]);
  const hasBudget = $derived(!!budget?.IsCurrentMonth);
  const cellTitle = (metric: string, remaining: number, total: number) => (
    hasBudget
      ? `${metric}: ${formatN(remaining) || '0'} ${tr('of|de')} ${formatN(total) || '0'} ${tr('credits left|créditos restantes')}`
      : tr('No budget is active for the current month.|No hay presupuesto activo para el mes actual.')
  );
</script>

<div class="grid grid-cols-2 gap-8" role="list" aria-label="Remaining credits">
  {#each meters as meter (meter.id)}
    <div class="min-w-0" role="listitem">
      <div class="flex items-center gap-4 text-[12px] text-slate-600">
        <T text={meter.label} />
        {#if !hasBudget}
          <span class="icon-[fa--exclamation-triangle] text-amber-500"></span>
        {/if}
      </div>
      <div
        class="mt-2 grid grid-cols-2 overflow-hidden rounded-[4px] border"
        class:border-slate-300={hasBudget}
        class:border-amber-300={!hasBudget}
      >
        {#each [{ metric: 'CPU', fill: 'bg-emerald-200', ...meter.cpu }, { metric: tr('AI|IA'), fill: 'bg-purple-200', ...meter.inference }] as cell, cellIndex (cell.metric)}
          <div
            class="relative h-18 min-w-0 leading-[18px]"
            class:bg-amber-50={!hasBudget}
            class:border-l={cellIndex > 0}
            class:border-slate-300={cellIndex > 0 && hasBudget}
            class:border-amber-300={cellIndex > 0 && !hasBudget}
            title={cellTitle(cell.metric, cell.remaining, cell.total)}
          >
            {#if hasBudget && cell.remaining > 0}
              <!-- The fill is what is left of the allowance, so a bar that empties is a company
                   running out: the same reading in both windows. -->
              <div class={`absolute inset-y-0 left-0 ${cell.fill}`} style={`width:${creditMeterFillPercent(cell.remaining, cell.total)}%`}></div>
            {/if}
            <span class="absolute inset-y-0 right-4 ff-mono">{hasBudget ? numberToK(cell.remaining) || '0' : '0'}</span>
          </div>
        {/each}
      </div>
    </div>
  {/each}
</div>
