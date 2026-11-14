---
schema: 1
page_id: dining.floor-layout
route: /dining/floor-layout
title: Floor Layout (Diseño de Salones)
status: implemented
visibility: tenant
description_en: >-
  Restaurant floor-plan editor. Draw each dining room on a grid with tables (square or round, seats,
  name), blocks, walls, texts, prep-station marks and pickup points; choose the warehouse its
  products come from; set the grid size, the wait-time thresholds and whether QR orders need a
  waiter's confirmation; manage the prep stations (Cocina, Bar...) with the products and categories
  routed to each and one default station. The whole layout is saved at once and can be deleted.
description_es: >-
  Editor del plano del restaurante. Dibujar cada salón en una cuadrícula con mesas (cuadradas o
  redondas, asientos, nombre), bloques, paredes, textos, marcas de estación de preparación y puntos
  de recojo; elegir el almacén de sus productos; definir el tamaño de la cuadrícula, los tiempos de
  alerta y retraso y si los pedidos por QR esperan la confirmación de un mozo; administrar las
  estaciones de preparación (Cocina, Bar...) con los productos y categorías que van a cada una y una
  estación por defecto. El salón se guarda completo de una vez y se puede eliminar.
---

# Floor Layout (Diseño de Salones)

<!-- DOC-ID: page-purpose -->
## Page purpose

Floor Layout (`Diseño de salones`) is where a restaurant manager draws each physical dining space
(`salón`: the main room, the terrace, the second floor) as a floor plan (`plano`) on a grid of cells.
On the plan go the tables (`mesas`) with their name and seats (`asientos`), blocks such as a bar
counter, column or planter (`bloques`), walls and dividers (`paredes`), free texts (`textos`), the
marks of where each prep station sits (`estación de preparación`) and where waiters collect its
dishes (`punto de recojo`).

Each layout also holds its configuration for the restaurant service: the warehouse its products
come from (`Almacén de sus productos`), the waiting thresholds (`Alerta a los (min)`, `Retraso a los
(min)`), whether orders placed by QR wait for a waiter's confirmation, and its prep stations
(`Cocina`, `Bar`, `Parrilla`...) with the products and categories routed to each one.

This page only designs and configures layouts. Waiters take the table orders on the live floor map
of **Atención de mesas** (Table service), which draws the layouts saved here, colors each table by
its wait thresholds and sends each product to the station this page routes it to. In this version
Genix has **no** table QR codes or kitchen display. Drawing or saving a layout does not move stock,
reserve products or create sales.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Layout (`salón`):** one physical space the restaurant serves in. A company can have many
  layouts; each one has its own plan, warehouse, stations and settings, and is saved as one whole
  document.
- **Grid and cells (`cuadrícula`, `celdas`):** the plan is measured in cells, `Ancho (celdas)` ×
  `Alto (celdas)`. Every element except a wall covers a whole number of cells; a wall joins two grid
  points. The zoom buttons (8px, 12px, 16px) only change how big a cell looks on screen, not the
  layout.
- **Elements of the plan (`elementos`):**
  - **Table (`Mesa`):** square (`Cuadrada`) or round (`Redonda`), with a name such as `M4` and a
    number of seats. The plan shows its name and, when the table is drawn tall enough, its seats as
    `4p`.
  - **Block (`Bloque`):** a solid area such as a bar counter, column or planter. It has no text.
  - **Wall / divider (`Pared / divisor`):** a straight line between two grid points.
  - **Text (`Texto`):** a free label on the plan.
  - **Prep station mark (`Estación de preparación`):** shows where a station sits; it displays the
    station's name.
  - **Pickup point (`Punto de recojo`):** shows where waiters collect a station's dishes; it displays
    `Recojo · <estación>`.
- **Prep station (`estación de preparación`):** a place that prepares part of the menu (kitchen,
  bar, grill). It is a list entry of the layout, separate from its marks on the plan: a station
  exists even if it is not drawn, and a mark only points to one.
- **Default station (`estación por defecto`):** exactly one station per layout receives every
  product no other station claims.
