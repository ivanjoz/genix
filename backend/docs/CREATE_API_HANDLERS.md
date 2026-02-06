# API Handler Development Guide

This guide provides a comprehensive overview of creating API handlers in the backend. It covers routing, synchronization logic, multi-tenancy, and advanced patterns for both data retrieval and modification. For specific ORM query syntax, refer to [ORM_DATABASE_QUERY.md](ORM_DATABASE_QUERY.md).

---

## 1. Handler Interface & Routing

### Signature
All backend handlers must implement a standard signature using `*core.HandlerArgs`. This unified interface ensures that handlers have access to all necessary request context while returning a consistent response type.

```go
func GetEntity(req *core.HandlerArgs) core.HandlerResponse { ... }
func PostEntity(req *core.HandlerArgs) core.HandlerResponse { ... }
```

### Request Context (`core.HandlerArgs`)
The `req` object is the primary source of information for the handler:
- **`req.Usuario`**: The authenticated user object, containing `EmpresaID`, `ID`, `RolesIDs`, and `AccesosIDs`. This is the single source of truth for multi-tenancy.
- **`req.Body`**: A pointer to the raw request body string. The system handles decompression (e.g., Zstd) before it reaches the handler.
- **`req.Route`**: The specific request path (e.g., `/productos`).
- **`req.GetQueryInt(name)`**: Extracts a query parameter as an `int32`.
- **`req.GetQueryInt64(name)`**: Extracts a query parameter as an `int64`.
- **`req.GetQueryString(name)`**: Extracts a query parameter as a `string`.
- **`req.GetQueryIntSlice(name)`**: Extracts comma-separated values as a slice of `int32`.
- **`req.GetHeader(name)`**: Accesses raw HTTP headers for specialized logic (e.g., `Authorization`, `X-Cipher-Key`).

### Response Patterns
Handlers must return a `core.HandlerResponse`. Available methods include:
- **`req.MakeResponse(data)`**: The standard way to return data. It automatically serializes the payload to JSON or CBOR based on the client's `Accept` header.
- **`req.MakeResponsePlain(bytes)`**: Returns raw bytes without wrapping. Ideal for file exports, raw text, or custom binary formats.
- **`req.MakeErr(msg, [err])`**: Returns a structured error. The `msg` parameter should always be in Spanish for consistent user-facing feedback.

### Routing Registry
Handlers are registered in `backend/handlers/main.go`. The naming convention for keys is `METHOD.route-name`.

```go
var ModuleHandlers = core.AppRouterType{
    "GET.productos":   GetProductos,
    "POST.productos":  PostProductos,
    "GET.sedes":        GetSedes,
    "POST.user-login":  PostLogin, // Routes with 'p-' prefix are typically public
}
```

---

## 2. The Request Lifecycle

Understanding how a request flows through the system is vital for debugging and implementing complex logic:

1.  **Transport Layer**: The request arrives via HTTP. If the `Accept` header is `application/cbor`, the system prepares for binary communication.
2.  **Authentication Middleware**: The system first validates the session token. If valid, it populates `req.Usuario` with the user's details and permissions.
3.  **Decompression**: If the payload is compressed (e.g., using Zstd), the system decompresses it before passing it to the handler.
4.  **Handler Execution**: The specific handler function is called with the populated `HandlerArgs`.
5.  **Business Logic & DB**: The handler performs validation, interacts with the database (ScyllaDB), and processes assets (S3).
6.  **Response Construction**: The handler returns a `HandlerResponse` via `MakeResponse` or `MakeErr`.
7.  **Serialization**: The system serializes the response based on headers (JSON/CBOR).
8.  **Compression (Optional)**: If the client supports it, the response may be compressed before being sent back.

---

## 3. Synchronization & Cache Protocol

The backend is architected to support high-performance mobile and web applications that maintain a local database (cache). Synchronization is managed through an `updated` (or `upd`) timestamp.

### The Protocol Logic
The frontend tracks its last successful sync time and sends it as a query parameter. The backend then filters records based on this timestamp:

1.  **Initial Load (`updated == 0`)**:
    - The client has no local data.
    - Return only "active" records (`Status >= 1`).
    - This reduces the initial sync payload by excluding historical or deleted data.

2.  **Incremental Sync (`updated > 0`)**:
    - The client is asking for changes since the last sync.
    - Return ALL records where `Updated > updated`.
    - **Crucial**: You must include records with `Status = 0` (deleted). This allows the frontend to remove these records from its local cache.

### Sunix Timestamps
The system uses a custom compact timestamp format called `Sunix`.
- Use `core.SUnixTime()` to get the current timestamp.
- Use `core.UnixToSunix(int64)` for conversions from standard Unix seconds.

