# WYSIWYG multi-fragment TextBlockEditor — plan

Status: **IMPLEMENTED — typecheck clean, 6 grouping unit tests pass. Pending manual
WYSIWYG verification in the running builder.**

## Note on `+` inside an `<li>` editor

Because each `<li>` is its own editor (locked Q2) and `+` copies the current line's tag
(Q3), pressing `+` in a list-item editor creates a **new sibling `<li>`, which renders as a
new editor below** rather than a second rule-separated line inside the same editor. For
`<p>`/`<h2>` groups, `+` adds a rule-separated line within the same editor as designed.
Flag if you'd prefer all `<li>`s to share one multi-line editor instead.

## Goal

Turn `TextBlockEditor` from a single-textarea / single-node editor into a small
WYSIWYG block that edits **several text fragments at once**, each fragment with its
own independent styling (font size, color, bg, padding, margin), and reduce the
number of `TextBlockEditor` instances `AstEditor` emits by **grouping adjacent
text nodes** into one editor.

## Vocabulary

- **Fragment (span)** — one inline run of text with its own `css`/`style`. Shown in
  the panel as one small text area. Maps to a `<span>` (or the original leaf tag).
- **Line** — a visual line. Fragments in the same line render contiguous/inline
  (no break). A line maps to a block boundary: a `<br>` or a separate block element
  (`<p>`/`<h2>`/`<li>`).
- **Block (one editor)** — the whole `TextBlockEditor`: an ordered list of lines,
  each line an ordered list of fragments.

## Behaviour (from the request)

1. The editor shows N stacked text areas, one per **fragment**. The toolbar acts on
   the **focused** fragment only (independent styling per fragment).
2. **Enter** inside a fragment → inserts a **new fragment on the same line** (a new
   contiguous `<span>`), NOT a newline. Lets `"The new shoes"` keep `"shoes"` in a
   bigger, differently-colored span.
3. A new **`+` (icon-plus) button** in the top-right of the toolbar → inserts a
   **new line** (a real break), shown in the panel separated by a horizontal rule.
   Maps to a `<br>` or a new block element.
4. **Parse-time grouping** in `AstEditor`: a run of adjacent sibling text nodes is
   merged into **one** `TextBlockEditor`:
   - `feature-checklist.ts:27-28` — `<h2 title>` + `<p content>` → one editor, two
     lines.
   - `feature-checklist.ts:30-33` — the four `<li>` items (each `✅`-span + text-span)
     → fewer editors instead of one TEXT block per span.

## Architecture decisions (confirmed from code)

- AST is the canonical model and the editor mutates it in place (reactive store).
  No AST→HTML serializer exists; `AstRenderer` renders AST directly. So all work is
  AST shape + editor UI; **no serializer to touch**.
- `AstRenderer` already renders `node.children` inline and splits `node.text` on
  `\n` into `<br>`. Multiple inline `<span>` children of a `<p>` already render
  contiguously — so the runtime side needs little/no change.

## Proposed implementation (pending answers)

### Data shape
One editor binds to a **group of sibling AST nodes** (the lines) plus the parent
`children` array + the start index, so it can splice siblings in/out.
- Each line node: a block-ish element (`<p>`, `<h2>`, `<li>`, …) whose inline
  fragments are its `children` (`<span>` / `#text`), OR a simple text leaf (`.text`).
- New fragment (Enter) → push a `<span>` into the current line's `children`.
- New line (`+`) → splice a new sibling node after the current line. **Tag of the
  new line: see open question Q3.**

### `AstEditor.svelte`
Add a grouping pass: when iterating a container's children, collect maximal runs of
"groupable text nodes" and render ONE `TextBlockEditor nodes={run}` instead of one
per node. Non-text nodes (images, custom components, nested containers — see Q1)
break the run.

### `TextBlockEditor.svelte`
- Accept `nodes: ComponentAST[]` (the line group) + parent/index handles.
- Render fragments as stacked auto-growing text areas; `<hr>`-style separators
  between lines.
- Track the focused fragment; toolbar reads/writes that fragment's `css`/`style`.
- Key handlers: Enter = new fragment; `+` button = new line; Backspace on empty
  fragment = merge into previous (see Q4).

## Locked decisions

- **Q1 — Grouping scope.** Merge a run of adjacent sibling text nodes. **A link
  (`<a>`) BREAKS the run** (it is not an inline fragment). Images, custom components,
  and containers with deeper structure also break the run.
- **Q2 — `<ul>`/`<li>`.** **One editor per `<li>`** — the `✅`-span + text-span are
  two fragments on the `<li>`'s single line. (No cross-`<li>` grouping.)
- **Q3 — New-line `+`.** Adds a **new sibling block element**, **copying the current
  line's tag** (`<p>`→`<p>`, `<li>`→`<li>`, `<h2>`→`<h2>`).
- **Q4 — Deleting.** Backspace in a non-empty fragment = normal char delete.
  Backspace in an **already-empty** fragment deletes that fragment and focuses the
  previous one; if it was the line's last fragment the line (block node) is removed;
  if it was the editor's last line the node is spliced out of the AST. (Two
  backspaces total: empty it, then remove it.)
- **Q5 — `✅` icon.** Rendered as a **compact read-only inline icon chip**. A
  fragment is an icon when its trimmed text has **no word characters** (`\p{L}`/`\p{N}`).
- **Q6 — Enter fragment.** New inline `<span>` **inherits styling from the fragment
  it was split from** (copy its `css`/`style`); user then restyles.
- **Q7 — Per-line tags.** Preserved. Each line keeps its own `tagName`.

## Model summary (locked)

- `<li>` is treated like `<p>`: a block = one visual line of contiguous fragments.
  `+` makes a new sibling block of the same tag; **Enter** adds a contiguous span on
  the same line.
- A leaf line (`<h2>Text</h2>`) holds its text in `node.text` as one virtual
  fragment. The first Enter **materializes** it: `node.text` → a clean first `<span>`,
  then appends a new `<span>`. The block keeps its own `css`/`style`; spans inherit
  via CSS cascade.
- One `TextBlockEditor` binds to a **group of sibling line-nodes** + the parent
  `children` array + start index (so `+` can splice a sibling and empty lines can be
  removed). `feature-checklist`: `[h2,p]` → 1 editor (2 lines); each `<li>` → 1
  editor. Total 5 editors (was 10).
