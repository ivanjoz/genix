<script lang="ts">
import { useUI } from '@genix/ui'
import Page from '$domain/Page.svelte'
import PurchaseOrderCreate from './PurchaseOrderCreate.svelte'
import PurchaseOrderReport from './PurchaseOrderReport.svelte'

const pageOptions = [{ id: 1, name: 'Orders|Órdenes' }, { id: 2, name: 'Report|Reporte' }]
const ui = useUI()
</script>

<Page title="Purchase Orders|Órdenes de Compra" options={pageOptions}>
  {#if ui.state.pageOptionSelected === 1}
    <PurchaseOrderCreate />
  {/if}
  {#if ui.state.pageOptionSelected === 2}
    <PurchaseOrderReport />
  {/if}
</Page>
