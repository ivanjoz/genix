# Plan: hacer que el tick de EventBridge ejecute las acciones programadas en Lambda

## Decisiones (acordadas)

| Tema | Decisión |
|---|---|
| Payload | `Input: '{"body":"{\"exec\":10}"}'`. El evento se deserializa como `APIGatewayV2HTTPRequest`, así que el JSON tiene que ir dentro de `body` para que llegue a `request.Body`. Mismo patrón que el branch `{"fn_exec":`. |
| Target | `BackendFunction` (256 MB), el que ya usa la regla. No se toca `BackendHeavyFunction`. |
| Regla | Se reutiliza `CronRule`: mismo logical ID y mismo `Name` físico, solo cambia el `Input`. No se crea una regla nueva ni se reemplaza el recurso. |
| Cadencia | Sigue en `cron(*/10 * * * ? *)`. El `10` del payload es el marcador de cadencia; se loguea, no altera la ventana de lookback. |

## Diagnóstico: qué está roto hoy

Tres roturas encadenadas. El detalle completo está en el análisis previo; resumen con referencias:

1. **`"exec:cron"` no lo reconoce nadie.** Cero coincidencias del string en Go. El único branch de exec en `backend/main.go:40` compara contra el prefijo `{"fn_exec":`; `"exec:cron"` (9 bytes) falla longitud y prefijo, cae al camino HTTP con `route = "MISSING"` (`main.go:58`) y muere en `core.CheckUser` (`main-handlers.go:127`) por falta de Authorization. Cada 10 min: una invocación, un cold start, una conexión a Scylla y un error.
2. **`ExecHandlersCron` está vacío.** Declarado en `backend/exec/main.go:32` y nunca poblado. El bloque `args.FuncToExec == "cron"` de `main-handlers.go:287-335` itera cero veces y retorna `FuncResponse{}`. Código muerto completo, con un bug latente en la línea 297 (lee `exec.ExecHandlers[hourMin]` en vez de `ExecHandlersCron[hourMin]`).
3. **El mecanismo real nunca corre en Lambda.** `core.StartCronWatcher` (`core/cron-action-scheduler.go:112`) está detrás de `!IS_SERVERLESS && !IS_LOCAL` (`main.go:295-300`), o sea solo VPS. Mientras tanto `sales/sale_order_create.go:244` inserta filas en `cron_actions` en cada orden creada: quedan en `Status = 0` para siempre, sin ejecutarse ni limpiarse (`CronAction` no tiene TTL). Y `business.ScheduleProductsDbRebuildCron()` se siembra en ese mismo bloque VPS-only, así que en Lambda la cadena de 30 min del rebuild de productos nunca arranca.

## Fuera de alcance

- La query `UnixMinutesFrame.Between(...).AllowFilter()` sobre la partition key de `cron_actions` (`core/cron-action-scheduler.go:138`): en Scylla es un scan de cluster. Es un problema preexistente que también afecta al VPS y se arregla aparte (13 queries por frame concreto en vez de un range).
- `BackendHeavyFunction`, las Function URLs, el frontend, la tabla DynamoDB.
- El resto de `ExecHandlers` (las funciones `fn-*` invocadas a mano). Solo se borra `ExecHandlersCron`.

---

## 1. `backend/core/cron-action-scheduler.go` — extraer el ejecutor

`runCronWatcherTick` hoy mezcla dos cosas: el watermark `lastUnixMinutesFrame` (que existe porque el ticker del VPS corre cada minuto y los frames son de 5) y la ejecución en sí. En Lambda el watermark no aplica: cada invocación es un contenedor distinto o reciclado, y la regla ya garantiza una sola llamada cada 10 min.

Separar en dos:

```go
// RunPendingCronActions ejecuta las acciones pendientes de la ventana de lookback y
// devuelve cuántas se procesaron. Sin watermark: el llamador decide la cadencia.
func RunPendingCronActions() int { /* cuerpo actual de runCronWatcherTick, líneas 132-198 */ }

// runCronWatcherTick conserva el watermark del ticker de un minuto del VPS.
func runCronWatcherTick() {
    currentUnixMinutesFrame := int32(time.Now().Unix() / fiveMinuteFrameLength)
    if currentUnixMinutesFrame == lastUnixMinutesFrame {
        return
    }
    RunPendingCronActions()
    lastUnixMinutesFrame = currentUnixMinutesFrame
}
```

`RunPendingCronActions` cuenta las ejecuciones exitosas (donde hoy se llama `markCronActionRowsAttempted_(1)`) y las devuelve. La ventana de lookback se queda en 12 frames (60 min): no depende de la cadencia, existe para reintentar las acciones que fallaron y quedaron en `Status = 0`.

## 2. `backend/main.go` — branch del tick en `runLambdaRequest`

Junto al branch de `{"fn_exec":` (línea 40), antes de resolver la ruta HTTP:

