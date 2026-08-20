<script lang="ts">
  import Card from '$components/cards/Card.svelte';
  import { formatN } from '$libs/helpers';
  import type { ICreditUsageDay } from './company-credit-usage.model';
  import { buildCompanyCreditCalendar, COMPANY_CREDIT_WEEKDAY_LABELS } from './company-credit-calendar';

  let {
    days,
    selectedDay,
    onSelect,
  }: {
    days: ICreditUsageDay[];
    selectedDay?: number;
    onSelect: (day: ICreditUsageDay) => void;
  } = $props();

  const weeks = $derived(buildCompanyCreditCalendar(days));
</script>

<div class="overflow-x-auto rounded-[9px] border border-slate-200 bg-white">
  <!-- Row wrappers use display:contents so every divider belongs to the same 8-column grid. -->
  <div
    class="grid w-full min-w-390 md:min-w-700"
    style="grid-template-columns:54px repeat(7,minmax(48px,1fr))"
    role="table"
    aria-label="30-day company credit calendar"
  >
    <div class="contents" role="row">
      <div class="flex h-38 items-center justify-center border-b border-r border-slate-200 bg-slate-50 text-slate-600" role="columnheader">
        Mes
      </div>
      {#each COMPANY_CREDIT_WEEKDAY_LABELS as weekday, weekdayIndex}
        <div
          class="flex h-38 items-center justify-center border-b border-slate-200 bg-slate-50 text-slate-600 ff-bold"
          class:border-r={weekdayIndex < COMPANY_CREDIT_WEEKDAY_LABELS.length - 1}
          role="columnheader"
        >
          {weekday}
        </div>
      {/each}
    </div>

    {#each weeks as week, weekIndex (week.mondayUnixDay)}
      <div class="contents" role="row">
        <div
          class="flex h-54 items-center justify-center border-r border-slate-200 bg-slate-50 text-slate-600 ff-bold"
          class:border-b={weekIndex < weeks.length - 1}
          role="rowheader"
        >
          {week.month}
        </div>
        {#each week.days as cell, weekdayIndex (cell.unixDay)}
          <div
            class="h-54 border-slate-200"
            class:border-r={weekdayIndex < week.days.length - 1}
            class:border-b={weekIndex < weeks.length - 1}
            role="cell"
          >
            {#if cell.usage}
              <Card
                label={`Open API credit detail for day ${cell.dayOfMonth}|Abrir detalle de créditos API del día ${cell.dayOfMonth}`}
                css={`relative h-full overflow-hidden transition-colors hover:bg-slate-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-violet-400 ${selectedDay === cell.unixDay ? 'bg-violet-50' : ''}`}
                onClick={() => { if (cell.usage) onSelect(cell.usage); }}
              >
                <div class="absolute inset-x-0 top-0 h-16 text-center text-[14px] leading-[16px] text-slate-700 ff-bold">
                  {cell.dayOfMonth}
                </div>
                <div
                  class="absolute inset-x-0 top-16 h-18 px-4 text-right leading-[18px] ff-mono"
                  style:background={cell.usage.CPU > 0
                    ? `linear-gradient(to right, rgb(16 185 129) 0 ${cell.cpuPercent}%, transparent ${cell.cpuPercent}% 100%)`
                    : 'transparent'}
                  title={`CPU: ${formatN(cell.usage.CPU)}`}
                >
                  {formatN(cell.usage.CPU)}
                </div>
                <div
                  class="absolute inset-x-0 top-36 h-18 px-4 text-right leading-[18px] ff-mono"
                  style:background={cell.usage.Inference > 0
                    ? `linear-gradient(to right, rgb(168 85 247) 0 ${cell.inferencePercent}%, transparent ${cell.inferencePercent}% 100%)`
                    : 'transparent'}
                  title={`IA: ${formatN(cell.usage.Inference)}`}
                >
                  {formatN(cell.usage.Inference)}
                </div>
              </Card>
            {:else}
              <div class="h-full bg-slate-50/50"></div>
            {/if}
          </div>
        {/each}
      </div>
    {/each}
  </div>
</div>
