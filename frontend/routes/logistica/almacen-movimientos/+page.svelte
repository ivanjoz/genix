<script lang="ts">
import Page from '$domain/Page.svelte';
import SearchSelect from '$components/SearchSelect.svelte';
import DateInput from '$components/DateInput.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { Loading, formatTime, throttle, highlString } from '$libs/helpers';
import ButtonLayer from '$components/ButtonLayer.svelte';
import KeyValueStrip from '$components/micro/KeyValueStrip.svelte';
import Input from '$components/Input.svelte';
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

  const fechaFin = getFechaUnix()
  const fechaInicio = fechaFin - 7

  let form = $state({ fechaFin, fechaInicio, almacenID: 0, productoID: 0, tipo: 0, lotCode: "", documentID: 0, serialNumber: "" })
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

    Loading.standard("Consultando registros...")
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
    const name = almacenes.AlmacenesMap.get(almacenID)?.Nombre || `Almacen-${almacenID}`
    return `<div class="flex items-center">
      <div class="mr-8">${name}</div>
      <div class="ff-mono text-blue-600">(</div>
      <div class="ff-mono">${cant}</div>
      <div class="ff-mono text-blue-600">)</div>
    </div>`
  }

  const columns: ITableColumn<IWarehouseProductMovement>[] = [
    {
      header: "Fecha Hora",
      headerCss: "w-120",
      cellCss: "ff-mono px-6",
      getValue: e => formatTime(e.Created, "d-M h:n") as string
    },
    {
      header: "Producto",
      render: e => {
        const nombre = productos.recordsMap.get(e.ProductoID)?.Nombre || `Producto-${e.ProductoID}`

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
      header: "Lote",
      headerCss: "w-100",
      cellCss: "text-purple-600 text-center px-6",
      getValue: e => getLotName(e.LotID)
    },
    {
      header: "SKU",
      headerCss: "w-100",
      cellCss: "text-purple-600 text-center px-6",
      getValue: e => e.SerialNumber || "-"
    },
    {
      header: "Movimiento",
      headerCss: "w-120",
      cellCss: "text-center px-6",
      render: e => {
        const mov = movimientoTiposMap.get(e.Tipo)
        return mov?.name || "-"
      }
    },
    {
      header: "Cantidad",
      headerCss: "w-100",
      cellCss: "text-right ff-mono px-6",
      render: e => {
        return `<div class="flex justify-end ${(e.Quantity || 0) < 0 ? 'text-red-500' : 'text-blue-600'}">
          ${e.Quantity || 0}
        </div>`
      }
    },
    {
      header: "Almacén Origen",
      render: e => almacenRender(e.WarehouseRefID, e.WarehouseRefQuantity)
    },
    {
      header: "Almacén Destino",
      render: e => almacenRender(e.WarehouseID, e.WarehouseQuantity)
    },
    {
      header: "Documento",
      render: e => String(e.DocumentID || "")
    },
    {
      header: "Usuario",
      headerCss: "w-120",
      cellCss: "text-center px-6",
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

<Page title="Almacén Movimientos">
  <div class="grid grid-cols-[auto_minmax(0,1fr)] items-start mb-12 gap-12 md:flex md:items-center md:justify-between">
    <div class="contents md:flex md:items-center md:w-full md:gap-12">
      <ButtonLayer buttonClass="bx-purple" bind:isOpen={isSearchOpen}
        horizontalOffset={0} useOutline={true}
        edgeMargin={0} buttonClassOnShow="bx-red"
        layerClass="w-560"
        icon="icon-search" iconOnShow="icon-cancel"
      >
        <div class="w-full grid grid-cols-24 gap-12 p-12">
          <SearchSelect
            bind:saveOn={form}
            save="almacenID"
            css="col-span-24"
            label="Almacén"
            keyId="ID"
            keyName="Nombre"
            options={almacenes?.Almacenes || []}
            placeholder=""
          />
          <DateInput
            label="Fecha Inicio"
            css="col-span-12"
            save="fechaInicio"
            bind:saveOn={form}
          />
          <DateInput
            label="Fecha Fin"
            css="col-span-12"
            save="fechaFin"
            bind:saveOn={form}
          />
          <SearchSelect
            bind:saveOn={form}
            save="productoID"
            css="col-span-12"
            label="Producto"
            keyId="ID"
            keyName="Nombre"
            options={productos.records}
            placeholder=""
          />
          <SearchSelect
            bind:saveOn={form}
            save="tipo"
            css="col-span-12"
            label="Tipo Movimiento"
            keyId="id"
            keyName="name"
            options={movimientoTipos}
            placeholder=""
          />
          <Input
            label="Código Lote"
            css="col-span-8"
            save="lotCode"
            bind:saveOn={form}
          />
          <Input
            label="N° Documento"
            css="col-span-8"
            save="documentID"
            type="number"
            bind:saveOn={form}
          />
          <Input
            label="N° Serie"
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
        value1={form.fechaInicio}
        getContent1={v => formatTime(v, "d-m-Y") as string}
        label2="Fec. Fin"
        value2={form.fechaFin}
        getContent2={v => formatTime(v, "d-m-Y") as string}
        label3="Almacén"
        value3={form.almacenID}
        getContent3={id => almacenes.AlmacenesMap.get(Number(id))?.Nombre || "Todos"}
        label4="Producto"
        value4={form.productoID}
        getContent4={id => id ? (productos.recordsMap.get(Number(id))?.Nombre || `Producto-${id}`) : "Todos"}
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
      const producto = productos.recordsMap.get(e.ProductoID)?.Nombre || ""
      const usuario = usuariosService.usuariosMap.get(e.CreatedBy || 1)?.Usuario || ""
      return [producto, usuario].join(" ").toLowerCase()
    }}
    useFilterCache={true}
  />
</Page>