---

## 4. Multi-Tenancy Enforcement

Multi-tenancy is strictly enforced at the handler level to prevent cross-company data access.

### Filtering in Queries
Every `SELECT` operation must include the `EmpresaID` filter. This ensures the database only returns records belonging to the authenticated user's company.

```go
query.Select().EmpresaID.Equals(req.Usuario.EmpresaID)
```

### Assignments in Modifications
Before any `INSERT` or `UPDATE`, the `EmpresaID` must be assigned from the authenticated user context. Never trust an `EmpresaID` provided in the request body.

```go
body.EmpresaID = req.Usuario.EmpresaID
```

---

## 5. Validation & Business Logic Patterns

Always perform thorough validation before interacting with the database.

### Mandatory Field Checks
```go
if len(body.Nombre) < 4 {
    return req.MakeErr("El nombre debe tener al menos 4 caracteres.")
}
if body.SedeID == 0 {
    return req.MakeErr("Debe especificar una Sede.")
}
```

### Authorization Checks
Use `req.Usuario.AccesosIDs` to verify specific permissions.
```go
const AccesoAlmacenEscritura = 105
if !core.SliceContains(req.Usuario.AccesosIDs, AccesoAlmacenEscritura) {
    return req.MakeErr("No tiene permiso para realizar esta acción.")
}
```

---

## 6. Advanced GET Patterns

### Parallel Queries with `errgroup`
To improve performance when a handler needs to return multiple independent datasets, use Go's `errgroup` to execute queries concurrently.

```go
func GetSedesAlmacenes(req *core.HandlerArgs) core.HandlerResponse {
    updated := req.GetQueryInt64("upd")
    almacenes := []s.Almacen{}
    sedes := []s.Sede{}
    eg := errgroup.Group{}

    eg.Go(func() error {
        q := db.Query(&almacenes)
        q.Select().EmpresaID.Equals(req.Usuario.EmpresaID)
        if updated > 0 { q.Updated.GreaterThan(updated) } else { q.Status.Equals(1) }
        return q.Exec()
    })

    eg.Go(func() error {
        q := db.Query(&sedes)
        q.Select().EmpresaID.Equals(req.Usuario.EmpresaID)
        if updated > 0 { q.Updated.GreaterThan(updated) } else { q.Status.Equals(1) }
        return q.Exec()
    })

    if err := eg.Wait(); err != nil {
        return req.MakeErr("Error al obtener sedes y almacenes:", err)
    }
    
    return core.MakeResponse(req, map[string]any{"Sedes": sedes, "Almacenes": almacenes})
}
```

### Time-UUID Range Queries
For log-like data (e.g., `CajaMovimientos`), we use `SUnixTimeUUID` for primary keys. These are sorted chronologically and can be queried efficiently using ranges.

```go
func GetCajaMovimientos(req *core.HandlerArgs) core.HandlerResponse {
    cajaID := req.GetQueryInt("caja-id")
    if cajaID == 0 { return req.MakeErr("No se envió la Caja-ID") }

    fechaHoraInicio := req.GetQueryInt64("fecha-hora-inicio")
    fechaHoraFin := req.GetQueryInt64("fecha-hora-fin")

    movimientos := []s.CajaMovimiento{}
    query := db.Query(&movimientos)
    query.Select().EmpresaID.Equals(req.Usuario.EmpresaID)
    
    // Range query using SUnixTimeUUID pattern
    query.ID.Between(
        core.SUnixTimeUUIDConcatID(cajaID, fechaHoraInicio),
        core.SUnixTimeUUIDConcatID(cajaID, fechaHoraFin + 1),
    )
    query.OrderDesc()

    if err := query.Exec(); err != nil {
        return req.MakeErr("Error al obtener los movimientos:", err)
    }

    return req.MakeResponse(movimientos)
}
```

---

## 7. POST Handler Patterns (Upsert)

Our POST handlers typically follow an "Upsert" pattern, handling both creation and modification in a single endpoint.

### Handling Transactions and State
When updating multiple related tables (e.g., recording a cash movement and updating the register balance), ensure all steps succeed or return a clear error.

