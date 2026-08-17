<script lang="ts">
  import { useUI } from '@genix/ui';
  import Button from '$components/buttons/Button.svelte';
  import FilterInput from '$components/form/FilterInput.svelte';
  import Layer from '$components/layers/Layer.svelte';
  import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
  import T from '$components/misc/T.svelte';
  import { formatN, formatTime } from '$libs/helpers';
  import { onDestroy, onMount, untrack } from 'svelte';
  import CompanyCreditCalendar from './CompanyCreditCalendar.svelte';
  import CompanyCreditCard from './CompanyCreditCard.svelte';
  import CompanyRouteCreditCards from './CompanyRouteCreditCards.svelte';
  import CompanyUserCreditCards from './CompanyUserCreditCards.svelte';
  import type { ICompany } from './empresas.svelte';
  import {
    getCompanyCreditUsageDetail,
    getCompanyCreditUsageReport,
    getCompanyCreditUsageUsers,
    type ICompanyCreditUsageDay,
    type ICompanyCreditUsageDetail,
    type ICompanyCreditUsageReport,
    type ICompanyCreditUsageUsersReport,
  } from './company-credit-usage';
  import {
    normalizeCompanyCreditDetail,
    normalizeCompanyCreditReport,
    normalizeCompanyCreditUsers,
    rankCompanyCreditUsage,
    type CompanyCreditMetric,
    type ICompanyCreditUsageRanked,
  } from './company-credit-usage.model';

  let {
    companies,
    refreshVersion = 0,
    onCreate,
    onEdit,
  }: {
    companies: ICompany[];
    refreshVersion?: number;
    onCreate: () => void;
    onEdit: (company: ICompany) => void;
  } = $props();

  const ui = useUI();
  const COMPANY_USAGE_LAYER_ID = 1;
  const metricOptions = [
    { id: 'CPU', name: 'CPU Credits|Créditos CPU' },
    { id: 'Inference', name: 'AI Credits|Créditos IA' },
  ];

  let report = $state<ICompanyCreditUsageReport | null>(null);
  let selectedMetric = $state<CompanyCreditMetric>('CPU');
  let filterText = $state('');
  let isLoading = $state(false);
  let reportError = $state('');
  let selectedCompany = $state<ICompanyCreditUsageRanked | null>(null);
  let selectedDay = $state<ICompanyCreditUsageDay | null>(null);
  let selectedDetail = $state<ICompanyCreditUsageDetail | null>(null);
  let isDetailLoading = $state(false);
  let detailError = $state('');
  let usersReport = $state<ICompanyCreditUsageUsersReport | null>(null);
  let isUsersLoading = $state(false);
  let usersError = $state('');
  let layerView = $state(1);
  let observedRefreshVersion = $state<number | null>(null);
  const detailMemo = new Map<string, ICompanyCreditUsageDetail>();
  const usersMemo = new Map<number, ICompanyCreditUsageUsersReport>();

  const normalizedCompanies = $derived(normalizeCompanyCreditReport(report?.Companies));
  const companiesByID = $derived(new Map(companies.map((company) => [company.id, company])));
  const masterSearchTextByCompanyID = $derived(new Map(companies.map((company) => [
    company.id,
    [company.Name, company.LegalName, company.RUC, company.Email].filter(Boolean).join(' '),
  ])));
  const rankedCompanies = $derived(
    rankCompanyCreditUsage(normalizedCompanies, selectedMetric, filterText, masterSearchTextByCompanyID),
  );
  const requestErrorText = (requestError: any): string => String(
    requestError?.errorMessage || requestError?.error || requestError?.message ||
    (typeof requestError === 'string' ? requestError : 'error')
  );

  const refreshReport = async () => {
    if (isLoading) return;
    isLoading = true;
    reportError = '';
    try {
      report = await getCompanyCreditUsageReport();
      detailMemo.clear();
      usersMemo.clear();
      selectedCompany = null;
      selectedDay = null;
      selectedDetail = null;
      usersReport = null;
      ui.openSideLayer(0);
      console.debug('[company-cards] report loaded', { companies: report?.Companies?.length || 0 });
    } catch (fetchError: any) {
      reportError = requestErrorText(fetchError);
      console.error('[company-cards] report failed', { error: reportError });
    } finally {
      isLoading = false;
    }
  };

  const openCompanyUsage = (company: ICompanyCreditUsageRanked) => {
    selectedCompany = company;
    selectedDay = null;
    selectedDetail = null;
    detailError = '';
    usersReport = null;
    usersError = '';
    layerView = 1;
    ui.openSideLayer(COMPANY_USAGE_LAYER_ID);
    console.debug('[company-cards] usage layer opened', { companyID: company.CompanyID });
  };

  const loadCompanyUsers = async (companyID: number) => {
    if (isUsersLoading || usersReport?.CompanyID === companyID) return;
    const memoReport = usersMemo.get(companyID);
    if (memoReport) {
      usersReport = memoReport;
      console.debug('[company-cards] users cache hit', { companyID, users: memoReport.Users.length });
      return;
    }
    isUsersLoading = true;
    usersError = '';
    console.debug('[company-cards] users requested', { companyID });
    try {
      const nextReport = normalizeCompanyCreditUsers(await getCompanyCreditUsageUsers(companyID));
      usersMemo.set(companyID, nextReport);
      if (selectedCompany?.CompanyID === companyID) usersReport = nextReport;
      console.debug('[company-cards] users loaded', { companyID, users: nextReport.Users.length });
    } catch (fetchError: any) {
      if (selectedCompany?.CompanyID === companyID) usersError = requestErrorText(fetchError);
      console.error('[company-cards] users failed', { companyID, error: requestErrorText(fetchError) });
    } finally {
      isUsersLoading = false;
    }
  };

  const openDayDetail = async (day: ICompanyCreditUsageDay) => {
    if (!selectedCompany) return;
    selectedDay = day;
    selectedDetail = null;
    detailError = '';
    layerView = 2;
    const memoKey = `${selectedCompany.CompanyID}:${day.Day}`;
    const memoDetail = detailMemo.get(memoKey);
    if (memoDetail) {
      selectedDetail = memoDetail;
      console.debug('[company-cards] detail cache hit', { companyID: selectedCompany.CompanyID, day: day.Day });
      return;
    }

    isDetailLoading = true;
    console.debug('[company-cards] detail requested', { companyID: selectedCompany.CompanyID, day: day.Day });
    try {
      const detail = normalizeCompanyCreditDetail(await getCompanyCreditUsageDetail(selectedCompany.CompanyID, day.Day));
      detailMemo.set(memoKey, detail);
      selectedDetail = detail;
      console.debug('[company-cards] detail loaded', { companyID: selectedCompany.CompanyID, day: day.Day, routes: detail.Routes.length });
    } catch (fetchError: any) {
      detailError = requestErrorText(fetchError);
      console.error('[company-cards] detail failed', { companyID: selectedCompany.CompanyID, day: day.Day, error: detailError });
    } finally {
      isDetailLoading = false;
    }
  };

  onMount(() => { void refreshReport(); });
  $effect(() => {
    if (observedRefreshVersion === null) {
      observedRefreshVersion = refreshVersion;
      return;
    }
    if (refreshVersion === observedRefreshVersion) return;
    observedRefreshVersion = refreshVersion;
    void refreshReport();
  });
  $effect(() => {
    const companyID = selectedCompany?.CompanyID;
    if (layerView !== 3 || !companyID) return;
    // Loading mutates tab state, so keep it outside this effect's reactive dependency graph.
    untrack(() => { void loadCompanyUsers(companyID); });
  });
  onDestroy(() => {
    if (ui.state.sideLayerId === COMPANY_USAGE_LAYER_ID) ui.openSideLayer(0);
  });
