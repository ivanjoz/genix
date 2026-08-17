---
schema: 1
page_id: webpage-builder.pages
route: /webpage-builder/pages
title: Pages (Páginas)
status: implemented
visibility: tenant
description_en: >-
  Website pages management. Create, edit, and remove store pages with name, route, and
  Active/Published status; open a page's builder to add, edit, remove, and reorder its content
  sections (hero, banners, featured products, etc.) and review its generated color palette. Also
  configures the storefront's domain and SEO metatags.
description_es: >-
  Gestión de páginas del sitio web. Crear, editar y eliminar páginas de la tienda con nombre, ruta
  y estado Activo/Publicado; abrir el constructor de una página para agregar, editar, eliminar y
  reordenar sus secciones de contenido (hero, banners, productos destacados, etc.) y revisar su
  paleta de colores generada. También configura el dominio y los metatags SEO de la tienda.
---

# Pages (Páginas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Pages (`Páginas`) is the admin page for the storefront's website pages (`páginas del sitio web`):
the records that give each page its name, its public route (`ruta`), and its Active/Published
status, plus the entry point into that page's visual content builder (`constructor visual`). This
same route also holds the storefront's domain and SEO configuration, under its second view,
**Config**.

This page does not itself design a page's sections — that happens in the separate per-page
builder route `/webpage-builder/<ID>` (opened from here through the pencil action) — and it does
not manage the image gallery (**Gallery/Galería**, `/webpage-builder/gallery`), which is a
separate menu item.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **page (página)** is one storefront route: an ID (reused as the builder's PageID), a Name, a
  Route starting with "/", a thumbnail Image, and a Status (1 Active/Activo or 2
  Published/Publicado; 0 marks it removed).
- **System pages (páginas del sistema)**, IDs 10-14, are injected by the client rather than stored
  rows: Home (`Inicio`, "/"), About Us (`Nosotros`, "/about"), and Store (`Tienda`, "/store")
  appear as cards here; the Product and Cart routes are reserved too but are not shown as cards.
  A system page's Name/Route/Status are fixed and cannot be edited from this page — only its
  thumbnail can change, and only as a side effect of saving its content in the builder.
- A page's **content (contenido)** — its sections (hero, banners, featured products, etc.) — is a
  separate record set edited in the per-page builder, not on this list. Renaming a page or
  changing its Route here never touches its saved content.
- **Publishing (publicar)** the site is a distinct action from anything on this Pages tab: it
  happens when the storefront's domain is (re)saved on the **Config** view, which renders and
  republishes every page currently eligible for it (see "Configure the storefront domain" below).
  Saving a page's Name/Route/Status here, or saving its content in the builder, does not by
  itself push anything to the live site.

<!-- DOC-ID: capability.browse-pages -->
## Find and preview a page (Buscar y previsualizar una página)

Open **Website (Página Web) → Pages (Páginas)** at `/webpage-builder/pages`, on the **Pages**
top tab (the second tab, **Config**, is described further below). Pages show as cards with a
thumbnail (the last screenshot captured from the builder, or a placeholder icon when none exists
yet), the page Name, a status badge, its Route, who last updated it, and when. The filter box
narrows the cards by Name or Route; it matches only the already-loaded list on the client, not a
server-side search.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Dónde veo todas las páginas de mi tienda?`
- `¿Por qué una página no tiene miniatura (thumbnail)?` Porque nunca se guardó contenido para ella
  en el constructor, o la captura de vista previa falló.
- Search terms: `páginas web`, `página de inicio`, `about`, `tienda`, `constructor`, `builder`.

<!-- DOC-ID: capability.create-edit-page -->
## Create or edit a page record (Crear o editar una página)

### User intention (Intención del usuario)

Register a new store page — for example a landing page or a promotional page — with a name and
a public route, or change an existing user-created page's name, route, or status.

### Where to find it (Dónde encontrarlo)

On the Pages tab, use the green **New (Nuevo)** button to open the form, or click any
user-created card (a system-page card's body does nothing when clicked — only its pencil works).
Save with the modal's **Save/Update** action.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre):** required, at least 3 characters (checked on the client and re-checked by
  the server).
- **Route (Ruta):** required when creating or editing (the route check is skipped only when
  deleting); must start with "/" and cannot be "/" itself.
- **Status (Estado):** Active (`Activo`) or Published (`Publicado`); no other status value is
  offered from this form.

### Business rules and rationale (Reglas y razón de negocio)

- A page in the reserved ID range (1-14, the system pages) can never be written from this form;
  the server rejects the save outright.
- A new page's Route must not collide with a reserved system route (`/`, `/about`, `/store`,
  `/product`, `/cart`) nor with the Route of another currently active/published page. The server
  assigns each new page the next sequential ID, starting at 15, instead of relying on database
  autoincrement, so a reserved ID is never produced.
- Editing a page's Name/Route/Status never touches its saved builder content or its thumbnail.

### Result and side effects (Resultado y efectos)

Saving creates or updates the page record. A brand-new page has no content yet — its builder
opens empty until the pencil action is used to add sections.

### Limitations (Limitaciones)

- The Status choice (Active vs Published) has no verified effect on which pages actually appear
  on the published site: the render/publish step (triggered from Config → Domain) includes every
  page with Status 1 or 2 alike. There is currently no confirmed behavior difference between the
  two beyond the badge color shown on this list.
- There is no field to reorder pages or to designate a different page as the site's homepage from
  here; Home is always the fixed system page.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo una página nueva para mi tienda?`
- `¿Qué diferencia hay entre "Activo" y "Publicado"?` Ninguna verificada en el comportamiento de
  publicación actual: ambas se incluyen igual al publicar el dominio.
- `¿Por qué no puedo editar la página de Inicio o Nosotros?` Son páginas del sistema; sólo su
  contenido es editable desde el lápiz, no su nombre/ruta/estado.
- Search terms: `crear página`, `nueva página`, `ruta`, `estado de página`, `activo`, `publicado`.

<!-- DOC-ID: capability.delete-page -->
## Remove a page (Eliminar una página)

Opening an existing user-created page and using the modal's **Delete (Eliminar)** action asks for
confirmation (`¿Eliminar "<Name>"?`) and, once confirmed, resubmits the loaded form with its
status forced to 0 (removed). This is a real server-side soft delete, not only a local hide: the
page's Status becomes 0, it disappears from this list and from the site's next publish, and its
Route becomes available again for a different page. System pages cannot be deleted — their cards
never open the editable form.

### Limitations (Limitaciones)

There is no restore/undelete action on this page for a page removed this way; getting it back
means creating a new page record (with a new ID) and, if wanted, its content again in the builder.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo elimino una página que ya no necesito?`
- `¿Puedo recuperar una página eliminada?` No hay una opción de restaurar en esta página.
- Search terms: `eliminar página`, `borrar página`, `quitar página`.

<!-- DOC-ID: capability.edit-page-content -->
## Open the page builder to edit content (Editar el contenido de la página)

### User intention (Intención del usuario)

Design or change what a page actually shows: its sections (hero, banners, featured products,
testimonials, etc.), each section's text/images/icons, and their layout.

### Where to find it (Dónde encontrarlo)

Hover any card on the Pages tab and click the pencil (visible on hover) to navigate to the
dynamic builder route `/webpage-builder/<PageID>`. This works for system-page cards too — only
their Name/Route/Status stay read-only; their content is fully editable. This builder route is
not itself a menu entry; it is reached only from this pencil action (or, for the default Home
page, from the bare `/webpage-builder` redirect).

### Required information and prerequisites (Requisitos previos)

None beyond the page already existing in this list, or being one of the fixed system pages.

### Business rules and rationale (Reglas y razón de negocio)

The builder loads the page's stored sections and color palette. It lets sections be added from a
template library, reordered by drag-and-drop, edited (text, images, icons, and — for
product/category sections — which category feeds them) through per-section panels, or edited
through the shared AI assistant chat by describing the change in a prompt (its "Build page" mode
targets the whole page, and "Edit section" targets only the section currently selected). A
desktop/mobile preview toggle renders the assembled page at each width.

The builder's own **Config** tab currently only displays the page's palette as reference swatches
(each one's index, e.g. `color="3"`, is what an AI prompt can reuse to pick an existing color) and
shows a "Global Configuration" placeholder message with no working fields yet. There is no
manual color-picker to add or edit a palette color from this tab — new colors are only added
automatically when the AI assistant introduces one while editing a section.

### Result and side effects (Resultado y efectos)

Saving inside the builder (not on this Pages list) persists every current section plus the page's
palette and, in the background, captures a screenshot of the canvas and uploads it as this page's
thumbnail image (the one shown back on the Pages tab). Saving is skipped as a no-op, with a
notice, when nothing changed since the last load/save.

### Limitations (Limitaciones)

- Editing and saving the Store card's content in the builder does not currently reach the live
  storefront: only Home, About, and user-created pages (ID ≥ 15) are included when the domain is
  (re)published; Store, Product, and Cart are resolved dynamically by the storefront itself
  rather than from saved builder content.
- The palette cannot currently be edited by hand from the builder's Config tab (see above).

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo edito el contenido/las secciones de una página?` Usa el lápiz sobre la tarjeta de la
  página.
- `¿Puedo cambiar los colores de la página desde Config?` Hoy esa pestaña sólo muestra la paleta
  como referencia; no hay un selector para editarla manualmente.
- `¿Por qué mi página "Tienda" no cambia en el sitio publicado aunque edité su contenido?` Tienda
  (y Producto/Carrito) se resuelven dinámicamente en la tienda, no desde el contenido guardado del
  constructor.
- Search terms: `constructor`, `builder`, `editar contenido`, `secciones`, `paleta de colores`,
  `miniatura`, `thumbnail`.

<!-- DOC-ID: capability.site-domain -->
## Configure the storefront domain (Configurar el dominio de la tienda)

### User intention (Intención del usuario)

Set or change the subdomain the store is published on, and publish the current pages onto it.

### Where to find it (Dónde encontrarlo)

Pages page → **Config** tab → **Domain (Dominio)** field and its Save action.

### Required information and prerequisites (Requisitos previos)

Enter only the subdomain label; the fixed zone suffix (for example `.un.pe`) is appended
automatically and shown next to the field. The subdomain may contain only lowercase letters,
digits, and hyphens (no leading or trailing hyphen), up to 63 characters.

### Business rules and rationale (Reglas y razón de negocio)

Saving a genuinely new or changed domain is subject to a 20-minute cooldown counted from the last
domain change, so as not to burn through the Cloudflare hostname-registration quota by repeatedly
registering and releasing hostnames. Re-saving the same domain again — for example to retry a
failed publish — is not treated as a change and is never blocked by the cooldown.

The save is synchronous: it registers/validates the hostname, saves it, and renders and publishes
every currently eligible page onto it before the request returns — this is why the page shows a
loading spinner for a few seconds. Only after the new hostname successfully serves content does
Genix release the previous domain in Cloudflare, so a failed publish leaves the old domain still
serving the store instead of taking it down.

### Result and side effects (Resultado y efectos)

On success, Genix stores the new domain and republishes the storefront: Home, About, and every
active/published user-created page (ID ≥ 15) get rendered onto that hostname. The response
reports the domain saved, the number of pages published, and a build identifier.

### Limitations (Limitaciones)

A failed render leaves the domain saved but not confirmed published; re-saving the same domain
retries only the render step. This field manages one domain per company; there is no visible
history of previous domains beyond the automatic cleanup of the one just replaced.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo cambio el dominio o subdominio de mi tienda?`
- `¿Por qué me pide esperar minutos para volver a cambiar el dominio?` Es un límite de 20 minutos
  para no agotar la cuota de registro de hostnames en Cloudflare.
- `¿Por qué tarda unos segundos al guardar el dominio?` Porque el guardado publica la tienda de
  forma síncrona antes de responder.
- Search terms: `dominio`, `subdominio`, `publicar tienda`, `dominio propio`.

<!-- DOC-ID: capability.site-seo -->
## Configure SEO metatags (Configurar metatags SEO)

### User intention (Intención del usuario)

Set the storefront's search-engine and social-sharing metadata: title, description, keywords,
Open Graph title/description/image, and favicon.

### Where to find it (Dónde encontrarlo)

Pages page → **Config** tab → **SEO Metatags** fields and its Save SEO action.

### Required information and prerequisites (Requisitos previos)

All fields (Title, Description, Keywords, OG Title, OG Description, OG Image URL, Favicon URL)
are free text; none are required by this form.

### Business rules and rationale (Reglas y razón de negocio)

Only these known SEO keys are persisted — the endpoint ignores any other key — and they are
stored separately from the domain, so saving SEO never touches or re-publishes the domain. These
metatags are company-wide: one set for the whole storefront, not per page.

### Result and side effects (Resultado y efectos)

Saved metatags are read by the public storefront's unauthenticated content read and by the
prerender build, so they take effect the next time a page is rendered/published; saving SEO alone
does not trigger a new render.

### Limitations (Limitaciones)

There is no per-page SEO override here — Title/Description/OG/favicon apply to the whole site.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo configuro el título y la descripción SEO de mi tienda?`
- `¿Puedo poner un SEO distinto para cada página?` No desde esta página; el SEO configurado aquí
  es único para todo el sitio.
- Search terms: `SEO`, `metatags`, `título`, `descripción`, `open graph`, `favicon`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- The access catalog's "Páginas Web" entry only offers two levels for this page's save-type
  actions — View and Full — with no intermediate Create/Edit granularity: granting someone the
  ability to save a page, upload its thumbnail, edit its content, or change the domain/SEO always
  means Full control of all of them together, not a partial slice. Viewing this page's list and
  the Config tab's current values has no separate access mapped, so it is open to any
  authenticated user of the company.
- Everything under this route — page records, builder content, domain, and SEO — is scoped to the
  fixed system pages (10-14) plus the user-created pages (≥15) of the current company; none of it
  is per-branch/per-site.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El nombre de la página debe tener al menos 3 caracteres":** lengthen the Name field.
- **"La ruta debe iniciar con "/" y no puede ser la raíz":** fix the Route to start with "/" and
  not be exactly "/".
- **"La ruta ... está reservada por una página del sistema" / "... ya está en uso por otra
  página":** pick a Route that is not `/`, `/about`, `/store`, `/product`, `/cart`, and not
  already used by another active/published page.
- **"Las páginas del sistema no se pueden editar":** appears only if a system page's metadata
  were sent for saving; use the pencil to edit its content instead.
- **A saved page never shows up on the live site:** confirm the Domain was (re)saved on the
  Config tab afterward — creating or editing a page here, or its content in the builder, does not
  by itself republish the storefront.
- **Editing "Tienda" content doesn't change the storefront:** expected — Store/Product/Cart are
  dynamic pages, not builder-rendered content.
- **Domain save rejected with a wait-time message:** another domain change happened recently;
  wait the stated number of minutes.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **The per-page builder (`/webpage-builder/<PageID>`)** is not a menu entry; it is reached only
  from this page's pencil action and owns section-level editing, the color palette display, and
  the AI-assisted "Build page"/"Edit section" chat modes.
- **Gallery (`/webpage-builder/gallery`)** manages the uploaded images available to sections in
  the builder; not covered by this document.
- **Security → Users (`/security/users`)** is the source of the "updated by" name shown on each
  page card.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, capability.browse-pages, related-pages]
  - path: frontend/routes/webpage-builder/pages/+page.svelte
    role: page
    hash: sha256:162096e4c59b6febb4d2776f780c0f4035aedbbdc53f3be27968edfc926b29ee
    supports: [page-purpose, concepts, capability.browse-pages, capability.create-edit-page, capability.delete-page, capability.edit-page-content, rules, troubleshooting]
  - path: frontend/routes/webpage-builder/pages/WebpageConfig.svelte
    role: user-interface
    hash: sha256:41531f442aa70fe5799a67c5a7fd42894e45991ea4bbc38432c749d4b0188bad
    supports: [capability.site-domain, capability.site-seo]
  - path: frontend/services/webpage/pages.svelte.ts
    role: frontend-service
    hash: sha256:c4e82dfc04bc81b8bb05c54022ba67a250f7d9c754f7e4df5d0af1325eacf3b6
    supports: [concepts, capability.browse-pages, capability.create-edit-page, capability.delete-page, capability.edit-page-content, capability.site-domain, capability.site-seo]
  - path: frontend/routes/webpage-builder/+page.ts
    role: page
    hash: sha256:17dd56e3cd9ba0b601d4c67d337f8afdc48002bbf4061278277daa0beec4ecfd
    supports: [concepts, capability.edit-page-content]
  - path: frontend/routes/webpage-builder/[pageID]/+page.svelte
    role: page
    hash: sha256:d4f89148e09bc180bc2e93c2db28f6e084df2692c8df79765598033021a85490
    supports: [capability.edit-page-content]
  - path: frontend/routes/webpage-builder/builder/EcommerceBuilder.svelte
    role: user-interface
    hash: sha256:650b104998658c50b2c46c77943b3d6e327e691eb4bd84a18fea9b76936b24a1
    supports: [capability.edit-page-content]
  - path: frontend/routes/webpage-builder/builder/SectionEditorLayer.svelte
    role: user-interface
    hash: sha256:d1e93f061850194591597dd48e31f6cdb3b851b6c8b8fe250cd97d1fcf4055dc
    supports: [capability.edit-page-content]
  - path: frontend/routes/webpage-builder/components/ConfigTab.svelte
    role: user-interface
    hash: sha256:5442ca5df949e6063aa17b6b30a0150b48a6bccf43462726700be60c123d0161
    supports: [capability.edit-page-content]
  - path: frontend/routes/webpage-builder/components/EditorTab.svelte
    role: user-interface
    hash: sha256:cb09eddda7fe8ff24523a537a545ef9f477db96c6a371faad8d0a1fce851f4ba
    supports: [capability.edit-page-content]
  - path: frontend/routes/webpage-builder/stores/editor.svelte.ts
    role: frontend-service
    hash: sha256:f9e7660e30133ff85452f7ef9df886820e10c762cc6aeebe2633dbbd9397ba7b
    supports: [concepts, capability.edit-page-content]
  - path: frontend/services/ecommerce/page-content.svelte.ts
    role: frontend-service
    hash: sha256:3f6b05fb83da69b323a77c74dcba937c2a10d709770a3b6875e062a90c47b8c7
    supports: [capability.edit-page-content]
  - path: frontend/routes/security/users/users.svelte.ts
    role: shared-domain
    hash: sha256:bcce5c06748b895562d680a2e698f082a5fc484247e96259fe48b89f16cdd597
    supports: [related-pages]
  - path: backend/webpage/webpage_pages.go
    role: backend-handler
    hash: sha256:4556f16d7bf4808ee3007bd81a46493a7e8d2d770d3238da79a9f1376c73f72d
    supports: [capability.browse-pages, capability.create-edit-page, capability.delete-page, rules, troubleshooting]
  - path: backend/webpage/types/webpages.go
    role: data-model
    hash: sha256:937db48df8b991e037cf1cbaf32447f6673969c3d540660c7048083e8cb54dc3
    supports: [concepts, capability.create-edit-page, capability.delete-page]
  - path: backend/webpage/webpage_showcase.go
    role: backend-handler
    hash: sha256:6a56679fe9bddb197c002c145ebc613e75a1639b01aa24d2938a7f29073318be
    supports: [capability.edit-page-content]
  - path: backend/webpage/webpage_config.go
    role: backend-handler
    hash: sha256:747c8286f496b2eb9c1ba6762139384ff12ccbdaf1d33e32f2e5a3764df7172e
    supports: [capability.site-domain, capability.site-seo]
  - path: backend/webpage/webpage_render.go
    role: business-logic
    hash: sha256:fa1be448cb0dbcbf4b5663ea3414808b1c4834e1bd34f8ffd78d389055d4a85d
    supports: [concepts, capability.create-edit-page, capability.edit-page-content, capability.site-domain]
  - path: backend/webpage/webpage_public.go
    role: backend-handler
    hash: sha256:860df08cc3c0a9c2afc4dcde95ba22683878bb0b99abe2cf53618d710918838c
    supports: [capability.site-seo]
  - path: backend/webpage/page_content.go
    role: backend-handler
    hash: sha256:61983a447011932a81ddb4dbe55dad359b266bba58fbfe78dca50d6210a689f2
    supports: [capability.edit-page-content]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [rules]
```
