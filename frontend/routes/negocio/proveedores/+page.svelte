<script lang="ts">
  import ClientesProveedoresView from '../clientes/ClientesProveedoresView.svelte'
  import { ClientProviderType } from '../clientes/clientes-proveedores.svelte'
</script>

<ClientesProveedoresView
  clientProviderType={ClientProviderType.PROVIDER}
  pageTitle="Proveedores"
  layerTitleSingular="Proveedor"
/>
