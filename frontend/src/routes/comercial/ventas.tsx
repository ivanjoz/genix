import { createEffect, createSignal } from "solid-js";
import { deviceType } from "~/app";
import { CardsList } from "~/components/Cards";
import { SearchSelect } from "~/components/SearchSelect";
import { Loading, throttle } from "~/core/main";
import { PageContainer } from "~/core/page";
import { IProductoStock, getProductosStock, useProductosAPI } from "~/services/operaciones/productos";
import { useSedesAlmacenesAPI } from "~/services/operaciones/sedes-almacenes";
import { arrayToMapG, arrayToMapN, arrayToMapS } from "~/shared/main";
import { Params } from "~/shared/security";

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
      <div>
        <CardsList data={productos()?.productos||[]}
          render={e => {
            return <div>
              <div class="flex">
                <div class="w12rem">{e.Nombre}</div>
                <div class="w12rem">{e.Precio}</div>
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