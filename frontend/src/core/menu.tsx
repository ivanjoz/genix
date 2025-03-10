import { For, Show, createMemo, createSignal } from "solid-js"
import { appModule, deviceType, setShowMenu, showMenu } from "~/app"
import { ButtonList } from "~/components/Cards"
import { LayerSelect } from "~/components/Layers"
import { Env, LocalStorage, getWindow } from "~/env"
import { fetchOnCourse } from "~/shared/http"
import { Params } from "~/shared/security"
import { IModule } from "./modules"

export interface IMenuRecord {
  name: string, minName?: string, id?: number, route?: string,
  options?: IMenuRecord[],   icon?: string
}

export const [innerPageName, setInnerPageName] = createSignal("")
export const [pageViews, setPageViews] = createSignal([])
export const [pageView, setPageView] = createSignal(0)
export const [isRouteChanging, setIsRouteChanging] = createSignal(false)

const uicolors = [
  { id: "light", name: "Claro" },
  { id: "dark", name: "Oscuro" }
]

export function MainTopMenu() {
  
  return <div class="main-header flex ai-center">
    <Show when={[1].includes(deviceType())}>
      <div class="logo-ctn2 h100 mr-12">
        <div class="h100 w100">
          <img class="w100 h100" src="/images/genix_logo4.svg" alt="" />
        </div>
      </div>
        {/*
         <div class="header-program">
        <SearchSelect selected={appModule().id} options={Modules} keys="id.name"
          css="menu-s1" notEmpty={true}
          onChange={mod => {
            Params.setValue("moduleSelected",mod.id)
            if(mod){ setAppModule(mod) }
            return
          }}
        />
              </div>
        */}
      { /*
      <div class="square-m1 mr-16 flex-center" onClick={ev => {
        ev.stopPropagation()
        document.body.classList.add("is-animated")
        setTimeout(() => document.body.classList.remove("is-animated"),300)
        if(viewType() === 1){ setViewType(2) } else { setViewType(1) }
      }}>
        <i class="icon-left-open" classList={{'rotatey-180g': viewType() === 2 }}></i>
      </div>
      */}
    </Show>
    <Show when={(pageViews()||[]).length === 0}>
      <div class="h3 h100 flex ai-center c-white title">
        {innerPageName()||"-"}
      </div>
    </Show>
    <Show when={(pageViews()||[]).length > 0 && [1,2].includes(deviceType())}>
      <div class="flex ai-center h100 title bn-c1c">
        <For each={pageViews()}>
          {e => {
            const cN =  () => {
              let cN_ = "h3 bn-c1 flex-center p-rel"
              if(e[0] === pageView()){ cN_ += " selected" }
              return cN_
            }
            return <div class={cN()} onClick={ev => {
              ev.stopPropagation()
              Params.setValue(`pview-${Env.getPathname()}`,e[0])
              setPageView(e[0])
            }}>
              <span>{e[1]}</span>
            </div>
          }}
        </For>
      </div>
    </Show>
    <Show when={(pageViews()||[]).length > 0 && [3].includes(deviceType())}>
      <SearchCard 
        options={pageViews()}
        selected={pageView()}
        onChange={(id) => { setPageView(id) }}
      />
    </Show>
    <Show when={[1,2].includes(deviceType())}>
      <MenuSearchTop />
    </Show>
    <div class="ml-auto flex ai-center h100"
      classList={{
        'mr-16': [1,2].includes(deviceType()),
        'mr-08': [3].includes(deviceType()) 
      }}
    >
      { fetchOnCourse().size > 0 &&
        <div class="pm-loading mr-06">
          <div class="bg"></div>
          <span>{"Cargando..."}</span>
        </div>
      }
      <Show when={[1].includes(deviceType())}>
        <LayerSelect buttonClass="bnr-4" containerClass="mr-08"
          layerClass="px-08 py-08" layerStyle={{ width: '12rem' }}
          icon={<i class="icon-cog"></i>}
        >
          <button class="bn1" onClick={ev => {
            ev.stopPropagation()
            Env.clearAccesos()
          }}>
            <i class="icon-logout-1"></i>Salir
          </button>
          <div class="w100"></div>
          <div>Tema</div>
          <ButtonList options={uicolors}
            keys="id.name" selected={LocalStorage.getItem("ui-color") as any}
            onClick={e => {
              for(let x of uicolors){ document.body.classList.remove(x.id) }
              document.body.classList.add(e.id)
              LocalStorage.setItem("ui-color",e.id)
            }}
          />
        </LayerSelect>
      </Show>
      <button class="bnr-4" onClick={ev => {
          ev.stopPropagation()
          const now5secodsMore = Math.floor((Date.now() / 1000)) + 5
          LocalStorage.setItem("force_sync_cache_until", String(now5secodsMore))
          window.location.reload()
        }}>
        <i class="icon-arrows-cw"></i>
      </button>
      <Show when={[2,3].includes(deviceType())}>
        <button class="h1 bn-m1 ml-10" onClick={ev => {
          ev.stopPropagation()
          setShowMenu(true)
        }}>
          <i class="icon-menu"></i>
        </button>
      </Show>
    </div>
  </div>
}

