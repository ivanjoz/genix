import { goto } from '$app/navigation';

/**
 * Shared store for SearchSelect components
 * Controls which mobile search layer is currently open
 */
export let Core = $state({
  openSearchLayer: 0 as number,
  deviceType: 1 as number,
  mobileMenuOpen: 0 as number,
  openLayers: [] as number[]
})

export const mainMenuOptions = [
  { name: "Iniciar Sesión",
    icon: "icon-user",
    onClick: () => { 
      if(!Core.openLayers.includes(1)){
        Core.openLayers.push(1)
      }
    }
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

export interface NavFlag {
  id: number
  close: () => void
  updated?: number
  elementId: string
}

let navFlags: NavFlag[] = []
let navFlagCounter = 0
let navReturns = 0

export const suscribeUrlFlag = (elementId: string, callbackOnClose: (() => void)) => {
  navFlags = navFlags.filter(x => document.getElementById(x.elementId))

  let flag = navFlags.find(x => x.elementId === elementId)
  const isIncluded = !!flag

  if(!flag){
    flag = { 
      id: navFlags.length + 1, 
      close: callbackOnClose,
      elementId
    }
    navFlags.push(flag)
  }

  flag.updated = Date.now()
  /*
  if(!Env._navFlagsIDs.has(elementId)){
    Env._navFlagsIDs.set(elementId, Env._navFlagsIDs.size + 1)
  }
  const id = Env._navFlagsIDs.get(elementId)

  const isIncluded = Env._navFlags.some(x => x.elementId === elementId)
  if(isIncluded){
    Env._navFlags = Env._navFlags.filter(x => x.elementId !== elementId)
  }
  Env._navFlags.unshift({
    id, updated: Date.now(), elementId: elementId, close: callbackOnClose
  })
  */

  let uriParams = window.location.search.substring(1).split("&").filter(x => x)
  const nf = (uriParams.find(x => x.substring(0,3) === "nf=")||"").replace("nf=","")

  if(navFlagCounter === 0 && nf){
    navFlagCounter = nf.split(",").map(x => parseInt(x))[0]
  }

  if(!isIncluded || navReturns > 0){
    navFlagCounter++
    if(navReturns > 0){ navReturns-- }
    // Env.urlHistory.add(window.location.search)
    uriParams = uriParams.filter(x => x.substring(0,3) !== "nf=")
    uriParams.push(`nf=${navFlagCounter},${navFlags.map(x => x.id).join(",")}`)
    /*
    Env.navigate(window.location.pathname +"?"+ uriParams.join("&"), 
      { scroll: false })
    */
    goto(window.location.pathname +"?"+ uriParams.join("&"), { noScroll: true, replaceState: false })
  }
}

if(typeof window !== 'undefined'){
  window.addEventListener('popstate', () => {
    navFlags = navFlags.filter(x => document.getElementById(x.elementId))

    let flag: NavFlag
    for(const e of navFlags){
      if(!flag || e.updated > flag.updated){
        flag = e
      }
    }
    if(flag){
      flag.close()
      navFlags = navFlags.filter(x => x.id !== flag.id)
    }
  })
}
