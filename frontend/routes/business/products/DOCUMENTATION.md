---
schema: 1
page_id: business.products
route: /business/products
title: Products (Productos)
status: implemented
visibility: tenant
description_en: >-
  Product catalog management. Create and edit products with price, discount, unit, currency,
  brand, categories, sub-units, presentations, and photos; bulk import and export via Excel.
  Also manages product categories and brands, each with name, description, and images.
description_es: >-
  Gestión del catálogo de productos. Crear y editar productos con precio, descuento, unidad,
  moneda, marca, categorías, sub-unidades, presentaciones y fotos; importar y exportar de forma
  masiva vía Excel. También administra categorías y marcas de productos, cada una con nombre,
  descripción e imágenes.
---

# Products (Productos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Products (`Productos`) is the catalog page for every sellable item (`producto`) a company
offers: name, pricing, unit, currency, brand, categories, optional sub-units and presentations
(variants), and photos. The same page also manages the two shared catalogs a product depends
on — Categories (`Categorías`) and Brands (`Marcas`) — as two extra top-level views.

This page does not manage stock quantities (`existencias`/`stock`) or warehouse assignment;
those are read-only figures here (`Stock`, `ReservedStock`, `StockStatus`) coming from Logistics.
It does not set up sites or warehouses either — use **Sites & Warehouses (Sedes & Almacenes)**
for that. Pricing, categories, and brand data configured here is what other modules (sales,
logistics stock changes, the public storefront) read to describe and sell the product.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **product (producto)** is one catalog item: name, base price (`Precio Base`), discount
  (`Descuento`), final price (`Precio Final`), currency (`Moneda`, PEN or USD), unit (`Unidad`,
  Kg/g/Libras), brand (`Marca`), one or more categories (`Categorías`), optional sub-units,
  presentations, and photos. These three currency and unit lists are fixed values built into the
  frontend, not per-tenant configuration; there is no in-app screen to add a new unit or currency
  here, and this page never converts amounts between PEN and USD.
- A **category (categoría)** and a **brand (marca)** are two independent shared lists
  (`shared-lists`, list IDs 1 and 2) reused across the whole business module — a product can
  belong to several categories but only one brand.
- **Sub-units (`Sub-Unidades`)** capture an alternate name, price, discount, final price, and
  quantity for the same product (for example selling by box in addition to by unit). They are
  plain fields on the product row itself, not a separate product or a related SKU.
- A **presentation (`Presentación`)**, also called a variant, is a product option distinguished
  by a fixed attribute (`Atributo`: Color, Talla, Tamaño, Forma, or Presentación), with its own
  name, price, price difference, optional SKU, and color swatch. Presentations are stored inside
  the product itself (`Presentations`) and are not separate catalog rows; they are only persisted
  when the whole product is saved.
- **Properties (`Properties`)** is a separate, older data field the product record still carries
  (a named group of options, distinct from presentations) but this page currently exposes no form
  to create or edit it — see Limitations under presentations below.
- The **main image (`Imagen Principal`)** is the first photo shown for a product; **additional
  photos (`Fotos`)** are the rest. Both live in the same `ImageIDs`/`ImageDescriptions` pair of
  lists on the product.

<!-- DOC-ID: capability.browse-products -->
## Browse and switch views (Buscar y cambiar de vista)

Open **Business (Negocio) → Products (Productos)** at `/business/products`. The top strip
switches between three views: **Products (Productos)**, **Categories (Categorías)**, and
**Brands (Marcas)**; switching views closes any open side layer/modal and resets the working
form. The filter box ("Filter products|Filtrar productos") narrows the currently loaded list by
product name (Products view) or by name/description (Categories & Brands view); it is a local,
client-side filter over the already-loaded records, not a server search.

The Products list table shows ID, product name, resolved category names, price, discount,
final price, and sub-unit summary (`quantity x unit`); new records that still need a
category/brand resolved (from an import) are flagged inline. Products are listed most-recently
created first (sorted by ID descending on the client). Selecting a row opens it in the side
layer described below.

