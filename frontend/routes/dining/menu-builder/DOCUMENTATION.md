---
schema: 1
page_id: dining.menu-builder
route: /dining/menu-builder
title: Menu Builder (Carta)
status: implemented
visibility: tenant
description_en: >-
  Build the menu (carta) the restaurant's customers see when they scan a table's QR code. Drag
  blocks onto a phone-sized preview: every product grouped by category, one category, hand-picked
  products, titles, subtitles, texts, images and dividers. Set the menu's name, accent color,
  background color and font. Create several menus; each dining layout shows the one chosen in its
  settings.
description_es: >-
  Arma la carta que ven los clientes del restaurante al escanear el QR de una mesa. Arrastra bloques
  sobre una vista del celular: todos los productos agrupados por categoría, una categoría, productos
  puntuales, títulos, subtítulos, textos, imágenes y divisores. Define el nombre de la carta, el
  color de acento, el color de fondo y la fuente. Se pueden crear varias cartas; cada salón muestra
  la que se elige en sus parámetros.
---

# Menu Builder (Carta)

<!-- DOC-ID: page-purpose -->
## Page purpose

`Carta` is where the restaurant composes its menu. The page has three columns: on the left the
menu picker and everything that can be dragged (`Bloques`, `Productos`, `Categorías`); in the center
a 390px phone preview of the menu; on the right the settings of the selected block, or of the menu
itself when no block is selected.

A new menu starts as one block, **Todos los productos**, so the whole catalog shows at once and the
user builds from there. Only products whose `Mostrar en` includes **Restaurante** (or has no channel
chosen) appear. Prices are shown in soles; a product priced in dollars is restated at the day's
exchange rate, and one with no rate available is left out.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Menu (`carta`):** a named, ordered list of blocks plus a theme. A company can have many.
- **Block kinds (`bloques`):**
  - **Todos los productos:** every restaurant product, with a heading per category (a product goes
    under its first category; products with none go last under `Otros`). The option **Solo los
    productos que no están en otro bloque** leaves out what a `Categoría` or `Productos puntuales`
    block of the same menu already lists.
  - **Categoría:** one category's products, under the category's name or the heading typed in
    `Encabezado (opcional)`.
  - **Productos puntuales:** products picked one by one, in the order given.
  - **Título**, **Subtítulo**, **Texto:** text with left or center alignment.
  - **Imagen:** a full-width image from the gallery or the stock images, with an optional caption.
  - **Divisor:** a line in the accent color.
- **Style (`Estilo`) of the product blocks:** `Lista compacta` (name and price on one line),
  `Lista con foto` (thumbnail, name, description, price) or `Tarjetas (2 columnas)`.
- **Theme:** `Color de acento` (titles, prices, lines), `Color de fondo` (the text color is chosen
  automatically to be readable on it) and `Fuente` (Sans, Serif, Redondeada).
- **Menu of a layout:** chosen in `Diseño de salones` > `Parámetros` > `Carta que se muestra en el
  QR`. A layout with none chosen shows the default menu: all the restaurant's products.

<!-- DOC-ID: workflow-build-menu -->
## Build and save a menu

1. Open **Restaurante > Carta**. Pick a menu in `Carta`, or press the green **+** to start a new one.
2. Drag from the left panel onto the phone:
   - a block tile (`Bloques`) to add that kind of block;
   - a product (`Productos`): dropped between blocks it becomes a new `Productos puntuales` block;
     dropped **on** an existing `Productos puntuales` block (it lights up green) it joins that block;
   - a category (`Categorías`) to add a `Categoría` block.
   A blue dashed line shows where a block will land.
3. Drag a block on the phone to reorder it. Click a block to edit it on the right; press
   `Suprimir` (Delete) or **Quitar** to remove the selected block.
4. With no block selected, the right panel edits the menu's `Nombre` and theme.
5. Press **Guardar**. `Cambios sin guardar` shows while the edits are not saved; switching menu or
   starting a new one asks before discarding them.

Saving is refused, with a message naming the block, when a `Categoría` block has no category, a
`Productos puntuales` block has no product, or an `Imagen` block has no image. The server also
requires a name (up to 80 characters), valid colors, titles and subtitles up to 120 characters,
texts up to 1000, at most 100 products per block and 200 blocks per menu, and that every category
and product named is still active.

<!-- DOC-ID: workflow-delete-menu -->
## Delete a menu

Open the menu and press the red trash button, then confirm. A menu that a layout still shows cannot
be deleted: the message names those layouts; change their menu in `Diseño de salones` first.

<!-- DOC-ID: limits -->
## Limits of this version

- The menu is previewed here; the customer's QR page that shows it is built with the QR section of
  the restaurant module and is not available yet.
- No AI assistant, per-block colors, printing, schedules (breakfast/lunch) or translations.
- The preview shows every restaurant product whatever its stock. The customer's page will show only
  what can be ordered at that moment.
- Access: the page and `POST dining-menu` belong to the **Carta** access, separate from **Diseño de
  Salones**.

<!-- DOC-ID: source-provenance -->
## Source provenance

Read from `frontend/routes/dining/menu-builder/`, `frontend/core/dining-menu.ts`,
`backend/dining/dining_menus.go` and `backend/dining/types/dining_menu*.go`. Not exercised in a
running app.
