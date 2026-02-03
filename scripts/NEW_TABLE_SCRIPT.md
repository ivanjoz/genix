# New Table Script

This script generates boilerplate code for new database table models, ensuring they conform to the project's custom ORM conventions.

## Purpose

The script automates the creation of the necessary Go structs and schema definitions for a new database table. This reduces manual effort and helps prevent common errors.

## Files Generated

For a given table name, the script generates a single `.go` file containing:

1.  **Base Struct (`MyType`)**: Contains the actual data fields with `json` and `db` tags.
2.  **Table Struct (`MyTypeTable`)**: Defines columns using `db.Col` and `db.ColSlice` for type-safe query building.
3.  **`GetSchema()` Method**: A method on the table struct that defines the table's name, partition key, and clustering keys.

## Mandatory Fields

The following fields are automatically added to every generated base struct, as they are fundamental to the ORM's design:

-   `EmpresaID` (Partition Key)
-   `Status`
-   `Updated`
-   `UpdatedBy`

## Usage

Run the script from the project root directory using the `app.sh` wrapper:

```bash
./app.sh new_table <output_path> <table_name> [field:type:key]...
```

### Arguments

-   `<output_path>`: The relative path and filename for the output Go file. Note that `app.sh` runs from the `scripts/` directory, so you should use paths relative to it (e.g., `../backend/types/my_new_table.go`).
-   `<table_name>`: The name of the table in snake_case (e.g., `my_new_table`).
-   `[field:type:key]...`: A space-separated list of fields to add to the table.

### Field Format

Each field is defined as a colon-separated string: `fieldName:type[:key]`

-   `fieldName`: The name of the field in snake_case.
-   `type`: The Go type for the field (e.g., `string`, `int32`, `[]string`).
-   `key` (optional): If present, marks this field as a clustering key in the database schema.

### Example

```bash
./app.sh new_table ../backend/types/customer_profile.go customer_profile first_name:string last_name:string age:int32 'tags:[]string' demo_number:int64:key
```
