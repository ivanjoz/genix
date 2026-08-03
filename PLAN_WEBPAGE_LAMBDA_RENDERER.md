# PLAN — Prerender de Webpages de Company en Lambda

**Estado: Fases 1, 2 y 3 hechas y verificadas. Faltan la 4, la 5 (resto) y la 6.**

Hasta que no esté la Fase 4, el sitio en vivo **no refleja** lo que publica la nueva tubería:
el HTML se escribe en R2 bajo `websites-html/<hostname>/` y el Worker sigue sirviendo desde
Workers Static Assets. Nada está desplegado todavía (sin commit, sin `deploy.sh 9`).

## 1. De dónde se parte (acción `[11]` original)

Todo corría en la máquina del dev:

1. `deploy.sh 11 <companyID>` → `backend/exec/company_webpage_deploy.go:DeployCompanyWebpage`.
2. Resuelve el dominio: tabla `Parameters` (group `10`, key `domain`) → `hostname`.
3. Ejecuta `bun scripts/prerender.mjs --company N --out .build-<host>`, que:
   - lanza el **build completo de Vite/SvelteKit** con `VITE_COMPANY_ID=N`;
   - ese flag activa `ssr`+`prerender`, pone `paths.base=''` y **queda inlineado en el
     bundle cliente** (`frontend/core/env.ts`);
   - `load()` llama `GET.p-webpage` **en build time** → SEO + secciones horneadas;
   - post-proceso: `inlineBootstrap`, `flattenAssets`, `rewriteAssetUrls`, `mergeCss`,
     `dropOrphanEnvChunk`, `inlineDompurifyStubChunk`, `inlineTrivialNodeChunks`.
4. Sube todo lo que no es `.html`/`sw.js` a R2 → `websites/<companyID>/*`.
5. Deja `*.html` + `sw.js` en `frontend/webpage/cloudflare/webpages/<host>/`.
6. `DeployCloudflareWorker()` → **Workers Static Assets**: manifest de **TODOS** los tenants
   del directorio local + republicación del Worker (`backend/exec/cloudflare_assets.go`).
7. `provisionStorefrontDomain(host)` → Worker Custom Domain.

Publicar exigía la máquina del dev con el monorepo, bun y node_modules, y tener el HTML de
todos los demás tenants en disco.

## 2. Decisiones tomadas

| Tema | Decisión |
|---|---|
| Copia de assets | **Una copia por company** en `websites/<companyID>/`, incluida en el zip. Sin versionado, sin pinning, sin GC: los assets de una company cambian solo cuando esa company se republica. |
| HTML publicado | **Bucket R2 + Cache API** en el Worker. Publicar = 1 PUT por página. |
| Versión del renderer | **Solo la última.** El Lambda baja siempre el zip más reciente (GET condicional por ETag, caché en `/tmp`). |
| Ubicación del artefacto | GitHub Pages con URL fija (`WEBPAGE_RENDERER_URL` en `credentials.json` la sobrescribe). |
| Páginas a renderizar | `/` (ID 10), `/about` (ID 11) y las páginas de usuario (ID >= 15). **Nunca** `/store`, `/product`, `/cart`. |

## 3. Los problemas de fondo y cómo se resolvieron

### 3.1 El companyID estaba inlineado en el bundle → se movió a runtime

`VITE_COMPANY_ID` no solo cambiaba URLs: Vite lo inlinea como literal dentro del chunk de
`env.ts`. El bundle SSR del Lambda es **uno solo** para todas las companies, así que el
tenant se resuelve por request: `hooks.server.ts` lo lee de `?cid=` y lo deja como
`<meta name="company-id">` en el HTML, de donde lo toma el cliente tras hidratar.

### 3.2 `adapter-static` descarta el bundle servidor

SvelteKit siempre escribe su servidor en `.svelte-kit/output/server/{index.js,manifest.js}`
(es lo que consume cualquier adapter), pero repartido en decenas de chunks que importan a
`@sveltejs/kit`. CI lo empaqueta con **esbuild en un solo `render.mjs`** → sin `node_modules`
en el zip. No hizo falta cambiar de adapter.

### 3.3 Workers Static Assets no permite deploy incremental

El manifest **reemplaza el namespace completo**: publicar la company 7 exigiría el hash y el
contenido del HTML de todas las demás. Con R2 + binding, publicar es un PUT (Fase 4).

### 3.4 Los assets no se reescriben por company

