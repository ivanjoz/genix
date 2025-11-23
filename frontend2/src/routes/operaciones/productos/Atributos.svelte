<script lang="ts">
    import Input from "$components/Input.svelte";
    import Modal from "$components/Modal.svelte";
    import SearchCard from "$components/SearchCard.svelte";
    import SearchSelect from "$components/SearchSelect.svelte";
    import { VTable, type ITableColumn } from "$components/VTable";
    import { closeAllModals, Core, openModal } from "$core/store.svelte";
    import { productoAtributos, type IProducto, type IProductoPresentacion } from "./productos.svelte";

  const produtcoAtributosMap = new Map(productoAtributos.map(e => [e.id, e]))

  let { 
    producto = $bindable()
  }: { producto: IProducto } = $props()

  let presentacionForm = $state({} as IProductoPresentacion)

  const columns: ITableColumn<IProductoPresentacion>[] = [
    { header: "Atributo",
      getValue: e => produtcoAtributosMap.get(e.at)?.name || ""
    },
    { header: "Nombre",
      getValue: e => e.nm
    },
    { header: "Precio",
      getValue: e => e.pc
    },
    { header: "Diff. Precio",
      getValue: e => e.pd
    },
    { header: "Color",
      getValue: e => e.cl
    },
    { header: "...",
      buttonEditHandler(e, value) {
        console.log("editando")
      },
    }
  ]

</script>

<SearchCard css="col-span-24 flex items-start" label="ATRIBUTOS ::"
  options={productoAtributos} keyId="id" keyName="name"
  cardCss="grow" inputCss="w-[35%] md:w-180" bind:saveOn={producto}
  save="AtributosIDs"
/>

{#if (producto.AtributosIDs||[]).length === 0}
  <div><i class="icon-attention"></i>Debe seleccionar al menos 1 atributo</div>
{/if}

<div class="flex justify-between mt-6">
  <div></div>
  <!-- svelte-ignore a11y_consider_explicit_label -->
  <button class="bx-green s1" onclick={() => {
    presentacionForm = { at: producto.AtributosIDs?.[0] as number } as IProductoPresentacion
    console.log("presentacionForm", presentacionForm)
    openModal(3)
  }}>
    <i class="icon-plus"></i>
  </button>
</div>

<VTable columns={columns} css="mt-4"
  data={producto.Presentaciones||[]}
  onRowClick={e => {
    presentacionForm = {...e}
  }}
/>

<Modal title="Producto Presentación" id={3} size={4}
  onSave={() => {
    producto.Presentaciones = producto.Presentaciones || []
    producto.Presentaciones.push(presentacionForm)
    producto.Presentaciones = [...producto.Presentaciones]
    closeAllModals()
  }}
>
  <div class="grid grid-cols-24 gap-10">
    <SearchSelect label="Atributo" saveOn={presentacionForm} css="col-span-12"
      save="at" keyId="id" keyName="name"
      options={productoAtributos}
    />
    <Input label="Nombre" saveOn={presentacionForm} css="col-span-12"
      save="nm"
    />
    <Input label="Precio" saveOn={presentacionForm} css="col-span-12"
      save="pc" type="number"
    />
  </div>
</Modal>