# webpage-renderer

Renderizador de las tiendas (webpages de company). Descarga el artefacto SSR, prerenderiza las
páginas de **una** company y publica el resultado: el HTML en **S3** (lo sirve CloudFront en
`<tienda>.un.pe`) y los js/css en **Cloudflare R2** (los sirve `frontend.cdn_url`).

Vive en la raíz del repo y no dentro de `frontend/` porque no es parte de la app: es un proceso
independiente que la app de SvelteKit no importa nunca, y el backend lo ejecuta directamente.

```
handler.mjs        la lógica. Único archivo que viaja en el zip de la Lambda.
cli.mjs            entrada por stdin/stdout, para cuando no hay Lambda.
handler.test.mjs   tests: bun test
```

## Los dos zips, que no son el mismo archivo

Es la confusión más fácil de este subsistema:

| Archivo | Qué contiene | Quién lo hace | Quién lo consume |
|---|---|---|---|
| `webpage-renderer-lambda.zip` | **solo** `handler.mjs` | `cloud/webpage-renderer.go` → S3 | CloudFormation, como código de la función |
| `webpage-renderer.zip` | el bundle SSR + los assets de la tienda | `scripts/build-renderer.mjs` → GitHub Pages | `handler.mjs` en caliente, vía `RENDERER_ZIP_URL` |

Esa separación es lo que permite **publicar un frontend nuevo sin redesplegar infraestructura**:
la función se descarga el artefacto en cada arranque frío y lo revalida por ETag en los calientes.

El entry del bundle SSR (`frontend/webpage/renderer-entry.js`) **no** está aquí: importa la salida
de build de SvelteKit (`../.svelte-kit/output/server/…`), así que pertenece a la app que la genera.

## Los dos caminos de ejecución

`backend/cloud/webpage_renderer.go:InvokeWebpageRenderer` decide según `core.Env.IS_SERVERLESS`:

- **En Lambda** → `Invoke` de la función `<APP_NAME>-webpage-renderer`, con el evento como payload.
- **Fuera** (VPS, local) → `node webpage-renderer/cli.mjs`, con el evento por stdin.

El handler es el mismo en los dos. Los llamadores no distinguen el caso.

En el VPS el backend corre como servicio de systemd, que arranca con un `PATH` mínimo
(`/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin`). Un node instalado con nvm vive bajo `~` y
**no está en esa lista**: el render falla con `exec: "node": executable file not found in $PATH`.
Conviene instalarlo del sistema (`dnf install nodejs`) y no enlazar el de nvm, que desaparece al
actualizar de versión.

## Contrato

Evento de entrada:

```json
{
  "companyID": 7,
  "hostname": "tienda.un.pe",
  "pages": [{ "id": 10, "path": "/" }, { "id": 11, "path": "/about" }],
  "forceAssets": false
}
```

Respuesta: `{ "buildId": "...", "pages": 2, "assets": 4, "bytes": 259 }`.

Con `cli.mjs`, **stdout es solo esa respuesta** y los logs van a stderr — el backend deserializa
stdout tal cual, sin filtrar líneas.

## Qué se escribe y dónde

| Qué | Dónde | Clave | Cache-Control |
|---|---|---|---|
| HTML por página | S3 (`storefront.bucket`) | `websites-html/<hostname>/<ruta>/index.html` | `public, max-age=60` |
| `favicon.ico` | S3 | `websites-html/<hostname>/favicon.ico` | `max-age=86400` |
| `sw.js` | S3 | `websites-html/<hostname>/sw.js` | `max-age=0, must-revalidate` |
| js/css | R2 (`cloudflare.bucket`) | `websites/<companyID>/_app/…` | `immutable` (llevan hash de contenido) |
| marcador | R2 | `websites/<companyID>/.renderer-build` | `no-store` |

El PUT a S3 se firma con SigV4 a mano (`signS3Request`), sin SDK, por la misma razón que el lector
de ZIP. Un test lo comprueba contra el vector oficial de AWS.

`sw.js` va bajo el hostname y no bajo el CDN porque un service worker **tiene que ser
same-origin**. Y no puede cachearse: es lo que gobierna las actualizaciones del sitio. Por colgar
del hostname y no de la company, los dos archivos de `site/` se suben en cada render, sin mirar el
marcador: si no, un dominio nuevo nacería sin ellos cuando la company ya tuviera sus assets
publicados.

El prefijo del CDN no se antepone solo en el HTML. Los `.js` llevan dentro sus propias rutas
—los arrays de `__vite__mapDeps`, y los `new URL('/_app/…', import.meta.url)`— y se reescriben al
subirlos (`applyAssetRewrites`). Sin eso la primera carga funciona y todo lo que pida el runtime
después se va al hostname de la tienda, donde no hay ningún `/_app/`.

Y con los deps de `mapDeps` ya absolutos hay que desactivar quien los componía: `__vitePreload`
los pasa por un helper `assetsURL` que antepone la base del build (`function(dep){return"/"+dep}`
con base `/`), y ese `/` sobrante produce `'/https://cdn…/_app/…'`, que el navegador resuelve
contra el hostname de la tienda. `applyAssetRewrites` lo deja en identidad.

El fallo es asimétrico y por eso cuesta leerlo: los `.js` cargan igual —el `import()` de verdad es
relativo al propio chunk y cae en el CDN—, así que solo se ven `<link rel=modulepreload>` en 404.
El que rompe es el dep `.css`, que `__vitePreload` **espera**: su `Unable to preload CSS for …`
deja la página sin hidratar. `rewriteAssets` comprueba el invariante entre chunks (si hay deps
relativos, tiene que haber exactamente un helper) y aborta la publicación si no cuadra, porque el
modo de fallo natural es terminar en verde.

