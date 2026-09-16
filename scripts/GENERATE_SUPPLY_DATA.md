# generate_supply_data

Siembra los **100 proveedores estáticos** y la **configuración de abastecimiento** de los primeros
N productos: stock mínimo, ventas/día estimadas y 2 proveedores por producto con capacidad, tiempo
de entrega y precio. Es lo que llena la página `logistics/purchase-management`.

```bash
cd scripts && go run . generate_supply_data
./app.sh generate_supply_data --products=2000          # equivalente
cd backend && go run . fn-generate-supply-data --dry-run  # invocación directa
```

## Qué escribe

| Paso | Ruta oficial usada |
|---|---|
| 100 proveedores desde `supply_providers.json` | `POST.client-provider` |
| Abastecimiento de 2000 productos, en lotes de 500 | `POST.product-supply` |
| 100 insumos y materiales desde `supply_materials.psv` | `POST.supply-material` |
| 100 activos desde `assets.psv` | `POST.asset` |
| Inyección de efectivo si la caja no alcanza (tipo 7) | `POST.cash-banks-movement` |
| Pagos de los activos comprados | `POST.asset-payment` |
| Depreciación de todo lo que ya venció | `POST.asset-depreciation-run` |

Los archivos están en `backend/tests/sample_records/`. Cada llamada arma el JSON que la ruta espera
y ejecuta el handler de producción, así que todas las validaciones se aplican igual que desde el
frontend.

## Insumos y activos

Los datos son estáticos y viven en dos `.psv` (valores separados por `|`, con fila de cabecera):

```text
supply_materials.psv   Name|SKU|Price|UnitID|CurrencyID|DepreciationMonths
assets.psv             MaterialSKU|Name|Description|AcquisitionValue|Quantity|SerialNumber
```

`supply_materials.psv` trae **40 equipos depreciables** (`DepreciationMonths > 0`: laptops, cocinas
industriales, montacargas, una camioneta…) y **60 consumibles** (`0`: embalaje, limpieza, útiles,
materia prima). Sólo los depreciables pueden convertirse en activos — `POST.asset` rechaza un
insumo sin esquema de depreciación.

`assets.psv` trae 100 adquisiciones, cada una apuntando a un SKU depreciable. Una fila con
`SerialNumber` genera **un activo con serie**; una sin serie genera **un activo agrupado** de
`Quantity` unidades. Así una corrida ejercita las dos granularidades que `POST.asset` distingue, y
cada fila produce exactamente un registro de activo.

De los 100 activos:

- Los **primeros 50 son comprados** (`PurchaseAmount` = valor en libros) y los **últimos 50 son
  donados** (`PurchaseAmount = 0`, aportados por accionistas). Un donado no debe nada y nunca es
  pagable, pero sí vale y sí deprecia.
- De los 50 comprados, aproximadamente **25 quedan pagados por completo, 15 parcialmente** (entre
  30 % y 70 %) y **10 sin pagar**, para que la página muestre los tres estados de `PaymentStatus`.

Las fechas de adquisición se reparten entre **30 y 1080 días atrás**. Esto no es decorativo: la
depreciación es perezosa y sólo escribe los períodos ya vencidos, así que un activo comprado hoy
no tendría ni un asiento y su cronograma saldría vacío. Con tres años de rango salen activos con
un par de períodos, otros con treinta y algunos ya totalmente depreciados.

Un test (`generate_assets_data_test.go`) valida que los dos `.psv` sean consistentes entre sí:
que cada activo apunte a un material que existe y es depreciable, que las series no se repitan y
que estén representadas ambas granularidades.

Por producto:

| Campo | Rango |
|---|---|
| `MinimunStock` | 2–50 |
| `SalesPerDayEstimated` | 0–20 |
| `Capacity` (por proveedor) | 50–500 |
| `DeliveryTime` (por proveedor) | 1–15 días |
| `Price` (por proveedor) | 55–75 % del precio de venta del producto, ±10 % por proveedor |

El precio sale del precio de venta del propio producto para que las dos cotizaciones queden en el
mismo orden de magnitud que lo que el producto vende, y el ±10 % garantiza que una de las dos sea
más barata que la otra.

## Dos diferencias con `generate_erp_history`

- **No toca el reloj.** Una configuración de abastecimiento es estado actual, no historia: se
  escribe con el reloj real. No hay `core.SetHistoricalUnix` en este script. Los activos sí se
  fechan en el pasado, pero con el campo `AcquisitionDate` del propio payload, que es justamente
  para lo que existe: un activo donado se registra mucho después de haber sido adquirido.
- **No duplica al volver a correrlo**, y por eso no tiene archivo de reanudación. Cada parte se
  protege sola:

| Parte | Por qué no se duplica |
|---|---|
| Proveedores | `POST.client-provider` deduplica por `RegistryNumber` |
| Abastecimiento | `product_supply` está indexada por `ProductID` y se escribe con `db.Merge` |
| Insumos | se cuentan primero; si ya hay 100 o más, no se generan |
| Activos | se cuentan primero; si ya hay 100 o más, no se generan |
| Depreciación | `POST.asset-depreciation-run` es idempotente por diseño (marca de agua por activo) |

Los conteos son un umbral, no un completado: con 100 insumos ya cargados no genera ninguno, y con
40 genera los 100 del archivo igual (quedarían 140). Está pensado para una base de demo vacía.

Correrlo dos veces sí **re-sortea** el abastecimiento: los mismos 2000 productos quedan con otro
stock mínimo y otros proveedores.

## Argumentos

| Flag | Default | |
|---|---|---|
| `--products=N` | 2000 | primeros N productos activos, por ID |
| `--providers-per-product=N` | 2 | proveedores distintos por producto |
| `--min-stock-min` / `--min-stock-max` | 2 / 50 | rango del stock mínimo |
| `--sales-per-day-min` / `--sales-per-day-max` | 0 / 20 | rango de ventas/día estimadas |
| `--supply-materials=N` | 100 | insumos a generar, y umbral para no generarlos. `0` los salta |
| `--assets=N` | 100 | activos a generar, y umbral para no generarlos. `0` los salta |
| `--dry-run` | off | arma todo, imprime los 5 primeros registros y no escribe nada |

Bajar `--supply-materials` por debajo de 40 recorta el catálogo de equipos, y los activos que
apunten a un SKU que no se sembró fallan con "el insumo INS-0xx no existe". Si se recorta uno, hay
que recortar el otro en la misma proporción (`--supply-materials=5 --assets=4` es una prueba de
humo válida).

## Requisitos

- Company 1, usuario 1 y al menos `--products` productos activos en el catálogo.
- Un almacén activo y una caja activa (se toman los de menor ID). La moneda de los activos es la de
  esa caja, porque `POST.asset-payment` rechaza un pago cuya moneda no coincida con la de la caja.
- Los proveedores se siembran solos; no hace falta crearlos antes.