```go
// Tick programado de EventBridge: la regla manda {"exec":<minutos>} como body. No hay
// ruta HTTP detrás, así que ejecuta las acciones pendientes de cron_actions y responde.
if strings.HasPrefix(request.Body, `{"exec":`) {
    return runScheduledCronTick(request.Body)
}
```

`runScheduledCronTick` vive en `main.go` porque necesita importar `business` (`core` no puede: ciclo). Hace, en orden:

1. `core.Env.LOGS_FULL = true`. Necesario: `clearEnvVariables()` lo pone en `false` al entrar, y en serverless `core.Log` descarta todo lo que no empiece con `*` o contenga "error"/"warn" (`core/logs.go:34-42`). Sin esto las líneas `StartCronWatcher action executed:` no llegan a CloudWatch y el tick es una caja negra.
2. `business.ScheduleProductsDbRebuildCron()` dentro de un `recover` — resiembra la cadena de 30 min del rebuild de productos, que en Lambda nunca se sembró. Es idempotente: `ScheduleCronAction` deduplica contra la fila pendiente del mismo frame (`core/cron-action-scheduler.go:91-103`), así que llamarlo cada 10 min no crea duplicados. Se protege con `recover` porque `ScheduleCronAction` hace `panic` ante un error de DB.
3. `executedCount := core.RunPendingCronActions()`.
4. Responder `{"executed":<n>}` reusando el mismo cierre que el branch `fn_exec` (`core.HandlerResponse` → `MakeStreamingResponseFinal` / `MakeResponseFinal` según `core.Env.LAMBDA_RESPONSE_STREAMING`). EventBridge descarta la respuesta, pero mantener la forma correcta evita un error del runtime.

El número del payload (`10`) se parsea solo para la línea de log del tick; no se usa para nada más.

## 3. `backend/main-handlers.go` — borrar el mecanismo muerto

- Eliminar el bloque `if args.FuncToExec == "cron" { ... }` (líneas 287-335) y el tipo local `FuncToInvoke` (279-284). El `else if len(args.FuncToExec) > 0` pasa a ser el único branch, un `if`.
- Eliminar `GetFunctionName` (249-251): su único uso estaba en ese bloque.
- Quitar los imports `reflect` y `runtime`, que quedan sin uso. `strings` y `exec` siguen usándose.

## 4. `backend/exec/main.go` — borrar `ExecHandlersCron`

Eliminar la línea 32. Sin el bloque del paso 3 no le queda ningún lector.

## 5. `cloud/template.yml` — nuevo Input

En `CronRule` (líneas 248-259), única línea funcional que cambia:

```yaml
Input: '{"body":"{\"exec\":10}"}'
```

Y corregir el comentario de encima, que hoy afirma algo falso ("The backend recognises this exact body"): pasa a decir que el body es el que `runLambdaRequest` detecta por el prefijo `{"exec":` y que dispara las filas pendientes de `cron_actions`. `Name`, `ScheduleExpression`, `Targets[0].Arn` y `CronRuleInvokePermission` no se tocan — al conservar el logical ID, CloudFormation hace un update in-place, no un reemplazo.

## 6. `DEPLOYMENT.md` — actualizar la descripción

La línea "Regla de EventBridge que invoca la Lambda pequeña cada 10 minutos con `{"body":"exec:cron"}`" queda desactualizada. Reemplazar por el payload nuevo y una frase de qué ejecuta.

---

## Verificación

1. `cd backend && go build ./...` — confirma que no quedan referencias a `ExecHandlersCron` ni imports huérfanos.
2. Local, con el server levantado: `curl -X POST localhost:3589 -d '{"exec":10}'`. **No pasa todavía por el branch nuevo**: `LocalHandler` (`main.go:139`) tiene su propia copia de la detección y solo mira `{"fn_exec":`. Decidir en la implementación si se replica ahí el branch (una llamada más a la misma función) o si en local se prueba solo con el ticker del VPS. Recomiendo replicarlo: son 3 líneas y hace el tick probable sin desplegar.
3. Desplegar con `cd cloud && go run . 3` (stack) y `go run . 1` (código). Esperar al siguiente múltiplo de 10 min y revisar el log group `/aws/lambda/<APP_NAME>-backend`: debe aparecer la línea del tick y, si había pendientes, un `StartCronWatcher action executed:` por acción.
4. `GET.cron-actions-scheduled` (`backend/config/cron-actions-scheduled.go`) debe mostrar la cola de `Status = 0` bajando en vez de creciendo.

## Riesgos

- **256 MB para `RebuildProductsDbHandler`.** Reconstruye el `.db` de todas las companies dirty en una sola invocación (`business/product-ecommerce-cron.go:33-42`). Con la cola atrasada que existe hoy, el primer tick después del deploy procesa todo lo acumulado de golpe. Si aparece un OOM o un timeout de 300s en CloudWatch, la salida es mover el target de la regla a `BackendHeavyFunction`: un cambio de una línea en el template.
- **Backlog acumulado.** Las filas viejas de `cron_actions` con `Status = 0` fuera de la ventana de 60 min no se ejecutan nunca (correcto: reprocesar resúmenes de ventas de hace meses no aporta), pero tampoco se borran. Limpiarlas es trabajo aparte; conviene mirar el volumen real antes de decidir.
- **Ejecución concurrente.** Si un tick tarda más de 10 min, el siguiente arranca encima. La ventana de lookback compartida hace que ambos vean las mismas filas pendientes y una acción se ejecute dos veces. Hoy no hay lock. Aceptable para las dos acciones registradas (ambas idempotentes: recalculan desde el origen), pero hay que tenerlo presente al registrar acciones nuevas.
