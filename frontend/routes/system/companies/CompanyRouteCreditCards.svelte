<script lang="ts">
  import T from '#components/misc/T.svelte';
  import { formatN } from '#libs/helpers.ts';
  import { splitCompanyCreditRoute, usagePercent, type ICompanyCreditRoute } from './company-credit-usage.model';

  let {
    routes,
    totalCPU,
  }: {
    routes: Required<ICompanyCreditRoute>[];
    totalCPU: number;
  } = $props();
</script>

{#if routes.length}
  <div class="grid grid-cols-2 gap-6" role="list" aria-label="API CPU credit usage">
    {#each routes as route (`${route.RouteID}-${route.Route}`)}
      {@const routeLabel = splitCompanyCreditRoute(route.Route)}
      {@const cpuPercent = usagePercent(route.CPU, totalCPU)}
      <div
        class="min-w-0 overflow-hidden rounded-[4px] border border-line-strong bg-surface"
        role="listitem"
        title={`${routeLabel.method} ${routeLabel.path}: ${formatN(route.CPU)} CPU cr.`}
      >
        <div class="flex h-24 min-w-0 items-center gap-6 px-6">
          <span class="shrink-0 text-fg ff-bold">{routeLabel.method}</span>
          <span class="min-w-0 truncate text-fg">{routeLabel.path}</span>
        </div>
        <div class="relative h-18 bg-surface-soft leading-[18px]">
          {#if route.CPU > 0}
            <!-- The fill represents this route's CPU share of the selected day's total. -->
            <div class="absolute inset-y-0 left-0 bg-green-bg-strong" style={`width:${cpuPercent}%`}></div>
          {/if}
          <span class="absolute inset-y-0 right-4 ff-mono">{formatN(route.CPU)}</span>
        </div>
      </div>
    {/each}
  </div>
{:else}
  <div class="rounded-[8px] border border-line bg-surface-soft p-14 text-center text-fg-soft">
    <T text="No API usage was recorded for this day.|No se registró uso de APIs para este día." />
  </div>
{/if}
