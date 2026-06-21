# Plan: Custom CSS por sección (agente)

## Objetivo

Permitir que el agente del builder autore **CSS crudo** para casos que las clases
utilitarias no cubren bien (gradientes, `clip-path`, `mask`, animaciones,
`background` con múltiples capas, etc.). El agente:

1. Pone clases con nombres **semánticos libres** en los elementos:
   `class="... hero-gradient ..."`.
2. Devuelve un bloque de CSS que las define: `.hero-gradient { background: linear-gradient(...) }`.

Al llegar al frontend, cada clase que esté **definida en el CSS y usada en el
HTML** se indexa y se renombra a una clase minificada **única en la página**
(`.x{n}`), reescribiendo tanto el selector como su uso en los elementos. El CSS
resultante se persiste en la BD para que el storefront lo sirva tal cual.

## Decisión de nombrado: `.x{n}` con contador GLOBAL de página

`SectionID` es **posicional** y cambia al reordenar/eliminar (se recalcula en
cada save — `page_content.go:131`), así que NO sirve para hornear en el nombre.

**Solución elegida**: un **contador global de página**, independiente de la
sección. Cada nombre de clase del agente (y cada `@keyframes`) que se indexe se
mapea al siguiente id global → `.x{n}` (p.ej. `.x5`).

- **Asignación**: `nextId = (máx `x{n}` encontrado entre TODAS las secciones
  actuales) + 1`. No se persiste un contador aparte; se deriva al vuelo, así
  sobrevive recargas y reordenamientos sin colisionar.
- **No requiere** campo `Sid` por sección ni check de colisión entre secciones.
- **Reorder-safe**: los nombres no dependen de la posición.
- **Re-edición**: al re-editar una sección se asignan ids nuevos; el `CustomCss`
  viejo de esa sección se reemplaza y el CSS de página se regenera desde las
  secciones actuales, así que no quedan huérfanos.

## Contrato del agente (cómo llega el CSS)

Extender la herramienta `apply_sections`: cada entrada de `sections` pasa de
`{ html }` a `{ html, css? }`.

- `html`: con clases de nombre semántico libre (`hero-gradient`, `fancy-card`…).
- `css`: bloque de CSS crudo que define esas clases. Selectores **de clase**
  (con combinadores/pseudo/media OK: `.hero-gradient:hover`, `.card > span`,
  `@media (...) { .hero-gradient { } }`) y `@keyframes`. Selectores globales
  (`body`, `*`, `:root`, tags sueltos) **se descartan** en el frontend (no se
  renombran ni se aplican) — el agente solo estiliza vía sus propias clases.

Prompt (`prompts.go`): añadir una sección corta "Custom CSS" explicando cuándo
usarlo (solo para lo que las utilidades no logran), que use nombres de clase
propios y los aplique en el HTML, y que devuelva el CSS en el campo `css`.
Mantener Tailwind como vía por defecto.

## Cambios por capa

### Backend — agente (`backend/agent/webpage/`)
- `loop.go`: `SectionEdit` gana `CSS string \`json:"css"\``. `applySections`
  parsea `css` por sección y lo pasa por `PushSections`.
- `tools.go`/`prompts.go`: schema de `apply_sections` con `css` por sección +
  instrucciones.
- `Sink.PushSections` (interface en `loop.go`) y su impl en `chat_ws.go`: la
  estructura de wire `agentSections` lleva `Css` por sección.

### Backend — persistencia (`backend/webpage/types/page_content.go`)
- `SectionContent` gana UN campo CBOR: `CustomCss string`. (No hace falta `Sid`.)
- No se requiere cambio en el manejo del `Css` column: el CSS custom se pliega
  en el string de página por el frontend (ver abajo), igual que hoy con UnoCSS.

### Frontend — tipos y transform
- `section-types.ts`: `SectionData` gana `CustomCss?: string`.
- **Nuevo** `routes/webpage-builder/html-ast/scope-custom-css.ts`:
  - `nextGlobalId(sections)`: escanea las clases `x{n}` existentes en todas las
    secciones (AST + CustomCss) y devuelve `máx + 1` (un `allocId()` monotónico).
  - `scopeCustomCss(cssText, ast, allocId)`:
    1. Recolecta el set de clases realmente **usadas** en el AST (tokens de
       `node.css`).
    2. Parsea el bloque CSS (parser ligero por reglas/bloques con conteo de
       llaves, reutilizable del `splitMediaRules`): para cada **selector de
       clase** cuya clase esté en el set usado, asigna `allocId()` y mapea
       `nombre → x{id}`. Para cada `@keyframes nombre`, asigna `allocId()` y mapea
       `nombre → x{id}`.
    3. Reescribe el CSS (selectores, nombres de `@keyframes` y sus referencias en
       `animation`/`animation-name` dentro del bloque) y el AST (tokens de clase),
       token-aware (no substring).
    4. **Descarta** reglas cuyo selector no referencie ninguna clase usada o que
       sean globales (`body`, `*`, `:root`, tags). Devuelve el CSS saneado.
