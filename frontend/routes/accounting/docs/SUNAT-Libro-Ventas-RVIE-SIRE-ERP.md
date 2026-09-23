# Libro de Ventas SUNAT, agrupación de ventas, RVIE/SIRE e integración con ERP

Documento de referencia para contribuyentes y equipos que desarrollan un ERP que emite comprobantes desde **sistemas del contribuyente** e integra el **Registro de Ventas e Ingresos Electrónico (RVIE)** a través del **SIRE**.

- Tema: SUNAT Perú
- Enfoque: Libro de ventas, consolidación de operaciones, SIRE/RVIE/RCE y arquitectura de un ERP
- Fecha de elaboración: 15 de septiembre de 2026
- Uso: guía operativa y técnica. No sustituye la norma vigente ni el criterio de un contador o abogado tributarista.

Fuentes principales consultadas: portal gob.pe / SUNAT, orientación.sunat.gob.pe, cpe.sunat.gob.pe, sire.sunat.gob.pe, resoluciones de superintendencia (entre ellas RS 286-2009/SUNAT, RS 112-2021/SUNAT, RS 042-2021/SUNAT, RS 138-2023/SUNAT, RS 125-2026/SUNAT) y el Reglamento del IGV.

---

## 1. Respuesta corta

**¿Se pueden agrupar ventas en el Libro de Ventas SUNAT?**

Sí, pero no todas ni de cualquier forma.

- Se puede **consolidar por día** ciertas operaciones que **no otorgan crédito fiscal** (sobre todo boletas y tickets), si existe un sistema computarizado que conserve el detalle de cada documento.
- **No** se consolidan facturas.
- **No** se agrupa por semana ni por mes.
- En **SIRE/RVIE**, si todo se emite electrónico y SUNAT lo recibió, casi no hace falta agrupar a mano: la propuesta ya viene documento por documento.

**¿Qué es el RVIE/SIRE?**

El SIRE es la plataforma de SUNAT para llevar, de forma electrónica y conjunta:

- **RVIE**: Registro de Ventas e Ingresos Electrónico (lo que vendes).
- **RCE**: Registro de Compras Electrónico (lo que compras).

SUNAT propone el registro con tus comprobantes electrónicos. Tú aceptas, complementas o reemplazas.

**¿Cómo aplica si el ERP emite desde sistemas del contribuyente?**

La emisión (SEE) y el libro (SIRE/RVIE) son **dos capas distintas**:

1. El ERP emite, firma y envía el XML (y el Resumen Diario de boletas).
2. SUNAT arma la propuesta del RVIE con lo que le llegó.
3. El ERP (o el contador) concilia esa propuesta y cierra el registro del mes.

---

## 2. Libro de Ventas / Registro de Ventas e Ingresos

### 2.1. Qué es

Es el registro obligatorio donde se anotan las ventas e ingresos del periodo. Sirve para:

- Control interno del negocio.
- Sustento del débito fiscal del IGV.
- Fiscalización de SUNAT.

Según el régimen y el nivel de ingresos, puede ir acompañado de Registro de Compras, Diario, Mayor, Inventarios y Balances, etc.

### 2.2. Qué documentos entran

Entre otros:

| Código típico | Documento |
|---|---|
| 01 | Factura |
| 03 | Boleta de venta |
| 07 | Nota de crédito |
| 08 | Nota de débito |
| 12 | Ticket o cinta de máquina registradora |
| 14 | Recibos de servicios públicos |
| 18 / 13 / 87 / 88 | Otros documentos previstos en las tablas SUNAT |

No todas las empresas emiten todos los tipos.

### 2.3. Información mínima (Reglamento del IGV)

En columnas separadas, como mínimo:

- Fecha de emisión
- Tipo de documento
- Serie
- Número correlativo
- RUC del cliente, cuando corresponda
- Base imponible, operaciones exoneradas/inafectas
- IGV / IPM, ISC y otros cargos
- Importe total
- Datos de la nota de crédito/débito y del comprobante que modifica

