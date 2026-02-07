# Plan de Implementación: Endpoint de Ventas

Este plan detalla los pasos para crear el endpoint `POST /comercial-sales` que registra una venta y procesa opcionalmente el pago (Caja) y la entrega (Almacén).

## 1. Actualización de Tipos
- **Archivo**: `backend/types/ventas.go`
- **Acción**: Cambiar `VentaID` de `int32` a `int64` en `CajaMovimiento` y `CajaMovimientoTable`. Esto es necesario porque `Sale.ID` es un `int64` debido al empaquetado de clave (`KeyIntPacking`).

## 2. Implementación del Handler `PostSales`
- **Archivo**: `backend/comercial/sales.go`
- **Lógica**:
    1. Deserializar el body en la estructura `s.Sale`.
    2. Inicializar metadatos: `EmpresaID`, `Fecha` (Unix día), `Created`, `Updated`, `Status = 1`.
    3. Insertar el registro de `Sale` para obtener el `ID` autoincremental.
    4. **Si `ProcessesIncluded_` contiene 2 (Pago)**:
        - Calcular el monto pagado: `InvoiceAmount - DebtAmount`.
        - Ejecutar `operaciones.ApplyCajaMovimientos` con `Tipo: 8` (Cobro Venta) y el `Sale.ID`.
        - Esta función se encarga de obtener la caja, actualizar el saldo y registrar el movimiento.
    5. **Si `ProcessesIncluded_` contiene 3 (Entrega)**:
        - Generar una lista de `MovimientoInterno` basada en `DetailProductsIDs` y `DetailQuantities`.
        - Las cantidades se enviarán en negativo para representar una salida de almacén.
        - Ejecutar `operaciones.ApplyMovimientos`.
    6. Retornar la venta creada.

## 3. Abstracción de Movimientos de Caja
- Se ha creado la función `ApplyCajaMovimientos` en `backend/operaciones/caja-movimientos.go` para estandarizar el registro de movimientos de dinero, similar a cómo funciona el stock de productos.
- Se utiliza la estructura `CajaMovimientoInterno` para definir los parámetros del movimiento.

## 3. Registro del Endpoint
- **Archivo**: `backend/handlers/main.go`
- **Acción**: Agregar `"POST.comercial-sales": comercial.PostSales`.

## 4. Verificación
- Compilar el backend para asegurar que no hay errores de tipos.
- Validar la integración con `ApplyMovimientos`.
