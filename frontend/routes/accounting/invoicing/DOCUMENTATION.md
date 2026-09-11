---
schema: 1
page_id: accounting.invoicing
route: /accounting/invoicing
title: Electronic Invoicing (Facturación Electrónica)
status: implemented
visibility: tenant
description_en: >-
  Register of the electronic documents (comprobantes) the company has issued: filter them by
  issue date, state or id, see whether each one reached SUNAT and what SUNAT answered,
  download its signed XML and its CDR, review the sale it bills, and push a document out
  now instead of waiting for the scheduled send.
description_es: >-
  Registro de los comprobantes electrónicos emitidos por la empresa: filtrarlos por fecha de
  emisión, estado o ID, ver si llegaron a SUNAT y qué respondió SUNAT, descargar el XML
  firmado y la CDR, revisar la venta que facturan, y enviar un comprobante ahora en vez de
  esperar el envío programado.
---

# Electronic Invoicing (Facturación Electrónica)

<!-- DOC-ID: page-purpose -->
## Page purpose

Electronic Invoicing (`Facturación`) is the register of the **comprobantes electrónicos** —
facturas and boletas — that the company has issued. It answers one question above all: *¿se
envió a SUNAT o no?* For each document it shows the number, the date, the sale it bills, the
amount, its state, and whatever SUNAT replied.

This page **does not create comprobantes**. A comprobante is created automatically together
with its sale at the Point of Sale (`Punto de Venta`, `/sales/sale_order_create`), at the
moment the operator picks a series like `FACTURA · F001` and presses **Generar**. What this
page owns is everything that happens afterwards: watching the document travel to SUNAT,
downloading its files, and pushing it out when it is still waiting.

It also does not maintain the series (`Series de Facturación`) or the SUNAT credentials —
those live in **Mi Empresa → Configuración**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **comprobante** here is one electronic document: a **factura** (serie que empieza con F,
  para clientes con RUC) or a **boleta** (serie que empieza con B). Its visible number is the
  series code and the correlativo together, for example `F001-45`.
- The **correlativo** is the document's number inside its series. It is reserved the instant
  the sale is generated, before anything is sent, so the numbering is decided at the till and
  never changes afterwards.
- A comprobante and its sale (`venta`) are **one to one and share the same id**: the sale
  `550301` is billed by the comprobante `550301`. That is why the **Venta** column doubles as
  the sale's id, and why searching by either number finds the same row.
- **Estados** a comprobante passes through, in the order the system numbers them:
  - **Anulado** — the sale was annulled before the comprobante was sent. It never went out,
    and its correlativo stays spent (see *Reglas generales*).
  - **Pendiente de envío** — created and numbered, waiting for the scheduled send. This is
    the state every comprobante starts in.
  - **Enviando** — signed and on its way to SUNAT.
  - **Aceptado** — SUNAT accepted it. This is the finished, valid state.
  - **Aceptado con observaciones** — accepted and valid, but SUNAT noted something worth
    reading in the observations.
  - **Rechazado** — SUNAT read the document and refused it. It will never be accepted as it
    stands; a corrected document is required, not another attempt.
  - **Error de envío** — it did not reach SUNAT (network, credentials, certificate). This one
    *is* retryable.
- The **XML** is the signed document; the **CDR** (`Constancia de Recepción`) is SUNAT's
  signed answer to it. Both must be kept for five years and both are downloadable here.

<!-- DOC-ID: capability.filter-invoices -->
## Find comprobantes by date, state or id (Buscar comprobantes)

### User intention (Intención del usuario)

Answer "¿qué se emitió entre estas dos fechas?", "¿qué quedó pendiente de envío?" or "¿dónde
está el comprobante de la venta 550301?".

### Where to find it (Dónde encontrarlo)

**Contabilidad → Facturación**. The filter row sits above the table: **Desde** and **Hasta**
(dates), **Estado** (a selector that starts on *Todos los estados*), and a search box labelled
**ID de comprobante o venta**.

### Required information and prerequisites (Requisitos previos)

Nothing. Every filter is optional and they combine: leaving all of them empty lists every
comprobante the company has issued, newest first.

### Business rules and rationale (Reglas y razón de negocio)

**Desde** and **Hasta** are inclusive and compare against the **fecha de emisión**, which is
the day the sale was generated — not the day SUNAT answered. The id box matches the
comprobante's id, the sale's id and the correlativo, and it matches partially, so typing `5503`
finds every number that contains it.

