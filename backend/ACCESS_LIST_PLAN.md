# Plan — Completar `access_list.yml` y arreglar el guard de rutas

## Diagnóstico (verificado contra la BD y el código)

Usuario de prueba `demo1` (company 3, perfil "Perfil de prueba"):
`profiles.accesos = [74,84,94,104,114,124,134,141,151]` → accesos **7,8,9,10,11,12,13 nivel 4**
y **14,15 nivel 1**. Sin Configuración, sin Finanzas, sin Órdenes de Compra.

### 1. El frontend deja pasar todo lo que no está en el catálogo

`packages/genix-ui/security/create-security.ts:152`

```ts
const matchedAccessEntries = resolveRouteAccessEntries?.(route) || []
if (matchedAccessEntries.length === 0) { return true }   // ruta no catalogada = libre
```

Y el catálogo tiene rutas obsoletas, así que la mayoría de las páginas no están catalogadas:

| Ruta real (`core/modules.ts`) | `access_list.yml` decía | Efecto |
|---|---|---|
| `/system/companies` | `system/empresas` | sin guard |
| `/finance/expenses` | `finanzas/gastos` | sin guard |
| `/finance/gestion-cuentas` | `finanzas/gestion-cuentas` | sin guard |
| `/finance/flujo-de-caja` | `finanzas/flujo-de-caja` | sin guard |
| `/business/suppliers` | — | sin guard |
| `/sales/sales-report`, `/sales/sale_planning` | — | sin guard |
| `/logistics/purchase-management`, `/logistics/supplies-materials` | — | sin guard |
| `/webpage-builder/pages`, `/webpage-builder/gallery`, `/webpage-builder/[pageID]` | — | sin guard |
| `/system/cron-actions`, `/system/testing` | — | sólo los tapa `onlySaaS` |

### 2. El backend bloquea de más, por el mismo desfase

`main-handlers.go:186` niega todo POST/PUT sin acceso mapeado para usuarios distintos del ID 1.
El YAML nombraba rutas inexistentes — `POST.almacenes` (real: `POST.sites` / `POST.warehouses`) y
`POST.productos` (real: `POST.products`) — así que `demo1`, que **sí** tiene "Almacenes" y
"Productos" en nivel 4, no podía guardar nada. 33 de 39 rutas POST/PUT estaban sin mapear.

## Cambios

1. **`backend/access_list.yml`** — reescribir manteniendo los IDs existentes (están persistidos en
   `profiles.accesos`), corregir los `frontend_routes` desfasados, mapear las 38 rutas POST/PUT no
   públicas y agregar 9 accesos nuevos (IDs 26–34) para las páginas que no tenían ninguno.
   Nuevo `access_group` 8 "System" para separar el módulo SYSTEM de "Configuración".

2. **`frontend/routes/security/access-profiles/access-list-catalog.ts`** — `getAccessEntriesForRoute`
   pasa de igualdad exacta a **prefijo por segmentos, el más largo gana**. Así `/webpage-builder/123`
   (el editor de páginas) hereda el acceso de `webpage-builder`, sin necesitar una entrada por ID.

3. **`frontend/libs/ui-runtime.svelte.ts`** — `/webpage-builder/template-preview` se declara ruta
   pública: es el preview sin chrome que usa el agente headless, ya exento del login en `+layout.svelte`.

4. **`backend/main-handlers.go`** — `selfServiceRoutes`: rutas autenticadas que no exigen acceso
   porque el usuario opera sobre sí mismo. Hoy sólo `POST.user-self`.

5. **GET cerrables uno por uno.** `main-handlers.go` pasa de "todo GET es libre" a "un GET **sin**
   acceso mapeado es libre; uno **mapeado** exige el acceso". Eso permite restringir lecturas de a
   una sin bloquear de golpe las que aún no están en el catálogo.

   Primer caso: `GET.company-parametros` traía un `req.User.ID != 1` a mano en
   `config/empresas.go`, lo que rompía el POS (necesita RUC y razón social para el comprobante)
   para cualquiera que no fuera el administrador. Se elimina ese check —y el gemelo en
   `PostEmpresaParametros`— y la ruta queda mapeada a los accesos 1 (Mi Empresa) y 10 (Punto de
   Venta). Eran los dos únicos chequeos de administrador hardcodeados del backend.

## Fuera de alcance (se reporta, no se toca)

- El resto de los GET sigue libre para cualquier sesión autenticada. Ahora se pueden ir cerrando
  agregándolos a `backend_apis`, sin más cambios de código.
- `/finance/gestion-cuentas` y `/finance/flujo-de-caja` están en el menú pero no existen como
  páginas. Se les deja su acceso (20, 21) para cuando se construyan.
