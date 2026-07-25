# PLAN — `@genix/ui` Component Showroom Page

## Goal

One dev page that renders every **visual** component of `frontend/packages/genix-ui`
grouped into tabs, with a toolbar switch that flips the page surface between
**white** and **light gray** so contrast/borders can be audited on both.

Decisions already confirmed:

| Decision | Choice |
| --- | --- |
| Backgrounds | Toolbar toggle flipping the whole page surface (white ↔ light gray) |
| Scope | Visual subset — skips components that need a live backend/CDN |
| Location | `frontend/packages/genix-ui/showroom/` — the catalogue ships with the components it documents. The route `frontend/routes/develop-ui/showroom/+page.svelte` is a 13-line shell that renders `<Showroom>`, and is **not** registered in `core/modules.ts` |

Route: `http://localhost:3570/develop-ui/showroom`

---

## Why this shape

- `develop-ui/` already holds ad-hoc test pages (`test-table`, `test-table-grid`,
  `test-cards`), none of them in `core/modules.ts`. The showroom follows that
  precedent, so it never appears in a client's sidebar.
- `Page` already accepts `containerCss`, which lands on `#page-container`
  (`domain-components/Page.svelte:101`). The background toggle is therefore a
  one-class change — no wrapper div, no new CSS.
- The skill recommends max 3 `Page options`; we need 7 groups, so the tab bar is a
  single `OptionsStrip` in the page body instead of the header tab menu.
- Each tab is its own `.svelte` file rendered under `{#if}`, so only the active tab
  mounts. That keeps the heavy demos (15k-row tables, canvas charts, RoosterJS
  editor) out of the tree until selected.
- Demo blocks are **transparent** (dashed border only). A white card would hide the
  gray surface and defeat the toggle.

---

## Files

All new, under `frontend/routes/develop-ui/showroom/`:

| File | ~Lines | Contents |
| --- | --- | --- |
| `+page.svelte` | 90 | `Page` shell, background toggle, tab `OptionsStrip`, `{#if}` per section |
| `ShowroomBlock.svelte` | 25 | Shared titled wrapper: name, one-line note, transparent body, dashed border |
| `showroom-data.ts` | 45 | Typed dummy datasets + shared `ITableColumn` definitions |
| `SectionForm.svelte` | 130 | `form/` controls |
| `SectionButtons.svelte` | 120 | `buttons/` + `cards/` |
| `SectionNavigation.svelte` | 70 | `navigation/` + `KeyValueStrip` |
| `SectionTables.svelte` | 180 | `vTable/` |
| `SectionOverlays.svelte` | 140 | `layers/` + `Popover` |
| `SectionCharts.svelte` | 80 | `charts/` + `SquareBarSized` |
| `SectionMisc.svelte` | 130 | `misc/` primitives, `FileUploadSelector`, `HTMLEditor` |

No changes to existing files. `core/modules.ts` is **not** touched.

`showroom-data.ts` reuses `usersDemo` from the sibling `routes/develop-ui/dummy-data.ts`
(name / language / id / bio / version) as the record source for selects, tables and card
lists, so no new fixture data is invented beyond chart series and tree nodes.

---

## `+page.svelte` structure

```svelte
<script lang="ts">
  import Page from '$domain/Page.svelte'
  import OptionsStrip from '$components/navigation/OptionsStrip.svelte'
  // …section imports

  // 1 = white surface, 2 = light gray surface. Drives Page.containerCss only.
  let surface = $state(1)
  let tab = $state(1)

  const tabs: [number, string][] = [
    [1, 'Form'], [2, 'Buttons & Cards'], [3, 'Navigation'], [4, 'Tables'],
    [5, 'Overlays'], [6, 'Charts'], [7, 'Misc'],
  ]
</script>

<Page title="UI Showroom" containerCss={surface === 2 ? 'bg-gray-100' : 'bg-white'}>
  <div class="flex items-center gap-12 mb-12">
    <OptionsStrip selected={tab} options={tabs} useMobileGrid
      onSelect={(e) => { tab = e[0] as number }} />
    <OptionsStrip css="ml-auto" selected={surface}
      options={[[1, 'White'], [2, 'Gray']]}
      onSelect={(e) => { surface = e[0] as number }} />
  </div>

  {#if tab === 1}<SectionForm />{/if}
  <!-- … one {#if} per section -->
</Page>
```

Two `OptionsStrip`s rather than a raw `<button>` toggle, so both controls stay
agent-visible per the `AGENTIC_COMPONENTS` contract. Gray tone is Tailwind
`bg-gray-100` (`#f3f4f6`).

---

## Tab contents

### 1. Form
`Input` (text, number with `baseDecimals`, password, `useTextArea`, `required` +
`validator`, `disabled`), `SearchSelect` (plain, `icon`, `max`), `Checkbox`
(`useNumber`), `CheckboxOptions` (`single`, `multiple`, `useButtons`), `DateInput`
(`type="unix"` with `usePopover`, `type="sunix"` inline), `FilterInput`,
`ColorPicker`, `LabelText`, `LoginForm`.

All bound to one local `$state` form object via `saveOn` + `save`, with a live
`<pre>{JSON.stringify(form)}</pre>` readout at the bottom so writes are visible.

