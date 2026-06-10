# Plan — Ecommerce Page Content persistence

Persist the builder's per-section content so `handleSave()` in
`SectionEditorLayer.svelte` actually saves, and the builder can reload it.

## Decisions (confirmed with user)
- **PageID = 11 (Inicio)** hardcoded for now (IDs 1-10 reserved for menu/other
  structures; multi-page selector later).
- **SectionID = position**, 1..N, recomputed on every save.
- **Content** is a typed Go struct mirror of `SectionData` (placed-instance
  fields only). The ORM serializes it to CBOR internally — we add
  `cbor:"N,keyasint"` tags for compactness/perf.
- **Hash = FNV-1a 64-bit**, computed in the backend only, over the JSON of the
  parsed content struct. Never sent by the client.
- **Backend-side dedup**: client POSTs every current section; backend loads
  existing rows for `(CompanyID, PageID)`, writes only rows whose hash differs
  or are new, and **soft-deletes** (`Status=0`) positions that no longer exist.

## AS-BUILT — final decisions
- Added **`Status int8`** — the ORM has no hard-delete; soft-delete is the
  project convention and is needed to drop trailing removed sections.
- **No json-rename tags.** Per project policy the frontend mirrors the Go field
  names: only the persisted `SectionData` fields were renamed to PascalCase
  (`type→Type, ast→Ast, content→Content, css→Css, attributes→Attributes`).
  `id` (a runtime uuid, regenerated on load), `category`, and authoring metadata
  (`html/name/...`) are **not persisted** and stay lowercase (Option A).
- The recursive HTML-AST node fields stay frontend-owned/lowercase. `Ast` and
  `Content` are **strict typed structs** (`[]AstNode` mirroring ComponentAST,
  `*ContentFields` mirroring StandardContent) with **camelCase json name tags** so
  the frontend casing round-trips unchanged (Option 1). cbor keyasint gives compact
  storage. `props` stays `map[string]any` (the frontend's Record of primitive
  coerced props — string-keyed, so it decodes safely); section-level `Attributes`
  likewise. StandardContent's `[key]: any` catch-all is **dropped** — only known
  fields persist.
- NOTE: the ORM uses default `cbor.Unmarshal`, so decoding a CBOR map into a bare
  `any`/`interface{}` would yield `map[interface{}]interface{}` and break
  `json.Marshal` on GET — which is why every dynamic spot uses a string-keyed
  typed map (`map[string]any`) with primitive values, never bare `any`.
- `UserUpdated` -> **`UpdatedBy int32`** (project naming convention).
- PageID is taken from the `page-id` query param, defaulting to **11 (Inicio)**
  (IDs 1-10 reserved for menu/other structures).
- `handleSave` **drops** the generated CSS (runtime artifact, regenerated from
  tokens) and just persists the sections.

## Backend — new module `backend/ecommerce/`

### `types/page_content.go` (as built — Option 1, strict typed)
Top-level fields are PascalCase (match the frontend rename, `json:",omitempty"`);
nested frontend-owned structs (`AstNode`, `ContentFields`, `TextLine`,
`GalleryImage`) carry camelCase json name tags so the frontend tree round-trips
unchanged. cbor keyasint throughout.

```go
type SectionContent struct {          // mirrors the persisted SectionData fields
    Type       string            `json:",omitempty" cbor:"1,keyasint,omitempty"`
    Ast        []AstNode         `json:",omitempty" cbor:"2,keyasint,omitempty"`
    Content    *ContentFields    `json:",omitempty" cbor:"3,keyasint,omitempty"`
    Css        map[string]string `json:",omitempty" cbor:"4,keyasint,omitempty"`
    Attributes map[string]any    `json:",omitempty" cbor:"5,keyasint,omitempty"`
}
// AstNode mirrors ComponentAST (recursive: tagName/css/style/text/children/role/
//   props/attributes); ContentFields mirrors StandardContent (title/subTitle/…/
//   textLines []TextLine/gallery []GalleryImage/productIDs []int32/…).
```

Table (paired `XRecord`/`XRecordTable`, `GetSchema`):

```go
type EcommercePageContent struct {
    db.TableStruct[EcommercePageContentTable, EcommercePageContent]
    CompanyID int32           `json:",omitempty"`
    PageID    int16           `json:",omitempty"`
    SectionID int16           `json:",omitempty"`
    Content   SectionContent  `json:",omitempty"`
    Hash      int64           `json:",omitempty"`
    Status    int8            `json:"ss,omitempty"`
    Updated   int32           `json:"upd,omitempty"`
    UpdatedBy int32           `json:",omitempty"`
}
// + EcommercePageContentTable with db.Col[...] mirrors
// Content column: db.Col[EcommercePageContentTable, SectionContent]

func (e EcommercePageContentTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:      "ecommerce_page_content",
        Partition: e.CompanyID,
        Keys:      []db.Coln{e.PageID, e.SectionID},   // (CompanyID,PageID,SectionID) PK
    }
}
```

### `main.go`
```go
var ModuleHandlers = core.AppRouterType{
    "GET.ecommerce-page-content":  GetPageContent,
    "POST.ecommerce-page-content": PostPageContent,
}
```

### `page_content_handlers.go`
- **GetPageContent**: `page-id` query (default 1). Query
  `CompanyID==user.CompanyID && PageID==page && Status>=1`, order by SectionID,
  return the rows.
- **PostPageContent**: body `{ pageId, sections: SectionContent[] }`.
  1. `CompanyID = user.CompanyID`, `pageID` defaults to 1.
  2. Load existing rows -> map `SectionID -> Hash`.
  3. For each incoming section at index i: `SectionID=i+1`,
     `Hash = fnv64a(json.Marshal(content))`. If new or hash differs ->
     stage upsert (Status=1, Updated, UpdatedBy).
  4. Positions beyond `len(sections)` that exist in DB -> stage Status=0.
  5. `db.Insert` (upsert) the changed/deleted batch.
  6. Return the resulting `{SectionID, Hash}` list.

### Register module
Add `ecommerce.ModuleHandlers` to `appHandlersModules` in
`backend/main-handlers.go` (+ `"app/ecommerce"` import).

### Validation
Run `static-project-validation` (`cd scripts && go run . check_tables`) and
`go build ./...`.

## Frontend

### `frontend/services/ecommerce/page-content.svelte.ts`
- `savePageContent(sections: SectionData[])` -> `POST` route
  `ecommerce-page-content`, body `{ pageId: 1, sections }`.
- `getPageContent()` -> `GET` route `ecommerce-page-content?page-id=1`.

### `SectionEditorLayer.svelte` `handleSave()`
Replace the console.logs with `await savePageContent(editorStore.sections)`
(keep generating CSS only if still needed — TBD; the css generation looks
unrelated to persistence, so drop it from save unless you want it stored).

### (Optional, recommended) load on builder mount
`builder-store/+page.svelte`: on mount call `getPageContent()` and populate
`editorStore.sections`, falling back to `storeExample` when empty.

## Open question for the user
- In `handleSave`, the current code also runs `generateCss(...)`. Should the
  generated CSS be **persisted** too (extra column / separate row), or is it a
  runtime artifact we drop from the save path? Plan currently **drops it**.
