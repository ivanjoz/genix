<script lang="ts">
    import { derived } from "svelte/store";
    import type { ListasCompartidasService } from "./productos.svelte";
    import Modal from "$components/Modal.svelte";
    import { closeModal, openModal } from "$core/store.svelte";


  const { 
    listas, 
    origin
  }: { 
    listas: ListasCompartidasService,
    origin: 1 /* Categorías */ | 2 /* Marcas */,
  } = $props()

  const categorias = $derived.by(() => {
    console.log("listas getted:", $state.snapshot(listas))
    return listas.ListaRecordsMap.get(origin) || []
  })

  /*

  <div class="w-full">
  <button onclick={() => {
    listas.addNew({ ListaID: 1, ID: 99, Nombre: "Nueva", ss: 1, upd: 111 })
  }}>Test Add</button>
</div>
  */
</script>

<div class="w-full flex flex-wrap gap-12">
  {#each categorias as e }
    <div class="_1 px-12 py-10 w-250 min-h-140" role="button" tabindex="0"
      onkeydown={ev => {
        ev.stopPropagation()
        if (ev.key === 'Enter' || ev.key === ' ') {
          ev.preventDefault()
          openModal(2)
        }
      }}
      onclick={ev => {
        ev.stopPropagation()
        openModal(2)
      }}>
      <div class="min-h-70 _2 mb-2">

      </div>
      <div class="fs17 ff-semibold">{e.Nombre}</div>
      <div class="fs15">{e.Descripcion}</div>
    </div>
  {/each}
</div>
<Modal title="CATEGORÍAS" id={2} size={6}
  onClose={() => {
    closeModal(2)
  }}
  onSave={() => {

  }}
  onDelete={() => {

  }}
  >
  <h1>hola</h1>
</Modal>

<style>
  ._1 {
    background-color: rgb(255, 255, 255);
    border-radius: 8px;
    box-shadow: rgba(0, 0, 0, 0.1) 0px 4px 6px -1px, rgba(0, 0, 0, 0.06) 0px 2px 4px -1px;
  }
  ._1:hover {
    outline: 2px solid rgba(0, 0, 0, 0.5);
  }

  ._2 {
    background-color: rgb(235, 233, 245);
  }
</style>