<!-- DOC-ID: capability.create-edit-product -->
## Create or edit a product (Crear o editar un producto)

### User intention (Intención del usuario)

Add a new item to the catalog, or update an existing product's price, categorization,
description, sub-units, presentations, or photos.

### Where to find it (Dónde encontrarlo)

On the **Products** view, use the green **New (Nuevo)** button to open the side layer for a
new product, or click an existing row to open the same layer pre-filled. The layer has four
tabs: **Info (Información)**, **Sheet (Ficha)**, **Presentations (Presentaciones)**, and
**Photos (Fotos)**. Save with the layer's Save action.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)** is required, at least 4 characters (checked by the frontend and re-checked
  by the server).
- **Base Price (Precio Base)**, **Discount (Descuento)**, and **Final Price (Precio Final)** are
  linked: changing any one of the three recalculates the other two using
  `Final Price = Base Price × (100 − Discount%) / 100`. Discount must stay under 100% (enforced
  only on the frontend).
- **Currency (Moneda)**, **Unit (Unidad)**, **Weight (Peso)**, **Volume (Volumen)**, **Brand
  (Marca)**, **SKU**, and **Short Description (Descripción Corta)** are optional.
- The **SKU Individual** checkbox stores a flag on the product but nothing on this page — or
  anywhere else in the codebase currently — reads or reacts to it; treat it as not wired to any
  visible behavior yet.
- **Categories (`CATEGORÍAS ::`)** lets you attach one or more existing categories by name; it
  cannot create a new category from here (use the Categories view or the Excel import).
- **Sub-Unidades** (Name, Base Price, Discount, Final Price, Quantity) are independent fields:
  unlike the main Price/Discount/Final Price trio, they do not recalculate each other. Note that
  the sub-unit **Base Price** field's `onChange` actually recomputes the *main* product's Final
  Price (from the main Price/Discount), not a sub-unit total — entering a sub-unit base price can
  unexpectedly change the main Final Price shown above it.

### Business rules and rationale (Reglas y razón de negocio)

The server rejects a product name already used by another **active** product for the same
company ("Ya existe un product activo con el nombre ..."). If the colliding name belongs only to
an inactive (deleted) product, saving a new product with that same name reuses the old inactive
row's ID instead of creating a duplicate — so a previously deleted product's ID and history can
resurface under a new save with the same name.

Saving any product, its images, or the Categories/Brands lists on this page requires the company
profile to grant the **"Productos"** access at Create (`Crear`), Edit (`Editar`), or Full
(`Todo`) level — this catalog access has no separate view-only level. Viewing the product,
category, and brand lists themselves is open to any authenticated user of the company, since no
read-only access is mapped for the underlying GET endpoints.

### Result and side effects (Resultado y efectos)

Saving creates or updates the product row and marks the company for a rebuild of the public
storefront's product snapshot file (the periodic e-commerce `.db` export), so catalog changes
made here eventually reach that separate public listing. A brand-new product is created active
immediately; editing preserves stock figures, prior main image assignment, and creation audit
fields untouched.

### Limitations (Limitaciones)

- There is no server-side range check on Discount, Price, or Final Price beyond the frontend's
  "< 100%" discount check.
