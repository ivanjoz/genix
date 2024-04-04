import { For, createEffect, createMemo, createSignal } from "solid-js";
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

export interface ProductoStock {
  key: string
  producto: IProducto
  searchText: string
}

export default function Ventas() {

  const [productosStock, setProductosStock] = createSignal([] as IProductoStock[])
  const [productos] = useProductosAPI()
  const [almacenes] = useSedesAlmacenesAPI()
  const [form, setForm] = createSignal({ })
  const [filterText, setFilterText] = createSignal("")
  const [productoSelected, setProductoSelected] = createSignal(-1)

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
    
    const productosStockMap = arrayToMapG(stock, "ProductoID")
    Loading.remove(); return
  }

  const [productosParsed, setProdustosParsed] = createSignal([] as ProductoStock[])
  const [productosParsedAll, setProductosParsedAll] = createSignal([] as ProductoStock[])

  createEffect(() => {    
      let newProductos: ProductoStock[] = []
      for(let i = 0; i < 40; i ++){
        for(let e of productos()?.productos||[]){
          newProductos.push({ 
            key: `p_${i}_${e.ID}`, 
            producto: e,
            searchText: e.Nombre.toLowerCase()
          })
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

  return <PageContainer title="Ventas" class="flex">
    <div class="jc-between mb-06" style={{ width: "46%" }}
      classList={{ "column": [2,3].includes(deviceType()) }}
    >
      <div class="flex">
        <SearchSelect saveOn={form()} save="almacenID" css="w16rem s6 mb-02 mr-12"
          label="" keys="ID.Nombre" options={almacenes()?.Almacenes || []}
          placeholder=" ALMACÉN"
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
              if(ev.key === 'ArrowUp'){
                const newIdx = productoSelected()-1
                if(newIdx < -1){ return }
                setProductoSelected(newIdx)
              } else if(ev.key === 'ArrowDown'){
                setProductoSelected(productoSelected()+1)
              } else {
                const key = ev.key.toLocaleLowerCase()
                if(productoSelected() >= 0 && numbers.includes(key)){
                  ev.preventDefault()
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

  return <div class={["flex ai-end h100",style.buttons_cantidad].join(" ")}>
    <For each={cantidades}>
      {e => {
        return <div class={`flex-center ff-bold ${style.button_venta_cantidad}`}>{e}</div>
      }} 
    </For>
  </div>
}

interface IProductoVentaCard {
  productoStock: ProductoStock
  isSelected: boolean
  filterText: string
  idx: number
  setProductoSelected: (e: number) => void
  onmouseover: () => void
}

const ProductoVentaCard = (props: IProductoVentaCard) => {

  return <div class={"px-04 py-02"}
    classList={{ "sld": props.isSelected }}
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
          { makeHighlString(props.productoStock.producto.Nombre, props.filterText) }
      </div>
      <div class="w100 grid ai-center"
        style={{ "grid-template-columns": '1fr 3.8rem 2.8rem 5rem' }}>                  
        <div class={`flex ai-center h100`}>
          <div class={`${style.card_venta_producto}`}>
            { makeHighlString(props.productoStock.producto.Nombre, props.filterText) }
          </div>
          <ProductoCantidad />             
        </div>
        <div class={joinb("p-rel flex ai-center",style.input_cantidad)}>
          <input type="number" value="1"
            class="ff-mono w100"
          />
          <div class="ml-04">/</div>
        </div>
        <div class="ff-mono t-r">2</div>
        <div class="ff-mono t-r">
          {formatN(props.productoStock.producto.Precio/100,2) as string}
        </div>
      </div>
    </div>
  </div>
}