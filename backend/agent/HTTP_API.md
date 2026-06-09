# Agent HTTP API

External-facing HTTP endpoint that lets an LLM agent (Claude Code, Gemini, …)
drive the running browser page. Sits on top of the SSE+POST bridge: the HTTP
request fans out to the same `agent.invoke` RPC the Go backend uses internally
(commands pushed to the browser over the tab's `/agent/stream` SSE, replies
POSTed back to `/agent/in`), then returns a fresh page snapshot in the same
response.

For the in-page registry / DOM contract, see
[`frontend/ui-components/AGENTIC_COMPONENTS.md`](../../frontend/ui-components/AGENTIC_COMPONENTS.md).

## Endpoints

```
POST http://localhost:3589/agent          # run actions, return post-action snapshot
GET  http://localhost:3589/agent?get=menu # read-only TSV menu structure
Content-Type: application/json
```

Local-only — bound to localhost, no auth, no CORS. Requires one connected
browser tab (the dev page that opens an SSE stream to `/agent/stream`).

## Request

```json
{
  "Actions": [
    { "ID": "58",       "Method": "select",   "Args": [235] },
    { "ID": "58:235",   "Method": "remove",   "Args": [] },
    { "ID": "60",       "Method": "setValue", "Args": ["Hello"] },
    { "ID": "38:101",   "Method": "setValue", "Args": ["3"] },
    { "ID": "38:100",   "Method": "select",   "Args": [], "ReturnPageContent": true }
  ]
}
```

| Field    | Type   | Notes                                                                 |
| -------- | ------ | --------------------------------------------------------------------- |
| `ID`     | string | Component id from the HTML snapshot. Plain (`"58"`) or composite (`"58:235"`, `"7:12"`). |
| `Method` | string | One of the names in the tag's `methods="..."` attribute.              |
| `Args`   | array  | Method arguments. Omit / pass `[]` for void methods.                  |
| `ReturnPageContent` | bool | Optional. When true on `navigate` or an invocation action, the browser waits for route/DOM updates and returns the fresh page snapshot in that action result. |

Actions run **sequentially, stop on first error**. An empty array
(`{"Actions": []}`) is valid and returns the current page snapshot — useful as
a "what does the page look like now?" call.

### ID format

The HTML snapshot tags every component with an `id` attribute:

```html
<SearchCard id="58" methods="select,remove,getOptions">
  <Option id="58:235" selected methods="remove">Zumos</Option>
</SearchCard>
<Input id="60" label="Nombre" methods="setValue"/>
<Table id="38">
  <TableRow id="38:100" methods="select"> … </TableRow>
  <CellInput id="38:101" value="3" type="number" methods="setValue"/>
</Table>
```

- **Plain** (`"58"`): the action targets handle 58 directly.
- **Composite** (`"58:235"`, `"38:101"`): the first segment is the parent
  handle. The server splits the id and routes the call to the parent. Two
  forms exist:
  - **Cell routing** — for `setValue` / `search` / `getOptions` the server
    rewrites the call to the parent's `*Child` variant and passes the bare
    child id (no prefix). Example: `{ID:"38:101", Method:"setValue",
    Args:["Hello"]}` → `Table(38).setValueChild(101, "Hello")`.
  - **Pass-through** — every other method receives the full composite id as
    `Args[0]` so the handle can split it itself. Examples:
    `{ID:"58:235", Method:"remove"}` → `SearchCard(58).remove("58:235")`,
    `{ID:"38:100", Method:"select"}` → `Table(38).select("38:100")`.

## Response

```json
{
  "Results": [
    { "OK": true,  "Value": null },
    { "OK": true,  "Value": null },
    { "OK": false, "Error": "no method setValue on handle 60" }
  ],
  "Page": {
    "Components": [
      { "ID": 58, "Type": "SearchCard", "Label": "Categorías", "Methods": ["select","remove","getOptions"] },
      { "ID": 60, "Type": "Input",      "Label": "Nombre",     "Methods": ["setValue"] }
    ],
    "HTML": "<div>...sanitized body...</div>"
  }
}
```

| Field             | Notes                                                                |
| ----------------- | -------------------------------------------------------------------- |
| `Results[i].OK`    | `true` if the action ran. `false` halts the batch.                  |
| `Results[i].Value` | Raw JSON of the method's reply (e.g. `getOptions` → option array).  |
| `Results[i].Error` | Error string when `OK:false`.                                       |
| `Page.Components`  | Live registry: `{ID, Type, Label, Methods}` per registered handle.  |
| `Page.HTML`        | Sanitized `document.body` HTML. Compact form is *not* applied here — agents that want the parsed view should run `agent.ParsePageHTML` themselves or rely on the components list. |

`Results.length ≤ Actions.length`. If action *k* fails, indices ≥ *k+1* are
absent. `Page` is **always** present, reflecting the state after the last
successful action (or the initial state if action 0 failed).

## HTTP status codes

