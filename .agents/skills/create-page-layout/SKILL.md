---
name: create-page-layout
description: Build a new Svelte page route with the project's `Page` shell, top-tab sections, `OptionsStrip` sub-views, side `Layer` detail panels, `LayerStatic` permanent side panels, and `Modal` dialogs. Use when scaffolding any `frontend/routes/<module>/<feature>/+page.svelte`, dividing a page into views, or adding a side panel.
version: 0.1.0
---

# Create a Page (Layout & Navigation)

Pages live at `frontend/routes/<module>/<feature>/+page.svelte`. Large sub-forms go in sibling `.svelte` files in the same folder; the service goes in `<feature>.svelte.ts` (see `delta-cache-api` / `fetch-record-by-id-api` skills — this skill only **instantiates** services).

## Register the page in the sidebar menu (REQUIRED)

A page is invisible until added to `frontend/core/modules.ts`. Append one entry to the right section's `options[]`:

```ts
{ name: "Suministros", route: "/logistica/supplies-materials", icon: "icon-cube" }
```

`name` = Spanish UI label (may differ from folder slug). `route` must match the `frontend/routes/...` folder exactly. Ask the user for the label/section if not obvious.


## Always use the project components (agent contract)

The automation agent reads the page through a sanitized HTML snapshot plus a registry of component handles (see `frontend/ui-components/AGENTIC_COMPONENTS.md`). Bare `<button>`, `<div onclick>`, or hand-rolled clickable elements are **invisible to the agent** — they have no `data-id`, no registered handle, and no `click` method. Always reach for the project components:

- **Buttons** → `$components/buttons/Button.svelte`. Never write a raw `<button>` for a command action.
- **Clickable cards / tiles** → `$components/cards/Card.svelte` with `onClick`. Avoid put `onclick` on a raw `<div>`.
- **All other interactive surfaces** (`Input`, `SearchSelect`, `Checkbox`, `DateInput`, `OptionsStrip`, `Layer`, `Modal`, `VTable`, `ImageUploader`, …) come from `frontend/ui-components/*`. See `AGENTIC_COMPONENTS.md` for the full spec.

Rule of thumb: if you're about to write a tag with an `onclick`, `class="cursor-pointer"`, or your own `role="button"`, **stop and use `Button` or `Card` instead.** The agent (and accessibility) depend on it.

## Picking the right container

| Need | Use | Open with |
|---|---|---|
| Top-level page sections that share a URL | `Page options` | persisted automatically |
| Sub-tabs within one page body | `OptionsStrip` | local `$state` |
| Edit one record, slide-in from right | `Layer type="side"` | `Core.openSideLayer(id)` |
| Always-visible companion column | `LayerStatic` | — |
| Center dialog (confirm / import / short form) | `Modal` | `Core.openModal(id)` |
| Shrink list when side layer opens | wrap list in `Layer type="content"` | — |
| Sub-tabs inside a `Layer` | `Layer options` + `bind:selected` | — |

## `Page` shell + top-tab menu

`Page` wraps every authenticated route. Passing `options` renders the top-tab menu (e.g. "Ventas / Configuración") and writes the active id to `Core.pageOptionSelected` — read it to branch. Max 3 options recommended. Selection is persisted per route in `localStorage`.

```svelte
<Page title="Ventas"
  options={[{ id: 1, name: "Ventas" }, { id: 2, name: "Configuración" }]}
>
  {#if Core.pageOptionSelected === 1}<!-- main -->{/if}
  {#if Core.pageOptionSelected === 2}<!-- settings -->{/if}
</Page>
```

Omit `options` for a single-view page.

## Service instantiation

Construct at the top of `<script>` once; bind reactive fields directly — don't copy them into local `$state`. The bool/array arg controls whether the constructor calls `this.fetch()`.

```ts
const productos = new ProductosService(true)            // `true` → fetch on construct
const listas    = new ListasCompartidasService([1, 2], true)
```

## Sub-views with `OptionsStrip`

Use for in-page sub-tabs (e.g. `Productos / Categorías / Marcas` on `/negocio/productos`) when the sections share a toolbar and scope. Use `Page options` instead when the sections are effectively unrelated screens.

```svelte
<script>
  let view = $state(1)
</script>

<OptionsStrip selected={view}
  options={[
    [1, "Productos"],
    [2, "Categorías"],
    [3, "Marcas", ["Mar-", "cas"]],   // long label → mobile two-line form
  ]}
  useMobileGrid={true}
  onSelect={(e) => { view = e[0] as number }}
/>
```

Option formats: `[id, label]`, `[id, label, [mobileLine1, mobileLine2]]`, or `{ id, name }` + `keyId="id" keyName="name"`. `useMobileGrid` makes tabs share width evenly on mobile.

## Side detail panel — `Layer type="side"`

The primary "edit one record" UI: slides in from the right with automatic Save/Delete/Close buttons.

