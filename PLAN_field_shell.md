# PLAN — `FieldShell`: one base component for every labelled field

## Goal

1. Make Input2's look **the** look: notched masked border, real `<label for>`, error
   only after first blur.
2. Extract that chrome into a single base component so `Input`, `SearchSelect`,
   `DateInput`, `ColorPicker`, `FilterInput` (and anything added later) stop
   re-implementing "label + box + border notch + focus ring".

## Status: executed

All 8 steps done. `svelte-check`: **0 errors and 0 warnings** in every touched file (the 12
errors it still reports are pre-existing, in 7 unrelated route files). Vite compiles all six
components. **Not yet verified visually** — see "Left for you" at the bottom.

Deviations from the plan as written, and why:

| Deviation | Reason |
| --- | --- |
| `ColorPicker` uses the **standard centred row**, not `autoHeight` | The swatch is 28px and fits the 38px row, which keeps the picker on the same 51px footprint as every other field. `autoHeight` would have made it 59px. Its `margin-bottom: 4px` — there to fight the old off-centre row — is gone too. |
| The showroom's Input2 block was **deleted**, not repointed | With `Input` == Input2 the block was a duplicate of the one above it. `form2` and the import went with it. |
| Also cleaned 4 call sites in `WarehouseLayoutEditor.svelte` | They passed `css="shadow-small bg-solid no-border …"`, which targeted `.input_div` and became inert dead strings. Those unlabelled fields now use the shell's standard no-label box. |
| Also deleted `ColorPicker`'s `contentClass` prop | Declared, never read, passed by nobody. |
| `components.module.css` lost **288 → 70 lines** | Two dead regions, not one: the chrome (1-195) *and* `input_post_value` + the `.input_div` modifier flags (266-289). |

## Decisions (confirmed)

| Decision | Choice |
| --- | --- |
| Green ✓ on valid required fields | **Keep it.** Lives in the `suffix` slot, not inside the label, so the notch never resizes. Shown as soon as the field is valid; the red ⚠ still waits for first blur (a check is reassurance, an error is an accusation). |
| Scope | **All five consumers in one pass**, including `FilterInput`. |

---

## Why this is the right shape (findings)

The chrome is already duplicated four times, verbatim:

| File | Chrome elements it repeats |
| --- | --- |
| `form/Input.svelte:221-231` | `input_lab_cell_left`, `input_lab`, `input_lab_cell_right`, `input_shadow_layer`, `input_div`, `input_div_1` |
| `form/SearchSelect.svelte:412-421` | same six |
| `form/DateInput.svelte:351-360` | same six |
| `form/ColorPicker.svelte:70-82` | same six |

That duplication is exactly why the seams differ between components: six elements,
each with its own radius/shadow/offset, rasterised independently. Input2 replaced all
six with **two** (`.box` + its masked `::before`), because the border and the focus
ring became one masked layer — so there is nothing left worth duplicating.

Supporting facts that make the migration cheap:

- **`frontend/domain-components/core.module.css` is dead** — a byte-for-byte older copy
  of the same chrome with **zero importers** (`rg "core\.module"` → no hits). Delete it.
- **The style variants have one call site each**, so they are trivial to carry over:
  - `SearchSelect useStyle={1}` → `sales/sale_order_create/+page.svelte:452` (rounded
    grey search pill)
  - `SearchSelect noStyle` → `webpage-builder/components/EditorTab.svelte:80`
  - `DateInput useInlineStyle` → `logistics/products-stock/PurchaseOrderEntry.svelte:520`
    (bare input filling a table cell)
- **`Input`'s `content` prop is passed by nobody** (`rg '<Input[^>]*content='` → no
  hits). Input2 already dropped it, along with the dead `makeValue()`.
- Focus/hover/invalid are all `:focus-within` / class-driven in `input2.module.css`, so
  they work for *any* control the shell wraps — including `SearchSelect`'s
  `role="button" tabindex="0"` mobile trigger and `ColorPicker`'s widget. No JS focus
  plumbing needed.

Call-site pressure: `Input` 32 files, `SearchSelect` 29, `DateInput` 10,
`ColorPicker` 3. **The public props of all four stay unchanged**, so no call site is
touched in this refactor.

`FilterInput` (27 call sites) is included by your call. It is the one consumer that is
not a labelled field — an unlabelled search box with a leading icon and its own
`border-gray-300` + blue focus ring. Two consequences, both handled in step 6:

- it needs a **`prefix`** snippet on the shell (leading icon), which no other consumer
  needs — a small, genuinely reusable addition;
