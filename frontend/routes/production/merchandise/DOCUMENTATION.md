---
schema: 1
page_id: production.merchandise
route: /production/merchandise
title: Merchandise (Mercaderías)
status: implemented
visibility: tenant
description_en: >-
  Catalog of the merchandise the company buys to resell. Create and edit items with price,
  discount, unit, currency, brand, categories, sub-units, presentations, photos and web store
  settings; bulk import and export via Excel. Also manages product categories and brands.
description_es: >-
  Catálogo de las mercaderías que la empresa compra para revender. Crear y editar mercaderías con
  precio, descuento, unidad, moneda, marca, categorías, sub-unidades, presentaciones, fotos y
  ajustes de la tienda web; importar y exportar de forma masiva vía Excel. También administra
  categorías y marcas.
---

# Merchandise (Mercaderías)

<!-- DOC-ID: page-purpose -->
## Page purpose

Merchandise (`Mercaderías`) is the catalog page for the items the company **buys to resell** —
goods bought from a supplier (`proveedor`) and sold without transforming them. Each item is a
product (`producto`) with a name, pricing, unit, currency, brand, categories, optional sub-units
and presentations (variants), photos and web store settings. The same page also manages the two
shared catalogs a product depends on — Categories (`Categorías`) and Brands (`Marcas`).

The catalog is one list split in two pages by the product's origin (`Origen`): this page shows
only bought products, while **Products (Productos)** at `/production/products` shows what the
company makes, and only there can a product have a recipe (`receta`) and production cost. Both
pages have the same list, form, import and export.

This page does not show or change stock quantities (`existencias`/`stock`); merchandise enters
stock through purchase orders or stock changes. It does not manage supplies (`insumos`) for
internal use either — those are in **Supplies & Materials (Insumos & Materiales)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Origin (`Origen`)** says whether a product is **Bought (merchandise) / Comprado (mercadería)**
  or **Made by the company (finished good) / Fabricado (producto terminado)**. This page lists only
  `Comprado`. The origin also decides how other modules treat the product:
  - Merchandise can be bought on a **Purchase Order (Orden de Compra)**; a made product cannot.
  - In **Consolidation & Accounts (Consolidado & Cuentas)**, the sale of merchandise posts revenue
    to 7012 (Ventas — mercaderías) and its cost of sales to 6912 against stock 2011 (Mercaderías);
    made products use 7022, 6922 and 2111 instead.
- **Merchandise vs. supply:** merchandise is sold to customers; a supply or material (`insumo`,
  `material`) is bought for internal use or production and is managed in Supplies & Materials, not
  here.
- A **product (producto)** has a base price (`Precio Base`), discount (`Descuento`), final price
  (`Precio Final`), currency (`Moneda`, PEN or USD), unit (`Unidad`, Kg/g/Libras), brand (`Marca`),
  one or more categories (`Categorías`), optional sub-units, presentations, and photos. The
  currency and unit lists are fixed, and this page never converts amounts between PEN and USD; the
  Point of Sale reads the product's currency to convert the price when it is sold in the other
  currency.
- A **category (categoría)** and a **brand (marca)** are two independent shared lists; a product
  can belong to several categories but only one brand. Both catalog pages use the same lists.
- **Sub-units (`Sub-Unidades`)** let the same item be sold fractionally — a box of six candies sold
  one at a time, or a kilogram sold by the gram. **`Quantity` (Cantidad) is the divisor: how many
  sub-units make one whole unit**, `Name` (Nombre) labels it, and `Final Price` (Precio Final) is
  what one sub-unit costs; a sub-unit is charged at its own price, never at a fraction of the
  whole-unit price.
- **A sub-unit can be added to an item that already has stock and sales**; existing records simply
  hold no sub-unit part. Once set, the divisor may only be **refined to a multiple of itself**
  (6 → 12 is accepted, 12 → 6 and 6 → 5 are rejected) and it can no longer be removed, because
  stored stock and movements are expressed in it. The maximum divisor is 1000.
