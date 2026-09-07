## The card's edit pencil reveals on `:focus-visible`, not `:focus-within`

**Context** — The pencil on `CompanyCreditCard` swaps places with the `#ID` on hover, and the swap
was also bound to `md:group-focus-within:*` so a keyboard user could reach it. But `Card` renders a
`<div role="button" tabindex="0">`, so **a mouse click focuses the card itself** — and clicking the
pencil focuses its `<button>`. Either click left `:focus-within` true on the card, so the pencil
stayed lit and the `#ID` stayed hidden long after the pointer had left, until the next click
elsewhere. Reproduced over CDP: clicked, pointer parked far away, `hovered: false`,
`focusWithin: true` → pencil `opacity: 1`, id `opacity: 0`.

**Decision** — The keyboard reveal now uses two variants instead of `group-focus-within`:
`md:group-focus-visible:*` (the card div itself tabbed to) and `md:group-has-[:focus-visible]:*`
(the pencil inside it tabbed to). The button's redundant `md:focus:opacity-100` is gone, since it
had the same mouse-latching defect on the button itself.

**Rationale** — `:focus-visible` is the browser's own "was this focus keyboard-driven?" heuristic,
which is exactly the condition the reveal wanted; `:focus-within` was answering a different
question. Two variants are needed because `has-[:focus-visible]` only looks at descendants and the
card is itself focusable. The cost is that a click no longer keeps the pencil reachable without
moving the mouse back onto the card — which is what the hover rule is for.

Verified in the live dev app over CDP, same DOM state, classes swapped in place: old classes →
pencil `1` / id `0` (latched); new classes → pencil `0` / id `1`. Programmatic
`focus({ focusVisible: true })` on the pencil → pencil `1` / id `0`, so the keyboard path is intact.
The hover reveal itself is **not** observable in the headless browser: Tailwind v4 wraps
`group-hover` in `@media (hover: hover)` and headless Chrome reports `hover: none`. Those classes
were not touched.
