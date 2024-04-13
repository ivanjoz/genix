import { For, Show, createEffect, createMemo, createSignal, on } from "solid-js";
import { deviceType } from "~/app";
import { CardsList } from "~/components/Cards";
import { SearchSelect, makeHighlString } from "~/components/SearchSelect";
import { Loading, include, throttle } from "~/core/main";
import { PageContainer } from "~/core/page";
import { IProducto, IProductoStock, getProductosStock, useProductosAPI } from "~/services/operaciones/productos";
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes";
import { arrayToMapG, arrayToMapN, arrayToMapS, formatN, joinb } from "~/shared/main";
import { Params } from "~/shared/security";
import style from "./ventas.module.css";
import { CheckBox } from "~/components/Input";

export interface ProductoVenta {
  key: string
  cant: number
  producto: IProducto
  searchText: string
  isSubunidad?: boolean
  skus?: IProductoStock[]
}

export interface VentaProducto {
  productoID: number
  sku?: string
  lote?: string
  cantidad: number
  subCantidad: number
}

const [ventaProductoInput, setVentaProductoInput] = createSignal(null as HTMLInputElement)

export default function Ventas() {

  const [productosStock, setProductosStock] = createSignal([] as IProductoStock[])
  const [productos] = useProductosAPI()
  const [almacenes] = useSedesAlmacenesAPI()
  const [form, setForm] = createSignal({ })
  const [filterText, setFilterText] = createSignal("")
  const [productoSelected, setProductoSelected] = createSignal(-1)
  const [almacenSelected, setAlmacenSelected] = createSignal(-1)

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
        producto: producto, cant: 0,  key: `M${producto.ID}`,
        searchText: producto.Nombre.toLowerCase()
      } as ProductoVenta

      const skusStock: IProductoStock[] = []
      const mainStock: IProductoStock[] = []

      for(let e of stocks){
        e.SKU ? skusStock.push(e) : mainStock.push(e)
      }

      if(skusStock.length > 0){
        const clone = {...base, skus: skusStock, key: `P${producto.ID}`}
        for(let e of skusStock){ clone.cant += e.Cantidad }
        newProductos.push(clone)
      }

      if(mainStock.length > 0){
        let clone: ProductoVenta
        if(producto.SbnCantidad > 1){
          clone = {...base, key: `S${producto.ID}`}
          clone.isSubunidad = true
          clone.cant = producto.SbnCantidad
        }

        for(let e of mainStock){ base.cant += e.Cantidad }
        newProductos.push(base)
        if(clone){ newProductos.push(clone) }
      }
    }

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

  const agregarStock = (e: ProductoVenta, cant: number) => {
    const producto = productosParsed()[productoSelected()]
    if(!producto){ return }
    console.log("agregar producto:: ", producto)
  }

  return <PageContainer title="Ventas" class="flex">
    <div class="jc-between mb-06" style={{ width: "46%" }}
      classList={{ "column": [2,3].includes(deviceType()) }}
    >
      <div class="flex">
        <SearchSelect css="w16rem s6 mb-02 mr-12"
          label="" keys="ID.Nombre" options={almacenes()?.Almacenes || []}
          placeholder=" ALMACÉN" selected={almacenSelected()}
          onChange={e => {
            if(e){ getStock(e.ID); return }
          }}
        />
        <div class="search-c4 mr-16 w14rem">
          <div><i class="icon-search"></i></div>
          <input class="w100" autocomplete="off" type="text" 
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
              console.log(ev.key)
              if(ev.key === 'ArrowUp'){
                const newIdx = productoSelected()-1
                if(newIdx < -1){ return }
                setProductoSelected(newIdx)
              } else if(ev.key === 'ArrowDown'){
                setProductoSelected(productoSelected()+1)
              } else if(ev.key === 'Enter' && productoSelected() >= 0){
                
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
              isSelected={isSelected()}
              productoStock={e}
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
    <div class="grow-1">
      <h1>hola</h1>
    </div>
  </PageContainer>
}

function ProductoCantidad(){

  const cantidades = [2,3,4,5,6,8,10,12]

  return <div class={["flex p-rel ai-end h100",style.buttons_cantidad].join(" ")}>
    <For each={cantidades}>
      {e => {
        return <div class={`flex-center ff-bold ${style.button_venta_cantidad}`}>{e}</div>
      }} 
    </For>
  </div>
}

interface IProductoVentaCard {
  productoStock: ProductoVenta
  isSelected: boolean
  filterText: string
  idx: number
  setProductoSelected: (e: number) => void
  onmouseover: () => void
}

const ProductoVentaCard = (props: IProductoVentaCard) => {

  let inputRef: HTMLInputElement
  createEffect(on(() => props.isSelected, () => {
    if(props.isSelected){ setVentaProductoInput(inputRef) }
  }))

  const makeNombre = () => {
    const nombre = makeHighlString(props.productoStock.producto.Nombre, props.filterText)
    if(props.productoStock.isSubunidad || props.productoStock.skus?.length > 0){
      return <div class="flex ai-center">
        {nombre}
        <Show when={props.productoStock.isSubunidad}>
          <div class="ml-04 mr-04">|</div>
          <div class="ff-bold c-purple2">{props.productoStock.producto.SbnUnidad}</div>
        </Show>
        <Show when={!props.productoStock.isSubunidad}>
          <div class="ff-bold ml-04 c-purple2">(SKU)</div>
        </Show>
      </div> 
    } else {
      return nombre
    }
  }

  const isSku = () => props.productoStock.skus?.length > 0

  const firstSkus = createMemo(() => {
    let skus = props.productoStock.skus || []
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
                return <div class={`h5 ${style.producto_sku}`}>
                  {e.SKU}
                </div>
              }}
              </For>
            </div>
          </Show>
          <Show when={!isSku()}>
            <ProductoCantidad/>
          </Show>           
        </div>
        <div class={joinb("p-rel flex ai-center",style.input_cantidad)}>
          <input type="number" value="" ref={inputRef}
            class="ff-mono w100"
          />
          <div class="ml-04">/</div>
        </div>
        <div class="ff-mono t-r">{props.productoStock.cant}</div>
        <div class="ff-mono t-r">
          {formatN(props.productoStock.producto.Precio/100,2) as string}
        </div>
      </div>
    </div>
  </div>
}