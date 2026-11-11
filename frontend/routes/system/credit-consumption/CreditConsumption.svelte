<script lang="ts">
  import { browser } from '$app/env';
  import { useUI } from '@genix/ui';
  import { ChartCanvas, type ChartCanvasSeries } from '@genix/ui/charts';
  import Button from '#components/buttons/Button.svelte';
  import FilterInput from '#components/form/FilterInput.svelte';
  import Layer from '#components/layers/Layer.svelte';
  import T from '#components/misc/T.svelte';
  import { formatTime } from '#libs/helpers.ts';
  import { onDestroy, onMount } from 'svelte';
  import {
    buildPairRows,
    companyDisplayName,
    CREDIT_CONSUMPTION_WINDOW_HOURS,
    creditConsumptionMatches,
    rejectedPerFrame,
    sumValues,
    type ICreditConsumptionCompany,
  } from './credit-consumption.model';
  import { getCreditConsumption } from './credit-consumption.svelte';

  const REFRESH_INTERVAL_MS = 15_000;
  const LABEL_EVERY_FRAMES = 12;
  const DETAIL_LAYER_ID = 1;
  const CPU_COLOR = '#4f7cf7';
  const REJECTED_COLOR = '#ef5b61';
  const DEVICES_COLOR = '#9b5de5';
  const ui = useUI();

  let frameIDs = $state<number[]>([]);
  let companies = $state<ICreditConsumptionCompany[]>([]);
  let hasLoaded = $state(false);
  let filterText = $state('');
  let isRefreshing = $state(false);
  let lastRefreshAt = $state<Date | null>(null);
  let refreshError = $state('');
  let selectedCompanyID = $state<number | null>(null);
  let refreshTimer: ReturnType<typeof setInterval> | null = null;

  const visibleCompanies = $derived(companies.filter(company => creditConsumptionMatches(company, filterText)));
  // Looked up by ID so the open detail follows each refresh instead of freezing on the first one.
  const selectedCompany = $derived(companies.find(company => company.CompanyID === selectedCompanyID) || null);
  const pairRows = $derived(buildPairRows(selectedCompany));

  const chartSeries = (company: ICreditConsumptionCompany): ChartCanvasSeries[] => [
    { type: 'bar', name: 'CPU credits|Créditos CPU', values: company.CPU, color: CPU_COLOR },
    {
      type: 'line', name: 'Refusals|Negaciones', values: rejectedPerFrame(company), color: REJECTED_COLOR,
      useOwnAxis: true, lineWidth: 2, pointSize: 0,
    },
    {
      type: 'line', name: 'IP-devices|IP-dispositivos', values: company.Devices, color: DEVICES_COLOR,
      useOwnAxis: true, lineWidth: 2, pointSize: 0,
    },
  ];

  const formatFrameLabel = (frameID: string | number) => String(formatTime(Number(frameID), 'h:n') || '');

  const refreshConsumption = async () => {
    if (isRefreshing) return;
    isRefreshing = true;
    try {
      const response = await getCreditConsumption();
      frameIDs = response.FrameIDs || [];
      companies = response.Companies || [];
      lastRefreshAt = new Date();
      refreshError = '';
    } catch (fetchError: any) {
      refreshError = String(fetchError?.message || fetchError || 'error');
    } finally {
      isRefreshing = false;
      hasLoaded = true;
    }
  };

  const openCompany = (company: ICreditConsumptionCompany) => {
    selectedCompanyID = company.CompanyID;
    ui.openSideLayer(DETAIL_LAYER_ID);
  };

  const stopRefreshTimer = () => {
    if (!refreshTimer) return;
    clearInterval(refreshTimer);
    refreshTimer = null;
  };

  const startRefreshTimer = () => {
    if (refreshTimer) return;
    refreshTimer = setInterval(() => {
      if (document.visibilityState === 'visible') void refreshConsumption();
    }, REFRESH_INTERVAL_MS);
  };

  const handleVisibilityChange = () => {
    if (document.visibilityState === 'visible') {
      void refreshConsumption();
      startRefreshTimer();
    } else {
      stopRefreshTimer();
    }
  };

  onMount(() => {
    ui.openSideLayer(0);
    if (!browser) return;
    void refreshConsumption();
    startRefreshTimer();
    document.addEventListener('visibilitychange', handleVisibilityChange);
  });

  onDestroy(() => {
    stopRefreshTimer();
    if (browser) document.removeEventListener('visibilitychange', handleVisibilityChange);
    if (ui.state.sideLayerId === DETAIL_LAYER_ID) ui.openSideLayer(0);
  });
</script>

