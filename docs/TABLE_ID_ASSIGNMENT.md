# Assigning `TableSchema.ID`

**Status: proposal.** Nothing here is implemented. It was written after adding four tables for the
invoicing module, where picking their IDs took three wrong attempts, and the failure turned out to
be a property of the tooling rather than of the person using it.

Every table declares a numeric `TableSchema.ID`. It occupies the low 14 bits of the by-IDs cache
key, so two tables sharing one silently share their cached slot versions — which is why
`ClaimTableID` panics on a duplicate, and why an ID, once assigned, is permanent.

Choosing one requires knowing which are already taken. That question has no good answer today.

## What went wrong

Three attempts to find a free ID by searching the source, three different answers, all wrong.

| Attempt | Pattern | Result |
| --- | --- | --- |
| 1 | `db\.TableSchema\{\s*\n\s*ID:\s*\n\s*Name:` | Said 44 and 45 were free. They are `user_logs` and `request_errors`. |
| 2 | `grep -A3 'db.TableSchema{'` | Said 47 was free. It is `server_metrics`. |
| 3 | Python, `db\.TableSchema\{(.{0,400}?)\}` | Contradicted both, missing 13 tables outright. |

The first answer reached `backend/invoicing/PLAN.md` and was reviewed without anyone noticing, since
checking it would have meant repeating the same unreliable search.

### Why they failed: comments

Every failure has one cause. These three schemas open with an explanatory comment:

```go
func (e ServerMetricTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		// Written externally in whole rows, like user_logs and the credit_usage_* tables: no
		// ORM-managed created/updated columns and no update counter. A sample is written once
		// and never touched again, so there is nothing for either to describe.
		ID:                    47,
		Name:                  "server_metrics",
```

- **Attempt 1** anchored `ID:` to the line after the opening brace. `user_logs` (2 comment lines),
  `request_errors` (1) and `server_metrics` (3) all became invisible.
- **Attempt 2** widened the window to three lines, which covers the first two. `server_metrics`
  keeps its `ID:` four lines down, so it stayed invisible — and it was the next ID chosen.
- **Attempt 3** scanned to the first nested brace, capped at 400 characters. A schema whose comments
  push that brace past the cap produces *no match at all*, not a partial one: `companies`, `users`,
  `purchase_order`, `warehouse_product_stock`, `shared_list_records` and eight others vanished.

The tables that defeat a text search are the ones that explain themselves. `zz_demo_struct`, a
throwaway, matched every pattern on the first try.

## What the tooling already knows

`scripts/validation/check_tables.go:336` builds `tableNamesByID` — a complete, authoritative map of
every claimed ID, produced by loading the packages and reading the compiled constants. It uses that
map to report collisions, and then discards it.

Fourteen lines later it prints:

```
Error: Table 'x' does not declare TableSchema.ID. Assign it the next free ID (1..16383).
```

It is instructing the reader to go and find something it is holding in a variable.

The scaffolder makes this the default path. `scripts/table/create_edit_table.go:156` writes
`Name:` and `Partition:` and no `ID:` at all, so every generated table starts by failing that check.
The loop is:

```
scaffolder omits the ID  →  human searches the source (unreliable)
                         →  validator rejects the guess
                         →  human guesses again
```

At no point does the component with the correct answer offer it.

## Proposed changes

### 1. The validator reports the free IDs

Roughly ten lines, and it would have prevented all three mistakes above.

```go
// after the collision pass, with tableNamesByID already built
free := make([]int64, 0, 8)
for id := int64(1); id <= maxTableID && len(free) < 8; id++ {
	if _, taken := tableNamesByID[id]; !taken {
		free = append(free, id)
	}
}
fmt.Printf("Free TableSchema.IDs: %v\n", free)
```

And fold the same list into the two error messages, so

> `Assign it the next free ID (1..16383)`

becomes

> `Assign it 9 (free: 9, 24, 29, 35, 37, 43, 44, 45)`

A rejection becomes an instruction.

### 2. The scaffolder assigns the ID

`create_edit_table.go create` should write the ID itself, from the same set. Then the choice never
reaches a human and the failure mode disappears instead of becoming easier to recover from.

More work than the first change: the generator lives in the `scripts` module and would need to load
the backend packages the way the validator does, or shell out to it and parse one line — which is
acceptable here only because the line would be produced by the authoritative tool rather than
scraped from source.

### 3. Reserve a range per domain

There are 16383 IDs available and 53 in use, so blocks are free:

| Range | Domain |
| --- | --- |
| 1–19 | config, core |
| 20–39 | business, logistics |
| 40–49 | sales, finance |
| 50–69 | invoicing |
| 70+ | unassigned |

Documentation only, no code. It makes intent visible in the place someone reads before adding a
table, and makes accidental collisions structurally unlikely. It only helps people who read it, so
it complements the first change rather than replacing it.

### 4. Run the validator automatically

`AGENTS.md` §6 already says to run `check_tables` after touching DB structs. A hook makes that
automatic rather than remembered — see the `update-config` skill for the mechanics.

## Considered and rejected

**Deriving the ID by hashing the table name.** It removes assignment altogether, which is
attractive, but IDs are permanent and packed into cache keys: renaming a table would silently
repoint its cached slot versions, which is exactly the corruption the ID exists to prevent. Fourteen
bits is also not a comfortable hash space. Recorded here so the idea does not have to be
re-evaluated later.

## The general rule

Beyond table IDs:

> Any invariant enforced by a runtime panic should ship with a tool that tells you the correct
> answer, not merely rejects yours.

`ClaimTableID` panicking is right as a last line of defence. A person searching the source for what
it will accept is the part that should not exist.