### 2. Buttons & Cards
`Button` × 6 colors, plus `useCircle`, `icon`-only, `hideNameOnMobile`, `disabled`.
`InlineButton` (`default` / `checked`), `ButtonLayer` (bindable `isOpen` panel),
`ButtonList` (3 action items), `Card` with `onClick`, `SearchCard` (multi-select),
`SearchDualCard` (two linked searches + `selectedItem` snippet).

### 3. Navigation
`OptionsStrip`: plain, `useMobileGrid`, and the `[id, label, [line1, line2]]`
two-line mobile form. `ArrowSteps` with 4 steps and `columnsTemplate`.
`KeyValueStrip` with 5 pairs and a `getContent` formatter.

### 4. Tables
Shared `ITableColumn<IShowroomUser>[]` from `showroom-data.ts` (`getValue`, `render`,
`width`, `align`, `cardColumn`).

- `TableGrid` — 5 000 rows, `getRowId`, `selectedRowId`, `onRowClick`
- `VTable` — bound to a `FilterInput` through `filterText` + `getFilterContent`
- `TableStream` — `maxRecords={30}`, rows appended on an interval (cleared in `onDestroy`)
- `TableTree` — 3 parents × children, `onNodeClick` / `onChildClick`
- `CardsList` — `cells` + `buttonDeleteHandler`
- `CellInput` / `CellSelect` — demonstrated as `cellInputType` / `cellOptions`
  columns inside the `TableGrid` instance, not standalone

### 5. Overlays
`Layer type="side"` (id `1`, `sideLayerSize={700}`, `options` sub-tabs, `onSave`/
`onDelete`/`onClose`), the list wrapped in `Layer type="content"` to show the shrink
behaviour, `LayerStatic` (desktop column / mobile drawer via
`useMobileLayerVertical`), `Modal` (id `11`, `size={9}`), `MobileLayerVertical`,
and `Popover` anchored to a `Button` (imports `$components/misc/popover.css` for the
default bubble skin).

Opened with `ui.openSideLayer(1)` / `ui.openModal(11)` from `useUI()`.

### 6. Charts
`ChartCanvas` (2 series, `dateLabels`, `dateLabelEvery`, `height={120}`),
`CellSimpleChart` (`barColors` + `colorScale` variants), `CellHorizontalBars`
(`values: [total, pending][]`, `logScaleFactor`), `SquareBarSized` row at 4 sizes.

Charts are canvas-rendered with fixed colors — this is the tab where the gray
toggle matters most, so each chart block also gets a one-line note about its
hardcoded background assumption if one shows up.

### 7. Misc
`T` (`"EN|ES"` resolution), `HighlightText`, `LoadingBar`, `Virtualizer`
(1 000 items, `children` snippet), `VirtualCards` (`maxColumns={3}`), `Renderer`
(small `ElementAST` tree), `Portal`, `FileUploadSelector` (local pick only, no
upload), `HTMLEditor` (`saveOn`/`save`, no backend).

---

## Deliberately excluded

| Component | Reason |
| --- | --- |
| `Image`, `Imagehash`, `ImageUploader` | Need CDN routes + a real upload API; would render broken/error states |
| `RecordByIDText` | Needs `ui.resolveRecord` against a live API route |
| `SideMenu`, `MobileMenu` | Already rendered by the app shell around this page |
| `TopLayerSelector`, `TopLayerDatePicker` | Mount-once singletons in the root layout; exercised implicitly by mobile `SearchSelect` / `DateInput` |
| `MobileCardsVirtualList` | Internal to the table components |
| `UiProvider` | The route layout already provides the runtime |
| Excel builder | Not a component |

Each exclusion gets a one-line comment in the relevant section file so the gap is
explicit rather than looking like an oversight.

---

## Conventions applied

- `--spacing: 1px` — `gap-12`, `mb-12`, `w-200` are pixels.
- No raw `<button>` / `<div onclick>`; every trigger is `Button` or `Card`.
- No `font-size` / `font-weight` in CSS — Tailwind classes only.
- Bilingual labels use the `"English|Español"` form so `translate` is exercised.
- `saveOn` + `save` everywhere, never `bind:value`.
- Concise comments on each block explaining what the demo is proving.

---

## Verification

1. `cd frontend && bun run check` → no new errors.
2. `bun run dev:main`, log in, open `/develop-ui/showroom`.
3. Walk all 7 tabs on both surfaces; confirm no component relies on a white
   background (white text/borders vanishing on gray, or vice versa).
4. Narrow the window below 750 px and re-check the mobile paths: `OptionsStrip`
   grid, table mobile cards, `LayerStatic` drawer, mobile `DateInput` /
   `SearchSelect` full-screen pickers.

---

## Open risks

- **`Layer type="side"` + `LayerStatic` on one page.** Both claim horizontal space.
  If they fight, `LayerStatic` moves into its own sub-view inside the Overlays tab
  rather than sharing the row.
- **`TableStream` interval.** Must be cleared in `onDestroy`, otherwise switching
  tabs leaves a timer appending rows to a dead component.
- **`HTMLEditor` (RoosterJS).** Heaviest import on the page. If it slows the Misc
  tab noticeably it gets gated behind a "Load editor" `Button`.
