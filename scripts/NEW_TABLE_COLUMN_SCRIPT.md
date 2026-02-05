# New Table Column Script

This script adds a new column to an existing database table model, ensuring it conforms to the project's custom ORM conventions.

## Purpose

The script automates adding a new field to both the base struct and table struct of an existing database table. This reduces manual effort and helps prevent common errors when modifying data models.

## What It Does

For a given table name and field definition, the script:

1. **Searches the codebase** for the table file (typically in `backend/types/`)
2. **Adds the field** to the base struct with proper `json` and `db` tags
3. **Adds the field** to the table struct with `db.Col` or `db.ColSlice` for type-safe query building
4. **Updates the GetSchema() method** if the field is marked as a clustering key
5. **Reformats the file** to maintain code quality

## Usage

Run the script from the project root directory using the `app.sh` wrapper:

```bash
./app.sh new_table_column <table_name> <field:type[:key]>
```

### Arguments

- `<table_name>`: The name of the existing table in snake_case (e.g., `product_inventory`)
- `<field:type[:key]>`: The field definition to add in colon-separated format

### Field Format

Each field is defined as a colon-separated string: `fieldName:type[:key]`

- `fieldName`: The name of the field in snake_case (will be converted to CamelCase)
- `type`: The Go type for the field (e.g., `string`, `int32`, `[]string`, `float64`)
- `key` (optional): If present, marks this field as a new clustering key in the database schema

### Examples

**Add a simple string field:**
```bash
./app.sh new_table_column product_inventory category:string
```

**Add a slice field (stored as set in ScyllaDB):**
```bash
./app.sh new_table_column product_inventory tags:[]string
```

**Add a field and make it a clustering key:**
```bash
./app.sh new_table_column product_inventory warehouse_id:int32:key
```

**Add a float field:**
```bash
./app.sh new_table_column product_inventory discount_rate:float64
```

## Important Notes

1. **Automatic Table Finding**: The script automatically searches the `backend/` directory for files containing the table definition. You don't need to specify the output path.

2. **Clustering Keys**: When you mark a field as a key (`:key`), the script automatically updates the `GetSchema()` method to include it in the clustering keys array.

3. **Type Handling**: The script automatically chooses between `db.Col` and `db.ColSlice` based on whether the field is a slice type.

4. **Code Formatting**: The script uses `go/format` to ensure the modified file follows Go formatting standards.

5. **Field Order**: New fields are appended to the end of both the base struct and table struct.

## Error Handling

If the script encounters an error, it will provide a descriptive message:

- **Table not found**: Verify the table name is correct and exists in the codebase
- **Invalid field format**: Ensure the field is in `fieldName:type` or `fieldName:type:key` format
- **File parsing error**: The table file may have syntax errors that need to be fixed first

## After Running the Script

After adding a column:

1. **Verify the changes** by checking the modified file
2. **Run the check_tables script** to validate the updated table:
   ```bash
   ./app.sh check_tables
   ```
3. **Update your database schema** to add the new column (this step is manual and depends on your deployment process)
4. **Test your code** to ensure the new field works correctly with the ORM queries