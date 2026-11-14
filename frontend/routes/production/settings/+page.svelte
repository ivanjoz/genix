<script lang="ts">
  import Page from '#domain/Page.svelte'
  import { useUI } from '@genix/ui'
  import LaborTypesTab from './LaborTypesTab.svelte'
  import StagesTab from './StagesTab.svelte'

  // Production settings (docs/MANUFACTURING_PLAN.md §9.1, §9.4): the labor types recipes and orders plan
  // in minutes, and the stages of the production orders board.
  const ui = useUI()

  const pageOptions = [
    { id: 1, name: 'Labor types|Tipos de trabajo' },
    { id: 2, name: 'Stages|Etapas' },
  ]
</script>

<Page title="Production settings|Configuración de producción" options={pageOptions}>
  {#if ui.state.pageOptionSelected === 2}
    <StagesTab />
  {:else}
    <LaborTypesTab />
  {/if}
</Page>
