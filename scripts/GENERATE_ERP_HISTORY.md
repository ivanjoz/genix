# generate_erp_history

Genera historial de ERP con **fechas pasadas reales**: por cada día simulado crea órdenes de
compra, las confirma, las paga desde la caja, recibe la mercadería (stock libre, con lote y con
número de serie por unidad) y luego registra las ventas del día.

```bash
cd scripts && go run . generate_erp_history --days=15
./app.sh generate_erp_history --days=15          # equivalente
cd backend && go run . fn-generate-erp-history --days=15   # invocación directa
```

## Cómo se consigue la fecha pasada

No se falsea el reloj del sistema ni hace falta un contenedor. El backend tiene un **reloj
efectivo global**, definido en `genix-orm/db/clock.go` y reexportado por `app/db` y `core`:

```go
core.Now()                    // reloj efectivo (real, o el instante congelado)
core.SetHistoricalUnix(unix)  // congela el proceso en ese segundo unix; 0 vuelve al reloj real
core.HistoricalUnix()         // 0 si no hay override
```

Se puede activar también desde el entorno, sin tocar código:

```bash
GENIX_HISTORICAL_UNIX=1754000000 go run .   # todo lo que escriba el proceso se fecha ahí
```

Todas las fechas persistidas derivan de ese reloj: `core.SUnixTime()`, `core.FechaUnix()`,
`TimeHelper.GetFechaUnix()` y — clave — las columnas `created` / `updated` que **el propio ORM**
escribe (`scylla/insert-update.go`). Por eso un registro backdated sale coherente de punta a
punta y no con la fecha de negocio en el pasado y la de auditoría en el presente.

Lo que **no** cambia son los `time.Now()` de medición de latencia, deadlines de red y
vencimiento de tokens: necesitan el reloj monotónico y un reloj congelado los rompería.

Al arrancar, `main.go` avisa por log si el override está activo: un reloj congelado cambia la
fecha de todo lo que escriba el proceso y nunca debe pasar inadvertido.

El generador mueve el reloj antes de cada escritura (inyección de efectivo 07:00, OC 08:00,
recepción 10:00, ventas 11:00→20:00 hora local) y lo devuelve a 0 al terminar.

## Qué escribe cada día simulado

| Paso | Ruta oficial usada |
|---|---|
| 2 órdenes de compra (una por almacén), 50 productos c/u | `POST.purchase-orders` |
| Confirmación | `PUT.purchase-orders?action=1` |
| Inyección de efectivo si la caja no alcanza (tipo 7) | `POST.cash-banks-movement` |
| Pago al proveedor (tipo 6) | `PUT.purchase-orders?action=3` |
| Recepción: 15 productos con lote, 10 con serie (1 línea por unidad), 25 libres | `POST.purchase-order-entry` |
| 100–150 ventas de 3 a 8 productos | `POST.sale-order` |

Cada llamada arma el JSON exacto que la ruta espera y ejecuta el handler de producción, así que
todas las validaciones y efectos de negocio se aplican igual que desde el frontend.

Estado de las ventas del día: entre 5 y 20 quedan **impagas** y entre 5 y 20 **no entregadas**
(sorteos independientes: una venta puede ser ambas cosas). Se traduce en `ActionsIncluded`:

| | Entregada | No entregada |
|---|---|---|
| **Pagada** | `[2,3]`, `DebtAmount 0` → Status 4 | `[2]`, `DebtAmount 0` → Status 2 |
| **No pagada** | `[3]`, `DebtAmount = Total` → Status 3 | `[]`, `DebtAmount = Total` → Status 1 |

Sólo las entregadas descuentan stock.

## Stock, lotes y series

El generador mantiene un ledger en memoria por almacén con los tres *buckets* que el backend
valida por separado: libre (`ProductStockV2.Quantity`), por lote y por serie (ambos en
`ProductStockDetail`). Cada línea de venta nombra su bucket — `DetailProductLotIDs` o
`DetailProductSkus` — porque si no, la validación mira el bucket libre y falla.

Si aun así el handler responde falta de stock, el ledger se recarga desde
`GET.productos-stock` y la venta se rearma **con otros productos** (hasta 3 intentos).

## Argumentos

