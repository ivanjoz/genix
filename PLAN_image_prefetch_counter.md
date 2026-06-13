# Plan — Prefetched image IDs + in-memory image map + header process tray

Goal: make image upload feel **fast**. When the user confirms an image (✓ button) we reserve
its final ID up front (a quick call), immediately show it as "saved" from an **in-memory
map**, and run the slow AVIF conversion + byte-upload **in the background**. A **header
process/notification tray** lists those background jobs while they run.

Builds on `PLAN_image_autoincrement.md` (scheme: `imageID = autoincrement*10 + configDigit`,
base CDN name `<companyID>_<imageID>`).

---

## Resolved design (decisions baked in)

- **Q1** `GET.image-id-counter` lives in a generic `backend/business/images.go`.
- **Q2** `PostProductoImage` **requires** a client-reserved `ImageID`; the reserve-at-save
  branch is removed (no back-compat).
- **Q3** dedupe guard: a retried `product-image` POST must not append the same `imageID` twice.
- **Q4** keep the ✓ confirm button — it's what triggers the save.
- **Q5** header tray is a **general process/notification tray** (button-layer), but for now
  lists only image processes, driven directly by `inMemoryImages`.
- **Q6** no auto-upload on file-pick; the ✓ click reserves the ID and starts the background work.
- **Q-COUPLING (final):** image bytes stay **coupled to the product** via the existing
  `product-image` endpoint (which needs `ProductID`). Fully optimistic UX. For a **new**
  product the product is saved **first** (to obtain its ID), then the pending images are
  uploaded with that `ProductID`.

---

## Timing

```
[pick file] → read base64 → card shows preview + ✓ / delete
                                   │
                              click ✓
                                   │
   ┌────────────────────────────────┴───────────────────────────────┐
   │ await GET.image-id-counter → { id, name }            (FAST)      │
   │ → inMemoryImages.set(name, {base64,status:'converting'})         │
   │ → onUploaded(name) → form ImageMain/ImageIDs set (optimistic)    │
   │ → card flips to "saved"   ← USER MOVES ON                        │
   │ → start background AVIF conversion (no ProductID needed)         │
   └────────────────────────────────┬───────────────────────────────┘
                                     │ image is now a "pending upload"
                          (product-save button: onSave)
   ┌────────────────────────────────┴───────────────────────────────┐
   │ 1. save the product FIRST  → productoForm.ID is now real         │
   │ 2. flush pending image uploads → POST product-image              │
   │        { ImageID:id, ProductID:realID, ...avifResolutions }      │
   │ 3. on success → inMemoryImages.delete(name) → CDN AVIF           │
   └──────────────────────────────────────────────────────────────────┘
       (header process tray shows convert/upload progress throughout)
```

The user only ever waits on the fast counter fetch. Conversion (3–8 s) runs in the background
between the ✓ click and the product save, so by save-time usually only the byte-upload remains.

---

## Current flow (being changed)

`frontend/ui-components/files/ImageUploader.svelte`
- `onFileChange` → base64 → preview. `uploadImage()` (✓ button, or the parent's deferred
  `imageUploaderHandler`) converts to AVIF (blocking overlay) → `POST_XMLHR` → swaps to CDN
  name. Pending uploads tracked in `imagesToUpload` keyed by `Env.imageCounter`.

Product flow `frontend/routes/negocio/productos/+page.svelte`
- `onChange` captures `uploadHandler` into `imageUploaderHandler`. `onSave` currently does
  `await imageUploaderHandler()` **then** `doPostProductos` — i.e. **image first, product
  second**. We flip this.
- `onUploaded` sets `productoForm.ImageMain/ImageIDs/Image`.

Backend `backend/business/productos.go::PostProductoImage`
- Reserves the autoincrement at save time, derives `imageID`, saves AVIFs, mutates the
  product (`ImageIDs`/`ImageMain`/`ImageDescriptions`); also handles delete (`ImageToDelete`).

---

## 1. Backend — `GET.image-id-counter` (generic `images.go`)

File: `backend/business/images.go`. Register in `backend/business/main.go`:
`"GET.image-id-counter": GetImageIdCounter,`