Los chunks se referencian entre sí con rutas relativas, que resuelven contra la URL del
propio chunk. Conservando la estructura `_app/immutable/**` funcionan desde cualquier
prefijo del CDN **sin tocar un byte**, así que el Lambda sube los assets del zip tal cual.

Lo único que depende del tenant son las URLs del `<head>`, que el SSR emite contra la raíz
del sitio (`/_app/…`, gracias a `paths.relative=false`): el Lambda antepone ahí la base del
CDN de la company. Una sola sustitución de prefijo.

> Esto reemplaza al `flattenAssets` del pipeline anterior y al placeholder `__ASSET_BASE__`
> que contemplaba la versión previa de este plan: ambos sobraban.

## 4. Arquitectura

```
GitHub Actions (push a main, paths: frontend/**)
  └─ scripts/build-renderer.mjs
       bun run build (VITE_RENDERER_BUILD=1, sin companyID)
         ├─ build/_app/**            → assets js/css (se descarta lo no-código)
         └─ .svelte-kit/output/server → esbuild → render.mjs (bundle único)
       une las hojas de estilo en una y reapunta las refs de los chunks
       render de prueba + validación de las reglas
       emite manifest.json  →  empaqueta webpage-renderer.zip (478 KB)
     copia el zip al artefacto de Pages → https://genix-dev.un.pe/webpage-renderer.zip

deploy.sh 11 <companyID>
  └─ Go: fn-deploy-company-webpage
       ├─ hostname (Parameters group 10)
       ├─ páginas: IDs 10 y 11 + tabla Webpage (ID >= 15, Status > 0)
       └─ InvokeWebpageRenderer → Lambda <APP_NAME>-webpage-renderer (síncrona)
              { companyID, hostname, pages: [{id, path}], forceAssets? }

Lambda <APP_NAME>-webpage-renderer (nodejs22.x, arm64, 1024 MB, 120 s)
  1. GET condicional del zip por ETag → unzip a /tmp (reutilizado en invocaciones warm)
  2. si websites/<companyID>/.renderer-build != buildId:
       sube assets → R2 websites/<companyID>/_app/**   (immutable, 1 año)
       sube sw.js + favicon.ico → R2 websites-html/<hostname>/
  3. por cada página, EN SECUENCIA:
       a. server.respond('https://<host><ruta>?cid=&pid=')  → HTML SSR
          · hooks.server.ts fija el tenant y rellena las metas company-id / page-id
          · load() lee el snapshot  live/pages/<companyID>-<pageID>.json
          · las precargas de la cabecera Link se reescriben como <link> en el <head>
       b. aplica manifest.htmlRewrites (unión de css) y antepone el CDN a '/_app/'
       c. PUT → R2 websites-html/<hostname>/<ruta>/index.html
  4. devuelve { buildId, pages, assets, bytes }

Go: provisionStorefrontDomain(hostname)   (solo la primera vez)

Cloudflare Worker genix-storefront:   ← PENDIENTE (Fase 4)
  hostname + path → caches.default → R2 websites-html/… → 404
```

## 5. Contenido de `webpage-renderer.zip` (478 KB comprimido, 1.7 MB en disco)

```
render.mjs                     bundle esbuild del Server SSR de SvelteKit (702 KB)
assets/_app/immutable/**       18 archivos js/css (se suben tal cual a websites/<cid>/)
site/sw.js, site/favicon.ico   van al origen del sitio, no al CDN
manifest.json                  { buildId, assetPathPrefix, htmlRewrites[], assets[], siteFiles[] }
```

`htmlRewrites` son las transformaciones deterministas que CI ya resolvió (colapsar los
`<link>` de css en el archivo unido). El Lambda las aplica y luego antepone
`<FRONTEND_CDN>/websites/<companyID>` a `assetPathPrefix` (`/_app/`).

## 6. Fases

### Fase 1 — Tenant y página en runtime (frontend) — ✅ HECHA
- `frontend/core/env.ts`: `readHeadMeta()` + `Env.pageID` + `Env.getPageID()`;
  `getCompanyID()` lee `<meta name="company-id">` como fuente autoritativa, antes del
  fallback de build.
- `frontend/webpage/hooks.server.ts` (nuevo) + `kit.files.hooks` en `svelte.config.js`:
  lee `?cid=&pid=` del Request del Lambda, lo fija en `Env` y `transformPageChunk`
  rellena las metas `company-id` / `page-id` de `app.html`.