```go
func PostMovimientoCaja(req *core.HandlerArgs) core.HandlerResponse {
    nowTime := core.SUnixTime()
    record := s.CajaMovimiento{}
    json.Unmarshal([]byte(*req.Body), &record)

    // 1. Fetch current state to verify balance
    caja, err := shared.GetCaja(req.Usuario.EmpresaID, record.CajaID)
    if err != nil { return req.MakeErr("Caja no encontrada.", err) }

    // 2. Validate balance consistency
    if record.SaldoFinal - record.Monto != caja.SaldoCurrent {
        return req.MakeResponse(map[string]any{"NeedUpdateSaldo": caja.SaldoCurrent})
    }

    // 3. Prepare records
    record.EmpresaID = req.Usuario.EmpresaID
    record.Created = nowTime
    record.ID = core.SUnixTimeUUIDConcatID(record.CajaID)
    
    caja.SaldoCurrent = record.SaldoFinal
    caja.Updated = nowTime

    // 4. Sequential persistence (Order matters!)
    if err := db.Insert(&[]s.CajaMovimiento{record}); err != nil {
        return req.MakeErr("Error al registrar movimiento.", err)
    }

    q1 := db.Table[s.Caja]()
    if err := db.Update(&[]s.Caja{caja}, q1.SaldoCurrent, q1.Updated); err != nil {
        return req.MakeErr("Error al actualizar saldo de caja.", err)
    }

    return req.MakeResponse(record)
}
```

### Batch POST and ID Mapping
For bulk operations where the client sends temporary negative IDs, the handler must return a mapping of these to the permanent IDs generated by the server.

```go
func PostBatch(req *core.HandlerArgs) core.HandlerResponse {
    records := []s.Entity{}
    json.Unmarshal([]byte(*req.Body), &records)
    
    newIDs := []s.NewIDToID{}
    createCount := 0
    for _, r := range records {
        if r.ID <= 0 { createCount++ }
    }
    
    // 1. Obtain block of IDs
    counter, _ := records[0].GetCounter(createCount, req.Usuario.EmpresaID)
    
    // 2. Assign and map
    for i := range records {
        r := &records[i]
        if r.ID <= 0 {
            tempID := r.ID
            r.ID = int32(counter)
            newIDs = append(newIDs, s.NewIDToID{NewID: r.ID, TempID: tempID})
            counter++
        }
    }
    
    db.Insert(&records)
    return req.MakeResponse(newIDs)
}
```

---

## 8. Full Implementation Examples

### Example: GetProductos (Standard GET with Sync)
```go
func GetProductos(req *core.HandlerArgs) core.HandlerResponse {
	// 1. Get updated param (handle multiple names for compatibility)
	updated := core.Coalesce(req.GetQueryInt64("upd"), req.GetQueryInt64("updated"))

	// 2. Initialize slice
	productos := []s.Producto{}
	
	// 3. Build and execute query
	query := db.Query(&productos)
	
	// Exclude large or calculated fields to keep payload small
	query.Exclude(query.Stock, query.StockStatus)
	query.Select().EmpresaID.Equals(req.Usuario.EmpresaID)
	
	if updated > 0 {
		query.Updated.GreaterThan(updated)
	} else {
		query.Status.GreaterEqual(1)
	}
	
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los productos:", err)
	}
	
	// 4. Log results for monitoring
	core.Log("Productos obtenidos::", len(productos))
	
	return core.MakeResponse(req, &productos)
}
```

### Example: PostCajas (Single Record Upsert)
```go
func PostCajas(req *core.HandlerArgs) core.HandlerResponse {
	// 1. Deserialize
	body := s.Caja{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al procesar los datos de la caja:", err)
	}

	// 2. Validation
	if len(body.Nombre) == 0 || body.SedeID == 0 {
		return req.MakeErr("Faltan parámetros: Nombre o SedeID")
	}

	// 3. Prepare common fields
	nowTime := core.SUnixTime()
	body.Updated = nowTime
	body.EmpresaID = req.Usuario.EmpresaID

	// 4. Handle Create vs Update
	if body.ID <= 0 {
		// New record - obtain ID and set creation metadata
		counter, err := body.GetCounter(1, req.Usuario.EmpresaID)
		if err != nil { return req.MakeErr("Error al obtener ID.", err) }
		body.ID = int32(counter)
		body.CreatedBy = req.Usuario.ID
		body.Created = nowTime
		body.Status = 1
		
		err = db.Insert(&[]s.Caja{body})
	} else {
		// Existing record - update tracking
		body.UpdatedBy = req.Usuario.ID
		
		// Protect specific fields from being overwritten during standard update
		q1 := db.Table[s.Caja]()
		err = db.UpdateExclude(&[]s.Caja{body}, q1.CuadreFecha, q1.CuadreSaldo, q1.SaldoCurrent)
	}

	if err != nil {
		return req.MakeErr("Error al guardar la caja:", err)
	}

	return req.MakeResponse(body)
}
```

---

## 9. Asset Management (Images & S3)

Images are processed and stored in S3, while the database stores only the reference.

