<script lang="ts">
  import { browser } from '$app/environment';
  import { ChartCanvas, type ChartCanvasSeries } from '@genix/ui/charts';
  import Page from '$domain/Page.svelte';
  import Button from '$components/buttons/Button.svelte';
  import FilterInput from '$components/form/FilterInput.svelte';
  import T from '$components/misc/T.svelte';
  import { formatTime } from '$libs/helpers';
  import { onDestroy, onMount } from 'svelte';
  import {
    buildObservabilityCards,
    buildObservabilityErrorPreviews,
    OBSERVABILITY_WINDOW_HOURS,
    observabilityCardMatches,
    type IObservabilityCard,
  } from './observability.model';
  import { ObservabilityService } from './observability.svelte';

  const REFRESH_INTERVAL_MS = 15_000;
  const LABEL_EVERY_FRAMES = 12;
  const SUCCESS_COLOR = '#4db76a';
  const ERROR_COLOR = '#ef5b61';
  const observabilityService = new ObservabilityService();

  let filterText = $state('');
  let isRefreshing = $state(false);
  let lastRefreshAt = $state<Date | null>(null);
  let refreshError = $state('');
  let refreshTimer: ReturnType<typeof setInterval> | null = null;

  const cards = $derived.by(() => {
    observabilityService.isReady;
    return buildObservabilityCards(observabilityService.frames, observabilityService.routes);
  });
  const visibleCards = $derived(cards.filter(card =>
    observabilityCardMatches(card, filterText, observabilityService.errorRecords)
  ));

  const chartSeries = (card: IObservabilityCard): ChartCanvasSeries[] => [
    { type: 'bar', name: 'Estimated successes', values: card.EstimatedSuccessValues, color: SUCCESS_COLOR },
    { type: 'bar', name: 'Failed requests', values: card.FailedRequestValues, color: ERROR_COLOR },
  ];

  const formatFrameLabel = (frameID: string | number) => {
    return String(formatTime(Number(frameID), 'h:n') || '');
  };

  const refreshObservability = async () => {
    if (isRefreshing) return;
    isRefreshing = true;
    try {
      await observabilityService.fetch();
      lastRefreshAt = new Date();
      refreshError = '';
    } catch (fetchError: any) {
      refreshError = String(fetchError?.message || fetchError || 'error');
    } finally {
      isRefreshing = false;
    }
  };

  const stopRefreshTimer = () => {
    if (!refreshTimer) return;
    clearInterval(refreshTimer);
    refreshTimer = null;
  };

  const startRefreshTimer = () => {
    if (refreshTimer) return;
    refreshTimer = setInterval(() => {
      if (document.visibilityState === 'visible') void refreshObservability();
    }, REFRESH_INTERVAL_MS);
  };

  const handleVisibilityChange = () => {
    if (document.visibilityState === 'visible') {
      void refreshObservability();
      startRefreshTimer();
    } else {
      stopRefreshTimer();
    }
  };

  onMount(() => {
    if (!browser) return;
    void refreshObservability();
    startRefreshTimer();
    document.addEventListener('visibilitychange', handleVisibilityChange);
  });

  onDestroy(() => {
    stopRefreshTimer();
    if (browser) document.removeEventListener('visibilitychange', handleVisibilityChange);
  });
</script>

