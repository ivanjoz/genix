# check_module_imports

Enforces the backend layering rule documented in `backend/docs/MODULE_BOUNDARIES.md`:
**a module body may never import another module body.** Shared code crosses module lines
through `<module>/types` only.

## Usage

```bash
cd scripts && go run . check_module_imports
```

Also available as "Validar Límites de Módulos" in the `./deploy.sh` TUI, next to
`check_tables`.

Exits 0 when clean, 1 with a report naming each illegal edge and the fix:

```
1 module boundary violation(s):

  app/accounting
      imports app/finance
      a module body may only be imported by a composition root; move the shared code into app/finance/types
```

## How it works

`go list -json ./...` in `backend/` gives every package its import path and imports (test
imports included). Each edge is classified against the layer table, which lives as **literal
path lists** in `boundaries/check_module_imports.go` rather than being inferred — so the policy
is greppable, and a newly added top-level package fails loudly instead of being silently
guessed at.

Run it after adding a module or moving code between them. When it fires, the fix is almost
always to move the shared function into the owning module's `types` package, not to add an
exception.

## Adding a package

New top-level packages need a line in the right list in `boundaries/check_module_imports.go`:

- a new domain module → `moduleBodies`
- a new dependency-free utility → covered by `foundationPrefixes` if it sits under `libs/`
- a new infrastructure service → `infrastructurePackages`
- a new entrypoint → `compositionRootPrefixes`

`check_module_imports_test.go` covers the classifier directly, including the six violations
the rule was written to remove and the leaf property of `*/types`.