- **Routing (`ruteo de productos`):** the products (`Productos`) and categories (`Categorías`) a
  station names. The rule shown on the page is: a product goes to the station that names it, else
  to the one naming its category, else to the default station.

<!-- DOC-ID: capability.manage-layouts -->
## Create, open and save a layout (Crear, abrir y guardar un salón)

### User intention (Intención del usuario)

Start the plan of a new dining room, reopen an existing one to change it, and save the changes.

### Where to find it (Dónde encontrarlo)

**Comercial → Restaurante (Restaurant) → Diseño de salones (Floor layout)**, route
`/dining/floor-layout`. The top bar has the **Salón** picker with the company's layouts (sorted by
name), **Nuevo salón** (New layout), and, once a layout is open, **Guardar** (Save) and, for a saved
layout, **Eliminar** (Delete). The user needs the access **Diseño de Salones** in their profile.
Until a layout is chosen the page shows "Elija un salón o cree uno nuevo."

### Required information and prerequisites (Requisitos previos)

- At least one active warehouse (`almacén`), created in Sites & Warehouses, to pick as the layout's
  warehouse.
- A **New layout** starts empty with these values: grid of 40 × 30 cells, one station **Cocina** set
  as default, **QR orders wait for a waiter's confirmation** checked, `Alerta a los` 10 minutes,
  `Retraso a los` 20 minutes, and no name and no warehouse — both must be filled before saving (tab
  **Salón** of the side panel).

### Business rules and rationale (Reglas y razón de negocio)

- **Save writes the whole layout at once:** the plan, the settings and the stations. Nothing is
  saved while drawing; the amber notice **Cambios sin guardar** shows that the layout on screen
  differs from what was opened or last saved.
- Opening another layout or pressing **Nuevo salón** with unsaved changes asks "El salón tiene
  cambios sin guardar. ¿Descartarlos?"; answering no keeps the current layout open.
- On save Genix trims spaces from the name, the table names, the texts and the station names, and
  fills a wait threshold left at 0 with its default (10 or 20 minutes). Then it checks every rule of
  the layout (see the cross-capability rules) and rejects the whole save with the first error found;
  nothing is saved partially.
- After a successful save ("Salón guardado.") the page reloads the layout as saved, so the undo
  history starts again from that point.
- **A table with an open order stays:** a save that removes a table (or turns it into another kind
  of element) while it has an open order in **Atención de mesas** is rejected with "La mesa "<nombre>"
  tiene un pedido abierto: no se puede eliminar del salón." Moving, resizing, renaming or rotating
  that table is allowed.

### Result and side effects (Resultado y efectos)

A new layout is created, or the existing one is replaced with what is on screen, and it appears in
the **Salón** picker. No stock, product, warehouse or sale changes.

### Limitations (Limitaciones)

- There is no copy or "save as" of a whole layout; to start a similar room, use **Nuevo salón** and
  draw it, or duplicate elements inside one layout.
