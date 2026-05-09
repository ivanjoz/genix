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
    type: "SearchSelect",
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
  data-id="SearchSelect:{componentID}"
  data-value={selectedId ? `[${selectedId}] ${selectedLabel}` : ""}
>
  …
</div>
```

- `type` is the component name (`"SearchSelect"`, `"Input"`, `"Table"`, …).
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
| `data-id="TableRow:<id>"` | a selectable table row (no handle of its own — owned by the parent `Table`) |
| `data-id="Button:<id>"` | a clickable command button (auto-registered with a `click` method) |
| `data-id="MenuHeader:<menuId>"` | a side-menu group header (collapses/expands its options — no handle, the agent reads the open state from `aria-expanded`) |
| `data-value="…"` | current value of the component (always present on inputs/selects, even when the value is also rendered visibly — keeps the agent's read path uniform) |
| `data-label="…"` | label or placeholder for input-style components (`Input`, `SearchSelect`, `DateInput`) — `label || placeholder || ""`. Saves the agent walking to the inner `<label>` / `placeholder`. |
| `data-type="text\|number\|other"` | input kind for input-style components: `"number"` for numeric inputs, `"text"` for free-text inputs, `"other"` for everything else (selects, dates, colors, …). |
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
| `SearchSelect` | `SearchSelect` | root has `data-value="[id] text"` (id of the selected option followed by its display text; empty string when nothing is selected) | `search`, `select`, `getOptions` (options not in DOM until opened) |
| `SearchCard` | `SearchCard` | wraps a child `SearchSelect`; renders selected chips as `Option:<id>` | `select(...ids)`, `remove(id)` |
| `SearchDualCard` | `SearchDualCard` | wraps two child `SearchSelect`s; chips as `Option:<id>` | `select(...ids)`, `remove(id)` |
| `Checkbox` | `Checkbox` | root carries `data-selected="true"` when checked | `click` |
| `CheckboxOptions` | `CheckboxOptions` | each option as `Option:<id>` with `data-selected="true"` when chosen | `select(...ids)`, `remove(id)` |
| `OptionsStrip` | `OptionsStrip` | each button as `Option:<id>` with `data-selected="true"` on the active one | `select(id)` |
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
| `vTable/VTable` | `Table` | each selectable row as `TableRow:<id>`, with `data-selected="true"` on the active one | `select(...ids)` |
| `vTable/TableGrid` | `Table` | same as `VTable` | `select(...ids)` |
| `vTable/TableTree` | `Table` | same | `select(...ids)` |
| `vTable/TableStream` | `Table` | same | `select(...ids)` |
| `vTable/MobileCardsVirtualList` | `Table` | same | `select(...ids)` |
| `vTable/CellEditable` | (no own handle) | rendered inside its parent `Table`. Root has `data-id="<tableID>:<cellID>"`, `data-cell-type="CellEditable"`, `data-value`, `data-label`, `data-type`. Methods are reached on the parent `Table` handle. | `setValueChild` (via Table) |
| `vTable/CellSelector` | (no own handle) | rendered inside its parent `Table`. Root has `data-id="<tableID>:<cellID>"`, `data-cell-type="CellSelector"`, `data-value="[id] text"`. | `searchChild`, `select`, `getOptionsChild` (via Table) |

### Table-level cell routing

`Table` is the only registered handle for the table; cells (`CellEditable`,
`CellSelector`) live underneath as `<tableID>:<cellID>` composite ids. The
table exposes distinct method names for cell-routed calls (`*Child`) so the
caller never has to guess whether a verb is acting on the table or on one of
its cells. `select` is the one shared verb because the id alone disambiguates
rows (multiples of 100) from cells (anything else):

| Call | Routing |
|---|---|
| `select(rowID, …)` | every arg is a row id (`rowIndex * 100`); each one selects/toggles its row. |
| `select(cellID, …optionIds)` | first arg is a cell id; remaining args go to that cell's `select`. |
| `setValueChild(cellID, value)` | forwarded to the cell's `setValue`. |
| `searchChild(cellID, text)` | forwarded to the cell's `search`. |
| `getOptionsChild(cellID, max?)` | forwarded to the cell's `getOptions`. |

Row ids are `rowIndex * 100`; cell ids are `rowID + columnIndex + 1` (column
position is 1-based so cells never collide with rows). Cap of ~99 cells per
row.
| `Renderer` | `Renderer` | each rendered button as `Button:<id>` | — (buttons register their own `click`) |
| `domain-components/SideMenu` | `MenuOption` (per leaf option) | menu group toggles as `MenuHeader:<menuId>` markers; each option as its own `MenuOption:<id>` registered handle with `data-label="<option name>"` and `data-value="<route>"` | `click` |

`TableRow` is markup only — it doesn't register a handle. All row interaction
goes through the parent `Table`'s `select(...ids)`. All five table variants
collapse to the single `type: "Table"` so the agent doesn't need to know the
difference.
