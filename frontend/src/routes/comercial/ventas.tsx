import { For, Show, createEffect, createMemo, createSignal, on } from "solid-js";
import { deviceType } from "~/app";
import { CardsList } from "~/components/Cards";
import { SearchSelect, makeHighlString } from "~/components/SearchSelect";
import { Loading, include, throttle } from "~/core/main";
import { PageContainer } from "~/core/page";
import { IProducto, IProductoStock, getProductosStock, useProductosAPI } from "~/services/operaciones/productos";
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes";
import { arrayToMapG, arrayToMapN, arrayToMapS, formatMo, formatN, joinb } from "~/shared/main";
import { Params } from "~/shared/security";
import style from "./ventas.module.css";
import { CheckBox, Input, InputDisabled } from "~/components/Input";
import { QTable } from "~/components/QTable";

export interface ProductoVenta {
  key: string
  cant: number
  producto: IProducto
  searchText: string
  isSubUnidad?: boolean
  skus?: IProductoStock[]
}

export interface SkuCant {
  sku: string, cant: number
}

export interface VentaProducto {
  key: string
  productoID: number
  skus?: Map<string,number>
  lote?: string
  cantidad: number
  isSubUnidad?: boolean
}

export interface IVenta {
  clienteID: number
  subtotal: number
  igv: number
  total: number
}

const [ventaProductoInput, setVentaProductoInput] = createSignal(null as HTMLInputElement)

