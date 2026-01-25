<script lang="ts">
import Input from '$ui/components/Input';
import Modal from '$ui/components/Modal';
import OptionsStrip from '$ui/components/micro/OptionsStrip';
import Page from '$ui/components/Page';
import SearchSelect from '$ui/components/SearchSelect';
import Checkbox from '$ui/components/Checkbox';
import VTable from '$ui/components/VTable/index';
  import type { ITableColumn } from "$ui/VTable/types.ts"
import { Loading, Notify, formatTime } from '$core/lib/helpers';
import { throttle } from '$core/lib/helpers';
import { Core } from '$core/core/store.svelte';
import { formatN } from '$core/lib/helpers';
  import { AlmacenesService } from "../sedes-almacenes/sedes-almacenes.svelte"
  import {
    CajasService,
    getCajaCuadres,
    getCajaMovimientos,
    postCaja,
    postCajaCuadre,
    postCajaMovimiento,
    cajaTipos,
    cajaMovimientoTipos,
    type ICaja,
    type ICajaCuadre,
    type ICajaMovimiento,
  } from "./cajas.svelte"

  const cajaMovimientoTiposMap = new Map(cajaMovimientoTipos.map(x => [x.id, x]))

  const almacenes = new AlmacenesService()
  const cajas = new CajasService()

  let filterText = $state("")
  let layerView = $state(1)
  let cajaForm = $state({} as ICaja)
  let cajaCuadreForm = $state({} as ICajaCuadre)
  let cajaMovimientoForm = $state({} as ICajaMovimiento)
  let cajaMovimientos = $state([] as ICajaMovimiento[])
  let cajaCuadres = $state([] as ICajaCuadre[])
  let isLoadingMovimientos = $state(false)
  let isLoadingCuadres = $state(false)

  const columns: ITableColumn<ICaja>[] = [
    {
      header: "ID",
      headerCss: "w-32",
      cellCss: "text-center text-purple-600 px-6",
      getValue: e => e.ID
    },
    {
      header: "Nombre", cellCss: "px-6 py-2",
      getValue: e => e.Nombre,
      render: e => {
        const tipoName = cajaTipos.find(x => x.id === e.Tipo)?.name || "-"
        return `<div class="leading-tight">
          <div class="ff-bold leading-[1.1] h3">${e.Nombre}</div>
          <div class="fs15 text-slate-500">${tipoName}</div>
        </div>`
      }
    },
    {
      header: "Cuadre",
      getValue: e => e.CuadreFecha ? String(e.CuadreFecha) : "",
      render: e => {
        if (!e.CuadreFecha) { return "" }
        const saldo = formatN(e.CuadreSaldo / 100, 2)
        const fecha = formatTime(e.CuadreFecha, "d-M h:n")
        return `<div class="leading-tight text-right">
          <div class="ff-mono text-[0.875rem]">${saldo}</div>
          <div class="text-[0.875rem] text-slate-500">${fecha}</div>
        </div>`
      }
    },
    {
      header: "Saldo",
      cellCss: "text-right ff-mono px-6",
      getValue: e => formatN(e.SaldoCurrent / 100, 2)
    },
  ]

  const saveCaja = async () => {
    const caja = cajaForm
    if (!caja.Nombre || !caja.Tipo || !caja.SedeID) {
      Notify.failure("Los inputs Nombre, Tipo y Sede son obligatorios")
      return
    }
    Loading.standard("Guardando caja...")
    try {
      var result = await postCaja(caja)
    } catch (error) {
      console.warn(error)
      return
    }
    Loading.remove()

    caja.SaldoCurrent = caja.SaldoCurrent || 0

    const selected = cajas.CajasMap.get(caja.ID)
    if (selected) {
      Object.assign(selected, caja)
    } else {
      caja.ID = result.ID
      cajas.Cajas.push(caja)
    }
    cajas.Cajas = [...cajas.Cajas]
    Core.closeModal(1)
  }

  const saveCajaCuadre = async () => {
    const form = cajaCuadreForm
    form.SaldoSistema = cajaForm.SaldoCurrent

    Loading.standard("Guardando caja...")
    let recordSaved: ICajaCuadre & { NeedUpdateSaldo: number }
    try {
      recordSaved = await postCajaCuadre(form)
    } catch (error) {
      console.warn(error)
      return
    }
    Loading.remove()
    const caja = cajas.CajasMap.get(form.CajaID)
    if (!caja) return
    
    if (typeof recordSaved?.NeedUpdateSaldo === 'number') {
      caja.SaldoCurrent = recordSaved.NeedUpdateSaldo
      cajaForm = { ...caja }
      const newForm = { ...cajaCuadreForm }
      newForm._error = `Hubo una actualización en el saldo de la caja. El saldo actual es "${formatN(caja.SaldoCurrent / 100, 2)}". Intente nuevamente con el cálculo actualizado.`
      newForm.SaldoDiferencia = newForm.SaldoReal - caja.SaldoCurrent
      cajaCuadreForm = newForm
    } else {
      caja.SaldoCurrent = form.SaldoReal
      cajas.Cajas = [...cajas.Cajas]
      Object.assign(cajaForm, caja)
      Core.closeModal(2)
      cajaCuadres.unshift(recordSaved)
    }
  }

  const saveCajaMovimiento = async () => {
    const form = cajaMovimientoForm
    if (!form.Tipo || !form.Monto) {
      Notify.failure("Se necesita seleccionar un monto y un tipo.")
      return
    }
    Loading.standard("Guardando Movimiento...")
    let movimientoSaved: ICajaMovimiento
    try {
      movimientoSaved = await postCajaMovimiento(form)
    } catch (error) {
      console.warn(error)
      return
    }
    Loading.remove()

    const caja = cajas.CajasMap.get(form.CajaID)
    if (!caja) return
    
    caja.SaldoCurrent = form.SaldoFinal
    cajas.Cajas = [...cajas.Cajas]
    Object.assign(cajaForm, caja)

    cajaMovimientos.unshift(movimientoSaved)
    Core.closeModal(3)
  }

  const isCajaMovimiento = $derived([3].includes(cajaMovimientoForm.Tipo))

  // Load cajaMovimientos when cajaForm changes
  $effect(() => {
    if (layerView === 1 && cajaForm.ID) {
      isLoadingMovimientos = true
      getCajaMovimientos({ CajaID: cajaForm.ID, lastRegistros: 200 })
        .then(result => {
          cajaMovimientos = result
        })
        .catch(error => {
          Notify.failure(error as string)
        })
        .finally(() => {
          isLoadingMovimientos = false
        })
    }
  })

  // Load cajaCuadres when cajaForm changes
  $effect(() => {
    if (layerView === 2 && cajaForm.ID) {
      isLoadingCuadres = true
      getCajaCuadres({ CajaID: cajaForm.ID, lastRegistros: 200 })
        .then(result => {
          cajaCuadres = result
        })
        .catch(error => {
          console.log("Error:", error)
          Notify.failure(error as string)
        })
        .finally(() => {
          isLoadingCuadres = false
        })
    }
  })

  const filteredCajas = $derived.by(() => {
    if (!filterText) return cajas.Cajas
    const text = filterText.toLowerCase()
    return cajas.Cajas.filter(e => {
      return e.Nombre?.toLowerCase().includes(text)
    })
  })
