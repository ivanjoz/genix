# Plan Abastecimiento

El objetivo es convertir `frontend/routes/logistica/productos-stock/+page.svelte` en una pantalla con 2 vistas reales:

1. `Gestion Stock`
2. `Abastecimiento`

La nueva vista `Abastecimiento` debe mostrar una tabla por producto con:

- Producto
- Stock minimo
- Ventas / Dia
- Proveedores configurados

Adicionalmente, cada relacion `producto-proveedor` debe guardar:

- capacidad
- tiempo minimo de entrega

## 1. Estado actual analizado

### Frontend

- `frontend/routes/logistica/productos-stock/+page.svelte` ya define `options={[{ id:1, name: "Gestión Stock" }, { id:2, name: "Abastecimiento" }]}`
- Hoy la pantalla solo implementa la vista de stock manual por almacén.
- `frontend/routes/logistica/productos-stock/productos-stock.svelte.ts` solo consume `GET/POST.productos-stock`.

### Backend

- `backend/operaciones/almacen-movimientos.go`
  - `GetProductosStock` devuelve stock por almacén.
  - `PostAlmacenStock` actualiza stock usando `ApplyMovimientos`.
- `backend/types/productos.go`
  - `Producto` no tiene configuración de abastecimiento.
  - `AlmacenProducto` guarda stock operativo por almacén.
- `backend/comercial/types/sales.go`
  - `SaleSummary` ya guarda ventas agregadas por día y por producto.
  - Esto permite calcular `Ventas / Dia` sin leer todas las órdenes.
- `backend/negocio/types/client_provider.go`
  - Ya existe `ClientProvider` y distingue clientes/proveedores por `Type`.

## 2. Decisión de modelo

Crear una sola tabla `ProductSupply` con una fila por producto.

Esa fila guardará:

- el `ProductID`
- el `MinimunStock`
- el `SalesPerDayEstimated`
- una lista embebida `ProviderSupply`

### Campos mínimos propuestos para `ProductSupplyTable`

- `EmpresaID int32`
- `ProductID int32`
- `MinimunStock int32`
- `SalesPerDayEstimated int32`
- `ProviderSupply []ProductSupplyProviderRow`
- `Status int8`
- `Updated int32`
- `UpdatedBy int32`

### Campos mínimos propuestos para `ProductSupplyProviderRow`

- `ProviderID int32`
- `Capacity int32`
- `DeliveryTime int16`

## 3. Diseño de tabla

Crear en backend:

- `type ProductSupply struct`
- `type ProductSupplyTable struct`

### Esquema propuesto

- `Name: "product_supply"`
- `Partition: EmpresaID`
- `Keys: []db.Coln{ProductID}`

### Vistas mínimas recomendadas

- `Status`
  - para obtener solo registros activos
- `Updated`
  - para delta sync si luego se convierte en servicio cacheado

## 4. API nueva

Crear un handler nuevo en backend para abastecimiento. La ruta debe ser específica y no reusar `productos-stock`.

### GET sugerido

- `GET.product-supply`

Debe devolver un dataset listo para la tabla de abastecimiento.

### POST sugerido

- `POST.product-supply`

Debe crear/actualizar la configuración de abastecimiento completa de un producto.

## 5. Respuesta del GET.product-supply

El GET no debe devolver solo `ProductSupply[]`. Debe devolver un payload enriquecido para evitar trabajo extra en frontend.

### Respuesta sugerida

```go
type ProductSupply struct {
  ProductID int32
  MinimunStock int32
  SalesPerDayEstimated int32
  ProviderSupply []ProductSupplyProviderRow
}

type ProductSupplyProviderRow struct {
  ProviderID int32
  Capacity int32
  DeliveryTime int16
}
```

### Origen de cada dato

- `ProductoNombre`: tabla `Producto`
- `MinimunStock`, `ProviderSupply`: tabla `ProductSupply`
- `SalesPerDayEstimated`: ingresado por usuario y persistido en `ProductSupply`
- nombre de proveedor: resuelto con `ClientProvider` filtrando `Type = Provider`

## 6. Alcance de esta primera versión

En esta primera iteración, el plan queda acotado a:

- separar la vista `Gestion Stock` de la vista `Abastecimiento`
- crear `GET.product-supply`
- crear `POST.product-supply`
- crear `ProductSupply` y `ProductSupplyTable`
- renderizar la tabla principal de productos en `Abastecimiento`
- abrir un `Side Layer` al hacer click en una fila
- guardar desde ese `Side Layer` la configuración del producto
- manejar la tabla inline de proveedores con una fila vacía persistente

### Fuera de alcance por ahora

- cálculo real de ventas por día
- regla final de prioridad entre histórico y estimado
- automatizaciones de abastecimiento
- sugerencias de compra
- forecast

### Para esta fase

El backend y frontend solo deben soportar estos campos:

- `ProductID`
- `MinimunStock`
- `SalesPerDayEstimated`
- `ProviderSupply[]`

## 7. Regla para `Stock minimo`

`MinimunStock` será un valor único por producto.

### Razón

- coincide con tu simplificación del modelo
- evita duplicar el mismo valor por proveedor
- reduce validaciones y tamaño de payload

## 8. Frontend de `Abastecimiento`

Modificar `frontend/routes/logistica/productos-stock/+page.svelte` para que use `Core.pageOptionSelected`, igual que `sedes-almacenes`.

### Vista 1: `Gestion Stock`

Mantener el flujo actual sin cambios funcionales.

### Vista 2: `Abastecimiento`

Mostrar:

