import { useZoomImageMove } from "@zoom-image/solid"
import { createEffect, createMemo, createSignal, For, on, onMount, Show } from "solid-js"
import { Portal } from "solid-js/web"
import { ImageCtn } from "~/components/Uploaders"
import { parseSVG, throttle } from "~/core/main"
import { IProducto } from "~/services/operaciones/productos"
import { formatN, makeEcommerceDB } from "~/shared/main"
import iconCancelSvg from "../assets/icon_cancel.svg?raw"
import iconCartSvg from "../assets/icon_cart.svg?raw"
import { setCartOption } from "./cart"
import s1 from "./components.module.css"
import { IHeader1, setShowCart, showCart } from "./headers"
import { useProductosCmsAPI } from "./productos-service"

export function ProductosCuadrilla(props: IHeader1) { // type: 10

  const [ productos ] = useProductosCmsAPI()

  return <div class={"w100 " + s1.product_cuadrilla_ctn}>
    <div class="w100 flex-wrap ai-baseline jc-center">
      { productos().isFetching && <div>Obteniendo productos...</div> }
      { (productos().productos||[]).map(e => {
          const cartCant = createMemo(() => {
            return cartProductos().get(e.ID)?.cant || 0
          })

          return <div class={`p-rel flex-column ai-center ${s1.product_card}`}
            classList={{ "has-products": cartCant() > 0 }}
            onClick={ev => {
              ev.stopPropagation()
              setProductoSelected(e)
            }}
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
              <Show when={cartCant() < e._stock}>
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
              </Show>
              <Show when={cartCant() >= e._stock}>
                <div class={`flex ai-center jc-center ${s1.product_cart_button}`}
                  onClick={ev => { ev.stopPropagation() }}
                >
                  <i class="icon-attention _icon"></i>
                  <div class={s1.product_cart_bn_cant}>{cartCant()}</div>
                  <div class={s1.product_cart_bn_text}>Stock Máximo = <span>{cartCant()}</span></div>     
                </div>
              </Show> 
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

export const [cartProductos, setCartProductos] = createSignal(new Map() as Map<number,ICartProducto>)

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
    <Show when={!showCart()}>
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
            <div class={`flex-center ${s1.floating_cart_card}`} onClick={ev => {
              ev.stopPropagation()
              setShowCart(true)
              setIsOpen(0)
              setCartOption(2)
            }}>
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
    </Show>
  </Portal>
}

export interface ICartFloatingProducto{
  producto: IProducto
  cant: number
  mode?: 1 | 2
}

const [productWarnMessage, setProductWarnMessage] = createSignal({ id: 0, msg: "" })

const clearProductoWarn = () => {
  if(productWarnMessage().id > 0){
    setProductWarnMessage({ id: 0, msg: "" })
  }
}

let prevCartSize = 0
createEffect(() => {
  if(prevCartSize !== cartProductos().size){
    prevCartSize = cartProductos().size
    setProductWarnMessage({ id: 0, msg: "" }) 
  }
})

export interface ICartProductoCache {
  id: number, cant: number
}

let isEmptyInit = true

createEffect(() => {
  const productosMap = cartProductos()
  if(isEmptyInit && productosMap.size === 0){ return }
  isEmptyInit = false

  throttle(async () => {
    const record = {
      key: "cart",
      productos: [...productosMap.values()].map(x => {
        return { id: x.producto.ID, cant: x.cant }
      }),
    }
    const db = await makeEcommerceDB()
    if(!db){ return }
    db.table("cache").put(record)
  },500)
})

export const CartFloatingProducto = (props: ICartFloatingProducto) => {

  const message = createMemo(() => {
    if(productWarnMessage().id === props.producto.ID){
      return productWarnMessage().msg
    }
  })

  const precio = createMemo(on(
    () => [ cartProductos() ], 
    () => { return props.cant * props.producto.PrecioFinal }
  ))

  const cant = createMemo(on(
    () => [ cartProductos() ], 
    () => { return props.cant }
  ))

  const add = (newCant: number) => {
    if(newCant <= 0){ return }
    const current = cartProductos().get(props.producto.ID)
    current.cant = newCant
    setCartProductos(new Map(cartProductos()))
  }

  return <div class="p-rel w100"
      classList={{ 
        "mb-10": (props.mode||1) === 1, 
        "mb-12": (props.mode||1) === 2 
      }}
    >
    <div class={`flex w100 ${s1.floating_producto_card} p-rel`}
        classList={{ "s2": props.mode === 2 }}
      >
      <div class={`h100 ${s1.floating_producto_card_img}`}>
        <ImageCtn src={"img-productos/"+ props.producto.Images[0]?.n} size={2}
          class={"h100 w100 object-cover"}
          types={["avif","webp"]}
        />
      </div>
      <div class={`p-rel w100 flex-column ml-06 ${s1.producto_card_ctn}`}>
        <div>{props.producto.Nombre}</div>
        <div class={`p-abs flex-center ${s1.producto_card_btn_del}`} onClick={ev => {
          ev.stopPropagation()
          clearProductoWarn()
          cartProductos().delete(props.producto.ID)
          setCartProductos(new Map(cartProductos()))
        }}>
          <i class="icon-cancel"></i>
        </div>
        <div class={`p-abs h3 ff-semibold ${s1.producto_card_price}`}>
          s./ {formatN(precio()/100, 2)}
        </div>
        <div class={`p-abs flex ai-center ff-semibold ${s1.producto_card_btn_ctn}`}>
          <button onClick={ev => {
            ev.stopPropagation(); add(props.cant - 1); clearProductoWarn()
          }} class={`${s1.producto_card_btn_cant}`}>-</button>
          <input type="number" value={cant()} 
            class={`mr-02 ml-02 ${s1.producto_card_btn_input}`}
            onChange={ev => {
              let newValue = parseInt(ev.target.value||"0")
              if(newValue > props.producto._stock){
                newValue = props.producto._stock
                ev.target.value = newValue as unknown as string
                setProductWarnMessage({ 
                  id: props.producto.ID,
                  msg: `El stock máximo es ${props.producto._stock}`
                })
                add(newValue)
              } else if(newValue > 0){
                add(newValue); clearProductoWarn()
              } else {
                ev.target.value = String(props.cant)
              }
            }}
          />
          { cant() < props.producto._stock &&
            <button onClick={ev => {
              ev.stopPropagation(); add(props.cant + 1); clearProductoWarn()
            }} class={`${s1.producto_card_btn_cant}`}>+</button>
          }
        </div>
      </div>
    </div>
    <Show when={!!message()}>
      <div class="c-red h5 mt-02"><i class="icon-attention"></i> {message()}</div>
    </Show>
  </div>
}

export const [productoSelected, setProductoSelected] = createSignal(null as IProducto)

export interface IProductoInfoLayer {

}

export const ProductoInfoLayer = (props: IProductoInfoLayer) => {

  let divRef: HTMLDivElement
  const { createZoomImage } = useZoomImageMove()

  onMount(() => {
    console.log("div container::", divRef)
    createZoomImage(divRef, { zoomFactor: 2 })
  })

  return <Portal mount={document.body}>
    <div class={`${s1.producto_layer_bg}`} onClick={ev => {
      ev.stopPropagation()
      setProductoSelected(null)
    }}></div>
    <div class={`${s1.producto_layer_ctn}`}>
      <div class="flex h1 ff-semibold mb-08">
        {productoSelected().Nombre}
      </div>
      <div class="flex w100">
        <div class={`p-rel ${s1.producto_layer_img_ctn}`}>
        { productoSelected().Images.map(img => {
            return <div class={`p-rel ${s1.producto_layer_img_min} mb-08`}>
              <ImageCtn src={"img-productos/"+ img.n} size={2}
                class={"h100 w100 object-contain"}
                types={["avif","webp"]}
              />
            </div>
          })
        }
        </div>
        <div class={`${s1.producto_layer_content}`}>
          <div class={`p-rel w100 ${s1.producto_layer_img1}`} ref={divRef}>
            <ImageCtn src={"img-productos/"+ productoSelected().Images[0]?.n} size={6}
              class={"h100 w100 object-contain"}
              types={["avif","webp"]}
            />
          </div>
          <div class="flex mt-12 ai-center">
            <div class="mr-06">Precio:</div>
            <div class="ff-bold h1">s/. {formatN(productoSelected()?.PrecioFinal/100,2)}</div>
            <div class="ml-auto"></div>
            <button class={`${s1.producto_cart_btn}`}>
              <i class="icon-basket"></i>
              Añadir a Carrito
            </button>
          </div>
        </div>
      </div>
    </div>
  </Portal>
}