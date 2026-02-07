# Plan de Implementación: Point of Sale - PostSaleOrder

Este plan detalla los cambios realizados para implementar la funcionalidad de envío de órdenes de venta desde el frontend hacia el backend, asegurando la consistencia de datos y tipos.

## 1. Backend: Completar Estructuras de Datos
**Archivo:** `backend/comercial/types/sales.go`

- Se actualizó `SaleOrderTable` para incluir todos los campos presentes en la estructura `SaleOrder`. Esto es necesario para que el ORM/DB handler pueda mapear correctamente todos los campos durante la inserción.
- Campos añadidos: `DetailProductsIDs`, `DetailPrices`, `DetailQuantities`, `TotalAmount`, `TaxAmount`, `DebtAmount`, `DeliveryStatus`, `CajaID_`, `ProcessesIncluded_`, `Created`, `UpdatedBy`, `Status`.

## 2. Frontend: Refactorización de Tipos y Estado
**Archivo:** `frontend/routes/comercial/point_of_sale/saleOrder.svelte.ts`

- Se renombró `IVenta` a `ISaleOrder` para mayor claridad.
- Se refactorizaron los campos de `ISaleOrder` de español a inglés para que coincidan exactamente con las etiquetas JSON esperadas por el backend Go.
- Campos actualizados:
    - `subtotal`, `igv`, `total` -> `TotalAmount`, `TaxAmount`.
    - `procesado` -> `ProcessesIncluded_`.
- Se implementó el método `postSaleOrder` en la clase `SaleOrderState`:
    - Valida que el carrito no esté vacío.
    - Valida que se haya seleccionado un almacén.
    - Prepara los arreglos de detalle (`DetailProductsIDs`, `DetailPrices`, `DetailQuantities`) a partir de los productos en el carrito.
    - Realiza una petición POST a `comercial/point_of_sale`.
    - Limpia el estado tras un envío exitoso.

## 3. Frontend: Actualización de la Interfaz de Usuario
**Archivo:** `frontend/routes/comercial/point_of_sale/+page.svelte`

- Se actualizaron las referencias a los campos de `ventasState.form` (ej. `TotalAmount` en lugar de `total`).
- Se vinculó el evento `onclick` del botón "Generar" al nuevo método `ventasState.postSaleOrder()`.
- Se aseguró que el `AlmacenID` se actualice en el estado global de la venta tanto en la selección manual como en la auto-selección inicial.
- Se actualizó `CheckboxOptions` para usar `ProcessesIncluded_`.

## 4. Correcciones Menores
**Archivo:** `frontend/routes/comercial/point_of_sale/ProductoVentaCard.svelte`

- Se corrigió el import de tipos para que apunte a `./saleOrder.svelte` en lugar de `./ventas.svelte`, ya que el archivo fue renombrado o se encontraba mal referenciado.

## Verificación Sugerida
1. Seleccionar un almacén.
2. Añadir productos al carrito.
3. Verificar que los totales se calculen correctamente.
4. Presionar "Generar" y verificar en la red (Network tab) que el JSON enviado coincida con la estructura de `SaleOrder` de Go.
5. Confirmar que el backend procese la venta (reducción de stock y registro en caja según los procesos incluidos).
