# Check Tables Script

This script performs static analysis on the Go source code in the `../backend` directory to validate data model conventions for the custom ORM.

## Purpose

The script ensures that for every data model, its corresponding "base" struct and "table" struct follow a strict set of rules. This helps catch potential bugs and inconsistencies before runtime.

## Validations Performed

1.  **Naming Convention**: For a base struct named `MyType`, the corresponding table struct must be named `MyTypeTable`.

2.  **Field Consistency**: Every field defined in the table struct must also exist in the base struct.

3.  **Type Mapping Rules**: It enforces the correct usage of `db.Col` and `db.ColSlice` based on the field's type in the base struct:
    *   **Non-Slice Fields**: Must use `db.Col[TableType, FieldType]`.
    *   **Primitive Slices** (`[]string`, `[]int`, etc.): Must use `db.ColSlice[TableType, ElementType]`.
    *   **Complex Slices** (e.g., slices of structs): Must use `db.Col[TableType, SliceType]`.

4.  **Table identity**: every `GetSchema()` declares a `TableSchema.ID` in `1..16383`, and no two
    tables share one. IDs name the table's cache files (slot files, delta logs), so a duplicate
    would silently merge two tables' caches. See `backend/docs/ORM_DATABASE_QUERY.md` §13 for the
    assigned list and the next free number.

5.  **Cache contract** (`backend/docs/ORM_DATABASE_QUERY.md` §3.1):
    *   `CacheByIDs: true` → the record struct declares `Updated int32` with the json tag
        `upd,omitempty`, and the table struct an `Updated` column.
    *   a `db.TypeDelta` index → the record struct declares `Version int16` with the json tag
        `ver,omitempty` and `Metadata *string` with the json tag `mdt,omitempty`; the table struct
        declares a `Version` column and **no** `Metadata` column.

If any of these rules are violated, the script prints a detailed error message, specifying the structs and field where the inconsistency occurred.
