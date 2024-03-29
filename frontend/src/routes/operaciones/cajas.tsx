import { Show, createSignal } from "solid-js";
import { deviceType } from "~/app";
import { BarOptions, LayerLoading } from "~/components/Cards";
import { CheckBox, Input, InputDisabled } from "~/components/Input";
import { Modal, setOpenModals } from "~/components/Modals";
import { ITableColumn, QTable } from "~/components/QTable";
import { SearchSelect } from "~/components/SearchSelect";
import { Loading, Notify, formatTime, throttle } from "~/core/main";
import { PageContainer } from "~/core/page";
import { cajaMovimientoTipos, cajaTipos } from "~/services/admin/shared";
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes";
import { ICaja, ICajaCuadre, ICajaMovimiento, getCajaMovimientos, postCaja, postCajaCuadre, useCajasAPI } from "~/services/operaciones/ventas";
import { arrayToMapN, formatN } from "~/shared/main";

export default function Cajas() {
  const cajaMovimientoTiposMap = arrayToMapN(cajaMovimientoTipos,'id')

  const [almacenes] = useSedesAlmacenesAPI()
  
  const [filterText, setFilterText] = createSignal("")
  const [layerView, setLayerView] = createSignal(1)
  const [cajaForm, setCajaForm] = createSignal({} as ICaja)
  const [cajaCuadreForm, setCajaCuadreForm] = createSignal({} as ICajaCuadre)
  const [cajaMovimientos, setCajaMovimientos] = createSignal([] as ICajaMovimiento[])
  const [cajas, setCajas] = useCajasAPI()
  
  const columns: ITableColumn<ICaja>[] = [
    { header: "ID", headerStyle: { width: '2.6rem' }, css: 't-c c-purple2',
      getValue: e => e.ID
    },
    { header: "Nombre", field: "Nombre", 
      cellStyle: { "padding-top": '2px', "padding-bottom": '3px' },
      render: e => {
        return <div class="lh-10">
          <div class="ff-bold h3">{e.Nombre}</div>
          <div class="h5 c-steel">
            {cajaTipos.find(x => x.id === e.Tipo)?.name || "-"}
          </div>
        </div>
      }
    },
    { header: "Cuadre", field: "Nombre",
      render: e => {
        return ""
      }
    },
    { header: "Saldo", field: "Nombre", css: 't-c',
      render: e => {
        return <div>{formatN(e.SaldoCurrent/100,2) as string}</div>
      }
    },
  ]
  
  const saveCaja = async () => {
    const caja = cajaForm()
    if(!caja.Nombre || !caja.Tipo || !caja.SedeID){
      Notify.failure("Los inputs Nombre, Tipo y Sede son obligatorios")
      return
    }
    Loading.standard("Guardando caja...")
    try {
      await postCaja(caja)
    } catch (error) {
      console.warn(error)
      return
    }
    Loading.remove()
  }

  const saveCajaCuadre = async () => {
    const form = cajaCuadreForm()
    form.SaldoSistema = cajaForm().SaldoCurrent

    Loading.standard("Guardando caja...")
    let result: any
    try {
      result = await postCajaCuadre(form)
    } catch (error) {
      console.warn(error)
      return
    }
    Loading.remove()
    if(typeof result?.NeedUpdateSaldo === 'number'){
      const caja = cajas().CajasMap.get(form.CajaID)
      caja.SaldoCurrent = result.NeedUpdateSaldo
      setCajaForm({...caja})
      const newForm = {...cajaCuadreForm()}
      newForm._error = `Hubo una actualización en el saldo de la caja. El saldo actual es "${formatN(caja.SaldoCurrent/100),2}". Intente nuevamente con el cálculo actualizado.`
      newForm.SaldoDiferencia = newForm.SaldoReal - caja.SaldoCurrent
      setCajaCuadreForm(newForm)
    }
  }

  return <PageContainer title="Cajas & Bancos">
    <div class="flex jc-between mb-06"
      classList={{ "column": [2,3].includes(deviceType()) }}
    >
      <div class="" 
        style={{ width: deviceType() === 1 ? '36%' : '100%' }}
      >
        <div class="flex jc-between w100 mb-10">
          <div class="search-c4 mr-16 w16rem">
            <div><i class="icon-search"></i></div>
            <input class="w100" autocomplete="off" type="text" onKeyUp={ev => {
              ev.stopPropagation()
              throttle(() => {
                setFilterText(((ev.target as any).value||"").toLowerCase().trim())
              },150)
            }}/>
          </div>
          <div class="flex ai-center">
            <button class="bn1 b-green" onClick={ev => {
              ev.stopPropagation()
              setCajaForm({ ID: -1, ss: 1 } as ICaja)
              setOpenModals([1])
            }}>
              <i class="icon-plus"></i>
            </button> 
          </div>
        </div>
        <QTable css="selectable w100" columns={columns} 
          maxHeight="calc(100vh - 8rem - 16px)"
          styleMobile={{ height: '100vh' }}
          data={cajas()?.Cajas || []}
          selected={cajaForm().ID}
          isSelected={(e,id) => e?.ID === id as number}
          filterText={filterText()} filterKeys={["nombre"]}
          onRowCLick={e => {
            const el = cajaForm().ID === e.ID ? {} as ICaja : {...e}
            setCajaForm(el)
          }}
        />
      </div>
      <div style={{ width: deviceType() === 1 ? 'calc(62.5% + 1rem)' : '100%' }}
        class="card-c4"
      >
        <BarOptions selectedID={layerView()} class="w100"
          options={[[1,'Movimientos'],[2,'Cuadres']]}
          buttonStyle={{ "min-height": '2.1rem' }} buttonClass="ff-bold"
          onSelect={id => {
            setLayerView(id)
          }}
        />
        <Show when={layerView() === 1}>
          <LayerLoading baseObject={cajaForm()}
            startPromise={async (e) => {
              if(!e.ID){ return }
              let result 
              try {
                result = await getCajaMovimientos({ cajaID: e.ID, lastRegistros: 200 })
              } catch (error) {
                Notify.failure(error as string); return
              }
              setCajaMovimientos(result)
            }}
          >
            <Show when={!cajaForm().ID}>
              <div class="box-error-ms mt-08">Seleccione una Caja</div>
            </Show>
            <Show when={!!cajaForm().ID}>
              <div class="flex w100 jc-between mt-08">
                <div></div>
                <div class="flex ai-center">
                  <button class="bn1 b-yellow" onclick={ev =>{
                    ev.stopPropagation()
                    setOpenModals([1])
                  }}>
                    <i class="icon-pencil"></i>
                  </button>
                </div>
              </div>
              <QTable css="w100 mt-08"
                maxHeight="calc(100vh - 8rem - 16px)"
                styleMobile={{ height: '100vh' }}
                data={cajaMovimientos()}
                columns={[
                  { header: "Fecha Hora",
                    getValue: e => {
                      return formatTime(e.Created,"Y-m-d h:n") as string
                    }
                  },
                  { header: "Tipo Mov.",
                    getValue: e => {
                      return cajaMovimientoTiposMap.get(e.Tipo)?.name || ""
                    }
                  },
                  { header: "Monto", css: "ff-mono t-r",
                    render: e => {
                      return <span class={e.Monto < 0 ? "c-red" : ""}>
                        {formatN(e.Monto/100,2) as string}
                      </span>
                    }
                  },
                  { header: "Saldo Final", css: "ff-mono t-r",
                    getValue: e => {
                      return formatN(e.SaldoFinal/100,2) as string
                    }
                  },
                  { header: "Nº Documento",
                    getValue: e => {
                      return ""
                    }
                  },
                  { header: "Usuario", css: "t-c",
                    getValue: e => {
                      return e.Usuario?.usuario || ""
                    }
                  }
                ]} 
              />
            </Show>
          </LayerLoading>
        </Show>
        <Show when={layerView() === 2}>
          <LayerLoading baseObject={cajaForm()}
            startPromise={async (e) => {
              return
              /*
              if(!e.ID){ return }
              let result 
              try {
                result = await getCajaMovimientos({ cajaID: e.ID, lastRegistros: 200 })
              } catch (error) {
                Notify.failure(error as string); return
              }
              console.log("results::", result)
              */
            }}
          >
            <Show when={!cajaForm().ID}>
              <div class="box-error-ms mt-08">Seleccione una Caja</div>
            </Show>
            <Show when={!!cajaForm().ID}>
              <div class="flex w100 jc-between mt-08">
                <div></div>
                <div class="flex ai-center">
                  <button class="bn1 b-green" onClick={ev => {
                    ev.stopPropagation()
                    setOpenModals([2])
                    setCajaCuadreForm({ CajaID: cajaForm().ID } as ICajaCuadre)
                  }}>
                    <i class="icon-plus"></i>
                  </button> 
                </div>
              </div>
              <QTable css="w100 mt-08"
                maxHeight="calc(100vh - 8rem - 16px)"
                styleMobile={{ height: '100vh' }}
                data={[]}
                onRowCLick={e => {
                  const el = cajaForm().ID === e.ID ? {} as ICaja : {...e}
                  setCajaForm(el)
                }}
                columns={[
                  { header: "Fecha Hora",
                    getValue: e => {
                      return ""
                    }
                  },
                  { header: "Saldo Sistema",
                    getValue: e => {
                      return ""
                    }
                  },
                  { header: "Diferencia",
                    getValue: e => {
                      return ""
                    }
                  },
                  { header: "Saldo Real",
                    getValue: e => {
                      return ""
                    }
                  },
                  { header: "Usuario",
                    getValue: e => {
                      return ""
                    }
                  }
                ]} 
              />
            </Show>
          </LayerLoading>
        </Show>
      </div>
    </div>
    <Modal id={1} title="Cajas"
      onSave={() => {
        saveCaja()
      }}
      onDelete={() => {

      }}
    >
      <div class="w100-10 flex-wrap in-s2">        
        <SearchSelect saveOn={cajaForm()} save="Tipo" css="w-10x mb-10"
          label="Tipo" keys="id.name" options={cajaTipos}
          placeholder="" required={true}
        />
        <Input saveOn={cajaForm()} save="Nombre" 
          css="w-14x mb-10" label="Nombre"required={true}
        />
        <Input saveOn={cajaForm()} save="Descripcion"
          css="w-24x mb-10" label="Descripcion"
        />
        <SearchSelect saveOn={cajaForm()} save="SedeID" 
          css="w-10x mb-10" label="Sede"required={true} options={almacenes().Sedes}
          keys="ID.Nombre"
        />
        <div class="w100 flex jc-between ai-center">
          <div></div>
          <CheckBox label="Saldo Negativo" saveOn={cajaForm()} save="Nombre"/>
        </div>
      </div>
    </Modal>
    <Modal id={2} title="Cuadre de Caja"
      onSave={() => {
        saveCajaCuadre()
      }}
    >
      <div class="w100-10 flex-wrap in-s2">        
        <InputDisabled css="w-14x mb-10" label="Saldo Sistema" 
          inputCss="h3 ff-mono jc-center"
          content={formatN(cajaForm().SaldoCurrent/100,2)}
        />
        <div class="w-10x">
          <button class="bn1 b-purple" style={{ "margin-top": "0.9rem" }}>
            <i class="h5 icon-arrows-cw"></i>
            Recalcular
          </button>
        </div>
        <Input saveOn={cajaCuadreForm()} save="SaldoReal" type="number" 
          inputCss="h3 ff-mono t-c" baseDecimals={2}
          css="w-14x mb-10" label="Saldo Encontrado" required={true}
          onChange={() => {
            console.log("caja cuadre::",cajaCuadreForm())
            setCajaCuadreForm({...cajaCuadreForm()})
          }}
        />
        <div class="w-10x"></div>
        <InputDisabled css="w-14x mb-10" label="Diferencia" 
          inputCss="h3 ff-mono jc-center"
          getContent={() => {
            const diff = (cajaCuadreForm().SaldoReal||0) - cajaForm().SaldoCurrent
            if(!diff){ return "" }
            return <span class={diff > 0 ? "c-blue" : "c-red"}>
              {formatN(diff/100,2)}
            </span>
          }}
        />
        <Show when={cajaCuadreForm()._error}>
          <div class="w100 c-red ff-bold">
            <i class="icon-attention"></i>{cajaCuadreForm()._error}
          </div>
        </Show>
      </div>
    </Modal>
  </PageContainer>
}