- `[pageID]/+page.svelte` `applyAgentSections`: tras parsear el HTML a AST, si la
  entrada trae `css`, construir un `allocId` con `nextGlobalId` (sobre las
  secciones actuales; en build-page el contador avanza entre secciones) y llamar
  `scopeCustomCss`, guardar el resultado en `section.CustomCss` y dejar el AST con
  las clases ya renombradas.

### Frontend — render y guardado
- `stores/live-css.svelte.ts` (builder): además del UnoCSS de tokens, concatenar
  `section.CustomCss` de todas las secciones, envuelto en `@layer ec-custom { … }`
  (después de `utilities` para que gane). Inyectado en el mismo `<style>`.
- `services/ecommerce/page-content.svelte.ts` `savePageContent`: al construir el
  `PageCss`, anexar el `@layer ec-custom { …todas las CustomCss… }`. Así el
  backend lo mueve al `Css` column y el storefront lo sirve verbatim (sin tocar
  el flujo actual). `CustomCss` y `Sid` además viajan en `Content` (CBOR) para el
  round-trip del builder.
- Storefront: sin cambios — ya inyecta el `Css` column tal cual.

## Orden de capas CSS

`ec-runtime` (base) → `utilities` (build) → `ec-runtime-media` (responsive) →
**`ec-custom`** (custom del agente, gana sobre utilidades). Confirmar que el
orden declarado en `routes/tailwind.css` incluya `ec-custom` al final.

### Backend — agente (crítico estético)
- `reviewAesthetics` (`loop.go`): incluir el CSS custom de cada sección en el
  input del crítico, y el prompt (`aestheticReviewSystemPrompt`) menciona que
  revise también el CSS crudo (gradiente realmente visible, contraste, que no
  quede invisible/near-white, animaciones sensatas).

## Edge cases / seguridad
- **Solo clases**: únicamente se indexan/aplican selectores de clase usados en el
  HTML. Globales (`body`, `*`, `:root`, tags) y clases definidas-pero-no-usadas se
  descartan. Evita que el agente pinte estilos globales.
- **`@keyframes`**: permitido; el nombre se namespacea a `x{n}` y sus referencias
  (`animation: x{n} …`) se reescriben dentro del mismo bloque.
- **Build-page**: el contador global avanza secuencialmente entre las secciones
  del listado, así no colisionan entre sí.
- **Re-edición**: al re-editar se asignan ids nuevos; el `CustomCss` previo de
  esa sección se reemplaza y el CSS de página se regenera, sin huérfanos.
- **Sin css**: si la entrada no trae `css`, comportamiento idéntico al actual.
- **`@media`/`@keyframes`**: el regex de scoping solo toca `.custom-N`;
  `@keyframes nombre` quedaría global — definir convención (prefijar nombres de
  keyframes con `custom-` también, o documentar que se permiten globales).

## Archivos tocados (resumen)
- `backend/agent/webpage/loop.go`, `tools.go`, `prompts.go`
- `backend/agent/chat_ws.go` (wire `agentSections`)
- `backend/webpage/types/page_content.go` (`CustomCss`, `Sid`)
- `frontend/webpage/renderer/section-types.ts`
- `frontend/routes/webpage-builder/html-ast/scope-custom-css.ts` (nuevo)
- `frontend/routes/webpage-builder/[pageID]/+page.svelte`
- `frontend/routes/webpage-builder/stores/live-css.svelte.ts`
- `frontend/services/ecommerce/page-content.svelte.ts`
- `frontend/webpage/routes/+page.svelte` (verificar layer order; probablemente sin cambio)

## Decisiones (cerradas)
1. ✅ Id global de página `.x{n}` (no depende de la sección).
2. ✅ Nombres de clase libres; se indexa la clase si está definida en el CSS y
   usada en el HTML. Selectores globales se descartan.
3. ✅ `@keyframes` permitido, con nombre namespaceado a `x{n}`.
4. ✅ El crítico estético revisa también el CSS custom.
