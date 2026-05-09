# Agent HTTP API

External-facing HTTP endpoint that lets an LLM agent (Claude Code, Gemini, …)
drive the running browser page. Sits on top of the existing WebSocket bridge:
the HTTP request fans out to the same `agent.invoke` RPC the Go backend uses
internally, then returns a fresh page snapshot in the same response.

For the in-page registry / DOM contract, see
[`frontend/ui-components/AGENTIC_COMPONENTS.md`](../../frontend/ui-components/AGENTIC_COMPONENTS.md).

## Endpoint

```
POST http://localhost:3589/agent
Content-Type: application/json
```

Local-only — bound to localhost, no auth, no CORS. Requires one connected
browser tab (the dev page that opens a WebSocket to `/ws/agent`).

## Request

```json
{
  "Actions": [
    { "ID": "58",      "Method": "select",   "Args": [235] },
    { "ID": "58:235",  "Method": "remove",   "Args": [] },
    { "ID": "60",      "Method": "setValue", "Args": ["Hello"] }
  ]
}
```

| Field    | Type   | Notes                                                                 |
| -------- | ------ | --------------------------------------------------------------------- |
| `ID`     | string | Component id from the HTML snapshot. Plain (`"58"`) or composite (`"58:235"`, `"7:12"`). |
| `Method` | string | One of the names in the tag's `methods="..."` attribute.              |
| `Args`   | array  | Method arguments. Omit / pass `[]` for void methods.                  |

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
<CellInput id="7:12" value="3" type="number"/>
```

- **Plain** (`"58"`): the action targets handle 58 directly.
- **Composite** (`"58:235"`, `"7:12"`): the first segment is the parent handle.
  The server splits the id, calls the parent's method, and **prepends the full
  composite id as `Args[0]`** so the handle's implementation can route by
  child. Examples:
  - `{ID:"58:235", Method:"remove"}` → `SearchCard(58).remove("58:235")`
  - `{ID:"7:12", Method:"setValueChild", Args:["Hello"]}` →
    `Table(7).setValueChild("7:12", "Hello")`

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
| 503    | No browser is currently connected to `/ws/agent`.     |
| 502    | Action ran but the post-action page snapshot failed.  |

## Method reference

The available methods come from `frontend/core/agent/registry.ts` —
`AgentMethodMap`. Each component advertises only the subset it implements via
its `methods="..."` HTML attribute. Common ones:

| Method            | Args                       | Used by                        |
| ----------------- | -------------------------- | ------------------------------ |
| `setValue`        | `(value)`                  | `Input`                        |
| `search`          | `(text)`                   | `SearchSelect`, `SearchBox`    |
| `select`          | `(...ids)`                 | `SearchSelect`, `SearchCard`   |
| `remove`          | `(id)` (composite ok)      | `SearchCard` (Option marker)   |
| `getOptions`      | `(max?)` → `AgentOption[]` | `SearchSelect`, `SearchCard`   |
| `click`           | `()`                       | `Button` markers, `ButtonLayer`|
| `open` / `close`  | `()`                       | `Layer`, `Modal`               |
| `setValueChild`   | `(childID, value)`         | `Table` → cell                 |
| `searchChild`     | `(childID, text)`          | `Table` → cell                 |
| `getOptionsChild` | `(childID, max?)`          | `Table` → cell                 |

When the server splits a composite id, the `childID` arg is supplied
automatically — you only pass the trailing args.

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
```

## Implementation pointers

| Concern                       | File                                                       |
| ----------------------------- | ---------------------------------------------------------- |
| HTTP handler + id resolution  | [`http.go`](./http.go)                                     |
| WS RPC to the browser         | [`ws.go`](./ws.go) (`request`, `IsConnected`)              |
| Public Go API (`Invoke`, …)   | [`agent.go`](./agent.go)                                   |
| Wire types                    | [`protocol.go`](./protocol.go) (`InvokePayload`, `PageContent`) |
| HTML cleaner / compact tags   | [`parse_html.go`](./parse_html.go)                         |
| Route registration            | [`../main.go`](../main.go) (`mux.HandleFunc("POST /agent", …)`) |