Las operaciones se anotan en el mes en que se realizan.

---

## 3. ¿Se pueden agrupar ventas?

### 3.1. Sí, con límites

La **anotación consolidada** está permitida para el **total diario** de operaciones que **no otorgan derecho a crédito fiscal**, siempre que:

1. La consolidación sea **por día** (no por semana ni por mes).
2. Se lleve un **sistema de control computarizado** que conserve el detalle y permita verificar cada documento.
3. Se respeten las estructuras del registro electrónico (campo 9 / número final del rango, cuando corresponda).

Base normativa habitual:

- Art. 10 del Reglamento del IGV (anotación consolidada de operaciones diarias que no dan crédito fiscal).
- Estructura del Registro de Ventas e Ingresos (formato 14.1) y modificaciones (p. ej. RS 042-2021/SUNAT y RS 138-2023/SUNAT).

### 3.2. Qué se puede consolidar por día

Normalmente:

- Boletas de venta (tipo 03), en especial electrónicas.
- Tickets o cintas de máquina registradora (tipo 12) que no dan crédito fiscal.
- Ciertos documentos tipo 00, 13, 18, 87 y 88, según las observaciones vigentes del campo 9.

Para tickets de máquina registradora que no dan crédito fiscal se puede anotar el **importe total del día por máquina**, consignando número de máquina y correlativos inicial y final.

### 3.3. Qué no se agrupa

- Facturas (tipo 01): van una por una.
- Notas vinculadas a facturas: van detalladas.
- Un solo renglón de “ventas de la semana” o “ventas del mes”.

### 3.4. Regla de los S/ 700

Desde el **1 de julio de 2016**, para boletas tipo 03:

- Si el monto total es **igual o mayor a S/ 700**, por regla general el registro debe ser **detallado** (con datos del cliente).
- En emisión: a partir de S/ 700 la boleta debe identificar al adquirente (tipo y número de documento), salvo excepciones.
- Matiz frecuente en orientación SUNAT:
  - Boletas **físicas** ≥ S/ 700: conviene / corresponde detalle en el registro.
  - Boletas **electrónicas**: se ha indicado que puede consolidarse el total diario, incluso con montos altos, si el sistema conserva el detalle.

En la duda, anota detallado todo lo ≥ S/ 700.

### 3.5. No confundir tres cosas distintas

| Concepto | ¿Se agrupa? | Para qué sirve |
|---|---|---|
| **Registro de Ventas (libro / RVIE)** | Sí, diario y solo ciertos documentos | Anotar ventas ante SUNAT |
| **Resumen Diario de Boletas** | Sí, es el mecanismo de envío | Informar a SUNAT las boletas electrónicas del día (hasta 7 días después) |
| **Emitir una sola boleta por todo el día** | Con límites muy estrechos | Es emisión de comprobante, no el libro |

Emitir 3 o 4 boletas de “venta varia” por el día es práctica frecuente en bodegas, pero **no reemplaza** emitir comprobante cuando corresponde (si el cliente lo pide o si el monto exige identificación). El riesgo principal no es el libro: es no emitir el comprobante correcto.

Ventas menores a S/ 5 que el cliente no pide comprobante pueden consolidarse en emisión, según el Reglamento de Comprobantes de Pago. Eso no autoriza dejar de emitir cuando la norma lo exige.

---

## 4. SIRE, RVIE y RCE

### 4.1. Definiciones

| Sigla | Nombre | Qué registra |
|---|---|---|
| **SIRE** | Sistema Integrado de Registros Electrónicos | La plataforma |
| **RVIE** | Registro de Ventas e Ingresos Electrónico | Ventas e ingresos (facturas, boletas, NC, ND, etc.) |
| **RCE** | Registro de Compras Electrónico | Compras y gastos con sustento |

