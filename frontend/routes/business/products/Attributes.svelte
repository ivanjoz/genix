<script lang="ts">
import Input from '$components/form/Input.svelte';
import Button from '$components/buttons/Button.svelte';
import ColorPicker from '$components/form/ColorPicker.svelte';
import Modal from '$components/layers/Modal.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import VTable from '$components/vTable/VTable.svelte';
import { closeAllModals, openModal, tr } from '$core/store.svelte';
import { formatN } from '$libs/helpers';
import { productoAtributos, type IProduct, type IProductPresentation } from "./products.svelte";
    import type { ITableColumn } from '$components/vTable/types';

  const produtcoAtributosMap = new Map(productoAtributos.map(e => [e.id, e]))

  let {
    producto = $bindable()
  }: { producto: IProduct } = $props()

  let presentacionForm = $state({} as IProductPresentation)
  let tempCounter = -1

  const columns: ITableColumn<IProductPresentation>[] = [
    { header: "Attribute|Atributo",
      getValue: e => produtcoAtributosMap.get(e.at)?.name || ""
    },
    { header: "Name|Nombre",
      getValue: e => e.nm
    },
    { header: "Price|Precio",
      getValue: e => e.pc ? formatN(e.pc / 100,2) : ""
    },
    { header: "Price Diff.|Diff. Precio",
      getValue: e => e.pd ? formatN(e.pd / 100,2) : ""
    },
    { header: "SKU",
      getValue: e => e.sk || ""
    },
    { header: "Color", id: "color",
      getValue: e => e.cl
    },
    { header: "...", css: "px-6 py-1", headerCss: "w-42",
      buttonEditHandler(e) {
        presentacionForm = {...e}
        openModal(3)
      },
    }
  ]

  const presentacionesFiltered = $derived((producto.Presentations||[]).filter(x => x.ss))

</script>

<div class="flex justify-between mt-4" aria-label="Product presentations toolbar">
  <div></div>
  <Button label="Opens the modal to add a new product presentation or variant."
    color="green" icon="icon-[fa--plus]" css="s1" onClick={() => {
    presentacionForm = {
      at: producto.AtributosIDs?.[0] as number,
      id: tempCounter,
      ss: 1
    } as IProductPresentation
    tempCounter--
    console.log("presentacionForm", presentacionForm)
    openModal(3)
  }} />
</div>

<VTable columns={columns} css="mt-6"
  data={presentacionesFiltered}
  onRowClick={e => {
    presentacionForm = {...e}
  }}
>
  {#snippet cellRenderer(record: IProductPresentation, col: ITableColumn<IProductPresentation>)}
    {#if col.id === "color" && record.cl}
      <div class="flex justify-center w-full">
        <div class="_1 h-24 w-36" style="background-color:{record.cl}">
        </div>
      </div>
    {/if}
  {/snippet}
</VTable>

<Modal title="Product Presentation|Producto Presentación" id={3} size={4}
  saveButtonLabel="Agregar" saveIcon="icon-[fa--check]"
  onSave={() => {
    producto.Presentations = producto.Presentations || []
    const current = producto.Presentations.find(x => x.id === presentacionForm.id)
    if(current){
      Object.assign(current, presentacionForm)
    } else {
      producto.Presentations.push(presentacionForm)
    }
    producto.Presentations = [...producto.Presentations]
    closeAllModals()
  }}
  onDelete={() => {
    const current = producto.Presentations.find(x => x.id === presentacionForm.id)
    if(current && presentacionForm.id > 0){
      current.ss = 0
    } else {
      producto.Presentations = producto.Presentations.filter(x => x.id !== presentacionForm.id)
    }
    producto.Presentations = [...producto.Presentations]
    closeAllModals()
  }}
>
  <div class="grid grid-cols-24 gap-10 p-4" aria-label="Product presentation form with attribute, name, price, and color">
    <SearchSelect label="Attribute|Atributo" saveOn={presentacionForm} css="col-span-12"
      save="at" keyId="id" keyName="name"
      options={productoAtributos}
    />
    <Input label="Name|Nombre" saveOn={presentacionForm} css="col-span-12"
      save="nm"
    />
    <Input label="Price|Precio" saveOn={presentacionForm} css="col-span-12"
      save="pc" type="number" baseDecimals={2}
    />
    <Input label="Price Difference|Diferencia Precio" saveOn={presentacionForm} css="col-span-12"
      save="pd" type="number" baseDecimals={2}
    />
    <Input label="SKU" saveOn={presentacionForm} css="col-span-12"
      save="sk"
    />
    <div class="col-span-12">
      <ColorPicker label="Color|Color" saveOn={presentacionForm} save="cl"/>
    </div>
    <div class="mt-12 col-span-24 fs15">
      <i class="icon-[fa--exclamation-triangle]"></i> La información se guardará cuando se guarde el producto.
    </div>
  </div>
</Modal>

<style>
  ._1 {
    border: 1px solid rgba(0, 0, 0, 0.7);
  }
</style>
