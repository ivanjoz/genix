<script lang="ts">
  import CrmReportPage from '../CrmReportPage.svelte'
  import ReportLostReasons from '../ReportLostReasons.svelte'
  import ReportScorecard from '../ReportScorecard.svelte'
</script>

<!-- Performance: how each seller did, and why opportunities were lost. -->
<CrmReportPage title="CRM Performance|Desempeño CRM" views={[
  {
    label: 'Sellers|Vendedores', component: ReportScorecard,
    description: 'Win rate, won amount, sales cycle, velocity, activities and goal of each seller, with team subtotals.|Tasa de cierre, monto ganado, ciclo de venta, velocidad, actividades y meta de cada vendedor, con subtotales por equipo.',
  },
  {
    label: 'Lost reasons|Motivos de pérdida', component: ReportLostReasons,
    description: 'Opportunities still lost at the end of the range, by reason, seller and stage.|Oportunidades que siguen perdidas al final del rango, por motivo, vendedor y etapa.',
  },
]} />
