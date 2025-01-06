import { Notify } from "~/core/main"
import { createEffect, createSignal, on } from "solid-js"
import { DatePicker } from "~/components/Datepicker"
import { QTable } from "~/components/QTable"
import { SearchSelect } from "~/components/SearchSelect"
import { Loading, formatTime, throttle } from "~/core/main"
import { PageContainer } from "~/core/page"
import { cajaMovimientoTipos } from "~/services/admin/shared"
import { ICajaMovimiento, getCajaMovimientos, useCajasAPI } from "~/services/operaciones/ventas"
import { arrayToMapN, formatN } from "~/shared/main"
import { Params } from "~/shared/security"

interface IForm {
  CajaID: number, fechaInicio: number, fechaFin: number
}

export default function CajasMovimientos() {
  const cajaMovimientoTiposMap = arrayToMapN(cajaMovimientoTipos,'id')
  const fechaFin = Params.getFechaUnix()
  const fechaInicio = fechaFin - 7
  const [form, setForm] = createSignal({ fechaFin, fechaInicio } as IForm)
  const [cajas] = useCajasAPI()

  const [cajaMovimientos, setCajaMovimientos] = createSignal([] as ICajaMovimiento[])
  const [filterText, setFilterText] = createSignal("")
  
  const consultarRegistros = async () => {
    const form_ = form()
    console.log("form::", form_)
    if(!form_.CajaID || !form_.fechaInicio || !form_.fechaFin){
      Notify.failure("Debe seleccionar una caja y un rango de fechas.")
      return
    }
    Loading.standard("Consultando registros...")
    let result: ICajaMovimiento[]
    try {
      result = await getCajaMovimientos(form_)
    } catch (error) {
      Notify.failure(error as string); return
    }
    Loading.remove()
    setCajaMovimientos(result)
    console.log("movimientos obtenidos: ", result)
  }

  createEffect(on(() => [cajas()], 
    () => {
      if((cajas()?.Cajas || []).length > 0){
        setForm({ ...form(), CajaID: cajas().Cajas[0].ID })
      }
    }
  ))

  return <PageContainer title="Almacén Stock">
    <div class="flex ai-center jc-between mb-06">
      <div class="flex ai-center w100-10" style={{ "max-width": "64rem" }}>
        <SearchSelect saveOn={form()} save="CajaID" css="w-08x mb-02 mr-12"
          label="Cajas & Bancos" keys="ID.Nombre" options={cajas()?.Cajas || []}
          placeholder="" required={true}
          onChange={e => {
            
          }}
        />
        <DatePicker label="Fecha Inicio" css="w-045x" 
          save="fechaInicio" saveOn={form()} />
        <DatePicker label="Fecha Fin" css="w-045x" save="fechaFin" 
          saveOn={form()}/>
        <button class="bn1 b-blue ml-10" onClick={ev => {
          ev.stopPropagation()
          consultarRegistros()
        }}>
          <i class="icon-search"></i>
        </button>
      
      </div>
      <div class="search-c4 mr-16 w14rem ml-auto">
        <div><i class="icon-search"></i></div>
        <input class="w100" autocomplete="off" type="text" onKeyUp={ev => {
          ev.stopPropagation()
          throttle(() => {
            setFilterText(((ev.target as any).value||"").toLowerCase().trim())
          },150)
        }}/>
      </div>
    </div>
    <QTable data={cajaMovimientos()} 
      css="w100 w-page-t" tableCss="w100"
      maxHeight="calc(100vh - 8rem - 12px)" 
      makeFilter={e => {
        const movTipo = cajaMovimientoTiposMap.get(e.Tipo)
        return [movTipo?.name||"", e.Usuario?.usuario].join(" ")
      }}
      filterText={filterText()}
      columns={[
        { header: "Fecha Hora",
          getValue: e => {
            return formatTime(e.Created,"d-M h:n") as string
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
  </PageContainer>
}