When any comprobante is **Pendiente de envío**, the count appears at the right of the filter
row (`N pendientes de envío`) so the situation is visible without filtering for it.

### Result and side effects (Resultado y efectos)

Filtering only changes what the table shows. Nothing is written.

### Limitations (Limitaciones)

There is no filter by customer or by amount, and no export. The table lists documents, not
sales: a sale generated with **SIN COMPROBANTE** never produces a row here.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo veo qué comprobantes no se han enviado a SUNAT?` Filtre **Estado** en *Pendiente de
  envío*, o mire el contador de pendientes.
- `¿Puedo buscar por el número de la venta?` Sí — la venta y el comprobante comparten el ID.
- Search terms: `comprobante`, `factura`, `boleta`, `SUNAT`, `correlativo`, `serie`,
  `pendiente`, `emitido`, `CDR`.

<!-- DOC-ID: capability.review-invoice -->
## Review a comprobante and the sale it bills (Ver el detalle)

### User intention (Intención del usuario)

Open one comprobante to see its state, what SUNAT said about it, and what was actually sold.

### Where to find it (Dónde encontrarlo)

Click any row in the table. A panel opens on the right, titled with the document number
(`F001-45`).

### Required information and prerequisites (Requisitos previos)

None beyond the comprobante existing.

### Business rules and rationale (Reglas y razón de negocio)

The panel shows the **Estado**, the **Fecha de Emisión**, the **Total** and the **IGV**. When
the document has been sent, a **Respuesta de SUNAT** block appears with SUNAT's **Código**, the
number of **Intentos**, SUNAT's observations, and the last error when the send failed. That
block is absent while nothing has been answered yet, because there is nothing to show.

Below it, the **Venta** section resolves the sale by its id and lists its lines — quantity,
product and amount — plus the sale's date and total.

### Result and side effects (Resultado y efectos)

Reading only. The panel writes nothing.

### Limitations (Limitaciones)

An **annulled sale is no longer readable from here**: the panel says *La venta está anulada o
ya no está disponible* and shows no lines. That is the normal case for a comprobante in state
**Anulado**, whose sale was annulled by design. The panel does not show the customer's name or
RUC — those are on the sale itself.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué SUNAT rechazó mi factura?` Abra el comprobante y lea el **Código** y las
  observaciones en *Respuesta de SUNAT*.
- `¿Qué se vendió en este comprobante?` La sección **Venta** del panel lista las líneas.
- Search terms: `detalle`, `respuesta de SUNAT`, `código`, `observaciones`, `rechazo`.

<!-- DOC-ID: capability.download-invoice-files -->
## Download the XML and the CDR (Descargar XML y CDR)

### User intention (Intención del usuario)

Get the signed document, or SUNAT's receipt for it, to send to the customer or to hand to the
accountant.

### Where to find it (Dónde encontrarlo)

Two buttons, **XML** and **CDR**, inside the detail panel.

### Required information and prerequisites (Requisitos previos)

The **XML** exists from the moment the document was signed, which is the step that takes it out
of *Pendiente de envío*. The **CDR** only exists once SUNAT answered. Each button is disabled
while its file does not exist yet — a comprobante *Pendiente de envío* or *Anulado* has neither.

### Business rules and rationale (Reglas y razón de negocio)

Both files are stored the day they are produced and kept for five years. The download names the
file after the document number, for example `F001-45-xml.xml` and `F001-45-cdr.zip`. The CDR
arrives as a zip because that is exactly what SUNAT returns.

### Result and side effects (Resultado y efectos)

The file is downloaded. Nothing changes on the comprobante.

### Limitations (Limitaciones)

