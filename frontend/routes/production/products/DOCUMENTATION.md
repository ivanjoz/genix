---
schema: 1
page_id: business.products
route: /production/products
title: Products (Productos)
status: implemented
visibility: tenant
description_en: >-
  Catalog of the products the company makes (finished goods). Create and edit them with price,
  discount, unit, currency, brand, categories, sub-units, presentations, photos and web store
  settings; define each product's recipe of supplies, yield and estimated unit cost; bulk import
  and export via Excel. Also manages product categories and brands.
description_es: >-
  Catálogo de los productos que fabrica la empresa (productos terminados). Crear y editar productos
  con precio, descuento, unidad, moneda, marca, categorías, sub-unidades, presentaciones, fotos y
  ajustes de la tienda web; definir la receta de insumos, el rendimiento y el costo estimado por
  unidad; importar y exportar de forma masiva vía Excel. También administra categorías y marcas.
---

# Products (Productos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Products (`Productos`) is the catalog page for the items the company **makes** itself — finished
goods (`productos terminados`), such as dishes, bakery items or anything produced in-house. Each
product (`producto`) has a name, pricing, unit, currency, brand, categories, optional sub-units and
presentations (variants), photos, web store settings and, only here, a production recipe
(`receta`) with an estimated cost per unit. The same page also manages the two shared catalogs a
product depends on — Categories (`Categorías`) and Brands (`Marcas`).

The catalog is one list split in two pages by the product's origin (`Origen`): this page shows
only made products, while **Merchandise (Mercaderías)** at `/production/merchandise` shows the
products the company buys to resell. Both pages have the same list, form, import and export; only
this page offers the **Production (Producción)** tab.

This page does not show or change stock quantities (`existencias`/`stock`) and does not create
warehouses; supplies (`insumos`) used in recipes are created in **Supplies & Materials (Insumos &
Materiales)**. Pricing, categories and brand configured here are what the Point of Sale, the web
store and accounting read to sell and classify the product.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Origin (`Origen`)** says whether a product is **Bought (merchandise) / Comprado (mercadería)**
  or **Made by the company (finished good) / Fabricado (producto terminado)**. It decides which
  catalog page lists the product (this page lists only `Fabricado`), whether it has a Production
  tab, and how other modules treat it:
  - A made product cannot be bought: a purchase order line with it is refused with
    "es un producto fabricado por la empresa y no se compra".
  - In **Consolidation & Accounts (Consolidado & Cuentas)**, the sale of a made product posts
    revenue to 7022 (Ventas — productos terminados) and its cost of sales to 6922 against stock
    2111 (Productos terminados); merchandise uses 7012, 6912 and 2011 instead.
- A **product (producto)** is one catalog item: name, base price (`Precio Base`), discount
  (`Descuento`), final price (`Precio Final`), currency (`Moneda`, PEN or USD), unit (`Unidad`,
  Kg/g/Libras), brand (`Marca`), one or more categories (`Categorías`), optional sub-units,
  presentations, and photos. The currency and unit lists are fixed values built into the
  frontend, not per-company configuration, and this page never converts amounts between PEN and
  USD. The product's currency says which currency its prices are in; the Point of Sale reads it
  to convert the price when the product is sold in the other currency.
- A **category (categoría)** and a **brand (marca)** are two independent shared lists; a product
  can belong to several categories but only one brand. Both catalog pages use the same lists.
- **Sub-units (`Sub-Unidades`)** let the same product be sold fractionally — a box of six candies
  sold one candy at a time, or a kilogram sold by the gram. They are fields on the product, not a
  separate product or SKU. **`Quantity` (Cantidad) is the divisor: how many sub-units make one
  whole unit** — 6 for a box of six, 1000 for a kilogram in grams. `Name` (Nombre) labels it, and
  `Final Price` (Precio Final) is what one sub-unit costs; a sub-unit is charged at its own price,
  never at a fraction of the whole-unit price.
- **A sub-unit can be added to a product that already has stock and sales**, and nothing is
  restated: existing records simply hold no sub-unit part. Once set, the divisor may only be
  **refined to a multiple of itself** (6 → 12 is accepted, 12 → 6 and 6 → 5 are rejected) and it
  can no longer be removed, because stored stock and movements are expressed in it. The maximum
  divisor is 1000.
