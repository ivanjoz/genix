<script lang="ts">
  import ClientesProveedoresView from './CustomersView.svelte'
  import { ClientProviderType } from './customers.svelte'
</script>

<ClientesProveedoresView
  clientProviderType={ClientProviderType.CLIENT}
  pageTitle="Clients|Clientes"
  layerTitleSingular="Client|Cliente"
/>
