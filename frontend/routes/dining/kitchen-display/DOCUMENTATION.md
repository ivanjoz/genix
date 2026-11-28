---
schema: 1
page_id: dining.kitchen-display
route: /dining/kitchen-display
title: Orders (Pedidos)
status: implemented
visibility: tenant
description_en: >-
  The restaurant's kitchen display. Lists the tickets still to prepare (one round of one table for
  one prep station), the longest-waiting first, colored by the layout's wait thresholds. Pick the
  layout and the station, open a ticket and mark its dishes ready one by one or all at once, or
  send a dish back to the kitchen within 2 minutes. Works with the keyboard.
description_es: >-
  La pantalla de cocina del restaurante. Lista las comandas por preparar (una ronda de una mesa
  para una estación de preparación), de la que más espera a la más nueva, con colores según los
  tiempos de espera del salón. Elige el salón y la estación, abre una comanda y marca sus platos
  listos uno por uno o todos, o devuelve un plato a cocina dentro de 2 minutos. Se usa con teclado.
---

# Orders (Pedidos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Orders (`Pedidos`) is the kitchen display (`pantalla de cocina`, `KDS`, `monitor de comandas`) of
the restaurant module. It shows the cooks and the bar what is still to prepare in one dining room
(`salón`), the oldest first, and lets them mark each dish ready (`listo`) so the waiters know it
can be picked up.

Despite its name, this page does not handle sale orders (`pedidos de venta`) or online orders. It
does not take orders, confirm QR orders, serve, cancel or charge a table: those happen in **Tables
(Mesas)**. Stations, routing and wait thresholds are configured in **Floor layout (Diseño de
salones)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **ticket** (`comanda`, `pedido`) is one round (`Ronda`) of one table for one prep station. If a
  round has dishes for the kitchen and drinks for the bar, each station sees its own ticket.
- A **prep station** (`estación de preparación`: Cocina, Bar, Parrilla…) receives each product by
  the layout's routing: the station naming the product, else the one naming its category, else
  the default station. The station is fixed when the dish is ordered; changing the routing later
  only affects new dishes.
- A dish (`plato`, line) shown here is **In the kitchen (En cocina)** or **Ready (Listo)**. QR dishes
  still **Por confirmar**, served and cancelled dishes are not shown.
- The **wait** (`espera`) of a ticket counts from its oldest dish still in the kitchen.

<!-- DOC-ID: capability.view-tickets -->
## See what to prepare (Ver las comandas por preparar)

### User intention (Intención del usuario)

Know which tables are waiting, in which order to cook, and how late each ticket is.

### Where to find it (Dónde encontrarlo)

**Commercial (Comercial) → Restaurant (Restaurante) → Orders (Pedidos)** at
`/dining/kitchen-display`. On the left, the layout strip and, when the layout has more than one
station, **Station (Estación)** with **All (Todas)**. Below, the ticket list. Click a ticket to see
its dishes on the right.

### Required information and prerequisites (Requisitos previos)

At least one active layout (`salón`) created in Floor layout, and open table orders taken in Tables.

### Business rules and rationale (Reglas y razón de negocio)

- Each ticket shows the table, **Ronda** number, station, the dishes still in the kitchen with
  their quantity, and the minutes waited.
- The list is ordered by wait, longest first. A ticket leaves the list once all its dishes are
  ready.
- The minutes badge turns amber at the layout's **Alerta a los (min)** and red at **Retraso a los
  (min)**, the same thresholds that color the floor map.
- With a single layout it opens by itself. The chosen layout is remembered on this device; changing
  layout resets the station to **Todas**.
- The list refreshes every 15 seconds while the page is visible.
- Each dish card shows its quantity, product, status and the waiter's note (`sin cebolla`).

### Limitations (Limitaciones)

- One layout at a time.
- Charged or annulled table orders disappear from the list.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde ve la cocina los pedidos? ¿Cómo veo solo lo del bar? → **Estación**.
- ¿Por qué un pedido sale en rojo? → it passed the layout's Retraso a los (min).
- ¿Por qué no aparece el pedido por QR? → it still waits for a waiter's confirmation in Mesas.
- Search terms: `comanda`, `cocina`, `KDS`, `pantalla de cocina`, `barra`, `bar`, `estación`,
  `ronda`, `demora`.

<!-- DOC-ID: capability.mark-ready -->
## Mark dishes ready (Marcar platos listos)

### User intention (Intención del usuario)

Tell the floor that a dish, several dishes or the whole ticket is ready to be picked up (`plato
listo`, `despachar`).

### Where to find it (Dónde encontrarlo)

Select a ticket. On the right, **All ready (Listo todo)** marks every dish of the ticket still in
the kitchen. To mark only some, tap the dish cards to select them, then press **Ready (Listo)**,
which shows how many it applies to. Until something is selected the page says **Toca los productos
para seleccionarlos**.

With the keyboard: ↑ ↓ move between tickets, → enters a ticket, the arrows move between its cards
(← on the first column goes back), Space selects a card, Enter marks ready the selected cards (or
the card under the keyboard), Esc clears the selection or goes back.

### Business rules and rationale (Reglas y razón de negocio)

