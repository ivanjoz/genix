import { Notify } from "notiflix"
import { Show, createEffect, createSignal } from "solid-js"
import { DatePicker } from "~/components/Datepicker"
import { CellEditable } from "~/components/Editables"
import { CheckBox, Input } from "~/components/Input"
import { CornerLayer, setOpenLayers } from "~/components/Modals"
import { QTable } from "~/components/QTable"
import { SearchSelect } from "~/components/SearchSelect"
import { Loading, throttle } from "~/core/main"
import { PageContainer } from "~/core/page"
import { IProductoStock, getProductosStock, postProductosStock, queryAlmacenMovimientos, useProductosAPI } from "~/services/operaciones/productos"
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes"
import { formatN } from "~/shared/main"
import { Params } from "~/shared/security"

export default function AlmacenMovimientos() {

  const [productos] = useProductosAPI()
  const [almacenes] = useSedesAlmacenesAPI()

  const fechaFin = Params.getFechaUnix()
  const fechaInicio = fechaFin - 7
  const [form, setForm] = createSignal({ fechaFin, fechaInicio, almacenID: 0 })
  const [almacenMovimientos, setAlmacenMovimientos] = createSignal([])

  createEffect(() => {
    const almacenID = (almacenes().Almacenes||[])[0]?.ID
    if(almacenID && almacenID !== form().almacenID){ 
      setForm({ ...form(), almacenID }) 
    }
  })
  
  const consultarRegistros = async () => {
    Loading.standard("Consultando registros...")
    let records: any[] = []
    try {
      records = await queryAlmacenMovimientos(form())
    } catch (error) {
      Loading.remove(); return
    }

    console.log("registros obtenidos: ", records)
    Loading.remove()
  }

  const [filterText, setFilterText] = createSignal("")


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
      css="" tableCss="w-page"
      maxHeight="calc(80vh - 13rem)" 
      columns={[
        { header: "Producto", css: "",
          getValue: e => {
            const producto = productos().productosMap.get(e.ProductoID)
            return producto?.Nombre || `Producto-${e.ProductoID}`
          }
        },
        { header: "Lote", css: "c-purple",
          getValue: e => e.Lote
        },
        { header: "SKU", css: "c-purple",
          getValue: e => e.SKU
        },
        { header: "Propiedades", css: "",
          getValue: e => ""
        },
        { header: "Stock Min", cardColumn: [3,2], cardCss: "h5 c-steel", css: "t-c",
          getValue: e => "", 
        },
      ]}
    />
  </PageContainer>
}