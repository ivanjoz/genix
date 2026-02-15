# Excel Builder (Genix)

This document describes the Excel library in `frontend/libs/excel`.

It covers:
- How to export Excel files.
- How to import Excel files in 2 steps (`loadFile` -> `extractRecords`).
- How errors are generated and formatted.
- How `rows` vs `rowsWithoutErrors` are built.
- Recommended patterns for page/services integration.

---

## 1) Files and Responsibilities

- `excelBuilder.ts`
  - Main fluent API.
  - Export orchestration (`export`, `download`).
  - Import orchestration (`loadFile`, `extractRecords`).
- `export.ts`
  - Writes workbook, styles, merged headers, validations, and buffer download.
- `import.ts`
  - Reads workbook, maps headers to columns, parses rows and types.
  - Produces formatted row errors and import metadata.
- `helpers.ts`
  - Shared utilities (header normalization, excelize setup, default parsing, etc).
- `types.ts`
  - Public types for columns, options, and results.

---

## 2) Core API Summary

### Builder class

`ExcelBuilder<T>` is generic and designed for fluent setup.

Main methods:
- `setColumns(columns)`
- `setRecords(records)`
- `setSheetName(sheetName)`
- `setExportSheet(sheetName, title?)`
- `setImportSheet(sheetName?)`
- `setHeaderRowIndex(row)`
- `setHeaderRows(rows)`
- `setBodyRowIndex(row?)`
- `setCreator(creator)`
- `setIncludeTitleRow(boolean)`
- `setIncludeGroupedHeaders(boolean)`
- `export(overrides?)`
- `download(fileName, overrides?)`
- `loadFile(source, overrides?)`
- `extractRecords(handler?)`

Important:
- `setSource` was removed.
- Import source is required in `loadFile(source, ...)`.

---

## 3) Export Flow

### Recommended usage

```ts
const builder = new ExcelBuilder<MyRecord>()
  .setColumns(columns)
  .setRecords(records)
  .setExportSheet('Productos', 'Productos')
  .setHeaderRowIndex(2)
  .setIncludeTitleRow(true)
  .setIncludeGroupedHeaders(true)
  .setCreator('Genix');

await builder.download('productos.xlsx');
```

### What export supports

- Grouped headers (`subcols`) with merge cells.
- Title row.
- Auto width per leaf column.
- Cell styles from column type/format.
- Body row start override.
- List validation through `excel.validationList`.

---

## 4) Import Flow (Two Steps)

Import is intentionally split:

1. `loadFile(source, options?)`
2. `extractRecords(handler?)`

### Minimal import

```ts
const builder = new ExcelBuilder<MyRecord>()
  .setColumns(columns)
  .setHeaderRows([2]);

await builder.loadFile(file);
const result = builder.extractRecords();
```

### Import with transformation/validation

```ts
const builder = new ExcelBuilder<MyRecord>()
  .setColumns(columns)
  .setHeaderRows([2]);

await builder.loadFile(file);

const result = builder.extractRecords((row) => {
  const errors: string[] = [];
  const current = row as MyRecord;

  if (!current.Nombre) errors.push('nombre requerido');
  if (current.Precio && current.Precio < 0) errors.push('precio no válido');

  return errors.length > 0 ? errors : undefined;
});
```

---

## 5) Header Mapping Rules

The importer maps Excel headers against leaf columns using normalized keys:
- Composed key (`Parent_Child`) when grouped headers exist.
- Direct header.
- Internal normalized key.

Normalization removes accents, normalizes spaces, and lowercases.

`headerRows`:
- Supports 1 or 2 header rows.
- Max 2 rows.
- Defaults to `[2]`.

---

## 6) Row Parsing Rules

For each mapped cell:
- Raw cell value is converted to string and trimmed by the importer.
- Parsing order:
  - If column defines `parseValue`, it is used.
  - Else default parser uses `excel.type`.
- Assignment order:
  - If column defines `setValue`, it is used.
  - Else, if `field` exists and value parsed, assign to `field`.
- If `importField` exists, trimmed raw value is assigned to that field.

