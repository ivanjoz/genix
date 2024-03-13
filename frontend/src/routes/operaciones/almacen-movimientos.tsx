import { Notify } from "notiflix"
import { Show, createSignal } from "solid-js"
import { DatePicker } from "~/components/Datepicker"
import { CellEditable } from "~/components/Editables"
import { CheckBox, Input } from "~/components/Input"
import { CornerLayer, setOpenLayers } from "~/components/Modals"
import { QTable } from "~/components/QTable"
import { SearchSelect } from "~/components/SearchSelect"
import { Loading, throttle } from "~/core/main"
import { PageContainer } from "~/core/page"
import { IProductoStock, getProductosStock, postProductosStock, useProductosAPI } from "~/services/operaciones/productos"
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes"
import { formatN } from "~/shared/main"

export default function AlmacenMovimientos() {

  const [productos] = useProductosAPI()
  const [almacenes] = useSedesAlmacenesAPI()
  const [form, setForm] = createSignal({})
  const [almacenMovimientos, setAlmacenMovimientos] = createSignal([])

  const guardarRegistros = async () => {
    
  }

  const [filterText, setFilterText] = createSignal("")


  return <PageContainer title="Almacén Stock">
    <div class="flex ai-center jc-between mb-06">
      <div class="flex ai-center">
        <SearchSelect saveOn={form()} save="almacenID" css="w20rem mb-02 mr-12"
          label="Almacén" keys="ID.Nombre" options={almacenes()?.Almacenes || []}
          placeholder="" required={true}
          onChange={e => {
            
          }}
        />
        <DatePicker label="Fecha Inicio"/>
        <DatePicker label="Fecha Fin"/>
        <button class="bn1 b-blue" onClick={ev => {
          ev.stopPropagation()
          setOpenLayers([2])
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