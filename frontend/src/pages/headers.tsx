import { createEffect, createMemo, createSignal, on } from "solid-js"
import { parseSVG } from "~/core/main"
import angleSvg from "../assets/angle.svg?raw"
import s1 from './components.module.css'
import { EcommerceCart } from "./cart"
import { getWindow } from "~/env"
import { ProductoSearchLayer } from "./productos"
import { IPageSection } from "./page-components"
import { deviceType } from "~/app"

export interface IHeader1 {
  args: IPageSection
}

export const [showCart, setShowCart] = createSignal(false)

export function Header1(props: IHeader1) { // type: 10

  let menuRef1: HTMLDivElement
  let menuRef2: HTMLDivElement
  let menuRefHeight = 0
  let eT: number, wH, apear, disapear // Variables
  const Window = getWindow()
  
  const updateVariables = () => {
    if(!menuRef1 || !menuRef2){ return }
    menuRefHeight = menuRef1.getBoundingClientRect().height
    // Distancia del margen superior del elemento hasta el Top
    eT = 0; let offsetParent = menuRef2
    while(offsetParent){
      eT += menuRef2.offsetTop
      offsetParent = offsetParent.offsetParent as HTMLDivElement
    }
    wH = Window.innerHeight // Altura de la pantalla
    // Distancia donde el elemento comienza a aparecer en el viewport
    apear = eT - wH
    // Distancia donde el elemento desaparece del viewport
    disapear = eT
  }

  const handleScroll = () => {
    if(!menuRef2){ return }
    if(typeof eT !== 'number' || !menuRefHeight){ updateVariables() }
    let wy = Window.scrollY - 15; // Posición del scroll
    console.log("haciendo scroll",menuRefHeight,Window.scrollY )
    if(Window.scrollY > (menuRefHeight + 8)){
      menuRef2.classList.add('menu-on')
    } else if( wy < eT){
      menuRef2.classList.remove('menu-on')
    }
  }

  Window.addEventListener("scroll", handleScroll)
  Window.addEventListener("resize", handleScroll)

  createEffect(() => {
    handleScroll()
  })

  createEffect(() => {
    console.log(props.args)
  })

  let divRef: HTMLDivElement

  const divParams = createMemo(on(
    () => [showCart()],
    () => {
      if(!divRef){ return {} }
      const rect = divRef.parentElement.getBoundingClientRect()
      const right = document.body.offsetWidth - rect.right
      const angleRight = Math.floor(right + (rect.width / 2))

      return { right, angleRight }
    }
  ))
  
  return <>
    { [1].includes(deviceType()) &&
      <div class="header1-menu1 flex" ref={menuRef1}>
        <div class="flex ai-center jc-between w100">
          <div>{props.args.Title}</div>
          <div>{props.args.Subtitle}</div>
        </div>
      </div> 
    }   
    { [1].includes(deviceType()) &&
      <div class="header1-menu2 flex jc-between ai-center w100" ref={menuRef2}>
        <div style={{ "min-width": "10rem" }}></div>
        <ProductoSearchLayer />
        <div class="flex p-rel h100 ai-center">
          <div class={`p-rel h100 flex ai-center jc-center ml-08 mr-08 ${s1.menu_cart_layer_btn_ctn}`}>
            <div class={`p-rel flex-center w100 h4 mt-02 ${s1.menu_cart_layer_btn}`}
              classList={{ [s1.menu_cart_layer_btn_selected]: !!showCart() }}
              ref={divRef}        
              onClick={ev => {
                ev.stopPropagation()
                if(showCart()){ 
                  setShowCart(false)
                } else {
                  setShowCart(true)
                }
              }}
            >
              <i class="icon-basket"></i>Carrito
            </div>
            { showCart() &&
              <div class={`${s1.menu_cart_layer}`} 
                style={{ right: `calc(1.2rem - ${divParams().right}px)`  }}> 
                <img class={`p-abs ${s1.menu_cart_layer_angle}`}
                  style={{ right: `calc(${divParams().angleRight}px - 1.2rem - 16px)`  }}
                  src={parseSVG(angleSvg)}
                />
                {/* https://codyhouse.co/demo/breadcrumbs-multi-steps-indicator/index.html#0 */ }
                <EcommerceCart />
              </div>
            }
          </div>
          <div class="p-rel flex h4 ai-center mt-04 ml-08 mr-08">
            <i class="icon-user"></i>Usuario
          </div>
        </div>
      </div>
    }
    { ![1].includes(deviceType()) &&
      <div class={`${s1.top_bar_mobile} flex ai-center mr-08`}>
        <div class={s1.top_bar_mobile_icon}>
          <i class="icon-menu"></i>
        </div>
        <ProductoSearchLayer />
        <div class={`ml-auto ${s1.top_bar_mobile_icon}`}>
          <i class="icon-basket"></i>
        </div>
      </div>
    }
  </>
}