- its `label` prop is **aria-only today** (`FilterInput.svelte:73`) and 6 call sites pass
  it (e.g. `business/products/+page.svelte:556`). It must keep going to `aria-label` and
  must **not** be forwarded as the shell's visual `label`, or those six toolbars sprout a
  notch label they never had.

Still out of scope: `Checkbox`, `CheckboxOptions`, `LabelText` — no box to share.

---

## The base component

New: `frontend/packages/genix-ui/form/FieldShell.svelte`
Renamed: `form/input2.module.css` → `form/field-shell.module.css` (Input2 goes away, so
the `input2` name stops making sense).

```svelte
interface IFieldShell {
  label?: string
  invalid?: boolean          // caller owns the WHEN (Input gates on first blur)
  disabled?: boolean
  css?: string               // extra classes on the root
  variant?: 'field' | 'bare' | 'pill'
  autoHeight?: boolean       // textarea / tall content: row grows instead of centring
  children: Snippet<[{ controlId: string, controlClass: string }]>
  prefix?: Snippet           // left adornment: FilterInput's filter/search icon
  suffix?: Snippet           // right adornments: unit text, ✓/⚠, select arrow
  overlay?: Snippet          // dropdown / calendar, absolutely positioned off the root
  ...rest                    // spread onto the root: the data-* agent attributes
}
```

Rendered:

```svelte
<div class={rootClass} style="--notch-w: {notchWidth}px" {...rest}>
  <div class={s.box}></div>                       <!-- fill + shadow; border in ::before -->
  {#if label}
    <label class="{s.lab} text-[15px]" for={controlId} bind:clientWidth={labelWidth}>
      <T text={label} />
    </label>
  {/if}
  <div class="{s.row}{autoHeight ? ' is-textarea' : ''}">
    {#if prefix}<div class={s.prefix}>{@render prefix()}</div>{/if}
    {@render children({ controlId, controlClass: s.inp })}
    {#if suffix}<div class={s.suffix}>{@render suffix()}</div>{/if}
  </div>
  {@render overlay?.()}
</div>
```

Five decisions worth stating, because each one removes a class of bug:

- **`controlId` is minted by the shell and handed to the child**, so `<label for>` is
  correct by construction — a consumer cannot forget to wire it.
- **`controlClass` is handed out too**, so consumers don't import the CSS module just to
  reach `.inp`. The shell stays the only file that knows the class names.
- **`...rest` is spread onto the root**, so the shell needs to know nothing about the
  agent registry, and `SearchSelect`'s extra `data-options-count` keeps working.
- **`overlay` renders inside the root**, which is the `position: relative` ancestor the
  dropdown and calendar already anchor against.
- **`prefix` / `suffix` presence drives the control's padding** via `has-prefix` /
  `has-suffix` on the root (the `has-suffix` rule already exists,
  `input2.module.css:158`). The value can therefore never slide under an adornment, and
  no consumer has to hand-tune `pl-34` the way `FilterInput.svelte:70` does today.

---

## Steps

### 1 — `FieldShell.svelte` + `field-shell.module.css`

Move the CSS, add `.pill` (ported from `components.module.css:58-72`, the
`use-style-1` look) and `.bare` (no box, no label, control fills the parent).

### 2 — `Input.svelte` := Input2 minus the chrome

`Input.svelte` becomes today's `Input2.svelte` with the markup replaced by a
`FieldShell` call. Value pipeline (`onKeyUp` / `doSave` / `transform` /
`baseDecimals` / `persistFieldValue` / `Agent.register`) is already identical in both
files, so it moves across verbatim.

Behaviour deltas vs today's `Input`, all inherited from Input2:

| Delta | Effect |
| --- | --- |
| Green ✓ **kept**, but moved out of the label into `suffix` | Same reassurance as today, except the notch no longer resizes when validity flips (today's `Input.svelte:222` puts the icon inside the label, so the notch grows) |
| ⚠ appears after first blur, not on mount | A pristine form no longer opens covered in red. The ✓ is *not* blur-gated — it can appear immediately |
| `type` defaults to `text`, was `search` | Drops Chrome's native clear-`✕` and search styling |
| Real `<label for>` | Clicking the label focuses the field; screen readers announce it |
| `content` prop and `makeValue()` removed | Dead code (verified unused) |
| `console.log` in the textarea `onblur` removed | `Input.svelte:246` |

Then **delete `Input2.svelte`** and point the showroom's Input2 block at `Input`.

### 3 — `SearchSelect.svelte`

Chrome → `FieldShell`. `iconValid()` and the `select-arrow` move into `suffix`
(so the arrow stops needing `absolute bottom-11 right-8`); the options list moves into
`overlay`. `useStyle={1}` → `variant="pill"`, `noStyle` → `variant="bare"`.
**Verify:** the dropdown's `top` offset — the box's top edge is now at `7px`, and the
footprint stays `51px`, so at most a small constant changes.

### 4 — `DateInput.svelte`

The two near-duplicate branches (`:349-399` labelled, `:400-443` bare) collapse to
**one** `FieldShell` call with `variant={useInlineStyle ? 'bare' : 'field'}`; that is
the single biggest code reduction in this plan. Calendar → `overlay`. `usePopover`
anchors to `inputElement` and is unaffected.

### 5 — `ColorPicker.svelte`

Chrome → `FieldShell` with `autoHeight`. **Verify:** the widget's swatch height
(`ColorPicker.svelte:104`, `calc(var(--input-height) - 16px)`) against the new row.

### 6 — `FilterInput.svelte`

`FieldShell` with `variant="pill"`, the icon in `prefix`, and **no `label` forwarded** —
`label` keeps going to `aria-label` on the input, exactly as today. The debounce,
`commitValue` lowercase/trim, and `Agent.register` are untouched; only the markup and
the `<style>` block go.

This is the one **deliberate visual change** of the refactor: the bespoke
`border-gray-300` + blue `#738dff` focus ring is replaced by the shell's box and purple
focus ring, so the 27 filter boxes finally match the fields beside them. That is the
"every component has the same style" outcome you asked for — but it *is* visible on 27
pages, so it is the first thing to eyeball in step 8. Reverting it costs one extra
`variant`, if you dislike it.

**Verify:** `css` is layout-only at every call site (`w-256`, `mr-16 w-320`, `w-full
md:w-220 …` — 27/27 checked), so it keeps working as a passthrough to the root. But the
shell's `align-self: end` (`input2.module.css:23`) is new for these — check the toolbars
that put a filter next to a button (`business/products/+page.svelte:556`).

### 7 — Delete the dead chrome

- All `.input*` rules from `frontend/packages/genix-ui/components.module.css:1-194`
  (~194 lines) once nothing references them.
- `frontend/domain-components/core.module.css` entirely (no importers).

### 8 — Verify

`npm run check` (or the project's svelte-check script), then walk the showroom Form tab
plus one real page per consumer:

| Page | What it proves |
| --- | --- |
| `business/products/+page.svelte:556` | `FilterInput` in a toolbar — the new look + `align-self: end` |
| a product form | `Input`, incl. `required` ✓/⚠ and the textarea variant |
| `sales/sale_order_create/+page.svelte:452` | `SearchSelect` `useStyle={1}` pill + dropdown placement |
| `logistics/products-stock/PurchaseOrderEntry.svelte:520` | `DateInput` `useInlineStyle` in a table cell — the tightest case |
| `webpage-builder/components/EditorTab.svelte:80` | `SearchSelect noStyle` → `variant="bare"` |

---

## Expected size change

| | Before | After |
| --- | --- | --- |
| Chrome markup | 4 copies × ~10 elements + `FilterInput`'s own | 1 copy × 2 elements |
| Chrome CSS | 194 lines (`components.module.css`) + 130 dead (`core.module.css`) + 196 (`input2.module.css`) + `FilterInput`'s `<style>` | ~210 lines, one file |
| `DateInput` render | 2 near-duplicate branches, 95 lines | 1 branch |

---

## Risks

1. **`FilterInput`'s new look lands on 27 pages** (step 6) — the only intended visual
   change, and the one to check first.
2. **Overlay placement** for `SearchSelect`'s dropdown and `DateInput`'s inline calendar:
   both are positioned against the root, whose box now starts at `7px`. The `51px`
   footprint is unchanged, so expect a constant offset at worst.
3. **`ColorPicker`'s third-party widget** sizes itself off `--input-height`
   (`ColorPicker.svelte:104`); the row it sits in changes height.

No call site changes in steps 1-5 — every public prop keeps its name and meaning.

---

## Left for you

Static verification is complete; the **visual** pass is not. Two headless-Chrome attempts
failed (`--dump-dom` and `--screenshot` both hang or fire before the SPA hydrates, because
the HMR socket keeps the page from ever settling), and the repo has no browser automation
to borrow. So the five pages in step 8 still need a human eye, in this order:

1. `business/products` — the FilterInput look change, on the page where it is most visible.
2. `sales/sale_order_create` — the `useStyle={1}` pill and the dropdown's placement.
3. `logistics/products-stock` → PurchaseOrderEntry — DateInput bare in a table cell.
4. The showroom Form tab — Input's ✓/⚠, the textarea, ColorPicker's swatch centring.
5. `webpage-builder/components` → the `noStyle` SearchSelect.