- Undo (Ctrl+Z) cannot go back past a save or past opening a layout.
- Closing or leaving the page does not save the draft.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo un salón nuevo?` Use **Nuevo salón**, draw the plan, set the name and the warehouse
  in the **Salón** tab and press **Guardar**.
- `¿Se guarda automáticamente?` No, only with **Guardar**; "Cambios sin guardar" warns while
  something is pending.
- `¿Puedo tener varios salones?` Sí, one per physical space (salón principal, terraza, segundo piso).
- Search terms: `diseño de salones`, `plano del restaurante`, `mapa de mesas`, `distribución de
  mesas`, `croquis del local`, `layout`, `salon`, `nuevo salón`, `guardar salón`.

<!-- DOC-ID: capability.draw-plan -->
## Draw and edit the plan (Dibujar y editar el plano)

### User intention (Intención del usuario)

Reproduce the real room: place the tables, the bar and columns, the walls, the signs, and where
each station and pickup point is, then move, resize, rotate, copy or remove them.

### Where to find it (Dónde encontrarlo)

The toolbar above the plan has the tools **Seleccionar** (Select), **Mesa** (square table),
**Redonda** (round table), **Bloque** (Block), **Pared** (Wall), **Texto** (Text), **Estación**
(Station mark) and **Recojo** (Pickup point), then the buttons to duplicate, rotate 90°, delete,
undo and redo the selection, and the zoom (8px / 12px / 16px).

### Required information and prerequisites (Requisitos previos)

An open layout. The station and pickup marks need a station in the **Estaciones** tab; a new
mark points to the default station and can be changed in the **Elemento** tab.

### Business rules and rationale (Reglas y razón de negocio)

- **Placing:** pick a tool and click a cell; the element's top-left corner goes on that cell and the
  tool returns to **Seleccionar**, so each click places one element. New elements have these
  sizes: table 3 × 3 cells with 4 seats, block 3 × 2, text 6 × 2 reading "Texto", station mark 4 × 2,
  pickup point 6 × 2.
- **Table names:** a new or duplicated table is named `M` plus one more than the highest number at
  the end of the existing table names (after `M7` comes `M8`).
- **Walls:** with **Pared**, press and drag from one grid point to another; a click without drag
  draws nothing.
- **Selecting:** click an element; **Shift+click** adds or removes it; dragging over empty space
  draws a band that adds the elements it covers **completely**. Clicking empty space without Shift
  clears the selection. When elements overlap, the last one drawn is picked.
- **Moving:** drag the selection or use the arrow keys (one cell per press). Elements move by whole
  cells and the group stops at the edge of the grid.
- **Resizing:** a single selected element shows a square handle at its bottom-right corner; a wall
  shows a handle at each end. An element is at least one cell, and dragging a wall's end never
  shrinks it to a single point.
- **Rotate (90°):** turns each selected element clockwise. A table, block or mark turns around its
  center, swapping its width and height, and its name and seats turn with it; a wall turns around
  its starting point. An element that would leave the grid is shifted back in whole, so a wall keeps
  its length; an element too long to fit the grid once turned (a wall longer than the plan is high)
  does not turn.
- **Duplicate (Ctrl+D):** copies the selection one cell down and right; copied tables get new names.
- **Delete (Delete / Backspace key or the trash button):** removes the selected elements from the
  plan.
- **Undo / redo:** Ctrl+Z undoes; Ctrl+Y or Ctrl+Shift+Z redoes; about 100 steps are kept. **Escape**
  clears the selection and returns to **Seleccionar**. Keyboard shortcuts do not act while the
  cursor is in a form field.
- **Overlapping tables** are drawn with a red outline and the warning "Algunas mesas se superponen
  (borde rojo)." It is only a warning: the layout can be saved with them.
- Under the plan Genix shows the number of tables (`Mesas: N`).

### Result and side effects (Resultado y efectos)

The changes stay in the draft until **Guardar**. Saving keeps the elements exactly as drawn.

### Limitations (Limitaciones)

- Elements snap to whole cells; there are no free angles: rotation is in 90° steps and walls are
  straight lines between grid points.
- Blocks and walls have no text; use a **Texto** element to label them.
- A layout holds at most 500 elements.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo agrego una mesa redonda?` Tool **Redonda**, click on the plan.
- `¿Cómo dibujo una pared?` Tool **Pared**, drag from one point to another.
- `¿Cómo selecciono varias mesas?` Shift+clic, or drag over the empty space around them.
- `¿Cómo copio una mesa?` Select it and press Ctrl+D or the duplicate button.
- `¿Por qué una mesa tiene borde rojo?` It overlaps another table.
- `Me equivoqué, ¿cómo deshago?` Ctrl+Z or the undo button, before saving.
- Search terms: `agregar mesa`, `mover mesas`, `girar mesa`, `rotar`, `duplicar mesa`, `copiar
  mesa`, `borrar mesa`, `pared`, `divisor`, `barra`, `columna`, `bloque`, `texto`, `deshacer`,
  `rehacer`, `zoom`, `mesas superpuestas`.

<!-- DOC-ID: capability.edit-element -->
## Name tables and set their seats (Configurar mesas y elementos)

### User intention (Intención del usuario)

Give each table the name waiters use, its number of seats and its shape, or type the exact
position and size of an element.

### Where to find it (Dónde encontrarlo)

Select one element and open the **Elemento** (Shape) tab of the side panel. With several elements
selected it only shows how many are selected.

