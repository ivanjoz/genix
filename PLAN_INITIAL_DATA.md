# PLAN — Datos Iniciales (bootstrap de Sede / Almacén / Caja)

## Objetivo

Una empresa recién creada no puede operar: `Warehouse` y `CashBank` exigen `SiteID`, y sin
almacén ni caja las ventas, los movimientos de stock y los cobros no tienen destino. El login
debe informar si falta ese mínimo operativo y llevar al usuario a una página **"Datos Iniciales"**
que crea los tres registros con nombres por defecto.

Valores por defecto: **Sede = `Principal`**, **Almacén = `Central`**, **Caja = `Caja Principal`**.
La ciudad (ubigeo) y la dirección de la sede **las ingresa el usuario** (igual que en el formulario
de Sede de `/business/branches-warehouses`), porque `Site.CityID` es obligatorio y `Company` sólo
guarda `City` como texto libre — no hay ubigeo del cual derivar un default.

Decisiones tomadas con el usuario:
- **Soft redirect**: sólo el handler de login navega a `/initial-data`. No hay guard en el layout.
- **Página sin chrome**: se agrega a `routesWithoutLayout` (como `/login`), tarjeta centrada.

---

## 1. Backend — flag en la respuesta del login

**`backend/security/login.go`**

`MakeUsuarioResponse` es compartido por `POST p-user-login` y `GET reload-login`, así que el flag
viaja en ambos y se auto-corrige en cada refresh de token (~40 min).

- Nueva función `hasPendingInitialData(companyID int32) (bool, error)`:
  consulta `warehouses` y luego `cash_banks` con
  `Select().CompanyID.Equals(companyID).Delta(0, 1)` — el mismo patrón "sólo filas activas" que
  usan `GetLocationsWarehouses` y `GetCashBanks`. Devuelve `true` si alguna de las dos está vacía
  (si `warehouses` está vacía corta ahí y no consulta las cajas).
  Va secuencial y con `recover`: el ORM hace panic cuando no puede conectar, y un panic dentro de
  una goroutine de `errgroup` no lo atraparía nadie. Un fallo degrada a "sin pendientes" en vez de
  convertir un login válido en un 500.
  Los volúmenes son mínimos (<50 filas por empresa), así que no se usa `Limit`.
- Importa sólo `app/business/types` y `app/finance/types` (paquetes de tipos que sólo dependen de
  `app/db`), de modo que no se crea ningún ciclo de imports con `business` / `finance`.
- Se agrega al mapa de respuesta: `"InitialDataPending": pending`.
- Si la query falla, se loguea y se devuelve `false` (no se rompe el login por esto).

## 2. Backend — endpoint que inserta los tres registros

**`backend/business/initial-data.go`** (nuevo) + registro `"POST.initial-data": PostInitialData`
en `backend/business/main.go`.

Un solo endpoint en lugar de tres POST desde el frontend: el `SiteID` autoincremental hace falta
para el almacén y la caja, y así el orden y la validación quedan del lado del servidor.

Body:

```go
type initialDataBody struct {
    SiteID        int32  // 0 = crear la sede; >0 = usar una existente
    SiteName      string
    SiteAddress   string
    CityID        int32
    WarehouseName string
    CashBankName  string
}
```

Validaciones (nunca confiar en el cliente):
- `WarehouseName` y `CashBankName` con al menos 4 caracteres.
- Si `SiteID == 0`: `SiteName` ≥ 4, `SiteAddress` ≥ 4 y `CityID > 0`.
- Si `SiteID > 0`: la sede debe existir y estar activa en la empresa del usuario.

Comportamiento (idempotente — el endpoint es alcanzable por URL):
1. Lee las sedes, almacenes y cajas activas de la empresa.
2. Si ya existen almacén **y** caja, responde sin insertar nada.
3. Crea la sede si hace falta (`db.Insert` deja el ID autoincremental en el registro).
4. Crea el almacén sólo si no había ninguno; `Layout` vacío.
5. Crea la caja sólo si no había ninguna; `Type = 1` (Caja), `CurrencyType = 1` (PEN),
   `CurrentAmount = 0`.
6. Sella `Created` / `CreatedBy` / `Updated` / `UpdatedBy` con `core.SUnixTime()` y `req.User.ID`.

Respuesta: `{ SiteID, WarehouseID, CashBankID }`.

## 3. Backend — arreglar el flujo de `PostCashBanks`

**`backend/finance/cash_banks.go:31`** — hoy asigna `body.Created = nowTime` **antes** de evaluar
`if body.Created == nowTime`, así que la rama de update es código muerto: todo POST es `Insert`.
Como en Scylla el INSERT es un upsert, editar una caja sobrescribe `Created`, `CurrentAmount`,
`ReconciliationAmount` y `ReconciliationDate` con ceros.

Arreglo:
- Discriminar por ID, que es el contrato real del frontend (`cajaForm = { ID: -1, ... }` para nueva):
  `isNewCashBank := body.ID <= 0`.