- `routes/[...path]/+page.svelte|ts` (movido desde `routes/`): un documento por página.
  `load()` lee el **snapshot CDN** vía `getStoreWebpageFromCDN(Env.getPageID())`
  (`frontend/services/ecommerce/page-content.svelte.ts:136`) en vez de `GET.p-webpage`;
  el refresco de `onMount` sí sigue yendo a la API, con el pageID correcto.
- `app.html`: `<body data-sveltekit-reload>` — toda navegación interna es carga completa.

Desviaciones respecto de lo planeado:
- **No** se tocó `routes/+layout.ts` en esta fase: sus flags `ssr`/`prerender` siguieron
  colgando de `VITE_COMPANY_ID`, así que la acción `[11]` original nunca dejó de funcionar
  durante la transición.
- `svelte.config.js` necesitó `entries: ['*', '/']` (`'*'` no cubre rutas dinámicas y la
  raíz vive ahora en un catch-all) y `crawl: false` (sin él el crawler prerenderizaba
  cualquier enlace interno, p. ej. `/shop`, con el contenido de la raíz).
- El hook necesita un guard con `building`: durante un prerender SvelteKit prohíbe leer
  `searchParams`. En ese caso cae a `VITE_COMPANY_ID`, como antes.
- Sin meta `cdn-url`: `Env.CDN_URL` viene de `PUBLIC_FRONTEND_CDN` y es idéntica para todos
  los tenants, no hace falta por documento.
- Eliminado `routes/+page.svelte.json` (`{"demo":"demo"}`, muerto).
- `frontend/webpage/tsconfig.json`: añadido `exclude` de `build`/`dist-prerender`.
  `bun run check` moría por OOM typecheckeando sus propios bundles minificados (previo).

Verificado: build admin/CSR OK (metas a `0`), `scripts/prerender.mjs --company 1` OK
(metas a `1`, contenido horneado, 12 assets), `vite dev` con `/promos?cid=5&pid=11` emite
`company-id=5` / `page-id=11`. `svelte-check` no reporta nada en los archivos tocados.

### Fase 2 — Build del renderer (CI) — ✅ HECHA
- `svelte.config.js`: modo `VITE_RENDERER_BUILD` (`paths.base=''`, `paths.relative=false`,
  `entries: []`). `routes/+layout.ts` activa `ssr` también en ese modo, sin prerender.
  `frontend/core/env.ts`: `isPrerenderStorefront` → `isStorefrontBuild`, que ahora cubre
  los dos builds de tienda (si no, el bundle publicado podría elegir el endpoint de API
  desde localStorage en vez de `PUBLIC_LAMBDA_URL`).
- `frontend/webpage/lambda/renderer-entry.js` (nuevo): expone `renderPage()` sobre el
  `Server` de SvelteKit.
- `scripts/build-renderer.mjs` (nuevo): build → bundle esbuild → assets → unión de css →
  render de prueba → validación → `manifest.json` → zip.
- `.github/workflows/ci-deploy.yml`: paso `Build webpage renderer` que copia el zip dentro
  del artefacto de Pages → `https://genix-dev.un.pe/webpage-renderer.zip`.

Resultado: **zip de 478 KB** (`render.mjs` 702 KB + 18 assets + `sw.js`/`favicon.ico`).

Desviaciones respecto de lo planeado:
- **Sin flatten y sin `__ASSET_BASE__`** (ver §3.4): se ahorran ~90 líneas de regex y una
  indirección. El manifest declara `assetPathPrefix: "/_app/"` y el Lambda antepone la base.
- **Descarte de no-código**: el grafo arrastraba 11.9 MB de wasm (excelize, encoders avif)
  que la tienda pública nunca ejecuta; sin ese filtro el zip pesaba 6.8 MB. El pipeline
  anterior también los descartaba, así que el sitio publicado se comporta igual.
- **Precargas**: SvelteKit solo emite `<link rel="modulepreload">` cuando prerenderiza; en
  SSR las manda en la cabecera `Link`, que se perdería al guardar el HTML como archivo.
  `renderer-entry.js` las reescribe como `<link>` en el `<head>` (9 en la página raíz).
  Sin esto los chunks se descubrirían en cascada.
- **NO implementado**: el inline del bootstrap como módulo blob y el inline de los chunks
  minúsculos (`env.js` huérfano, stub de DOMPurify, nodo trivial). Coste medido: **2
  peticiones extra de ~3 KB** —los entries `start`/`app`, que además van precargados, así
  que no añaden ida y vuelta— más 2 chunks minúsculos en carga diferida. A cambio se evita
  la parte más frágil del pipeline anterior. Se puede añadir después.
