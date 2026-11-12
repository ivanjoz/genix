<script lang="ts">
  import CrmReportPage from '../CrmReportPage.svelte'
  import ReportDailyState from '../ReportDailyState.svelte'
  import ReportWaterfall from '../ReportWaterfall.svelte'
</script>

<!-- Flow: what the open pipeline holds day by day, and how it went from the opening to the closing of the range. -->
<CrmReportPage title="CRM Flow|Flujo CRM" views={[
  {
    label: 'Daily state|Estado diario', component: ReportDailyState,
    description: 'Open opportunities in each stage at the end of each day.|Oportunidades abiertas en cada etapa al cierre de cada día.',
  },
  {
    label: 'Waterfall|Cascada', component: ReportWaterfall,
    description: 'How the open pipeline went from the opening to the closing of the range: new, renegotiated, won, lost, reopened and handed over.|Cómo pasó el pipeline abierto de la apertura al cierre del rango: nuevas, renegociadas, ganadas, perdidas, reabiertas y reasignadas.',
  },
]} />