- Only a dish **En cocina** can be marked ready.
- After marking, the keyboard moves to the next dish still in the kitchen; when the ticket has none
  left it leaves the list and the next ticket takes its place.

### Result and side effects (Resultado y efectos)

The dish becomes **Listo**. In Tables, the table shows **Listo para recoger** and the waiter can mark
it **Servido**.

### Limitations (Limitaciones)

From this page a dish can only be marked ready or sent back to the kitchen. A user who only has
the kitchen access, not the table service one, is refused any other change.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo marco un plato como listo? ¿Cómo despacho toda la comanda? ¿Hay atajos de teclado?

<!-- DOC-ID: capability.return-to-kitchen -->
## Undo a ready tap (Devolver a cocina)

### User intention (Intención del usuario)

Take back a dish marked ready by mistake.

### Where to find it (Dónde encontrarlo)

Select the ticket, tap the **Listo** dish card and press **Back to kitchen (Devolver a cocina)**.

### Business rules and rationale (Reglas y razón de negocio)

Only within 2 minutes after the dish was marked ready. After that, the option disappears and Genix
refuses it: "Un plato marcado listo solo se puede devolver a cocina en los 2 minutos siguientes."

### Result and side effects (Resultado y efectos)

The dish returns to **En cocina** and its ticket comes back to the list; its wait still counts
from when the dish was ordered.

### Common questions and vocabulary (Preguntas y vocabulario)

- Marqué listo por error, ¿cómo lo regreso? ¿Puedo deshacer un plato listo?

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Elija un salón." (Pick a layout):** choose a layout in the strip at the top.
- **"Nada por preparar." (Nothing to prepare):** no dish of this layout and station is in the
  kitchen; check the station filter or the layout.
- **A dish went to the wrong station:** the routing is set in Floor layout and applies to dishes
  ordered after the change.
- **"El pedido de la mesa no existe o ya se cerró":** the table was charged or annulled meanwhile.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Tables (Mesas)** takes the orders, confirms QR orders, serves and cancels dishes and charges
  the bill.
- **Floor layout (Diseño de salones)** sets the stations, the product and category routing and the
  wait thresholds.
- **Menu (Carta)** builds the carta customers see when they scan the table's QR.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.view-tickets]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, related-pages]
  - path: frontend/routes/dining/kitchen-display/+page.svelte
    role: page
    hash: sha256:596dd83e2ae24ab7b6fd27c2f84130c2674ecf1f65ef5843b7eda31489cca348
    supports: [page-purpose, capability.view-tickets, capability.mark-ready, capability.return-to-kitchen, troubleshooting]
  - path: frontend/routes/dining/kitchen-display/kitchen-display.ts
    role: business-logic
    hash: sha256:31b1f8f525bdc7c4c990fc7af36ea071a7db049eb9f83650d54e7c36b376b221
    supports: [concepts, capability.view-tickets, capability.mark-ready, capability.return-to-kitchen]
  - path: frontend/routes/dining/dining.ts
    role: shared-domain
    hash: sha256:41a36c9cc5ca0021192aab2c50e7420a585dd2ee9d82d74a8cedf5094f5f536d
    supports: [concepts, capability.view-tickets, capability.return-to-kitchen]
  - path: frontend/routes/dining/dining.svelte.ts
    role: frontend-service
    hash: sha256:7e06d56ddc68a211a527fce0791b4a97f0ff96b3f378825fd9fa0709b0fbeb00
    supports: [capability.view-tickets]
  - path: frontend/routes/dining/OrderLineCards.svelte
    role: user-interface
    hash: sha256:7aa8fcf618993abe02b94b29b83d574698abd9288f7f7311d1a96c33c4f10da5
    supports: [capability.view-tickets, capability.mark-ready, capability.return-to-kitchen]
  - path: frontend/routes/dining/floor-layout/FloorLayoutSettings.svelte
    role: user-interface
    hash: sha256:7314ca1d9e45cb22374925c0ba744ad5defa0292e2fe788531ef619c13610a6e
    supports: [capability.view-tickets, related-pages]
  - path: backend/dining/table_orders.go
    role: backend-handler
    hash: sha256:697cc76c99ce7eb5cbd2df251b19e516f4865f8a7dcf8902c068a77945bd162b
    supports: [capability.view-tickets, capability.mark-ready, troubleshooting]
  - path: backend/dining/types/table_order_rules.go
    role: business-logic
    hash: sha256:9c498f6b18e8988362d1e317246939bec603a924c004687940bbf7e18b231791
    supports: [capability.mark-ready, capability.return-to-kitchen]
  - path: backend/dining/types/table_order.go
    role: data-model
    hash: sha256:065c873e62acf72d7f2e33e1fee4aa2a4a1edb1d1a20112d5082fdf1b2da1c3e
    supports: [concepts]
  - path: backend/dining/types/dining_layout_rules.go
    role: business-logic
    hash: sha256:40a72fdfa154967e80553da83a8bf9874d058f304e43b0b1b9c0c6937b4a2805
    supports: [concepts, troubleshooting]
  - path: backend/access.toml
    role: permissions
    hash: sha256:1e71e45906aed3d07f42a65ad2ad13b38ad2055938fd5098547be4d4898f769a
    supports: [capability.mark-ready]
```
