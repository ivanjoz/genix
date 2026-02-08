# Plan for Sales Orders Page by Status

The goal is to create a page that displays sales orders filtered by status: "Finalizadas", "Pendiente de Pago", and "Pendiente de Entrega".

## 1. Backend: API Implementation

### 1.1. Create `backend/comercial/sale_order_get.go`
- Implement `GetSaleOrders` handler.
- Logic:
    - Support delta sync using the `upd` query parameter.
    - Multi-tenancy: Filter by `req.Usuario.EmpresaID`.
    - Status mapping:
        - `0`: Anulado
        - `1`: Generado (Pend. Pago & Pend. Entrega)
        - `2`: Pagado (Pend. Entrega)
        - `3`: Entregado (Pend. Pago)
        - `4`: Finalizado (Pagado + Entregado)
- The API will return all sales orders for the company (respecting `upd` for delta sync) to allow the frontend to manage the state efficiently using its local cache mechanism.

### 1.2. Register Handler
- Add `"GET.sale_orders": comercial.GetSaleOrders` to `backend/handlers/main.go`.

## 2. Frontend: Service and UI Implementation

### 2.1. Create `frontend/services/services/sale_orders.svelte.ts`
- Implement a cached service for `SaleOrder` using the `CachedService` pattern (if available) or a standard fetch with delta sync support.
- Follow the patterns in `frontend/docs/SERVICES_GUIDE.md`.

### 2.2. Implement UI in `frontend/routes/comercial/sale_orders_manage/`
- Use `+page.svelte` to display the orders.
- Create 3 tabs or a filter to switch between:
    - **Finalizadas**: `Status == 4`
    - **Pend. Pago**: `Status == 1 || Status == 3`
    - **Pend. Entrega**: `Status == 1 || Status == 2`
- Use `VTable` from the UI components library to display the list.
- Implement columns like: ID, Date, Total, Status, etc.

## 3. Verification
- Test API with `curl` or similar.
- Verify frontend rendering and filtering logic.
- Ensure delta sync works correctly.
