# webpage-renderer

Renderizador de las tiendas (webpages de company). Descarga el artefacto SSR, prerenderiza las
páginas de **una** company y publica el resultado en Cloudflare R2.

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

## Qué se escribe en R2

| Qué | Clave | Cache-Control |
|---|---|---|
| HTML por página | `websites-html/<hostname>/<ruta>/index.html` | `public, max-age=60` |
| js/css | `websites/<companyID>/_app/…` | `immutable` (llevan hash de contenido) |
| `favicon.ico` | `websites-html/<hostname>/favicon.ico` | `max-age=86400` |
| `sw.js` | `websites-html/<hostname>/sw.js` | `max-age=0, must-revalidate` |
| marcador | `websites/<companyID>/.renderer-build` | `no-store` |

`sw.js` va bajo el hostname y no bajo el CDN porque un service worker **tiene que ser
same-origin**. Y no puede cachearse: es lo que gobierna las actualizaciones del sitio.

El marcador guarda el `buildId` publicado: si coincide, los assets no se resuben y editar una
página cuesta solo los PUT del HTML. `forceAssets: true` lo ignora.

Quien sirve todo esto es el Worker `frontend/webpage/cloudflare/serve-worker.js`, que resuelve
`<hostname>/<ruta>` a la misma clave que escribe el handler. Los dos comparten el prefijo
`websites-html`, así que **cambiar `HTML_KEY_ROOT` obliga a cambiarlo en ambos**.

## Variables de entorno

Las cinco son obligatorias y el handler aborta si falta alguna:

`RENDERER_ZIP_URL`, `FRONTEND_CDN`, `CLOUDFLARE_ACCOUNT`, `CLOUDFLARE_TOKEN`, `CLOUDFLARE_BUCKET`.

En Lambda las inyecta `UpdateRendererEnviromentVariables` (`cloud/webpage-renderer.go`) tras cada
deploy. En local las pasa el backend desde `core.Env`.

`CLOUDFLARE_API_BASE` es opcional y solo existe para los tests: redirige la API de R2 a un
servidor local.

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

Ninguno rompe el camino feliz. Están aquí porque los cuatro fallan **en silencio**: el backend
responde que publicó, y el problema aparece más tarde y en otro sitio.

### El Worker no se despliega solo

`serve-worker.js` se sube únicamente con `fn-deploy-cloudflare-worker` (`./deploy.sh 10`, o el
`6`, que además hace tablas y datos iniciales). Guardar un dominio **no** lo despliega: registra el
hostname, sube el HTML y nada más.

Así que al cambiar `serve-worker.js` hay que desplegarlo a mano. Si se olvida, las tiendas
devuelven 404 mientras el backend informa de una publicación correcta —el render y la subida sí
funcionaron—, y el 404 parece del renderer cuando en realidad el Worker está sirviendo una versión
vieja que busca otras claves.

Agregar o cambiar un dominio **no** requiere desplegar nada: el Worker es uno solo, común a todas
las tiendas, y resuelve el hostname en cada request. El alta y la baja del Custom Domain las hace
el propio handler (`provisionStorefrontDomain` / `removeStorefrontDomain`).

Un GET al hostname después de publicar convertiría esto en un error explícito.

### El nombre del Worker está declarado dos veces

`genix-storefront` aparece en `backend/exec/cloudflare_worker_deploy.go` y en
`backend/webpage/cloudflare_domain.go`. Hoy coinciden. Si divergen, los dominios se adjuntan a un
Worker distinto del que se despliega, con el mismo 404 del punto anterior y ninguna pista de por
qué. Es una constante compartida y debería estar definida una sola vez.

### Una página sin snapshot se publica vacía

El SSR pide el contenido a `live/pages/<companyID>-<pageID>.json` en el CDN. Si no existe, el
`load()` de la tienda captura el 404, devuelve secciones vacías y el render **sigue adelante**: se
publica un HTML válido pero sin contenido, y el resultado dice `pages: 2` como si nada. En el log
solo queda un `[StorePage] webpage load failed`, que no llega a la respuesta.

Se reconoce por el tamaño: una página vacía pesa ~10 KB contra los ~27 KB de una con contenido.

### El artefacto se vuelve a descargar en cada publicación (VPS)

`loadRenderer()` compara el ETag y reutiliza lo extraído, pero `loadedRenderer` es una variable de
módulo: solo sirve mientras el proceso siga vivo. En Lambda caliente ahorra el trabajo; en el VPS
cada publicación arranca un `node` nuevo, así que el 304 nunca llega a usarse y se bajan y
descomprimen los ~460 KB otra vez. Es lo que mide el `en NNNms` del log `artefacto cargado`: entre
decenas de ms y algo más de un segundo según la red. No es un problema — pero una caché en disco
bajo `/tmp` con el ETag en la clave lo dejaría en cero.
