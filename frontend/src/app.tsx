// @refresh reload
import { MetaProvider, Title } from "@solidjs/meta";
import { Router } from "@solidjs/router";
import { FileRoutes } from "@solidjs/start";
import { Show, Suspense, createEffect, createSignal } from "solid-js";
import LoginPage from "./core/login";
import { MainMenu, MainMenuMobile, MainTopMenu } from "./core/menu";
import Modules from "./core/modules";
import { createIndexDB } from "./shared/main";
import { Params } from "./shared/security";

const defaultModule = Modules[0]
export const [appModule, setAppModule] = createSignal(defaultModule)
export const [showMenu, setShowMenu] = createSignal(false)

const DEV_HOSTS = ["d16qwm950j0pjf.cloudfront.net","genix-dev.un.pe"]

const isClient = typeof window !== 'undefined'
if(isClient){
  document.head.querySelector('title')?.remove()

  let api = "http://localhost:3589/api/"
  
  if(DEV_HOSTS.includes(window.location.host)){
    api = "https://dnh72xkkh3junf57p3vexemlvm0emgys.lambda-url.us-east-1.on.aws/api/"
  }
  window.API_ROUTES = { MAIN: api }

  window._dexieVersion = 1
  window._cache = {}
  window._params = { fetchID: 1001, fetchProcesses: new Map() }
  window.appId = "gerp"
  window._pendingRequests = []
  window._counterID = 1
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

  const isLogin = () => loginStatus() && isClient
  
  createEffect(() => {
    if(viewType() === 2 && deviceType() === 1){
      document.body.classList.add('view-min')
    } else {
      document.body.classList.remove('view-min')
    }
  })
  
  return <>
    <Router root={props => (
        <MetaProvider>
          <Title>GENIX - MyPes</Title>
          <Suspense>{props.children}</Suspense>
        </MetaProvider>
      )}>
      <Show when={isLogin()}>
        <FileRoutes />
      </Show>
    </Router>
    <Show when={!loginStatus() && isClient}>
      <LoginPage setLogin={setLoginStatus} />
    </Show>
    <Show when={isLogin()}>
      <Show when={[1].includes(deviceType())}><MainMenu/></Show>
      <Show when={[2,3].includes(deviceType()) && showMenu()}>
        <MainMenuMobile/>
      </Show>
      <MainTopMenu />
    </Show>
  </>
     
  

  return (
    <html lang="es">
      <head>
        <base href="/"/>
        <meta charset="utf-8"/>
        <link rel="stylesheet" href="libs/fontello-embedded.css"/>
      </head>
      <body classList={{ 'view-min': viewType() === 2 && deviceType() === 1 }}>  
      <Router root={props => (
              <MetaProvider>
                <Title>GENIX - MyPes</Title>
                <Suspense>{props.children}</Suspense>
              </MetaProvider>
            )}>
            <FileRoutes />
          </Router>  
        <Show when={true}>
          <Show when={[1].includes(deviceType())}><MainMenu/></Show>
          <Show when={[2,3].includes(deviceType()) && showMenu()}>
            <MainMenuMobile/>
          </Show>
          <MainTopMenu />
        </Show>
        {/*
        <Show when={!loginStatus() && isClient}>
          <LoginPage setLogin={setLoginStatus} /> 
        </Show>
            */
        }
      </body>
    </html>
  )
}
