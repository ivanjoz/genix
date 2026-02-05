# Create Edit Table Script

This script generates boilerplate code for new database table models and adds columns to existing tables, ensuring they conform to the project's custom ORM conventions.

## Overview

The script provides two main commands:
- **create**: Generate new table structures from scratch
- **edit**: Add new columns to existing table models

Both commands automate the creation and modification of Go structs and schema definitions, reducing manual effort and preventing common errors.

## Command Summary

```bash
./app.sh <command> [args...]
```

Available commands:
- `create <output_path> <table_name> [field:type:key]...` - Create a new table
- `edit   <table_name> <field:type[:key]>` - Add a column to an existing table

Or run directly from the project root:
```bash
go run scripts/table/create_edit_table.go <command> [args...]
```

---

## Create Command

The `create` command generates a single `.go` file containing all necessary boilerplate for a new database table.

### Purpose

Automates the creation of:
1. **Base Struct (`MyType`)**: Contains the actual data fields with `json` and `db` tags
2. **Table Struct (`MyTypeTable`)**: Defines columns using `db.Col` and `db.ColSlice` for type-safe query building
3. **`GetSchema()` Method**: A method on the table struct that defines the table's name, partition key, and clustering keys

### Mandatory Fields

The following fields are automatically added to every generated base struct, as they are fundamental to the ORM's design:
- `EmpresaID` (Partition Key)
- `Status`
- `Updated`
- `UpdatedBy`

### Usage

**Via app.sh (recommended):**
```bash
./app.sh create <output_path> <table_name> [field:type:key]...
```

**Or run directly from the project root:**
```bash
go run scripts/table/create_edit_table.go create <output_path> <table_name> [field:type:key]...
```

#### Arguments

- `<output_path>`: The relative path and filename for the output Go file. Note that `app.sh` runs from the `scripts/` directory, so you should use paths relative to it (e.g., `../backend/types/my_new_table.go`).
- `<table_name>`: The name of the table in snake_case (e.g., `my_new_table`).
- `[field:type:key]...`: A space-separated list of fields to add to the table.

### Example

**Via app.sh:**
```bash
./app.sh create ../backend/types/customer_profile.go customer_profile first_name:string last_name:string age:int32 'tags:[]string' demo_number:int64:key
```

**Or run directly:**
```bash
go run scripts/table/create_edit_table.go create ../backend/types/customer_profile.go customer_profile first_name:string last_name:string age:int32 'tags:[]string' demo_number:int64:key
```

---

## Edit Command

The `edit` command adds a new column to an existing database table model.

### Purpose

The script automates adding a new field to both the base struct and table struct of an existing database table. This reduces manual effort and helps prevent common errors when modifying data models.

### What It Does

For a given table name and field definition, the script:

1. **Searches the codebase** for the table file (typically in `backend/types/`)
2. **Adds the field** to the base struct with proper `json` and `db` tags
3. **Adds the field** to the table struct with `db.Col` or `db.ColSlice` for type-safe query building
4. **Updates the GetSchema() method** if the field is marked as a clustering key
5. **Reformats the file** to maintain code quality

### Usage

**Via app.sh (recommended):**
```bash
./app.sh edit <table_name> <field:type[:key]>
```

**Or run directly from the project root:**
```bash
go run scripts/table/create_edit_table.go edit <table_name> <field:type[:key]>
```

#### Arguments

- `<table_name>`: The name of the existing table in snake_case (e.g., `product_inventory`)
- `<field:type[:key]>`: The field definition to add in colon-separated format

### Examples

**Add a simple string field:**
```bash
./app.sh edit product_inventory category:string
```

**Add a slice field (stored as set in ScyllaDB):**
```bash
./app.sh edit product_inventory tags:[]string
```

**Add a field and make it a clustering key:**
```bash
./app.sh edit product_inventory warehouse_id:int32:key
```

**Add a float field:**
```bash
./app.sh edit product_inventory discount_rate:float64
```

---

## Common Information

### Field Format

Both commands use the same field format: `fieldName:type[:key]`

- `fieldName`: The name of the field in snake_case (will be converted to CamelCase)
- `type`: The Go type for the field (e.g., `string`, `int32`, `[]string`, `float64`)
- `key` (optional): If present, marks this field as a clustering key in the database schema

### Clustering Keys

- **In the `create` command**: The first key field becomes the primary clustering key, subsequent keys become secondary clustering keys.
- **In the `edit` command**: The `key` suffix adds the field as a new clustering key to the existing schema.
- Keys are automatically added to the `Keys` array in the `GetSchema()` method.
- `EmpresaID` is always the partition key and is automatically handled by the ORM.

### Type Handling

- The script automatically chooses between `db.Col` and `db.ColSlice` based on whether the field is a slice type.
- Slice fields (e.g., `[]string`) are stored as sets in ScyllaDB.
- All standard Go types are supported (string, int32, int64, float64, bool, etc.).

### Code Formatting

Both commands use `go/format` to ensure generated or modified code follows Go formatting standards.

### Field Order

- **For new tables**: Fields are generated in the order specified by the user, after the mandatory system fields.
- **For edited tables**: New fields are appended to the end of both the base struct and table struct.

---

## Important Notes

### Create Command

- The script checks if the table type already exists and will fail if it does.
- The output directory must exist; the script will not create directories.
- All generated files are placed in the `types` package.

### Edit Command

- **Automatic Table Finding**: The script automatically searches the `backend/` directory for files containing the table definition. You don't need to specify the output path.
- **Multiple Keys**: When you mark a field as a key (`:key`), the script automatically updates the `GetSchema()` method to include it in the clustering keys array.
- **AST Manipulation**: The script uses Go's AST (Abstract Syntax Tree) to safely modify existing code while preserving comments and structure.
- **Receiver Support**: The script handles both value receivers (`func (e Table) GetSchema()`) and pointer receivers (`func (e *Table) GetSchema()`).

---

## Error Handling

If the script encounters an error, it will provide a descriptive message:

- **Table already exists** (create only): Choose a different table name or delete the existing one.
- **Table not found** (edit only): Verify the table name is correct and exists in the codebase.
- **Invalid field format**: Ensure the field is in `fieldName:type` or `fieldName:type:key` format.
- **File parsing error**: The table file may have syntax errors that need to be fixed first.

---

## After Running the Script

### After Creating a Table

1. **Verify the generated file** by checking the output path.
2. **Review the generated code** to ensure it meets your requirements.
3. **Update your database schema** to create the new table (this step is manual and depends on your deployment process).
4. **Test your code** to ensure the new table works correctly with ORM queries.

### After Editing a Table

1. **Verify the changes** by checking the modified file.
2. **Run the check_tables script** to validate the updated table:
   ```bash
   ./app.sh check_tables
   ```
3. **Update your database schema** to add the new column (this step is manual and depends on your deployment process).
4. **Test your code** to ensure the new field works correctly with the ORM queries.