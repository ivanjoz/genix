<script lang="ts">
import Page from '$domain/Page.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import DateInput from '$components/form/DateInput.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import RecordByIDText from '$components/misc/RecordByIDText.svelte';
import { Loading, formatTime, throttle, Notify } from '$libs/helpers';
import { formatN } from '$libs/helpers';
import { tr } from '$core/store.svelte';
  import { untrack } from "svelte"
  import {
    CajasService,
    getCajaMovimientos,
    cajaMovimientoTipos,
    type ICashBankMovement,
  } from "../cash-banks/cajas.svelte"

  const cajas = new CajasService()
  const cajaMovimientoTiposMap = new Map(cajaMovimientoTipos.map(x => [x.id, x]))

  const getFechaUnix = (): number => {
    return Math.floor(Date.now() / (1000 * 60 * 60 * 24))
  }

  const dateFin = getFechaUnix()
  const dateInicio = dateFin - 7

  let form = $state({ dateFin, dateInicio, CajaID: 0 })
  let cajaMovimientos = $state([] as ICashBankMovement[])
  let filterText = $state("")

  $effect(() => {
    if(cajas?.Cajas?.length > 0){
      untrack(() => {
        const CajaID = (cajas.Cajas || [])[0]?.ID
        form = { ...form, CajaID }
      })
    }
  })

  const consultarRegistros = async () => {
    if (!form.CajaID || !form.dateInicio || !form.dateFin) {
      Notify.failure(tr("Please select a cash register and a date range.|Debe seleccionar una caja y un rango de dates."))
      return
    }

    Loading.standard(tr("Querying records...|Consultando registros..."))
    let result: ICashBankMovement[]
    try {
      result = await getCajaMovimientos(form)
    } catch (error) {
      Loading.remove()
      return
    }

    Loading.remove()
    cajaMovimientos = result || []
    console.log("movimientos obtenidos: ", result)
  }

  const columns: ITableColumn<ICashBankMovement>[] = [
    {
      header: "Date & Time|Fecha Hora",
      headerCss: "w-140",
      css: "ff-mono px-6",
      getValue: e => formatTime(e.Created, "d-M h:n") as string,
      mobile: { order: 1, css: "col-span-12 ff-mono", icon: "[fa--clock-o]" }
    },
    {
      header: "Movement Type|Tipo Mov.",
      headerCss: "w-160",
      css: "px-6",
      getValue: e => cajaMovimientoTiposMap.get(e.Type)?.name || "",
      mobile: { order: 2, css: "col-span-12", contentCss: "text-right" }
    },
    {
      header: "Amount|Monto",
      headerCss: "w-120",
      css: "ff-mono text-right px-6",
      render: e => {
        const cssClass = e.Amount < 0 ? "text-red-500" : ""
        return `<span class="${cssClass}">${formatN(e.Amount / 100, 2)}</span>`
      },
      mobile: { order: 3, css: "col-span-12", labelLeft: "Monto:", contentCss: "ff-mono ff-bold" }
    },
    {
      header: "Final Balance|Saldo Final",
      headerCss: "w-120",
      css: "ff-mono text-right px-6",
      getValue: e => formatN(e.FinalAmount / 100, 2) as string,
      mobile: { order: 4, css: "col-span-12", labelLeft: "Saldo:", contentCss: "ff-mono" }
    },
    {
      header: "Document #|Nº Documento",
      headerCss: "w-140",
      css: "text-center px-6",
      getValue: e => e.DocumentID ? String(e.DocumentID) : "",
      mobile: { order: 5, css: "col-span-12", labelLeft: "Doc:", if: (e) => !!e.DocumentID }
    },
    {
      // id triggers cellRenderer snippet so we can mount RecordByIDText per row.
      id: "movimientoUsuario",
      header: "User|Usuario",
      headerCss: "w-120",
      css: "text-center px-6",
      getValue: e => e.CreatedBy,
      mobile: { order: 6, css: "col-span-12", labelLeft: "Usuario:" }
    }
  ]
</script>

<Page title="Cash Movements|Cajas Movimientos">
  <div class="flex flex-col gap-8 mb-12 md:flex-row md:items-center md:justify-between" aria-label="Cash movements search filter with cash register, date range, and search">
    <!-- The four filters exceed a phone's width in one row, so they wrap into a grid below md. -->
    <div class="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-end gap-x-8 w-full md:flex md:items-center" style="max-width: 64rem;">
      <SearchSelect
        bind:saveOn={form}
        save="CajaID"
        css="col-span-3 w-full md:w-240 md:mr-12"
        label="Cash & Banks|Cajas & Bancos"
        keyId="ID"
        keyName="Name"
        options={cajas?.Cajas || []}
        placeholder=""
        required={true}
      />
      <DateInput
        label="Start Date|Fecha Inicio"
        css="w-full md:w-140 md:mr-12"
        save="dateInicio"
        bind:saveOn={form}
      />
      <DateInput
        label="End Date|Fecha Fin"
        css="w-full md:w-140 md:mr-12"
        save="dateFin"
        bind:saveOn={form}
      />
      <button class="px-16 py-8 bx-blue mt-8 h-44"
        aria-label="Query records"
        onclick={ev => {
          ev.stopPropagation()
          consultarRegistros()
        }}
      >
        <i class="icon-[fa--search]"></i>
      </button>
    </div>
    <div class="flex items-center w-full relative md:mr-16 md:w-224 md:ml-auto">
      <div class="absolute left-12 text-gray-400">
        <i class="icon-[fa--search]"></i>
      </div>
      <input
        class="w-full pl-36 pr-12 py-8 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
        autocomplete="off"
        type="text"
        placeholder={tr("Search...|Buscar...")}
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
      const movTipo = cajaMovimientoTiposMap.get(e.Type)?.name || ""
      return movTipo.toLowerCase()
    }}
    useFilterCache={true}
  >
    {#snippet cellRenderer(record, column)}
      {#if column.id === 'movimientoUsuario'}
        <RecordByIDText apiRoute="users-ids" recordID={record.CreatedBy} placeholder="" />
      {/if}
    {/snippet}
  </VTable>
</Page>