interface ISearchCard {
  options: [number, string][]
  selected: number
  onChange: ((id: number) => void)
}

export function SearchCard(props: ISearchCard) {

  const [show, setShow] = createSignal(false)

  const selected = () => {
    return props.options.find(x => x[0] === props.selected) || [0,""] as [number, string]
  }

  return <div class="flex ai-center card-16 ml-04 p-rel" onClick={ev => {
    ev.stopPropagation()
    setShow(!show())
  }}>
    <div class="flex card-16e ai-center jc-between w100"
      classList={{ 'open': show() }}
    >
      <div class="h3 c-white mr-02">{selected()[1] as string || "-"}</div>
      <div class="icon-c1 icon-down-open-1 c-white"></div>
    </div>
    <Show when={show()}>
      <div class="p-abs layer-c2 layer-angle2 px-08 py-08"
        style={{ left: '0.6rem', top: 'calc(100% - 2px)' }}
      >
        { props.options.map(e => {
            return <div class="bn1 c-steel ff-bold b-gray mt-02 mb-02 w100" 
              classList={{ "sel-1": props.selected === e[0] }}
              onClick={ev => {
                ev.stopPropagation()
                props.onChange(e[0])
                console.log("cambiando a:: ", e[0])
                setShow(false)
              }}
            >
              { e[1] }
            </div>
          })
        }
      </div>
    </Show>
  </div>
}

export function MainMenu() {
  const pathname = Env.getPathname()

  const getMenuOpenFromRoute = (module: IModule): [number, string] => {
    for(let menu of module.menus){
      for(let opt of menu.options){
        if(opt.route === pathname){ return [menu.id, opt.route] }
      }
    }
    return [0,""]
  }

  const [menuOpen, setMenuOpen] = createSignal(getMenuOpenFromRoute(appModule()))
  const [isMenuHover, setIsMenuHover] = createSignal(false)
  const [mode, setMode] = createSignal(1)

  const mouseEnterHandler = (idx: number) =>{

  }

  return <div class="main-menu-c">
    <div class="main-menu">
      <div class="main-ctn" style={{ height: '3rem' }}>
        
      </div>
      {/*
      <div class="logo-ctn">
        <img class="w100 h100" src="/images/genix_logo_w.svg" alt="" />
      </div>
      */}
      <div class={"menu-main-c1"}
        onMouseEnter={() => mouseEnterHandler(1)}
        onMouseLeave={() => mouseEnterHandler(0)}
        style={{ overflow: isMenuHover() ? "auto" : "hidden" }}
      >
        <For each={appModule().menus}>
          {(menu) => 
            <MenuElement menu={menu} menuOpen={menuOpen()} 
              setMenuOpen={setMenuOpen}
              mode={mode()} 
            />
          }
        </For>
      </div>
    </div>
  </div>
}

interface IMenuElement {
  menuOpen: [number, string]
  mode: number
  menu: IMenuRecord
  setMenuOpen: (v: [number, string]) => void
}

