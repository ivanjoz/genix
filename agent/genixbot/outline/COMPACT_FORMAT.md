# Compact page format (draft v2, 2026-11-24)

An outline of a Genix UI page for small fine-tuned models. Produced by `outline.Render` (this package) from a
captured page state (`<name>.snapshot.html` + `<name>.raw.html` + `<name>.components.json`), written to
`<name>.compact.txt`. Run: `cd generators && go run ./cmd/compact-page`.

## Example (customer form open over the customer list)

```
Page /crm/customers | Clientes
Layer 71 | en segundo plano: Table 75 (24 filas)
Layer 76 | Nuevo Cliente
 Button 77 | Guardar
 Button 78 | Cerrar
 Input 79 | Nombre = ""
 Select 82 | Tipo Persona = [1] Persona | opciones: [1] Persona, [2] Empresa
 Input 84 | Email = ""
 Input 87 | Registry Number (RUC) = ""
 Input 90 | Días de crédito = "0"
 Checkbox 93 | Agente de retención del IGV (retiene 3 %) = no
 Select 94 | Departamento / Provincia / Distrito = "" | 2088 opciones
```

After filling two fields, the action result is a delta instead of the whole page:

```
Delta /crm/customers
~ Input 79 | Nombre = "Pedro Pascal"
~ Input 87 | Registry Number (RUC) = "10456789012"
```

## One schema for every line

Every line is `Type [id] | content`, indented one space per level under its parent. Nothing is printed
outside that shape: loose text is a `Text` line, the page is a `Page` line whose id slot holds the route.
A line without an id cannot be acted on (text, read-only tables and their rows, `More` notes).

| Type | Meaning |
|---|---|
| Genix component types (`Input`, `Select`, `Button`, `Layer`, `Table`, `OptionsStrip`, `Card`…) | registered handles, with their Genix id |
| `Page` | first line: `Page <route> \| <title>` |
| `Delta` | first line of an action result that lists only changes: `Delta <route>` |
| `Text` | visible text: `Text \| Debe seleccionar un almacén.` |
| `Toast` | a notification on screen, right after the `Page` line: `Toast \| Debe ingresar un email válido.` (a refused save says why only there) |
| `Table` / `Row` without id | an unregistered, read-only table |
| `More` | items left out: `More \| +12 filas`, `More \| +25 tarjetas` |

## Rules

| Element | Rendering |
|---|---|
| Component | `Type id \| label`; the id is Genix's own handle, passed verbatim to the action tool |
| Value fields (Input, FilterInput, DateInput, Select, MultiSelect, Cell*) | always ` = "value"`; empty is `= ""` |
| Select value | `= [id] text`; options inline when the frontend lists them (≤ 12), else `\| N opciones` (the model searches) |
| Checkbox | `= sí` / `= no` |
| Other state | `[abierto]`, `[vacío]` (LayerStatic, ImageUploader) |
| Selected / active item | ` *` after the label (options, tabs, rows) |
| Methods | omitted when they are the type's default (Input → setValue, Button → click, Select → search,select,getOptions…); otherwise `{m1,m2}` |
| Tabs | `OptionsStrip \| [id] Texto * · [id] Texto` (same for `PageViews`) |
| Registered table | `Table id \| header \| …`, then `Row tableId:rowId \| cell \| …` in header order; **5 rows**, then `More \| +N filas` (N counts every row the table holds: a virtualized table reports them with `data-rows-count`); no rows → `Text \| sin filas` |
| Grouped headers (colspan/rowspan) | flattened to one name per leaf column: `Costo Unitario › Consolidado` (read from the raw DOM) |
| Cell with a component | `{CellInput 75:101 = "x"}` |
| Card grids | **5 cards** per group, then `More \| +N tarjetas` |
| Unregistered table | `Table \| header \| …` with `Row \| …` rows and no ids |
| Div grid (`role="row"` / `role="cell"`, genix-ui TableGrid) | read like a table: `columnheader` cells are the headers, each `row` a `Row` line |
| Text | `Text \| …`; a long list of items splits into one `Text` line per item, capped at 5 |
| Labels | Spanish half of `English\|Español`; `word\|palabra` inside an English sentence keeps the Spanish word; ` \| ` inside labels and cells becomes ` / ` |
| **Focus mode** | while a floating layer is open (raw DOM `data-layer-id`), every container outside it collapses to `Type id \| en segundo plano: Table N (k filas)`; leaf components outside (buttons, filters) stay |
| **Delta** | action results list only changed lines (`~` changed, `+` new, `-` removed key); the full outline is sent instead when the route changes or more than half of the lines changed (a layer opened) |
| Dropped | app header (title kept), empty `TopLayerSelector`/`TopLayerDatePicker`, unregistered `button`/`input`/`img`, `aria-live` regions (their toasts are the `Toast` lines), text repeating the enclosing component's label |

## Token choices (measured on LFM3's tokenizer)

- All separator candidates cost one token (` | `, tab, `;`, `:`, `·`); ` | ` stays.
- Words cost: `(seleccionado)` 5 tokens and `(activa)` 4 vs ` *` 1; `, usa search` 3; long `More` sentences.
- Indentation: one space per level is the cheapest (−1.4% vs two spaces; a tab costs +1.2%).
- Together these micro-choices save ~3%; the large savings come from structure (rows, cards, focus, delta).

## Size (LFM3 tokenizer)

Current Genix snapshot capped at 12 rows per table (as the agent sends it) vs this format:

| State | Genix snapshot | Compact v1 (12 rows) | **Compact v2** | v2 / snapshot |
|---|---:|---:|---:|---:|
| branches-warehouses | 492 | 73 | 72 | 15% |
| cash-banks | 715 | 253 | 244 | 34% |
| customers-list | 1 481 | 717 | 334 | 23% |
| customers-new | 1 818 | 853 | **151** | 8% |
| pos | 9 506 | 1 186 | **438** | 5% |
| products-list | 1 276 | 578 | 311 | 24% |
| products-new | 2 255 | 1 030 | 532 | 24% |
| purchase-orders | 3 588 | 661 | 475 | 13% |
| stock-changes | 339 | 99 | 98 | 29% |
| suppliers-list | 1 417 | 701 | 341 | 24% |
| suppliers-new | 1 709 | 814 | **130** | 8% |
| users-profiles | 797 | 263 | 256 | 32% |
| **Total** | **25 393** | **7 228** | **3 382** | **13%** |

Delta after filling two fields of the customer form: 38 tokens vs 160 for the full outline. The agent also
prepends `PageSnapshotGrammar` (502 tokens) to every page result today; the fine-tuned model drops it.

## Known gaps

- Several action buttons have only English descriptions as labels ("Opens the side layer to create a new
  cliente."): Genix registers no Spanish label for them.
- Some registered components are not in the DOM (e.g. Select 88 "document type" on the customer form,
  shown only for one person type); like the current snapshot, the outline omits them.
- Product cards keep their visible figures unlabeled (`Stock 327 13.50`): the price has no label in the DOM.
- Text-list splitting is heuristic (blocks over 160 characters).
- Focus mode relies on `data-layer-id` marking floating layers; modals need the same check once captured.
