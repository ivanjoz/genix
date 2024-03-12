import { Notify } from "notiflix"
import { Show, createSignal } from "solid-js"
import { Input } from "~/components/Input"
import { CornerLayer, setOpenLayers } from "~/components/Modals"
import { QTable } from "~/components/QTable"
import { SearchSelect } from "~/components/SearchSelect"
import { Loading } from "~/core/main"
import { PageContainer } from "~/core/page"
import { IProductoStock, getProductosStock, postProductosStock, useProductosAPI } from "~/services/operaciones/productos"
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes"
import { formatN } from "~/shared/main"

export default function ProductosStock() {

  const [productosStock, setProductosStock] = createSignal([] as IProductoStock[])
  const [productos] = useProductosAPI()
  const [almacenes] = useSedesAlmacenesAPI()
  const [form, setForm] = createSignal({})
  const [formStock, setFormStock] = createSignal({} as IProductoStock)
  const [almacenSelected, setAlmacenSelected] = createSignal(0)
  const [keysUpdated, setKeysUpdated] = createSignal(new Set() as Set<string>)

  let productoStockMap: Map<string,IProductoStock> = new Map()

  const getStock = async (almacenID: number) => {
    let stock: IProductoStock[] = []
    Loading.standard("Obteniendo Registros...")
    try {
      stock = await getProductosStock(almacenID) 
    } catch (error) {
      Loading.remove(); return
    }
    console.log("productos stock::", stock)
    setProductosStock(stock)

    for(let e of stock){
      const key = [e.ProductoID,e.SKU||"0",e.Lote||"0"].join("_")
      productoStockMap.set(key,e)
    }
    Loading.remove(); return
  }

  const addProductoStock = (rec: IProductoStock) => {
    if(!rec.ProductoID || !rec.Cantidad){
      Notify.failure("Debe seleccionar un producto y asignar una cantidad.")
      return
    }
    const key = [rec.ProductoID,rec.SKU||"0",rec.Lote||"0"].join("_")
    const currentStock = productosStock()
    keysUpdated().add(key)

    if(productoStockMap.has(key)){
      const current = productoStockMap.get(key)
      current.Cantidad = rec.Cantidad
      current.CostoUn = rec.CostoUn || current.CostoUn
      current._cantidadPrev = current._cantidadPrev || current.Cantidad || -1
    } else {
      productoStockMap.set(key, rec)
      currentStock.unshift(rec)
      rec._cantidadPrev = -1
      rec.AlmacenID = almacenSelected()
    }
    setKeysUpdated(new Set(keysUpdated()))
    setProductosStock([...currentStock])
    setFormStock({} as IProductoStock)
  }

  const guardarRegistros = async () => {
    const recordsForUpdate = productosStock().filter(e => e._cantidadPrev)
    if(recordsForUpdate.length === 0){
      Notify.failure("No hay registros a actualizar."); return
    }

    Loading.standard("Enviando registros...")
    let result
    try {
      result = postProductosStock(recordsForUpdate)
    } catch (error) {
      Loading.remove(); return
    }

    for(let e of recordsForUpdate){
      e._cantidadPrev = 0
    }
    console.log("resultado obtenido::", result)
    setProductosStock([...productosStock()])
    Loading.remove()
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
        <Show when={keysUpdated().size > 0}>
          <button class="bn1 b-blue mr-08" onClick={ev => {
            ev.stopPropagation()
            guardarRegistros()
          }}>
            <i class="icon-floppy"></i>
            Guardar ({keysUpdated().size})
          </button>
        </Show>
        <button class="bn1 b-green" onClick={ev => {
          ev.stopPropagation()
          setOpenLayers([2])
        }}>
          <i class="icon-plus"></i>
        </button>
      </div>
    </div>
    <QTable data={productosStock()} 
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
        { header: "Stock", cardColumn: [3,2], field: "Nombre",
          cardCss: "h5 c-steel",
          render: e => {
            if(e._cantidadPrev && e._cantidadPrev !== e.Cantidad){
              const prev = e._cantidadPrev === -1 ? 0 : e._cantidadPrev
              return <div class="flex ai-center jc-end">
                <div>{prev}</div>
                <div class="mr-02 ml-02">→</div>
                <div class="c-red ff-bold">{e.Cantidad}</div>
              </div>
            } else {
              return String(e.Cantidad)
            }
          },
        },
        { header: "Costo UN", cardColumn: [3,2], cardCss: "h5 c-steel", css: "t-c",
          getValue: e => e.CostoUn ? formatN(e.CostoUn,2) : "", 
        },
        { header: "Precio UN", cardColumn: [3,2], cardCss: "h5 c-steel", css: "t-c",
          getValue: e => "", 
        },
      ]}
    />
    <CornerLayer title="Nuevo Producto Stock" id={2} css="py-12 px-12"
      buttonSave={<><i class="icon-ok"></i>Agregar</>}
      onSave={() => {
        addProductoStock(formStock())
      }}
    >
      <div class="w100-10 flex-wrap in-s2">        
        <SearchSelect saveOn={formStock()} save="ProductoID" css="w-24x mb-10"
          label="Producto1" keys="ID.Nombre" options={productos()?.productos || []}
          placeholder=" " required={true}
        />
        <Input saveOn={formStock()} save="SKU" 
          css="w-16x mb-10" label="SKU"
        />
        <Input saveOn={formStock()} save="Cantidad" required={true}
          css="w-08x mb-10" label="Cantidad" type="number"
        />
        <Input saveOn={formStock()} save="Lote"
          css="w-16x mb-10" label="Lote" type="text"
        />
        <Input saveOn={formStock()} save="CostoUn"
          css="w-08x mb-10" label="Costo x Unidad" type="number"
        />
      </div>
    </CornerLayer>
  </PageContainer>
}