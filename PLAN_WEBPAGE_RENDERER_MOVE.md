# Plan — mover el renderer de webpages a la raíz, con tests y ejecución local

Objetivo: que toda la lógica del renderer viva en un solo lugar, tenga tests propios, y que
fuera de un entorno Lambda el backend la ejecute como un archivo `.js` local.

---

## 0. Hallazgo que condiciona el plan

`frontend/webpage/lambda/` **no es una unidad**. Contiene dos archivos de naturaleza opuesta:

| Archivo | Qué es | ¿Se mueve? |
|---|---|---|
| `handler.mjs` | El handler del Lambda. Cero dependencias, se empaqueta **solo** en el zip de la función (`cloud/webpage-renderer.go:38`). Habla HTTP contra R2. | **Sí** |
| `renderer-entry.js` | El entry del bundle SSR. Hace `import { Server } from '../.svelte-kit/output/server/index.js'` y esbuild lo bundlea dentro del **artefacto** (`scripts/build-renderer.mjs:76`). | **No** |

`renderer-entry.js` es parte de la app SvelteKit, no del Lambda: sus imports son relativos a
la salida de build de `frontend/webpage/`. Moverlo a la raíz rompería esos imports y dejaría
un archivo que depende de `.svelte-kit/output` fuera de la app que lo genera.

**Decisión: se mueve `handler.mjs`; `renderer-entry.js` se queda donde está.** La carpeta
`lambda/` desaparece como tal y `renderer-entry.js` sube un nivel a `frontend/webpage/`.

---

## 1. Estructura destino

```
webpage-renderer/              (nuevo, en la raíz)
  handler.mjs                  el handler — misma lógica, + exports para test
  cli.mjs                      entrada CLI que usa el backend fuera de Lambda
  handler.test.mjs             tests (bun test descubre .test.mjs, verificado)
  README.md                    los dos zips, el contrato del evento, cómo correr en local

frontend/webpage/
  renderer-entry.js            movido desde lambda/ (un nivel arriba)
```

El zip de la función sigue llevando **un solo archivo**: `CompileRendererToS3` crea la entrada
`handler.mjs` y nada más, así que `cli.mjs` y los tests nunca viajan a AWS.

---

## 2. `cli.mjs` — ejecución local

Archivo aparte, no una rama dentro de `handler.mjs`: mantiene el handler puro y el zip de la
función mínimo.

```js
// Lee el evento por stdin, llama a render(), escribe el resultado JSON por stdout.
// stdout es SOLO el JSON del resultado; los logs del handler van a stderr para que el
// backend pueda parsear la respuesta sin filtrar líneas.
import { render } from './handler.mjs';
```

Contrato con Go:
- **stdin**: el mismo JSON que hoy recibe el Lambda (`{ companyID, hostname, pages, forceAssets }`).
- **stdout**: solo `WebpageRenderResult` en JSON.
- **stderr**: logs.
- **exit code**: 0 OK, 1 error (mensaje en stderr).

Requiere redirigir `console.log` del handler a stderr dentro de `cli.mjs`, para no contaminar
stdout. Es la única razón por la que el handler necesita un wrapper y no se invoca directo.

---

## 3. Go — un solo punto de entrada, dos caminos

`backend/cloud/webpage_renderer.go`:

```go
func InvokeWebpageRenderer(request WebpageRenderRequest) (WebpageRenderResult, error) {
    if core.Env.IS_SERVERLESS {
        return invokeWebpageRendererLambda(request)   // el cuerpo actual, sin cambios
    }
    return runWebpageRendererLocally(request)          // nuevo
}
```

`runWebpageRendererLocally`:
1. Resuelve la raíz del repo.
2. `exec.Command("node", filepath.Join(root, "webpage-renderer/cli.mjs"))`.
3. Escribe el payload por stdin, captura stdout y stderr por separado.
4. Loguea stderr vía `core.Log` y deserializa stdout en `WebpageRenderResult`.
5. Pasa por `cmd.Env` las cinco variables que el handler exige:
   `RENDERER_ZIP_URL`, `FRONTEND_CDN`, `CLOUDFLARE_ACCOUNT`, `CLOUDFLARE_TOKEN`, `CLOUDFLARE_BUCKET`.

Ventaja de reutilizar `InvokeWebpageRenderer`: `DeployCompanyWebpage` y cualquier futuro
llamador no cambian ni se enteran del entorno.

### 3.1 Dos piezas que hoy no existen y hay que añadir

**a) `core.Env.WEBPAGE_RENDERER_URL`.** Hoy solo vive en `cloud/main.go:36` (`DeployParams`,
leído de `credentials.json`) y nunca llega al backend. Sin él, la ejecución local no sabe de
dónde descargar el artefacto. Se añade a `core.EnvStruct` (`backend/core/security.go:95-120`)
con el mismo default que `cloud/webpage-renderer.go:27`
(`https://genix-dev.un.pe/webpage-renderer.zip`).