- The **Properties** field the product record can carry is preserved when editing an existing
  product but this page has no UI to create, populate, or clear it.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo un producto nuevo?` / `¿Cómo edito el precio o el descuento de un producto?`
- `¿Por qué cambió el Precio Final si sólo toqué la Sub-Unidad?` Es un comportamiento actual del
  formulario: el campo "Precio Base" de la sub-unidad recalcula el Precio Final del producto
  principal, no el de la sub-unidad.
- `¿Puedo crear una categoría nueva desde el formulario de producto?` No directamente; usa la
  vista Categorías o la importación de Excel.
- Search terms: `producto`, `precio base`, `descuento`, `precio final`, `sub-unidad`, `SKU`,
  `moneda`, `unidad`.

<!-- DOC-ID: capability.product-sheet -->
## Edit the product sheet (Editar la ficha del producto)

The **Sheet (Ficha)** tab loads a rich text editor (lazily imported on first use) bound to the
product's `ContentHTML` field — a longer, formatted description separate from the short
"Descripción Corta" on the Info tab. It is only persisted when the product itself is saved.

<!-- DOC-ID: capability.presentations -->
## Manage presentations / variants (Gestionar presentaciones)

### User intention (Intención del usuario)

Offer the same product in several variants distinguished by an attribute — for example
different colors or sizes — each with its own name, price, price difference, and SKU.

### Where to find it (Dónde encontrarlo)

Inside the product layer, open the **Presentations (Presentaciones)** tab. Use the green add
button to open the presentation modal, or click an existing row to edit it.

### Required information and prerequisites (Requisitos previos)

**Attribute (Atributo)** must be one of a fixed list — Color, Talla, Tamaño, Forma, or
Presentación — this list is hardcoded and not configurable per company. **Name (Nombre)** is
free text. **Price (Precio)**, **Price Difference (Diferencia Precio)**, **SKU**, and **Color**
are optional. A new presentation's attribute field starts empty (the code that would default it
from the product's own attribute list is present, but nothing on this page ever populates that
list, so it always defaults to nothing); the user must pick an attribute manually.

### Business rules and rationale (Reglas y razón de negocio)

Presentations are edited entirely inside the modal and only become permanent once the whole
product is saved (the modal shows a warning: "La información se guardará cuando se guarde el
producto."). Deleting an already-saved presentation from the modal marks it inactive
(`ss = 0`) instead of removing it, so it can still appear in historical data; deleting a
presentation that was only added in this same editing session (never saved before) removes it
outright.

### Result and side effects (Resultado y efectos)

On save, the server reconciles the submitted presentations against the previously stored ones by
matching attribute + name: it keeps existing IDs for unchanged presentations, assigns new IDs to
new ones, and marks any previously stored presentation absent from the submitted list as inactive
— presentations are never physically deleted from the stored history.

### Limitations (Limitaciones)

- Presentation price fields do not participate in the main product's Price/Discount/Final Price
  calculation; each presentation's price and price difference are independent numbers the user
  enters directly.
- The five available attributes are fixed for every company; there is no admin screen to add a
  sixth attribute type.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo agrego variantes de color o talla a un producto?`
- `¿Se guardan las presentaciones aunque no guarde el producto?` No, se guardan sólo cuando se
  guarda el producto completo.
- Search terms: `presentación`, `variante`, `atributo`, `color`, `talla`, `diferencia de precio`.

<!-- DOC-ID: capability.product-photos -->
## Upload and remove photos (Subir y eliminar fotos)

### User intention (Intención del usuario)

Attach one main image and any number of additional photos to a product.

### Where to find it (Dónde encontrarlo)

The main image uploader is on the **Info** tab, next to the pricing fields. Additional photos
are managed on the **Photos (Fotos)** tab.

### Required information and prerequisites (Requisitos previos)

The **Photos** tab's uploader only renders once the product has an ID — a brand-new,
never-saved product must be saved at least once (from the Info tab) before extra photos can be
added; until then the tab shows a message asking to save first. The main-image picker on the
Info tab has no such restriction: a new product can have a main image picked before the first
save.

### Business rules and rationale (Reglas y razón de negocio)

For a brand-new product, picking a main image only stages it locally (no ProductID exists yet
to attach the upload to); the actual reservation and CDN upload happen automatically right after
the product save completes and receives its real ID. For an existing product, both the main
image and any additional photo upload and persist immediately upon confirming/selecting the
file, independent of the Save button. Removing a photo from the Photos tab also sends its own
request immediately; it does not wait for the layer's Save action.

### Result and side effects (Resultado y efectos)