| Flag | Default | |
|---|---|---|
| `--days=N` | 15 | días hacia atrás, incluido hoy |
| `--products=N` | 400 | pool fijo, elegido una vez y reutilizado todos los días |
| `--po-products=N` | 50 | productos por orden de compra |
| `--lot-products=N` | 15 | de esos, con lote |
| `--serial-products=N` | 10 | de esos, con serie por unidad |
| `--sales-min` / `--sales-max` | 100 / 150 | ventas por día |
| `--sale-lines-min` / `--sale-lines-max` | 3 / 8 | productos por venta |
| `--unpaid-min` / `--unpaid-max` | 5 / 20 | ventas impagas por día |
| `--undelivered-min` / `--undelivered-max` | 5 / 20 | ventas no entregadas por día |
| `--warehouses=1,2` | 1,2 | una orden de compra por almacén |
| `--reset` | off | ignora el checkpoint y rehace el rango completo |
| `--dry-run` | off | arma e imprime los payloads del primer día y no escribe nada |

Los rangos se validan al inicio (`--lot-products + --serial-products ≤ --po-products`,
`--unpaid-max ≤ --sales-min`, etc.) para no fallar recién a mitad del historial.

## Reanudación

El estado se guarda en **`tmp/genix_erp_history_state.json`** (la carpeta `tmp/` de la raíz del
repo) y se reescribe **después de cada orden de compra y después de cada venta**, no una vez por
día. Si se cierra la terminal o se mata el proceso, se pierde como mucho un registro y la
siguiente corrida sigue exactamente desde ahí: mismo día, misma orden, misma venta.

```json
{
  "UnixDay": 20678,              // día en curso
  "CompletedPurchaseOrders": 2,  // OCs ya hechas de ese día
  "SalesPlanCount": 90,          // plan del día, sorteado una sola vez
  "CompletedSales": 23,          // ventas ya escritas
  "SalesUnpaidFlags": [...],     // qué venta va impaga / no entregada
  "ProductPool": [...],          // los 400 productos, para no re-sortearlos
  "Stats": {...}, "AffectedUnixDays": [...]
}
```

Detalles que importan:

- **Nada de esto es idempotente**: reintentar un día ya escrito duplicaría las OCs y las ventas.
  Por eso el punto de reanudación se guarda *después* de cada escritura, nunca antes.
- La escritura es atómica (archivo temporal + `rename`), así que un proceso muerto a mitad no
  deja un JSON ilegible — que es justo el evento para el que existe el archivo.
- El **pool de 400 productos** y el **plan de ventas del día** viajan en el estado: si se
  re-sortearan al reanudar, la corrida quedaría partida en dos catálogos distintos y el día
  terminaría con otras cantidades de impagas/no entregadas de las que empezó.
- El estado guarda la **firma del plan** (días, productos, rangos, almacenes). Reanudar con
  argumentos distintos se rechaza con un mensaje claro en vez de mezclar dos corridas. `--reset`
  y `--dry-run` no entran en la firma: eligen cómo corre esta invocación, no qué genera — así el
  comando de reanudación es literalmente el mismo, sin `--reset`.
- Al terminar bien, el archivo se borra: si quedara, la siguiente corrida creería que reanuda.

```bash
cd scripts && go run . generate_erp_history --days=15 --reset   # empieza de cero
# … se cierra la terminal …
cd scripts && go run . generate_erp_history --days=15           # continúa donde iba
```

## Requisitos y notas

- Company 1, usuario 1, los almacenes indicados y al menos una caja activa (se toma la de menor
  ID). Proveedores y clientes se siembran solos: 5 proveedores desde
  `backend/tests/sample_records/erp_history_providers.json` y los 50 clientes de
  `sale_order_clients.json`. `POST.client-provider` deduplica, así que repetir la corrida no
  los duplica.
- **Es acumulativo**: no borra nada. Correrlo dos veces duplica el historial de esos días.
- Los resúmenes de venta los reconstruye la acción cron 2, que no corre en un proceso `fn-…`.
  El resultado devuelve `unixDaysToReprocess` con los días tocados.
- Empieza siempre con `--dry-run` y luego un `--days=1`, verificando con
  `fn-db` que `purchase_order.date`, `sale_order.date`, `warehouse_product_movement.date`,
  `product_stock_lot.date` y `cash_bank_movements.date` cayeron en el día esperado.