- A **presentation (`Presentación`)**, or variant, is an option distinguished by a fixed attribute
  (`Atributo`: Color, Talla, Tamaño, Forma, or Presentación), with its own name, price, price
  difference, optional SKU, and color swatch, stored inside the product.
- The **main image (`Imagen Principal`)** is the first photo shown; **additional photos (`Fotos`)**
  are the rest.

<!-- DOC-ID: capability.browse-products -->
## Browse and switch views (Buscar y cambiar de vista)

Open **Commercial (Comercial) → Catalog (Catálogo) → Merchandise (Mercaderías)** or **Operations
(Operaciones) → Production (Producción) → Merchandise (Mercaderías)**; both open
`/production/merchandise`, listed before **Productos**. The page title and the first tab of the top
strip read **Merchandise (Mercaderías)**; the strip switches between **Merchandise
(Mercaderías)**, **Categories (Categorías)**, and **Brands (Marcas)**. Switching views closes any
open side layer and resets the working form.

The list shows only bought products (`Origen = Comprado`). Made products are never listed here —
look for them in **Productos**. The filter box ("Filter products|Filtrar productos") narrows the
loaded list by name (or by name/description in Categories and Brands); it is a local filter, not
a server search.

The table shows ID, name, category names, price, discount, final price, and the sub-unit summary
(`quantity x unit`), most recently created first. Selecting a row opens it in the side layer.

<!-- DOC-ID: capability.create-edit-product -->
## Create or edit merchandise (Crear o editar una mercadería)

### User intention (Intención del usuario)

Add a new item bought for resale, or update an item's price, categorization, origin, description,
sub-units, presentations, photos or web store settings.

### Where to find it (Dónde encontrarlo)

On the **Merchandise** view, use the green **New (Nuevo)** button, or click an existing row. The
side layer has the tabs **Info (Información)**, **Sheet (Ficha)**, **Presentations
(Presentaciones)**, and **Photos (Fotos)**. Save with the layer's Save action.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)** is required, at least 4 characters (checked by the form and again by the
  server).
- **Base Price (Precio Base)**, **Discount (Descuento)**, and **Final Price (Precio Final)** are
  linked: changing one recalculates the others with
  `Final Price = Base Price × (100 − Discount%) / 100`. Discount must stay under 100% (checked only
  by the form).
