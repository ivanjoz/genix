---
name: static-project-validation
description: Run static validation checks on the backend Go codebase to catch structural inconsistencies in DB table definitions. Use when the user asks to "validate the project", "run static checks", "check tables", or after modifying Go structs that embed db.TableStruct.
version: 0.1.0
---

# Static Project Validation

Run the static validation suite to catch structural issues in the backend before they cause runtime errors.

## Available validations

### Check Tables (`check_tables`)

Validates consistency between base structs and their corresponding `*Table` structs in the backend Go code. Specifically checks:

- Table struct name follows the convention `<BaseName>Table`
- Every field in a `*Table` struct exists in the corresponding base struct
- `db.Col` vs `db.ColSlice` usage is correct:
  - Primitive slices (`[]string`, `[]int32`, etc.) must use `db.ColSlice`
  - Complex/struct slices must use `db.Col`
  - Non-slice fields must use `db.Col`, never `db.ColSlice`
  - The type argument in `Col`/`ColSlice` must match the base struct field type

## How to run

From the project root:

```bash
cd scripts && go run . check_tables
```

Or run the validation package directly:

```bash
cd scripts && go run ./validation
```

## When to run

Run this validation after:
- Adding or modifying any struct that embeds `db.TableStruct`
- Changing field types in base structs or table structs
- Creating new table definitions

## Output

Errors are printed to stdout with the format:
```
Error: <description of the inconsistency>
```

No output means all checks passed.