| Status | When                                                  |
| ------ | ----------------------------------------------------- |
| 200    | Request was processed; check `Results[i].OK` per action. |
| 400    | Body is not valid JSON / wrong shape.                 |
| 405    | Wrong method (only POST is mounted).                  |
| 503    | No browser is currently connected to `/agent/stream`. |
| 502    | Action ran but the post-action page snapshot failed.  |

## Method reference

The available methods come from `frontend/core/agent/registry.ts` —
`AgentMethodMap`. Each component advertises only the subset it implements via
its `methods="..."` HTML attribute. Common ones:

| Method           | Args                       | Used by                                         |
| ---------------- | -------------------------- | ----------------------------------------------- |
| `setValue`       | `(value)`                  | `Input`, `CellInput` (composite id)             |
| `search`         | `(text)`                   | `Select`, `SearchBox`, `CellSelect`       |
| `select`         | `(...ids)`                 | `Select`, `SearchCard`, `Table`/row/cell  |
| `remove`         | `(id)` (composite ok)      | `SearchCard` (Option marker)                    |
| `getOptions`     | `(max?)` → `AgentOption[]` | `Select`, `SearchCard`, `CellSelect`      |
| `click`          | `()`                       | `Button` markers, `ButtonLayer`                 |
| `open` / `close` | `()`                       | `Layer`, `Modal`                                |
| `navigate`       | `(route)`                  | global (no `ID`); calls `goto(route)` in the SPA — see "Navigate action" below |

For cell calls (composite id, `setValue`/`search`/`getOptions`) the server
strips the parent prefix and rewrites the method to the table's `*Child`
variant — callers always use the bare verb that appears in the cell's
`methods="..."` attribute.

### Navigate action

The side-menu DOM is hidden from the page snapshot — agents read the menu via
`GET /agent?get=menu` (below) and move between pages with a global `navigate`
action that takes a route. There is no `ID` because the action is page-level,
not bound to a handle:

```json
{
  "Actions": [
    { "Method": "navigate", "Args": ["/comercial/sale_order_create"], "ReturnPageContent": true }
  ]
}
```

After the route change the same response shape applies: the post-action
`Page` snapshot reflects the new page.

### GET /agent?get=menu

Returns the current user's accessible side-menu as TSV (same access filter as
the visual menu). Use it to discover routes before issuing a `navigate` action.

```bash
curl -s 'http://localhost:3589/agent?get=menu'
```

```tsv
group_id	group_name	option_name	route	description
1	CONFIGURACIÓN	Mi Empresa	/configuracion/parametros	Company settings. Edit general data...
1	CONFIGURACIÓN	Usuarios	/seguridad/usuarios	System user management...
4	Logística	Cambios Stock	/logistica/products-stock	Warehouse stock management...
4	Logística	Gestión de Compras	/logistica/gestion-compras	Supply management...
```

Groups with no accessible options are omitted. Status codes mirror POST
(`503` if no browser is connected, `502` if the browser RPC fails).

## Example workflow

Pull the page, find the component you want, then act:

```bash
# 1. Snapshot
curl -s -X POST http://localhost:3589/agent \
  -H 'Content-Type: application/json' \
  -d '{"Actions":[]}' | jq '.Page.Components'

# 2. Type into the "Nombre" Input (id=60), then add category 235 to SearchCard 58
curl -s -X POST http://localhost:3589/agent \
  -H 'Content-Type: application/json' \
  -d '{
    "Actions": [
      { "ID": "60",  "Method": "setValue", "Args": ["Jugo de Naranja"] },
      { "ID": "58",  "Method": "select",   "Args": [235] }
    ]
  }' | jq '.Results, .Page.Components'

# 3. Remove the option we just added
curl -s -X POST http://localhost:3589/agent \
  -H 'Content-Type: application/json' \
  -d '{ "Actions": [ { "ID": "58:235", "Method": "remove" } ] }' | jq .

# 4. Discover the menu, then navigate to a page from it
curl -s 'http://localhost:3589/agent?get=menu'
curl -s -X POST http://localhost:3589/agent \
  -H 'Content-Type: application/json' \
  -d '{ "Actions": [ { "Method": "navigate", "Args": ["/comercial/sale_order_create"] } ] }' | jq .Results
```

## Implementation pointers

| Concern                       | File                                                       |
| ----------------------------- | ---------------------------------------------------------- |
| HTTP handler + id resolution  | [`http.go`](./http.go)                                     |
| SSE+POST bridge to the browser | [`ws.go`](./ws.go) (`request`, `HandleStream`, `HandleIn`, `IsConnected`) |
| Public Go API (`InvokeBatch`, `Navigate`, …) | [`agent.go`](./agent.go)                    |
| Wire types                    | [`protocol.go`](./protocol.go) (`InvokePayload`, `InvocationResult`, `PageContent`) |
| HTML cleaner / compact tags   | [`parse_html.go`](./parse_html.go)                         |
| Route registration            | [`../main.go`](../main.go) (`mux.HandleFunc("POST /agent", …)`) |