### Required information and prerequisites (Requisitos previos)

The fields depend on the element:

- **Mesa:** **Nombre de mesa**, **Asientos** (seats) and **Forma** (`Cuadrada` / `Redonda`).
- **Texto:** **Texto**.
- **Estación de preparación** and **Punto de recojo:** **Estación**, the station it stands for.
- Every element: **X** and **Y** (position in cells) and **Ancho** / **Alto** (width and height in
  cells); a wall has **X**, **Y**, **X2**, **Y2** for its two ends.

### Business rules and rationale (Reglas y razón de negocio)

- Every table needs a name, and table names cannot repeat inside the same layout, **without
  distinguishing upper and lower case**: `m4` and `M4` are the same table to a waiter. Two layouts
  may each have their own `M1`.
- Seats go from 0 to 40.
- Table names and texts have at most 40 characters.
- A typed position or size that leaves the grid is pulled back inside when the field is left. An
  empty or zero width or height counts as one cell, and a wall typed with both ends on the same
  point gets one cell of length.

### Result and side effects (Resultado y efectos)

The plan updates on screen; the change is saved with **Guardar**.

### Limitations (Limitaciones)

- Rotation is not typed: use the rotate button.
- A block or a wall has no name.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo cambio el número de una mesa?` Select it and edit **Nombre de mesa**.
- `¿Cómo pongo cuántas personas entran en la mesa?` Field **Asientos** (capacidad, cubiertos,
  sillas, pax).
- `¿Puedo tener dos mesas con el mismo nombre?` No in the same layout.
- Search terms: `nombre de mesa`, `número de mesa`, `asientos`, `capacidad de la mesa`, `sillas`,
  `mesa cuadrada`, `mesa redonda`, `forma de la mesa`.

<!-- DOC-ID: capability.layout-settings -->
## Configure the layout (Configurar el salón)

### User intention (Intención del usuario)

Name the room, choose the warehouse its products come from, size the grid, and set the waiting
times and the QR confirmation for the restaurant service.

### Where to find it (Dónde encontrarlo)

The **Salón** (Layout) tab of the side panel: **Nombre**, **Almacén de sus productos**, **Ancho
(celdas)**, **Alto (celdas)**, **Alerta a los (min)**, **Retraso a los (min)** and the checkbox **Los
pedidos por QR esperan la confirmación de un mozo**.

### Required information and prerequisites (Requisitos previos)

- **Nombre** is required.
- **Almacén de sus productos** is required and lists only active warehouses.
- Grid width and height from 8 to 200 cells each.
- **Alerta a los** must be greater than 0 and lower than **Retraso a los**; a value left at 0 is
  saved as the default (10 for the warning, 20 for late).

### Business rules and rationale (Reglas y razón de negocio)

- The warehouse is checked on save: it must still exist and be active.
- **Making the grid smaller pulls every element back inside it**: elements beyond the new edge move
  in, and an element larger than the grid shrinks to fit. Use undo (Ctrl+Z) if the result is not
  wanted.
- The warning and late thresholds (`tiempo de espera`, `alerta`, `retraso`) color the tables of
  **Atención de mesas**: a table whose oldest dish in the kitchen has waited **Alerta a los** minutes
  shows as `Esperando`, and **Retraso a los** minutes as `Retrasado`.
- The QR confirmation setting is stored with the layout for QR ordering, which no page offers yet.
- The warehouse is read when a table's order is opened: an order already open keeps the warehouse it
  started with if the layout's warehouse is changed.

### Result and side effects (Resultado y efectos)

Saved with the rest of the layout. Choosing the warehouse moves no stock and reserves nothing.

### Limitations (Limitaciones)

- One warehouse per layout.
- The zoom (cell size on screen) is not a setting of the layout.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿De qué almacén salen los productos del salón?` The **Almacén de sus productos** of the layout.
- `¿Cómo agrando el salón?` Increase **Ancho (celdas)** or **Alto (celdas)** (máximo 200).
- `¿Por qué se movieron mis mesas al achicar el salón?` A smaller grid pulls the elements inside.
- `¿Los pedidos por QR necesitan que el mozo los confirme?` It is the checkbox of the layout; it is
  checked by default.