El SIRE **no reemplaza todos los libros**. Solo sustituye el Registro de Ventas y el Registro de Compras que antes iban por PLE o Portal. Diario, Mayor e Inventarios, si corresponden, siguen por otro camino.

### 4.2. Diferencia frente al PLE

Antes:

1. El contribuyente armaba un TXT.
2. Lo validaba en el PLE.
3. Lo enviaba a SUNAT.

Ahora:

1. SUNAT ya tiene los comprobantes electrónicos.
2. Desde el **2.º día calendario** del mes siguiente pone una **propuesta** de RVIE (se actualiza con lo recibido el día anterior).
3. El contribuyente compara con su sistema.
4. Acepta, complementa o reemplaza.
5. Genera preliminar y luego el registro.
6. Con RVIE + RCE, SUNAT propone casillas del **Formulario 621** (IGV mensual).

El almacenamiento del RVIE queda a cargo de SUNAT.

### 4.3. Tres acciones sobre la propuesta

1. **Aceptar**  
   La propuesta coincide con lo emitido. No se modifica.

2. **Complementar**  
   La propuesta está bien, pero falta información: boletas físicas, tickets, documentos que SUNAT no vio. Se agregan a mano o por archivo plano.

3. **Reemplazar**  
   Hay muchas diferencias. Se sube un archivo plano con la estructura oficial. Si no hubo operaciones, el archivo puede ir en blanco (“sin movimiento”).

### 4.4. Flujo mensual del RVIE

1. Día 2 del mes siguiente: SUNAT publica / actualiza la propuesta (archivo plano).
2. Comparar con el sistema interno.
3. Aceptar, complementar o reemplazar.
4. Generar el **preliminar** (desde el **día 8** del mes siguiente; día 10 para ciertos recibos de servicios públicos).
5. **Generación de registros** en el módulo RVIE.
6. Constancia de recepción en el buzón SOL (fecha, hora, RUC, periodo, hash, cantidad de documentos).
7. Si hay error después: **ajustes posteriores** (rectificar lo anotado o incorporar lo omitido).

Si un mes no hay operaciones, igual se genera el registro vacío.

### 4.5. Cómo acceder

Tres vías:

- **SUNAT Operaciones en Línea (SOL)** — la más usada en negocios pequeños y medianos.
- **Aplicativo cliente SIRE** — escritorio, útil con más volumen.
- **Servicio web API SUNAT** — para que el ERP hable con SUNAT.

Se pueden combinar canales. Atención: en orientación oficial se ha señalado que la **generación final del registro** se realiza en el **portal SOL**, aunque la API cubre propuesta, cargas, preliminar, tickets e inconsistencias. Verificar siempre el manual API vigente.

Herramienta opcional: **PVSIRE** (Programa Validador SIRE), para validar el TXT/ZIP de reemplazo antes de enviarlo.

### 4.6. Quién está obligado (cronograma vigente a 2026)

Incorporación por oleadas. Referencia consolidada:

- **Julio 2023:** primer grupo (anexo 7 de la RS 112-2021/SUNAT).
- **Octubre 2023:** RER y MYPE que ya estaban en PLE/Portal.
- **Agosto 2024 / enero 2025:** resto de obligados no PRICO.
- **Enero 2026:** PRICOS designados al 31/12/2024 con ingresos netos 2024 **hasta 2 300 UIT**.
- **Octubre 2026:** PRICOS designados al 31/12/2024 con ingresos netos 2024 **mayores a 2 300 UIT** (postergado; antes se habló de junio 2026 — RS 125-2026/SUNAT).

Si el contribuyente está obligado a llevar Registro de Ventas y de Compras, en la práctica ya le toca SIRE o le toca en octubre 2026 si es PRICO grande.

SUNAT ha aplicado **facultad discrecional** (no multa inmediata) en varios tramos para regularizar. Eso no elimina la obligación. Revisar la resolución discrecional vigente al momento de implementar.

