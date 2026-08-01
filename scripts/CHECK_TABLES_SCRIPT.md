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
    tables share one. IDs are packed into `cache_updated_version`'s partition key, so a duplicate
    would silently merge two tables' cached slot versions. See `backend/docs/ORM_DATABASE_QUERY.md`
    §13 for the assigned list and the next free number.

5.  **Incremental-sync contract**: a table declaring `SaveUpdatedVersion: true` or a `db.TypeDelta`
    index must expose `UpdatedVersion int32` with the json tag `upv,omitempty` on the record struct
    **and** a matching `UpdatedVersion` column on the table struct. Without the table-struct column
    the value never reaches the client and both caches break.

6.  **No removed cache-version field**: no record may still declare `CacheVersion` or a `ccv` json
    tag. It was replaced by `UpdatedVersion` / `upv`.

If any of these rules are violated, the script prints a detailed error message, specifying the structs and field where the inconsistency occurred.