- Search terms: `almacén del salón`, `tamaño del salón`, `tiempo de espera`, `alerta`, `retraso`,
  `demora`, `confirmación de mozo`, `pedidos por QR`, `código QR`, `mesero`, `mozo`, `camarero`.

<!-- DOC-ID: capability.prep-stations -->
## Manage prep stations and their routing (Estaciones de preparación)

### User intention (Intención del usuario)

Define where each dish or drink is prepared — kitchen, bar, grill — by naming the products and
categories each station handles and which station takes everything else.

### Where to find it (Dónde encontrarlo)

The **Estaciones** (Stations) tab of the side panel: an editable table with the columns **Estación**
(name), **Defecto** (default), **Productos**, **Categorías** and a delete button per row, and the
**Agregar estación** button. The tab explains: "Un producto va a la estación que lo nombra, si no a
la que nombra su categoría, si no a la por defecto."

### Required information and prerequisites (Requisitos previos)

- Every station needs a name of up to 40 characters; a new station starts without a name.
- **Productos** lists the active products; **Categorías** lists the product categories maintained
  on the Products page.

### Business rules and rationale (Reglas y razón de negocio)

- A layout has between 1 and 20 stations, and **exactly one** default station. Clicking a row's
  **Defecto** cell makes it the default and unsets the previous one.
- **Routing order:** a product goes to the first station (in the list order) that names it in
  **Productos**; if none does, to the first station that names one of its categories; otherwise to
  the default station. Naming a product directly therefore wins over its category.
- **The default station cannot be deleted** ("La estación por defecto no se puede eliminar."): make
  another one the default first.
