<script>
  import Layer from "../../../components/Layer.svelte";
  import Page from "../../../components/Page.svelte";
  import { Core } from "../../../core/store.svelte";

  let searchText = $state("")

</script>

<Page sideLayerSize={770}>
  <div class="flex items-center">
    
    <div>hola</div>
    <button class="bx-green ml-auto" onclick={ev => {
      ev.stopPropagation()
      Core.showSideLayer = 1
    }}>
      <i class="icon-plus"></i>Nuevo
    </button>
  </div>
  <Layer css="p-8" title="nueva capa">
    <div><h1>hola mundo 2</h1></div>
  </Layer>
</Page>