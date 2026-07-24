<script lang="ts">
  import Page from '$domain/Page.svelte'
  import T from '$components/misc/T.svelte'
  import { useUI } from '@genix/ui'
  import SalesPlanningMantainer from './SalesPlanningMantainer.svelte'
  const ui = useUI()
</script>

<Page title="Sales Planning|Proyección Ventas"
  options={[
    { id: 1, name: 'Planning|Planificación' },
    { id: 2, name: 'Report|Reporte' },
  ]}
>
  {#if ui.state.pageOptionSelected === 1}
    <SalesPlanningMantainer />
  {/if}
  {#if ui.state.pageOptionSelected === 2}
    <div class="flex items-center justify-center h-200 c-gray-500">
      <T text="Coming soon|Próximamente" />
    </div>
  {/if}
</Page>
