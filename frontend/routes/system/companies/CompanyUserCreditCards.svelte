<script lang="ts">
  import { ChartCanvas, type ChartCanvasSeries } from '@genix/ui/charts';
  import T from '$components/misc/T.svelte';
  import { formatN } from '$libs/helpers';
  import type { ICompanyCreditUsageUser } from './company-credit-usage';

  let { users }: { users: ICompanyCreditUsageUser[] } = $props();

  const chartSeries = (user: ICompanyCreditUsageUser): ChartCanvasSeries[] => [
    { type: 'bar', name: 'CPU credits', color: '#10b981', values: user.Days.map((day) => day.CPU) },
    { type: 'bar', name: 'Inference credits', color: '#a855f7', values: user.Days.map((day) => day.Inference) },
  ];
</script>

{#if users.length}
  <div class="grid grid-cols-2 gap-6" role="list" aria-label="User credit usage">
    {#each users as user (user.UserID)}
      <div class="min-w-0 rounded-[6px] border border-slate-200 bg-white p-8" role="listitem">
        <div class="flex min-w-0 items-center gap-6">
          <span class="min-w-0 truncate ff-bold text-slate-900">{user.Name}</span>
          <span class="ml-auto shrink-0 ff-mono text-slate-500">#{user.UserID}</span>
        </div>
        {#if user.User && user.User !== user.Name}
          <div class="truncate text-slate-500">@{user.User}</div>
        {/if}
        <div class="mt-4 flex min-w-0 items-center gap-8 ff-mono">
          <span class="flex min-w-0 items-center gap-4"><i class="h-8 w-8 shrink-0 bg-emerald-500"></i>{formatN(user.CPU)} CPU</span>
          <span class="flex min-w-0 items-center gap-4"><i class="h-8 w-8 shrink-0 bg-purple-500"></i>{formatN(user.Inference)} <T text="AI|IA" /></span>
        </div>
        <div class="mt-4 overflow-hidden">
          <!-- Thirty paired columns preserve the daily pattern without making the compact card taller. -->
          <ChartCanvas
            id={`company-user-credit-${user.UserID}`}
            data={chartSeries(user)}
            barMode="grouped"
            useHtmlRendered={true}
            showBottomBaseline={true}
            height={40}
            showTooltip={true}
          />
        </div>
      </div>
    {/each}
  </div>
{:else}
  <div class="rounded-[8px] border border-slate-200 bg-slate-50 p-14 text-center text-slate-600">
    <T text="No users were found for this company.|No se encontraron usuarios para esta empresa." />
  </div>
{/if}
