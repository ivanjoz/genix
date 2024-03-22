import { Notify } from "notiflix"
import { Show, createEffect, createSignal } from "solid-js"
import { CellEditable } from "~/components/Editables"
import { CheckBox, Input } from "~/components/Input"
import { CornerLayer, setOpenLayers } from "~/components/Modals"
import { QTable } from "~/components/QTable"
import { SearchSelect, makeHighlString } from "~/components/SearchSelect"
import { Loading, throttle } from "~/core/main"
import { PageContainer } from "~/core/page"
import { IProductoStock, getProductosStock, postProductosStock, useProductosAPI } from "~/services/operaciones/productos"
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes"
import { formatN } from "~/shared/main"
import { Params } from "~/shared/security"

export default function ProductosStock() {

  const [productosStock, setProductosStock] = createSignal([] as IProductoStock[])
  const [productos] = useProductosAPI()
  const [almacenes] = useSedesAlmacenesAPI()
  const [form, setForm] = createSignal({ almacenID: 0 })
  const [formStock, setFormStock] = createSignal({} as IProductoStock)
  const [almacenSelected, setAlmacenSelected] = createSignal(-1)
  const [keysUpdated, setKeysUpdated] = createSignal(new Set() as Set<string>)
  const [todosProductosCheck, setTodosProductosCheck] = createSignal(false)

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
    Params.setValue("almacen_id", almacenID)
    
    for(let e of stock){
      const key = [e.ProductoID,e.SKU||"0",e.Lote||"0"].join("_")
      productoStockMap.set(key,e)
    }
    Loading.remove(); return
  }

  createEffect(() => {
    if(almacenSelected() == -1 && almacenes()?.Almacenes?.length > 0){
      const form_ = form()
      form_.almacenID = Params.getValueInt("almacen_id") || 0
      if(!form_.almacenID){ form_.almacenID = almacenes().Almacenes[0].ID }
      setAlmacenSelected(form_.almacenID)
      setForm({...form_ })
      getStock(form_.almacenID)
    }
  })

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
      current._hasUpdated = true
    } else {
      productoStockMap.set(key, rec)
      currentStock.unshift(rec)
      rec._cantidadPrev = -1
      rec.AlmacenID = almacenSelected()
      rec._hasUpdated = true
    }
    setKeysUpdated(new Set(keysUpdated()))
    setProductosStock([...currentStock])
    setFormStock({} as IProductoStock)
  }

  const guardarRegistros = async () => {
    const recordsForUpdate = productosStock().filter(e => e._hasUpdated)
    if(recordsForUpdate.length === 0){
      Notify.failure("No hay registros a actualizar."); return
    }

    Loading.standard("Enviando registros...")
    let result
    try {
      result = await postProductosStock(recordsForUpdate)
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

  const [filterText, setFilterText] = createSignal("")

  const populateProductos = (productosStock: IProductoStock[], almacenID: number) => {
    const usedProductosIDSet = new Set(productosStock.map(x => x.ProductoID))
    const newProductosStock = [...productosStock]

    for(let producto of productos().productos){
      if(usedProductosIDSet.has(producto.ID)){ continue }
      newProductosStock.push({
        AlmacenID: almacenID,
        ProductoID: producto.ID,
        Cantidad: 0,
        _isVirtual: true
      } as IProductoStock)
    }
    return newProductosStock
  }

  return <PageContainer title="Almacén Stock">
    <div class="flex ai-center jc-between mb-06">
      <div class="flex ai-center">
        <SearchSelect saveOn={form()} save="almacenID" css="w20rem s6 mb-02 mr-12"
          label="" keys="ID.Nombre" options={almacenes()?.Almacenes || []}
          placeholder=" ALMACÉN"
          onChange={e => {
            setAlmacenSelected(e?.ID||0)
            if(e){ getStock(e.ID); return }
          }}
        />
        <Show when={almacenSelected() > 0}>
          <div class="search-c4 mr-16 w14rem">
            <div><i class="icon-search"></i></div>
            <input class="w100" autocomplete="off" type="text" onKeyUp={ev => {
              ev.stopPropagation()
              throttle(() => {
                setFilterText(((ev.target as any).value||"").toLowerCase().trim())
              },150)
            }}/>
          </div>
          <CheckBox label="Todos Productos" checked={todosProductosCheck()}
            onChange={bool =>{
              setTodosProductosCheck(bool)
              let records = []
              if(bool){
                records = populateProductos(productosStock(), almacenSelected())
              } else {
                records = productosStock().filter(x => !x._isVirtual)
              }
              setProductosStock(records)
            }}
          />
        </Show>
        <Show when={!almacenSelected()}>
          <div class="flex ai-center c-red"> 
            <i class="icon-attention"></i> 
            Debe seleccionar un almacén.
          </div>
        </Show>
      </div>
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
      filterText={filterText()}
      makeFilter={e => {
        const producto = productos().productosMap.get(e.ProductoID)?.Nombre || ""
        return producto
      }}
      columns={[
        { header: "Producto", css: "",
          render: e => {
            const producto = productos().productosMap.get(e.ProductoID)?.Nombre || `P-${e.ProductoID}`
            return makeHighlString(producto, filterText())
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
          css: "t-r", headerStyle: { width: '8rem' },
          render: e => {
            return <CellEditable saveOn={e} save="" 
              contentClass="px-06 flex ai-center jc-end"
              inputClass="t-c" type="number"
              getValue={() => { return e.Cantidad }}
              onChange={c => {
                if(e.Cantidad === c){ return }
                e._cantidadPrev = e._cantidadPrev || e.Cantidad || -1
                e.Cantidad = parseInt(c as string||"0")
                e._hasUpdated = true
                const key = [e.ProductoID,e.SKU||"0",e.Lote||"0"].join("_")
                keysUpdated().add(key)
                setKeysUpdated(new Set(keysUpdated()))
              }}
              render={() => {
                if(e._cantidadPrev && e._cantidadPrev !== e.Cantidad){
                  const prev = e._cantidadPrev === -1 ? 0 : e._cantidadPrev
                  return <div class="flex ai-center jc-end">
                    <div>{prev}</div>
                    <div class="mr-02 ml-02">→</div>
                    <div class="c-red ff-bold">{e.Cantidad}</div>
                  </div>
                } else {
                  if(!e.Cantidad){
                    return <div class="c-red ff-bold">{e.Cantidad}</div>
                  }
                  return String(e.Cantidad)
                }
              }}
            />
          },
        },
        { header: "Costo UN", cardColumn: [3,2], cardCss: "h5 c-steel", css: "t-c",
          getValue: e => e.CostoUn ? formatN(e.CostoUn,2) : "", 
          render: e => {
            return <CellEditable saveOn={e} save="CostoUn" 
              contentClass="px-06 flex ai-center jc-end"
              inputClass="t-c" type="number"
              onChange={c => {
                e._hasUpdated = true
                const key = [e.ProductoID,e.SKU||"0",e.Lote||"0"].join("_")
                keysUpdated().add(key)
                setKeysUpdated(new Set(keysUpdated()))                
              }}
              render={() => {
                return e.CostoUn ? formatN(e.CostoUn,2) : ""
              }}
            />
          },
        },
        { header: "Stock Min", cardColumn: [3,2], cardCss: "h5 c-steel", css: "t-c",
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