- **Origin (Origen)** starts as **Comprado (mercadería)** for an item created from this page.
  Changing it to **Fabricado (producto terminado)** and saving moves the item to the **Productos**
  page; while the form says `Fabricado`, a **Production (Producción)** tab for the recipe appears.
  The server refuses the change to `Fabricado` while the item has stock in any warehouse, in any
  presentation, lots and serial numbers included ("tiene stock: no puede pasar a fabricado mientras
  lo tenga"). The opposite change, a made product back to `Comprado`, is refused while delivered
  sales of it are still **Por regularizar** in Sales Management.
- **Currency (Moneda)**, **Unit (Unidad)**, **Weight (Peso)**, **Volume (Volumen)**, **Brand
  (Marca)**, **SKU**, and **Short Description (Descripción Corta)** are optional.
- The **SKU Individual** checkbox stores a flag, but nothing in Genix currently reacts to it.
- **Categories (`CATEGORÍAS ::`)** attaches existing categories by name; it cannot create one.
- The gray **Tienda Web** box holds the web store settings:
  - **Allowed overselling (`Sobreventa permitida`)** is a number of whole units the web store may
    sell beyond the stock. With 5 units in stock and 100 here, the web store accepts up to 105
    units in total. Orders not yet delivered (till sales, web orders and open restaurant table
    orders) count as reserved and use up the margin. Delivering an oversold order still needs the goods in the warehouse, so it waits
    for the purchase to arrive. The value cannot be negative, and the Point of Sale ignores it for
    merchandise: a till sale needs stock that is not reserved.
  - **Hide (`Ocultar`)** removes the item from the web store catalog, and checkout refuses it ("ya
    no está disponible en la tienda") even if it is still in a shopper's cart.
- **Sub-Unidades** (Name, Base Price, Discount, Final Price, Quantity) are on the **Presentations
  (Presentaciones)** tab. They do not recalculate each other; the sub-unit **Base Price** field
  recalculates the *main* Final Price from the main Price and Discount.
- Saving is rejected when the sub-unit **Quantity** is above 1000, a sub-unit has no **Name**, the
  divisor is changed to a value that is not a multiple of the current one, or a sub-unit that was
  already set is removed.

### Business rules and rationale (Reglas y razón de negocio)

The server rejects a name already used by another active product — made, bought, or a supply of
Supplies & Materials — with "Ya existe un producto activo con el nombre ...". If the name belongs
only to a deleted product, saving a new item with that name reuses the deleted product's ID and
history (with the origin chosen now). Two rows with the same name in one save are refused.

A merchandise item has no recipe or production costs: saving with Origin `Comprado` drops any the
product had, and the server refuses them on a bought product.

### Result and side effects (Resultado y efectos)

Saving creates or updates the product; a new item is active immediately, and editing keeps the
stored stock figures, images and creation data. The web store catalog picks up the change on its
next sync and lists an active, not hidden item only while it has something sellable (stock in the
warehouses open to online sales, or overselling margin left). The overselling margin and the
accounting fields are never sent to the public web store.

### Limitations (Limitaciones)

- There is no server-side range check on Discount, Price, or Final Price beyond the form's
  "< 100%" discount check.
- The page does not show stock or purchase cost; use **Stock Changes (Cambios Stock)**.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo registro una mercadería nueva para revender?` / `¿Cómo cambio el precio de venta?`
- `¿Dónde está mi producto? No aparece en Mercaderías.` Si su origen es `Fabricado`, está en
  Productos.
- `¿Por qué no puedo comprar un producto en una orden de compra?` Sólo se compran mercaderías e
  insumos; un producto `Fabricado` se rechaza.
- `¿Qué es sobreventa permitida?` Unidades que la tienda web puede vender por encima del stock.
- Search terms: `mercadería`, `mercaderia`, `producto para reventa`, `artículo`, `comprado`,
  `origen`, `precio de venta`, `descuento`, `sub-unidad`, `SKU`, `sobreventa`, `ocultar`.

<!-- DOC-ID: capability.product-sheet -->
## Edit the product sheet (Editar la ficha)

The **Sheet (Ficha)** tab opens a rich text editor for a longer, formatted description, separate
from the "Descripción Corta". It is saved with the item. The web store's product detail shows it
unless the item is hidden.

<!-- DOC-ID: capability.presentations -->
## Manage presentations / variants (Gestionar presentaciones)

### User intention (Intención del usuario)

Sell the same item in variants — colors, sizes, shapes — each with its own name, price, price
difference, and SKU.

### Where to find it (Dónde encontrarlo)

Inside the item's layer, open **Presentations (Presentaciones)**. The green add button opens the
**Producto Presentación** modal; a row's edit button changes it.

### Required information and prerequisites (Requisitos previos)

**Attribute (Atributo)** is one of Color, Talla, Tamaño, Forma, or Presentación — the same list for
every company. **Name (Nombre)** is free text; **Price (Precio)**, **Price Difference (Diferencia
Precio)**, **SKU**, and **Color** are optional. A new presentation's attribute starts empty.

### Business rules and rationale (Reglas y razón de negocio)

Presentations become permanent only when the item is saved ("La información se guardará cuando se
guarde el producto."). Deleting a saved presentation marks it inactive instead of removing it,
because sales may already reference it; one added in the same session is removed outright.

### Result and side effects (Resultado y efectos)

On save, presentations keep their IDs when attribute and name match the stored ones, new ones get
new IDs, and stored ones missing from the submission are kept as inactive.

### Limitations (Limitaciones)

- Presentation prices do not take part in the main Price/Discount/Final Price calculation.
- There is no screen to add another attribute type.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo agrego tallas o colores a una mercadería?`
- `¿Se guardan las presentaciones sin guardar el producto?` No.
- Search terms: `presentación`, `variante`, `atributo`, `color`, `talla`, `diferencia de precio`.

<!-- DOC-ID: capability.product-photos -->
## Upload and remove photos (Subir y eliminar fotos)

### User intention (Intención del usuario)

Attach one main image and additional photos to an item.

### Where to find it (Dónde encontrarlo)

The main image uploader is on the **Info** tab; additional photos are on **Photos (Fotos)**.

### Required information and prerequisites (Requisitos previos)

The **Photos** tab uploader appears only after the item has been saved once ("Guarde el producto
para poder cargar más imágenes"). The main image can be picked for a new item before saving.

### Business rules and rationale (Reglas y razón de negocio)

For a new item, the picked main image uploads automatically right after the first save. For an
existing item, a confirmed image is added at once and uploads in the background, shown in the
header's process tray, independent of Save. Removing a photo, after its confirmation, is applied
immediately.

### Result and side effects (Resultado y efectos)

A confirmed upload goes first in the image list and becomes the main image; removing the main image
promotes the next remaining photo, if any.

### Limitations (Limitaciones)

No manual reordering and no "set as main" action for an existing photo.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué no puedo subir más fotos a una mercadería nueva?` Guárdela primero.
- Search terms: `foto`, `imagen principal`, `subir imagen`, `eliminar imagen`.

<!-- DOC-ID: capability.delete-product -->
## Remove merchandise (Eliminar una mercadería)

Open an item and use **Delete (Eliminar)**; after the "¿Está seguro que desea eliminar ...?"
confirmation it is saved as inactive. It leaves the list and the web store, but its data is kept
and its name becomes free for reuse. There is no reactivation action; saving a new item with the
same name reuses the deleted record's ID.

<!-- DOC-ID: capability.categories-brands -->
## Manage categories and brands (Gestionar categorías y marcas)

### User intention (Intención del usuario)

Maintain the shared category and brand lists used to organize merchandise and products.

### Where to find it (Dónde encontrarlo)

Switch the top strip to **Categories (Categorías)** or **Brands (Marcas)**; use the green **New
(Nuevo)** button or click a card to open the **CATEGORÍA** modal. These are the same lists shown
on the Productos page.

### Required information and prerequisites (Requisitos previos)

**Name (Nombre)** is required (at least 4 characters in the form); **Description (Descripción)** is
optional; up to three images can be attached.

### Business rules and rationale (Reglas y razón de negocio)

A name already used by another active record of the same list is refused ("El nombre ... ya existe
en la lista ..."); a deleted record's name is reused with its ID. Renaming a brand also updates the
brand name stored on its products, so a product search by the new brand name finds them.

### Result and side effects (Resultado y efectos)

Saving updates the shared list for both catalog pages; the web store catalog reads the active
categories and brands.

### Limitations (Limitaciones)

The image slots upload to a route (`product-categoria-image`) the server does not have (it registers
`product-category-image`): the record saves with its name, description and the reserved image name,
but the upload fails and the image does not display.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo una marca nueva?` / `¿Las categorías de Mercaderías y Productos son las mismas?` Sí.
- Search terms: `categoría`, `marca`, `crear categoría`, `crear marca`, `imagen de categoría`.

<!-- DOC-ID: capability.import-export -->
## Bulk import and export via Excel (Importar y exportar vía Excel)

### User intention (Intención del usuario)

Load or update many items from a spreadsheet, or download the merchandise catalog.

### Where to find it (Dónde encontrarlo)

On the **Merchandise** view toolbar: the blue upload button opens **Import Products from Excel
(Importar Productos desde Excel)**; the purple download button exports.

### Required information and prerequisites (Requisitos previos)

Export writes only the items listed on this page to `productos.xlsx` with ID, Product, Categories,
Price, Discount, Final Price, and Sub-units — not Brand, Unit, Volume, Weight, or Currency. Import
reads those columns plus Brand, Unit, Volume, Weight, and Currency; headers accept the English or
Spanish name shown in the app.

### Business rules and rationale (Reglas y razón de negocio)

Only ID, Name, Categories, Price, Discount, Final Price, Brand, Unit, Volume, Weight, and Currency
are applied; everything else, the origin included, is kept from the existing product. A row matches
by valid ID ("ID de producto no encontrado" if missing) or else by exact name; an unmatched row
becomes **new merchandise**, because new rows take this page's origin. Matching searches the whole
catalog: a row that matches a made product updates it, and that product stays on Productos.

Unknown category and brand names are created on save. Unit and Currency must match exactly
("Kg"/"g"/"Libras", "PEN"/"USD") or the row is rejected ("unidad no válida" / "moneda no
válida"). The preview highlights the cells that differ from the stored item, and saving is blocked
while any file error or unresolved category or brand remains.

### Result and side effects (Resultado y efectos)

New categories and brands are created first, then items are sent in batches of up to 500, each
subject to the duplicate-name rule.

### Limitations (Limitaciones)

Import cannot change description, sheet, SKU, sub-units, presentations, photos or origin.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo cargo mis mercaderías desde Excel?` / `¿Cómo exporto mis mercaderías?`
- `¿Por qué el Excel no trae la columna Marca?` La exportación no la incluye.
- Search terms: `importar mercaderías`, `exportar mercaderías`, `excel productos`, `plantilla`,
  `unidad no válida`, `moneda no válida`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Mercaderías and Productos are two views of one catalog: names must be unique across both (and
  across active supplies), and categories and brands are shared.
- The origin decides the page: a new item created or imported here is merchandise; changing the
  origin in the form moves it to the other page. Bought → made is refused while the item has
  stock, and made → bought while delivered sales of it are pending regularization. A made product
  changed to bought loses its recipe and production costs.
- One access, **Productos**, covers both Mercaderías and Productos; there is no separate access per
  page.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **An item disappeared from Mercaderías after saving:** its Origin was changed to `Fabricado`; it
  is now on Productos.
- **"tiene stock: no puede pasar a fabricado mientras lo tenga":** the item still has units in some
  warehouse, presentation, lot or serial; its stock must be zero before it can become `Fabricado`.
- **"tiene ventas entregadas pendientes del cierre de producción" when a made product is changed
  back to `Comprado`:** its delivered sales are still **Por regularizar** in Sales Management; the
  change stays refused while they are. Closing their days on **Operaciones → Producción → Cierre
  del día** (or annulling such a sale) clears it.
- **"Ya existe un producto activo con el nombre ...":** a made product, another item or a supply
  already uses that name.
- **"El nombre debe tener al menos 4 caracteres":** lengthen the name.
- **A purchase order refuses a product ("es un producto fabricado por la empresa y no se
  compra"):** the product's origin is `Fabricado`; if it is really bought, change its Origin to
  `Comprado` on the Productos page.
- **A category or brand image does not appear:** known upload issue described above.
- **Import row rejected with "unidad no válida" / "moneda no válida":** use exactly Kg, g, Libras /
  PEN, USD.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Products (Productos)** at `/production/products`: what the company makes, with recipe and
  production cost.
- **Purchase Orders (Órdenes Compra)** at `/logistics/purchase-orders`: buy merchandise from
  suppliers.
- **Stock Changes (Cambios Stock)** at `/logistics/products-stock`: stock quantities and value.
- **Supplies & Materials (Insumos & Materiales)** at `/production/supplies-materials`: items bought
  for internal use or production, not for resale.
- **Point of Sale (Punto de Venta)**: sells merchandise at its price, discount and presentations; a
  product priced in one currency and sold in the other is converted with the rate of **Exchange
  Rate (Tipo de Cambio)**, `/finance/exchange-rate`.
- **Consolidation & Accounts (Consolidado & Cuentas)**: posts merchandise sales and their cost to
  the merchandise accounts.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:4f2d4c16777913fddf196959c4341290e69a12a78323a5b95f8fa5d4738ca34d
    supports: [capability.browse-products, related-pages]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:cb61a6f79a034e4d468b46c173733b815f0af9007a01d1f3e69b92d358913d60
    supports: [page-purpose, capability.browse-products, related-pages]
  - path: frontend/routes/production/merchandise/+page.svelte
    role: page
    hash: sha256:9f0923e4c2f83006660569b948758990cd7152c0333293ae1b617ae49ea0070b
    supports: [page-purpose, capability.browse-products]
  - path: frontend/routes/production/products/ProductCatalog.svelte
    role: user-interface
    hash: sha256:35abd284f9c646be57aa3ae40ec6a643276556d1c1f09dcaf642eadd442221f3
    supports: [page-purpose, concepts, capability.browse-products, capability.create-edit-product, capability.product-sheet, capability.product-photos, capability.delete-product, capability.categories-brands, capability.import-export, rules, troubleshooting]
  - path: frontend/routes/production/products/Attributes.svelte
    role: user-interface
    hash: sha256:73cee110d9ffb176c6729689a8233312be613ec7712170e7d10d7a0cd7ae1740
    supports: [concepts, capability.presentations]
  - path: frontend/routes/production/products/CategoriesBrands.svelte
    role: user-interface
    hash: sha256:d0e846a48d6e206c147dca80628de184db239f0c30a9b8165de5d4fe53c00bd4
    supports: [capability.categories-brands, troubleshooting]
  - path: frontend/routes/production/products/products.excel.ts
    role: frontend-service
    hash: sha256:9c41ea0f24f918edec706063e38c105e389c34454b6a8c1d09c95a1464f62362
    supports: [capability.import-export, troubleshooting]
  - path: frontend/services/production/products.svelte.ts
    role: frontend-service
    hash: sha256:5d63b845bd08143f60d893e7a8a61fde34df5177c18d8a147afdba524d12847d
    supports: [concepts, capability.browse-products, capability.create-edit-product, capability.presentations, rules]
  - path: frontend/core/products-lists.ts
    role: shared-domain
    hash: sha256:8259b07d640e22b297c675c28abde05da72ae96111e15dd9eb35f96b8b408524
    supports: [concepts, capability.create-edit-product, capability.import-export, troubleshooting]
  - path: frontend/services/business/shared-lists.svelte.ts
    role: frontend-service
    hash: sha256:af5729ff929326c970bbe6cfc505f13446ff26f74f1acffe4677c904858f9940
    supports: [concepts, capability.categories-brands, capability.import-export]
  - path: frontend/routes/sales/sale_order_create/sale_order.svelte.ts
    role: shared-domain
    hash: sha256:38d8fe131a97b9a7755ddefe4c5c56be576bdc1369cda4b5d73ff374816ace9f
    supports: [concepts, related-pages]
  - path: frontend/routes/accounting/accounts-consolidation/accounts-consolidation.ts
    role: shared-domain
    hash: sha256:f5fed6c413e71f5692cb726187cf810351aaa8577d6a8a18cf79cf3dc180f7fc
    supports: [concepts, related-pages]
  - path: backend/production/main.go
    role: backend-handler
    hash: sha256:4bb3d1501a006160150bce4e4cc9ba9fbebe0f4b3397dde4ae26da47446ef5d1
    supports: [capability.categories-brands, troubleshooting]
  - path: backend/production/products.go
    role: backend-handler
    hash: sha256:2759e1f9997be3b2ad3666393080d64980bf029581910d05807cb42fae736b03
    supports: [concepts, capability.create-edit-product, capability.presentations, capability.product-sheet, capability.product-photos, capability.delete-product, capability.import-export, rules, troubleshooting]
  - path: backend/production/types/product.go
    role: data-model
    hash: sha256:8c27ae40434282c1bb0eaf3ffd371391288915e053dda42fa7072d298fae1bf6
    supports: [concepts, capability.create-edit-product, capability.presentations]
  - path: backend/production/types/product_production.go
    role: business-logic
    hash: sha256:dbfd1a1b4ef1d848c9b56f06b4b2ce3ecd10a5c07ce66fad45b626109c672901
    supports: [capability.create-edit-product]
  - path: backend/core/quantity.go
    role: business-logic
    hash: sha256:8ec1551b25eaa108290b9ad21d3186fe7b44bb7073cac0909aa8cd2c1be26c20
    supports: [concepts, capability.create-edit-product]
  - path: backend/business/shared-lists.go
    role: backend-handler
    hash: sha256:d19e00bbf01c088110214585e6bded3aa297d32c0361492e97e41f0ed43c47b9
    supports: [capability.categories-brands, troubleshooting]
  - path: backend/business/ecommerce-catalog.go
    role: business-logic
    hash: sha256:af0b7da6455ac1ea1ffd21bb335b2f8164eb04daac90403a12a010a93407cd22
    supports: [capability.create-edit-product, capability.delete-product, capability.categories-brands]
  - path: backend/logistics/types/web_sellable.go
    role: business-logic
    hash: sha256:86e47492cdd858a77ec4502047f4c9929d951e875b9cc93bf0fc770f7d5524ab
    supports: [capability.create-edit-product]
  - path: backend/sales/web_checkout.go
    role: business-logic
    hash: sha256:d1b7bb9dcf20725e6de006a5a1b42bd4e9fa168077cca71a4c83e5087aa266c8
    supports: [capability.create-edit-product]
  - path: backend/sales/sale_order_create.go
    role: business-logic
    hash: sha256:0601dd000d061f432f4a7dfa0228d673b2debc0a744448d6ddfb78a30978c78d
    supports: [capability.create-edit-product]
  - path: backend/sales/types/sale_stock_check.go
    role: business-logic
    hash: sha256:952f769c018817d9ba53b4055d0b890578bc17ce8536156bc65f756ad15563f9
    supports: [capability.create-edit-product]
  - path: backend/sales/types/sale_currency.go
    role: business-logic
    hash: sha256:d94fcf68ff93412a00250b54e2bb4c2c59865472b1f30d86a21e2eefcc6d5f92
    supports: [concepts, related-pages]
  - path: backend/dining/table_orders.go
    role: business-logic
    hash: sha256:1972ba4ecd49a4b4d39fa89610fee7386c590f59e7076b662607c3c37c63a76a
    supports: [capability.create-edit-product]
  - path: backend/logistics/purchase-order-management.go
    role: business-logic
    hash: sha256:d9c2b28be2e49e0ddd10c677f4f343dc5f79fb2c354e134804be6c621f11dbf4
    supports: [concepts, troubleshooting, related-pages]
  - path: backend/access.toml
    role: permissions
    hash: sha256:c19cf85130065dcc0f839796e1e9b4b98a1b78df465ccb20d2b63b59a1a94e59
    supports: [rules]
  - path: backend/production/product_origin.go
    role: business-logic
    hash: sha256:a53fd15103a1d58292f4b56bef9ccd8d81986bb32959cdd04d3c8a92c11ebc31
    supports: [capability.create-edit-product, rules, troubleshooting]
  - path: backend/sales/sale_orders_status.go
    role: business-logic
    hash: sha256:c34311168921683a414fe2a0bbcf5dd450dfa8af8f5e2963186fb4fdd48c0262
    supports: [capability.create-edit-product, troubleshooting]
  - path: backend/sales/sale_order_annul.go
    role: business-logic
    hash: sha256:5d184227f73c0fc0630fb2357132bf09bc4dd39f2bda73d74d9676b611aa36f7
    supports: [troubleshooting]
```
