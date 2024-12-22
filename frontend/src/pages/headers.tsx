import { createEffect } from "solid-js"
import { IPageSection, ISectionParams } from "./page"

export interface IHeader1 {
  args: IPageSection
}

export function Header1(props: IHeader1) { // type: 10

  let menuRef1: HTMLDivElement
  let menuRef2: HTMLDivElement
  let menuRefHeight = 0
  let eT: number, wH, apear, disapear // Variables
    
  const updateVariables = () => {
    menuRefHeight = menuRef1.getBoundingClientRect().height
    // Distancia del margen superior del elemento hasta el Top
    eT = 0; let offsetParent = menuRef2
    while(offsetParent){
      eT += menuRef2.offsetTop
      offsetParent = offsetParent.offsetParent as HTMLDivElement
    }
    wH = window.innerHeight // Altura de la pantalla
    // Distancia donde el elemento comienza a aparecer en el viewport
    apear = eT - wH
    // Distancia donde el elemento desaparece del viewport
    disapear = eT
  }

  const handleScroll = () => {
    if(typeof eT !== 'number' || !menuRefHeight){ updateVariables() }
    let wy = window.scrollY - 15; // Posición del scroll
    console.log("haciendo scroll",menuRefHeight,window.scrollY )
    if(window.scrollY > (menuRefHeight + 8)){
      menuRef2.classList.add('menu-on')
    } else if( wy < eT){
      menuRef2.classList.remove('menu-on')
    }
  }

  window.addEventListener("scroll", handleScroll)
  window.addEventListener("resize", handleScroll)

  createEffect(() => {
    handleScroll()
  })

  createEffect(() => {
    console.log(props.args)
  })

  return <>
    <div class="header1-menu1 flex" ref={menuRef1}>
      <div class="flex ai-center jc-between w100">
        <div>{props.args.Title}</div>
        <div>{props.args.Subtitle}</div>
      </div>
    </div>    
    <div class="header1-menu2 flex w100" ref={menuRef2}>

    </div>
  </>
}