### Image Processing Pattern
1.  Receive the base64 content.
2.  Generate a unique name using `core.ToBase36(time.Now().UnixMilli())`.
3.  Process via `aws.SaveConvertImage`, which handles resizing for multiple resolutions (e.g., x6, x4, x2) and format optimization (e.g., AVIF).
4.  Store the folder and name in the record.

```go
func PostProductoImage(req *core.HandlerArgs) core.HandlerResponse {
    image := aws.ImageArgs{}
    json.Unmarshal([]byte(*req.Body), &image)
    
    imageName := core.ToBase36(time.Now().UnixMilli())
    image.Name = imageName
    image.Folder = "img-productos"
    image.Resolutions = map[uint16]string{980: "x6", 540: "x4", 340: "x2"}

    if _, err := aws.SaveConvertImage(image); err != nil {
        return req.MakeErr("Error al guardar la imagen.", err)
    }

    return req.MakeResponse(map[string]string{"imageName": image.Folder + "/" + imageName})
}
```

---

## 10. Troubleshooting & FAQ

### Q: Why isn't my query returning anything?
**A**: Ensure you are filtering by `EmpresaID`. If you are filtering by a non-primary key column, check if you need to call `.AllowFilter()`. Also, verify if the `Status` field matches your sync logic.

### Q: Why do we use Sunix timestamps instead of standard Unix timestamps?
**A**: Sunix timestamps are optimized for synchronization and primary key generation patterns, providing higher precision and better compatibility with our `SUnixTimeUUID` format.

### Q: How do I handle batch updates to unrelated tables?
**A**: Use `errgroup` to execute the updates in parallel, and ensure each operation handles its own errors properly. Since ScyllaDB doesn't support cross-table transactions, verify the state manually if one fails.

---

## 11. Developer's Handbook: Best Practices

1.  **Always use Spanish** for user-facing error messages to maintain consistency across the platform.
2.  **Never trust client input** for fields like `EmpresaID`, `CreatedBy`, or `Updated`. Always assign these from the trusted `req.Usuario` context.
3.  **Optimize payloads** by using `query.Exclude` for large fields like images or long descriptions in list views.
4.  **Use `errgroup`** for parallelizing independent queries to significantly reduce API latency.
5.  **Follow naming conventions**: `GetEntity` for data retrieval and `PostEntity` for modifications.
6.  **Protect immutable fields**: Use `db.UpdateExclude` for fields like `Created`, `CreatedBy`, or calculated balances during a standard update.
7.  **Use `core.Log`** for technical debugging, but keep user messages clear and polite.

---

## 12. Serialization: JSON vs CBOR Tags

Structs in `app/types` must be tagged for both formats to ensure consistency.

### JSON Tagging Rules
- **NEVER** use explicit field names in JSON tags (e.g., `json:"nombre,omitempty"` is forbidden).
- **ALWAYS** use `json:",omitempty"` as the default for all fields.
- **EXCEPTIONS**:
    - `Updated` MUST use `json:"upd,omitempty"`.
    - `Status` MUST use `json:"ss,omitempty"`.

### CBOR Tagging Rules
- **CBOR**: Used for high-performance internal and mobile traffic. Use integer keys.
- Integer keys in CBOR are significantly more efficient than string keys.

```go
type Entity struct {
    ID     int32  `json:",omitempty" cbor:"1,keyasint,omitempty"`
    Nombre string `json:",omitempty" cbor:"2,keyasint,omitempty"`
    Status int8   `json:"ss,omitempty" cbor:"3,keyasint,omitempty"`
    Updated int64 `json:"upd,omitempty" cbor:"4,keyasint,omitempty"`
}
```

---

## 13. Summary Checklist for Developers

- [ ] Every GET query includes the `EmpresaID` filter.
- [ ] Every POST operation assigns `EmpresaID` from `req.Usuario`.
- [ ] `updated > 0` logic correctly includes records with `Status = 0`.
- [ ] New records (`ID <= 0`) use `GetCounter` for ID generation.
- [ ] Sensitive fields are protected using `db.UpdateExclude`.
- [ ] Structs are properly tagged with both JSON and CBOR integer keys.
- [ ] All user-facing error messages are in Spanish.
- [ ] Use `errgroup` for multiple unrelated data fetches.
- [ ] Verify permissions using `req.Usuario.AccesosIDs`.

---

## Appendix: String Prefix Queries

To query strings with a specific prefix (e.g., all IDs starting with "REF-"):

```go
prefix := "REF-"
query.ID.GreaterEqual(prefix).
      ID.LessThan(prefix + "\uFFFF")
```

The `\uFFFF` character represents the highest possible Unicode value in the Basic Multilingual Plane, effectively creating an upper bound for the prefix query.

---
*End of Document*
