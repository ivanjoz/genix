<script lang="ts">
  import ClientesProveedoresView from '../clientes/ClientesProveedoresView.svelte'
  import { ClientProviderType } from '../clientes/clientes-proveedores.svelte'
</script>

<ClientesProveedoresView
  clientProviderType={ClientProviderType.PROVIDER}
  pageTitle="Suppliers|Proveedores"
  layerTitleSingular="Supplier|Proveedor"
/>
