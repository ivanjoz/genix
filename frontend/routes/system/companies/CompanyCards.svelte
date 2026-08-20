<script lang="ts">
  import { useUI } from '@genix/ui';
  import { getStaticRecordsByID } from '@genix/ui/cache';
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
  import { CompanyCreditReportService, CompanyCreditUsageService } from './company-credit-usage.svelte';
  import {
    COMPANY_ADMINISTRATOR_USER_ID,
    packCompanyUserLabelID,
    buildCompanyDays,
    buildCompanyCreditSummaries,
    buildCompanyDayDetail,
    buildCompanyUserSummaries,
    companyUserDisplayName,
    rankCompanyCreditUsage,
    type CompanyCreditMetric,
    type ICompanyCreditSummaryRanked,
    type ICompanyUserLabel,
    type ICreditUsageDay,
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
  const COMPANY_USER_LABELS_ROUTE = 'company-users-by-ids';
  const metricOptions = [
    { id: 'CPU', name: 'CPU Credits|Créditos CPU' },
    { id: 'Inference', name: 'AI Credits|Créditos IA' },
  ];

  const report = new CompanyCreditReportService(true);
  let selectedMetric = $state<CompanyCreditMetric>('CPU');
  let filterText = $state('');
  let isLoading = $state(false);
  let reportError = $state('');
  let selectedCompany = $state<ICompanyCreditSummaryRanked | null>(null);
  let selectedDay = $state<ICreditUsageDay | null>(null);
  let layerView = $state(1);
  let observedRefreshVersion = $state<number | null>(null);
  // One usage service per opened company: each keeps its own delta collection and watermark.
  let usageServices = $state(new Map<number, CompanyCreditUsageService>());
  let userLabels = $state(new Map<number, ICompanyUserLabel>());

  const summaries = $derived(buildCompanyCreditSummaries(report.days, companies));
  const budgetMeterByCompanyID = $derived(new Map(report.budgetMeters.map((meter) => [meter.ID, meter])));
  const companiesByID = $derived(new Map(companies.map((company) => [company.id, company])));
  const masterSearchTextByCompanyID = $derived(new Map(companies.map((company) => [
    company.id,
    [company.Name, company.LegalName, company.RUC, company.Email].filter(Boolean).join(' '),
  ])));
  const administratorTextByCompanyID = $derived(new Map(summaries.map((company) => {
    const label = userLabels.get(packCompanyUserLabelID(company.CompanyID, COMPANY_ADMINISTRATOR_USER_ID));
    return [company.CompanyID, [label?.FirstName, label?.LastName, label?.User].filter(Boolean).join(' ')];
  })));
  $effect(() => {
    console.debug(`[company-cards] summaries=${summaries.length} `
      + summaries.map((c) => `#${c.CompanyID}:cpu=${c.CPU},today=${c.TodayCPU},days=${c.Days.length}`).join(' '));
  });
  const rankedCompanies = $derived(rankCompanyCreditUsage(
    summaries, selectedMetric, filterText, masterSearchTextByCompanyID, administratorTextByCompanyID,
  ));

  const selectedUsage = $derived(selectedCompany ? usageServices.get(selectedCompany.CompanyID) : undefined);
  const selectedCompanyDays = $derived(buildCompanyDays(selectedUsage?.companyDays));
  const selectedDetail = $derived(
    selectedDay ? buildCompanyDayDetail(selectedUsage?.companyDays, selectedDay.Day) : null,
  );
  const selectedUsers = $derived(buildCompanyUserSummaries(selectedUsage?.userDays));

  const requestErrorText = (requestError: any): string => String(
    requestError?.errorMessage || requestError?.error || requestError?.message ||
    (typeof requestError === 'string' ? requestError : 'error')
  );

  // Labels are packed (company, user) ids so one static endpoint answers across tenants; the cache
  // only reaches the server for ids it holds in neither memory nor IndexedDB.
  const resolveUserLabels = async (packedIDs: number[]) => {
    const missingIDs = packedIDs.filter((packedID) => !userLabels.has(packedID));
    if (missingIDs.length === 0) return;
    const resolved = await getStaticRecordsByID<ICompanyUserLabel>(COMPANY_USER_LABELS_ROUTE, missingIDs);
    userLabels = new Map([...userLabels, ...resolved]);
    console.debug(`[company-cards] labels requested=${missingIDs.length} resolved=${resolved.size}`);
  };

  const refreshReport = async () => {
    if (isLoading) return;
    isLoading = true;
    reportError = '';
    try {
      // fetchOnline, not fetch: the button means "go to the server now", while fetch would answer
      // from the cache snapshot whenever its TTL is still valid.
      await report.fetchOnline();
      console.debug(`[company-cards] report loaded days=${report.days.length}`);
    } catch (fetchError: any) {
      reportError = requestErrorText(fetchError);
      console.error('[company-cards] report failed', { error: reportError });
    } finally {
      isLoading = false;
    }
  };

  const openCompanyUsage = (company: ICompanyCreditSummaryRanked) => {
    selectedCompany = company;
    selectedDay = null;
    layerView = 1;
    if (!usageServices.has(company.CompanyID)) {
      usageServices = new Map(usageServices).set(
        company.CompanyID, new CompanyCreditUsageService(company.CompanyID, true),
      );
    }
    ui.openSideLayer(COMPANY_USAGE_LAYER_ID);
    console.debug(`[company-cards] usage layer opened company=${company.CompanyID}`);
  };

  onMount(() => { ui.openSideLayer(0); });
  $effect(() => {
    if (observedRefreshVersion === null) {
      observedRefreshVersion = refreshVersion;
      return;
    }
    if (refreshVersion === observedRefreshVersion) return;
    observedRefreshVersion = refreshVersion;
    void refreshReport();
  });
  // Administrator labels are resolved for every company, not just the visible ones: the filter
  // searches administrator names, so a name that is not resolved yet could never be matched.
  // Reading `summaries` rather than `rankedCompanies` also keeps this out of a cycle, since the
  // ranking depends on the very label map this effect fills.
  $effect(() => {
    const packedIDs = summaries.map(
      (company) => packCompanyUserLabelID(company.CompanyID, COMPANY_ADMINISTRATOR_USER_ID),
    );
    untrack(() => { void resolveUserLabels(packedIDs); });
  });
  $effect(() => {
    const companyID = selectedCompany?.CompanyID;
    const users = selectedUsers;
    if (layerView !== 3 || !companyID) return;
    untrack(() => {
      void resolveUserLabels(users.map((user) => packCompanyUserLabelID(companyID, user.UserID)));
    });
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

  {#if rankedCompanies.length === 0}
    <div class="rounded-[10px] border border-slate-200 bg-slate-50 px-14 py-24 text-center text-slate-600">
      {#if summaries.length === 0}
        <T text="Loading companies and credit usage...|Cargando empresas y uso de créditos..." />
      {:else}
        <T text="No companies match the filter.|No hay empresas que coincidan con el filtro." />
      {/if}
    </div>
  {:else}
    <Layer type="content">
      <div class="grid grid-cols-1 gap-12 lg:grid-cols-2 2xl:grid-cols-3">
        {#each rankedCompanies as company (company.CompanyID)}
          {@const masterCompany = companiesByID.get(company.CompanyID)}
          {@const adminLabel = userLabels.get(packCompanyUserLabelID(company.CompanyID, COMPANY_ADMINISTRATOR_USER_ID))}
          <CompanyCreditCard
            {company}
            administratorName={adminLabel ? companyUserDisplayName(adminLabel, COMPANY_ADMINISTRATOR_USER_ID) : ''}
            budget={budgetMeterByCompanyID.get(company.CompanyID)}
            onOpen={() => { openCompanyUsage(company); }}
            onEdit={masterCompany ? () => { onEdit(masterCompany); } : undefined}
          />
        {/each}
      </div>
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
          days={selectedCompanyDays.length ? selectedCompanyDays : selectedCompany.Days}
          selectedDay={selectedDay?.Day}
          onSelect={(day) => { selectedDay = day; layerView = 2; }}
        />
      </div>
    {:else if selectedCompany && layerView === 2}
      <div class="flex flex-col gap-12 text-[14px]">
        {#if !selectedDay || !selectedDetail}
          <div class="rounded-[8px] border border-slate-200 bg-slate-50 p-14 text-center text-slate-600">
            <T text="Select a day in the By day tab.|Seleccione un día en la pestaña Por día." />
          </div>
        {:else}
          <div class="flex flex-wrap items-center gap-12 rounded-[8px] bg-slate-50 p-10">
            <span class="ff-bold">{formatTime(selectedDay.Day, 'd-M-Y')}</span>
            <span>{formatN(selectedDetail.CPU)} CPU cr.</span>
            <span>{formatN(selectedDetail.Inference)} <T text="AI cr.|cr. IA" /></span>
          </div>
          <CompanyRouteCreditCards routes={selectedDetail.Routes} totalCPU={selectedDetail.CPU} />
        {/if}
      </div>
    {:else if selectedCompany && layerView === 3}
      <div class="text-[14px]">
        <CompanyUserCreditCards
          users={selectedUsers}
          companyID={selectedCompany.CompanyID}
          labels={userLabels}
        />
      </div>
    {/if}
  </Layer>
</div>
