<script lang="ts">
  import ClientesProveedoresView from './ClientesProveedoresView.svelte'
  import { ClientProviderType } from './clientes-proveedores.svelte'
</script>

<ClientesProveedoresView
  clientProviderType={ClientProviderType.CLIENT}
  pageTitle="Clientes"
  layerTitleSingular="Cliente"
/>
