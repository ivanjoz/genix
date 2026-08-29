<script lang="ts">
  import ClientesProveedoresView from '$domain/ClientProviderMaintainer.svelte'
  import { ClientProviderType } from '$services/crm/client-provider.svelte'
</script>

<ClientesProveedoresView
  clientProviderType={ClientProviderType.CLIENT}
  pageTitle="Clients|Clientes"
  layerTitleSingular="Client|Cliente"
/>
