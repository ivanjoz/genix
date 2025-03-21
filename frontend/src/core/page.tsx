import { JSX, Show, createMemo, createSignal, onCleanup, onMount } from "solid-js"
import { isRouteChanging, setInnerPageName, setIsRouteChanging, setPageView, setPageViews } from "./menu"
import { fetchPending } from "~/shared/http"
import { Env, isLogged, isLogin, Params } from "~/shared/security"
import { useLocation } from "@solidjs/router"
import { deviceType } from "~/app"

interface IPageContainer {
  fetchLoading?: boolean
  children: JSX.Element
  class?: string
  title?: string
  views?: [number, string][] /* [id, nombre][] */
  accesos?: number[]
  pageStyle?: JSX.CSSProperties
}

export const PageContainer = (props: IPageContainer) => {
  const location = useLocation()
  
  onMount(() => {
    if(!isLogged()){
      Env.navigate("login")
      return
    }
    setIsRouteChanging(false)
    setInnerPageName(props.title||"")
    setPageViews(props.views||[])
    const view = Params.getValueInt(`pview-${location.pathname}`)
    if(props.views?.length > 0){
      setPageView(view ? view : (props.views[0][0] || 0))
    } else {
      setPageView(0)
    }
  })
  
  onCleanup(() => { setInnerPageName("") })
  
  const isLoading = createMemo(() => {
    console.log("is rute charging::", isRouteChanging())
    if(isRouteChanging()){ return true }
    console.log("Loading:: ", fetchPending().size)
    if(!props.fetchLoading) return false
    console.log("setting is loading::",  fetchPending().size > 0)
    return fetchPending().size > 0
  })

  const cN = () => {
    let cN = "page-container"
    if(props.class){ cN += " " + props.class }
    if([2,3].includes(deviceType())){ cN += " mobile" }
    return cN
  }

  const hasAccess = createMemo(() => {
    if(!isLogged()){ 
      return false
    } else if(props.accesos?.length > 0){
      return props.accesos.some(x => Params.checkAcceso(x,1))
    }
    return true
  })

  console.log("has acces::", hasAccess(),"|",isLogin())

  return <div class={cN()} style={props.pageStyle}>
    <Show when={!isLoading() && hasAccess()}>
      {props.children}   
    </Show> 
    <Show when={isLoading() && hasAccess()} >
      <Spinner1 message="Cargando Módulo..."/>    
    </Show>
    <Show when={!hasAccess()}>
      <div class="box-error-ms">
        No posee accesos para visualizar este módulo.
      </div>
    </Show>
  </div>
}

export const Spinner1 = (props: { message: string }) => {
  return <div class="flex ai-center" style={{ padding: "5rem 12rem" }}>
    <div class="spinner"></div>
    <h2 class="ml-08">{props.message}</h2>
  </div>
} 

export const PageLoadingElement = (
  <div class="page-container">
    <Spinner1 message="Cargando Aplicación..."/>   
  </div>
)

export const PageLoading = () => {
  return PageLoadingElement
}