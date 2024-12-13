"use-strict"
// Names of the two caches used in this version of the service worker.
// Change to v2, etc. when you update any of the local resources, which will
// in turn trigger the install event again.
const PRECACHE = 'precache-v1'
const CACHE_ASSETS = 'assets-v1'
const CACHE_STATIC = 'static'

// A list of local resources we always want to be cached.
const PRECACHE_URLS = [
  // 'index.html',
  // './', // Alias for index.html
]

const parseFileExtension = (filename) => {
  const ix1 = filename.indexOf("?")
  if (ix1 !== -1) filename = filename.substring(0, ix1)
  const ix2 = filename.indexOf("@")
  if (ix2 !== -1) filename = filename.substring(0, ix2)
  filename = filename.substring(filename.indexOf("/", 8))
  const ix3 = filename.lastIndexOf(".")
  if (filename === "/" || (ix3 === -1 && filename[2]) === "/") {
    return [filename, "*"]
  }
  return [filename, filename.substring(ix3 + 1)]
}

// The install handler takes care of precaching the resources we always need.
self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(PRECACHE)
      .then(cache => cache.addAll(PRECACHE_URLS))
      .then(self.skipWaiting())
  )
})

// The activate handler takes care of cleaning up old caches.
self.addEventListener('activate', event => {
  const currentCaches = [PRECACHE, CACHE_ASSETS, CACHE_STATIC];
  event.waitUntil(
    caches.keys().then(cacheNames => {
      return cacheNames.filter(cacheName => !currentCaches.includes(cacheName));
    }).then(cachesToDelete => {
      return Promise.all(cachesToDelete.map(cacheToDelete => {
        return caches.delete(cacheToDelete);
      }));
    }).then(() => self.clients.claim())
  )
})

const extractVersion = (bodyHTML) => {
  const idx1 = bodyHTML.indexOf(`name="build-version"`)
  if (idx1 === -1) return null
  const idx2 = bodyHTML.indexOf(`"`, idx1 + 28)
  const buildNumber = bodyHTML.substring(idx2 + 1, idx2 + 7)
  return buildNumber
}

const versionToHour = (buildNumber) => {
  const nowTime = Math.floor(Date.now() / 1000)
  const buildTime = parseInt(buildNumber.toLowerCase(), 36) * 10
  const seconds = nowTime - buildTime
  const hours = Math.floor(seconds / 60 / 60)
  const minutes = Math.floor((seconds - (hours * 60 * 60)) / 60)
  const horasText = `${hours} hora${hours === 1 ? '' : 's'}`
  let hace = `${horasText} ${minutes} min`
  if (hours === 0) hace = `${minutes} min`
  if (hours > 12) hace = `más de ${horasText}`
  return hace
}

const VersionInfo = { build: "", hasUpdated: false }

const broadcast = new BroadcastChannel('main')

broadcast.onmessage = ({ data }) => {
  console.log("service worker event::", data)
  if (data && data.checkVersion) {
    console.log("build code compare:: ", data.checkVersion, " | ", VersionInfo.build)
    if (VersionInfo.build && data.checkVersion !== VersionInfo.build) {
      caches.delete(CACHE_ASSETS).then(() => {
        broadcast.postMessage({
          versionHasUpdated: versionToHour(VersionInfo.build)
        })
          .catch(err => {
            console.log("========== Error: al eliminar caché =======")
            console.log(err)
          })
      })
    }
  }
}

const compareVersionUpdate = (versionCurrent) => {
  VersionInfo.build = versionCurrent

  const headers = new Headers()
  headers.append('pragma', 'no-cache')
  headers.append('cache-control', 'no-cache')

  fetch(self.location.origin + "/app-version", { method: 'GET', headers })
    .then(res => res.text())
    .then(bodyHTML => {
      const versionUpdated = extractVersion(bodyHTML)
      VersionInfo.build = versionUpdated
      console.log('build code compare fetch:: ', versionCurrent, " | ", versionUpdated)

      if (versionCurrent !== versionUpdated) {
        VersionInfo.hasUpdated = true
        caches.delete(CACHE_ASSETS).then(() => {
          broadcast.postMessage({ versionHasUpdated: versionToHour(versionUpdated) })
        })
          .catch(err => {
            console.log("========== Error: al eliminar caché =======")
            console.log(err)
          })
      }
    })
    .catch(err => {
      console.log(self.location.origin + "/app-version")
      console.warn('Error::', err)
    })
}

// The fetch handler serves responses for same-origin resources from a cache.
// If no response is found, it populates the runtime cache with the response from the network before returning it to the page.
self.addEventListener('fetch', event => {
  console.log("event URL:: ", event.request.url)
  if (!event.request.url.startsWith(self.location.origin)) return

  const [filename, ext] = parseFileExtension(event.request.url)
  if (filename === '/app-version') return

  // como es una SPA todas las URL internas van hacia "/"
  // ext === '*' significa que es una ruta interna del SAP (sin extension) en vez de un archivo 
  if (ext === '*') { event.request.url = self.location.origin + "/" }

  // Skip cross-origin requests, like those for Google Analytics.
  event.respondWith(
    caches.match(event.request).then(cachedResponse => {
      
      if (cachedResponse) {
        const contentType = cachedResponse.headers.get("content-type") || ""
        console.log("Enviando Cache Response:: ", contentType, event.request.url)

        if (ext === '*') {
          /*
          console.log("========== Comparing: Current Version =======")
          cachedResponse.clone().text().then(bodyHTML => {
            const versionCurrent = extractVersion(bodyHTML) // || '2S474W'
            if (versionCurrent) compareVersionUpdate(versionCurrent)
            else {
              console.log("No se encontró la versión dentro del HTML")
            }
          })
          */
        // Evita enviar un 'text/html' cuando se está solicitando una extensión en concreto (como '.js' o '.css')
        } else if(!contentType.includes("/html")){
          return cachedResponse
        }
      }
      
      const CACHE_NAME = ['js', 'css', 'html', '*', 'ts', 'mjs', 'tsx'].includes(ext)
        ? CACHE_ASSETS : CACHE_STATIC

      return caches.open(CACHE_NAME).then(cache => {
        return fetch(event.request).then(response => {
          const contentType = response.headers.get("content-type") || ""
          console.log("Guardando en caché: ", CACHE_NAME, " | " + event.request.url, " | ", contentType)

          // Si se está retornando un 'text/html' pero se está solicitando otra extensión, ejemplo un '.js', entonces evita guardarlo en caché
          if(contentType.includes("/html") || ["webmanifest"].includes(ext)){
            return response 
          }
          
          // Put a copy of the response in the runtime cache.
          return cache.put(event.request, response.clone())
            .then(() => {
              return response;
            })
            .catch(err => {
              console.error(err)
              console.log("error en:: ", event.request)
              console.log(response)
              return response;
            })
        })
      })
    })
  )
})