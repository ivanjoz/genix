# DateInput Mobile Layer Plan

## Goal

Make `frontend/ui-components/DateInput.svelte` behave like `frontend/ui-components/SearchSelect.svelte` on mobile:

- Do not render the calendar inline inside the form card on mobile.
- Open a full-width top layer with dark translucent background.
- Show a month selector made of 3 rows of buttons.
- Show the day calendar below the month selector.
- Keep the existing desktop inline calendar behavior unchanged.

## Current Behavior Analysis

### How `SearchSelect.svelte` solves mobile

`SearchSelect.svelte` avoids clipping on mobile because it does **not** open its dropdown inline when `Core.deviceType === 3`.

The current flow is:

1. `SearchSelect.svelte` derives `isMobile` from `Core.deviceType === 3`.
2. It switches to `useLayerPicker` on mobile.
3. Instead of focusing a real text input, it renders a button-like container.
4. On click, it writes a payload into `Core.showMobileSearchLayer`.
5. `frontend/ui-components/TopLayerSelector.svelte` reacts to that global state and renders a fixed overlay with:
   - black translucent background
   - top search field
   - scrollable results grid
   - close button
6. Selection is committed through the callback stored in the global payload, then the layer is closed.

This matters because the overlay is rendered outside the local form layout, so it is not clipped by the filter card/container.

### Why `DateInput.svelte` fails on mobile

`DateInput.svelte` still renders the calendar as an inline absolutely-positioned popup:

- `.date-picker-c` uses `position: absolute; top: 100%; left: 0;`
- the popup lives inside the local `.date-input-container`
- on mobile, the parent filter card constrains the visible area

So the calendar is technically opening, but it is opening in the wrong rendering layer for small screens.

## Proposed Approach

### 1. Keep desktop as-is

Do not touch the current desktop interaction beyond shared helper extraction if needed.

Desktop should continue using:

- inline input
- inline popup calendar
- existing blur/focus logic

### 2. Add a dedicated global mobile date layer

Follow the same architecture used by `SearchSelect`, but for date picking.

Minimal-risk approach:

- add a new global state entry in `frontend/core/store.svelte.ts`
- create a dedicated mobile date layer component
- mount that component once in the shared UI shell, alongside `TopLayerSelector.svelte`

Suggested state shape:

```ts
interface ITopDateLayer {
  selectedUnixDay: number
  focusedUnixDay: number
  selectedMonth: number
  label?: string
  onSelect: (unixDay: number) => void
  onClose?: () => void
}
```

This keeps the search layer and date layer independent, which is simpler than forcing both into a single generic overlay contract.

### 3. Add a mobile-only mode inside `DateInput.svelte`

In `DateInput.svelte`:

- derive `isMobile` from `Core.deviceType === 3`
- on mobile, render a button-like field instead of opening the inline popup
- clicking that field should populate the new global date-layer state

This should mirror the `SearchSelect` pattern:

- desktop: local popup
- mobile: global top layer

### 4. Build the new mobile date layer UI

Create a component such as:

- `frontend/ui-components/TopLayerDatePicker.svelte`

Visual structure:

1. dark translucent background with blur, matching `TopLayerSelector`
2. top bar with:
   - current selected date or label
   - close button
3. year/month navigation row
4. month grid with 12 month buttons in 3 rows
5. calendar grid below for the active month

Recommended month grid:

- 4 columns x 3 rows
- selected month visually highlighted
- tapping a month changes the visible calendar below without closing the layer

### 5. Reuse the date math already inside `DateInput.svelte`

The current component already contains the needed date logic:

- `parseMonth`
- `startOfISOWeek`
- `getISOWeek`
- `getISOWeekYear`
- `semanasDias`
- `makeFechaFormat`

To avoid duplication, move the pure date helpers into a small shared file if extraction stays small and clean.

Suggested file:

- `frontend/ui-components/date-input.helpers.ts`

Only extract pure logic. Keep write/update behavior inside each component.

### 6. Preserve current save semantics

When the user selects a day in the mobile layer:

- call the same date save path used by `DateInput.svelte`
- update `saveOn[save]`
- update visible formatted text
- call `onChange`
- close the mobile layer

This must preserve the current UnixDay contract already used by the component.

## Implementation Steps

1. Add a new mobile date layer state contract to `frontend/core/store.svelte.ts`.
2. Create `frontend/ui-components/TopLayerDatePicker.svelte`.
3. Mount `TopLayerDatePicker.svelte` in the same shared place where `TopLayerSelector.svelte` is already rendered.
4. Extract only the reusable pure date helpers from `DateInput.svelte` if duplication becomes noisy.
5. Update `DateInput.svelte` to branch by device type:
   - desktop uses current inline popup
   - mobile opens the global date layer
6. Style the mobile layer with:
   - black translucent background
   - 3-row month selector
   - calendar below
   - explicit selected/focused/today states
7. Verify that date selection still updates the bound model and triggers `onChange`.

## Verification Checklist

- Mobile tap on `DateInput` opens a full-screen/top-layer picker, not an inline popup.
- The layer is not clipped by the report filter card.
- The month selector shows 12 buttons arranged in 3 rows.
- Changing month updates the calendar below without closing the layer.
- Selecting a day saves the UnixDay value correctly.
- Existing desktop behavior remains unchanged.
- Existing keyboard/manual typing behavior on desktop remains unchanged.

## Open Questions

1. Should the mobile layer allow manual typing of the date at the top, or should it be picker-only?
2. For year navigation, do you want:
   - the current `« ‹ month year › »` controls
   - only year arrows plus 12 month buttons
   - or a dedicated year grid as well?
3. When the user taps a month button, should the calendar keep the current selected day if possible, or always reset focus to the first valid day of that month?
4. Do you want the mobile layer to open from the top exactly like `TopLayerSelector`, or can it use a full-screen centered overlay if the styling is otherwise the same?

## Recommended Default Assumptions

If no extra clarification is needed, I would implement with these defaults:

- mobile is picker-only
- top-layer style matches `TopLayerSelector`
- year changes through left/right controls in the header
- month buttons change only the visible month
- day tap saves immediately and closes the layer
- desktop behavior remains unchanged
