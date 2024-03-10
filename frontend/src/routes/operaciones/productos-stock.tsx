import { createSignal } from "solid-js"
import { QTable } from "~/components/QTable"
import { SearchSelect } from "~/components/SearchSelect"
import { Loading } from "~/core/main"
import { PageContainer } from "~/core/page"
import { getProductosStock } from "~/services/operaciones/productos"
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes"
import { formatN } from "~/shared/main"

export default function ProductosStock() {

  const [productosStock, setProductosStock] = createSignal([] as any[])
  const [almacenes, setAlmacenes] = useSedesAlmacenesAPI()
  const [form, setForm] = createSignal({})
  const [almacenSelected, setAlmacenSelected] = createSignal(0)

  const getStock = async (almacenID: number) => {
    let stock = []
    Loading.standard("Obteniendo Registros...")
    try {
      stock = await getProductosStock(almacenID) 
    } catch (error) {
      Loading.remove(); return
    }
    console.log("productos stock::", stock)
    setProductosStock(stock)
    Loading.remove(); return
  }

  return <PageContainer title="Almacén Stock">
    <div class="flex ai-center jc-between mb-06">
      <SearchSelect saveOn={form()} save="almacenID" css="w20rem"
        label="" keys="ID.Nombre" options={almacenes()?.Almacenes || []}
        placeholder=" ALMACÉN"
        onChange={e => {
          setAlmacenSelected(e?.ID||0)
          if(e){ getStock(e.ID); return }
        }}
      />
      <div class="flex ai-center">
        <button class="bn1 b-green" onClick={ev => {
          ev.stopPropagation()

        }}>
          <i class="icon-plus"></i>
        </button>
      </div>
    </div>
    <QTable data={productosStock()} 
      css="" tableCss="w-page"
      maxHeight="calc(80vh - 13rem)" 
      columns={[
        { header: "Producto", css: "t-c c-purple",
          getValue: e => e.ID
        },
        { header: "Lote", css: "t-c c-purple",
          getValue: e => e.ID
        },
        { header: "SKU", css: "t-c c-purple",
          getValue: e => e.ID
        },
        { header: "Propiedades", css: "t-c c-purple",
          getValue: e => e.ID
        },
        { header: "Stock", cardColumn: [3,2], field: "Nombre",
          getValue: e => e.Nombre, cardCss: "h5 c-steel",
        },
        { header: "Costo UN", cardColumn: [3,2], cardCss: "h5 c-steel", css: "t-c",
          getValue: e => e.Precio ? formatN(e.Precio,2) : "", 
        },
        { header: "Precio UN", cardColumn: [3,2], cardCss: "h5 c-steel", css: "t-c",
          getValue: e => e.Precio ? formatN(e.Precio,2) : "", 
        },
      ]}
    />
  </PageContainer>
}