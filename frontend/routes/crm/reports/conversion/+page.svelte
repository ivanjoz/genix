<script lang="ts">
  import CrmReportPage from '../CrmReportPage.svelte'
  import ReportFunnel from '../ReportFunnel.svelte'
  import ReportNetProgress from '../ReportNetProgress.svelte'
</script>

<!-- Conversion: how far opportunities get through the stages. -->
<CrmReportPage title="CRM Conversion|Conversión CRM" views={[
  {
    label: 'Net progress|Avance neto', component: ReportNetProgress,
    description: 'Opportunities that reached each stage, net of steps back, losses and hand-overs. Negatives mean the seller lost ground.|Oportunidades que llegaron a cada etapa, netas de retrocesos, pérdidas y reasignaciones. Los negativos indican que el vendedor perdió terreno.',
  },
  {
    label: 'Funnel|Embudo', component: ReportFunnel,
    description: 'Of the opportunities that entered the pipeline in the range, how far they got, and how long they stayed in each stage.|De las oportunidades que entraron al pipeline en el rango, hasta dónde llegaron y cuánto estuvieron en cada etapa.',
  },
]} />