</script>

<Page title="Cajas & Bancos">
  <div class="flex flex-col md:flex-row justify-between mb-6 gap-10">
    <div class="w-full md:w-[36%]">
      <div class="flex justify-between items-center w-full mb-10">
        <div class="i-search mr-16 w-256">
          <div><i class="icon-search"></i></div>
          <input class="w-full" autocomplete="off" type="text" onkeyup={ev => {
            ev.stopPropagation()
            throttle(() => {
              filterText = ((ev.target as any).value || "").toLowerCase().trim()
            }, 150)
          }} />
        </div>
        <div class="flex items-center">
          <button class="bx-green" aria-label="agregar"  onclick={ev => {
            ev.stopPropagation()
            cajaForm = { ID: -1, ss: 1 } as ICaja
            Core.openModal(1)
          }}>
            <i class="icon-plus"></i>
          </button>
        </div>
      </div>
      <VTable css="w-full" columns={columns}
        maxHeight="calc(100vh - 8rem - 16px)"
        data={filteredCajas}
        selected={cajaForm.ID}
        isSelected={(e, id) => e?.ID === id as number}
        tableCss="cursor-pointer"
        onRowClick={(record) => {
          const el = cajaForm.ID === record.ID ? {} as ICaja : { ...record }
          cajaForm = el
        }}
      />
    </div>
    <div class="w-full md:w-[calc(64%-22px)] bg-white rounded-md shadow-sm p-12">
      <OptionsStrip selected={layerView}
        options={[[1, 'Movimientos'], [2, 'Cuadres']]}
        buttonCss="ff-bold"
        onSelect={e => {
          layerView = e[0] as number
        }}
      />
      {#if layerView === 1}
        {#if isLoadingMovimientos}
          <div class="flex justify-center items-center py-40">
            <div class="text-slate-400">Cargando...</div>
          </div>
        {:else if !cajaForm.ID}
          <div class="bg-red-100 text-red-700 p-8 mt-8 rounded">Seleccione una Caja</div>
        {:else}
          <div class="flex w-full justify-between mt-8">
            <div class="flex items-center">
              <div class="text-[1.1rem] ff-bold mr-8">{cajaForm?.Nombre || ""}</div>
              <button class="bx-yellow" aria-label="edit" onclick={ev => {
                ev.stopPropagation()
                Core.openModal(1)
              }}>
                <i class="icon-pencil"></i>
              </button>
            </div>
            <div class="flex items-center">
              <button class="bx-green" aria-label="add" onclick={ev => {
                ev.stopPropagation()
                Core.openModal(3)
                cajaMovimientoForm = {
                  CajaID: cajaForm.ID, SaldoFinal: cajaForm.SaldoCurrent,
                } as ICajaMovimiento
              }}>
                <i class="icon-plus"></i>
              </button>
            </div>
          </div>
          <VTable css="w-full mt-8"
            maxHeight="calc(100vh - 8rem - 16px)"
            data={cajaMovimientos}
            columns={[
              {
                header: "Fecha Hora",
                getValue: e => formatTime(e.Created, "d-M h:n") as string
              },
              {
                header: "Tipo Mov.",
                getValue: e => cajaMovimientoTiposMap.get(e.Tipo)?.name || ""
              },
              {
                header: "Monto",
                cellCss: "ff-mono text-right px-6",
                render: e => {
                  const monto = formatN(e.Monto / 100, 2)
                  const color = e.Monto < 0 ? "text-red-600" : ""
                  return `<span class="${color}">${monto}</span>`
                }
              },
              {
                header: "Saldo Final",
                cellCss: "ff-mono text-right px-6",
                getValue: e => formatN(e.SaldoFinal / 100, 2)
              },
              {
                header: "Nº Documento",
                getValue: e => ""
              },
              {
                header: "Usuario",
                cellCss: "text-center px-6",
                getValue: e => e.Usuario?.usuario || ""
              }
            ]}
          />
        {/if}
      {/if}
      {#if layerView === 2}
        {#if isLoadingCuadres}
          <div class="flex justify-center items-center py-40">
            <div class="text-slate-400">Cargando...</div>
          </div>
        {:else if !cajaForm.ID}
          <div class="bg-red-100 text-red-700 p-8 mt-8 rounded">Seleccione una Caja</div>
        {:else}
          <div class="flex w-full justify-between mt-8">
            <div></div>
            <div class="flex items-center">
              <button class="bx-green" aria-label="add" onclick={ev => {
                ev.stopPropagation()
                Core.openModal(2)
                cajaCuadreForm = { CajaID: cajaForm.ID } as ICajaCuadre
              }}>
                <i class="icon-plus"></i>
              </button>
            </div>
          </div>
          <VTable css="w-full mt-8"
            maxHeight="calc(100vh - 8rem - 16px)"
            data={cajaCuadres}
            columns={[
              {
                header: "Fecha Hora",
                getValue: e => formatTime(e.Created, "d-M h:n") as string
              },
              {
                header: "Saldo Sistema",
                cellCss: 'ff-mono text-right px-6',
                getValue: e => formatN((e.SaldoSistema || 0) / 100, 2)
              },
              {
                header: "Diferencia",
                cellCss: 'ff-mono text-right px-6',
                getValue: e => formatN((e.SaldoDiferencia || 0) / 100, 2)
              },
              {
                header: "Saldo Real",
                cellCss: 'ff-mono text-right px-6',
                getValue: e => formatN((e.SaldoReal || 0) / 100, 2)
              },
              {
                header: "Usuario",
                cellCss: 'text-center px-6',
                getValue: e => e.Usuario?.usuario || ""
              }
            ]}
          />
        {/if}
      {/if}
    </div>
  </div>

  <Modal id={1} title="Cajas" size={6} bodyCss="px-16 py-14"
    onSave={() => {
      saveCaja()
    }}
    onDelete={() => {
      // TODO: implement delete
    }}
  >
    <div class="grid grid-cols-24 gap-10">
      <SearchSelect bind:saveOn={cajaForm} save="Tipo" css="col-span-24 md:col-span-10"
        label="Tipo" keyId="id" keyName="name" options={cajaTipos}
        placeholder="" required={true}
      />
      <Input bind:saveOn={cajaForm} save="Nombre"
        css="col-span-24 md:col-span-14" label="Nombre" required={true}
      />
      <Input bind:saveOn={cajaForm} save="Descripcion"
        css="col-span-24" label="Descripcion"
      />
      <SearchSelect bind:saveOn={cajaForm} save="SedeID"
        css="col-span-24 md:col-span-10" label="Sede" required={true} options={almacenes.Sedes}
        keyId="ID" keyName="Nombre"
      />
      <div class="col-span-24 flex justify-between items-center">
        <div></div>
        <Checkbox label="Saldo Negativo" bind:saveOn={cajaForm} save="Nombre" />
      </div>
    </div>
  </Modal>

  <Modal id={2} title="Cuadre de Caja" size={6}
    onSave={() => {
      saveCajaCuadre()
    }}
  >
    <div class="flex items-start w-full p-16">
      <div class="w-260 mr-16">
        <div class="w-full mb-12">
          <div class="text-sm mb-4 text-slate-600">Saldo Sistema</div>
          <div class="text-[1.1rem] ff-mono text-center bg-slate-100 py-8 rounded">
            {formatN((cajaForm.SaldoCurrent||0) / 100, 2)}
          </div>
        </div>
        <Input bind:saveOn={cajaCuadreForm} save="SaldoReal" type="number"
          inputCss="text-[1.1rem] ff-mono text-center" baseDecimals={2}
          css="w-full mb-12" label="Saldo Encontrado" required={true}
          onChange={() => {
            console.log("caja cuadre::", cajaCuadreForm)
            cajaCuadreForm = { ...cajaCuadreForm }
          }}
        />
        <div class="mb-12"></div>
        <div class="w-full">
          <div class="text-sm mb-4 text-slate-600">Diferencia</div>
          <div class="text-[1.1rem] min-h-32 ff-mono text-center bg-slate-100 py-8 rounded">
            {#if cajaCuadreForm.SaldoReal}
              {@const diff = (cajaCuadreForm.SaldoReal || 0) - cajaForm.SaldoCurrent}
              {#if diff}
                <span class="{diff > 0 ? 'text-blue-600' : 'text-red-600'}">
                  {formatN(diff / 100, 2)}
                </span>
              {/if}
            {/if}
          </div>
        </div>
      </div>
      
      <div class="flex items-end">
        <button class="bx-purple w-full mt-24">
          <i class="text-[0.875rem] icon-arrows-cw"></i>
          Recalcular
        </button>
      </div>
      {#if cajaCuadreForm._error}
        <div class="col-span-24 text-red-600 ff-bold">
          <i class="icon-attention"></i>{cajaCuadreForm._error}
        </div>
      {/if}
    </div>
  </Modal>

  <Modal id={3} title="Movimiento de Caja" size={6}
    onSave={() => {
      saveCajaMovimiento()
    }}
  >
    <div class="grid grid-cols-24 gap-10">
      <SearchSelect bind:saveOn={cajaMovimientoForm} save="Tipo" css="col-span-24 md:col-span-12"
        label="Tipo" keyId="id" keyName="name"
        options={cajaMovimientoTipos.filter(x => x.group === 2)}
        placeholder="" required={true}
        onChange={() => {
          cajaMovimientoForm.CajaRefID = 0
          cajaMovimientoForm = { ...cajaMovimientoForm }
        }}
      />
      <SearchSelect bind:saveOn={cajaMovimientoForm} save="CajaRefID" css="col-span-24 md:col-span-12"
        label="Caja Destino" keyId="id" keyName="name" options={cajaTipos}
        disabled={!isCajaMovimiento}
        placeholder={isCajaMovimiento ? "seleccione" : "no aplica"}
        required={true}
      />
      <Input bind:saveOn={cajaMovimientoForm} save="Monto" inputCss="ff-mono text-[1.1rem] text-center"
        css="col-span-24 md:col-span-12" label="Monto" baseDecimals={2}
        required={true} type="number"
        transform={v => {
          const movTipo = cajaMovimientoTiposMap.get(cajaMovimientoForm.Tipo)
          console.log("movimiento tipo::", movTipo)
          if (movTipo?.isNegative && typeof v === 'number' && v > 0) { v = v * -1 }
          return v
        }}
        onChange={() => {
          const form = { ...cajaMovimientoForm }
          form.SaldoFinal = cajaForm.SaldoCurrent + (form.Monto || 0)
          cajaMovimientoForm = form
        }}
      />
      <div class="col-span-24 md:col-span-12"></div>
      <div class="col-span-24 md:col-span-12">
        <div class="text-sm mb-4 text-slate-600">Saldo Inicial</div>
        <div class="text-[1.1rem] ff-mono text-center bg-slate-100 py-8 rounded">
          {formatN(cajaForm.SaldoCurrent / 100, 2)}
        </div>
      </div>
      <div class="col-span-24 md:col-span-12">
        <div class="text-sm mb-4 text-slate-600">Saldo Final</div>
        <div class="text-[1.1rem] ff-mono text-center bg-slate-100 py-8 rounded">
          {#if cajaMovimientoForm.SaldoFinal !== undefined}
            {@const saldo = cajaMovimientoForm.SaldoFinal}
            <span class="{saldo >= 0 ? '' : 'text-red-600'}">
              {formatN(saldo / 100, 2)}
            </span>
          {/if}
        </div>
      </div>
    </div>
  </Modal>
</Page>

