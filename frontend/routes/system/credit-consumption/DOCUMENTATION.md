---
schema: 1
page_id: system.credit-consumption
route: /system/credit-consumption
title: Credit Usage (Consumo Créditos)
status: implemented
visibility: saas
description_en: >-
  Per-company credit monitoring, SaaS only. One card per company with its CPU credits, refusals and
  distinct IP-devices in five-minute frames over the last four hours; opening a card lists the
  IP-devices that sent the most requests, with the browser, system and locale decoded from their
  device signature.
description_es: >-
  Monitoreo de créditos por empresa, exclusivo SaaS. Una tarjeta por empresa con sus créditos CPU,
  negaciones e IP-dispositivos distintos cada cinco minutos en las últimas cuatro horas; al abrir una
  tarjeta se listan los IP-dispositivos con más solicitudes, con el navegador, sistema e idioma
  decodificados de su firma de dispositivo.
---

# Credit Usage (Consumo Créditos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Credit Usage (`Consumo Créditos`) shows, company by company, how much each tenant consumed in the
last four hours and whether it was refused. Open **System → Observability → Credit Usage (Sistema →
Observabilidad → Consumo Créditos)** at `/system/credit-consumption`. Only the company that
administers the SaaS platform sees it. The thirty-day history per company and per user is in
**Companies (Empresas)**; route-level activity and errors are in **Services (Servicios)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Credits (`créditos`)** measure charged API work, as everywhere else in the platform.
- **Refusals (`negaciones`)** are requests the credit limiter turned away. Two causes are counted:
  - credit refusals: the company was over its short or long window, or out of daily or monthly
    credits;
  - device refusals: the company had already used, in the short or long window, as many distinct
    IP-devices as the platform allows.
- **IP-device (`IP-dispositivo`)** is the pair of the address a request came from and the device
  signature its browser sent: a 64-bit number packing browser, version, operating system, time
  zone, language or country, screen, cores, pixel ratio and memory. The same browser on the same
  network is one IP-device; the same signature from two addresses is two.

<!-- DOC-ID: capability.monitor-companies -->
## Monitor companies (Monitorear empresas)

Each card shows the company's CPU and AI credits, its total refusals, and the most distinct
IP-devices seen in any single five-minute frame. The chart has one bar per five-minute frame for CPU
credits, a red line for refusals and a purple line for distinct IP-devices; the two lines use their
own scale. Cards are ordered by CPU credits. **Filter companies (Filtrar empresas)** matches the
company name or number. The page refreshes every 15 seconds while its browser tab is visible.

Common questions: `¿qué empresa consume más créditos?`, `¿a qué empresa se le están negando
solicitudes?`, `¿cuántos dispositivos usa una empresa?`, `bloqueos por dispositivos`, `429`.

<!-- DOC-ID: capability.inspect-devices -->
## Inspect a company's IP-devices (Ver IP-dispositivos)

Click a card to open the side panel. It splits the company's refusals into credit and device
refusals and lists the IP-devices with the most requests in the four hours: address, decoded
device (browser and version, operating system, locale, screen, time zone and hardware), requests,
CPU credits and the number of five-minute frames the pair was active in. A device sent without a
signature shows as **No signature (Sin firma)**. A device whose browser reports automation
(Selenium, Puppeteer, headless browsers, known bots) is marked **automated (automatizado)**.

<!-- DOC-ID: rules -->
## Business rules (Reglas)

- Only requests that pass through the HAProxy edge carry an IP-device; a company served only
  through other paths shows credits and credit refusals but no IP-devices.
- Each five-minute frame stores its twelve busiest IP-devices. The four-hour figures in the side
  panel add those frames up: exact for the busy pairs the table is about, a lower bound for pairs
  that were not always among the twelve.
- The distinct IP-devices of a frame count every pair served in it, not only the twelve stored.
- The current five-minute frame keeps changing until it closes, because the limiter writes its
  figures every 15 seconds.
