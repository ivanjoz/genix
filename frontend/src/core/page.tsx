import { JSX, Show, createMemo, createSignal, onCleanup, onMount } from "solid-js"
import { setInnerPageName, setPageView, setPageViews } from "./menu"
import { fetchPending, setfetchPending } from "~/shared/http"
import { Params } from "~/shared/security"
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
    console.log("Loading:: ", fetchPending().size)
    if(!props.fetchLoading) return false
    return fetchPending().size > 0
  })

  const cN = () => {
    let cN = "page-container"
    if(props.class){ cN += " " + props.class }
    if([2,3].includes(deviceType())){ cN += " mobile" }
    return cN
  }

  if(props.accesos?.length > 0){
    const isAuth = props.accesos.some(x => Params.checkAcceso(x,1))
    if(!isAuth){
      return <div class={cN()}>
        <div class="box-error-ms">
          No posee accesos para visualizar este módulo.
        </div>
      </div>
    }
  }
  
  return <div class={cN()} style={props.pageStyle}>
    <Show when={!isLoading()}>
      { props.children }       
    </Show> 
    <Show when={isLoading()}>
      <h2>Cargando...</h2>    
    </Show> 
  </div>
}