- **A station with a mark on the plan cannot be deleted** ("Una marca del plano representa esta
  estación: elimínela primero."): delete or repoint its station and pickup marks first.

### Result and side effects (Resultado y efectos)

Saved with the layout. Each layout has its own stations: a station of one layout does not exist in
another.

### Limitations (Limitaciones)

- Each table order line takes its station when the waiter sends the round in **Atención de mesas**,
  where ready dishes are grouped by station under "Listos para recoger". Changing the routing does
  not move lines already ordered. In this version there is no kitchen display per station.
- Stations are not shared between layouts; two rooms with the same kitchen each need their station.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo hago que las bebidas vayan al bar?` Create the station **Bar** and add the drinks category
  (or the products) to it.
- `¿Qué pasa con un producto que no está en ninguna estación?` It goes to the default station.
- `¿Por qué no puedo eliminar la estación Cocina?` It is the default, or a mark on the plan points to
  it.
- Search terms: `estación de preparación`, `cocina`, `bar`, `parrilla`, `barra`, `comandas por
  estación`, `ruteo de productos`, `estación por defecto`, `punto de recojo`, `pase`.

<!-- DOC-ID: capability.delete-layout -->
## Delete a layout (Eliminar un salón)

### User intention (Intención del usuario)

Remove a dining room that is no longer used or was created by mistake.

### Where to find it (Dónde encontrarlo)

Open the layout and press **Eliminar** in the top bar, then confirm "¿Eliminar "<nombre>"? Sus mesas
dejan de recibir pedidos." The button appears only for a layout that has been saved.

### Required information and prerequisites (Requisitos previos)

A saved layout. A new layout that was never saved has no **Eliminar**: just open another one or
leave.

### Business rules and rationale (Reglas y razón de negocio)

- Unsaved edits on screen are not applied: the layout is removed as it was last saved.
- **A layout with open table orders cannot be deleted** ("El salón tiene N mesa(s) con pedidos
  abiertos: ciérrelos antes de eliminarlo."): the orders would point to tables that no longer exist.

### Result and side effects (Resultado y efectos)

The layout disappears from the **Salón** picker ("Salón eliminado.") and the page returns to "Elija
un salón o cree uno nuevo." No warehouse, product or stock is affected.

### Limitations (Limitaciones)

The page has no way to restore a deleted layout; it must be drawn again.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo borro un salón?` Open it and press **Eliminar**.
- `¿Puedo recuperar un salón eliminado?` Not from this page.
- Search terms: `eliminar salón`, `borrar salón`, `quitar salón`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

When saving, Genix rejects the layout with one of these messages:

- "El salón necesita un nombre." / "El salón necesita el almacén del que salen sus productos."
- "El almacén del salón no existe o está inactivo."
- "El tamaño del salón debe estar entre 8 y 200 celdas por lado."
- "La espera de alerta debe ser mayor a 0 y menor a la espera de retraso."
- "El salón admite hasta 500 elementos."
- "El salón necesita entre 1 y 20 estaciones de preparación." / "Cada estación necesita un nombre
  de hasta 40 caracteres." / "Debe haber exactamente una estación por defecto."
- "Cada mesa necesita un nombre." / "El nombre de mesa "<nombre>" está repetido." (case-insensitive)
- "La mesa "<nombre>" debe tener entre 0 y 40 asientos." / "La forma de la mesa "<nombre>" es
  inválida."
- "El texto "<texto>" supera los 40 caracteres."
- "El elemento N (<nombre>) queda fuera del salón o no tiene tamaño."
- "Una marca de estación o de recojo apunta a una estación que no existe."
- "La mesa "<nombre>" tiene un pedido abierto: no se puede eliminar del salón."

Genix gives each element an internal number when the layout is saved and never reuses a number in
that layout, so a deleted table's identity never passes to a new table, even one with the same
name.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El salón necesita el almacén del que salen sus productos." or "El almacén del salón no existe
  o está inactivo.":** pick an active warehouse in the **Salón** tab; reactivate the warehouse in
  Sites & Warehouses if it was deactivated.
- **"El nombre de mesa "M4" está repetido.":** two tables have the same name, maybe differing only
  in upper/lower case; rename one in the **Elemento** tab.
- **"Cada estación necesita un nombre de hasta 40 caracteres.":** a station added with **Agregar
  estación** was left without a name; name it or delete it.
- **"La espera de alerta debe ser mayor a 0 y menor a la espera de retraso.":** make **Alerta a
  los** smaller than **Retraso a los**.
- **"La forma de la mesa ... es inválida." or "Una marca de estación o de recojo apunta a una
  estación que no existe.":** a table's **Forma** or a mark's **Estación** was left empty; choose
  one in the **Elemento** tab.
- **"El elemento N no existe en el salón guardado." or "El salón no existe.":** the layout was
  changed or deleted from another session after it was opened here. Reload the page and open the
  layout again; the unsaved changes on this screen are lost.
- **"El elemento N (...) queda fuera del salón, no tiene tamaño o sus esquinas no forman un
  rectángulo.":** the editor keeps every element inside the grid and never leaves a wall without
  length, so this only appears when the saved data was not produced by this screen. Reload the
  page, open the layout and redraw the element.
- **"No puedo eliminar una estación":** it is the default or a mark on the plan points to it.
- **"La mesa ... tiene un pedido abierto" or "El salón tiene N mesa(s) con pedidos abiertos":** the
  table, or some table of the layout, is being served in **Atención de mesas**. Wait until its order
  is closed, or keep the table on the plan.
- **The tables moved or shrank:** the grid was made smaller; press Ctrl+Z, or enlarge the grid.
- **Shortcuts (Delete, arrows, Ctrl+Z) do nothing:** the cursor is in a side-panel field; leave the
  field (for example by clicking the selected element on the plan) first.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sites & Warehouses (Sedes & Almacenes)** at `/business/branches-warehouses` (Empresa → Mi
  Empresa): create or reactivate the warehouse a layout uses.
- **Products (Productos)** at `/production/products`: the products and the categories a station
  routes.
- **Table service (Atención de mesas)** at `/dining/floor-service`, in the same Restaurante menu: the
  live floor map where waiters open tables, send rounds to the stations and follow each dish.
- **Daily close (Cierre del día)** at `/production/daily-close`, in the same Restaurante menu: the
  production close of made products sold and delivered from a warehouse.

### FILES

```yaml
# Replace pending only after the completed documentation is reviewed against exact file bytes.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/routes/dining/floor-layout/+page.svelte
    role: page
    hash: sha256:3e5f9137b494f7e4aef2ea01b741906f2ef8f8750f07db77cd7ed4e9f15f35c9
    supports: [page-purpose, capability.manage-layouts, capability.delete-layout, troubleshooting]
  - path: frontend/routes/dining/floor-layout/FloorLayoutEditor.svelte
    role: user-interface
    hash: sha256:271f655c67a3f7964c5f81669253aa12d71b59a7c258bfba7b19cd4a27ee5f7e
    supports: [capability.draw-plan, troubleshooting]
  - path: frontend/routes/dining/floor-layout/FloorLayoutSidePanel.svelte
    role: user-interface
    hash: sha256:b8c231bea81203fc6149d8be51d00587505a74651cc77e1abb4275036fb9bf6d
    supports: [concepts, capability.edit-element, capability.layout-settings, capability.prep-stations, troubleshooting]
  - path: frontend/routes/dining/floor-layout/floor-layout.ts
    role: business-logic
    hash: sha256:5b5753188f9c13a297e80e0d3a6ebf9f8377cb6aab0fd03a0ba2d74729e1007f
    supports: [capability.manage-layouts, capability.draw-plan, capability.edit-element, capability.layout-settings, capability.prep-stations, troubleshooting]
  - path: frontend/routes/dining/floor-layout/floor-layout.svelte.ts
    role: user-interface
    hash: sha256:f6056a3365f2c6ac5ebb378bb820797741dd64b85296a4bfaf3e0fb867bdbb88
    supports: [capability.manage-layouts, capability.draw-plan]
  - path: frontend/routes/dining/LayoutCanvas.svelte
    role: user-interface
    hash: sha256:939eb16f592af21aa07d823365b16e31f5d3e4897797b6b770cc1302f29f491c
    supports: [concepts, capability.draw-plan]
  - path: frontend/routes/dining/dining.ts
    role: shared-domain
    hash: sha256:6fd904c5753945a587d1c2f696b2a6ec93d505382c362a4a645a4734186b69ed
    supports: [concepts, capability.draw-plan, capability.layout-settings, capability.prep-stations]
  - path: frontend/routes/dining/dining.svelte.ts
    role: frontend-service
    hash: sha256:43495109cc9fbb91105d283b033ed185f1cf067895944fb7837a6570e8d1de58
    supports: [capability.manage-layouts, capability.delete-layout]
  - path: backend/dining/dining_layouts.go
    role: backend-handler
    hash: sha256:6dfaa4e6006aadf31fd60a77a46135e6fd1240b6e6d7f48cd7cc1667683cf260
    supports: [capability.manage-layouts, capability.layout-settings, capability.delete-layout, rules, troubleshooting]
  - path: backend/dining/types/dining_layout_rules.go
    role: business-logic
    hash: sha256:40a72fdfa154967e80553da83a8bf9874d058f304e43b0b1b9c0c6937b4a2805
    supports: [concepts, capability.manage-layouts, capability.draw-plan, capability.edit-element, capability.layout-settings, capability.prep-stations, rules, troubleshooting]
  - path: backend/dining/types/dining_layout.go
    role: data-model
    hash: sha256:394483f6b284ceafd722d326543589b30f4383845ead739cb0721a64aa1e843d
    supports: [page-purpose, concepts, capability.delete-layout, rules]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:cb61a6f79a034e4d468b46c173733b815f0af9007a01d1f3e69b92d358913d60
    supports: [page-purpose, capability.manage-layouts]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:4f2d4c16777913fddf196959c4341290e69a12a78323a5b95f8fa5d4738ca34d
    supports: [capability.manage-layouts, related-pages]
  - path: backend/access.toml
    role: permissions
    hash: sha256:c19cf85130065dcc0f839796e1e9b4b98a1b78df465ccb20d2b63b59a1a94e59
    supports: [capability.manage-layouts]
  - path: backend/dining/table_orders.go
    role: backend-handler
    hash: sha256:1972ba4ecd49a4b4d39fa89610fee7386c590f59e7076b662607c3c37c63a76a
    supports: [page-purpose, capability.manage-layouts, capability.layout-settings, capability.prep-stations, capability.delete-layout]
```