A confirmed upload prepends the new image to the product's image list and, if it is the first
or was the previous main image, becomes the new main image. Removing an image drops it from the
list and, if it was the main image, promotes the next remaining photo (if any) to main.

### Limitations (Limitaciones)

There is no manual reordering of photos or explicit "set as main" action for an existing photo;
the main image is always the most recently confirmed upload (or whatever remains first after a
removal).

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué no puedo subir más fotos a un producto nuevo?` Debe guardar el producto primero para
  obtener su ID.
- `¿Cómo elimino una foto?` En la pestaña Fotos, use el ícono de eliminar sobre la imagen; se
  borra de inmediato, sin esperar a Guardar.
- Search terms: `foto`, `imagen principal`, `subir imagen`, `eliminar imagen`.

<!-- DOC-ID: capability.delete-product -->
## Remove a product (Eliminar un producto)

Selecting an existing product and using the layer's **Delete (Eliminar)** action, after a
"¿Está seguro...?" confirmation, resubmits the product with its status turned off. The server
does not physically delete the row: the product becomes inactive and disappears from the active
list, but its data (and its name, freed for reuse as described above) is preserved rather than
wiped, and images or other steps of the save flow are skipped for a delete.

### Limitations (Limitaciones)

There is no separate reactivation action on this page for a product removed this way; if the
same name is used again for a new product it will silently reuse the old inactive record's ID.

<!-- DOC-ID: capability.categories-brands -->
## Manage categories and brands (Gestionar categorías y marcas)

### User intention (Intención del usuario)

Maintain the two shared lists every product is organized by: product categories and product
brands.

### Where to find it (Dónde encontrarlo)

Switch the top strip to **Categories (Categorías)** or **Brands (Marcas)**. Both views share the
same card-grid layout and the same modal form; use the green **New (Nuevo)** button or click an
existing card to open it.

### Required information and prerequisites (Requisitos previos)

**Name (Nombre)** is required (at least 4 characters); **Description (Descripción)** is
optional. Up to three images can be attached per category/brand.

### Business rules and rationale (Reglas y razón de negocio)

The server rejects a name already used by another active record in the same list
(Categories and Brands are two separate lists, so the same name can exist once per list). If the
colliding name belongs to a previously deleted record in that list, saving reuses its ID instead
of creating a duplicate — the same reuse-by-name behavior as products.

### Result and side effects (Resultado y efectos)

Saving updates the shared list and, like product saves, marks the company for a rebuild of the
public storefront's category/brand snapshot.

### Limitations (Limitaciones)

The image slots on this form send their upload to a route name
(`product-categoria-image`) that has no matching registered backend endpoint (the actual
endpoint is named `product-category-image`); confirming a category/brand image therefore fails
with an upload error instead of attaching the image. Name and description still save normally —
only the image attachment is affected.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo una categoría o marca nueva?`
- `¿Por qué falla al subir la imagen de una categoría o marca?` Actualmente el formulario envía a
  un endpoint que no existe en el backend; el nombre y la descripción sí se guardan, pero la
  imagen no se adjunta.
- Search terms: `categoría`, `marca`, `crear categoría`, `crear marca`, `imagen de categoría`.

<!-- DOC-ID: capability.import-export -->
## Bulk import and export via Excel (Importar y exportar vía Excel)

### User intention (Intención del usuario)

Load or update many products at once from a spreadsheet, or download the current catalog for
offline editing or reporting.

### Where to find it (Dónde encontrarlo)

On the **Products** view toolbar: the upload-icon button opens the **Import Products from Excel
(Importar Productos desde Excel)** modal; the download-icon button exports directly.

### Required information and prerequisites (Requisitos previos)

Export writes the currently loaded products to `productos.xlsx` using the same columns visible
in the main table: ID, Product, Categories, Price, Discount, Final Price, and Sub-units — it
does **not** include Brand, Unit, Volume, Weight, or Currency, so a straight export→edit→import
round trip will not carry those values unless the columns are added back manually with matching
headers.