El marcador guarda el `buildId` publicado: si coincide, los assets no se resuben y editar una
página cuesta solo los PUT del HTML. `forceAssets: true` lo ignora.

El HTML lo sirve la distribución de CloudFront de `cloud/storefront.yml` (stack
`<app_name>-storefront`, acción 5 de `cloud`), con un certificado comodín `*.un.pe`. Su CloudFront
Function resuelve `<hostname>/<ruta>` a la misma clave que escribe el handler. Los dos comparten el
prefijo `websites-html`, así que **cambiar `HTML_KEY_ROOT` obliga a cambiarlo en ambos**. Cada
tienda es un alias A/AAAA en Route 53 hacia esa distribución
(`backend/webpage/route53_domain.go`).

## Variables de entorno

Todas son obligatorias y el handler aborta si falta alguna:

`RENDERER_ZIP_URL`, `FRONTEND_CDN`, `CLOUDFLARE_ACCOUNT`, `CLOUDFLARE_TOKEN`, `CLOUDFLARE_BUCKET`,
`STOREFRONT_BUCKET`, `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
(`AWS_SESSION_TOKEN` si las credenciales son temporales).

En Lambda las de AWS las pone el runtime (rol de la función; `cloud/storefront.yml` le da
`s3:PutObject` sobre el bucket) y el resto las inyecta `UpdateRendererEnviromentVariables`
(`cloud/webpage-renderer.go`). En local las pasa el backend desde `core.Env`, con las credenciales
de AWS ya resueltas por el SDK (profile o rol).

`CLOUDFLARE_API_BASE` y `STOREFRONT_S3_ENDPOINT` son opcionales y solo existen para los tests:
redirigen R2 y S3 a un servidor local.

## Tests

```bash
cd webpage-renderer && bun test
```

No tocan la red ni AWS: levantan un `node:http` que hace de origen del artefacto y de bucket R2,
y construyen un zip mínimo a mano (igual que el lector del handler, para no arrastrar una
dependencia).

Dos cosas que hay que tener en cuenta al añadir casos:

1. El módulo lee `process.env` en constantes de nivel superior **al cargarse**. Cambiar una
   variable después de importarlo no tiene efecto; hay que reimportar con una query distinta
   (`loadHandler()` en el test lo hace).
2. `loadedRenderer` es estado de módulo y sobrevive entre tests del mismo archivo — a propósito,
   es lo que reproduce una Lambda caliente.

## Pendientes conocidos

Ninguno rompe el camino feliz. Están aquí porque fallan **en silencio**: el backend responde que
publicó, y el problema aparece más tarde y en otro sitio.

### La función de ruteo vive en la plantilla

La CloudFront Function que traduce `<hostname>/<ruta>` a la clave está inline en
`cloud/storefront.yml` y solo cambia con la acción 5. Si el handler cambia la forma de las claves y
la función no, las tiendas devuelven 404 mientras el backend informa de una publicación correcta.
Agregar o cambiar un dominio **no** requiere desplegar nada: el alias de Route 53 lo crea y borra
el backend (`provisionStorefrontDomain` / `removeStorefrontDomain`) y el certificado es comodín.

Un GET al hostname después de publicar convertiría esto en un error explícito.

### Código de Cloudflare Workers sin uso

`frontend/webpage/cloudflare/serve-worker.js` y `fn-deploy-cloudflare-worker`
(`backend/exec/cloudflare_worker_deploy.go`) servían el HTML desde R2 antes de S3 + CloudFront.
Ya no se usan y se pueden borrar; R2 sigue siendo el origen de los js/css.

### Una página sin snapshot se publica vacía

El SSR pide el contenido a `live/pages/<companyID>-<pageID>.json` en el CDN. Si no existe, el
`load()` de la tienda captura el 404, devuelve secciones vacías y el render **sigue adelante**: se
publica un HTML válido pero sin contenido, y el resultado dice `pages: 2` como si nada. En el log
solo queda un `[StorePage] webpage load failed`, que no llega a la respuesta.

Se reconoce por el tamaño: una página vacía pesa ~10 KB contra los ~27 KB de una con contenido.

### El artefacto se vuelve a descargar en cada publicación (VPS)

`loadRenderer()` compara el ETag y reutiliza lo extraído, pero `loadedRenderer` es una variable de
módulo: solo sirve mientras el proceso siga vivo. En Lambda caliente ahorra el trabajo; en el VPS
cada publicación arranca un `node` nuevo, así que arranca sin `if-none-match`, el 304 nunca llega a
usarse y se bajan y descomprimen los ~460 KB otra vez. El `en NNNms` del log `artefacto cargado` es
una cota inferior de lo que cuesta: el cronómetro arranca después de las cabeceras, así que cuenta
el cuerpo, la descompresión y el `import`, pero no la latencia hasta la primera respuesta.

La caché en disco está a medias, y por eso el arreglo es corto: el directorio ya es
`/tmp/renderer-<etag>`, con el ETag en la clave. Lo que falta es mirar si existe antes de
descargar, en vez de extraer siempre encima.

De la misma variable de módulo cuelga un segundo efecto: la limpieza del artefacto anterior
(`if (loadedRenderer) rmSync(...)`) tampoco se ejecuta nunca en el VPS, porque `loadedRenderer`
siempre llega vacío. Queda un directorio por versión del renderer publicada —no por publicación,
que reutiliza el path del mismo ETag—. Se limpia solo al reiniciar el servicio, porque la unidad
usa `PrivateTmp=yes` y su `/tmp` muere con el proceso.
