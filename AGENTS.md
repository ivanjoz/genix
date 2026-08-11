# AGENT OPERATIONAL PROTOCOL (V2)

IMPORTANT: You are not an autonomous agent. The human is watching you very step and it has all the answers. Keep the human in the loop and ask questions.

IMPORTANT: If you have any questions, ask. NEVER have long trains of thought with yourself. Explan your rationale, you are pair programming.

## 1. INTELLIGENT RESEARCH
- **One-Shot Research:** Gather enough info in one search to form a hypothesis. Avoid "research loops" where you search for the same thing multiple times without writing code.

## 2. LOOP DETECTION & PREVENTION
- **The "Two-Strike" Rule:** If a tool call (shell command or file edit) returns the same error twice, or if the code state doesn't change after an application, you are STUCK.
- **STALL PROTOCOL:** Upon detecting a loop, you MUST stop all autonomous actions and present the following to the user:
    1. **Summary:** What was attempted and why it failed.
    2. **Hypothesis:** Your best guess on the root cause.
    3. **Assumption:** A specific assumption you are making to move forward.
    4. **The Ask:** "I'm stuck. Should I try [Proposed Fix] based on my assumption, or do you have a different direction?"

## 3. ITERATION STYLE
- **Keep in Loop:** Every tool execution should be preceded by a 1-sentence "Intent" (e.g., "Updating the ScyllaDB connection string to test the timeout hypothesis").
- **Extensive Logging**: Always implement and use debug logs extensively to diagnose errors and trace execution flow.

## 4. RULES
- **IMPORTANT — ALWAYS CHECK SKILLS FIRST:** At the start of every task, review the available skills list (shown in the system-reminder block at conversation start). Skills contain authoritative documentation and project conventions for common operations. If any skill matches the task, invoke it via the Skill tool BEFORE doing manual exploration or writing code. The skills list may change — never rely on memory, always re-check.
- This project in is pre-alpha, you can remove deprecated stuff. DO NOT implemente backwards compatibility.
- NEVER write more code than necesary. ALWAYS TRY to reduce code size to the minimun posible.
- Search for the correct .md documentation before proceed
- If some points in the task are unclear, stop and ask for clarification
- ALWAYS add concise comments in every code block to explain the rationale and the goal, especially when code contains business logic.
- ALWAYS use expresive names for varibles and functions. DONT USE generic names.
## Project Overview

The Genix project is an ERP and E-commerce platform for small businesses. It consists of a Go backend and a Svelte.js frontend. The project is currently migrating its frontend from Solid.js to Svelte.js.

## Backend

The backend is written in Go and uses ScyllaDB/Cassandra as its database. The backend code is located in the `backend/` directory.

## Key Documentation Files

### Project Overview
- **README.md** - General project overview: ERP+Ecommerce for small businesses

### Deployment
- **DEPLOYMENT.md** - Deployment options including AWS Lambda + ScyllaDB on VPS/EC2, self-host deployment with systemd