Implication:
- Consumers do not need to trim imported fields again.

---

## 7) Number Parsing Behavior

Default numeric parser:
- First tries strict `Number(raw)`.
- If invalid, attempts to extract first numeric token and parse it.

Examples that now parse:
- `"23%"` -> `23`
- `"S/ 19.90"` -> `19.9`
- `"abc 12,5"` -> `12.5`

If parsing still fails, importer pushes a formatted row error.

---

## 8) Error Formatting Contract

All errors returned in `ExcelImportResult.errors` are already formatted strings.

Format:
- `Fila {n}: {Message}`

Behavior:
- First letter is capitalized.
- `"no valido"` is normalized to `"no válido"`.

This applies to:
- Parser errors (`import.ts`).
- Handler errors (`extractRecords`).

Do not reformat errors again in consumers.

---

## 9) Result Object Contract

`ExcelImportResult<T>`:
- `rows: Partial<T>[]`
  - All parsed non-empty rows.
- `rowsWithoutErrors: Partial<T>[]`
  - Only rows with zero errors.
  - Excludes rows with parser errors.
  - Also excludes rows with handler errors.
- `rowNumbers?: number[]`
  - Excel row numbers aligned with `rows` by index.
- `errors: string[]`
  - Already formatted strings (`Fila X: ...`).
- `mappedColumns: string[]`
- `ignoredHeaders: string[]`
- `sheetName: string`

---

## 10) How `rowsWithoutErrors` Is Calculated

Stage 1 (`parseExcelFile`):
- Tracks row numbers with parser errors.
- Adds all non-empty rows to `rows`.
- Adds only clean rows to `rowsWithoutErrors`.

Stage 2 (`extractRecords(handler)`):
- Starts from parser output.
- Re-evaluates each row through handler.
- If handler returns errors for row, row is excluded from final `rowsWithoutErrors`.
- If handler has no errors and parser had no errors for row, row is included.

---

## 11) Column Config Quick Reference

Inside each `ExcelTableColumn<T>`:
- `excel.type`: `'string' | 'number' | 'date' | 'boolean'`
- `excel.format`: Excel display format
- `excel.exportValue(record, rowIndex)`
- `excel.parseValue(raw, saveError, rowDraft)`
- `excel.setValue(rowDraft, parsedValue, raw)`
- `excel.importField`
- `excel.validationList`
- `excel.ignoreOnExport`

Common pattern:
- Use `importField` to keep raw labels from Excel.
- Use handler in `extractRecords` to resolve labels to IDs.

---

## 12) Recommended Integration Pattern

In route/service module:

1. Build columns in page/service.
2. Create builder and call `loadFile(file)`.
3. Call `extractRecords(handler)` with domain validation + enrichment.
4. Use:
   - `result.rows` for preview (including invalid rows).
   - `result.rowsWithoutErrors` for safe submit batches.
   - `result.errors` for modal/error panel.

---

## 13) Migration Notes (Current State)

- Old `setSource(...)` pattern is deprecated/removed.
- Old one-call `import()` flow is replaced by:
  - `await loadFile(source)`
  - `extractRecords(handler?)`
- Consumer modules should not prepend `Fila ...` manually.
- Consumer modules should not trim imported values for `importField` validations.

---

## 14) Troubleshooting

`No loaded file found. Call loadFile() before extractRecords().`
- Call `await builder.loadFile(file)` before extraction.

`headerRows supports a maximum of 2 rows`
- Pass one row (`[2]`) or two rows (`[1, 2]`) only.

No rows imported:
- Verify selected sheet and header row indexes.
- Check `ignoredHeaders` and `mappedColumns`.

Too many row errors:
- Inspect `errors` list and fix source values.
- Prefer `rowsWithoutErrors` for backend submission.

---

## 15) Implementation Notes

- Runtime uses `excelize-wasm`.
- WASM module is lazily initialized and cached.
- Import/export logs use `[excel-builder]` prefix.

Keep this document updated when changing:
- `ExcelImportResult` shape.
- Error formatting rules.
- Parsing behavior for default types.
