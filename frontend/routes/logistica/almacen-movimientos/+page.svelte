<script lang="ts">
import Page from '$domain/Page.svelte';
import { tr } from '$core/store.svelte';
import T from '$components/misc/T.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import DateInput from '$components/form/DateInput.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { Loading, formatTime, throttle, highlString } from '$libs/helpers';
import ButtonLayer from '$components/buttons/ButtonLayer.svelte';
import KeyValueStrip from '$components/misc/KeyValueStrip.svelte';
import Input from '$components/form/Input.svelte';
import { getStaticRecordsByID } from '$libs/cache/cache-by-ids.svelte';
import { SvelteMap } from 'svelte/reactivity';
  import { AlmacenesService } from "../../negocio/sedes-almacenes/sedes-almacenes.svelte"
  import { ProductosService } from "../../negocio/productos/productos.svelte"
  import { UsuariosService } from "../../seguridad/usuarios/usuarios.svelte"
  import {
    queryAlmacenMovimientos, movimientoTipos,
    type IWarehouseProductMovement, type IProductStockLot
  } from "./almacen-movimientos.svelte"
    import { untrack } from "svelte";

  const almacenes = new AlmacenesService()
  const productos = new ProductosService(true)
  const usuariosService = new UsuariosService()

  // Get current date and 7 days ago as Unix days
  const getFechaUnix = (): number => {
    return Math.floor(Date.now() / (1000 * 60 * 60 * 24))
  }

  const dateFin = getFechaUnix()
  const dateInicio = dateFin - 7

  let form = $state({ dateFin, dateInicio, almacenID: 0, productoID: 0, tipo: 0, lotCode: "", documentID: 0, serialNumber: "" })
  let almacenMovimientos = $state([] as IWarehouseProductMovement[])
  let filterText = $state("")
  let isSearchOpen = $state(false)

  const lotsByID = new SvelteMap<number, IProductStockLot>()

  const getLotName = (lotID?: number) => {
    if (!lotID) { return "-" }
    return lotsByID.get(lotID)?.Name || `LOT-${lotID}`
  }

  const movimientoTiposMap = new Map(movimientoTipos.map(x => [x.id, x]))

  // Set default almacen when almacenes are loaded
  $effect(() => {
    if(almacenes?.Almacenes?.length > 0){
      untrack(() => {
        const almacenID = (almacenes.Almacenes || [])[0]?.ID
        form = { ...form, almacenID }
      })
    }
  })

  const consultarRegistros = async () => {
    const isDirectLookup = !!(form.serialNumber?.trim() || form.lotCode?.trim() || (form.documentID || 0) > 0)
    if (!isDirectLookup && !form.almacenID) {
      return
    }

    Loading.standard(tr("Querying records...|Consultando registros..."))
    try {
      const movimientos = await queryAlmacenMovimientos(form)

      const missingLotIDs: number[] = []
      for (const m of movimientos) {
        const lotID = m.LotID || 0
        if (lotID > 0 && !lotsByID.has(lotID)) {
          missingLotIDs.push(lotID)
        }
      }
      if (missingLotIDs.length > 0) {
        const fetched = await getStaticRecordsByID<IProductStockLot>('product-stock-lots-by-ids', missingLotIDs)
        for (const [lotID, lotRecord] of fetched) {
          lotsByID.set(lotID, lotRecord)
        }
      }
      for (const m of movimientos) {
        if (m.LotID) { m.lot = lotsByID.get(m.LotID) }
      }

      almacenMovimientos = movimientos
    } catch (error) {
      console.error("[almacen-movimientos] query error", error)
    } finally {
      Loading.remove()
    }
  }

  const almacenRender = (almacenID: number, cant: number) => {
    if (!almacenID) { return "" }
    const name = almacenes.AlmacenesMap.get(almacenID)?.Name || `Almacen-${almacenID}`
    return `<div class="flex items-center">
      <div class="mr-8">${name}</div>
      <div class="ff-mono text-blue-600">(</div>
      <div class="ff-mono">${cant}</div>
      <div class="ff-mono text-blue-600">)</div>
    </div>`
  }

  const columns: ITableColumn<IWarehouseProductMovement>[] = [
    {
      header: "Date & Time|Fecha Hora",
      headerCss: "w-120",
      css: "ff-mono px-6",
      getValue: e => formatTime(e.Created || 0, "d-M h:n") as string
    },
    {
      header: "Product|Producto",
      render: e => {
        const nombre = productos.recordsMap.get(e.ProductID || 0)?.Name || `Producto-${e.ProductID}`

        const words = filterText.toLowerCase().trim().split(" ").filter(x => x)
        const segments = highlString(nombre, words)

        let html = ""
        for (const seg of segments) {
          if (seg.highl) {
            html += `<span class="bg-yellow-200">${seg.text}</span>`
          } else {
            html += seg.text
          }
        }
        return html
      }
    },
    {
      header: "Batch|Lote",
      headerCss: "w-100",
      css: "text-purple-600 text-center px-6",
      getValue: e => getLotName(e.LotID)
    },
    {
      header: "SKU",
      headerCss: "w-100",
      css: "text-purple-600 text-center px-6",
      getValue: e => e.SerialNumber || "-"
    },
    {
      header: "Movement|Movimiento",
      headerCss: "w-120",
      css: "text-center px-6",
      render: e => {
        const mov = movimientoTiposMap.get(e.Type || 0)
        return mov?.name || "-"
      }
    },
    {
      header: "Quantity|Cantidad",
      headerCss: "w-100",
      css: "text-right ff-mono px-6",
      render: e => {
        return `<div class="flex justify-end ${(e.Quantity || 0) < 0 ? 'text-red-500' : 'text-blue-600'}">
          ${e.Quantity || 0}
        </div>`
      }
    },
    {
      header: "Source Warehouse|Almacén Origen",
      render: e => almacenRender(e.WarehouseRefID || 0, e.WarehouseRefQuantity || 0)
    },
    {
      header: "Destination Warehouse|Almacén Destino",
      render: e => almacenRender(e.WarehouseID || 0, e.WarehouseQuantity || 0)
    },
    {
      header: "Document|Documento",
      render: e => String(e.DocumentID || "")
    },
    {
      header: "User|Usuario",
      headerCss: "w-120",
      css: "text-center px-6",
      render: e => {
        const usuario = usuariosService.usuariosMap.get(e.CreatedBy || 1)?.Usuario || `Usuario-${e.CreatedBy}`
        const words = filterText.toLowerCase().trim().split(" ").filter(x => x)
        const segments = highlString(usuario, words)

        let html = ""
        for (const seg of segments) {
          if (seg.highl) {
            html += `<span class="bg-yellow-200">${seg.text}</span>`
          } else {
            html += seg.text
          }
        }
        return html
      }
    },
  ]