- A **presentation (`Presentación`)**, also called a variant, is a product option distinguished by
  a fixed attribute (`Atributo`: Color, Talla, Tamaño, Forma, or Presentación), with its own name,
  price, price difference, optional SKU, and color swatch. Presentations are stored inside the
  product, not as separate catalog rows, and are only persisted when the whole product is saved.
- A **recipe (`Receta`)** is the list of supplies (`insumos`) and quantities needed to make a
  number of portions (`Rinde (porciones)`) of a made product. Together with an extra cost per unit
  (`Costo extra por unidad`) and an optional standard cost (`Costo estándar`) it forms the
  product's production configuration, edited on the Production tab.
- The **main image (`Imagen Principal`)** is the first photo shown for a product; **additional
  photos (`Fotos`)** are the rest.

<!-- DOC-ID: capability.browse-products -->
## Browse and switch views (Buscar y cambiar de vista)

Open **Commercial (Comercial) → Catalog (Catálogo) → Products (Productos)** or **Operations
(Operaciones) → Production (Producción) → Products (Productos)**; both menus list **Mercaderías**
first and then **Productos**, and both open `/production/products`. The page title and the first
tab of the top strip read **Products (Productos)**; the strip switches between three views:
**Products (Productos)**, **Categories (Categorías)**, and **Brands (Marcas)**. Switching views
closes any open side layer and resets the working form.

The Products list shows only made products (`Origen = Fabricado`). Bought products are never
listed here, even when their name matches the filter — look for them in **Mercaderías**. The filter
box ("Filter products|Filtrar productos") narrows the loaded list by product name (Products view)
or by name/description (Categories and Brands views); it is a local filter, not a server search.

The table shows ID, product name, category names, price, discount, final price, and the sub-unit
summary (`quantity x unit`). Products are listed most recently created first. Selecting a row
opens it in the side layer described below.

<!-- DOC-ID: capability.create-edit-product -->
## Create or edit a product (Crear o editar un producto)

### User intention (Intención del usuario)

Add a new made product to the catalog, or update an existing product's price, categorization,
origin, description, sub-units, presentations, photos or web store settings.

### Where to find it (Dónde encontrarlo)

On the **Products** view, use the green **New (Nuevo)** button to open the side layer for a new
product, or click an existing row to open it pre-filled. The layer has the tabs **Info
(Información)**, **Sheet (Ficha)**, **Presentations (Presentaciones)**, **Photos (Fotos)** and,
for a made product, **Production (Producción)**. Save with the layer's Save action.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)** is required, at least 4 characters (checked by the form and again by the
  server).
- **Base Price (Precio Base)**, **Discount (Descuento)**, and **Final Price (Precio Final)** are
  linked: changing one recalculates the others with
  `Final Price = Base Price × (100 − Discount%) / 100`. Discount must stay under 100% (checked only
  by the form).
