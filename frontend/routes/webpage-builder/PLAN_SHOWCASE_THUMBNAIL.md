# Plan — Page showcase thumbnail on "Guardar"

When the user clicks **Guardar** in the builder, capture a screenshot of the page
content, convert it to a single AVIF (~0.4 Mpx), store it via a new
`POST.webpage-showcase-image` endpoint, and show it as the page thumbnail in the
`/webpage-builder/pages` list.

## Decisions (confirmed)
1. Capture **clean content** — hide builder chrome (`.section-outline`, `.section-label`) during capture.
2. **Square-ish, width-driven**: walk `.section-wrapper`s from the top, include whole
   sections until cumulative height ≥ canvas width (≥ 1× width). Crop at the section boundary.
3. **Single AVIF, no x2/x4/x6 variants.** ~0.4 Mpx → `fileToImage(blob, 632, 'avif')` (the worker squares the number: 632² ≈ 400 000 px).
4. **imageID ends in 0** (single image), reserved **server-side** inside the new endpoint
   (`autoincrement*10 + 0`) to avoid the product path's `configDigit = 7` override.
   The endpoint also updates the page's `Image` field (the cache-buster).
5. **Background**: block (with `Loading`) only until the DOM screenshot blob is ready;
   conversion + upload run in the background.
6. **System pages (B)**: persist a minimal `webpages` row for IDs 10–14 so their `Image`
   can be stored. The `webpages` table + `Webpage` struct already exist — **no schema change**.

## Backend

### New handler `webpage/webpage_showcase.go`
- Register `"POST.webpage-showcase-image": PostWebpageShowcaseImage` in `webpage/main.go`.
- Body: `{ Content: string /* base64 avif data-url */ }`; query param `page-id` (int16).
- Steps:
  1. Validate `page-id > 0` and `Content` length.
  2. Reserve ID: `autoincrement, _ := db.GetAutoincrementID("images_<companyID>", 1)`;
     `imageID := int32(autoincrement*10) + 0`.
  3. Strip an optional `...base64,` prefix, `core.Base64ToBytes`, then `cloud.SaveFile`:
     - `Path: "img-webpage"`, `Name: "<companyID>_<imageID>.avif"`, `ContentType: "image/avif"`.
     - **Direct save, no `SaveConvertImage`** (client already produced the AVIF).
  4. Upsert the `webpages` row (load-then-write to avoid wiping `Name`/`Route`):
     - Load row `CompanyID == user.CompanyID && ID == pageID`.
     - If found: set `Image = imageID`, `Updated = SUnixTime()`, `UpdatedBy = user.ID`.
     - If not found (system page 10–14): build a new row with the known system
       `Name`/`Route` (from a small map mirroring `systemRoutes`), `Status = 1`, plus the fields above.
     - `db.Insert(&[]Webpage{row})`.
  5. Respond `{ ImageID: imageID }`.

Note: writing IDs 10–14 here does not affect `PostWebpage` (its next-ID math already
floors at `firstUserPageID = 15`).

## Frontend

### Capture util `routes/webpage-builder/builder/showcase-capture.ts`
`export async function captureShowcaseBlob(): Promise<Blob | null>`
- `const canvas = document.querySelector('.builder-canvas')`.
- Add a `capturing` class to it (CSS hides chrome — see EcommerceBuilder change).
- Measure: iterate `.section-wrapper` children, accumulate `offsetHeight` until
  `sum >= canvas.clientWidth`; record that height `cropH` and the included count.
- `domToCanvas(canvas, { filter, backgroundColor: '#fff' })` from `modern-screenshot`.
- Crop: draw region `(0,0, fullW, cropH*scale)` onto a new canvas sized `width × cropH`,
  `toBlob('image/png')`. Remove the `capturing` class in a `finally`.

### EcommerceBuilder.svelte
- Add CSS: `.builder-canvas.capturing .section-outline,
  .builder-canvas.capturing .section-label { display: none !important; }`.

### SectionEditorLayer.svelte `handleSave()`
```
Loading.standard('Generando vista previa...')        // blocking
const blob = await captureShowcaseBlob()
Loading.remove()
await savePageContent(editorStore.sections)          // existing save
if (blob) uploadShowcaseImage(pageID, blob)          // fire-and-forget (background)
```
`pageID` comes from `page-content.svelte.ts` `currentPageID` (export a getter, or pass it down — it's already set by the route).

### Service `services/webpage/pages.svelte.ts`
- `uploadShowcaseImage(pageID, blob)`:
  - `const avif = await fileToImage(blob, 632, 'avif')`
  - `await POST({ data: { Content: avif }, route: \`webpage-showcase-image?page-id=${pageID}\` })`
  - Run inside an `addProcess`/`updateProcess` header entry (same UX as ImageUploader background upload).
- `WebpagesService.handler()`: instead of **dropping** rows with `ID <= LAST_SYSTEM_PAGE_ID`,
  **merge** their `Image` (+ `upd`, `UpdatedBy`) into the matching `SYSTEM_PAGES` entry, so
  system-page thumbnails appear.
- Add `showcaseImageSrc(page)` helper → `Env.makeCDNRoute('img-webpage', \`${Env.getCompanyID()}_${page.Image}\`) + '.avif'` when `page.Image > 0`.

### pages/+page.svelte
- Replace the placeholder `_thumb` block: if `page.Image > 0`, render
  `<img src={showcaseImageSrc(page)} ...>`; else keep the `icon-picture` placeholder.
- Unique `imageID` per save → URL changes → cache busts automatically.

## Files touched
- `backend/webpage/webpage_showcase.go` (new), `backend/webpage/main.go`
- `frontend/routes/webpage-builder/builder/showcase-capture.ts` (new)
- `frontend/routes/webpage-builder/builder/EcommerceBuilder.svelte`
- `frontend/routes/webpage-builder/builder/SectionEditorLayer.svelte`
- `frontend/services/webpage/pages.svelte.ts`
- `frontend/services/ecommerce/page-content.svelte.ts` (export `getCurrentPageID`)
- `frontend/routes/webpage-builder/pages/+page.svelte`

## Open / assumptions
- CDN folder name `img-webpage` (product images use `img-productos`).
- AVIF quality 0.8 (worker default).
- System-page thumbnail rows use `Status = 1` so the initial `GetWebpages` fetch returns them.