<Page title="Observability|Observabilidad">
  <div class="flex flex-col gap-12">
    <div class="flex flex-wrap items-center gap-10">
      <FilterInput
        css="w-280 max-w-full"
        icon="icon-[fa--search]"
        label="Filter routes and errors|Filtrar rutas y errores"
        placeholder="Route or error|Ruta o error"
        bind:value={filterText}
      />
      <div class="flex flex-wrap items-center gap-8 text-[12px] text-slate-600">
        <span class="rounded-full border border-slate-300 bg-slate-50 px-9 py-4 text-slate-800">
          <T text={`Last ${OBSERVABILITY_WINDOW_HOURS} hours|Últimas ${OBSERVABILITY_WINDOW_HOURS} horas`} />
        </span>
        <span><T text="Requests are estimated from credits|Solicitudes estimadas desde créditos" /></span>
        <span><T text="Updated|Actualizado" />: {lastRefreshAt ? formatTime(lastRefreshAt, 'h:n') : '--'}</span>
      </div>
      <Button
        name="Refresh|Actualizar"
        icon="icon-[fa--refresh]"
        css="ml-auto"
        disabled={isRefreshing}
        role="refresh"
        onClick={refreshObservability}
      />
    </div>

    {#if refreshError}
      <div class="rounded-[8px] border border-red-200 bg-red-50 px-10 py-8 text-[12px] text-red-800">
        {refreshError}
      </div>
    {/if}

    {#if isRefreshing && observabilityService.isReady === 0}
      <div class="rounded-[10px] border border-slate-200 bg-slate-50 px-14 py-24 text-center text-[13px] text-slate-600">
        <T text="Loading observability data...|Cargando datos de observabilidad..." />
      </div>
    {:else if cards.length === 0}
      <div class="rounded-[10px] border border-slate-200 bg-slate-50 px-14 py-24 text-center text-[13px] text-slate-600">
        <T text="No API usage or errors were recorded in this window.|No se registró uso de API ni errores en esta ventana." />
      </div>
    {:else if visibleCards.length === 0}
      <div class="rounded-[10px] border border-slate-200 bg-slate-50 px-14 py-24 text-center text-[13px] text-slate-600">
        <T text="No routes match this filter.|Ninguna ruta coincide con este filtro." />
      </div>
    {/if}

    <div class="grid grid-cols-1 gap-12 xl:grid-cols-2 2xl:grid-cols-3">
      {#each visibleCards as card (card.RouteID)}
        {@const errorPreviews = buildObservabilityErrorPreviews(card, observabilityService.errorRecords)}
        <article class="min-w-0 rounded-[10px] border border-slate-200 bg-white p-12 shadow-sm">
          <div class="mb-8 flex min-w-0 items-start gap-8">
            <span class="shrink-0 rounded-[5px] bg-slate-900 px-7 py-3 ff-mono text-[11px] text-white">
              {card.Method || 'API'}
            </span>
            <div class="min-w-0 flex-1 break-words text-[14px] font-bold text-slate-900">{card.Path}</div>
            <span class="shrink-0 ff-mono text-[11px] text-slate-500">#{card.RouteID}</span>
          </div>

          <div class="mb-8 flex flex-wrap items-center gap-x-12 gap-y-5 text-[12px] text-slate-700">
            {#if card.IsMetered}
              <span>~{card.EstimatedRequests.toLocaleString()} <T text="requests|solicitudes" /></span>
            {:else}
              <span class="rounded-full bg-amber-100 px-7 py-3 text-amber-800"><T text="Unmetered|Sin medición" /></span>
            {/if}
            <span>{card.CPU.toLocaleString()} CPU cr.</span>
            {#if card.Inference > 0}<span>{card.Inference.toLocaleString()} AI cr.</span>{/if}
            <span class="text-red-700">{card.FailedRequests.toLocaleString()} <T text="failed|fallidas" /></span>
            {#if card.ErrorOccurrences !== card.FailedRequests}
              <span>{card.ErrorOccurrences.toLocaleString()} <T text="errors|errores" /></span>
            {/if}
          </div>

          <div class="mb-7 flex items-center gap-12 text-[11px] text-slate-600">
            <span class="flex items-center gap-5"><i class="inline-block h-7 w-7 bg-[#4db76a]"></i><T text="Estimated success|Éxito estimado" /></span>
            <span class="flex items-center gap-5"><i class="inline-block h-7 w-7 bg-[#ef5b61]"></i><T text="Actual failures|Fallas reales" /></span>
          </div>

          <ChartCanvas
            id={`observability-${card.RouteID}`}
            data={chartSeries(card)}
            dateLabels={card.FrameIDs}
            dateLabelEvery={LABEL_EVERY_FRAMES}
            dateLabelFormatter={formatFrameLabel}
            useHtmlRendered={true}
            showBottomBaseline={true}
            height={105}
          />

          {#if errorPreviews.length > 0}
            <div class="mt-8 flex flex-col gap-5 border-t border-slate-100 pt-7">
              {#each errorPreviews as preview (preview.ID)}
                {#if preview.Entries.length > 0}
                  {#each preview.Entries as entry (entry.CodeLine)}
                    <div class="text-[11px] leading-[1.35] text-red-700">
                      <span class="font-bold">{preview.Count}×</span>
                      {entry.Text}
                      <span class="text-purple-600">[{entry.CodeLine}]</span>
                    </div>
                  {/each}
                {:else}
                  <div class="text-[11px] text-red-700">{preview.Count}× <T text="Error preview unavailable|Vista previa no disponible" /> [#{preview.ID}]</div>
                {/if}
              {/each}
            </div>
          {/if}
        </article>
      {/each}
    </div>
  </div>
</Page>