- buscador
- tabla con columnas:
  - Producto
  - Stock minimo
  - Ventas / Dia
  - Proveedores configurados
  - acciones

### Render de `Proveedores configurados`

Mostrar resumen compacto:

- cantidad de proveedores
- nombres principales
- o chips simples

### Acción de edición

Abrir un `Side Layer`, no un `Modal`.

Dentro del `Side Layer` debe haber:

- producto
- stock minimo
- ventas por día estimadas editable
- una tabla inline de proveedores

### Regla de la tabla inline

La tabla inline debe comportarse así:

- siempre existe una fila vacía al final
- cuando una fila ya tiene `ProviderID` o algún valor útil, automáticamente aparece otra fila vacía debajo
- la fila vacía no se envía al backend
- filas completamente vacías se filtran antes de guardar

### Columnas mínimas de la tabla inline

- proveedor
- capacidad
- tiempo de entrega
- acción de eliminar solo para filas ya completas

## 9. Servicio frontend nuevo

Crear un archivo nuevo, por ejemplo:

- `frontend/routes/logistica/productos-stock/abastecimiento.svelte.ts`

### Contenido sugerido

- `interface IProductSupplyProviderRow`
- `interface IProductSupplyRow`
- `createEmptyProviderSupplyRow()`
- `normalizeProviderSupplyRows()`
- `getProductSupply()`
- `postProductSupply()`

No usar `GetHandler` todavía.

### Razón

- esta pantalla es más de configuración que de catálogo offline
- el GET necesita agregaciones y joins
- un `GET` simple reduce complejidad inicial

## 10. Backend: pasos concretos

### 10.1. Modelo

Crear `ProductSupply` y `ProductSupplyTable` en `backend/types/productos.go` o en un archivo nuevo de `backend/types/`.

Recomendación:

- archivo nuevo: `backend/types/product_supply.go`

### 10.2. Handler

Crear un archivo nuevo:

- `backend/operaciones/product_supply.go`

Implementar:

- `GetProductSupply`
- `PostProductSupply`

### 10.3. Registro de rutas

Actualizar:

- `backend/operaciones/main.go`

Agregar:

- `"GET.product-supply": GetProductSupply`
- `"POST.product-supply": PostProductSupply`

### 10.4. Controller DB

Actualizar el registro de controllers Scylla si aplica, para incluir `ProductSupply`.

## 11. Validaciones backend obligatorias

En `POST.product-supply` validar:

- `ProductID > 0`
- `MinimunStock >= 0`
- `SalesPerDayEstimated >= 0`
- cada `ProviderSupply.ProviderID > 0`
- cada proveedor exista y sea `ClientProviderTypeProvider`
- `Capacity >= 0`
- `DeliveryTime >= 0`
- no duplicar `ProviderID` dentro del mismo producto
- eliminar filas vacías antes de persistir
- `EmpresaID`, `Updated`, `UpdatedBy` deben salir del backend

## 12. UX mínima recomendada

Para minimizar código en la primera iteración:

- una sola tabla en `Abastecimiento`
- un solo `Side Layer` de edición
- click sobre fila para editar
- tabla inline repetible de proveedores

No incluir todavía:

- forecast
- sugerencia automática de compra
- filtros por sede/almacén en abastecimiento
- cálculo de stock proyectado
- edición masiva de varios productos a la vez

## 13. Riesgos / puntos a decidir

### Punto 1

`Ventas / Dia` puede calcularse:

- global por empresa
- o por almacén

### Recomendación

Primera versión:

- global por empresa

Razón:

- `SaleSummary` ya está a nivel empresa+fecha
- evita crear otra tabla agregada

### Punto 2

`SalesPerDayCalculated` puede:

- persistirse en `ProductSupply`
- o calcularse en cada GET y enviarse solo como dato derivado

### Recomendación

Primera versión:

- calcularlo en cada `GET.product-supply`
- no confiar en el valor enviado por frontend
- persistir solo `SalesPerDayEstimated`

Razón:

- evita inconsistencia entre configuración y dato analítico
- `SaleSummary` ya existe
- el frontend no debería ser dueño de ese cálculo

## 14. Orden de implementación recomendado

1. Crear `ProductSupplyTable` y registrarlo en backend.
2. Implementar `GET.product-supply` con cálculo de `SalesPerDay`.
3. Implementar `POST.product-supply` limpiando filas vacías.
4. Crear `abastecimiento.svelte.ts`.
5. Separar `+page.svelte` por `Core.pageOptionSelected`.
6. Renderizar tabla de `Abastecimiento`.
7. Agregar `Side Layer` con tabla inline de proveedores.
8. Probar guardado y recarga.

## 15. Suposiciones usadas en este plan

- `Ventas / Dia` será por empresa, no por almacén.
- `MinimunStock` será por producto.
- `SalesPerDayEstimated` será editable por usuario.
- `SalesPerDayCalculated` será derivado desde `SaleSummary`.
- `SalesPerDay` será resuelto por backend para consumo directo de UI.
- `ProviderSupply` será una lista embebida en `ProductSupply`.
- `ProviderID` apuntará a `ClientProvider` con tipo proveedor.
- La edición será con `Side Layer`.
- No se requiere compatibilidad hacia atrás.

## 16. Preguntas abiertas para confirmar antes de implementar

1. `Ventas / Dia` debe ser global por empresa o filtrado por almacén?
2. El fallback final de `SalesPerDay` debe preferir siempre el histórico cuando exista, correcto?
3. En la tabla inline del `Side Layer`, quieres autoguardado o guardado manual con botón?
