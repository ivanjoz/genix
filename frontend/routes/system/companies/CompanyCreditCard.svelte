<script lang="ts">
  import { useUI } from '@genix/ui';
  import { ChartCanvas, type ChartCanvasSeries } from '@genix/ui/charts';
  import Button from '$components/buttons/Button.svelte';
  import Card from '$components/cards/Card.svelte';
  import T from '$components/misc/T.svelte';
  import { formatN, formatTime } from '$libs/helpers';
  import type { ICompanyCreditUsageRanked } from './company-credit-usage.model';

  let {
    company,
    onOpen,
    onEdit,
  }: {
    company: ICompanyCreditUsageRanked;
    onOpen: () => void;
    onEdit?: () => void;
  } = $props();

  const ui = useUI();
  const chartSeries = $derived<ChartCanvasSeries[]>([
    { type: 'bar', name: 'CPU credits', color: '#10b981', values: company.Days.map((day) => day.CPU) },
    { type: 'bar', name: 'Inference credits', color: '#a855f7', values: company.Days.map((day) => day.Inference) },
  ]);
  const administratorName = $derived(
    company.AdminName && company.AdminName !== company.AdminUser ? company.AdminName : '',
  );
  const formatDayLabel = (day: string | number) => String(formatTime(Number(day), 'd-M') || '');
</script>

<Card
  label={`Open 30-day credit usage for ${company.Company}|Abrir uso de créditos de 30 días de ${company.Company}`}
  css="group relative flex h-full min-h-250 flex-col rounded-[12px] border border-slate-200 bg-white p-14 shadow-sm transition-shadow hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-violet-400 focus-visible:ring-offset-2"
  onClick={onOpen}
>
  <div class="min-w-0 pr-96 md:pr-42">
    <div class="truncate text-[18px] ff-bold text-slate-900">{company.Company}</div>
    <div class="text-blue-700">
      {#if administratorName}
        <div class="truncate ff-bold">{administratorName}</div>
      {:else}
        <div class="text-slate-500"><T text="Administrator unavailable|Administrador no disponible" /></div>
      {/if}
    </div>
  </div>

  <div class="absolute right-12 top-12 z-10 flex h-32 items-center justify-end gap-8">
    <!-- Desktop swaps the identifier for edit; touch layouts keep both controls discoverable. -->
    <span class="ff-mono text-slate-500 transition-opacity md:group-hover:opacity-0 md:group-focus-within:opacity-0">
      #{company.CompanyID}
    </span>
    {#if onEdit}
      <Button
        icon="icon-[fa--pencil]"
        color="purple"
        useCircle={true}
        label={`Edit ${company.Company}|Editar ${company.Company}`}
        css="shrink-0 transition-opacity md:absolute md:right-0 md:opacity-0 md:group-hover:opacity-100 md:group-focus-within:opacity-100 md:focus:opacity-100"
        onClick={onEdit}
      />
    {/if}
  </div>

  <div class="mt-12 rounded-[9px] border border-slate-200 bg-slate-50 p-10">
    <div class="flex flex-wrap items-center gap-x-14 gap-y-4">
      <span class="ff-bold text-slate-700"><T text="30-day usage|Uso de 30 días" /></span>
      <span class="ml-auto flex items-center gap-4 ff-mono">
        <span class="h-8 w-8 rounded-[2px] bg-emerald-500"></span>{formatN(company.CPU)} CPU
      </span>
      <span class="flex items-center gap-4 ff-mono">
        <span class="h-8 w-8 rounded-[2px] bg-purple-500"></span>{formatN(company.Inference)} <T text="AI|IA" />
      </span>
    </div>
    <div class="mt-8">
      <ChartCanvas
        id={`company-credit-card-${company.CompanyID}`}
        data={chartSeries}
        barMode="grouped"
        dateLabels={company.Days.map((day) => day.Day)}
        dateLabelEvery={ui.state.deviceType === 3 ? 10 : 6	}
        dateLabelFormatter={formatDayLabel}
        useHtmlRendered={true}
        showBottomBaseline={true}
        yAxisStepSize={100}
        height={92}
        showTooltip={true}
      />
    </div>
  </div>
</Card>