### 4.7. Relación con agrupar ventas

En SIRE/RVIE **no se agrupan a mano** las facturas ni las boletas electrónicas que SUNAT ya recibió.

Lo que sí puede faltar en la propuesta:

- Boletas o tickets no electrónicos.
- Operaciones consolidables por día que el SEE no informó documento a documento.
- Documentos físicos o especiales.

Si todo se emite electrónico, el trabajo del mes es **conciliar**, no consolidar.

### 4.8. Conexión con el IGV (Formulario 621)

Al generar RVIE y RCE, SUNAT propone las casillas de ventas y compras del 621. Aceptar un RVIE incompleto o inflado distorsiona la declaración mensual. No aceptar la propuesta a ciegas.

### 4.9. Errores frecuentes

- Aceptar la propuesta sin cruzarla con el sistema interno.
- No cerrar el registro anterior en PLE/Portal: el módulo puede **no dejar generar el RVIE**.
- Olvidar el mes sin movimiento.
- Confundir Resumen Diario de boletas (emisión) con RVIE (libro del mes).
- Complementar mal boletas ≥ S/ 700 sin datos de cliente.
- Dejar ajustes posteriores para “después” y descuadrar el 621.

### 4.10. Rutina práctica recomendada

1. Emitir todo electrónico.
2. El día 2 del mes siguiente, bajar la propuesta del RVIE.
3. Cruzar totales y cantidad de comprobantes.
4. Si calza: aceptar. Si no: complementar o reemplazar.
5. Generar preliminar + registro.
6. Revisar la propuesta del 621 antes de declarar.

---

## 5. ERP que emite desde sistemas del contribuyente

### 5.1. Principio de diseño

La emisión y el libro son dos sistemas de SUNAT:

```
ERP
 ├── Módulo de emisión (SEE)
 │     Factura / Boleta / NC / ND
 │     → XML UBL → firma → envío a SUNAT o OSE
 │     → CDR (aceptado / rechazado)
 │     → Resumen Diario (boletas)
 │     → Comunicación de Baja
 │
 └── Módulo SIRE (RVIE + RCE)
       No emite nada
       Toma lo que SUNAT ya recibió
       → propuesta mensual → conciliar → preliminar → registro
```

- **SEE del contribuyente** (directo a SUNAT o vía OSE/PSE): da validez al comprobante.
- **SIRE / RVIE**: es el libro del mes.

Si el XML no llegó o fue rechazado, no estará bien en el RVIE.

### 5.2. Cómo entra al RVIE lo que emite el ERP

La propuesta se arma con lo que SUNAT recibió por:

- envío del **ejemplar** (factura, NC, ND);
- **Resumen Diario** (boletas);
- **comunicación de baja**.

FAQ oficial relevante:

- Comprobante **activo** o con **baja** → propuesta / incluidos.
- Correlativo **rechazado** → propuesta / no incluidos. No darlo por registrado.

Plazos de referencia:

- Propuesta: desde el día 2 del mes siguiente; se actualiza con lo del día anterior.
- Preliminar: desde el día 8 (día 10 en recibos de servicios públicos de empresas supervisadas).
- Facturas y notas (SEE contribuyente): envío desde el día siguiente de la emisión hasta 3 días calendario posteriores (verificar plazo vigente al implementar).
- Resumen Diario de boletas: el mismo día o hasta el 7.º día calendario siguiente.

### 5.3. Pipeline A — Emisión diaria (SEE)

1. Generar UBL (factura serie F, boleta serie B, NC, ND, y los demás documentos que el producto soporte).
2. Firmar con el certificado digital del contribuyente.
3. Enviar:
   - Facturas y notas: una por una.
   - Boletas: Resumen Diario.
4. Guardar CDR, ticket, hash, XML y PDF.
5. Reintentar rechazos. No reutilizar correlativos rechazados.

