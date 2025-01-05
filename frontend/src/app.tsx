import { Meta, MetaProvider, Title } from "@solidjs/meta";
import { Route, Router, useNavigate } from "@solidjs/router";
import { clientOnly } from "@solidjs/start";
import { FileRoutes } from "@solidjs/start/router";
import { Suspense, createEffect, createSignal, onMount } from "solid-js";
import Modules from "./core/modules";
import { PageLoading, PageLoadingElement } from "./core/page";
import { Env, LocalStorage, getWindow } from "./env";
import PageBuilder from "./pages/page";
import LoginPage from "./routes/login";
import { createIndexDB } from "./shared/main";
import { checkIsLogin, Params } from "./shared/security";

const PageMenu = clientOnly(() => import("./core/menu"))

let defaultModule = Modules[0]
const isClient = typeof window !== 'undefined'
if(isClient){
  const moduleSelected = Params.getValueInt("moduleSelected")
  if(moduleSelected){ defaultModule = Modules.find(x => x.id === moduleSelected) }
}
export const [appModule, setAppModule] = createSignal(defaultModule)
export const [showMenu, setShowMenu] = createSignal(false)

if(isClient){
  document.head.querySelector('title')?.remove()

  createIndexDB(Modules)
}

export const checkDevice = () => {
  const Window = getWindow()
  if(Window.innerWidth <= 600) return 3
  else if(Window.innerWidth <= 930) return 2
  else { 
    if(Window.innerWidth <= 1140){
      setViewType(2)
    }
    return 1 
  }
}

export const [deviceType, setDeviceType] = createSignal(checkDevice())
export const [viewType, setViewType] = createSignal(Params.getValueInt('viewType')||2)

export default function Root() {

  const IS_LOCAL = isClient && window.location.hostname.includes("localhost")

  if(isClient){
    window.addEventListener('resize', ()=> {
      const newDeviceType = checkDevice()
      console.log('device type::', newDeviceType)
      if(newDeviceType !== deviceType()){ setDeviceType(newDeviceType) }
    })
  
    if ('serviceWorker' in navigator && !IS_LOCAL) {
      window.addEventListener('load', () => {
        navigator.serviceWorker.register('/sw.js', { scope: '/' })
      })
    }
    if(LocalStorage.getItem("ui-color") === "dark"){
      document.body.classList.add('dark')
    }
  }

  createEffect(() => {
    if(viewType() === 2 && deviceType() === 1){
      document.body.classList.add('view-min')
    } else {
      document.body.classList.remove('view-min')
    }
    Params.setValue("viewType", viewType())
  })

  onMount(() => {
    if(checkIsLogin() === 3){
      Env.navigate("login")
      return
    }
  })

  return <>
    <Router root={props => {
        Env.pathname = props.location.pathname
        const navigate = useNavigate()
        Env.navigate = navigate
        return <MetaProvider>
          <Title>GENIX - MyPes</Title>
          <Meta name="loc" content={props.location.pathname}/>
          <Suspense fallback={PageLoadingElement}>{props.children}</Suspense>
        </MetaProvider>
      }}>
      <Route path="/_loading" component={PageLoading} />
      <Route path="/page/:name" component={PageBuilder} />
      <Suspense fallback={PageLoadingElement}>{<FileRoutes />}</Suspense>
    </Router>
    <PageMenu/>
  </>
}
