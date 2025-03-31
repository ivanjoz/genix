import { createEffect, createMemo, createSignal, JSXElement, on, Show } from "solid-js"
import { parseSVG } from "~/core/main"
import angleSvg from "../assets/angle.svg?raw"
import s1 from './components.module.css'
import { EcommerceCart } from "./cart"
import { Env, getWindow } from "~/env"
import { cartProductos, ProductoSearchLayer } from "./productos"
import { IPageSection } from "./page-components"
import { deviceType } from "~/app"
import { Portal } from "solid-js/web"
import { MobileLayerForm, setOpenMobileLayer } from "~/components/Modals"
import { FlipButton } from "~/components/Cards"

export interface IHeader1 {
  args: IPageSection
}

export const [showCart, setShowCart] = createSignal(false)

createEffect(() => {
  if(showCart() === true){
    Env.suscribeUrlFlag("mob-pcart", ()=>{ setShowCart(false) })
  }
})

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

  const [prerenderCart, setPrerenderCart] = createSignal(false)
  createEffect(() => {
    if(!showCart()){ setPrerenderCart(false) }
  })

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
              <div id="mob-pcart" class={`${s1.menu_cart_layer}`} 
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
        <div class={s1.top_bar_mobile_icon} 
          onClick={ev => {
            ev.stopPropagation()
            openMobileSideMenu(true)
          }}
        >
          <i class="icon-menu"></i>
        </div>
        <ProductoSearchLayer />
        <FlipButton class={`ml-auto flex-center ${s1.top_bar_mobile_icon}`}
          isFliped={showCart()}
          front={<>
            { cartProductos().size > 0 &&
              <div class={`p-abs flex-center ${s1.top_bar_mobile_icon_counter}`}
                classList={{[s1.disabled]: showMobileSideMenu() > 0}} 
              >{cartProductos().size}</div>
            }
            <i class="icon-basket"
              style={{ 
                "margin-bottom": cartProductos().size > 0 ? "-6px" : undefined,
                "margin-left": cartProductos().size > 0 ? "-4px" : undefined    
              }}
            ></i>
          </>}
          back={
            <div class={`ml-auto flex-center ${s1.top_bar_mobile_icon}`}
              onClick={ev => {
                ev.stopPropagation()
                setShowCart(false)
              }}
            >
              <div class={`flex-center ml-04 ${s1.top_bar_cart_icon_cancel}`}>
                <i class="icon-cancel"></i>
              </div>
            </div>
          }
          onMouseDown={() => {
            if(!showCart()){
              setPrerenderCart(true)
              setTimeout(() => { if(!showCart()){ setPrerenderCart(false) } },1500)
            }
          }}
          onClick={() => {
            if(showCart()){
              setShowCart(false)
            } else {
              setShowCart(true)
              setPrerenderCart(false)
              if(Env.closeProductosSearchLayer){ Env.closeProductosSearchLayer() }
            }
          }}
        />
        { (showCart() || prerenderCart() ) &&
          <div id="mob-pcart" class={`${s1.menu_cart_layer}`}
            classList={{[s1.is_hidden]: !showCart() }}
            style={{ 
              right: `calc(1.2rem - ${divParams().right}px)`, 
            }}> 
            <img class={`p-abs ${s1.menu_cart_layer_angle}`}
              style={{ right: `1.1rem`  }} src={parseSVG(angleSvg)}
            />
            <EcommerceCart />
          </div>
        }
      </div>
    }
    <MobileSideMenu />
  </>
}

export interface IMobileSideMenu {

}

export const [showMobileSideMenu, setShowMobileSideMenu] = createSignal(0)

export const openMobileSideMenu = (open?: boolean) => {
  if(open){
    Env.suscribeUrlFlag("mob-menu", () => openMobileSideMenu(false) )
  }

  if(open && showMobileSideMenu() === 1){
    setShowMobileSideMenu(2)
  } else if(open && !showMobileSideMenu()){
    setShowMobileSideMenu(2)
  } else if(!open){
    setShowMobileSideMenu(3)
    setTimeout(() => { setShowMobileSideMenu(0) },340)
  }
}

const mainMenuOptinos = [
  { name: "Iniciar Sesión",
    icon: "icon-user",
    onClick: () => { setOpenMobileLayer(1) }
  },
  { name: "Regístrate",
    icon: "icon-doc",
  },
  { name: "Mis Pedidos",
    icon: "icon-box",
  },
  { name: "Tienda",
    icon: "icon-home",
  }
]

export const MobileSideMenu  = (props: IMobileSideMenu) => {

  createEffect(() => {
    if(showMobileSideMenu() === 1){
      setShowMobileSideMenu(2)
    }
  })

  return <>
    <div class={`${s1.mobile_side_menu_ctn}`}
      classList={{
        [s1.is_open]: showMobileSideMenu() === 2,
        [s1.is_closing]: showMobileSideMenu() === 3
      }}
    >
    <Show when={[2,3].includes(showMobileSideMenu())}>
      <button class={`p-abs ${s1.mobile_side_btn_close}`}
        onClick={ev => {
          ev.stopPropagation()
          openMobileSideMenu(false)
        }}
      >
        <i class="h3 icon-cancel"></i>
      </button>
      <div class="ff-semibold h2 flex a-center ml-04">
        <div>Hola!</div>
      </div>
      <div class="mb-20"></div>
      <div id="mob-menu" class={`grid ${s1.mobile_side_menu_options_ctn}`}>
      { mainMenuOptinos.map(e => {
          return <div class={`p-rel items-center flex flex-column ${s1.mobile_side_menu_btn}`}
            onClick={ev => {
              ev.stopPropagation()
              if(e.onClick){ e.onClick() }
            }}
          >
            <div class={`${s1.mobile_side_menu_btn_top}`}>
              
            </div>
            <div class={`${s1.mobile_side_menu_icon_ctn}`}>
              { e.icon && <i class={"h2 " + e.icon}></i> }
            </div>
            <div class={`ff-semibold flex items-center grow ${s1.mobile_side_menu_name}`}>
              {e.name}
            </div>
          </div>
        })
      }
      </div>
    </Show>
    </div>
    <div class={`${s1.mobile_side_menu_background}`}
      classList={{
        [s1.is_open]: showMobileSideMenu() === 2,
        [s1.is_closing]: showMobileSideMenu() === 3 
      }}
      onClick={ev => {
        ev.stopPropagation()
        openMobileSideMenu(false)
      }}
    >

    </div>
    <MobileLayerForm id={1} title="Iniciar Sesión">
      <div>hola</div>
      <div>dasd</div>
    </MobileLayerForm>
  </>
}