export default function Ventas() {

  const [productosStock, setProductosStock] = createSignal([] as IProductoStock[])
  const [productos] = useProductosAPI()
  const [almacenes] = useSedesAlmacenesAPI()
  const [form, setForm] = createSignal({ } as IVenta)
  const [filterText, setFilterText] = createSignal("")
  const [productoSelected, setProductoSelected] = createSignal(-1)
  const [almacenSelected, setAlmacenSelected] = createSignal(-1)
  const [ventaErrorMessage, setVentaErrorMessage] = createSignal("")
  const [ventaProductos, setVentaProductos] = createSignal([] as VentaProducto[])

  let searchInput: HTMLInputElement

  const ventaProductosMap = createMemo(() => {
    return arrayToMapS(ventaProductos(), "key")
  })

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
    Loading.remove(); return
  }

  createEffect(() => {
    if(almacenSelected() == -1 && almacenes()?.Almacenes?.length > 0){
      const form_ = form()
      let almacenID = Params.getValueInt("almacen_id") || 0
      if(!almacenID){ almacenID = almacenes().Almacenes[0].ID }
      setAlmacenSelected(almacenID)
      getStock(almacenID)
    }
  })

  const [productosParsed, setProdustosParsed] = createSignal([] as ProductoVenta[])
  const [productosParsedAll, setProductosParsedAll] = createSignal([] as ProductoVenta[])

  createEffect(() => {    
    const productoToStockMap = arrayToMapG(productosStock(), "ProductoID")
    console.log("productos stock map::", productosStock(), productoToStockMap)
    const newProductos: ProductoVenta[] = []

    for(let producto of productos()?.productos||[]){
      const stocks = productoToStockMap.get(producto.ID) || []
      if(stocks.length === 0){ continue }

      const base = { 
        producto: producto, cant: 0,  key: `P${producto.ID}`,
        searchText: producto.Nombre.toLowerCase()
      } as ProductoVenta

      const skusStock: IProductoStock[] = []
      const mainStock: IProductoStock[] = []

      for(let e of stocks){
        e.SKU ? skusStock.push(e) : mainStock.push(e)
      }

      if(skusStock.length > 0){
        const clone = {...base, skus: skusStock, key: `K${producto.ID}`}
        for(let e of skusStock){ clone.cant += e.Cantidad }
        newProductos.push(clone)
      }

      if(mainStock.length > 0){
        let clone: ProductoVenta
        if(producto.SbnCantidad > 1){
          clone = {...base, key: `S${producto.ID}`}
          clone.isSubUnidad = true
          clone.cant = producto.SbnCantidad
        }

        for(let e of mainStock){ base.cant += e.Cantidad }
        newProductos.push(base)
        if(clone){ newProductos.push(clone) }
      }
    }
    console.log("productos parsed::", newProductos)
    setProdustosParsed(newProductos) 
    setProductosParsedAll(newProductos)        
  })
  
    
  const letters = "abcdefghijklmnopqrstuvwxyz"
  const numbers = "123456789"
  let lastSearchText = ""

  const filterProductos = (text: string) => {
    if(text === filterText() || text === lastSearchText){ return }
    console.log("buscando:: ", text)
    lastSearchText = text
    throttle(() => {
      const textSlice = text.toLowerCase().split(" ")
      const filtered = []
      for(let e of productosParsedAll()){
        if(include(e.searchText, textSlice)){
          filtered.push(e)
        }
      }
      console.log("filtrados::", filtered)
      setProdustosParsed(filtered)
      setFilterText(text)
      setProductoSelected(-1)
    },80)
  }
  
  const agregarProductoVenta = (e: ProductoVenta, cant: number, sku?: string) => {
    const ventaCant = ventaProductosMap().get(e.key)?.cantidad || 0
    const stock = e.cant - ventaCant
    if(stock < cant){ 
      setVentaErrorMessage(`No hay suficiente stock de "${e.producto.Nombre}" para agregar ${cant} unidades.`)
      return
    }
    
    console.log("producto venta::", e)

    const ventaProducto = ventaProductos().find(x => x.key === e.key)
    if(ventaProducto){
      ventaProducto.cantidad += cant
      if(sku){
        const skuAdded = ventaProducto.skus.get(sku)
        if(skuAdded){
          const stockCant = e.skus.find(x => x.SKU === sku)?.Cantidad || 0
          if(skuAdded >= stockCant){
            setVentaErrorMessage(`El SKU ${sku} del producto "${e.producto.Nombre}" sólo posee ${stockCant} unidad(es).`)
            return
          }
          ventaProducto.skus.set(sku, skuAdded +1)
        } else {
          ventaProducto.skus.set(sku, 1)
        }
      }
    } else {
      ventaProductos().push({ 
        key: e.key, cantidad: cant, productoID: e.producto.ID,
        skus: new Map(sku ? [[sku,1]] : []),
        isSubUnidad: e.isSubUnidad || false
      })
    }
    const newVentaProductos = [...ventaProductos()]
    console.log("venta productos::", newVentaProductos)
    setVentaProductos(newVentaProductos)
    const input = ventaProductoInput()
    if(input){ input.value = "" }
    if(searchInput){ searchInput.value = "" }
    setFilterText("")
    setProdustosParsed([...productosParsedAll()])
    setProductoSelected(-1)
  }

  const recalcVentaTotales = (newVentaProductos: VentaProducto[]) => {
    console.log("recalculando::", newVentaProductos)
    const newForm = {...form(), total: 0, subtotal: 0, igv: 0}

    for(let vp of newVentaProductos){
      const producto = productos().productosMap.get(vp.productoID)
      const monto = producto.PrecioFinal * vp.cantidad
      newForm.total += monto
    }
    newForm.subtotal = Math.floor(newForm.total / 1.18)
    newForm.igv = newForm.total - newForm.subtotal
    setForm(newForm)
  }

  createEffect(on(()=> ventaProductos(), 
    () => {
      recalcVentaTotales(ventaProductos())
    }))

  const layerWidth = 52

  return <PageContainer title="Ventas" class="flex">
    <div class="jc-between mb-06 mr-16" style={{ width: `${100 - layerWidth}%` }}
      classList={{ "column": [2,3].includes(deviceType()) }}
    >
      <div class="flex">
        <SearchSelect css="w16rem s6 mb-02 mr-12"
          label="" keys="ID.Nombre" options={almacenes()?.Almacenes || []}
          placeholder=" ALMACÉN" selected={almacenSelected()}
          onChange={e => {
            setAlmacenSelected(e?.ID || -1)
            if(e){ getStock(e.ID); return }
          }}
        />
        <div class="search-c4 mr-16 w14rem">
          <div><i class="icon-search"></i></div>
          <input class="w100" autocomplete="off" type="text" ref={searchInput}
            onKeyUp={ev => {
              ev.stopPropagation()
              const text = ((ev.target as any).value||"").toLowerCase().trim()
              filterProductos(text)
            }}
            onkeypress={() => {
              return false
            }}
            onkeydown={ev => {
              ev.stopPropagation()
              if(ventaErrorMessage()?.length > 0){
                setVentaErrorMessage("")
              }
              console.log(ev.key)
              
              if(ev.key === 'ArrowUp'){
                const newIdx = productoSelected()-1
                if(newIdx < -1){ return }
                setProductoSelected(newIdx)
              } else if(ev.key === 'ArrowDown'){
                setProductoSelected(productoSelected()+1)
              } else if(ev.key === 'Enter' && productoSelected() >= 0){
                const input = ventaProductoInput()
                const cant = parseInt(input.value||"0") || 1
                const producto = productosParsed()[productoSelected()]
                if(producto.skus?.length > 0){
                  setVentaErrorMessage("Debe seleccionar 1 SKU.")
                  return
                }
                agregarProductoVenta(producto, cant)
              } else {
                const key = ev.key.toLocaleLowerCase()
                if(productoSelected() >= 0 && (numbers.includes(key) || ev.key === 'Backspace')){
                  // Cambia el valor del input del element
                  const input = ventaProductoInput()
                  console.log("key backspace:: ", input.value)
                  if(input){
                    if(ev.key === 'Backspace'){
                      if(input.value){
                        input.value = ""
                        ev.preventDefault()
                      }
                    } else {
                      input.value += key
                      ev.preventDefault()
                    }
                  }
                } else if(letters.includes(key) || numbers.includes(key)){
                  filterProductos(lastSearchText + ev.key)
                }
              }
            }}
          />
        </div>
      </div>
      <div class="flex jc-between w100">
        <div class="mr-auto"></div>
        <CheckBox label="Buscar SKU"></CheckBox>
      </div>
      <div style={{ height: 'calc(100% - 12px - 4.2rem)', position: 'relative' }}>
        <CardsList data={productosParsed()}
          render={(e,i) => {
            console.log("card rerender:: ", i, e)
            const isSelected = () => { return i === productoSelected() }
            return <ProductoVentaCard idx={i}
              isSelected={isSelected()} ventasProductosMap={ventaProductosMap()}
              productoStock={e}
              agregarProductoVenta={(cant, sku) => {
                agregarProductoVenta(e, cant, sku)
              }}
              setProductoSelected={setProductoSelected}
              filterText={filterText()}
              onmouseover={() => {
                if(productoSelected() >= 0){ setProductoSelected(-1) }
              }}
            />
          }}
        />
      </div>
    </div>
    <div class="side-layer1 grow-1 px-12 py-08" 
      style={{ width: `calc(var(--page-width) * ${layerWidth/100})` }}>
      { ventaErrorMessage() &&
        <div class="box-error-ms mb-08">{ventaErrorMessage()}</div>
      }
      <div class="flex ai-center jc-between w100 mb-12">
        <div class="ff-bold h3 mb-04">
          DETALLE DE VENTA
        </div>
        <div class="flex ai-center">
          <button class="bn1 b-blue">
            Guardar
            <i class="icon-floppy"></i>
          </button>
        </div>
      </div>
      <div class="w100-10 flex-wrap">
        <Input label="Cliente" css="w-06x" saveOn={form()} save="clienteID" />
        <InputDisabled label="Código" css="w-05x" />
        <InputDisabled label="Sub Total" css="ff-mono w-045x" 
          getContent={() => formatMo(form().subtotal)}/>
        <InputDisabled label="IGV" css="ff-mono w-04x" 
          getContent={() => formatMo(form().igv)}/>
        <InputDisabled label="Total" css="ff-mono w-045x" content={form().total}
          getContent={() => formatMo(form().total)}/>
      </div>
      <QTable data={ventaProductos()} 
        css="w100 mt-12" tableCss="w-page-t w100"
        maxHeight="calc(100vh - 12rem - 12px)"
        columns={[
          { header: "Nº",
            getValue: (e,idx) => {
              return String(idx + 1)
            }
          },
          { header: "Producto",
            render: e => {
              const producto = productos().productosMap.get(e.productoID)
              let nombre = producto?.Nombre || ""
              if(e.isSubUnidad){
                nombre = `${producto.SbnUnidad} de ${nombre}`
              }
              return nombre
            }
          },
          { header: "SKU",
            render: e => {
              console.log("skuss",e)
              if(!e.skus){ return "" }
              return <>
                { [...e.skus].map(([sku,cant]) => {
                    return <div>{sku} {cant > 1 ? <span class="ml-04">({cant})</span> : null}</div>
                  })

                }
              </>
            }
          },
          { header: "Cantidad",
            render: e => {
              return formatN(e.cantidad)
            }
          },
          { header: "Monto", css: "ff-mono t-r",
            render: e => {
              const producto = productos().productosMap.get(e.productoID)
              const monto = producto.PrecioFinal * e.cantidad
              return <span>
                {formatN(monto/100,2) as string}
              </span>
            }
          },
          { header: "...", headerStyle: { width: '2rem' }, cellStyle: { padding: '0 4px' },
            render: e => {
              return  <button class="bn1 s5 b-red" onClick={ev => {
                ev.stopPropagation()
                const newVentaProductos = ventaProductos().filter(x => x.key !== e.key)
                setVentaProductos(newVentaProductos)
              }}>
                <i class="icon-trash"></i>
              </button>
            }
          },
        ]}
      />
    </div>
  </PageContainer>
}

