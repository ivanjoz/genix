<script lang="ts">
  import Page from "$components/Page.svelte"
  import SearchSelect from "$components/SearchSelect.svelte"
  import DateInput from "$components/DateInput.svelte"
  import VTable from "$components/VTable/vTable.svelte"
  import type { ITableColumn } from "$components/VTable"
  import { Loading, formatTime, throttle, Notify } from '$core/helpers'
  import { formatN } from "$shared/main"
  import { untrack } from "svelte"
  import { 
    CajasService, 
    getCajaMovimientos,
    cajaMovimientoTipos,
    type ICajaMovimiento,
    type IUsuario
  } from "../cajas/cajas.svelte"

  const cajas = new CajasService()
  const cajaMovimientoTiposMap = new Map(cajaMovimientoTipos.map(x => [x.id, x]))

  // Get current date and 7 days ago as Unix days
  const getFechaUnix = (): number => {
    return Math.floor(Date.now() / (1000 * 60 * 60 * 24))
  }

  const fechaFin = getFechaUnix()
  const fechaInicio = fechaFin - 7

  let form = $state({ fechaFin, fechaInicio, CajaID: 0 })
  let cajaMovimientos = $state([] as ICajaMovimiento[])
  let filterText = $state("")

  // Set default caja when cajas are loaded
  $effect(() => {
    if(cajas?.Cajas?.length > 0){
      untrack(() => {
        const CajaID = (cajas.Cajas || [])[0]?.ID
        form = { ...form, CajaID }
      })
    }
  })

  const consultarRegistros = async () => {
    if (!form.CajaID || !form.fechaInicio || !form.fechaFin) {
      Notify.failure("Debe seleccionar una caja y un rango de fechas.")
      return
    }

    Loading.standard("Consultando registros...")
    let result: ICajaMovimiento[]
    try {
      // Zone offset for Peru: -5 hours = -18000 seconds
      const zoneOffset = -18000
      const fechaHoraInicio = form.fechaInicio * 24 * 60 * 60 + zoneOffset
      const fechaHoraFin = (form.fechaFin + 1) * 24 * 60 * 60 + zoneOffset
      
      result = await getCajaMovimientos({
        CajaID: form.CajaID,
        fechaInicio: fechaHoraInicio,
        fechaFin: fechaHoraFin
      })
    } catch (error) {
      Loading.remove()
      return
    }

    Loading.remove()
    cajaMovimientos = result || []
    console.log("movimientos obtenidos: ", result)
  }

  const columns: ITableColumn<ICajaMovimiento>[] = [
    { 
      header: "Fecha Hora", 
      headerCss: "w-140",
      cellCss: "ff-mono px-6",
      getValue: e => formatTime(e.Created, "d-M h:n") as string
    },
    { 
      header: "Tipo Mov.",
      headerCss: "w-160",
      cellCss: "px-6",
      getValue: e => cajaMovimientoTiposMap.get(e.Tipo)?.name || ""
    },
    { 
      header: "Monto", 
      headerCss: "w-120",
      cellCss: "ff-mono text-right px-6",
      render: e => {
        const cssClass = e.Monto < 0 ? "text-red-500" : ""
        return `<span class="${cssClass}">${formatN(e.Monto / 100, 2)}</span>`
      }
    },
    { 
      header: "Saldo Final", 
      headerCss: "w-120",
      cellCss: "ff-mono text-right px-6",
      getValue: e => formatN(e.SaldoFinal / 100, 2) as string
    },
    { 
      header: "Nº Documento",
      headerCss: "w-140",
      cellCss: "text-center px-6",
      getValue: e => ""
    },
    { 
      header: "Usuario", 
      headerCss: "w-120",
      cellCss: "text-center px-6",
      getValue: e => e.Usuario?.usuario || ""
    }
  ]
</script>

<Page title="Cajas Movimientos">
  <div class="flex items-center justify-between mb-12">
    <div class="flex items-center w-full" style="max-width: 64rem;">
      <SearchSelect 
        bind:saveOn={form} 
        save="CajaID" 
        css="w-240 mr-12"
        label="Cajas & Bancos" 
        keyId="ID" 
        keyName="Nombre" 
        options={cajas?.Cajas || []}
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
      <button class="px-16 py-8 bx-blue mt-8 h-44"
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
    data={cajaMovimientos} 
    columns={columns}
    css="w-full"
    tableCss="w-full"
    maxHeight="calc(100vh - 8rem - 12px)" 
    filterText={filterText}
    getFilterContent={e => {
      const movTipo = cajaMovimientoTiposMap.get(e.Tipo)?.name || ""
      const usuario = e.Usuario?.usuario || ""
      return [movTipo, usuario].join(" ").toLowerCase()
    }}
    useFilterCache={true}
  />
</Page>
