// @refresh reload
import { MetaProvider, Title } from "@solidjs/meta";
import { Route, Router } from "@solidjs/router";
import { FileRoutes } from "@solidjs/start/router";
import { Show, Suspense, createEffect, createSignal } from "solid-js";
import LoginPage from "./core/login";
import { MainMenu, MainMenuMobile, MainTopMenu } from "./core/menu";
import Modules from "./core/modules";
import { createIndexDB } from "./shared/main";
import { Params } from "./shared/security";
import PageBuilder from "./pages/page";
import CmsWebpage from "./routes/cms/webpage";

let defaultModule = Modules[0]
const isClient = typeof window !== 'undefined'
if(isClient){
  const moduleSelected = Params.getValueInt("moduleSelected")
  if(moduleSelected){ defaultModule = Modules.find(x => x.id === moduleSelected) }
}
export const [appModule, setAppModule] = createSignal(defaultModule)
export const [showMenu, setShowMenu] = createSignal(false)

const DEV_HOSTS = ["d16qwm950j0pjf.cloudfront.net","genix-dev.un.pe"]
if(isClient){
  document.head.querySelector('title')?.remove()

  let api = "http://localhost:3589/api/"
  // let api = "https://asdxv2tjzpe2biexmp7p2robau0uussc.lambda-url.us-east-1.on.aws/api/"

  if(DEV_HOSTS.includes(window.location.host)){
    api = "https://dnh72xkkh3junf57p3vexemlvm0emgys.lambda-url.us-east-1.on.aws/api/"
  }
  window.API_ROUTES = { MAIN: api }

  window._dexieVersion = 1
  window._cache = {}
  window._params = { fetchID: 1001, fetchProcesses: new Map() }
  window.appId = "genix"
  window._pendingRequests = []
  window._counterID = 1
  window.S3_URL = "https://genix-dev.un.pe/"
  window._zoneOffset = (new Date()).getTimezoneOffset() * 60
  createIndexDB(Modules)
}

export const [loginStatus, setLoginStatus] = createSignal(isClient && Params.checkAcceso(1))

export const checkDevice = () => {
  if(!isClient) return 1
  if(window.innerWidth <= 600) return 3
  else if(window.innerWidth <= 930) return 2
  else { 
    if(window.innerWidth <= 1140){
      setViewType(2)
    }
    return 1 
  }
}

export const [deviceType, setDeviceType] = createSignal(checkDevice())
export const [viewType, setViewType] = createSignal(1)

export default function Root() {

  const IS_LOCAL = isClient && window.location.hostname.includes("localhost")

  if(isClient){
    window._route = window.location.pathname

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
  }

  const isLogin = () => {
    if(!isClient){ return 3 }
    if(window.location.pathname.substring(0,5) === '/page'){ return 1 }
    else if(loginStatus() && isClient){ return 2 }
    else { return 3 }
  }
  
  createEffect(() => {
    if(viewType() === 2 && deviceType() === 1){
      document.body.classList.add('view-min')
    } else {
      document.body.classList.remove('view-min')
    }
  })

  console.log("is login:: ", isLogin())
  
  return <>
    <Router root={props => (
        <MetaProvider>
          <Title>GENIX - MyPes</Title>
          <Suspense>{props.children}</Suspense>
        </MetaProvider>
      )}>
      <Route path="/pages" component={PageBuilder} />
      <Route path="/pages/:name" component={PageBuilder} />
      <Route path="/cms/webpage/:name" component={CmsWebpage} />
      <Show when={isLogin() === 2}>
        <FileRoutes />
      </Show>
    </Router>
    <Show when={isLogin() === 3}>
      <LoginPage setLogin={setLoginStatus} />
    </Show>
    <Show when={isLogin() === 2}>
      <Show when={[1].includes(deviceType())}><MainMenu/></Show>
      <Show when={[2,3].includes(deviceType()) && showMenu()}>
        <MainMenuMobile/>
      </Show>
      <MainTopMenu />
    </Show>
  </>
}