**b) Un localizador de la raíz del repo accesible desde `cloud`.** `findGenixProjectRoot` existe
pero es privado de `backend/exec` (`cloudflare_worker_deploy.go:157`), y `cloud` no puede
importar `exec` porque `exec` ya importa `cloud`. Se mueve a `core` como `core.FindProjectRoot()`,
prefiriendo `GENIX_REPOSITORY_ROOT` (que `main.go:290` ya imprime en el arranque) y cayendo al
recorrido hacia arriba buscando `deploy.sh` + `AGENTS.md`. `backend/exec` pasa a usar el helper
de `core` en vez de su copia.

---

## 4. Tests — `webpage-renderer/handler.test.mjs`

El handler ya trae la costura que lo hace testeable: `CLOUDFLARE_API_BASE` es redirigible
(`handler.mjs:26`, comentario explícito "para ejercitar el handler completo contra un R2
simulado") y `RENDERER_ZIP_URL` es un `fetch`. Los tests montan un `node:http` local que hace
de **origen del zip** y de **R2 falso**, guardando cada PUT en un mapa.

Casos:

| # | Qué verifica |
|---|---|
| 1 | Un render completo escribe `websites-html/<host>/index.html` y `websites-html/<host>/about/index.html` |
| 2 | Los assets van a `websites/<companyID>/_app/…` con `immutable`; `sw.js` con `max-age=0, must-revalidate` |
| 3 | Segunda invocación con el mismo `buildId`: **no** resube assets (marcador `.renderer-build`) |
| 4 | `forceAssets: true` los resube aunque el marcador coincida |
| 5 | `304` del origen del zip reusa el artefacto ya extraído (no vuelve a inflar) |
| 6 | `normalizePagePath` normaliza `/`, `about`, `/about/` y **rechaza** rutas con caracteres inválidos |
| 7 | `applyHtmlRewrites` lanza cuando una regla `required` no casa |
| 8 | `readZipEntries` lee entradas *stored* (método 0) e *inflate* (método 8), y lanza sin End of Central Directory |
| 9 | Validación del evento: `companyID` <= 0, hostname inválido, `pages` vacío, variable de entorno ausente |
| 10 | Una página que devuelve status != 200 aborta con error |

Los casos 6-8 necesitan que `normalizePagePath`, `applyHtmlRewrites`, `readZipEntries` y
`contentTypeFor` pasen de privados a exportados. Es el único cambio de superficie del handler;
la lógica no se toca.

Para los casos 1-5 hace falta un zip mínimo construido en el propio test (manifest.json +
render.mjs + assets/ + site/sw.js) con método *stored*, para no depender de una librería de
compresión.

---

## 5. Referencias a actualizar

| Archivo | Cambio |
|---|---|
| `cloud/webpage-renderer.go:21` | `rendererHandlerPath` → `/webpage-renderer/handler.mjs` |
| `scripts/build-renderer.mjs:76` | entry → `resolve(appDir, 'renderer-entry.js')` |
| `frontend/webpage/cloudflare/serve-worker.js:3` | comentario que apunta a `frontend/webpage/lambda/handler.mjs` |
| `PLAN_WEBPAGE_LAMBDA_RENDERER.md` | rutas en el diagrama y en el texto |
| `frontend/webpage/ECOMMERCE.md` | ruta del handler |
| `backend/core/security.go` | nuevo `WEBPAGE_RENDERER_URL` |
| `backend/exec/cloudflare_worker_deploy.go:157` | `findGenixProjectRoot` → `core.FindProjectRoot` |
| `AGENTS.md` / `CLAUDE.md` | si mencionan la ruta |

`.github/workflows/ci-deploy.yml:77-78` **no cambia**: construye el artefacto, que sigue
saliendo de `frontend/webpage/`.

---

## 6. Verificación

1. `cd backend && go build ./... && go vet ./cloud/... ./core/... ./exec/...`
2. `cd webpage-renderer && bun test`
3. `cd frontend && bun scripts/build-renderer.mjs --skip-build` — que el artefacto siga
   construyéndose tras mover `renderer-entry.js`.
4. `cd cloud && go build ./...` (el CLI de deploy lee el nuevo `rendererHandlerPath`).
5. Humo local: `cd backend && go run . fn-deploy-company-webpage <companyID>` sobre una company
   con dominio, verificando que sale por el camino `node` y no por el `Invoke` de AWS.

---

## 7. Fuera de alcance (decidir aparte)

- **Unificar la subida a R2.** Hoy `handler.mjs:248` y `backend/cloud/s3.go:340` hablan la misma
  API REST desde dos lenguajes. Este plan no lo toca: el handler tiene que seguir siendo
  autónomo dentro del Lambda de Node.
- **Limpiar `websites-html/<hostname-viejo>/` en R2** al cambiar de dominio, y disparar un
  re-render sobre el hostname nuevo. Encaja con la acción de cron que acabamos de añadir
  (`backend/webpage/domain_cleanup_cron.go`), pero es un cambio de comportamiento aparte.
