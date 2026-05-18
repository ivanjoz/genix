<script lang="ts">
import Input from '$components/form/Input.svelte';
import LayerStatic from '$components/layers/LayerStatic.svelte';
import Modal from '$components/layers/Modal.svelte';
import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
import Page from '$domain/Page.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import RecordByIDText from '$components/misc/RecordByIDText.svelte';
import { Loading, Notify, formatTime } from '$libs/helpers';
import FilterInput from '$components/form/FilterInput.svelte';
import Button from '$components/buttons/Button.svelte';
import { Core } from '$core/store.svelte';
import { formatN } from '$libs/helpers';
  import { AlmacenesService } from "../../negocio/sedes-almacenes/sedes-almacenes.svelte"
  import CajaForm from "./CajaForm.svelte"
  import {
    CajasService,
    getCajaCuadres,
    getCajaMovimientos,
    postCaja,
    postCajaCuadre,
    postCajaMovimiento,
    cajaTipos,
    cajaMovimientoTipos,
    type ICashBank,
    type ICashReconciliation,
    type ICashBankMovement,
  } from "./cajas.svelte"

  const cajaMovimientoTiposMap = new Map(cajaMovimientoTipos.map(x => [x.id, x]))

  const almacenes = new AlmacenesService()
  const cajas = new CajasService()

  let filterText = $state("")
  let layerView = $state(1)
  let cajaForm = $state({} as ICashBank)
  let cajaCuadreForm = $state({} as ICashReconciliation)
  let cajaMovimientoForm = $state({} as ICashBankMovement)
  let cajaMovimientos = $state([] as ICashBankMovement[])
  let cajaCuadres = $state([] as ICashReconciliation[])
  let isLoadingMovimientos = $state(false)
  let isLoadingCuadres = $state(false)

  const columns: ITableColumn<ICashBank>[] = [
    {
      header: "ID",
      headerCss: "w-32",
      css: "text-center text-purple-600 px-6",
      getValue: e => e.ID
    },
    {
      header: "Nombre", css: "px-6 py-2",
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
        const date = formatTime(e.CuadreFecha, "d-M h:n")
        return `<div class="leading-tight text-right">
          <div class="ff-mono text-[0.875rem]">${saldo}</div>
          <div class="text-[0.875rem] text-slate-500">${date}</div>
        </div>`
      }
    },
    {
      header: "Saldo",
      css: "text-right ff-mono px-6",
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
    let recordSaved: ICashReconciliation & { NeedUpdateSaldo: number }
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
    let movimientoSaved: ICashBankMovement
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
  <div class="flex h-full gap-20">
    <div class="flex-1 flex flex-col min-w-0 relative">
      <div class="flex justify-between items-center w-full mb-10" aria-label="Cash registers toolbar with filter and create button">
        <FilterInput bind:value={filterText} css="mr-16 w-256" />
        <div class="flex items-center">
          <Button color="green" icon="icon-plus" label="Opens the modal to create a new cash register (caja)." onClick={ev => {
            cajaForm = { ID: -1, ss: 1 } as ICashBank
            Core.openModal(1)
          }} />
        </div>
      </div>
      <VTable css="w-full" columns={columns}
        maxHeight="calc(100vh - 8rem - 16px)"
        data={filteredCajas}
        selected={cajaForm.ID}
        isSelected={(e, id) => e?.ID === id as number}
        tableCss="cursor-pointer"
        onRowClick={(record) => {
          const el = cajaForm.ID === record.ID ? {} as ICashBank : { ...record }
          cajaForm = el
        }}
      />
    </div>
    <LayerStatic
      css="w-[64%] min-w-350 bg-white border-l border-gray-200 flex flex-col h-[calc(100vh-var(--header-height))] shadow-lg md:-m-10 overflow-hidden"
      mobileLayerTitle="Detalle de Caja"
    >
      <div class="px-12 pt-12">
        <OptionsStrip selected={layerView}
          options={[[1, 'Movimientos'], [3, 'Config.'], [2, 'Cuadres']]}
          buttonCss="ff-bold"
          onSelect={e => {
            layerView = e[0] as number
          }}
        />
      </div>
      <div class="flex-1 min-h-0 flex flex-col px-12 pb-12">
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
            </div>
            <div class="flex items-center">
              <Button color="green" icon="icon-plus" label="Opens the modal to add a new cash movement." onClick={() => {
                Core.openModal(3)
                cajaMovimientoForm = {
                  CajaID: cajaForm.ID, SaldoFinal: cajaForm.SaldoCurrent,
                } as ICashBankMovement
              }} />
            </div>
          </div>
          <VTable css="w-full mt-8"
            maxHeight="calc(100vh - var(--header-height) - 140px)"
            data={cajaMovimientos}
            columns={[
              {
                header: "Date Hora",
                getValue: e => formatTime(e.Created, "d-M h:n") as string
              },
              {
                header: "Tipo Mov.",
                getValue: e => cajaMovimientoTiposMap.get(e.Tipo)?.name || ""
              },
              {
                header: "Monto",
                css: "ff-mono text-right px-6",
                render: e => {
                  const monto = formatN(e.Monto / 100, 2)
                  const color = e.Monto < 0 ? "text-red-600" : ""
                  return `<span class="${color}">${monto}</span>`
                }
              },
              {
                header: "Saldo Final",
                css: "ff-mono text-right px-6",
                getValue: e => formatN(e.SaldoFinal / 100, 2)
              },
              {
                header: "Nº Documento",
                css: "text-right px-6",
                getValue: e => e.DocumentoID ? String(e.DocumentoID) : ""
              },
              {
                // id triggers cellRenderer snippet so we can mount RecordByIDText per row.
                id: "movimientoUsuario",
                header: "Usuario",
                css: "text-center px-6",
                getValue: e => e.CreatedBy
              }
            ]}
          >
            {#snippet cellRenderer(record, column)}
              {#if column.id === 'movimientoUsuario'}
                <RecordByIDText apiRoute="usuarios-ids" recordID={record.CreatedBy} placeholder="" />
              {/if}
            {/snippet}
          </VTable>
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
              <Button color="green" icon="icon-plus" label="Opens the modal to add a new cash balance reconciliation." onClick={() => {
                Core.openModal(2)
                cajaCuadreForm = { CajaID: cajaForm.ID } as ICashReconciliation
              }} />
            </div>
          </div>
          <VTable css="w-full mt-8"
            maxHeight="calc(100vh - var(--header-height) - 140px)"
            data={cajaCuadres}
            columns={[
              {
                header: "Date Hora",
                getValue: e => formatTime(e.Created, "d-M h:n") as string
              },
              {
                header: "Saldo Sistema",
                css: 'ff-mono text-right px-6',
                getValue: e => formatN((e.SaldoSistema || 0) / 100, 2)
              },
              {
                header: "Diferencia",
                css: 'ff-mono text-right px-6',
                getValue: e => formatN((e.SaldoDiferencia || 0) / 100, 2)
              },
              {
                header: "Saldo Real",
                css: 'ff-mono text-right px-6',
                getValue: e => formatN((e.SaldoReal || 0) / 100, 2)
              },
              {
                // id triggers cellRenderer snippet so we can mount RecordByIDText per row.
                id: "cuadreUsuario",
                header: "Usuario",
                css: 'text-center px-6',
                getValue: e => e.CreatedBy
              }
            ]}
          >
            {#snippet cellRenderer(record, column)}
              {#if column.id === 'cuadreUsuario'}
                <RecordByIDText apiRoute="usuarios-ids" recordID={record.CreatedBy} placeholder="" />
              {/if}
            {/snippet}
          </VTable>
        {/if}
      {/if}
      {#if layerView === 3}
        {#if !cajaForm.ID}
          <div class="bg-red-100 text-red-700 p-8 mt-8 rounded">Seleccione una Caja</div>
        {:else}
          <div class="mt-8">
            <CajaForm bind:form={cajaForm} sedes={almacenes.Sedes} />
            <div class="flex justify-end mt-12">
              <Button color="blue" name="Guardar" icon="icon-floppy" label="Saves the cash register configuration changes." onClick={saveCaja} />
            </div>
          </div>
        {/if}
      {/if}
      </div>
    </LayerStatic>
  </div>

  <Modal id={1} title="Cajas" size={6} bodyCss="px-16 py-14"
    onSave={() => {
      saveCaja()
    }}
    onDelete={() => {
      // TODO: implement delete
    }}
  >
    <CajaForm bind:form={cajaForm} sedes={almacenes.Sedes} />
  </Modal>

  <Modal id={2} title="Cuadre de Caja" size={6}
    onSave={() => {
      saveCajaCuadre()
    }}
  >
    <div class="flex items-start w-full p-16" aria-label="Cash balance reconciliation form with system balance, found balance, and difference">
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
        <Button color="purple" icon="icon-arrows-cw" name="Recalcular" label="Recalculates the balance difference using the latest system saldo." css="w-full mt-24" />
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
    <div class="grid grid-cols-24 gap-10" aria-label="Cash movement form with type, destination cash register, and amount">
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
