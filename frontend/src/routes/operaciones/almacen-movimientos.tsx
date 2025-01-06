import { createEffect, createSignal } from "solid-js"
import { DatePicker } from "~/components/Datepicker"
import { QTable } from "~/components/QTable"
import { SearchSelect, makeHighlString } from "~/components/SearchSelect"
import { Loading, formatTime, throttle } from "~/core/main"
import { PageContainer } from "~/core/page"
import { IUsuario } from "~/services/admin/empresas"
import { IAlmacenMovimiento, IAlmacenMovimientosResult, IProducto, movimientoTipos, queryAlmacenMovimientos, useProductosAPI } from "~/services/operaciones/productos"
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes"
import { arrayToMapN } from "~/shared/main"
import { Params } from "~/shared/security"

export default function AlmacenMovimientos() {

  const [productos] = useProductosAPI()
  const [almacenes] = useSedesAlmacenesAPI()

  const fechaFin = Params.getFechaUnix()
  const fechaInicio = fechaFin - 7
  const [form, setForm] = createSignal({ fechaFin, fechaInicio, almacenID: 0 })
  const [almacenMovimientos, setAlmacenMovimientos] = createSignal([] as IAlmacenMovimiento[])

  createEffect(() => {
    const almacenID = (almacenes().Almacenes||[])[0]?.ID
    if(almacenID && almacenID !== form().almacenID){ 
      setForm({ ...form(), almacenID }) 
    }
  })
  
  const usuariosMap: Map<number,IUsuario> = new Map()
  const productosMap: Map<number,IProducto> = new Map()
  const movimientoTiposMap = arrayToMapN(movimientoTipos,'id')

  const consultarRegistros = async () => {
    Loading.standard("Consultando registros...")
    let result: IAlmacenMovimientosResult
    try {
      result = await queryAlmacenMovimientos(form())
    } catch (error) {
      Loading.remove(); return
    }

    console.log("registros obtenidos: ", result)

    for(let e of result.Productos){ productosMap.set(e.ID,e) }
    for(let e of result.Usuarios){ usuariosMap.set(e.id,e) }

    setAlmacenMovimientos(result.Movimientos)
    Loading.remove()
  }

  const [filterText, setFilterText] = createSignal("")

  const almacenRender = (almacenID: number, cant: number) => {
    if(!almacenID){ return "" }
    const name = almacenes().AlmacenesMap.get(almacenID)?.Nombre || `Almacen-${almacenID}`
    return <div class="flex a-center">
      <div class="mr-08">{name}</div>
      <div class="ff-mono c-blue">(</div>
      <div class="ff-mono">{cant}</div>
      <div class="ff-mono c-blue">)</div>
    </div>
  }

  return <PageContainer title="Almacén Stock">
    <div class="flex ai-center jc-between mb-06">
      <div class="flex ai-center w100-10" style={{ "max-width": "64rem" }}>
        <SearchSelect saveOn={form()} save="almacenID" css="w-08x mb-02 mr-12"
          label="Almacén" keys="ID.Nombre" options={almacenes()?.Almacenes || []}
          placeholder="" required={true}
          onChange={e => {
            
          }}
        />
        <DatePicker label="Fecha Inicio" css="w-045x" save="fechaInicio" saveOn={form()} />
        <DatePicker label="Fecha Fin" css="w-045x" save="fechaFin" saveOn={form()}/>
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
    <QTable data={almacenMovimientos()} 
      css="w100 w-page-t" tableCss="w100"
      maxHeight="calc(100vh - 8rem - 12px)" 
      makeFilter={e => {
        const producto = productosMap.get(e.ProductoID)?.Nombre || ""
        const usuario = usuariosMap.get(e.CreatedBy||1)?.usuario || ""
        return [producto, usuario].join(" ")
      }}
      filterText={filterText()}
      columns={[
        { header: "Fecha Hora", css: "ff-mono",
          getValue: e => formatTime(e.Created,"d-M h:n") as string
        },
        { header: "Producto", css: "",
            render: e => {
            const nombre = productosMap.get(e.ProductoID)?.Nombre || `Producto-${e.ProductoID}`
            return makeHighlString(nombre, filterText())
          }
        },
        { header: "Lote", css: "c-purple t-c",
          getValue: e => e.Lote
        },
        { header: "SKU", css: "c-purple t-c",
          getValue: e => e.SKU
        },
        { header: "Movimiento", css: "t-c",
          render: e => {
            const mov = movimientoTiposMap.get(e.Tipo)
            return mov?.name || "-"
          }
        },
        { header: "Cantidad", css: "t-r ff-mono",
          render: e => {
            return <div class={"flex jc-end " + (e.Cantidad < 0 ? "c-red" : "c-blue")}>
              {e.Cantidad}
            </div>
          }
        },
        { header: "Almacén Origen", css: "",
          render: e => almacenRender(e.AlmacenOrigenID, e.AlmacenOrigenCantidad)
        },
        { header: "Almacén Destino", css: "",
          render: e => almacenRender(e.AlmacenID, e.AlmacenCantidad)
        },
        { header: "Usuario", cardColumn: [3,2], cardCss: "h5 c-steel", css: "t-c",
          render: e => {
            const usuario = usuariosMap.get(e.CreatedBy||1)?.usuario || `Usuario-${e.CreatedBy}`
            return makeHighlString(usuario, filterText())
          }
        },
      ]}
    />
  </PageContainer>
}