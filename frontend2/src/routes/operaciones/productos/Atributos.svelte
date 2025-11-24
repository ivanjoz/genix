<script lang="ts">
    import Input from "$components/Input.svelte";
    import ColorPicker from "$components/micro/ColorPicker.svelte";
    import Modal from "$components/Modal.svelte";
    import SearchSelect from "$components/SearchSelect.svelte";
    import { VTable, type ITableColumn } from "$components/VTable";
    import { closeAllModals, openModal } from "$core/store.svelte";
    import { formatN } from "$shared/main";
    import { productoAtributos, type IProducto, type IProductoPresentacion } from "./productos.svelte";

  const produtcoAtributosMap = new Map(productoAtributos.map(e => [e.id, e]))

  let { 
    producto = $bindable()
  }: { producto: IProducto } = $props()

  let presentacionForm = $state({} as IProductoPresentacion)
  let tempCounter = -1

  const columns: ITableColumn<IProductoPresentacion>[] = [
    { header: "Atributo",
      getValue: e => produtcoAtributosMap.get(e.at)?.name || ""
    },
    { header: "Nombre",
      getValue: e => e.nm
    },
    { header: "Precio",
      getValue: e => e.pc ? formatN(e.pc / 100,2) : ""
    },
    { header: "Diff. Precio",
      getValue: e => e.pd ? formatN(e.pd / 100,2) : ""
    },
    { header: "Color", id: "color",
      getValue: e => e.cl
    },
    { header: "...", cellCss: "px-6 py-1", headerCss: "w-42",
      buttonEditHandler(e) {
        presentacionForm = {...e}
        openModal(3)
      },
    }
  ]

  const presentacionesFiltered = $derived(producto.Presentaciones.filter(x => x.ss))

</script>

<div class="flex justify-between mt-4">
  <div></div>
  <!-- svelte-ignore a11y_consider_explicit_label -->
  <button class="bx-green s1" onclick={() => {
    presentacionForm = { 
      at: producto.AtributosIDs?.[0] as number,
      id: tempCounter,
      ss: 1
    } as IProductoPresentacion
    tempCounter--
    console.log("presentacionForm", presentacionForm)
    openModal(3)
  }}>
    <i class="icon-plus"></i>
  </button>
</div>

<VTable columns={columns} css="mt-6"
  data={producto.Presentaciones||[]}
  onRowClick={e => {
    presentacionForm = {...e}
  }}
>
  {#snippet cellRenderer(record: IProductoPresentacion, col: ITableColumn<IProductoPresentacion>)}
    {#if col.id === "color" && record.cl}
      <div class="flex justify-center w-full">
        <div class="_1 h-24 w-36" style="background-color:{record.cl}">
        </div>  
      </div>
    {/if}
  {/snippet}
</VTable>

<Modal title="Producto Presentación" id={3} size={4}
  saveButtonLabel="Agregar" saveIcon="icon-ok"
  onSave={() => {
    producto.Presentaciones = producto.Presentaciones || []
    const current = producto.Presentaciones.find(x => x.id === presentacionForm.id)
    if(current){
      Object.assign(current, presentacionForm)
    } else {
      producto.Presentaciones.push(presentacionForm)
    }
    producto.Presentaciones = [...producto.Presentaciones]
    closeAllModals()
  }}
  onDelete={() => {
    const current = producto.Presentaciones.find(x => x.id === presentacionForm.id)
    if(current && presentacionForm.id > 0){
      current.ss = 0
    } else {
      producto.Presentaciones = producto.Presentaciones.filter(x => x.id !== presentacionForm.id)
    }
    producto.Presentaciones = [...producto.Presentaciones]
    closeAllModals()
  }}
>
  <div class="grid grid-cols-24 gap-10 p-4">
    <SearchSelect label="Atributo" saveOn={presentacionForm} css="col-span-12"
      save="at" keyId="id" keyName="name"
      options={productoAtributos}
    />
    <Input label="Nombre" saveOn={presentacionForm} css="col-span-12"
      save="nm"
    />
    <Input label="Precio" saveOn={presentacionForm} css="col-span-12"
      save="pc" type="number" baseDecimals={2}
    />
    <Input label="Diferencia Precio" saveOn={presentacionForm} css="col-span-12"
      save="pd" type="number" baseDecimals={2}
    />
    <div class="col-span-12">
      <ColorPicker label="Color" saveOn={presentacionForm} save="cl"/>
    </div>
    <div class="mt-12 col-span-24 fs15">
      <i class="icon-attention"></i> La información se guardará cuando se guarde el producto.
    </div>
  </div>
</Modal>

<style>
  ._1 {
    border: 1px solid rgba(0, 0, 0, 0.7);
  }
</style>