- El render de prueba y la **validación de reglas** corren dentro del build: si una regla
  deja de casar o el HTML referencia un asset ausente, CI falla en vez de publicar una
  página con css muerto.

Verificado: los tres modos de build conviven — admin/CSR (`/webpage-app/_app/…`, metas a
`0`), flujo antiguo `[11]` (`--company 1` sigue generando su carpeta plana) y renderer.

### Fase 3 — Lambda de Node en CloudFormation — ✅ HECHA

- `frontend/webpage/lambda/handler.mjs` (nuevo, sin dependencias npm): GET condicional del
  zip por ETag → unzip a `/tmp` → `import('render.mjs')` → render de N páginas → aplica
  `htmlRewrites` → sube assets y HTML a R2 con `fetch` (misma API REST que
  `cloud.SaveFileToR2`, `backend/cloud/s3.go:340`). Devuelve un resumen JSON.
- `cloud/template.yml`: `WebpageRendererFunction` (nodejs22.x, arm64, 1024 MB, 120 s,
  reusa `LambdaIamRole`), `WebpageRendererInvokePermission` y `WebpageRendererLogGroup`.
  Parámetros nuevos `RendererS3Key` y `RendererZipUrl`.
  El stack no crea recursos IAM, así que el permiso de invocación se concede como política
  de recurso sobre la función, con el rol del backend como `Principal`: dentro de la misma
  cuenta eso basta.
- `cloud/webpage-renderer.go` (nuevo): `CompileRendererToS3` zipea el handler y lo sube a
  `DEPLOYMENT_BUCKET/gerp-artifacts/webpage-renderer-lambda.zip` antes de
  `DeployCloudFormation`; `UpdateRendererEnviromentVariables` reinyecta después
  `FRONTEND_CDN`, `CLOUDFLARE_ACCOUNT`, `CLOUDFLARE_TOKEN` y `CLOUDFLARE_BUCKET` — el bloque
  `Environment` de la plantilla se reemplaza entero en cada deploy y los secretos no pueden
  quedar visibles en la consola de CloudFormation. Se pasan sueltas, no como el `CONFIG`
  zstd+base64 del backend, para no obligar al handler de Node a descomprimir zstd.
- `backend/cloud/webpage_renderer.go` (nuevo): `InvokeWebpageRenderer` — invoke **síncrono**
  vía el SDK de Lambda. No reusa `ExecLambda` (`backend/cloud/lambda.go:30`) porque ese
  payload es el protocolo `fn_exec` de Go y en local se desvía a un HTTP contra el backend
  local; aquí siempre hay que llamar a la Lambda real de AWS.

> Ojo con los nombres: `webpage-renderer-lambda.zip` es el **código** de la función;
> `webpage-renderer.zip` es el **artefacto de CI** con el SSR y los assets.

Notas de implementación:
- **Lector de ZIP propio** (~45 líneas) en el handler: el runtime de Lambda no trae
  descompresión de zip y una librería obligaría a empaquetar `node_modules`. Se recorre el
  directorio central (fuente autoritativa de tamaños) y se infla con `node:zlib`.
- **Los assets no se resuben si no cambiaron**: el handler deja un marcador
  `websites/<companyID>/.renderer-build` con el `buildId`. Republicar contenido cuesta solo
  los PUT del HTML (verificado: 2ª invocación → `assets: 0`). El evento acepta `forceAssets`.
- `CLOUDFLARE_API_BASE` es redirigible para poder ejercitar el handler completo contra un
  R2 simulado; por defecto apunta a la API real.
- `WEBPAGE_RENDERER_URL` es opcional en `credentials.json`; por defecto
  `https://genix-dev.un.pe/webpage-renderer.zip`. `FRONTEND_CDN` y las claves de Cloudflare
  ya estaban.

Verificado: handler ejercitado de punta a punta contra un servidor local que simula GitHub
Pages y la API de R2 — descarga, unzip (38 ms), render de `/` y `/about`, reescrituras y 23
PUT con sus content-type y cache-control correctos. El HTML resultante lleva 1 hoja de
estilo, 9 `modulepreload`, las 12 URLs de assets apuntando al CDN de la company,
`company-id=7`, `page-id=10` y el favicon same-origin. `go build`/`go vet`/`go test` OK en
`backend` y `cloud`; la plantilla pasa `aws cloudformation validate-template`.

### Fase 5 (adelantada) — Orquestación Go — ✅ HECHA en la Fase 3

