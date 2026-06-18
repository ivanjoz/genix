<script lang="ts">
  import ClientesProveedoresView from '../customers/CustomersView.svelte'
  import { ClientProviderType } from '../customers/customers.svelte'
</script>

<ClientesProveedoresView
  clientProviderType={ClientProviderType.PROVIDER}
  pageTitle="Suppliers|Proveedores"
  layerTitleSingular="Supplier|Proveedor"
/>
