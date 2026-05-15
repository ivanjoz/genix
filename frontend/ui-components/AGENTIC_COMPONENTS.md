# Agentic Components

This document describes how UI components expose themselves to the automation agent
defined in `frontend/core/agent/`.

## Core principle

The agent reads the page through two channels:

1. A **sanitized HTML snapshot** of `document.body`.
2. A **registry of handles**, each carrying a few callable methods.

If the state of a component is already visible in the HTML snapshot, the
component does **not** need a getter method for it. Methods exist only for
actions the agent cannot perform by reading the DOM (trigger filtering,
dispatch a click, write a value the user couldn't otherwise type).

That principle decides what each component exposes.

## Registration

Every interactive component registers itself in an `$effect` that returns the
cleanup callback:

```ts
const componentID = Env.getComponentID();

$effect(() => {
  return Agent.register({
    id: componentID,
    type: "Select",
    label: label || placeholder || "",
    search: (text: string) => { applySearch(text, true); },
    select: (...ids: (number | string)[]) => { /* ... */ },
    getOptions: (max = 50) => { /* ... */ },
  });
});
```

The selected value is exposed through the DOM, not the handle:

```svelte
<div
  data-id="Select:{componentID}"
  data-value={selectedId ? `[${selectedId}] ${selectedLabel}` : ""}
>
  …
</div>
```

- `type` is the component name (`"Select"`, `"Input"`, `"Table"`, …).
- `label` falls back to `placeholder` when no label prop is set, so every
  handle is identifiable. Same rule applies to the SearchBox / SearchSelect.
- The handle only carries the methods that component actually supports;
  `registry.ts` types the methods structurally (no per-component union).

## DOM contract

Every registered component root sets:

```html
<div data-id="<Type>:<componentID>"> … </div>
```

State that the agent needs to read is mirrored into attributes on that root or
on its children:

| Attribute | Meaning |
|---|---|
| `data-id="<Type>:<id>"` | identifies a registered handle |
| `data-id="Option:<id>"` | a selectable option / chip / step inside a component |
| `data-id="TableRow:<tableID>:<rowID>"` | a selectable table row (no handle of its own — owned by the parent `Table`); the composite id is what the agent passes back to `Table.select` |
| `data-id="Button:<id>"` | a clickable command button (auto-registered with a `click` method) |
| `data-menu-root="true"` | marks the side-menu DOM (desktop nav + mobile drawer); the backend parser drops these subtrees from the agent snapshot — agents read the menu via `GET /agent?get=menu` and move via the `navigate` action |
| `data-agent-hidden="true"` | opts an element (and its entire subtree) out of the page snapshot at the *frontend* layer — `getPageContent` strips it before sanitizing. Use it on parts of the UI the agent must never see / interact with (e.g. the agent chat widget itself, debug panels, dev-only overlays). |
| `data-value="…"` | current value of the component (always present on inputs/selects, even when the value is also rendered visibly — keeps the agent's read path uniform) |
| `data-label="…"` | label or placeholder for input-style components (`Input`, `SearchSelect`, `DateInput`) — `label || placeholder || ""`. Saves the agent walking to the inner `<label>` / `placeholder`. |
| `data-type="text\|number\|other"` | input kind for input-style components: `"number"` for numeric inputs, `"text"` for free-text inputs, `"other"` for everything else (selects, dates, colors, …). |
| `data-options-count="<n>"` | total number of options on a `Select` (always present, even when options are not in the DOM). The agent reads this to decide whether to use the inline option list, call `getOptions`, or `search`. |
| `aria-label="…"` | the human label of a button when its body is an icon |
| `data-selected="true"` | present on the currently-selected option / row / checkbox (the bare HTML `selected` attribute only validates on `<option>`, so we use the `data-` form everywhere for uniformity) |

### Buttons

`Modal`, `Layer`, `ButtonList`, `Renderer`, etc. render command buttons. Every
such button uses the same convention so the agent can address it directly:

```svelte
<button
  data-id="Button:{buttonID}"
  aria-label={label}
  data-value={role}     <!-- "save" | "delete" | "close" | custom -->
  onclick={…}
>{label}</button>
```

`data-value` carries the semantic role (so the agent can find "the save
button" without guessing from text). Each button auto-registers a handle with
a `click()` method; the parent component does not need to expose `save`,
`delete`, etc. separately.

## Method vocabulary

Components draw from this shared set. Names are stable across components so
the agent doesn't learn a new verb per component.

| Method | Signature | Used when |
|---|---|---|
| `search` | `(text: string) => void` | type into a search input that filters options |
| `select` | `(...ids: (number \| string)[]) => void` | select one or many options. Multi-select components accept several ids in one call; single-select components ignore ids past the first |
| `remove` | `(id: number \| string) => void` | remove one selected item from a multi-select |
| `setValue` | `(value: string \| number) => void` | write a value the user would otherwise type |
| `click` | `() => void` | dispatch the component's primary action (used by buttons and components whose only verb is "click me") |
| `open` / `close` | `() => void` | open or close a layer / drawer |
| `deleted` | `() => void` | destructive removal action (e.g. delete an uploaded image) |
| `getOptions` | `(max?: number) => AgentOption[]` | return options that are **not** currently in the DOM (e.g. a closed dropdown's full list). Don't expose this if the options are visibly rendered |
| `getValue` | `() => AgentOption \| AgentOption[] \| undefined` | return the current value. Avoid when `data-value` is enough; use only for multi-value components where serialising into one attribute would be lossy |

`selectMany` does not exist — `select` is variadic.

## Skipped components

Pure-display, infrastructural, or trivially composed components don't register:
`Virtualizer`, `LoginForm`, `Charts`, `ChartCanvas`, `Imagehash`, `charts/*`,
`popover2/*`, `micro/*`, `files/*`, and `vTable/CardsList`.

## Component spec

| Component | `type` | data-id additions | Methods |
|---|---|---|---|
| `Input` | `Input` | root has `data-value="<current value>"` | `setValue` |
| `SearchSelect` (Svelte file) | `Select` | root has `data-value="[id] text"` (id of the selected option followed by its display text; empty string when nothing is selected) and `data-options-count="<n>"` (total option count). When the full list is short (≤12) the backend renderer additionally injects `options="[id] label\|[id] label\|…"` on the `<Select>` tag from a frontend `getOptions(13)` probe — agents read the inline list directly and skip the round-trip. | `search`, `select`, `getOptions` (options not in DOM until opened) |
| `SearchCard` | `SearchCard` | wraps a child `Select`; renders selected chips as `Option:<id>` | `select(...ids)`, `remove(id)` |
| `SearchDualCard` | `SearchDualCard` | wraps two child `Select`s; chips as `Option:<id>` | `select(...ids)`, `remove(id)` |
| `Card` | `Card` | bare `<div>` wrapper. Only registers a handle when an `onClick` prop is set; otherwise it is invisible to the agent | `click` (only when `onClick` is set) |
| `Checkbox` | `Checkbox` | root carries `data-selected="true"` when checked | `click` |
| `CheckboxOptions` | `CheckboxOptions` | each option as `Option:<id>` with `data-selected="true"` when chosen | `select(...ids)`, `remove(id)` |
| `OptionsStrip` | `OptionsStrip` | parent tag renders bare (no id/methods — structural, like `PageViews`); each option is `Option:<stripID>:<optionID>` with `data-selected="true"` on the active one. Routing goes through the strip's `select` method, which strips the parent prefix from the composite id. | `select(id)` (parent routing only; the agent always calls select on the composite Option id) |
| `PageViews` (in `AppHeader`) | `PageViews` | the header tab strip rendered when a `Page` declares `options`. The tag itself renders as a bare `<PageViews>` (no id/methods) — interaction lives on the children. Each tab is an `Option:<pageViewsID>:<optionID>` with `data-selected="true"` on the active one; routing goes through the PageViews handle's `select` method | `select(id)` (parent routing only; never called directly on `<PageViews>`) |
| `ArrowSteps` | `ArrowSteps` | each step as `Option:<id>` with `data-selected="true"` on the current step | `select(id)` |
| `DateInput` | `DateInput` | root has `data-value="YYYY-MM-DD"` | `setValue` |
| `ColorPicker` | `ColorPicker` | root has `data-value="<hex>"` | `setValue` |
| `ImageUploader` | `ImageUploader` | — | `click` (open file picker), `deleted` |
| `ButtonLayer` | `ButtonLayer` | root has `data-value="open"\|"closed"` | `open`, `close`, `click` |
| `ButtonList` | `ButtonList` | each item as `Button:<id>` | — (each item button registers its own `click`) |
| `Layer` | `Layer` | header buttons as `Button:<id>` with `data-value` and `aria-label` | `open`, `close`, `select` (only when an `options` prop is set) |
| `LayerStatic` | `LayerStatic` | — | `open`, `close` |
| `MobileLayerVertical` | `MobileLayerVertical` | — | `open`, `close` |
| `Modal` | `Modal` | header save/delete/close as `Button:<id>` with `data-value="save"\|"delete"\|"close"` | `close` |
| `TopLayerSelector` | `TopLayerSelector` | each option as `Option:<id>` | `search`, `select`, `close` |
| `TopLayerDatePicker` | `TopLayerDatePicker` | root has `data-value="YYYY-MM-DD"` | `setValue`, `close` |
| `vTable/VTable` | `Table` | each selectable row as `TableRow:<tableID>:<rowID>`, with `data-selected="true"` on the active one. Rows expose `methods="select"`. | (none on the Table tag — methods live on the rows/cells) |
| `vTable/TableGrid` | `Table` | same as `VTable` | (same) |
| `vTable/TableTree` | `Table` | same | (same) |
| `vTable/TableStream` | `Table` | same | (same) |
| `vTable/MobileCardsVirtualList` | `CardList` | same row/cell shape as `VTable`, with the parent type `CardList` instead of `Table`. Container root has `data-id="CardList:<componentID>"`. | (none on the CardList tag — methods live on the rows/cells) |
| `vTable/CellInput` | (no own handle) | rendered inside its parent `Table` or `CardList`. Root has `data-id="<parentID>:<cellID>"`, `data-cell-type="CellInput"`, `data-value`, `data-label`, `data-type`, `methods="setValue"`. The agent calls `setValue` on the cell id; the bridge rewrites it to the parent's `setValueChild`. | `setValue` (via parent) |
| `vTable/CellSelect` | (no own handle) | rendered inside its parent `Table` or `CardList`. Root has `data-id="<parentID>:<cellID>"`, `data-cell-type="CellSelect"`, `data-value="[id] text"`, `methods="search,select,getOptions"`. | `search`, `select`, `getOptions` (via parent) |

### Table-level cell routing

`Table` (and `CardList`) is the only registered handle for the container;
rows and cells (`CellInput`, `CellSelect`, `CellClick`) live underneath
with composite ids `<parentID>:<rowID>` and `<parentID>:<cellID>`. The
container itself doesn't advertise any methods — the agent reads them off
the row/cell tags and calls them with the composite id. The bridge handles
the rest:

| Call (from the agent) | Routing |
|---|---|
| `select("<table>:<rowID>")` | row id (multiple of 100) → toggles/selects that row. |
| `select("<table>:<cellID>", …optionIds)` | non-multiple-of-100 → forwarded to that cell's `select`. |
| `setValue("<table>:<cellID>", value)` | bridge rewrites to `Table.setValueChild(cellID, value)` → cell's `setValue`. |
| `search("<table>:<cellID>", text)` | bridge → `Table.searchChild` → cell's `search`. |
| `getOptions("<table>:<cellID>", max?)` | bridge → `Table.getOptionsChild` → cell's `getOptions`. |

Row ids are `(rowIndex + 1) * 100` (the lowest row id is `100`, never `0`).
Cell ids are `rowID + columnIndex + 1` so the column slot is 1-based and
cells never collide with rows. Cap of ~99 cells per row.
| `Renderer` | `Renderer` | each rendered button as `Button:<id>` | — (buttons register their own `click`) |
| `domain-components/SideMenu` | — (DOM only) | side-menu wrappers carry `data-menu-root="true"` so the parser drops them; agents fetch the menu via `GET /agent?get=menu` and move with the `navigate` action | `navigate` (global, not on a handle) |

`TableRow` is markup only — it doesn't register a handle. All row interaction
goes through the parent's `select(...ids)`. The four desktop table variants
(`VTable`, `TableGrid`, `TableTree`, `TableStream`) register as `type: "Table"`;
`MobileCardsVirtualList` (and the cards rendered through `CardsList`) registers
as `type: "CardList"`. The cell-routing semantics are identical between the
two.