export function MenuElement(props: IMenuElement) {

  const isOpen = createMemo(() => {
    return props.menuOpen[0] === props.menu.id
  })

  const height = createMemo(() => {
    const isOpen = props.menuOpen[0] === props.menu.id
    let _height = props.mode === 3 ? 2.7 : 3
    if(isOpen){ 
      if(props.mode === 3){
        _height += Math.ceil(props.menu.options.length) * 3.1  
      } else {
        _height += props.menu.options.length * 2.8  
      }
    }
    return _height
  })

  const routeSelected = createMemo(() => props.menuOpen[1])

  return <div class={"menus-c1" + (isOpen() ? " open" : "")} 
    style={{ "max-height": `${height()}rem` }}>
    <div class={"menu-main-label w100 p-rel flex ai-center"}
      onClick={() => {
        console.log("seteando menu open:: ", props.menu.id)
        props.menuOpen[0] = isOpen() ? 0 : props.menu.id
        props.setMenuOpen([...props.menuOpen])
      }}
    >
      <div class="flex ai-center w100">
        {/* 
        <div class="flex ai-center lh-10" style={{ width: '1.5rem' }}>
          <i class={props.menu.icon}></i>
        </div>
        */}
        <div class="m-label ff-mono500 ml-06">{props.menu.name.toUpperCase()}</div> 
        <div class="m-label-min ff-mono500 mr-auto ml-06">
          {props.menu.minName.toUpperCase()}
        </div> 
      </div>
      <span class="icon-c1">
        <i class="icon-down-open-1"></i>
      </span>
    </div>
    { [1].includes(props.mode) && 
      props.menu.options.map(opt => {
        const isSelected = opt.route && opt.route === routeSelected()
        return MakeMenuRecord(props, opt, isSelected)
      })
    }
    { [2,3].includes(props.mode) && 
      props.menu.options.map((opt,i) => {
        if(i % 2 !== 0){ return null }
        const options = [opt]
        if(props.menu.options[i + 1]){ options.push(props.menu.options[i + 1]) }

        return <div class="flex jc-between opt-m w100">
          { options.map(e => {
              const isSelected = e.route && e.route === routeSelected()
              return MakeMenuRecord(props, e, isSelected)
            })
          }
        </div>
      })
    }
  </div>
}

const MakeMenuRecord = (props: IMenuElement, opt: IMenuRecord, selected?: boolean) => {
  const pathname = Env.getPathname()
  const Window = getWindow()
  let cN = `menu-option-c1`
  if(selected) cN += ' selected'

  return <a style={{ "text-decoration": 'none', display: 'contents' }} 
    href={opt.route||"/"}>
    <div class={cN} onClick={ev => {
      const route = opt.route||"/"
      if(pathname === route){ return }
      ev.stopPropagation()
      Window.history.pushState(null, opt.name, opt.route||"/")
      setIsRouteChanging(true)
      props.menuOpen[1] = opt.route
      props.setMenuOpen([...props.menuOpen])
      if(showMenu()){ setShowMenu(false) }
    }}>
      { props.mode === 2 &&
        <div class="submenu-label-min">
          <span class="flex min-menu">
            <span>{opt.minName}</span>
          </span>
        </div>
      }
      <div class="submenu-label">
        <i class={(opt.icon || "icon-cube") + " h3 mr-01"}></i>
        <span class="mn-1 ff-mono" style={{ "margin-bottom": '-2px' }}>
        { opt.name.split(" ").map(x => {
            return <span class="mr-04">{x}</span>
          })
        }
        </span>
        <span class="mn-2 ff-mono500" style={{ "margin-bottom": '-2px' }}>{opt.name.substring(0,3).trim()}</span>
      </div>
    </div>
  </a>
}

interface MenuSearchTop {

}

export function MenuSearchTop(props: MenuSearchTop) {

  const [menuOpen, setMenuOpen] = createSignal({})

  return <div class="menu-search flex ai-center p-rel">
    <i class="icon-search icon-1"></i>
    <input type="text" />
  </div>
}

/* MOBILE */

export function MainMenuMobile() {

  const getMenuOpenFromRoute = (module: IModule): [number, string] => {
    const pathname = Env.getPathname()
    for(let menu of module.menus){
      for(let opt of menu.options){
        if(opt.route === pathname){ return [menu.id, opt.route] }
      }
    }
    return [0,""]
  }

  const [menuOpen, setMenuOpen] = createSignal(getMenuOpenFromRoute(appModule()))
  const [isMenuHover, setIsMenuHover] = createSignal(false)

  const mouseEnterHandler = (idx: number) =>{

  }

  return <div class="main-menu-mob">
    <div class="menu-bar w100 flex jc-between">
      <div></div>
      <div class="flex ai-center h100">
        <button class="h1 bn-m1 mr-04" onClick={ev => {
          ev.stopPropagation()
          setShowMenu(false)
        }}>
          <i class="icon-cancel"></i>
        </button>
      </div>
    </div>
    <div class="menu-c2 w100 h100">
      <For each={appModule().menus}>
        {(menu) => 
          <MenuElement menu={menu} menuOpen={menuOpen()} 
            setMenuOpen={setMenuOpen}
            mode={3}
          />
        }
      </For>
    </div>
  </div>
}

export default function PageMenu(){
  
  return <>
    <Show when={[1].includes(deviceType())}><MainMenu/></Show>
    <Show when={[2,3].includes(deviceType()) && showMenu()}>
      <MainMenuMobile/>
    </Show>
    <MainTopMenu />
  </>
}