<script lang="ts">
import Page from '$domain/Page.svelte';
import SearchSelect from '$components/SearchSelect.svelte';
import DateInput from '$components/DateInput.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { Loading, formatTime, throttle, highlString } from '$libs/helpers';
  import { AlmacenesService } from "../../negocio/sedes-almacenes/sedes-almacenes.svelte"
  import { ProductosService } from "../../negocio/productos/productos.svelte"
  import {
    queryAlmacenMovimientos, movimientoTipos,
    type IAlmacenMovimiento, type IUsuario, type IProducto
  } from "./almacen-movimientos.svelte"
    import { untrack } from "svelte";

  const almacenes = new AlmacenesService()
  const productos = new ProductosService()

  // Get current date and 7 days ago as Unix days
  const getFechaUnix = (): number => {
    return Math.floor(Date.now() / (1000 * 60 * 60 * 24))
  }

  const fechaFin = getFechaUnix()
  const fechaInicio = fechaFin - 7

  let form = $state({ fechaFin, fechaInicio, almacenID: 0 })
  let almacenMovimientos = $state([] as IAlmacenMovimiento[])
  let filterText = $state("")

  const usuariosMap: Map<number, IUsuario> = new Map()
  const productosMap: Map<number, IProducto> = new Map()
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
    if (!form.almacenID) {
      return
    }

    Loading.standard("Consultando registros...")
    let result
    try {
      result = await queryAlmacenMovimientos(form)
    } catch (error) {
      Loading.remove()
      return
    }

    console.log("registros obtenidos: ", result)

    for (const e of result.Productos) {
      productosMap.set(e.ID, e)
    }
    for (const e of result.Usuarios) {
      usuariosMap.set(e.id, e)
    }

    almacenMovimientos = result.Movimientos
    Loading.remove()
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

  const columns: ITableColumn<IAlmacenMovimiento>[] = [
    {
      header: "Fecha Hora",
      headerCss: "w-120",
      cellCss: "ff-mono px-6",
      getValue: e => formatTime(e.Created, "d-M h:n") as string
    },
    {
      header: "Producto",
      render: e => {
        const nombre = productosMap.get(e.ProductoID)?.Nombre || `Producto-${e.ProductoID}`
        
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
      getValue: e => e.Lote || "-"
    },
    {
      header: "SKU",
      headerCss: "w-100",
      cellCss: "text-purple-600 text-center px-6",
      getValue: e => e.SKU || "-"
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
        return `<div class="flex justify-end ${e.Cantidad < 0 ? 'text-red-500' : 'text-blue-600'}">
          ${e.Cantidad}
        </div>`
      }
    },
    {
      header: "Almacén Origen",
      render: e => almacenRender(e.AlmacenOrigenID, e.AlmacenOrigenCantidad)
    },
    {
      header: "Almacén Destino",
      render: e => almacenRender(e.AlmacenID, e.AlmacenCantidad)
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
        const usuario = usuariosMap.get(e.CreatedBy || 1)?.usuario || `Usuario-${e.CreatedBy}`
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
  <div class="flex items-center justify-between mb-12">
    <div class="flex items-center w-full" style="max-width: 64rem;">
      <SearchSelect
        bind:saveOn={form}
        save="almacenID"
        css="w-240 mr-12"
        label="Almacén"
        keyId="ID"
        keyName="Nombre"
        options={almacenes?.Almacenes || []}
        placeholder=""
        required={true}
      />
      <DateInput
        label="Fecha Inicio"
        css="w-140 mr-12"
        save="fechaInicio"
        bind:saveOn={form}
      />
      <DateInput
        label="Fecha Fin"
        css="w-140 mr-12"
        save="fechaFin"
        bind:saveOn={form}
      />
      <button class="px-16 py-8 bx-purple mt-8 h-44"
        aria-label="Consultar registros"
        onclick={ev => {
          ev.stopPropagation()
          consultarRegistros()
        }}
      >
        <i class="icon-search"></i>
      </button>
    </div>
    <div class="flex items-center mr-16 w-224 ml-auto relative">
      <div class="absolute left-12 text-gray-400">
        <i class="icon-search"></i>
      </div>
      <input
        class="w-full pl-36 pr-12 py-8 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
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
    css="w-full"
    tableCss="w-full"
    maxHeight="calc(100vh - 8rem - 12px)"
    filterText={filterText}
    getFilterContent={e => {
      const producto = productosMap.get(e.ProductoID)?.Nombre || ""
      const usuario = usuariosMap.get(e.CreatedBy || 1)?.usuario || ""
      return [producto, usuario].join(" ").toLowerCase()
    }}
    useFilterCache={true}
  />
</Page>