</script>

<div class="flex flex-col gap-12 text-[14px]">
  <div class="grid min-w-0 grid-cols-1 gap-8 md:flex md:flex-wrap md:items-center md:gap-10" aria-label="Company cards toolbar">
    <FilterInput
      css="w-full md:w-280"
      icon="icon-[fa--search]"
      label="Filter companies|Filtrar empresas"
      placeholder="Company, administrator or RUC|Empresa, administrador o RUC"
      bind:value={filterText}
    />
    <OptionsStrip
      options={metricOptions}
      selected={selectedMetric}
      keyId="id"
      keyName="name"
      css="w-full md:w-auto"
      useMobileGrid={true}
      onSelect={(metric) => { selectedMetric = metric.id as CompanyCreditMetric; }}
    />
    <div class="flex min-w-0 flex-wrap items-center gap-8 md:contents">
      <span class="shrink-0 rounded-full border border-slate-300 bg-slate-50 px-9 py-4 text-slate-800">
        <T text="Last 30 days|Últimos 30 días" />
      </span>
      {#if report}
        <span class="min-w-0 text-slate-600"><T text="Updated|Actualizado" />: {formatTime(report.GeneratedAt, 'd-M-Y h:n')}</span>
      {/if}
      <div class="ml-auto flex shrink-0 items-center gap-8 md:contents">
        <Button
          name="Refresh|Actualizar"
          icon="icon-[fa--refresh]"
          css="md:ml-auto"
          hideNameOnMobile={true}
          disabled={isLoading}
          onClick={refreshReport}
        />
        <Button
          color="green"
          icon="icon-[fa--plus]"
          label="Create a new company|Crear una empresa nueva"
          onClick={onCreate}
        />
      </div>
    </div>
  </div>

  {#if reportError}
    <div class="rounded-[8px] border border-red-200 bg-red-50 px-10 py-8 text-red-800">{reportError}</div>
  {/if}

  {#if isLoading && !report}
    <div class="rounded-[10px] border border-slate-200 bg-slate-50 px-14 py-24 text-center text-slate-600">
      <T text="Loading companies and credit usage...|Cargando empresas y uso de créditos..." />
    </div>
  {:else if report}
    <Layer type="content">
      {#if rankedCompanies.length === 0}
        <div class="rounded-[10px] border border-slate-200 bg-white px-14 py-24 text-center text-slate-600">
          <T text="No companies match the filter.|No hay empresas que coincidan con el filtro." />
        </div>
      {:else}
        <div class="grid grid-cols-1 gap-12 lg:grid-cols-2 2xl:grid-cols-3">
          {#each rankedCompanies as company (company.CompanyID)}
            {@const masterCompany = companiesByID.get(company.CompanyID)}
            <CompanyCreditCard
              {company}
              onOpen={() => { openCompanyUsage(company); }}
              onEdit={masterCompany ? () => { onEdit(masterCompany); } : undefined}
            />
          {/each}
        </div>
      {/if}
    </Layer>
  {/if}

  <Layer
    type="side"
    id={COMPANY_USAGE_LAYER_ID}
    sideLayerSize={760} css="p-8"
    title={selectedCompany ? `${selectedCompany.Company} (#${selectedCompany.CompanyID})` : 'Company credits|Créditos de empresa'}
    titleCss="text-[18px] ff-bold"
    options={[[1, 'By day|Por día'], [2, 'APIs for day|APIs del día'], [3, 'Users|Usuarios']]}
    bind:selected={layerView}
    onClose={() => {
      selectedCompany = null;
      selectedDay = null;
      selectedDetail = null;
      detailError = '';
      usersReport = null;
      usersError = '';
    }}
  >
    {#if selectedCompany && layerView === 1}
      <div class="flex flex-col gap-14 text-[14px]">
        <div class="grid grid-cols-2 gap-10">
          <div class="rounded-[8px] bg-emerald-50 p-10">
            <div class="text-slate-600">CPU 30d</div>
            <div class="mt-4 text-[18px] ff-bold ff-mono">{formatN(selectedCompany.CPU)}</div>
          </div>
          <div class="rounded-[8px] bg-purple-50 p-10">
            <div class="text-slate-600"><T text="AI 30d|IA 30d" /></div>
            <div class="mt-4 text-[18px] ff-bold ff-mono">{formatN(selectedCompany.Inference)}</div>
          </div>
        </div>

        <CompanyCreditCalendar
          days={selectedCompany.Days}
          selectedDay={selectedDay?.Day}
          onSelect={(day) => { void openDayDetail(day); }}
        />
      </div>
    {:else if selectedCompany && layerView === 2}
      <div class="flex flex-col gap-12 text-[14px]">
        {#if !selectedDay}
          <div class="rounded-[8px] border border-slate-200 bg-slate-50 p-14 text-center text-slate-600">
            <T text="Select a day in the By day tab.|Seleccione un día en la pestaña Por día." />
          </div>
        {:else}
          <div class="flex flex-wrap items-center gap-12 rounded-[8px] bg-slate-50 p-10">
            <span class="ff-bold">{formatTime(selectedDay.Day, 'd-M-Y')}</span>
            <span>{formatN(selectedDay.CPU)} CPU cr.</span>
            <span>{formatN(selectedDay.Inference)} <T text="AI cr.|cr. IA" /></span>
          </div>
          {#if detailError}
            <div class="rounded-[8px] border border-red-200 bg-red-50 p-10 text-red-800">{detailError}</div>
          {:else if isDetailLoading}
            <div class="rounded-[8px] border border-slate-200 bg-slate-50 p-14 text-center text-slate-600">
              <T text="Loading API detail...|Cargando detalle por API..." />
            </div>
          {:else if selectedDetail}
            <CompanyRouteCreditCards routes={selectedDetail.Routes} totalCPU={selectedDetail.CPU} />
          {/if}
        {/if}
      </div>
    {:else if selectedCompany && layerView === 3}
      <div class="text-[14px]">
        {#if usersError}
          <div class="rounded-[8px] border border-red-200 bg-red-50 p-10 text-red-800">{usersError}</div>
        {:else if isUsersLoading && !usersReport}
          <div class="rounded-[8px] border border-slate-200 bg-slate-50 p-14 text-center text-slate-600">
            <T text="Loading user usage...|Cargando uso por usuario..." />
          </div>
        {:else if usersReport}
          <CompanyUserCreditCards users={usersReport.Users} />
        {/if}
      </div>
    {/if}
  </Layer>
</div>