<Layer type="content">
  <div class="flex flex-col gap-12">
    <div class="flex flex-wrap items-center gap-10">
      <FilterInput
        css="w-280 max-w-full"
        icon="icon-[fa--search]"
        label="Filter companies|Filtrar empresas"
        placeholder="Company or ID|Empresa o ID"
        bind:value={filterText}
      />
      <div class="flex flex-wrap items-center gap-8 text-[14px] text-fg-soft">
        <span class="rounded-full border border-line-strong bg-surface-soft px-9 py-4 text-fg">
          <T text={`Last ${CREDIT_CONSUMPTION_WINDOW_HOURS} hours|Últimas ${CREDIT_CONSUMPTION_WINDOW_HOURS} horas`} />
        </span>
        <span class="flex items-center gap-4"><i class="inline-block h-10 w-10 rounded-[2px]" style:background={CPU_COLOR}></i><T text="CPU credits|Créditos CPU" /></span>
        <span class="flex items-center gap-4"><i class="inline-block h-3 w-12" style:background={REJECTED_COLOR}></i><T text="Refusals|Negaciones" /></span>
        <span class="flex items-center gap-4"><i class="inline-block h-3 w-12" style:background={DEVICES_COLOR}></i><T text="IP-devices|IP-dispositivos" /></span>
        <span><T text="Updated|Actualizado" />: {lastRefreshAt ? formatTime(lastRefreshAt, 'h:n') : '--'}</span>
      </div>
      <Button
        name="Refresh|Actualizar"
        icon="icon-[fa--refresh]"
        css="ml-auto"
        disabled={isRefreshing}
        role="refresh"
        onClick={refreshConsumption}
      />
    </div>

    {#if refreshError}
      <div class="rounded-[8px] border border-red-border bg-red-bg px-10 py-8 text-[14px] text-red-fg">
        {refreshError}
      </div>
    {/if}

    {#if !hasLoaded}
      <div class="rounded-[10px] border border-line bg-surface-soft px-14 py-24 text-center text-[14px] text-fg-soft">
        <T text="Loading credit usage...|Cargando consumo de créditos..." />
      </div>
    {:else if companies.length === 0}
      <div class="rounded-[10px] border border-line bg-surface-soft px-14 py-24 text-center text-[14px] text-fg-soft">
        <T text="No company used credits in this window.|Ninguna empresa consumió créditos en esta ventana." />
      </div>
    {:else if visibleCompanies.length === 0}
      <div class="rounded-[10px] border border-line bg-surface-soft px-14 py-24 text-center text-[14px] text-fg-soft">
        <T text="No company matches this filter.|Ninguna empresa coincide con este filtro." />
      </div>
    {/if}

    <div class="grid grid-cols-1 gap-12 xl:grid-cols-2 2xl:grid-cols-3">
      {#each visibleCompanies as company (company.CompanyID)}
        <button
          type="button"
          class="min-w-0 cursor-pointer rounded-[10px] border bg-surface p-12 text-left shadow-sm transition-colors hover:border-line-strong
            {selectedCompanyID === company.CompanyID ? 'border-purple-fg' : 'border-line'}"
          onclick={() => openCompany(company)}
        >
          <div class="mb-8 flex min-w-0 items-start gap-8">
            <div class="min-w-0 flex-1 break-words font-bold text-fg">{companyDisplayName(company)}</div>
            <span class="shrink-0 ff-mono text-[14px] text-fg-muted">#{company.CompanyID}</span>
          </div>

          <div class="mb-8 flex flex-wrap items-center gap-x-12 gap-y-5 text-[14px] text-fg-soft">
            <span>{company.TotalCPU.toLocaleString()} CPU cr.</span>
            {#if company.TotalInference > 0}<span>{company.TotalInference.toLocaleString()} AI cr.</span>{/if}
            <span class={company.TotalRejected > 0 ? 'text-red-fg' : ''}>
              {company.TotalRejected.toLocaleString()} <T text="refusals|negaciones" />
            </span>
            <span class="text-purple-fg">
              <T text="max|máx" /> {company.MaxDevices.toLocaleString()} <T text="IP-devices / 5 min|IP-dispositivos / 5 min" />
            </span>
          </div>

          <ChartCanvas
            id={`credit-consumption-${company.CompanyID}`}
            data={chartSeries(company)}
            dateLabels={frameIDs}
            dateLabelEvery={LABEL_EVERY_FRAMES}
            dateLabelFormatter={formatFrameLabel}
            useHtmlRendered={true}
            showBottomBaseline={true}
            height={105}
          />
        </button>
      {/each}
    </div>
  </div>
</Layer>

<Layer
  type="side"
  id={DETAIL_LAYER_ID}
  sideLayerSize={760}
  css="p-12"
  title={selectedCompany ? `${companyDisplayName(selectedCompany)} (#${selectedCompany.CompanyID})` : 'IP-devices|IP-dispositivos'}
  titleCss="text-[18px] ff-bold"
  onClose={() => { selectedCompanyID = null; }}
>
  {#if selectedCompany}
    <div class="mt-4 mb-10 flex flex-wrap gap-x-14 gap-y-5 text-[14px] text-fg-soft">
      <span>{selectedCompany.TotalCPU.toLocaleString()} CPU cr.</span>
      <span class="text-red-fg">
        {sumValues(selectedCompany.CreditRejected).toLocaleString()} <T text="credit refusals|negaciones por créditos" />
      </span>
      <span class="text-red-fg">
        {sumValues(selectedCompany.DeviceRejected).toLocaleString()} <T text="device refusals|negaciones por dispositivos" />
      </span>
      <span class="text-purple-fg">
        <T text="max|máx" /> {selectedCompany.MaxDevices.toLocaleString()} <T text="IP-devices / 5 min|IP-dispositivos / 5 min" />
      </span>
    </div>

    {#if pairRows.length === 0}
      <div class="rounded-[10px] border border-line bg-surface-soft px-14 py-24 text-center text-[14px] text-fg-soft">
        <T text="No IP-device was recorded for this company. Only requests that go through HAProxy carry one.|No se registraron IP-dispositivos para esta empresa. Solo las solicitudes que pasan por HAProxy los traen." />
      </div>
    {:else}
      <div class="overflow-x-auto rounded-[8px] border border-line">
        <table class="w-full border-collapse text-[14px]">
          <thead class="bg-surface-soft text-left text-fg-soft">
            <tr>
              <th class="px-8 py-6 font-normal">#</th>
              <th class="px-8 py-6 font-normal">IP</th>
              <th class="px-8 py-6 font-normal"><T text="Device|Dispositivo" /></th>
              <th class="px-8 py-6 text-right font-normal"><T text="Requests|Solicitudes" /></th>
              <th class="px-8 py-6 text-right font-normal">CPU cr.</th>
              <th class="px-8 py-6 text-right font-normal"><T text="Frames|Tramos" /></th>
            </tr>
          </thead>
          <tbody>
            {#each pairRows as pair, index (`${pair.IP}|${pair.Device}`)}
              <tr class="border-t border-line-soft align-top">
                <td class="px-8 py-6 text-fg-muted">{index + 1}</td>
                <td class="px-8 py-6 ff-mono whitespace-nowrap">{pair.IP}</td>
                <td class="px-8 py-6">
                  {#if pair.Label.IsEmpty}
                    <span class="text-fg-muted"><T text="No signature|Sin firma" /></span>
                  {:else}
                    <div class="flex flex-wrap items-center gap-x-6 gap-y-2">
                      <span class="font-bold text-fg">{pair.Label.Browser || '?'}</span>
                      {#if pair.Label.OS}<span>{pair.Label.OS}</span>{/if}
                      {#if pair.Label.Locale}<span>{pair.Label.Locale}</span>{/if}
                      {#if pair.Label.Screen}<span>{pair.Label.Screen}</span>{/if}
                      {#if pair.Label.IsTouch}<span class="text-fg-soft"><T text="touch|táctil" /></span>{/if}
                      {#if pair.Label.IsAutomated}
                        <span class="rounded-full bg-red-bg px-6 py-1 text-red-fg"><T text="automated|automatizado" /></span>
                      {/if}
                    </div>
                    <div class="text-[13px] text-fg-muted">
                      {[pair.Label.TimeZone, pair.Label.Hardware].filter(Boolean).join(' · ')}
                      <span class="ff-mono" title={pair.Device}>#{pair.Device}</span>
                    </div>
                  {/if}
                </td>
                <td class="px-8 py-6 text-right ff-mono">{pair.Requests.toLocaleString()}</td>
                <td class="px-8 py-6 text-right ff-mono">{pair.CPU.toLocaleString()}</td>
                <td class="px-8 py-6 text-right ff-mono">{pair.Frames}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <p class="mt-8 text-[13px] text-fg-muted">
        <T text="Each five-minute frame keeps its twelve busiest IP-devices; these totals add up those frames, so they are exact for the busy pairs and a lower bound for the quiet ones.|Cada tramo de 5 minutos guarda sus 12 IP-dispositivos más activos; estos totales suman esos tramos, así que son exactos para los pares más activos y un mínimo para los menos activos." />
      </p>
    {/if}
  {/if}
</Layer>
