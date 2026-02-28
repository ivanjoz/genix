# Plan: Migrar `sendServiceMessage` a respuesta directa por `fetch` (sin `message` listener para cache)

## Objetivo
- Eliminar la dependencia de `navigator.serviceWorker.addEventListener('message', ...)` para el flujo de cache (`accion = 3`), usando solo `fetch('/_sw_...')` y su `Response` JSON.
- Mantener el comportamiento funcional de `GetHandler` (`offline -> updateOnly/refresh`) sin romper merge delta ni invalidación por `refreshRoutes`.

## Alcance (sin WebRTC)
- Incluido: acciones síncronas que hoy usan `sendServiceMessage` y esperan un resultado directo (`3`, `15`, `24`, `26`, y opcionalmente `11`, `12`, `14`, `22`, `23`).
- Excluido: WebRTC (`30`, `31`, `32`, `40`) por deprecado.

## Diagnóstico actual
- Hoy `/_sw_` ejecuta el handler correcto, pero:
  - Responde al `fetch` con `{ ok: 1 }`.
  - Envía el payload real por `client.postMessage(...)`.
- En cliente, `sendServiceMessage`:
  - Registra resolver por `reqID`.
  - Hace `fetch` con reintentos por timeout.
  - Espera el resultado real vía listener de `message`.
- Conclusión: para `accion=3` no es obligatorio `postMessage`; el SW ya tiene el resultado y puede retornarlo en el `Response` del mismo `fetch`.

## Diseño objetivo
1. `/_sw_` retorna directamente `{ ...response, __response__, __req__ }` en el body JSON del `Response`.
2. `sendServiceMessage` consume `await fetch(...).then(r => r.json())` y resuelve esa promesa sin mapa de resolvers ni listener para estas acciones.
3. `fetchCache/fetchCacheParsed` permanecen con la misma interfaz (`content`, `error`, `isEmpty`, `notUpdated`).
4. `setFetchProgress`:
   - Opción A (recomendada): deshabilitar temporalmente el progreso por chunks y simplificar.
   - Opción B: mantenerlo con un endpoint/evento aparte si se requiere UX de progreso.

## Fases

### Fase 1: Corte directo (breaking)
- En SW (`service-worker-cache.ts`), para `/_sw_`:
  - Eliminar envío de resultado por `client.postMessage(...)` para acciones síncronas.
  - Retornar siempre payload JSON directo en `Response`.
- En cliente (`sw-cache.ts`):
  - Reescribir `sendServiceMessage` para request/response directo.
  - Eliminar `serviceWorkerResolverMap`.
  - Eliminar `successfulResponses`.
  - Eliminar polling/reintentos atados a eventos `message`.
- Simplificar control de duplicados en SW:
  - Corregir clave usada en `usedRequestIDs` (usar `clientReqID` consistentemente).

### Fase 2: Limpieza total
- Eliminar listener `navigator.serviceWorker.addEventListener('message', ...)` del módulo de cache.
- Eliminar ramas de código para `__response__ = 3` y `__response__ = 5` en cliente.
- Mantener solo utilidades de inicialización SW necesarias para `register/ready`.
- Actualizar docs (`SERVICES_GUIDE.md` o doc corto de transporte SW) indicando el nuevo contrato directo.

## Riesgos y mitigaciones
- Riesgo: pérdida de indicador de progreso (`__response__ = 5`).
  - Mitigación: aceptar retiro temporal o implementar estrategia de progreso desacoplada.
- Riesgo: respuestas grandes de cache.
  - Mitigación: ya se serializan/transportan hoy por `postMessage`; validar memoria/latencia en dataset grande.
- Riesgo: duplicados por reintentos.
  - Mitigación: al hacer request/response directo, reducir reintentos automáticos y usar timeout controlado en cliente.

## Criterios de aceptación
- `EmpresaParametrosService` con `useCache` funciona sin listener de mensajes para obtener datos (`offline` y `updateOnly`).
- `fetchCacheParsed` sigue respetando:
  - `offline` con `null` cuando cache vacío.
  - `updateOnly` con `null` cuando `notUpdated`.
- `POST(...refreshRoutes)` sigue marcando rutas para refresh sin regresión.
- No hay logs de timeout/reintento infinito en `sendServiceMessage`.

## Plan de pruebas
1. Cargar pantalla con cache vacío:
   - Esperado: primer offline `null`, luego datos de red.
2. Recargar con cache existente dentro de TTL:
   - Esperado: render inmediato desde cache.
3. Forzar cambio en backend (`updated` mayor):
   - Esperado: delta merge correcto en cache.
4. Ejecutar `refreshRoutes` tras POST:
   - Esperado: próxima sincronización fuerza red.
5. Simular error 500 en endpoint:
   - Esperado: `fetchCacheParsed` muestra `Notify.failure` y retorna `null`.

## Decisión recomendada
- Ejecutar corte directo en una sola iteración, validar `empresa-parametros`, y limpiar legacy en el mismo PR.


Implementado con **forma 1**: progreso por `ReadableStream` en cliente, sin listener `message`.

Qué quedó:
- `sendServiceMessage` ahora hace request/response directo y parsea el body por chunks.
- En cada chunk recibido, si `accion === 3`, llama `setFetchProgress(value.length)`.
- Se eliminó la dependencia del canal `postMessage` para resolver requests de cache.
- El SW `/_sw_` devuelve el payload real en el `Response` JSON.

Archivos:
- [frontend/libs/sw-cache.ts](/home/ivanjoz/projects/genix/frontend/libs/sw-cache.ts:36)
- [frontend/libs/workers/service-worker-cache.ts](/home/ivanjoz/projects/genix/frontend/libs/workers/service-worker-cache.ts:161)
- [frontend/libs/workers/service-worker.ts](/home/ivanjoz/projects/genix/frontend/libs/workers/service-worker.ts:67)

Cómo se emite progreso ahora:
1. Cliente llama `fetch('/_sw_?...')`.
2. Cliente lee `response.body.getReader()`.
3. Por cada chunk: `setFetchProgress(chunk.length)`.
4. Al finalizar, parsea JSON y cierra `fetchEvent(reqID, 0)`.

Nota práctica: si la respuesta del SW llega en un solo chunk grande, verás pocos updates (o uno); eso depende del streaming real del navegador/red para ese payload.
