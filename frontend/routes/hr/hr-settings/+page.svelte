<script lang="ts">
  import Page from '#domain/Page.svelte'
  import LegalParametersPanel from '#domain/hr/LegalParametersPanel.svelte'
  import { useUI } from '@genix/ui'
  import { LegalParametersService } from '#services/hr/legal-parameters.svelte.ts'
  import { PayrollConfigService } from '#services/hr/payroll.svelte.ts'
  import ConceptsTab from './ConceptsTab.svelte'

  const ui = useUI()
  const config = new PayrollConfigService(true)
  const legal = new LegalParametersService(true)

  const pageOptions = [
    { id: 1, name: 'Concepts|Conceptos' },
    { id: 2, name: 'Legal values|Valores legales' },
  ]
</script>

<Page title="HR Settings|Configuración RRHH" options={pageOptions}>
  {#if ui.state.pageOptionSelected === 2}
    <LegalParametersPanel {legal} />
  {:else}
    <ConceptsTab {config} />
  {/if}
</Page>
