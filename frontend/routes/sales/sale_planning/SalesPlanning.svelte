<script lang="ts">
  import Page from '$domain/Page.svelte'
  import T from '$components/misc/T.svelte'
  import { Core } from '$core/store.svelte'
  import SalesPlanningMantainer from './SalesPlanningMantainer.svelte'
</script>

<Page title="Sales Planning|Proyección Ventas"
  options={[
    { id: 1, name: 'Planning|Planificación' },
    { id: 2, name: 'Report|Reporte' },
  ]}
>
  {#if Core.pageOptionSelected === 1}
    <SalesPlanningMantainer />
  {/if}
  {#if Core.pageOptionSelected === 2}
    <div class="flex items-center justify-center h-200 c-gray-500">
      <T text="Coming soon|Próximamente" />
    </div>
  {/if}
</Page>