Sin este pipeline, el RVIE queda incompleto aunque el ERP “tenga la venta”.

### 5.4. Pipeline B — Cierre mensual (SIRE / RVIE)

Desde el día 2 del mes siguiente:

1. Autenticar API SIRE (OAuth 2.0: `client_id`, `client_secret` + usuario/clave SOL).
2. Descargar la propuesta RVIE.
3. Conciliar contra la tabla de ventas (serie-número, tipo, fecha, base, IGV, total, estado CDR).
4. Decidir aceptar / complementar / reemplazar.
5. Registrar preliminar.
6. Completar la generación del registro (confirmar en manual vigente qué queda 100 % por API y qué exige SOL).
7. Guardar constancia, hash y ticket.
8. Ajustes posteriores si hay error.

### 5.5. Credenciales API

El contribuyente las genera en SOL, URI **“MIGE RCE y RVIE – SIRE”**, alcance Web o Desktop.

Token (referencia de manuales SUNAT):

```
POST https://api-seguridad.sunat.gob.pe/v1/clientessol/{client_id}/oauth2/token/
```

- `grant_type`: password
- `scope`: `https://api-sire.sunat.gob.pe`
- Además: usuario SOL y clave SOL

El ERP debe guardar `client_id` / `client_secret` **por RUC**.

Base de servicios:

```
https://api-sire.sunat.gob.pe
```

Documentación de partida:

- Formas de acceso: https://cpe.sunat.gob.pe/node/158
- Manual API RVIE (verificar versión vigente; en 2026 circulaba v30)
- Anexo 6 de la RS 112-2021/SUNAT (REST, ZIP/JSON, OAuth 2.0, hash)

### 5.6. Servicios que un ERP serio debe cubrir

| Función | Para qué |
|---|---|
| Api Seguridad | Token |
| Consultar periodos | Meses abiertos (`codLibro = 140000` para RVIE) |
| Descargar propuesta | Base del matching |
| Descargar resumen / inconsistencias | Detectar huecos |
| Aceptar propuesta | Camino feliz |
| Upload de reemplazo (TXT/ZIP) | Corrección masiva |
| Registrar preliminar | Avanzar el proceso |
| Consultar estado de ticket | Todo es asíncrono |
| Descargar RVIE / casillas 621 | Auditoría y declaración |
| Ajustes posteriores | Rectificaciones |

Código de libro RVIE habitual: **140000**.

Nombre típico de archivo de importación/reemplazo (estructura tipo LE):

```
LERRRRRRRRRRRAAAAMM0014040002OIM2.txt
```

Validar siempre contra el anexo vigente. Usar PVSIRE en QA antes de subir reemplazos.

### 5.7. Modelo de datos mínimo

No generar un “RVIE interno” que pretenda reemplazar a SUNAT.  
Fuente operativa: la venta del ERP.  
Fuente fiscal del libro: la propuesta SUNAT conciliada.

Por cada CPE guardar:

- tipo, serie, número, fecha, moneda, tipo de cambio
- cliente (tipo y número de documento; obligatorio en boleta ≥ S/ 700)
- base gravada / exonerada / inafecta, IGV, ISC, ICBPER, total
- estado SEE: generado / enviado / aceptado / rechazado / baja
- ticket, CDR, hash XML
- estado SIRE: en propuesta / no incluido / complementado / en preliminar / en RVIE / ajuste
- CAR (código de anotación), cuando exista

Clave de matching:

```
RUC emisor + periodo + tipo + serie + número
```

### 5.8. Agrupar ventas desde el ERP

Si se emite electrónico desde sistemas del contribuyente:

- **Facturas**: nunca agrupar. Un XML = una línea que SUNAT propondrá.
- **Boletas**: el agrupado fiscal de envío es el **Resumen Diario**, no un asiento inventado en el RVIE.
- La consolidación diaria en el libro solo entra si hay tickets/boletas que SUNAT **no** trae solas (típicamente no electrónicos) y se cumplen las reglas del campo 9 / S/ 700.