```svelte
<script>
  let form      = $state({} as IFoo)
  let layerView = $state(1)
</script>

<!-- Open from a toolbar button or row click -->
<Button color="green" icon="icon-plus"
  onClick={() => { form = { ss: 1 } as IFoo; Core.openSideLayer(1) }} />

<Layer type="side" id={1} sideLayerSize={780}
  title={form?.Nombre || "Nuevo"}
  titleCss="h2 mb-6"
  bind:selected={layerView}
  options={[[1, "Información"], [2, "Ficha"], [3, "Fotos"]]}
  onSave={onSave}
  onDelete={onDelete}
  onClose={() => { form = {} as IFoo }}
>
  {#if layerView === 1}<InfoForm bind:form />{/if}
  {#if layerView === 2}<FichaForm bind:form />{/if}
</Layer>
```

Rules:
- `id` is the numeric handle used with `Core.openSideLayer(id)`. Use small distinct numbers per page (`1`, `2`, …); `0` closes whatever is open (`Core.hideSideLayer()` is equivalent).
- Buttons render automatically: `onSave` → blue Save, `onDelete` → red Trash, Close is always shown. **Don't add your own Save button in the body.** Customize with `saveButtonName` / `saveButtonIcon`.
- `options` + `bind:selected` give the layer an internal sub-tab bar (same option formats as `OptionsStrip`).
- `sideLayerSize` is desktop width in px (default 800). Mobile is always full-viewport.
- Reset form state in `onClose`, not after save — otherwise the layer flashes empty during the close animation.

### `Layer type="content"`

Wrap the main list in `<Layer type="content">…</Layer>`. It subscribes to the open side layer's width and shrinks itself so the list and the side layer never overlap.

## Permanent side panel — `LayerStatic`

Not a modal. A fixed side column always visible on desktop; off-canvas drawer on mobile. Use for split layouts where the right side is a live working surface (cart, totals — see `/comercial/sale_order_create`).

```svelte
<div class="flex h-full gap-20">
  <div class="flex-1 flex flex-col min-w-0"><!-- main content --></div>

  <LayerStatic
    css="w-[40%] min-w-350 bg-white border-l h-[calc(100vh-var(--header-height))]"
    mobileLayerTitle="Detalle de Venta"
    useMobileLayerVertical={124}
  >
    <!-- cart, totals, items -->
  </LayerStatic>
</div>
```

- `css` — your responsibility for sizing.
- `useMobileLayerVertical={N}` — bottom-anchored drawer with `N`-px peek when collapsed; omit for a side drawer.

**Side `Layer` vs `LayerStatic`:** side `Layer` is transient (one open at a time, with Save/Delete/Close). `LayerStatic` is a permanent companion column that's part of the page's primary workflow.

## Modal

Centered dialog for confirm/import/short forms. `Core.openModal(id)` / `Core.closeModal(id)`. Size `1` (~600px) to `9` (~1000px).

```svelte
<Modal id={11} title="Importar Productos" size={9}
  onSave={() => doImport()} saveButtonLabel="Importar" saveIcon="icon-upload"
>
  <!-- body -->
</Modal>
```

For heavy mobile-first forms, prefer a side `Layer`.

## Typical list + side editor page

```svelte
<script lang="ts">
import Page         from '$domain/Page.svelte'
import Layer        from '$components/layers/Layer.svelte'
import VTable       from '$components/vTable/VTable.svelte'
import Button       from '$components/buttons/Button.svelte'
import FilterInput  from '$components/form/FilterInput.svelte'
import Input        from '$components/form/Input.svelte'
import { Core }     from '$core/store.svelte'
import { Loading, Notify } from '$libs/helpers'
import { FooService, type IFoo } from './foo.svelte'

  const foo = new FooService(true)

  let filterText = $state("")
  let form       = $state({} as IFoo)
  let layerView  = $state(1)

  const onSave = async () => {
    if ((form.Nombre || "").length < 3) { Notify.failure("Nombre muy corto"); return }
    Loading.standard("Guardando…")
    await foo.postAndSync([form])
    Loading.remove()
    Core.openSideLayer(0)
  }
</script>

<Page title="Foos">
  <div class="flex items-center mb-8">
    <FilterInput css="w-200" icon="icon-search" bind:value={filterText} />
    <Button name="Nuevo" color="green" icon="icon-plus" css="ml-auto"
      onClick={() => { form = { ss: 1 } as IFoo; Core.openSideLayer(1) }} />
  </div>

  <Layer type="content">
    <VTable data={foo.records} columns={columns}
      {filterText} getFilterContent={(e) => e.Nombre}
      selected={form?.ID} isSelected={(e, id) => e.ID === id}
      onRowClick={(e) => { form = { ...e }; Core.openSideLayer(1) }}
    />
  </Layer>

  <Layer type="side" id={1} sideLayerSize={680}
    title={form?.Nombre || "Nuevo"}
    titleCss="h2 mb-6"
    bind:selected={layerView}
    options={[[1, "Información"], [2, "Detalle"]]}
    onSave={onSave}
    onClose={() => { form = {} as IFoo }}
  >
    {#if layerView === 1}
      <div class="grid grid-cols-24 gap-10 mt-12">
        <Input label="Nombre" saveOn={form} save="Nombre" required
          css="col-span-24 md:col-span-12" />
      </div>
    {/if}
  </Layer>
</Page>
```

Forms use a 24-column grid (`grid-cols-24`, `gap-10`). Filtering is `VTable.filterText` + `getFilterContent` — don't reimplement. Loading/toasts: `Loading.standard(msg)` → work → `Loading.remove()`; `Notify.success` / `Notify.failure`.