### Backend Documentation
- **backend/README.md** - Brief overview of the Go backend for Genix
- **backend/db/** - The project's single ORM entry point. ALL application code imports `app/db` and never a database driver. `db/driver.go` holds the one declaration that names a database; pointing it at another driver switches the whole project.
- **backend/genix-orm/db/** - Driver-agnostic ORM layer shared by every driver: schema declaration, columns, predicates, the accessor engine, the `Executor` contract
- **backend/genix-orm/scylla/ORM_INTERNALS.md** - Deep dive into ScyllaDB driver internals: memory model, reflection engine, and query optimization
- **backend/docs/CREATE_API_HANDLERS.md** - API handler development guide, MUST read before creating APIs. Key concepts: "updated" parameter for delta responses, query examples, conventions.
- **backend/docs/ORM_DATABASE_QUERY.md** - Comprehensive ORM documentation covering model definitions, CRUD operations, query building

### Server Utils (Rust)
- **server_utils/README.md** - Un solo binario Rust (`genix-server-utils`) con dos transportes: el puerto **TCP crudo** (sección `[server_utils]` de config.toml — no `rate_limit.address`: el puerto es del proceso, no de un servicio; `public` decide el bind, `0.0.0.0` o loopback, y `host` es sólo la dirección que marca el cliente), donde el opcode del frame enruta al **credit rate limiter** (`0x01`) o al **lock service** (`0x02`/`0x03`), y el **SSE bridge** (HTTP) que sostiene el stream del navegador cuando el backend corre en Lambda (que no puede mantener conexiones abiertas). Contratos, tokens y los dos secretos (`internal_apikey` proceso-a-proceso, `secret_phrase` sólo para el token de sesión). Diseños en **server_utils/PLAN.md**, **server_utils/PLAN_LOCK_SERVICE.md** y **server_utils/PLAN_SSE_BRIDGE.md**.
- El **lock service** serializa una acción entre Lambdas concurrentes: clave `(action:u16, identifier:i64)` elegida por el backend, un solo poseedor, y la propiedad va atada a la conexión TCP (si el cliente muere, el lock se libera solo). Cliente Go en `backend/core/server_utils/` (conexión multiplexada compartida con el limiter), reexportado en `core` por `backend/core/server_utils_api.go`; `ErrLockBusy` es una respuesta real (rechaza al cliente) y `ErrLockUnavailable` es ausencia de respuesta (cada call site decide si sigue sin lock o corta).
- **server_utils/LOCK_SERVICE_WALKTHROUGH.md** - Recorrido end-to-end con diagramas: la carrera que el lock resuelve, el handshake, los bytes exactos de un `LOCK_ACQUIRE` real, los tres modos de fallo y dónde vive cada pieza del código. Leer esto antes que los PLAN_*.
- **scripts/CONFIGURE_SERVER_UTILS.md** - Despliegue en un servidor: `configure_server_utils.py` compila con cargo, instala las units de systemd y el vhost de Nginx del bridge (HTTP/3 cuando hay certificado, sin buffering para SSE). El puerto del rate limiter no se expone.

### Frontend Documentation
- **frontend/FRONTEND.md** - Monorepo architecture with independent ecommerce app, directory structure, package system, development workflow
- **frontend/docs/UI_COMPONENTS.md** - UI component library documentation: Page, OptionsStrip, Layer/Modal components, form components, VTable, services
- **frontend/ecommerce/ECOMMERCE.md** - Ecommerce integration notes: thumbhash implementation, routes, CSS hashing
- **frontend/docs/SERVICES_GUIDE.md** - Guide for creating frontend services (connectors), explaining Cached Services (Delta Cache) vs. Report Services. ALWAYS read before creating one.

### Scripts
- **scripts/CREATE_EDIT_TABLE.md** - Creates new database table structures and adds columns to existing tables. USE ALWAYS.
- **scripts/CHECK_TABLES_SCRIPT.md** - Validates data model conventions for the custom ORM
- **scripts/SCRIPTS.md** - Central dispatcher and wrapper script management for project utilities
- **scripts/DEPLOYER.md** - `deploy.sh` es un wrapper del TUI en `scripts/deployer/` (Go + Bubble Tea v2): navegación Environment / Actions / Scripts con teclado y mouse, y el orden fijo de ejecución. Reemplaza a `app.sh`, que queda deprecado
- **scripts/CONFIGURE_SERVER.md** - Despliegue del backend en un VPS: units de systemd, auto-reload del binario y proxy inverso de Nginx
- **scripts/CONFIGURE_DB.md** - Despliegue del host de datos: `configure_db.py` configura ScyllaDB (sysconfig, yaml, superusuario, keyspace) e instala GenixSearch y Qdrant desde sus releases musl estáticos, escribiendo `search.url`/`search.password` y `qdrant.host` en config.toml. Qdrant se configura por variables `QDRANT__*` en un EnvironmentFile y su api key es `internal_apikey`

### General Rules
- ALWAYS save dates in UnixDay int16 format: The number of days since unix-epoch
- ALWAYS save datetime as int32 SUnixTime(). SUnixTime = int32((time.Now().Unix() - 1e9) / 2)

### Frontend Rules
- Use untrack inside $effect to avoid render loops
- GetHandler fetched records need fields: "upd" (Updated) and "ID" (unique id) for delta cache. Or use GetHandler.keyID or .KeysIDs for setting another field.
- Tailwind --spacing is 1px. So "h-4" is actually 4px.
- NEVER use font-weight or font-size in a css class. USE tailwind instead.
- ALWAYS use this helpers functions:
	- formatTime(unix day | unix time, layout)

### Backend Rules
- NEVER trust the client. ALWAYS validate the required field and consistency of the data, and return a descriptive error if any validation fails.
- Naming for parallel-array `Detail*` columns: use SINGULAR words for the field name; the only exception is the `IDs` suffix, which stays plural. Examples: `DetailProductIDs`, `DetailProductQuantity` (not `DetailProductQuantities`), `DetailProductPrice` (not `DetailProductPrices`), `DetailProductPresentationIDs`, `DetailSupplyIDs`, `DetailSupplyQuantity`, `DetailSupplyPrice`. Applies to both the Go struct fields and the frontend interface mirrors.