`backend/exec/company_webpage_deploy.go` ya no ejecuta `bun` ni `DeployCloudflareWorker()`:
resuelve el hostname, arma la lista de páginas (`getCompanyWebpagePages`) e invoca la Lambda.
Se eliminaron `uploadCompanyWebpageAssets` y `removeUploadedWebpageAssets`, que quedaron
muertas. Lo único que faltaba de esta fase —retirar `base_template_deploy.go` y la acción
`[13]`— se hace en la Fase 6.

Payload real:
```json
{ "companyID": 7, "hostname": "tienda.un.pe",
  "pages": [ {"id": 10, "path": "/"}, {"id": 11, "path": "/about"},
             {"id": 15, "path": "/promos"} ] }
```
Respuesta: `{ "buildId": "…", "pages": 3, "assets": 20, "bytes": 29714 }`.

### Fase 4 — Worker sirviendo HTML desde R2 — ⏳ PENDIENTE
- `frontend/webpage/cloudflare/src/serve-worker.ts` **y** el `serve-worker.js` desplegado
  (hoy es una copia despojada a mano; hay que mantenerlos en sync): binding R2 en vez de
  `ASSETS`, clave `websites-html/<hostname><ruta>/index.html`, con `caches.default`.
- `wrangler.jsonc`: `r2_buckets` en vez del bloque `assets`.
- `backend/exec/cloudflare_assets.go` y `cloudflare_worker_deploy.go`: el deploy del Worker
  deja de construir el manifest de tenants y `validateWebpageArtifacts` desaparece (exige un
  directorio `webpages/` que ya no existirá).

### Fase 6 — Limpieza — ⏳ PENDIENTE
- Borrar `scripts/prerender.mjs`, `frontend/webpage/dist-prerender/`,
  `frontend/webpage/cloudflare/webpages/`, `frontend/webpage/routes/base/`,
  `backend/exec/base_template_deploy.go` y la acción `[13]` de `deploy.sh`.
- Retirar el modo `VITE_COMPANY_ID` de `svelte.config.js`, `routes/+layout.ts`,
  `routes/[...path]/+page.ts`, `hooks.server.ts` y `frontend/core/env.ts`: al desaparecer
  el prerender antiguo, `isStorefrontBuild` se reduce a `VITE_RENDERER_BUILD`.
- Retirar de `backend/exec/company_webpage_deploy.go` lo que quede muerto tras la Fase 4:
  `companyWebpageAssetBase`, `isFingerprintedWebpageAsset`, `verifyGeneratedWebpage`,
  `replaceWebpageDirectory`, `webpageAssetUploadConcurrency` y sus casos en
  `cloudflare_worker_deploy_test.go`.
- Actualizar `frontend/webpage/ECOMMERCE.md`.

## 7. Riesgos

| Riesgo | Mitigación |
|---|---|
| Cold start del Lambda (descarga + unzip) | Zip de 478 KB; unzip medido en 38 ms. Caché en `/tmp` por ETag, así que solo afecta a la primera invocación de cada versión. |
| Hydration mismatch si el snapshot cambia entre el SSR y el `onMount` | Ya contemplado: `routes/[...path]/+page.svelte` compara contenido y solo reasigna si difiere. |
| `getCounterForKey` (hash de clases CSS) debe coincidir server/client | Ambas pasadas ocurren en el mismo build de CI → sigue siendo determinista. |
| SvelteKit cambia el formato de sus `<link>` y una regla deja de casar | El build hace un render de prueba y falla si una regla `required` no casa o si el HTML referencia un asset ausente. El handler repite la comprobación en cada render. |
| Renders concurrentes en un mismo proceso Lambda | `hooks.server.ts` fija el tenant sobre el singleton `Env`; por eso el handler renderiza **en secuencia**. Está comentado en los tres archivos implicados. |
| Primera request por POP pega a R2 (~30-80 ms) | `caches.default` + `cache-control`; solo afecta al primer visitante de cada POP. |
| `sw.js` debe ser same-origin | Se sube a `websites-html/<hostname>/sw.js`, no al CDN. |
| Publicaciones concurrentes de la misma company | El PUT a R2 es atómico por objeto; la última gana. Aceptable. |

## 8. Fuera de alcance

- **Handler "Publicar" desde el builder**: el objetivo de producto es publicar sin máquina
  de dev, y la tubería ya lo permite. Se añade después reusando `InvokeWebpageRenderer`.
- **Inline del bootstrap y de los chunks minúsculos** (ver Fase 2), con su coste medido.