interface IProductoCantidad {
  onSelect: (cant: number) => void
}

function ProductoCantidad(props: IProductoCantidad){

  const cantidades = [2,3,4,5,6,8,10,12]

  return <div class={["flex p-rel ai-end h100",style.buttons_cantidad].join(" ")}>
    <For each={cantidades}>
      {e => {
        return <div class={`flex-center ff-bold ${style.button_venta_cantidad}`} 
        onClick={ev => {
          ev.stopPropagation()
          props.onSelect(e)
        }}>
          {e}
        </div>
      }} 
    </For>
  </div>
}

interface IProductoVentaCard {
  idx: number
  productoStock: ProductoVenta
  isSelected: boolean
  filterText: string
  ventasProductosMap: Map<string,VentaProducto>
  setProductoSelected: (e: number) => void
  agregarProductoVenta: (cant: number, sku?: string) => void
  onmouseover: () => void
}

const ProductoVentaCard = (props: IProductoVentaCard) => {

  let inputRef: HTMLInputElement
  createEffect(on(() => props.isSelected, () => {
    if(props.isSelected){ setVentaProductoInput(inputRef) }
  }))

  const makeNombre = () => {
    const nombre = makeHighlString(props.productoStock.producto.Nombre, props.filterText)
    if(props.productoStock.isSubUnidad || props.productoStock.skus?.length > 0){
      return <div class="flex ai-center">
        {nombre}
        <Show when={props.productoStock.isSubUnidad}>
          <div class="ml-04 mr-04">|</div>
          <div class="ff-bold c-purple2">{props.productoStock.producto.SbnUnidad}</div>
        </Show>
        <Show when={!props.productoStock.isSubUnidad}>
          <div class="ff-bold ml-04 c-purple2">(SKU)</div>
        </Show>
      </div> 
    } else {
      return nombre
    }
  }

  const isSku = () => props.productoStock.skus?.length > 0
  const getCant = createMemo(() => {    
    const key = props.productoStock.key
    let ventaCant = props.ventasProductosMap.get(key)?.cantidad || 0
    // revisa si hay sub-unidades agregadas
    if(key[0] === 'P'){
      const keySub = `S${key.slice(1)}`
      const subUnidadCant = props.ventasProductosMap.get(keySub)?.cantidad || 0
      if(subUnidadCant > 0){
        const producto = props.productoStock.producto
        ventaCant = ventaCant + Math.ceil(subUnidadCant / producto.SbnCantidad)
      }
    }
    return props.productoStock.cant - ventaCant
  })

  const firstSkus = createMemo(() => {
    let skus = props.productoStock.skus || []
    const ventaProducto = props.ventasProductosMap.get(props.productoStock.key)
    if(ventaProducto?.skus?.size > 0){
      skus = skus.filter(x => {
        if(!ventaProducto.skus.has(x.SKU)){
          return true
        }
        return x.Cantidad > ventaProducto.skus.get(x.SKU)
      })
    }
    if(skus.length > 4){ skus = skus.slice(0,4) }
    return skus
  })

  return <div class={"px-04 py-02"}
    classList={{ "sld": props.isSelected, "sku": isSku() }}
    style={{ "margin-top": props.idx === 0 ? '8px' : undefined }}
    onmouseover={ev => {
      ev.stopPropagation()
      props.onmouseover()
    }}
  >
    <div class={joinb("flex p-rel jc-between", style.card_venta)}
      onClick={() => {
        props.setProductoSelected(props.idx)
      }}
    >
      <div class={style.card_venta_nombre}>
        <div class={style.card_venta_nombre_line}></div>
          { makeNombre() }
      </div>
      <div class="w100 grid ai-center"
        style={{ "grid-template-columns": '1fr 3.8rem 2.8rem 5rem' }}>                  
        <div class={`h100 ${style.card_venta_producto_ctn}`}
            classList={{ "flex ai-center": !isSku() }}
          >
          <div class={`${style.card_venta_producto}`}>
            { makeNombre() }
          </div>
          <Show when={isSku()}>
            <div class="flex-wrap z20 p-rel" style={{ margin: '0 -3px' }}>
              <For each={firstSkus()}>
              {e => {
                return <div class={`h5 ${style.producto_sku}`} onClick={ev =>{
                  ev.stopPropagation()
                  props.agregarProductoVenta(1,e.SKU)
                }}>
                  {e.SKU}
                </div>
              }}
              </For>
            </div>
          </Show>
          <Show when={!isSku()}>
            <ProductoCantidad onSelect={cant => {
              props.agregarProductoVenta(cant)
            }}/>
          </Show>           
        </div>
        <div class={joinb("p-rel flex ai-center",style.input_cantidad)}>
          <input type="number" value="" ref={inputRef}
            class="ff-mono w100"
          />
          <div class="ml-04">/</div>
        </div>
        <div class="ff-mono t-r">{getCant()}</div>
        <div class="ff-mono t-r">
          {formatN(props.productoStock.producto.PrecioFinal/100,2) as string}
        </div>
      </div>
    </div>
  </div>
}