```go
// GetImageIdCounter reserves ONE per-company image autoincrement and returns the derived
// imageID + base name. Universal: any uploader prefetches its final ID before converting, so
// the client shows the image as "saved" immediately and uploads in the background.
func GetImageIdCounter(req *core.HandlerArgs) core.HandlerResponse {
    configDigit := int32(req.GetQueryInt64("config")) // resolution-set scheme
    if configDigit == 0 { configDigit = imageConfigDigitFull }

    autoincrement, err := db.GetAutoincrementID(fmt.Sprintf("images_%v", req.User.CompanyID), 1)
    if err != nil {
        return req.MakeErr("Error al reservar el id de imagen:", err)
    }
    imageID := int32(autoincrement*10) + configDigit
    return req.MakeResponse(map[string]any{
        "id":   imageID,
        "name": fmt.Sprintf("%v_%v", req.User.CompanyID, imageID),
    })
}
```
- Reuses the same `images_<companyID>` counter and `imageConfigDigitFull` (move/share from
  `productos.go`), so IDs stay consistent with `PostProductoImage`.

## 2. Backend — `PostProductoImage` uses the pre-reserved ID

File: `backend/business/productos.go::PostProductoImage`
- Add `ImageID int32` to the `productoImage` request struct.
- Add path: require `ImageID > 0`; **remove** the `db.GetAutoincrementID` reserve branch (Q2).
  Use `baseName = "<companyID>_<imageID>"` as today.
- Dedupe guard (Q3): `if !slices.Contains(product.ImageIDs, imageID) { prepend… }` so flushing
  a pending upload after the product was already saved with that ID won't double-append (and
  skip the product re-insert when nothing changed).
- Delete path unchanged.

---

## 3. Frontend — global in-memory image map + tray state

New module `frontend/core/inMemoryImages.svelte.ts`:

```ts
import { SvelteMap } from 'svelte/reactivity'

export type InMemoryImageStatus = 'converting' | 'pending' | 'uploading' | 'error'

export interface InMemoryImage {
  name: string          // base CDN name "<companyID>_<imageID>" (map key)
  id: number            // imageID
  folder: string
  base64: string        // shown while in flight
  description?: string
  status: InMemoryImageStatus
  progress: number      // 0..100 (-1 = none)
  error?: string
  upload?: () => Promise<void> // flush: convert (if needed) + POST product-image
}

// Keyed by base name so ANY renderer resolves the same in-flight image by its final CDN name.
export const inMemoryImages = $state(new SvelteMap<string, InMemoryImage>())

export const getInMemoryImageBase64 = (baseName: string): string =>
  inMemoryImages.get(baseName)?.base64 || ""
```
- Keying by base name lets the product list, detail panel, ecommerce card, etc. resolve the
  in-flight image with no state threading. **Replaces** `imagesToUpload`.

## 4. Frontend — `ImageUploader.svelte` refactor

File: `frontend/ui-components/files/ImageUploader.svelte`
- **Remove**: `Env.imageCounter++` / local `imageID`, the `imagesToUpload` effect, the blocking
  "Converting…" gate on the visible card.
- **Keep** the ✓ confirm button (Q4) + delete.
- `onFileChange`: unchanged (base64 + preview), **no fetch** (Q6).
- ✓ click → `confirmImage()`:
  1. `await GET("image-id-counter", { config })` → `{ id, name }` *(only wait)*,
  2. `inMemoryImages.set(name, { name, id, folder, base64, description, status:'converting',
     progress:-1, upload })`,
  3. set `imageSrc` to the final CDN identity, fire `onUploaded(name, description)` +
     `onChange(imageSrc)` → parent form updates as saved; flip card to saved state,
  4. start background **conversion** immediately (no ProductID needed); when done set
     `status:'pending'` and store the converted resolutions on the entry.
  5. expose `entry.upload()` = the byte-upload step (POST `product-image` with `ImageID:id`,
     `ProductID` read live via `setDataToSend`, the converted resolutions, `onUploadProgress`→
     `entry.progress`); on success `inMemoryImages.delete(name)`, on error keep base64 +
     `status:'error'` + `error`.
