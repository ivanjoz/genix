import { createEffect, createSignal, For } from "solid-js"
import { IHeader1 } from "./headers"
import { IPageSection } from "./page"
import { useProductosCmsAPI } from "./productos-service"
import { ImageCtn, ImageUploader } from "~/components/Uploaders"
import s1 from "./components.module.css"
import { formatN } from "~/shared/main"
import { Portal } from "solid-js/web"
import iconCartSvg from "../assets/icon_cart.svg?raw"
import iconCancelSvg from "../assets/icon_cancel.svg?raw"
import { parseSVG } from "~/core/main"
import { IProducto } from "~/services/operaciones/productos"

export function ProductosCuadrilla(props: IHeader1) { // type: 10

  const [ productos ] = useProductosCmsAPI()

  createEffect(() => {
    console.log("icon cart svg::", iconCartSvg)
    console.log("productos obtenidos component::", productos())
  })

  return <div class={"w100 " + s1.product_cuadrilla_ctn}>
    <div class="w100 flex-wrap jc-center">
      { productos().isFetching && <div>Obteniendo productos...</div> }
      { (productos().productos||[]).map(e => {
          return <div class={`p-rel flex-column ai-center ${s1.product_card}`}>
            { e.Images?.length > 0 &&
              <ImageCtn src={"img-productos/"+ e.Images[0].n} size={2}
                class={s1.product_card_image}
                types={["avif","webp"]}
              />
            }
            <div class={`flex jc-start w100 mt-08 ${s1.product_name_ctn}`}>
              <div class="grow-1">
                <div>{e.Nombre}</div>
                <div class="ff-bold h3">s/. {formatN(e.PrecioFinal,2)}</div>
              </div>
              <div class={`flex ai-center jc-center ${s1.product_cart_button}`}>
                <i class="icon-basket"></i>
                <div class={s1.product_cart_bn_text} 
                  onClick={ev => {
                    ev.stopPropagation()
                    addProducto(e)
                  }}
                >
                  Agregar <br /> al Carrito
                </div>
              </div>
            </div>
          </div>
        })
      }
    </div>
    <h2>Productos aqui = {productos().productos?.length}</h2>
    <h2>hola mundo</h2>
  </div>
}

export interface ICartProducto {
  producto: IProducto
  cant: number
}

const [cartProductos, setCartProductos] = createSignal([] as ICartProducto[])

export const addProducto = (producto: IProducto) => {

  const productos = cartProductos()
  productos.push({
    producto, cant: 1
  })

  console.log("productos 1", productos)
  setCartProductos([...productos])
}

export interface ICartFloating {

}

export function CartFloating(props: ICartFloating){

  const [isOpen, setIsOpen] = createSignal(0)

  return <Portal mount={document.body}>
    <div class={`p-rel ${s1.floating_cart_ctn}`}
      classList={{ [s1.floating_cart_ctn_open]: isOpen() === 1 }}
    >
      <div class={`p-rel ${s1.floating_cart_btn}`}
        onClick={ev => {
          ev.stopPropagation()
          setIsOpen(isOpen() ? 0 : 1)
        }}
      >
        <img class="w100 h100" src={parseSVG(isOpen() ? iconCancelSvg : iconCartSvg)}/>
      </div>
      <div class={`p-rel ${s1.floating_cart_content}`}>
        <div>
          <For each={cartProductos()}>
          {e => {
            console.log("rendering productos::", cartProductos())
            return <div>{e.producto.Nombre}</div>
          }}
          </For>
        </div>
        <div class={`flex-center ${s1.floating_cart_card}`}>
          <div class="ff-bold mr-08">Ir a Pagar -</div>
          <div class="ff-bold mr-08">S/. 45.00</div>
        </div>
      </div>
    </div>
  </Portal>

}