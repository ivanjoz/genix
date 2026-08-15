# Generador de historial ERP con fechas pasadas — IMPLEMENTADO

Documentación de uso: **`scripts/GENERATE_ERP_HISTORY.md`**.
Este archivo sólo deja el registro de qué se cambió y por qué.

## El reloj efectivo global

El plan original usaba `HandlerArgs.HistoricalUnix`, un override **por request**. Se descartó:
el ORM escribe `created`/`updated` por su cuenta y no ve el request, así que un registro
backdated salía con la fecha de negocio en el pasado y la de auditoría en el presente.

En su lugar hay un **reloj de proceso**, en la capa más baja que comparten la aplicación y el
driver de base de datos:

```
genix-orm/db/clock.go          Now() · SetHistoricalUnix() · HistoricalUnix() · GENIX_HISTORICAL_UNIX
   ├── genix-orm/scylla        currentManagedUnixTime() → columnas created/updated del ORM
   └── app/db → app/core       core.Now() · core.SUnixTime() · core.FechaUnix()
```

`HandlerArgs.HistoricalUnix` y `core/request_time.go` (los helpers `Effective*`) se eliminaron:
con el reloj global son redundantes y sus 5 call sites volvieron a los helpers normales, que
ahora ya son históricos. El efecto secundario buscado es que **todo** handler quedó backdateable,
incluidos los 3 de órdenes de compra que eran el hueco original.

Los `time.Now()` de latencia, deadlines de red y expiración de token se dejaron intactos:
necesitan el reloj monotónico.

## Archivos tocados

| Archivo | Cambio |
|---|---|
| `genix-orm/db/clock.go` | **Nuevo.** El reloj efectivo (submódulo) |
| `genix-orm/scylla/insert-update.go` | `currentManagedUnixTime()` usa `db.Now()` |
| `backend/db/db.go` | Reexporta `Now` / `SetHistoricalUnix` / `HistoricalUnix` |
| `backend/core/clock.go` | **Nuevo.** `core.Now()`, `core.FechaUnix()`, aviso de arranque |
| `backend/core/helpers.go`, `core/time-helpers.go` | `SUnixTime`, `SUnixTimeMilli`, `SUnix5Min`, `GetFechaUnix(P/I)` y las semanas leen el reloj efectivo |
| `backend/core/request_time.go` | **Eliminado** junto con `HandlerArgs.HistoricalUnix` |
| `backend/logistics/purchase-order-management.go` | Las 3 fechas dejan de venir de `time.Now()` |
| `backend/sales/sale_summary_status.go`, `security/signup.go` | Rango por defecto y week code del reloj efectivo |
| `backend/main.go` | Loguea el reloj congelado al arrancar |
| `backend/tests/sample_records/generate_erp_history.go` + `erp_history_providers.json` | **Nuevo.** El generador |
| `backend/tests/sample_records/generate_sale_orders.go` | Migrado del override por request al global |
| `backend/exec/{main,sample_records}.go` | `fn-generate-erp-history` |
| `scripts/main.go`, `app.sh`, `scripts/GENERATE_ERP_HISTORY.md`, `AGENTS.md` | Dispatcher y documentación |

## Reanudación por paso

El estado vive en `tmp/genix_erp_history_state.json` y se reescribe **después de cada orden de
compra y de cada venta** (no una vez por día): día en curso, OCs completadas, plan de ventas del
día y ventas ya escritas, más el pool de productos y los acumulados. Escritura atómica
(temporal + `rename`). Nada del flujo es idempotente, así que reanudar salta lo ya escrito en
lugar de repetirlo. La firma del plan guardada excluye `reset` y `dryRun`, para que el comando de
reanudación sea el mismo sin `--reset`.

Verificado: corrida de 3 días × 90 ventas matada a mitad del día 20678 (2 OCs y 23 de 90 ventas
hechas); al reanudar terminó en 6 OCs y 270 ventas — exactamente el plan — y la base quedó con
2 OCs por día, sin duplicados.

## Bugs de fondo corregidos

**1. Respuestas decodificadas con `encoding/json`.** `generate_sale_orders.go` leía los bodies de
los handlers con `json.Unmarshal`, pero `core.MakeResponse` los serializa con **minijson** (el
formato compacto `[keys, content]` que decodifica el frontend). El generador de ventas demo moría
en `No se pudo cargar el stock del almacén: json: cannot unmarshal array into ...`. Ahora ambos
generadores usan `decodeResponse`, que va por `minijson.Unmarshal`.

**2. Búsqueda de lotes con `IN`.**
`resolveLotIDsForMovements` (`logistics/product-stock-movement.go`) buscaba los lotes con
`CompanyID.Equals(...) + Hash.In(hashes...)`. `Hash` es un índice global, cuya *capability* no
lleva prefijo de partición, así que la consulta no se enrutaba por el índice y Scylla la veía
como una restricción sobre columna no-clave: la acepta con **un** valor y la rechaza con dos o
más pidiendo `ALLOW FILTERING`. **Recibir una OC con más de un lote fallaba siempre**, no sólo
en este generador. Ahora se consulta un hash por query, en paralelo, cada una por el índice.

## Verificado contra la base real (company 1)

Corrida de 4 días (`--days=4`), comprobada con `fn-db`:

- `purchase_order` — 8 OC, `date` y `created` en el día simulado (08:00 local), status 4, deuda 0
- `sale_order` — 32 ventas repartidas por día, mezcla de estados 1 / 2 / 3 / 4
- `product_stock_lot` — 12 lotes por día, fechados en el día simulado
- `cash_bank_movements` — tipos 6 (pago proveedor), 7 (inyección) y 8 (cobro venta) por día
- `warehouse_product_movement` — 583 movimientos, tipos 1 y 8, en ambos almacenes,
  187 con número de serie y 142 con lote

`cd scripts && go run . check_tables` pasa (42 pares de structs).