</script>

<Page title="Warehouse Movements|Almacén Movimientos">
  <div class="grid grid-cols-[auto_minmax(0,1fr)] items-start mb-12 gap-12 md:flex md:items-center md:justify-between">
    <div class="contents md:flex md:items-center md:w-full md:gap-12">
      <ButtonLayer buttonClass="bx-purple" bind:isOpen={isSearchOpen}
        horizontalOffset={0} useOutline={true}
        edgeMargin={0} buttonClassOnShow="bx-red"
        layerClass="w-560"
        icon="icon-search" iconOnShow="icon-cancel"
        label="Opens the search filter for warehouse movements."
      >
        <div class="w-full grid grid-cols-24 gap-12 p-12" aria-label="Warehouse movements search filter with date range, product, movement type, lot, and serial number">
          <SearchSelect
            bind:saveOn={form}
            save="almacenID"
            css="col-span-24"
            label="Warehouse|Almacén"
            keyId="ID"
            keyName="Name"
            options={almacenes?.Almacenes || []}
            placeholder=""
          />
          <DateInput
            label="Start Date|Fecha Inicio"
            css="col-span-12"
            save="dateInicio"
            bind:saveOn={form}
          />
          <DateInput
            label="End Date|Fecha Fin"
            css="col-span-12"
            save="dateFin"
            bind:saveOn={form}
          />
          <SearchSelect
            bind:saveOn={form}
            save="productoID"
            css="col-span-12"
            label="Product|Producto"
            keyId="ID"
            keyName="Name"
            options={productos.records}
            placeholder=""
          />
          <SearchSelect
            bind:saveOn={form}
            save="tipo"
            css="col-span-12"
            label="Movement Type|Tipo Movimiento"
            keyId="id"
            keyName="name"
            options={movimientoTipos}
            placeholder=""
          />
          <Input
            label="Batch Code|Código Lote"
            css="col-span-8"
            save="lotCode"
            bind:saveOn={form}
          />
          <Input
            label="Document #|N° Documento"
            css="col-span-8"
            save="documentID"
            type="number"
            bind:saveOn={form}
          />
          <Input
            label="Serial #|N° Serie"
            css="col-span-8"
            save="serialNumber"
            bind:saveOn={form}
          />
          <div class="col-span-24 flex items-center justify-center">
            <button class="px-16 py-8 bx-purple mt-8 h-44"
              aria-label="Consultar registros"
              onclick={ev => {
                ev.stopPropagation()
                consultarRegistros()
                isSearchOpen = false
              }}
            >
              Buscar <i class="icon-search"></i>
            </button>
          </div>
        </div>
      </ButtonLayer>
      <KeyValueStrip
        css="col-span-2 row-start-2 w-full md:w-auto"
        label1="Fec. Inicio"
        value1={form.dateInicio}
        getContent1={v => formatTime(v, "d-m-Y") as string}
        label2="Fec. Fin"
        value2={form.dateFin}
        getContent2={v => formatTime(v, "d-m-Y") as string}
        label3="Almacén"
        value3={form.almacenID}
        getContent3={id => almacenes.AlmacenesMap.get(Number(id))?.Name || "Todos"}
        label4="Producto"
        value4={form.productoID}
        getContent4={id => id ? (productos.recordsMap.get(Number(id))?.Name || `Producto-${id}`) : "Todos"}
        label5="Tipo"
        value5={form.tipo}
        getContent5={id => movimientoTipos.find(t => t.id === Number(id))?.name || "Todos"}
      />
    </div>
    <div class="relative col-start-2 row-start-1 flex items-start self-start w-full max-w-224 ml-auto md:mr-16 md:w-224">
      <div class="absolute left-12 text-gray-400">
        <i class="icon-search"></i>
      </div>
      <input
        class="w-full pl-36 bg-white pr-12 py-8 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
        autocomplete="off"
        type="text"
        placeholder="Buscar..."
        onkeyup={ev => {
          ev.stopPropagation()
          throttle(() => {
            filterText = ((ev.target as any).value || "").toLowerCase().trim()
          }, 150)
        }}
      />
    </div>
  </div>
  <VTable
    data={almacenMovimientos}
    columns={columns}
    css="w-full" cellCss="text-nowrap px-6 leading-[1.1]"
    tableCss="w-full"
    maxHeight="calc(100vh - 8rem - 12px)"
    filterText={filterText}
    getFilterContent={e => {
      const producto = productos.recordsMap.get(e.ProductID || 0)?.Name || ""
      const usuario = usuariosService.usuariosMap.get(e.CreatedBy || 1)?.Usuario || ""
      return [producto, usuario].join(" ").toLowerCase()
    }}
    useFilterCache={true}
  />
</Page>
