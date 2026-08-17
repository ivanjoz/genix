<script lang="ts">
  import { browser } from '$app/environment';
  import { ChartCanvas, type ChartCanvasSeries } from '@genix/ui/charts';
  import T from '$components/misc/T.svelte';
  import { tr } from '$core/store.svelte';
  import { onDestroy, onMount } from 'svelte';
  import {
    buildServerMetricsSeries,
    WINDOW_HOURS,
    type IServerMetricsSeries,
    type ServerMetricField
  } from './server-metrics.model';
  import { ServerMetricsService } from './server-metrics.svelte';

  interface ChartDefinition {
    id: string;
    title: string;
    // CPU is pinned to 0-100 so an idle service reads as idle instead of being auto-scaled to
    // fill the plot. Memory and network have no known ceiling, so they auto-scale.
    sharedAxisMaxValue?: number;
    series: Array<{
      field: ServerMetricField;
      name: string;
      color: string;
      unit: string;
      decimals: number;
      // Memory shares a chart with CPU but not a scale, so it gets its own axis. Its values are
      // therefore not readable off the y-axis labels, which is why the legend prints them.
      useOwnAxis?: boolean;
    }>;
  }

  const REFRESH_INTERVAL_MS = 15_000;
  const CPU_AXIS_MAX = 100;
  // 360 points into 8 spans, so a label lands every half hour of the four-hour window.
  const LABEL_EVERY_POINTS = 45;

  const CPU_COLOR = '#4874f5';
  const MEMORY_COLOR = '#e67676';

  // One chart per service, CPU and memory together, which is the comparison that answers "is this
  // one busy or is it just big". Disk is deliberately absent: it moves by fractions of a percent
  // over four hours, so a line of it is a flat line, and it shows as a headline instead.
  const chartDefinitions: ChartDefinition[] = [
    {
      id: 'host', title: 'Host: CPU & Memory|Host: CPU y Memoria', sharedAxisMaxValue: CPU_AXIS_MAX,
      series: [
        { field: 'CpuPercent', name: 'CPU', color: CPU_COLOR, unit: '%', decimals: 1 },
        { field: 'MemPercent', name: 'MEM', color: MEMORY_COLOR, unit: '%', decimals: 1 }
      ]
    },
    {
      id: 'backend', title: 'Backend Service', sharedAxisMaxValue: CPU_AXIS_MAX,
      series: [
        { field: 'BackendCpuPercent', name: 'CPU', color: CPU_COLOR, unit: '%', decimals: 1 },
        { field: 'BackendMemMb', name: 'MEM', color: MEMORY_COLOR, unit: 'MB', decimals: 0, useOwnAxis: true }
      ]
    },
    {
      id: 'scylla', title: 'ScyllaDB', sharedAxisMaxValue: CPU_AXIS_MAX,
      series: [
        { field: 'ScyllaCpuPercent', name: 'CPU', color: CPU_COLOR, unit: '%', decimals: 1 },
        { field: 'ScyllaMemMb', name: 'MEM', color: MEMORY_COLOR, unit: 'MB', decimals: 0, useOwnAxis: true }
      ]
    },
    {
      id: 'search', title: 'GenixSearch', sharedAxisMaxValue: CPU_AXIS_MAX,
      series: [
        { field: 'SearchCpuPercent', name: 'CPU', color: CPU_COLOR, unit: '%', decimals: 1 },
        { field: 'SearchMemMb', name: 'MEM', color: MEMORY_COLOR, unit: 'MB', decimals: 0, useOwnAxis: true }
      ]
    },
    {
      id: 'server_utils', title: 'Server Utils', sharedAxisMaxValue: CPU_AXIS_MAX,
      series: [
        { field: 'ServerUtilsCpuPercent', name: 'CPU', color: CPU_COLOR, unit: '%', decimals: 1 },
        { field: 'ServerUtilsMemMb', name: 'MEM', color: MEMORY_COLOR, unit: 'MB', decimals: 0, useOwnAxis: true }
      ]
    },
    {
      id: 'network', title: 'Network|Red',
      series: [
        { field: 'NetRxRate', name: 'RX', color: '#22c55e', unit: 'KB/s', decimals: 1 },
        { field: 'NetTxRate', name: 'TX', color: '#a855f7', unit: 'KB/s', decimals: 1 }
      ]
    }
  ];

  const metricsService = new ServerMetricsService();

  let nowUnixSeconds = $state(Math.floor(Date.now() / 1000));
  let isRefreshing = $state(false);
  let lastRefreshAt = $state<Date | null>(null);
  let refreshError = $state('');
  let refreshTimer: ReturnType<typeof setInterval> | null = null;

  // isReady increments on every merge, so reading it keeps the series in step with the cache.
  const metricsSeries = $derived.by<IServerMetricsSeries>(() => {
    metricsService.isReady;
    return buildServerMetricsSeries(metricsService.hourBuckets, nowUnixSeconds);
  });

  const hasSamples = $derived(metricsSeries.sampledSlots > 0);

  const diskPercent = $derived(metricsSeries.latest.DiskPercent);

  // The right edge of every plot: the window ends now, not at the newest stored sample.
  const windowEndUnixSeconds = $derived(nowUnixSeconds);

  const formatMetricValue = (value: number | null, unit: string, decimals: number) => {
    if (value === null) return '--';
    return `${value.toFixed(decimals)} ${unit}`.trim();
  };

  const formatClock = (unixSeconds: number) => {
    return new Date(unixSeconds * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  // A point covers 40 seconds, so several consecutive points share a minute. The axis prints one
  // label per half hour and never has to tell them apart; the tooltip names a single point and does.
  const formatTooltipClock = (unixSeconds: string | number) => {
    return new Date(Number(unixSeconds) * 1000)
      .toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  };

  // The tooltip receives the series as ChartCanvas knows it — a name and a raw number — so the unit
  // and decimals come back from the definition the series was built from.
  const formatTooltipValue = (chartDefinition: ChartDefinition, seriesName: string, value: number) => {
    const seriesDefinition = chartDefinition.series.find((candidate) => tr(candidate.name) === seriesName);
    if (!seriesDefinition) return String(value);
    return formatMetricValue(value, seriesDefinition.unit, seriesDefinition.decimals);
  };

  // Each label names the start of the span it covers, except the last one: it is drawn hard against
  // the right edge of the plot, so printing its span's start would say the chart ends 45 points
  // before it actually does. The right edge is the window's end, and that is what it must read.
  const formatClockLabel = (unixSeconds: string | number, labelIndex: number) => {
    const isFinalLabel = labelIndex + LABEL_EVERY_POINTS >= metricsSeries.timestamps.length;
    if (isFinalLabel) return formatClock(windowEndUnixSeconds);
    return formatClock(Number(unixSeconds));
  };

  // An own-axis series scaled to exactly its own peak always touches the top of the plot, whatever
  // its value, which makes a flat 545 MB look like a service at its limit. A quarter of headroom
  // puts the peak at 80% of the height, the way a chart normally reads.
  const OWN_AXIS_HEADROOM = 1.25;

  const getOwnAxisMax = (values: Array<number | null>) => {
    const peakValue = values.reduce<number>((maxValue, pointValue) => {
      return pointValue === null ? maxValue : Math.max(maxValue, pointValue);
    }, 0);
    return peakValue > 0 ? peakValue * OWN_AXIS_HEADROOM : undefined;
  };

  const buildChartSeries = (chartDefinition: ChartDefinition): ChartCanvasSeries[] => {
    return chartDefinition.series.map((seriesDefinition) => {
      const seriesValues = metricsSeries.values[seriesDefinition.field];
      return {
        type: 'line',
        name: tr(seriesDefinition.name),
        values: seriesValues,
        color: seriesDefinition.color,
        lineWidth: 2,
        pointSize: 0,
        useOwnAxis: seriesDefinition.useOwnAxis,
        // Memory starts at zero: half a gigabyte is only meaningful against nothing, not against
        // the lowest half-gigabyte the window happened to contain.
        yAxisMin: seriesDefinition.useOwnAxis ? 0 : undefined,
        yAxisMax: seriesDefinition.useOwnAxis ? getOwnAxisMax(seriesValues) : undefined
      };
    });
  };

  const refreshMetrics = async () => {
    if (isRefreshing) return;
    isRefreshing = true;
    try {
      nowUnixSeconds = Math.floor(Date.now() / 1000);
      await metricsService.fetch();
      lastRefreshAt = new Date();
      refreshError = '';
    } catch (fetchError: any) {
      refreshError = String(fetchError?.message || fetchError || 'error');
    } finally {
      isRefreshing = false;
    }
  };

  // A background tab polling a server panel is traffic nobody asked for, so the interval only runs
  // while the document is visible and catches up on the way back.
  const handleVisibilityChange = () => {
    if (document.visibilityState === 'visible') {
      void refreshMetrics();
      return;
    }
    stopRefreshTimer();
  };

  const startRefreshTimer = () => {
    if (refreshTimer) return;
    refreshTimer = setInterval(() => {
      if (document.visibilityState === 'visible') void refreshMetrics();
    }, REFRESH_INTERVAL_MS);
  };

  const stopRefreshTimer = () => {
    if (!refreshTimer) return;
    clearInterval(refreshTimer);
    refreshTimer = null;
  };

  onMount(() => {
    if (!browser) return;
    void refreshMetrics();
    startRefreshTimer();
    document.addEventListener('visibilitychange', handleVisibilityChange);
  });

  onDestroy(() => {
    stopRefreshTimer();
    if (browser) document.removeEventListener('visibilitychange', handleVisibilityChange);
  });
</script>

<div class="flex flex-col gap-12">
  <div class="flex flex-wrap items-center justify-between gap-12">
    <div class="flex flex-wrap items-center gap-10 text-[13px] text-slate-700">
      <span class="font-bold text-slate-900"><T text={`Last ${WINDOW_HOURS} hours|Últimas ${WINDOW_HOURS} horas`} /></span>
      <span class="rounded-full border border-slate-300 bg-slate-50 px-10 py-2 text-[12px] font-semibold text-slate-900">
        DISK: {formatMetricValue(diskPercent, '%', 1)}
      </span>
      <span class="text-slate-600">
        <T text="updated|actualizado" />: {lastRefreshAt ? lastRefreshAt.toLocaleTimeString() : '--'}
      </span>
    </div>

    <button
      class="cursor-pointer rounded-lg border border-slate-300 bg-slate-50 px-10 py-6 text-[12px] text-slate-900 hover:bg-slate-200 disabled:opacity-50"
      disabled={isRefreshing}
      onclick={() => refreshMetrics()}
    >
      <T text="Refresh|Actualizar" />
    </button>
  </div>

  {#if refreshError}
    <div class="rounded-lg border border-red-200 bg-red-100 px-10 py-8 text-[12px] text-red-900">{refreshError}</div>
  {/if}

  {#if !hasSamples}
    <div class="rounded-[10px] border border-slate-200 bg-slate-50 px-14 py-24 text-center text-[13px] text-slate-600">
      <T text="No samples in this window. Check that genix-server-utils is running on this host.|Sin muestras en esta ventana. Verifica que genix-server-utils esté corriendo en este host." />
    </div>
  {/if}

  <div class="grid grid-cols-1 gap-14 xl:grid-cols-2">
    {#each chartDefinitions as chartDefinition (chartDefinition.id)}
      <div class="rounded-[10px] border border-slate-200 bg-white p-12">
        <div class="mb-8 flex flex-wrap items-baseline justify-between gap-10">
          <div class="text-[14px] font-bold text-slate-900"><T text={chartDefinition.title} /></div>
          <div class="flex flex-wrap items-center gap-12">
            {#each chartDefinition.series as seriesDefinition (seriesDefinition.field)}
              <div class="flex items-center gap-6 text-[12px] text-slate-700">
                <span class="inline-block h-8 w-8 rounded-full" style={`background:${seriesDefinition.color}`}></span>
                <span>{tr(seriesDefinition.name)}</span>
                <span class="ff-mono text-slate-900">
                  {formatMetricValue(metricsSeries.latest[seriesDefinition.field], seriesDefinition.unit, seriesDefinition.decimals)}
                </span>
              </div>
            {/each}
          </div>
        </div>

        <ChartCanvas
          id={chartDefinition.id}
          data={buildChartSeries(chartDefinition)}
          dateLabels={metricsSeries.timestamps}
          dateLabelFormatter={formatClockLabel}
          dateLabelEvery={LABEL_EVERY_POINTS}
          sharedAxisMaxValue={chartDefinition.sharedAxisMaxValue}
          height={150}
          showTooltip={true}
          tooltipLabelFormatter={formatTooltipClock}
          tooltipValueFormatter={(value, series) => formatTooltipValue(chartDefinition, series.name, value)}
        />
      </div>
    {/each}
  </div>
</div>