- **Origin (Origen)** starts as **Fabricado (producto terminado)** for a product created from this
  page. Changing it to **Comprado (mercadería)** and saving moves the product to the
  **Mercaderías** page, removes the Production tab and drops the product's recipe and production
  costs. A product saved without an origin is stored as bought. The server refuses an origin change
  in two cases:
  - **Fabricado → Comprado** while delivered sales still have lines of this product waiting for the
    production close ("tiene ventas entregadas pendientes del cierre de producción: ciérrelas antes
    de cambiarlo a comprado"). These are the sales listed as **Por regularizar** in Sales
    Management. Such a sale stops blocking the change once its day is closed on **Operaciones →
    Producción → Cierre del día**, or when the sale is annulled.
  - **Comprado → Fabricado** while the product has stock in any warehouse, in any presentation,
    lots and serial numbers included ("tiene stock: no puede pasar a fabricado mientras lo tenga").
- **Currency (Moneda)**, **Unit (Unidad)**, **Weight (Peso)**, **Volume (Volumen)**, **Brand
  (Marca)**, **SKU**, and **Short Description (Descripción Corta)** are optional and are shown again
  when the product is reopened.
- The **SKU Individual** checkbox stores a flag on the product, but nothing in Genix currently
  reacts to it.
- **Categories (`CATEGORÍAS ::`)** attaches one or more existing categories by name; it cannot
  create a new category (use the Categories view or the Excel import).
- The gray **Tienda Web** box holds the web store settings:
  - **Allowed overselling (`Sobreventa permitida`)** is a number of whole units that may be sold
    beyond the stock. With 5 units in stock and 100 here, the web store accepts up to 105 units in
    total. Orders not yet delivered (till sales, web orders and open restaurant table orders) count
    as reserved and use up the margin: once 105 are pending, nothing more can be ordered online. The
    value cannot be negative. For a made product the Point of Sale applies the same rule: a sale is
    accepted only while the warehouse's stock, less the reserved units, plus this margin covers it;
    otherwise it is refused with "no tiene stock ni sobreventa permitida suficiente en el almacén". A
    sale of a made product also cannot name a lot or serial number.
  - **Hide (`Ocultar`)** removes the product from the web store catalog, and checkout refuses it
    ("ya no está disponible en la tienda") even if it is still in a shopper's cart.
- **Sub-Unidades** (Name, Base Price, Discount, Final Price, Quantity) are on the
  **Presentations (Presentaciones)** tab, below the presentations editor. They do not recalculate
  each other. The sub-unit **Base Price** field recalculates the *main* product's Final Price from
  the main Price and Discount, not a sub-unit total.
- Saving is rejected with a descriptive message when the sub-unit **Quantity** (the divisor) is
  above 1000, when a sub-unit is defined without a **Name**, when the divisor is changed to a value
  that is not a multiple of the current one, or when a sub-unit that was already set is removed.
- For a made product, the Production tab must pass its own checks (see the recipe capability).

### Business rules and rationale (Reglas y razón de negocio)

The server rejects a name already used by another product that is active — including a supply of
Supplies & Materials — with "Ya existe un producto activo con el nombre ...". If the name belongs
only to a deleted product, saving a new product with that name reuses the deleted product's ID, so
its history resurfaces under the new save (with the origin chosen now). Two rows with the same name
in one save are refused ("Hay nombres duplicados en la solicitud").

### Result and side effects (Resultado y efectos)

Saving creates or updates the product. A new product is active immediately; editing keeps the
stored stock figures, images and creation data untouched. The web store catalog picks up the change
on its next sync: it lists an active, not hidden product only while it has something sellable
(stock in the warehouses open to online sales, or overselling margin left). The recipe and its
costs, the overselling margin and the accounting fields are never sent to the public web store.

### Limitations (Limitaciones)

- There is no server-side range check on Discount, Price, or Final Price beyond the form's
  "< 100%" discount check.
- The page does not show stock; use **Stock Changes (Cambios Stock)**.
- A delivered sale of a made product does not take its units out of stock at delivery: they stay
  counted as reserved, so they keep using stock and the `Sobreventa permitida` margin, and the sale
  is listed as **Por regularizar** in Sales Management until the **Cierre del día** (daily
  production close, `/production/daily-close`) of that day consumes the recipe's supplies, enters
  the product and takes the sold units out of stock.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo un producto fabricado?` / `¿Cómo edito el precio o el descuento de un producto?`
- `¿Dónde está mi producto? No aparece en Productos.` Si su origen es `Comprado (mercadería)`, está
  en Mercaderías.
- `¿Cómo paso un producto de Mercaderías a Productos?` Cambie el campo Origen a `Fabricado` y
  guarde.
- `¿Por qué cambió el Precio Final si sólo toqué la Sub-Unidad?` El campo "Precio Base" de la
  sub-unidad recalcula el Precio Final del producto principal.
- `¿Por qué no puedo vender en el POS un producto fabricado sin stock?` Para un producto fabricado
  el POS acepta la venta sólo con stock libre o `Sobreventa permitida` suficiente.
- Search terms: `producto`, `producto terminado`, `fabricado`, `origen`, `precio base`,
  `descuento`, `precio final`, `sub-unidad`, `SKU`, `moneda`, `unidad`, `sobreventa`, `ocultar`.

<!-- DOC-ID: capability.production-recipe -->
## Define the recipe and production cost (Definir receta y costo de producción)

### User intention (Intención del usuario)

Record what it takes to make a product — which supplies (`insumos`) and how much of each for a
batch of portions — and see what one unit costs, either from the supplies' current average cost or
from a fixed standard cost (`costo estándar`).

### Where to find it (Dónde encontrarlo)

Open a product and select the **Production (Producción)** tab. It appears only while the form's
Origin is **Fabricado (producto terminado)**.

### Required information and prerequisites (Requisitos previos)

- Supplies must already exist as active supplies in **Supplies & Materials (Insumos &
  Materiales)**. A supply that should be used by fractions (grams of a kilogram) needs its
  **Sub-unit (Sub-unidad)** and **Sub-units per unit (Sub-unidades por unidad)** set there.
- The supply's **Default account (Cuenta contable por defecto)** must be a purchase account that
  has an inventory (`existencias`) account, such as 6031 Materiales auxiliares or 6032 Suministros.
  A supply on an expense account (656) or a fixed-asset account (33x) cannot be in a recipe.
- **Yield (portions) / Rinde (porciones)**: how many units of the product the recipe makes; the
  supply quantities are for that many units (for example "20 porciones: 1 kg de arroz, 50 g de
  sal"). It is required as soon as the recipe has a supply.
- **Extra cost per unit / Costo extra por unidad** (labor, overhead) and **Standard cost / Costo
  estándar** are optional and cannot be negative. Amounts are in soles (PEN), net of IGV.
- **Recipe (Receta)**: the green **Supply (Insumo)** button adds an empty line; each line has
  **Supply (Insumo)**, **Units (Unidades)**, **Sub-units (Sub-unidades)** — editable only when the
  supply has a sub-unit —, **Unit (Unidad)** (the supply's unit and sub-unit, for example
  `Kg · 1000 g`) and **Est. cost (Costo est.)**. The red delete icon removes a line.

### Business rules and rationale (Reglas y razón de negocio)

- Quantities are normalized: typing 1500 g on a kilogram supply becomes 1 kg + 500 g. Choosing a
  different supply on a line keeps the whole units and drops the sub-units, because they were in
  the previous supply's sub-unit.
- **Est. cost (Costo est.)** values each line at the supply's average cost over every warehouse —
  the total stock value of the supply divided by the units held. A supply with no units or no
  stock value shows **sin costo**.
- **Estimated cost per unit (Costo estimado por unidad)**: when a standard cost is set it is the
  unit cost, labeled `Costo estándar`, whatever the supplies cost. Otherwise it is the recipe's
  value ÷ yield + extra cost per unit; if some line has no cost, it is left out and the box warns
  **Incompleto: algún insumo no tiene costo**. With neither a standard cost nor a recipe with yield
  the box shows **Sin fuente de costo: agregue insumos o un costo estándar.**
- On Save the form checks the recipe first and, if something is wrong, shows the message and
  returns to the Production tab: "Los costos de producción no pueden ser negativos.", "Indique
  cuántas porciones rinde la receta.", "Elija el insumo de cada línea de la receta.", "Un insumo
  está repetido en la receta." or "Cada línea de la receta necesita una cantidad."
- The server re-checks the recipe and refuses it when the product is not made ("solo un producto
  fabricado tiene configuración de producción"), when a supply does not exist or is not active,
  is repeated, has an invalid quantity, uses a sub-unit divisor different from the supply's current
  one ("usa el divisor X, pero el insumo tiene Y"), or has an account without inventory ("que no es
  de existencias; no puede consumirse en producción").

### Result and side effects (Resultado y efectos)

The recipe and costs are saved with the product when the whole product is saved; closing the layer
without saving discards the edits. Saving the product with Origin `Comprado` drops its recipe and
production costs. Saving them does not consume supplies, move stock or post
accounting entries. They are private: the public web store never receives them.

### Limitations (Limitaciones)

- The estimate is informative; it uses the supplies' current average cost, which changes as
  supplies are bought and used.
- A supply can appear only once per recipe, and the recipe has no versions or alternatives.
- The tab is not available for merchandise.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo registro la receta de un producto?` / `¿Dónde pongo los insumos de un plato?`
- `¿Cuánto me cuesta producir una unidad?` Mire "Costo estimado por unidad" en la pestaña
  Producción.
- `¿Por qué el costo dice "sin costo"?` El insumo no tiene stock valorizado en ningún almacén.
- `¿Para qué sirve el costo estándar?` Si lo indica, reemplaza el costo calculado de la receta.
- Search terms: `receta`, `insumos`, `rendimiento`, `rinde`, `porciones`, `costo de producción`,
  `costo unitario`, `costo estándar`, `costo extra`, `mano de obra`, `escandallo`, `ficha técnica`.

<!-- DOC-ID: capability.product-sheet -->
## Edit the product sheet (Editar la ficha del producto)

The **Sheet (Ficha)** tab opens a rich text editor for a longer, formatted description, separate
from the short "Descripción Corta" on the Info tab. It is only saved when the product is saved. The
web store's product detail shows this sheet unless the product is hidden.

<!-- DOC-ID: capability.presentations -->
## Manage presentations / variants (Gestionar presentaciones)

### User intention (Intención del usuario)

Offer the same product in several variants distinguished by an attribute — for example different
colors or sizes — each with its own name, price, price difference, and SKU.

### Where to find it (Dónde encontrarlo)

Inside the product layer, open the **Presentations (Presentaciones)** tab. Use the green add button
to open the **Producto Presentación** modal, or use a row's edit button to change it.

### Required information and prerequisites (Requisitos previos)

**Attribute (Atributo)** must be one of a fixed list — Color, Talla, Tamaño, Forma, or
Presentación — which is the same for every company. **Name (Nombre)** is free text. **Price
(Precio)**, **Price Difference (Diferencia Precio)**, **SKU**, and **Color** are optional. A new
presentation's attribute starts empty; pick it manually.

### Business rules and rationale (Reglas y razón de negocio)

Presentations become permanent only when the whole product is saved (the modal warns: "La
información se guardará cuando se guarde el producto."). Deleting an already saved presentation
marks it inactive instead of removing it, because sales may already reference it; deleting one
added in the same editing session removes it outright.

### Result and side effects (Resultado y efectos)

On save, the server matches the submitted presentations with the stored ones by attribute and
name: matching presentations keep their IDs, new ones get new IDs, and stored presentations missing
from the submission are kept as inactive.

### Limitations (Limitaciones)

- Presentation prices do not take part in the main Price/Discount/Final Price calculation.
- There is no screen to add a sixth attribute type.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo agrego variantes de color o talla a un producto?`
- `¿Se guardan las presentaciones aunque no guarde el producto?` No, sólo al guardar el producto.
- Search terms: `presentación`, `variante`, `atributo`, `color`, `talla`, `diferencia de precio`.

<!-- DOC-ID: capability.product-photos -->
## Upload and remove photos (Subir y eliminar fotos)

### User intention (Intención del usuario)

Attach one main image and any number of additional photos to a product.

### Where to find it (Dónde encontrarlo)

The main image uploader is on the **Info** tab, next to the pricing fields. Additional photos are
managed on the **Photos (Fotos)** tab.

### Required information and prerequisites (Requisitos previos)

The **Photos** tab uploader appears only once the product has been saved; before that the tab
shows "Guarde el producto para poder cargar más imágenes". The main-image picker on the Info tab
works for a new product too.

### Business rules and rationale (Reglas y razón de negocio)

For a new product, the picked main image is kept in the form and uploaded automatically right
after the first save gives the product its ID. For an existing product, a confirmed image is added
at once and uploads in the background, independent of the Save button; its progress and any
failure appear in the header's process tray. Removing a photo from the Photos tab, after its
confirmation, is applied immediately and does not wait for Save.

### Result and side effects (Resultado y efectos)

A confirmed upload is placed first in the product's image list and becomes the main image.
Removing the main image promotes the next remaining photo, if any.

### Limitations (Limitaciones)

There is no manual reordering of photos and no "set as main" action for an existing photo; the main
image is always the most recently confirmed upload, or whatever remains first after a removal.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué no puedo subir más fotos a un producto nuevo?` Guarde el producto primero.
- `¿Cómo elimino una foto?` En la pestaña Fotos, use el ícono de eliminar sobre la imagen y
  confirme; se borra de inmediato.
- Search terms: `foto`, `imagen principal`, `subir imagen`, `eliminar imagen`.

<!-- DOC-ID: capability.delete-product -->
## Remove a product (Eliminar un producto)

Open an existing product and use the layer's **Delete (Eliminar)** action; after the "¿Está seguro
que desea eliminar ...?" confirmation the product is saved as inactive. It disappears from the list
and from the web store, but its data is kept and its name becomes free for reuse. Images are not
processed on a delete.

### Limitations (Limitaciones)

There is no reactivation action; saving a new product with the same name reuses the deleted
product's ID. The recipe is not checked on delete, so a recipe naming a supply that is no longer
active does not block it.

<!-- DOC-ID: capability.categories-brands -->
## Manage categories and brands (Gestionar categorías y marcas)

### User intention (Intención del usuario)

Maintain the two shared lists every product is organized by: product categories and brands.

### Where to find it (Dónde encontrarlo)

Switch the top strip to **Categories (Categorías)** or **Brands (Marcas)**. Both views show cards and
use the same **CATEGORÍA** modal; use the green **New (Nuevo)** button or click a card. The lists
are the same ones the Mercaderías page and Supplies & Materials use.

### Required information and prerequisites (Requisitos previos)

**Name (Nombre)** is required (at least 4 characters in the form); **Description (Descripción)** is
optional. Up to three images can be attached.

### Business rules and rationale (Reglas y razón de negocio)

The server rejects a name already used by another active record of the same list ("El nombre ... ya
existe en la lista ..."); Categories and Brands are separate lists, so a name can exist once in each.
If the name belongs to a deleted record of that list, saving reuses its ID. Renaming a brand also
updates the brand name stored on its products, so a product search by the new brand name finds them.

### Result and side effects (Resultado y efectos)

Saving updates the shared list used by both catalog pages. The web store catalog reads the active
categories and brands.

### Limitations (Limitaciones)

The image slots send their upload to a route (`product-categoria-image`) that the server does not
have (it registers `product-category-image`). The category or brand still saves with its name,
description and the reserved image name, but the upload fails, so the image does not display.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo una categoría o marca nueva?`
- `¿Por qué no se ve la imagen de una categoría o marca?` La subida de imágenes de categorías y
  marcas falla actualmente.
- Search terms: `categoría`, `marca`, `crear categoría`, `crear marca`, `imagen de categoría`.

<!-- DOC-ID: capability.import-export -->
## Bulk import and export via Excel (Importar y exportar vía Excel)

### User intention (Intención del usuario)

Load or update many products at once from a spreadsheet, or download this page's catalog.

### Where to find it (Dónde encontrarlo)

On the **Products** view toolbar: the blue upload button opens **Import Products from Excel
(Importar Productos desde Excel)**; the purple download button exports directly.

### Required information and prerequisites (Requisitos previos)

Export writes only the products listed on this page (made products) to `productos.xlsx`, with the
table columns: ID, Product, Categories, Price, Discount, Final Price, and Sub-units. It does
**not** include Brand, Unit, Volume, Weight, or Currency.

Import reads the same columns plus Brand, Unit, Volume, Weight, and Currency (shown only in the
import preview). Headers accept the English or Spanish name shown in the app (for example "Price" or
"Precio").

### Business rules and rationale (Reglas y razón de negocio)

Only these fields are applied from an import: ID, Name, Categories, Price, Discount, Final Price,
Brand, Unit, Volume, Weight, and Currency. Everything else — description, sheet, SKU, sub-units,
presentations, photos, origin, recipe — is kept from the existing product.

A row is matched by ID when it has a valid ID ("ID de producto no encontrado" if it does not exist),
otherwise by exact name; an unmatched row becomes a **new made product**, because new rows take
this page's origin. Matching searches the whole catalog, not only this page: a row that matches a
merchandise product updates that product, which keeps its origin and stays on Mercaderías.

Category and Brand names not found are created as new records when the import is saved. Unit and
Currency must match the fixed options exactly ("Kg"/"g"/"Libras", "PEN"/"USD") or the row is
rejected with "unidad no válida" / "moneda no válida". The preview highlights only the cells whose
value differs from the stored product. Saving is blocked while any file error remains or a category
or brand could not be resolved.

### Result and side effects (Resultado y efectos)

On save, new categories and brands are created first, then products are sent in batches of up to
500, each subject to the same duplicate-name rule as a manual save.

### Limitations (Limitaciones)

Import cannot change description, sheet, SKU, sub-units, presentations, photos, origin or the
recipe. Export does not include Brand, Unit, Volume, Weight, or Currency.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo importo productos desde Excel?` / `¿Cómo exporto el catálogo a Excel?`
- `¿Por qué el Excel exportado no trae mis mercaderías?` Cada página exporta sólo sus productos.
- `¿Qué pasa si pongo una marca que no existe?` Se crea al guardar la importación; con Unidad o
  Moneda inválidas, en cambio, la fila se rechaza.
- Search terms: `importar productos`, `exportar productos`, `excel productos`, `plantilla`,
  `marca no encontrada`, `unidad no válida`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Mercaderías and Productos are two views of one catalog: names must be unique across both (and
  across active supplies), and categories and brands are shared.
- The origin decides the page: a new product created or imported here is made; changing the
  origin in the form moves the product to the other page. Made → bought is refused while delivered
  sales of the product are pending regularization, and bought → made while it has stock.
- Only made products have a recipe and production cost.
- One access, **Productos**, covers both Mercaderías and Productos; there is no separate access per
  page. It also covers reading the supplies list used by the Production tab.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **A product disappeared from Productos after saving:** its Origin was changed to `Comprado
  (mercadería)`; it is now on Mercaderías.
- **"tiene ventas entregadas pendientes del cierre de producción" when changing a product to
  `Comprado`:** delivered sales of the product are still **Por regularizar** in Sales Management;
  the origin cannot change while they are. Close their days on **Cierre del día** first. The
  refused save leaves the recipe in the form, so setting Origin back to `Fabricado` keeps it.
- **"tiene stock: no puede pasar a fabricado mientras lo tenga" when changing to `Fabricado`:** the
  product still has units in some warehouse, presentation, lot or serial. Its stock must be zero
  first (see Stock Changes).
- **"usa el divisor X, pero el insumo tiene Y":** the supply's sub-unit changed after the line was
  added. Remove that line, add the supply again and re-enter its quantity.
- **"que no es de existencias; no puede consumirse en producción":** change the supply's default
  account in Supplies & Materials to an inventory purchase account, or use another supply.
- **"el insumo ... de la receta no existe o no está activo":** remove the line of the deleted
  supply.
- **"Ya existe un producto activo con el nombre ...":** another product, merchandise or supply
  already uses that name.
- **"El nombre debe tener al menos 4 caracteres":** lengthen the product, category, or brand name.
- **A category or brand image does not appear:** known upload issue described above.
- **A deleted product "comes back" under the same name:** a new save with that name reuses the
  deleted record.
- **Import row rejected with "unidad no válida" / "moneda no válida":** use exactly Kg, g, Libras /
  PEN, USD.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Merchandise (Mercaderías)** at `/production/merchandise`: the bought products; same form,
  without the Production tab.
- **Supplies & Materials (Insumos & Materiales)** at `/production/supplies-materials`: create the
  supplies used in recipes, their sub-units and default account.
- **Stock Changes (Cambios Stock)** at `/logistics/products-stock`: stock quantities and value.
- **Purchase Orders (Órdenes Compra)**: buy merchandise and supplies; made products are refused.
- **Sales Management (Gestión Ventas)** at `/sales/sale_orders_status`: its **Por regularizar**
  tab lists the delivered sales of made products whose stock exit is still pending.
- **Daily close (Cierre del día)** at `/production/daily-close`: produces what was sold from this
  page's recipes and costs, consuming the supplies and settling the pending sales.
- **Point of Sale (Punto de Venta)**: sells these products at their price, discount and
  presentations; a product priced in one currency and sold in the other is converted with the
  exchange rate of **Exchange Rate (Tipo de Cambio)**, `/finance/exchange-rate`.
- **Consolidation & Accounts (Consolidado & Cuentas)**: posts made products to the finished goods
  accounts.

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
  - path: frontend/routes/production/products/+page.svelte
    role: page
    hash: sha256:c640063619e07ca13708faf767b476370062841cfc839509765cec104cc716ef
    supports: [page-purpose, capability.browse-products]
  - path: frontend/routes/production/products/ProductCatalog.svelte
    role: user-interface
    hash: sha256:35abd284f9c646be57aa3ae40ec6a643276556d1c1f09dcaf642eadd442221f3
    supports: [page-purpose, concepts, capability.browse-products, capability.create-edit-product, capability.production-recipe, capability.product-sheet, capability.product-photos, capability.delete-product, capability.categories-brands, capability.import-export, rules, troubleshooting]
  - path: frontend/routes/production/products/ProductProductionTab.svelte
    role: user-interface
    hash: sha256:f0b6cd77f7bb03b4784d1c3988771976036f17a6636b176b6d0e2477f4205d5d
    supports: [concepts, capability.production-recipe, troubleshooting]
  - path: frontend/routes/production/products/product-production.ts
    role: business-logic
    hash: sha256:3430fe9a1e479a4b4d4a85e317427e90e3e9448c7c57c0a43f98ccf11d9757f9
    supports: [capability.production-recipe, troubleshooting]
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
    supports: [concepts, capability.browse-products, capability.create-edit-product, capability.production-recipe, capability.presentations, rules]
  - path: frontend/routes/production/supplies-materials/supply-material.svelte.ts
    role: frontend-service
    hash: sha256:de1599acfcae19c7ff9901773449a5f29b76110988c5ac4f3b58ecd08e005545
    supports: [capability.production-recipe]
  - path: frontend/routes/production/supplies-materials/+page.svelte
    role: user-interface
    hash: sha256:8c6c1f347aeb695691f3cdd4b1b1ec36333ed88f275d6d15363c44e6cf242878
    supports: [capability.production-recipe, troubleshooting, related-pages]
  - path: frontend/routes/logistics/products-stock/stock-movement.ts
    role: frontend-service
    hash: sha256:5b40e3b6c6ef8421e4a341e887d67b3bece49badd16259898d659e778ce06115
    supports: [capability.production-recipe]
  - path: frontend/core/products-lists.ts
    role: shared-domain
    hash: sha256:8259b07d640e22b297c675c28abde05da72ae96111e15dd9eb35f96b8b408524
    supports: [concepts, capability.create-edit-product, capability.production-recipe, capability.import-export, troubleshooting]
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
    supports: [concepts, capability.create-edit-product, capability.production-recipe, capability.presentations, capability.product-photos, capability.delete-product, capability.import-export, rules, troubleshooting]
  - path: backend/production/types/product.go
    role: data-model
    hash: sha256:8c27ae40434282c1bb0eaf3ffd371391288915e053dda42fa7072d298fae1bf6
    supports: [concepts, capability.create-edit-product, capability.presentations]
  - path: backend/production/types/product_production.go
    role: business-logic
    hash: sha256:dbfd1a1b4ef1d848c9b56f06b4b2ce3ecd10a5c07ce66fad45b626109c672901
    supports: [concepts, capability.production-recipe, capability.delete-product, rules, troubleshooting]
  - path: backend/accounting/types/pcge_accounts.go
    role: data-model
    hash: sha256:7b2044550d593ba5a579cc55ebcb5a7971bcb1ceb89734d0717ad4a5dce71204
    supports: [capability.production-recipe]
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
  - path: backend/sales/saleissue/types/new_sale.go
    role: business-logic
    hash: sha256:79480f8435d4182f16c8ebca2b5717be396423cd73a571f5ef1e2a84a229e33e
    supports: [capability.create-edit-product]
  - path: backend/sales/types/sale_actions.go
    role: business-logic
    hash: sha256:ad4c117e0bce8f4c8e88f85e6aa3677090935cf6b6127c944f72bf1601413109
    supports: [capability.create-edit-product]
  - path: backend/dining/table_orders.go
    role: business-logic
    hash: sha256:1972ba4ecd49a4b4d39fa89610fee7386c590f59e7076b662607c3c37c63a76a
    supports: [capability.create-edit-product]
  - path: backend/sales/types/pending_delivery.go
    role: business-logic
    hash: sha256:fdefbd8a318a6cafff7823707c0c42bbf28c5ca1077b9b54dccd150094e009fa
    supports: [capability.create-edit-product]
  - path: backend/logistics/purchase-order-management.go
    role: business-logic
    hash: sha256:d9c2b28be2e49e0ddd10c677f4f343dc5f79fb2c354e134804be6c621f11dbf4
    supports: [concepts, related-pages]
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
    supports: [capability.create-edit-product, related-pages]
  - path: backend/sales/sale_order_annul.go
    role: business-logic
    hash: sha256:5d184227f73c0fc0630fb2357132bf09bc4dd39f2bda73d74d9676b611aa36f7
    supports: [capability.create-edit-product]
  - path: backend/production/daily_close.go
    role: business-logic
    hash: sha256:7ae0b03e927362b3c467d29657303b41f9f0dbc979ad54b92f79807ab9b025a1
    supports: [capability.create-edit-product, troubleshooting, related-pages]
```
