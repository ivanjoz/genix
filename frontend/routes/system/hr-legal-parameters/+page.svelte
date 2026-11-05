<script lang="ts">
  import Page from '#domain/Page.svelte'
  import LegalParametersPanel from '#domain/hr/LegalParametersPanel.svelte'
  import { LegalParametersService } from '#services/hr/legal-parameters.svelte.ts'

  // SaaS only: the legal payroll values every Peruvian company's payroll reads (RMV, UIT, rates)
  // and the monthly SBS table of the AFPs. A change takes effect from its date; closed payslips
  // keep the values they were calculated with.
  const legal = new LegalParametersService(true)
</script>

<Page title="Labor Parameters|Parámetros Laborales">
  <LegalParametersPanel {legal} editable={true} />
</Page>
