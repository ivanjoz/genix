# TableGrid Plan (Approved)

## Goal
Create a new component, **simpler and faster than `VTable`**, optimized for large datasets with:
- Fixed row height
- Virtualized rendering with `@humanspeak/svelte-virtual-list`
- No `<table>` (use CSS Grid)
- Mostly fixed column widths, with optional flexible columns

## Final Name
- `TableGrid`

## Why Separate Component (not modifying VTable)
- `VTable` includes complex features (multi-level headers, cell editors/selectors, mobile cards, mixed render paths).
- A dedicated lean component avoids feature-branching and keeps render logic minimal.
- Lower branching + fewer dynamic behaviors should improve runtime performance and maintainability.

## Functional Scope (V1)
Include:
- Single header row
- Fixed row height (single `rowHeight` prop)
- Virtual list for rows
- Grid-based rows and header (shared `grid-template-columns`)
- Column definitions with `width` support:
  - fixed pixel widths (`"120px"`)
  - flexible tracks (`"minmax(160px,1fr)"`)
- Optional row click and selected row styling
- Optional empty-state message
- Optional snippet-based cell renderer

Exclude (V1):
- Grouped/sub-headers
- Inline cell editing/selectors
- Mobile card mode
- Complex filter cache logic
- Row auto-height

## File Structure
- `frontend/ui-components/vTable/TableGrid.svelte`
- `frontend/ui-components/vTable/tableGridTypes.ts`
- `frontend/routes/develop-ui/test-table-grid/+page.svelte` (performance playground)
- `frontend/docs/UI_COMPONENTS.md` (add usage section)

## Types (Approved)
```ts
// Goal: explicit names + minimal API surface for predictable rendering.
export interface TableGridColumn<TRecord> {
  id: string | number;
  header: string;
  width: string; // e.g. "120px" | "180px" | "minmax(160px,1fr)"
  align?: 'left' | 'center' | 'right';
  headerCss?: string;
  cellCss?: string;
  getValue?: (record: TRecord, rowIndex: number) => string | number;
}
```

## Props (Approved)
- `columns: TableGridColumn<T>[]`
- `data: T[]`
- `height?: string` (default fixed container height, e.g. `"480px"`)
- `rowHeight?: number` (default `36`)
- `bufferSize?: number` (default `12`)
- `css?: string`
- `headerCss?: string`
- `rowCss?: string`
- `emptyMessage?: string`
- `onRowClick?: (record: T, index: number) => void`
- `selectedRowId?: string | number`
- `selectedRecord?: T`
- `getRowId?: (record: T, index: number) => string | number`
- `cellRenderer?: Snippet<[record, column, defaultValue, rowIndex]>`

## Rendering Model
- Outer scroll container with fixed height (`height`).
- Sticky header row at top using `position: sticky`.
- Body rows rendered through `SvelteVirtualList`.
- Each row is `display: grid; grid-template-columns: <computedTemplate>; height: <rowHeight>px;`.
- Text overflow controlled with ellipsis for predictable row height.

## Performance Guidelines
- Avoid per-cell complex branching.
- Keep row DOM shape constant.
- Memoize `gridTemplateColumns` derived from columns.
- Use stable row identity via `getRowId` when provided.
- Keep default render path text-only (`getValue`) and make `renderCell` optional.

## Implementation Steps
1. Create `types.ts` with `TableGridColumn<TRecord>` and snippet renderer type.
2. Implement `TableGrid.svelte` with fixed-height virtualized grid rows.
3. Add concise debug logs behind a local `debug` flag (creation, row-click, rendered item count).
4. Create `test-table-grid/+page.svelte` with 10k+ rows and mixed fixed/flex columns.
5. Validate behavior:
   - scrolling smoothness
   - sticky header alignment with body columns
   - selection state
   - empty state
6. Document usage in `UI_COMPONENTS.md`.

## Acceptance Criteria
- Renders 10k rows with smooth scroll on desktop.
- Header and rows remain perfectly aligned while scrolling.
- No `<table>` element used.
- Row height remains fixed and consistent.
- API is smaller than `VTable` and easier to reason about.
- Desktop-only behavior (no mobile card mode in V1).
- Horizontal scrolling allowed when widths exceed viewport.

## Risks & Mitigations
- Risk: header/body misalignment with scrollbar width.
  - Mitigation: reserve scrollbar gutter or add right padding compensation.
- Risk: long text breaking fixed row height.
  - Mitigation: enforce `white-space: nowrap; overflow: hidden; text-overflow: ellipsis`.
- Risk: renderer returning heavy content.
  - Mitigation: document `renderCell` as lightweight and optional.

## Confirmed Decisions
1. Final component name: `TableGrid`.
2. Include snippet rendering support in V1.
3. Column widths are static in V1 (no resizing).
4. Selection supports both `selectedRowId` and `selectedRecord`.
5. Horizontal scrolling is enabled when needed.
6. V1 is desktop-only.
