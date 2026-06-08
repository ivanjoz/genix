# Plan — Autoincrement image IDs + product image fields refactor

Scope: **backend only** (this round). Frontend consumers (PSV builder output, ProductCard/delta-service URL building) are intentionally left for a later pass and will break until then.

## 1. ORM helper — `db.GetAutoincrementID`
File: `backend/db/main.go` (next to `GetCounter`).

```go
// GetAutoincrementID reserves `recordsSize` consecutive autoincrement IDs for the
// given counter key and returns the FIRST reserved raw value (1, 2, 3, …).
// Uses the configured keyspace automatically. `key` is an arbitrary counter name
// (e.g. "images_<companyID>"). Reusable for any per-key sequence.
func GetAutoincrementID(key string, recordsSize int) (int64, error) {
    if recordsSize < 1 {
        recordsSize = 1
    }
    return GetCounter(connParams.Keyspace, key, recordsSize)
}
```
- Returns the raw autoincrement (NOT multiplied by 10). Caller derives the image ID.
- Backed by the existing `sequences` table via `GetCounter`.

## 2. Image ID + filename scheme
- `autoincrement = db.GetAutoincrementID("images_<companyID>", 1)`  ← **per-company** key.
- `imageID = autoincrement*10 + configDigit`.
- `configDigit = 7` for the standard product upload (base x6 + x4 + x2). Dictionary of digit→resolution-set is TODO (a comment/const placeholder for now).
- Filenames in folder `img-productos`:
  - base (x6 / 980px): `<companyID>_<imageID>.avif`        (no suffix)
  - x4 (570px):         `<companyID>_<imageID>-x4.avif`
  - x2 (360px):         `<companyID>_<imageID>-x2.avif`
- Stored on the product: `ImageMain`/`ImageIDs` hold `imageID` (the autoincrement*10+config value).

## 3. Product struct — replace `Images`
File: `backend/business/types/productos.go`
- Remove `ProductImage` type and the `Images []ProductImage` field.
- Add to `Product`:
  - `ImageMain int32` — imageID of the primary image (defaults to first uploaded).
  - `ImageIDs []int32` — all image IDs.
  - `ImageDescriptions []string` — parallel to `ImageIDs`.
- Mirror in `ProductTable`:
  - `ImageMain db.Col[ProductTable, int32]`
  - `ImageIDs db.ColSlice[ProductTable, int32]`
  - `ImageDescriptions db.ColSlice[ProductTable, string]`

## 4. cloud/s3.go — base resolution emits bare name
File: `backend/cloud/s3.go`
- In `SaveConvertImage`, build the object name conditionally: when the resolution label is empty → `<name>.<format>`; else `<name>-<label>.<format>`.
- Product upload passes resolutions `{980: "", 570: "x4", 360: "x2"}` so x6 becomes the bare base file.
- `PostProductoCategoriaImage` keeps passing `"x6"/"x4"/"x2"` labels → unaffected (no regression).

## 5. PostProductoImage rewrite
File: `backend/business/productos.go`
- `productoImage.ImageToDelete` changes from `string` (name) to `int32` (imageID).
- On add:
  1. `autoincrement, _ := db.GetAutoincrementID("images_<companyID>", 1)`
  2. `imageID := int32(autoincrement*10 + 7)`
  3. `baseName := fmt.Sprintf("%v_%v", companyID, imageID)`
  4. Save via `SaveConvertImage`/`SaveImage` with the new `{980:"",570:"x4",360:"x2"}` map.
  5. Prepend `imageID` to `ImageIDs`, prepend description to `ImageDescriptions`, set `ImageMain = imageID`.
  6. Response `imageName` = `img-productos/<baseName>`.
- On delete: drop the matching `imageID` from `ImageIDs` + its description; if it was `ImageMain`, repoint `ImageMain` to the new first ID (or 0 if none).
- The pre-resolution path (`Content_x6/x4/x2`) updated to the same base/suffix naming.

## 6. Update remaining `Images` references
File: `backend/business/productos.go`
- `db.Merge` selective-columns list: replace `t.Images` with `t.ImageMain, t.ImageIDs, t.ImageDescriptions`.
- Merge closure: replace `current.Images = prev.Images` with preservation of the three new fields.
- `GetProductosCMS` Select: replace `q1.Images` with `q1.ImageMain, q1.ImageIDs, q1.ImageDescriptions`.

File: `backend/business/product-ecommerce.go` (read-only PSV builder)
- Selects + `product.Images[0].Name` reference must compile. Minimal change: select `ImageMain` and emit it (frontend wiring deferred). Will emit `ImageMain` int as the image token so the file still builds — exact PSV format change is part of the deferred frontend pass, flagged with a TODO.

## 7. Validate
- `go build ./...` and run the static table validation skill (`create-database-tables` / check-tables) since `ProductTable` columns changed.