There is no printable representation (`representación impresa`) — no PDF and no HTML ticket,
only the two files SUNAT's process produces. A rejected document has both files: the XML that
was sent and the CDR explaining the refusal.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Dónde descargo el XML de la factura?` Abra el comprobante y presione **XML**.
- `¿Por qué el botón CDR está deshabilitado?` Porque SUNAT todavía no ha respondido.
- Search terms: `XML`, `CDR`, `constancia`, `descargar`, `archivo`, `zip`.

<!-- DOC-ID: capability.send-invoice-now -->
## Send a comprobante now (Enviar ahora)

### User intention (Intención del usuario)

Not wait for the scheduled send — the customer is standing there — or try again after a send
failed.

### Where to find it (Dónde encontrarlo)

The **Enviar ahora** button at the top of the detail panel. It is only rendered when the
comprobante's state allows it.

### Required information and prerequisites (Requisitos previos)

The comprobante must be in **Pendiente de envío**, **Enviando** or **Error de envío**. The
company must have its SUNAT credentials and certificate loaded, or the send fails with a
configuration error.

### Business rules and rationale (Reglas y razón de negocio)

The button never appears for **Aceptado**, **Aceptado con observaciones** or **Rechazado** —
SUNAT already answered and the answer is final. It never appears for **Anulado** either: a
voided comprobante must never reach SUNAT.

**The send waits for SUNAT.** The screen stays on *Enviando a SUNAT, puede demorar un momento*
until SUNAT answers, which regularly takes tens of seconds, and then reports the verdict —
*SUNAT aceptó el comprobante*, *SUNAT lo aceptó con observaciones*, or the refusal SUNAT
returned. The panel stays open on the answer instead of closing, so a rejection code can be
read where it appears.

The comprobante keeps the number it already has. Re-sending never consumes a new correlativo.

### Result and side effects (Resultado y efectos)

The document is signed, transmitted, and its state updated with SUNAT's answer before the
screen comes back. Its XML — and its CDR, if SUNAT answered — become downloadable immediately.

### Limitations (Limitaciones)

A **Rechazado** cannot be re-sent by any means: the same document will be refused forever, and
the correct response is a corrected document, which this system cannot issue yet (see
*Problemas comunes*).

Because the request waits, a very slow SUNAT can make the browser give up before the answer
arrives. **That does not undo anything**: the send runs to completion on the server and the
document's state is written there, so reopening the page shows where it actually landed.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo envío una factura ahora mismo?` Abra el comprobante y presione **Enviar ahora**; la
  pantalla espera la respuesta de SUNAT.
- `¿Por qué demora tanto al enviar?` Porque se espera a que SUNAT conteste, y SUNAT suele
  tardar decenas de segundos.
- `¿Se gasta otro correlativo si reenvío?` No: conserva su número.
- Search terms: `enviar`, `reenviar`, `pendiente`, `reintento`, `demora`, `respuesta de SUNAT`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- **A comprobante is born with its sale.** Generating a sale under a series creates the
  comprobante, numbered, in *Pendiente de envío*. If the comprobante cannot be created — a
  factura without a client with RUC, for example — **the sale itself is refused**, so a sale
  registered under a series always has its comprobante.
- **The send is deferred and automatic.** A scheduled process picks up everything in
  *Pendiente de envío* and sends it, so under normal operation nobody has to press anything
  here. A send that fails for transport reasons is retried on its own a limited number of
  times before it stops and waits for a person.
- **Annulling a sale voids its comprobante, but only while it has not been sent.** Once SUNAT
  has it, the sale can no longer be annulled: it requires a **nota de crédito**, which this
  system does not emit yet, and Sales Management will refuse the annulment naming the series
  and correlativo of the document in the way.
- **A voided correlativo leaves a gap in the series.** The number is not reused. SUNAT expects
  such a gap to be declared through a **comunicación de baja**, which Genix does not generate
  yet — that declaration has to be handled outside the system for now.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **A comprobante has been *Pendiente de envío* for a long time.** The scheduled send runs in
  frames of a few minutes; if it stays pending well past that, press **Enviar ahora** — it
  waits for SUNAT and reports the actual error instead of leaving it pending.
- **The send fails with a certificate or credentials error.** The company's SUNAT
  configuration is incomplete or expired. Fix it in **Mi Empresa → Configuración** and press
  **Enviar ahora** again — the comprobante keeps its number.
- **`La empresa no tiene ciudad configurada: selecciónela en Mi Empresa para poder emitir`.**
  Every comprobante declares the issuer's fiscal address (`domicilio fiscal`), and SUNAT validates
  its ubigeo along with the department, province and district. Pick the district in **Mi Empresa →
  Configuración**, field **Departamento | Provincia | Distrito**, and press **Enviar ahora** again —
  the comprobante keeps its number.
- **SUNAT accepted the comprobante but the panel shows observations 4093, 4096, 4097 or 4098.**
  Those are exactly the fiscal-address checks above, on a document issued before the district was
  set. The document is valid — SUNAT accepted it — and the observations stop appearing on the ones
  issued afterwards.
- **SUNAT rejected the document (*Rechazado*).** Read the code and the observations in the
  detail panel. The document cannot be re-sent; the sale has to be corrected with a nota de
  crédito, which this version does not issue.