Caso feliz de un ERP moderno: **cero consolidación manual**.  
Emitir bien → CDR aceptado → aceptar propuesta RVIE.

### 5.9. Casos de borde que el producto debe contemplar

- CPE emitido pero rechazado: no está en incluidos; corregir y reenviar.
- Boleta emitida sin Resumen Diario: existe en el ERP y no en SUNAT.
- Mes mixto: unas series por SEE-SOL y otras por el ERP. La propuesta junta ambas.
- Mes sin operaciones: generar registro vacío.
- Migración PLE → SIRE: cerrar el periodo anterior en PLE/Portal.
- Multi-RUC: certificado y credenciales API por empresa.
- Tipo de cambio, exportaciones y notas que referencian un CPE de otro periodo.

### 5.10. Diseño de producto recomendado

Tres módulos:

1. **Facturación electrónica** (SEE + OSE/SUNAT + resúmenes + bajas).
2. **Conciliador SIRE** (job días 2–7: baja propuesta, compara, muestra diferencias).
3. **Cierre tributario** (aceptar / reemplazar / preliminar / constancia / casillas 621).

Pantalla de mayor valor:

> Emitidos en ERP vs Propuesta SUNAT vs Rechazados vs Sin CDR

Si esa grilla queda en cero diferencias, el usuario acepta la propuesta.

---

## 6. Checklist de implementación para el equipo de desarrollo

### Emisión (SEE)

- [ ] UBL de factura, boleta, NC y ND según catálogos vigentes
- [ ] Firma digital y control de vigencia del certificado
- [ ] Envío a SUNAT directo y/o OSE
- [ ] Persistencia de XML, CDR, ticket y PDF
- [ ] Resumen Diario de boletas (alta, baja/anulación según reglas)
- [ ] Comunicación de baja de facturas
- [ ] Reintentos y cola de rechazos
- [ ] Correlativos por serie y establecimiento
- [ ] Identificación de cliente en boletas ≥ S/ 700
- [ ] Consulta pública de CPE (obligación del emisor electrónico)

### SIRE / RVIE

- [ ] Alta de credenciales API por RUC
- [ ] Job de descarga de propuesta desde el día 2
- [ ] Motor de conciliación por tipo-serie-número
- [ ] UI de diferencias (faltan en SUNAT / sobran en SUNAT / montos distintos)
- [ ] Aceptar propuesta
- [ ] Complementar / reemplazar con TXT validado (PVSIRE)
- [ ] Registro de preliminar y consulta de ticket
- [ ] Guardado de constancia y hash
- [ ] Ajustes posteriores
- [ ] Mes sin movimiento
- [ ] Cierre del libro previo (PLE/Portal) en onboarding
- [ ] Lectura de casillas propuestas del 621 (informativa)

### Cumplimiento

- [ ] Calendario de atraso / cronograma SUNAT
- [ ] Alertas de CPE sin CDR al cierre del día
- [ ] Alertas de boletas sin Resumen Diario antes del día 7
- [ ] No permitir “aceptar propuesta” si hay diferencias materiales
- [ ] Trazabilidad de quién aceptó o reemplazó el RVIE

---

## 7. Orden sugerido de llamadas API (ciclo RVIE)

Referencia de diseño. Confirmar paths exactos en el manual vigente.

1. Obtener token (Api Seguridad).
2. Consultar periodos habilitados (`codLibro = 140000`).
3. Descargar propuesta del periodo.
4. Descargar resumen e inconsistencias.
5. Comparar en el ERP.
6. Si hay match: aceptar propuesta.
7. Si no hay match: upload de complemento o reemplazo → consultar ticket hasta “atendido”.
8. Registrar preliminar → consultar ticket.
9. Generar registro (API y/o SOL, según capacidad vigente).
10. Descargar RVIE generado, constancia y, si aplica, casillas 621.
11. Si aparece un error posterior: flujo de ajustes posteriores.

