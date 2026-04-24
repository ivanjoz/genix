<script lang="ts">
import { Core } from '$core/store.svelte'
import Page from '$domain/Page.svelte'
import PurchaseOrderCreate from './PurchaseOrderCreate.svelte'
import PurchaseOrderReport from './PurchaseOrderReport.svelte'

const pageOptions = [{ id: 1, name: 'Órdenes' }, { id: 2, name: 'Reporte' }]
</script>

<Page title="Órdenes de Compra" options={pageOptions}>
  {#if Core.pageOptionSelected === 1}
    <PurchaseOrderCreate />
  {/if}
  {#if Core.pageOptionSelected === 2}
    <PurchaseOrderReport />
  {/if}
</Page>
