<script lang="ts">
import Input from '$components/Input.svelte';
import SearchSelect from '$components/SearchSelect.svelte';
import WebRTCPanel from './WebRTCPanel.svelte';
import Page from '$domain/Page.svelte';
    import { Core } from '$core/store.svelte';

  let options = [
    { id: 1, nombre: "Ivan Joseph Angulo" },
    { id: 2, nombre: "Pedro Pascal Armando" },
    { id: 3, nombre: "Sergio Bonamasa" },
    { id: 4, nombre: "Ricardo Ríos del Agua" },
  ]

  let form = {};
</script>

<Page title="New Page"
  options={[{ id:1, name: "WebRTC" }]}
>
  {#if Core.pageOptionSelected === 1}
 	<WebRTCPanel />
  {/if}
</Page>

<style>

</style>