- **The sale cannot be annulled because it has a comprobante emitido.** That means the
  document already reached SUNAT. This is not a bug and there is no override on this page.
- **A comprobante appears as *Anulado*.** Its sale was annulled before the document was sent.
  Nothing was declared to SUNAT, and the correlativo stays consumed.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Punto de Venta** (`/sales/sale_order_create`) is where a comprobante is actually created,
  by choosing a series before pressing **Generar**.
- **Gestión Ventas** (`/sales/sale_orders_status`) is where a sale is annulled — and where the
  refusal appears when its comprobante has already been sent.
- **Mi Empresa → Configuración** maintains the **Series de Facturación** and the SUNAT
  credentials and certificate that every send depends on.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:010936e67bd5d99e9bbf9916814ed818def3e45451511301c46f070e133d6231
    supports: [page-purpose, related-pages]
  - path: frontend/routes/accounting/invoicing/+page.svelte
    role: page
    hash: sha256:f1bcb001b1bfbca510266d11f6d2d9959d3cd1194e280769c514532f8fcb689e
    supports: [page-purpose, capability.filter-invoices, capability.review-invoice, capability.send-invoice-now]
  - path: frontend/routes/accounting/invoicing/InvoiceDetailLayer.svelte
    role: user-interface
    hash: sha256:aa88bd4959a720c59008606b1b1b1ddc5d5ea040dab377220286d10ec029e0b2
    supports: [capability.review-invoice, capability.download-invoice-files, capability.send-invoice-now]
  - path: frontend/routes/accounting/invoicing/invoicing.ts
    role: business-logic
    hash: sha256:2c03d923e62b57c61680efcaa989adc2b9787888aa29eee25a94ddf120cde684
    supports: [concepts, capability.filter-invoices, capability.download-invoice-files, capability.send-invoice-now]
  - path: frontend/routes/accounting/invoicing/invoicing.svelte.ts
    role: frontend-service
    hash: sha256:c6f540a25230e7fd19691cfbfe5cc3c826166c7631dd984b0283516eb4694e23
    supports: [capability.download-invoice-files, capability.send-invoice-now]
  - path: backend/invoicing/invoice_api.go
    role: backend-handler
    hash: sha256:a34cf6f0c5d9bf80f1fec5fdfb7261eabaced339c92ccdeee1c0988cbef04c22
    supports: [capability.filter-invoices, capability.download-invoice-files, capability.send-invoice-now, troubleshooting]
  - path: backend/invoicing/issuer.go
    role: business-logic
    hash: sha256:221496f3d5d892a8594edb17aed62d7df4986c234560aa4735664789a48fe72f
    supports: [troubleshooting, capability.send-invoice-now]
  - path: backend/invoicing/emit_worker.go
    role: business-logic
    hash: sha256:a71d60c3600d7b7bea9a0073d566406e0854b0cdf9f43ce80d25c5aa40f36c7f
    supports: [concepts, rules, capability.send-invoice-now, troubleshooting]
  - path: backend/invoicing/types/invoice_document.go
    role: data-model
    hash: sha256:499902303432fcd04f18fca719230f9d948fb6af02b3cd5d005ece7c2a3d152d
    supports: [concepts, capability.review-invoice]
  - path: backend/invoicing/types/document_reserve.go
    role: business-logic
    hash: sha256:e03710c72bede974f98202a02af67ec73bf8cf9706feedb173ce7827157f50e1
    supports: [concepts, rules]
  - path: backend/sales/sale_order_issuance.go
    role: business-logic
    hash: sha256:fb10b3dfed01ff5b2fa20f86b35eb0f60d994388bb87ff457e718e05cde4facb
    supports: [rules, page-purpose]
  - path: backend/sales/sale_order_annul.go
    role: business-logic
    hash: sha256:bdbec7379072bcb6ab06c1a6ceafda748e8547f463963339e4e2579cc08b377e
    supports: [rules, troubleshooting]
  - path: backend/invoicing/types/invoice_lookup.go
    role: business-logic
    hash: sha256:6e8c81b134b11362bf751f57f57e954e21b0fdf502acf8b2f3c3e0b60b2ad87b
    supports: [rules]
  - path: backend/access.toml
    role: permissions
    hash: sha256:a9de756bd703f10b679f6877d83eb5fde0544f772ff8d558c598d02716c3ebc1
    supports: [page-purpose]
```
