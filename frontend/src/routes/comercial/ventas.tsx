import { createEffect, createMemo, createSignal } from "solid-js";
import { deviceType } from "~/app";
import { CardsList } from "~/components/Cards";
import { SearchSelect } from "~/components/SearchSelect";
import { Loading, throttle } from "~/core/main";
import { PageContainer } from "~/core/page";
import { IProductoStock, getProductosStock, useProductosAPI } from "~/services/operaciones/productos";
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes";
import { arrayToMapG, arrayToMapN, arrayToMapS, formatN } from "~/shared/main";
import { Params } from "~/shared/security";
import style from "./ventas.module.css";
import { CheckBox } from "~/components/Input";

export default function Ventas() {

  const [productosStock, setProductosStock] = createSignal([] as IProductoStock[])
  const [productos] = useProductosAPI()
  const [almacenes] = useSedesAlmacenesAPI()
  const [form, setForm] = createSignal({ })
  const [filterText, setFilterText] = createSignal("")

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

  const [productosParsed, setProdustosParsed] = createSignal([])

  createEffect(() => {    
      let newProductos = []
      for(let i = 0; i < 40; i ++){
        newProductos = newProductos.concat(productos()?.productos||[])
      }
      setProdustosParsed(newProductos)      
  })
  
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
          <input class="w100" autocomplete="off" type="text" onKeyUp={ev => {
            ev.stopPropagation()
            throttle(() => {
              setFilterText(((ev.target as any).value||"").toLowerCase().trim())
            },150)
          }}/>
        </div>
      </div>
      <div class="flex jc-between w100">
        <div class="mr-auto"></div>
        <CheckBox label="Buscar SKU"></CheckBox>
      </div>
      <div style={{ height: '70vh', position: 'relative' }}>
        <CardsList data={productosParsed()}
          render={(e,i) => {
            return <div class="px-04 py-02" 
              style={{ "margin-top": i === 0 ? '8px' : undefined }}>
              <div class={`flex jc-between ${style.card_venta}`}>
                <div class="w100 grid" 
                  style={{ "grid-template-columns": '1fr 3rem 5rem' }}>
                  
                  <div>{e.Nombre}  </div>
                  <div class="ff-mono t-r">2</div>
                  <div class="ff-mono t-r">
                    {formatN(e.Precio/100,2) as string}
                  </div>
                </div>
              </div>
            </div>
          }}
        />
      </div>
    </div>
    <div class="grow-1">
      <h1>hola</h1>
    </div>
  </PageContainer>
}