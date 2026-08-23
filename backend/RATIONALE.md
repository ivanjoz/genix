## The boundary rule is enforced by a script, not by good intentions

**Context** — Go cannot catch a module-to-module import: `accounting` importing `finance`
compiles cleanly. The six violations this refactor removed had accumulated exactly that way,
and two of them had already produced copy-pasted code rather than a compile error. Left
unchecked the rule would decay again.

**Decision** — `scripts/boundaries/check_module_imports.go`, dispatched as
`go run . check_module_imports` and registered in the `deploy.sh` TUI beside `check_tables`.
It reads `go list -json ./...` from `backend/` and classifies every import edge against the
layer table. The rule and its consequences are documented in
`backend/docs/MODULE_BOUNDARIES.md`, with pointers from `AGENTS.md`.

**Rationale** — the layer membership is written as **literal path lists**, not inferred from
directory shape. A new top-level package then fails loudly ("not classified") instead of being
silently guessed into the wrong layer, and `rg moduleBodies` shows the whole policy. Test
imports are included, since a `_test.go` in `accounting` importing `finance` is the same
violation.

Writing the test first paid for itself: it caught that the checker allowed
`finance/types -> cloud`, because the infrastructure allowance was evaluated before the
`*/types`-must-stay-a-leaf rule. That edge does not exist today, so nothing would have
surfaced it until someone added it — which is exactly the case a checker is for. The order is
now leaf-rule first, and `check_module_imports_test.go` pins all six original violations plus
the leaf property.

The checker was also verified end-to-end by injecting `accounting -> finance` and confirming a
non-zero exit and a message naming the fix, then reverting.