Import reads an Excel file against the same columns plus the extra Brand, Unit, Volume, Weight,
and Currency columns (only shown in the import preview grid). Column headers accept either the
English or Spanish name shown in the app (for example "Price" or "Precio").

### Business rules and rationale (Reglas y razón de negocio)

Only these product fields are ever applied from an import: ID, Name, Categories, Price,
Discount, Final Price, Brand, Unit, Volume, Weight, and Currency. Every other field (description,
sheet content, SKU, sub-units, presentations, photos) is left untouched, copied forward from the
currently cached product when the row matches an existing product.

A row is matched to an existing product by ID when a valid ID is given (and rejected with
"ID de producto no encontrado" if that ID does not exist), otherwise by exact product name; an
unmatched row becomes a new product. Category and Brand names not found in the existing
Categories/Brands lists are automatically created as new (pending) records during the import,
resolved to real IDs when the import is saved. Unit and Currency values, in contrast, must match
one of the fixed option lists exactly ("Kg"/"g"/"Libras", "PEN"/"USD") or the row is rejected
with a "unidad no válida"/"moneda no válida" error before saving.

The preview grid highlights, per cell, only the imported fields whose value differs from the
currently stored product, so the user can review exactly what an import will change before
saving. Saving is blocked entirely (no partial import) if any row still has an unresolved
category or brand, or if any file-level validation error remains.

### Result and side effects (Resultado y efectos)

On save, the pending temporary categories/brands are created first, then products are sent to
the server in batches of up to 500 rows, each subject to the same duplicate-active-name rule as
a manual product save.

### Limitations (Limitaciones)