- `makeImageSrc`: if `getInMemoryImageBase64(<src without -x{size}>)` is non-empty, render a
  single `<img src={base64}>` and skip the `<picture><source>`s (avoid base64 under the wrong
  MIME type) until the entry is freed.
- Capture `imageFile` per-confirm so conversion can run after `confirmImage` returns.
- **Parent integration:** `onChange(imageSrc, flushUpload)` exposes a flush function the parent
  calls after saving its own record (replaces the old `uploadHandler`). For standalone callers
  with no parent save, ✓ may flush immediately once a `ProductID`/key is available.

## 5. Frontend — product save reorders (the core behavior change)

File: `frontend/routes/negocio/productos/+page.svelte`
- `onChange` captures the new flush handler (was `imageUploaderHandler`).
- `onSave`: **save the product first**, then flush pending image uploads:
  1. `await doPostProductos([productoForm])` → product persisted, `productoForm.ID` real,
  2. flush pending uploads (the captured handler / iterate this form's `inMemoryImages`
     entries) → each posts `product-image` with the now-real `ProductID` + prefetched `ImageID`,
  3. close the layer. Because the image already shows from `inMemoryImages` (optimistic), the
     user isn't blocked; uploads finish in the background and show in the tray.
- The product body already carries the optimistic `ImageMain`/`ImageIDs` (set by `onUploaded`),
  so the saved product references the images even before their bytes finish uploading.

## 6. Frontend — other renderers consult the map

- `frontend/routes/negocio/productos/productos.svelte.ts`: helper returning
  `getInMemoryImageBase64(name) || <CDN url>`, used by `mainProductImage` consumers (list rows,
  detail). Other `<ImageUploader>` callers inherit via the component; only hand-built
  `<img>`/`<picture>` CDN URLs need the lookup.

## 7. Frontend — header process / notification tray (Q5)

The red box in the mock-up: slot left of `icon-cog` in `frontend/domain-components/AppHeader.svelte`.
- Add a `<ButtonLayer>` (same component as settings), designed as a general
  process/notification tray but for now driven by `inMemoryImages`:
  - **Button**: visible when `inMemoryImages.size > 0`; spinner + count badge.
  - **Layer**: new `frontend/domain-components/HeaderProcesses.svelte` listing each entry —
    base64 thumbnail, name, status (`Converting…`/`Saving…` + `progress%`), retry for `error`.
    i18n via `tr()`.
- A future `frontend/core/processes.svelte.ts` can generalize it; not built now (minimal code).

---

## Files touched

Backend
- `backend/business/images.go` — **new** `GetImageIdCounter` (+ home for `imageConfigDigitFull`).
- `backend/business/main.go` — register `GET.image-id-counter`.
- `backend/business/productos.go` — `productoImage.ImageID`; require pre-reserved ID, drop
  reserve branch (Q2); dedupe guard (Q3).

Frontend
- `frontend/core/inMemoryImages.svelte.ts` — **new** map + lookup + entry type.
- `frontend/ui-components/files/ImageUploader.svelte` — confirm→reserve→optimistic→bg-convert,
  expose flush; remove `imagesToUpload`/`Env.imageCounter`.
- `frontend/core/store.svelte.ts` — remove `imagesToUpload`.
- `frontend/core/env.ts` — remove `imageCounter`.
- `frontend/routes/negocio/productos/+page.svelte` — `onSave` saves product first, then flushes
  pending image uploads.
- `frontend/routes/negocio/productos/productos.svelte.ts` — in-memory-aware image-URL helper.
- `frontend/domain-components/AppHeader.svelte` — process tray button-layer.
- `frontend/domain-components/HeaderProcesses.svelte` — **new** tray UI.

---

## Implementation order
1. Backend: `GetImageIdCounter` + `PostProductoImage` ImageID/dedupe.
2. Frontend: `inMemoryImages` module.
3. `ImageUploader` refactor (confirm + optimistic + bg-convert + flush handler).
4. Product `onSave` reorder + renderer lookup.
5. Header process tray.

Ready to implement on your go.