- Insert: setea `Created` / `CreatedBy` / `Updated` / `UpdatedBy`.
- Update: setea sólo `Updated` / `UpdatedBy` y extiende el `db.UpdateExclude` para excluir también
  `Created` y `CreatedBy`, además de `ReconciliationDate`, `ReconciliationAmount` y `CurrentAmount`:
  el cliente no envía ninguna de esas columnas, así que escribirlas pondría ceros.
- Quitar el `core.Env.LOGS_DEBUG = true` que quedó del debug.
- El guard de campos obligatorios pasa a rechazar `body.ID == 0` (ni nueva ni existente).

## 4. Backend/Frontend — arreglar el nombre del campo dirección de la Sede

`Site.Address` se serializa como `"Address"`, pero el frontend usa `Direccion` de punta a punta
(`ISite.Direccion`, el `Input`, la columna de la tabla y el filtro), así que **la dirección que el
usuario escribe nunca se guarda**. Lo mismo con `Ciudad` vs. el `City` que devuelve el backend.
Sin esto, la dirección de la página de Datos Iniciales tampoco persistiría.

Proyecto pre-alpha, sin compatibilidad hacia atrás: se renombra en el frontend
`Direccion → Address` y `Ciudad → City` en
`frontend/routes/business/branches-warehouses/branches-warehouses.svelte.ts` y sus 5 usos en
`+page.svelte`.

## 5. Frontend — servicio y página

**`frontend/routes/initial-data/initial-data.svelte.ts`** (nuevo)

```ts
export interface IInitialData { SiteID, SiteName, SiteAddress, CityID, WarehouseName, CashBankName }

export const postInitialData = (data: IInitialData) =>
  POST({ data, route: "initial-data", refreshRoutes: ["locations-warehouses", "cash-banks"] })
```

`refreshRoutes` invalida los delta-caches de sedes/almacenes y cajas para que las páginas que ya
los tengan en IndexedDB vean los registros nuevos.

**`frontend/routes/initial-data/+page.svelte`** (nuevo)

- Tarjeta centrada con el logo de Genix y título `T text="Initial Data|Datos Iniciales"`, reusando
  el estilo de `/login` (no usa el shell `Page`).
- Componentes del proyecto (contrato con el agente): `Input`, `SearchSelect`, `Button`.
- Instancia `new WarehousesService()` y `new CountryCitiesService(true)`.
  - Si `Sedes.length === 0`: `Input` Nombre (`Principal`), `Input` Dirección y `SearchSelect` de
    distrito (`keyId="ID" keyName="_nombre" options={distritos}`), igual que el modal de Sede.
  - Si ya hay sedes: `SearchSelect` de Sede preseleccionada con la primera, y se ocultan los campos
    de creación (evita duplicar una sede cuando sólo falta el almacén o la caja).
- `Input` Almacén (`Central`) e `Input` Caja (`Caja Principal`), ambos editables.
- Validación en cliente espejo de la del backend, `Loading.standard` / `Notify`, y al terminar
  `Notify.success` + `Env.navigate("/")`.
- Todos los textos con `T` / `tr()` en formato `EN|ES`.

**`frontend/core/types/common.ts`** — `ILoginResult` gana `InitialDataPending?: boolean`.

**`frontend/services/login.ts:44`** — tras `parseLogin`, navega a `/initial-data` cuando
`loginInfo.InitialDataPending` es `true`; en caso contrario a `/` como hoy.

**`frontend/routes/+layout.svelte:112`** — agregar `"/initial-data"` a `routesWithoutLayout`.
La ruta sigue exigiendo sesión (`redirectsToLogin` no la excluye) y no necesita entrada en el
access-list: `canAccessRoute` deja pasar las rutas ausentes del catálogo.

**No** se agrega al menú lateral (`frontend/core/modules.ts`): es un paso de bootstrap, no una
sección del sistema.

---

## Archivos tocados

| Archivo | Cambio |
|---|---|
| `backend/security/login.go` | flag `InitialDataPending` + `hasPendingInitialData` |
| `backend/business/initial-data.go` | **nuevo** handler `PostInitialData` |
| `backend/business/main.go` | ruta `POST.initial-data` |
| `backend/finance/cash_banks.go` | arreglo insert/update de `PostCashBanks` |
| `frontend/routes/initial-data/+page.svelte` | **nuevo** |
| `frontend/routes/initial-data/initial-data.svelte.ts` | **nuevo** |
| `frontend/core/types/common.ts` | `ILoginResult.InitialDataPending` |
| `frontend/services/login.ts` | redirect condicional |
| `frontend/routes/+layout.svelte` | `routesWithoutLayout` |
| `frontend/routes/business/branches-warehouses/*` | `Direccion → Address`, `Ciudad → City` |

## Verificación

1. `go build ./...` en `backend/` y `scripts/` de validación estática de tablas (no se modifican
   estructuras de tablas, así que no hay migración).
2. Empresa sin almacén/caja → el login navega a `/initial-data`; aceptar los defaults inserta las
   tres filas; recargar `/login` y entrar de nuevo ya lleva a `/`.
3. Empresa con sede pero sin almacén → la página muestra el selector de Sede y no crea una sede nueva.
4. Editar una caja existente en `/finance/cash-banks` conserva `Created` y `CurrentAmount`.