Import cannot change a product's description, rich sheet content, SKU, sub-units, presentations,
or photos — those must be edited individually per product. Export does not include Brand, Unit,
Volume, Weight, or Currency columns.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo importo productos desde Excel?` / `¿Cómo exporto el catálogo a Excel?`
- `¿Por qué mi Excel exportado no tiene la columna Marca o Moneda?` El export actual no incluye
  esas columnas; sólo aparecen en la vista previa de importación.
- `¿Qué pasa si pongo una marca que no existe?` Se crea automáticamente al guardar la
  importación; con Unidad o Moneda inválidas, en cambio, la fila se rechaza.
- Search terms: `importar productos`, `exportar productos`, `excel productos`, `plantilla`,
  `marca no encontrada`, `unidad no válida`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Product names, and category/brand names within their own list, must be unique among **active**
  records for the company; the collision check also considers inactive (deleted) records so a
  reused name silently continues the old record's ID and history instead of erroring.
- Saving anything on this page (products, categories, brands, or any of their images) requires
  the company profile to grant the "Productos" access at Create/Edit/Full level; only viewing is
  open to any authenticated company user, since no read-only level is mapped for the underlying
  list endpoints.
- Product, category, and brand saves each register the company for a rebuild of the public
  storefront's product snapshot file, which is refreshed on its own periodic cycle rather than
  instantly.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Name must be at least 4 characters" / "El nombre debe tener al menos 4 caracteres":**
  lengthen the product, category, or brand name.
- **"Ya existe un product activo con el nombre ...":** another active product already uses that
  exact name; rename the new product or edit the existing one instead.
- **"El nombre ... ya existe en la lista ...":** the same collision rule for categories/brands.
- **Uploading a category or brand image fails with an error:** this is a known issue — the
  upload targets an endpoint name (`product-categoria-image`) that the backend does not expose
  (`product-category-image` is the real one); the name/description still save.
- **A deleted product/category/brand "comes back" under the same name:** the delete only
  deactivates the record; saving a new one with the identical name reuses the old ID and its
  history rather than creating an unrelated record.
- **Can't add more photos to a brand-new product:** save it once from the Info tab first, then
  return to the Photos tab.
- **Import row rejected with "unidad no válida" / "moneda no válida":** the Unit or Currency text
  in the spreadsheet does not exactly match one of the fixed option names (Kg, g, Libras / PEN,
  USD).

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sites & Warehouses (Sedes & Almacenes)** at `/business/branches-warehouses`: configure the
  sites/warehouses that hold stock for these products; this page does not create them.
- **Stock Changes (Cambios Stock)** under Logistics reads/updates stock quantities for these
  products; this page only shows stock as a read-only figure.
- Sales and Point of Sale workflows read this page's price, discount, final price, and
  presentation data when selling a product, but do not edit the catalog from there.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, related-pages]
  - path: frontend/routes/business/products/+page.svelte
    role: page
    hash: sha256:fe3a456a075eec7e1174e98ed783f018df6a75073080e4b7a33833238b40bb23
    supports: [page-purpose, concepts, capability.browse-products, capability.create-edit-product, capability.product-sheet, capability.product-photos, capability.delete-product, rules, troubleshooting]
  - path: frontend/routes/business/products/Attributes.svelte
    role: user-interface
    hash: sha256:7bf7b27546743f9b8d20976e5b83bc863514b445b01a65b5968950e6ae3882b8
    supports: [concepts, capability.presentations]
  - path: frontend/routes/business/products/CategoriesBrands.svelte
    role: user-interface
    hash: sha256:1113f2b82649fd332fb579a3b83e931601822861fb6a6a6523e62acf37ad6688
    supports: [capability.categories-brands, troubleshooting]
  - path: frontend/routes/business/products/products.svelte.ts
    role: frontend-service
    hash: sha256:77bb3c75bd2663b000da54b9e84385f92c2a09dcc20b44234899388f51cc49d6
    supports: [concepts, capability.create-edit-product, capability.presentations, capability.product-photos, capability.delete-product, capability.browse-products, rules]
  - path: frontend/routes/business/products/products.excel.ts
    role: frontend-service
    hash: sha256:228618822d23f333a447f7289bbf5c51a8c761bc374643f01d3e3e5286d797c9
    supports: [capability.import-export]
  - path: frontend/core/products-lists.ts
    role: shared-domain
    hash: sha256:8259b07d640e22b297c675c28abde05da72ae96111e15dd9eb35f96b8b408524
    supports: [concepts, capability.create-edit-product, capability.import-export]
  - path: frontend/services/business/shared-lists.svelte.ts
    role: frontend-service
    hash: sha256:87aab26fe0ba54b7f6b7aa33313f73c356d56a949126a7cb4e617d50bf531244
    supports: [concepts, capability.categories-brands]
  - path: backend/business/main.go
    role: backend-handler
    hash: sha256:2672bd44b6ca86e692c601c6c7389de2bc72eef25ccfa584c7b75233e5786a83
    supports: [capability.categories-brands, troubleshooting]
  - path: backend/business/products.go
    role: backend-handler
    hash: sha256:0142daf60d7e0d4b3196977bc7382ff906fc31a1f60b4aab96745441c117e2fd
    supports: [capability.create-edit-product, capability.delete-product, capability.product-photos, capability.import-export, rules, troubleshooting]
  - path: backend/business/shared-lists.go
    role: backend-handler
    hash: sha256:96520ce6eda5bf98ed074e6de3a5579ccef39340b7e4643910712c28de08b907
    supports: [capability.categories-brands, rules]
  - path: backend/business/types/productos.go
    role: data-model
    hash: sha256:a26703a968048adf7aa708d753e7d689bfeea5e0d9ad5c98447f078cc1a63ef5
    supports: [concepts, capability.create-edit-product, capability.presentations, rules]
  - path: backend/business/product-ecommerce.go
    role: business-logic
    hash: sha256:e396e5fc6059db4716dbe375a38c41101d14677e7754523e740f1082e56b8043
    supports: [rules]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [rules]
  - path: backend/core/api_routes.generated.go
    role: backend-handler
    hash: sha256:6e91c22103d2d352bf41ca7b5a4083656f4a60930aaffbb88da8c4c59319e190
    supports: [capability.categories-brands, troubleshooting]
```