Todo envío masivo es **asíncrono**: no avanzar de etapa sin ticket en estado atendido.

---

## 8. Esquema de tablas sugerido (mínimo)

### `cpe_emitidos`

- id, ruc_emisor, periodo (`YYYYMM`)
- tipo_cdp, serie, numero, fecha_emision
- moneda, tipo_cambio
- tipo_doc_cliente, num_doc_cliente, razon_social
- base_gravada, exonerada, inafecta, igv, isc, icbper, total
- estado_see (generado, enviado, aceptado, rechazado, baja)
- ticket_see, cdr_codigo, cdr_descripcion, xml_hash
- resumen_diario_id (si es boleta)
- created_at, updated_at

### `sire_propuesta_rvie`

- id, ruc_emisor, periodo
- tipo_cdp, serie, numero
- montos SUNAT
- origen (propuesta, complementado, reemplazo)
- flag_incluido
- car
- raw_line

### `sire_diff`

- id, ruc_emisor, periodo
- cpe_id / propuesta_id
- tipo_diff (falta_sunat, falta_erp, monto, estado_rechazado, sin_cdr)
- detalle

### `sire_proceso`

- id, ruc_emisor, periodo, etapa
- num_ticket, estado_ticket
- hash, constancia_url
- usuario, timestamps

Índice único recomendado en emitidos y propuesta:

```
(ruc_emisor, periodo, tipo_cdp, serie, numero)
```

---

## 9. Enlaces oficiales de partida

- SIRE: https://sire.sunat.gob.pe/
- Orientación SIRE: https://orientacion.sunat.gob.pe/05-registros-electronicos-sire
- RVIE (gob.pe): https://www.gob.pe/26232-registro-de-ventas-e-ingresos-electronico-rvie
- Pasos de generación RVIE: https://www.gob.pe/26263-pasos-para-la-generacion-del-rvie-y-anotacion-de-operaciones
- Accesos y manuales API: https://cpe.sunat.gob.pe/node/158
- FAQ SIRE/RVIE: https://cpe.sunat.gob.pe/node/131
- SEE sistemas del contribuyente: https://www.gob.pe/25681-sistema-de-emision-electronica-del-contribuyente
- SEE-OSE: https://www.gob.pe/26398-sistema-de-emision-electronica-operador-de-servicios-electronicos-see-ose
- Resumen Diario de boletas: https://orientacion.sunat.gob.pe/04-resumen-diario-boleta-de-venta-electronica

Antes de cerrar desarrollo, descargar de cpe.sunat.gob.pe el **manual API RVIE vigente** y los anexos de estructura de archivos planos de la RS 112-2021/SUNAT y modificatorias.

---

## 10. Resumen ejecutivo

1. En el libro de ventas se puede agrupar **solo por día** y **solo documentos que no dan crédito fiscal**, con detalle conservado en sistema.
2. Las facturas no se agrupan.
3. S/ 700 es el umbral crítico de detalle e identificación del cliente en boletas.
4. El Resumen Diario no es el libro; es el envío de las boletas electrónicas.
5. SIRE propone RVIE y RCE con lo que SUNAT ya tiene.
6. Un ERP que emite desde sistemas del contribuyente debe integrar **SEE + SIRE**, no solo facturación.
7. El valor del producto no está en “dibujar el libro”, sino en **emitir bien y conciliar la propuesta**.
8. Si la conciliación da cero diferencias, se acepta la propuesta y se cierra el mes.

---

*Documento generado a partir de la conversación sobre Libro de Ventas SUNAT, agrupación de ventas, RVIE/SIRE e integración ERP. Contrastar siempre con la norma y los manuales técnicos vigentes al momento de implementar.*
