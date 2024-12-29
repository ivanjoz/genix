import { createComputed, createEffect, createMemo, createSignal, For, on } from "solid-js"
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
    // console.log("icon cart svg::", iconCartSvg)
    console.log("productos obtenidos component::", productos())
  })

  return <div class={"w100 " + s1.product_cuadrilla_ctn}>
    <div class="w100 flex-wrap ai-baseline jc-center">
      { productos().isFetching && <div>Obteniendo productos...</div> }
      { (productos().productos||[]).map(e => {
          const cartCant = createMemo(() => {
            return cartProductos().get(e.ID)?.cant || 0
          })

          return <div class={`p-rel flex-column ai-center ${s1.product_card}`}
            classList={{ "has-products": cartCant() > 0 }}
          >
            { e.Images?.length > 0 &&
              <ImageCtn src={"img-productos/"+ e.Images[0]?.n} size={2}
                class={s1.product_card_image}
                types={["avif","webp"]}
              />
            }
            <div class={`flex jc-start w100 mt-04 ${s1.product_name_ctn}`}>
              <div class="grow-1">
                <div>{e.Nombre}</div>
                <div class="ff-bold h3">s/. {formatN(e.PrecioFinal/100,2)}</div>
              </div>
              <div class={`flex ai-center jc-center ${s1.product_cart_button}`}
                onClick={ev => {
                  ev.stopPropagation()
                  addProducto(e)
                }}
              >
                <i class="icon-basket _icon"></i>
                <div class={s1.product_cart_bn_cant}>{cartCant()}</div>
                { cartCant() > 0
                  ? <div class={s1.product_cart_bn_text}>
                      Añadir más <span>({cartCant()})</span>
                    </div>
                  : <div class={s1.product_cart_bn_text}>Añadir al Carrito</div>
                }                
              </div>
            </div>
          </div>
        })
      }
    </div>
  </div>
}

export interface ICartProducto {
  producto: IProducto
  cant: number
}

const [cartProductos, setCartProductos] = createSignal(new Map() as Map<number,ICartProducto>)

export const addProducto = (producto: IProducto) => {

  const productosMap = cartProductos()
  const current = productosMap.get(producto.ID)
  if(current){
    current.cant ++
  } else {
    productosMap.set(producto.ID,{ producto, cant: 1 })
  }

  console.log("productos 1", productosMap)
  setCartProductos(new Map(productosMap))
}

export interface ICartFloating {

}

export const CartFloating = (props: ICartFloating) => {

  const [isOpen, setIsOpen] = createSignal(0)
  const precio = createMemo(() => {
    let precio = 0
    for(const pr of cartProductos().values()){
      precio += (pr.cant * pr.producto.PrecioFinal)
    }
    return precio
  })

  return <Portal mount={document.body}>
    <div class={`p-rel flex-center ${s1.floating_cart_ctn_ref}`}>
      <div class={`p-rel h100 w100 ${s1.floating_cart_ctn}`}
        classList={{ [s1.floating_cart_ctn_open]: isOpen() === 1 }}
      >
        <div class={`flex-center p-rel ${s1.floating_cart_btn}`}
          classList={{ "has-productos": cartProductos().size > 0 }}
          onClick={ev => {
            ev.stopPropagation()
            setIsOpen(isOpen() ? 0 : 1)
          }}
        >
          <img class="" 
            src={parseSVG(isOpen() ? iconCancelSvg : iconCartSvg)}
          />
        </div>
        <div class={`p-rel ${s1.floating_cart_content}`}>
          <div class="h100 w100">
            <For each={Array.from(cartProductos().values())}>
            {e => {
              console.log("rendering productos::", cartProductos())
              return <CartFloatingProducto producto={e.producto} cant={e.cant} />
            }}
            </For>
          </div>
          <div class={`flex-center ${s1.floating_cart_card}`}>
            <div class="ff-bold-italic mr-08">Ir a Pagar -</div>
            <div class="ff-bold-italic mr-08">S/. {formatN(precio()/100,2)}</div>
          </div>
        </div>
      </div>
      { cartProductos().size > 0 && !isOpen() &&
        <div class={`p-abs ff-semibold flex-center ${s1.floating_cart_count}`}>
          {cartProductos().size}
        </div>
      }
    </div>
  </Portal>
}

export interface ICartFloatingProducto{
  producto: IProducto
  cant: number
}

export const CartFloatingProducto = (props: ICartFloatingProducto) => {
  const precio = createMemo(on(
    () => [ cartProductos() ], 
    () => { return props.cant * props.producto.PrecioFinal }
  ))

  const cant =  createMemo(on(
    () => [ cartProductos() ], 
    () => { return props.cant }
  ))

  const add = (newCant: number) => {
    if(newCant <= 0){ return }
    const current = cartProductos().get(props.producto.ID)
    current.cant = newCant
    setCartProductos(new Map(cartProductos()))
  }

  return <div class={`flex w100 ${s1.floating_producto_card} mb-08 p-rel`}>
    <div class={`h100 ${s1.floating_producto_card_img}`}>
      <ImageCtn src={"img-productos/"+ props.producto.Images[0]?.n} size={2}
        class={"h100 w100 object-cover"}
        types={["avif","webp"]}
      />
    </div>
    <div class={`p-rel w100 flex-column ml-08 ${s1.producto_card_ctn}`}>
      <div>{props.producto.Nombre}</div>
      <div class={`p-abs flex-center ${s1.producto_card_btn_del}`} onClick={ev => {
        ev.stopPropagation()
        cartProductos().delete(props.producto.ID)
        setCartProductos(new Map(cartProductos()))
      }}>
        <i class="icon-trash"></i>
      </div>
      <div class={`p-abs h3 ff-semibold ${s1.producto_card_price}`}>
        s./ {formatN(precio()/100, 2)}
      </div>
      <div class={`p-abs flex ai-center ff-semibold ${s1.producto_card_bnt_cant}`}>
        <button onClick={ev => {
          ev.stopPropagation(); add(props.cant - 1)
        }}>-</button>
        <input type="number" value={cant()} 
          onChange={ev => {
            const newValue = parseInt(ev.target.value||"0")
            if(newValue > 0){
              add(newValue)
            } else {
              ev.target.value = String(props.cant)
            }
          }}
        />
        <button onClick={ev => {
          ev.stopPropagation(); add(props.cant + 1)
        }}>+</button>
      </div>
    </div>
  </div>
}