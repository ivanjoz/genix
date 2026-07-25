import { browser } from "$app/environment";
import { Env } from '$core/env';
import { fetchEvent, tr, Core } from '$core/store.svelte';
import { addProcess, updateProcess } from '$core/notifications.svelte';
import { createUiRuntime } from '@genix/ui';
import { formatN, Notify } from '$libs/helpers';
import type { IUser } from '$core/types/common';

let progressTimeStart = 0
let progressBytes = 0

export const setFetchProgress = (bytesLen: number) => {
  const nowTime = Date.now()
  if(!progressBytes){
    progressTimeStart = nowTime
  }

  progressBytes += bytesLen

  let mbps = 0
  const kb = progressBytes/1000
  const elapsed = nowTime - progressTimeStart

  if(elapsed > 50){
    mbps = kb / elapsed
  }

  let msg = `Descargando... ${formatN(kb)} kb`
  if(mbps){
    if(mbps > 10){ mbps = 10 }
    msg += ` (${formatN(mbps,2)} MB/s)`
  }

  const loadingMsgDiv = document.getElementById("NotiflixLoadingMessage")
  if(loadingMsgDiv){
    let nextElement = loadingMsgDiv.nextElementSibling
    if(!nextElement && loadingMsgDiv.parentNode){
      nextElement = document.createElement("div")
      nextElement.setAttribute("id","NotifyProgressMessage")
      loadingMsgDiv.parentNode.insertBefore(nextElement, loadingMsgDiv.nextSibling)
    }
    if(nextElement){
      nextElement.innerHTML = msg
    }
  }
}

export const isPublicFrontendRoute = (routeValue?: string | null): boolean => {
  const normalizedRoute = String(routeValue || "").trim()
  // Use the exact-or-trailing-slash form so the public storefront (/webpage-app, /webpage-app/*)
  // is public WITHOUT also matching the authed admin builder route (/webpage-builder).
  return normalizedRoute === '/' || normalizedRoute === '/login'
    || normalizedRoute === '/webpage-app' || normalizedRoute.startsWith('/webpage-app/')
}

// Single configuration entry for @genix/ui: every host parameter — routing, tenant,
// translation, telemetry, notifications and session policy — is set right here.
export const genixUiRuntime = createUiRuntime<IUser>({
  applicationName: 'Genix',
  defaultLanguage: Core.languaje,
  translate: tr,
  makeCdnRoute: Env.makeCDNRoute,
  makeRoute: Env.makeRoute,
  getCompanyID: Env.getCompanyID,
  getEnvironment: () => Env.enviroment || 'main',
  getWorkerUrl: () => Env.serviceWorker,
  getPathname: Env.getPathname,
  navigate: Env.navigate,
  notify: Notify,
  security: {
    storageNamespace: Env.appId,
    // The public storefront has no /login route, so a logout there returns to its home.
    onLogout: () => {
      Env.navigate(isPublicFrontendRoute(Env.getPathname()) ? "/" : "/login")
    },
    messages: {
      sessionExpired: 'La sesión ha expirado, vuelva a iniciar sesión.',
      sessionExpiresIn: (minutes) => `La sesión expirará en ${minutes} minutos`,
    },
    isPublicRoute: isPublicFrontendRoute,
    autoStartRefreshCheck: true,
    // resolveRouteAccessEntries is registered by the authenticated app (routes/+layout.svelte)
    // so the public storefront bundle never embeds the backend access catalog.
  },
  onUnauthorized: () => {
    security.clearSession();
    if (browser) {
      document.dispatchEvent(new Event('userLogout'));
    }
    Notify.failure('La sesión ha expirado, vuelva a iniciar sesión.');
  },
  startRequest: (route) => {
    if (!browser) { return 0; }
    const requestId = fetchEvent(0, 0) as number;
    if (requestId > 0) {
      fetchEvent(requestId, { url: route });
    }
    return requestId;
  },
  finishRequest: (requestId) => fetchEvent(requestId, 0),
  reportFetch: fetchEvent,
  reportProgress: setFetchProgress,
  verifyRouteMemoryState: () => Env.DELTA_CACHE_VERIFY_ROUTE_MEMORY,
  addProcess,
  updateProcess,
})

// Session and access control for the whole app.
export const security = genixUiRuntime.security
