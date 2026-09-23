# Libro Diario de Formato Simplificado (SUNAT)

Guía práctica para llevar el libro y para implementarlo en un ERP.

- Tema: Libro Diario de Formato Simplificado (LDFS)
- Formatos electrónicos PLE: **5.2** (movimientos) y **5.4** (plan contable)
- Relación con SIRE/RVIE: el LDFS **no** se genera en SIRE. Sigue en **PLE**.
- Fecha: 15 de septiembre de 2026

Contrastar siempre con el Anexo 2 vigente de la RS 286-2009/SUNAT y modificatorias, y con el PLE instalado.

---

## 1. Qué es

Es la versión simplificada del Libro Diario para empresas de tercera categoría con **ingresos anuales hasta 300 UIT**.

Sustituye, en ese tramo, al Libro Diario completo (5.1) y, en la práctica, evita tener que llevar también Mayor y Caja y Bancos como libros separados de esa escala.

No reemplaza:

- Registro de Ventas (hoy **RVIE / SIRE**)
- Registro de Compras (hoy **RCE / SIRE**)

Esquema típico hasta 300 UIT:

```
SIRE  →  RVIE + RCE          (compras y ventas / IGV)
PLE   →  Libro Diario 5.2    (contabilidad)
         + Plan contable 5.4 (enero o cuando cambia el plan)
```

---

## 2. Quién está obligado

Base: art. 65 de la Ley del Impuesto a la Renta y art. 11 del D.L. 1269 (RMT).

| Régimen | Ingresos anuales | Libros |
|---|---|---|
| NRUS | — | No lleva libros. Solo conserva comprobantes. |
| RER | Cualquiera | Solo Registro de Compras y de Ventas (SIRE). **No** LDFS. |
| MYPE o General | Hasta **300 UIT** | Ventas + Compras + **Libro Diario de Formato Simplificado** |
| MYPE o General | Más de 300 hasta 500 UIT | Diario completo + Mayor + Compras + Ventas |
| MYPE o General | Más de 500 hasta 1 700 UIT | Lo anterior + Inventarios y Balances |
| General | Más de 1 700 UIT | Contabilidad completa |

Si el contribuyente lleva Diario completo (5.1) en vez del simplificado, SUNAT ha interpretado que **no hay infracción** por omitir el simplificado (Informe 089-2013-SUNAT; el umbral citado entonces era otro). Aun así, si la obligación legal es el 5.2, lo correcto es generar el 5.2.

UIT de referencia 2025/2026: confirmar el valor vigente. A modo ilustrativo, 300 UIT en 2025 se calculó sobre UIT de S/ 5 350 → S/ 1 605 000. Usar siempre la UIT del ejercicio que corresponda.

---

## 3. Cómo se “hace” el libro (lógica contable)

El LDFS es un diario de **partida doble**. Cada asiento equilibra Debe = Haber.

### 3.1. Asientos que deben existir

1. **Apertura del ejercicio** (enero o inicio de actividades): saldos del Balance.
2. **Operaciones del mes**.
3. **Ajustes de meses anteriores**, si hay.
4. **Ajustes del mes**.
5. **Cierre del ejercicio** (diciembre): cierra ingresos y gastos contra resultado.

### 3.2. Información mínima (formato 5.2 visible / RS 234-2006 y ss.)

- Número correlativo o código único de la operación (CUO)
- Fecha o periodo
- Glosa
- Cuentas según el Plan Contable General Empresarial (PCGE)
- Movimiento Debe / Haber
- Totales
- Si el asiento **no** es consolidado: vínculo con ventas/compras (CAR o dato estructurado)

En la vista “tabular” clásica las columnas se agrupan así:

```
CUO | Fecha | Glosa | ACTIVOS | PASIVOS | PATRIMONIO | GASTOS | INGRESOS | Totales
                         (Debe/Haber por cuenta o por elemento)
```

En el PLE electrónico **no** se mandan esas columnas anchas. Se manda **una fila por cada línea del asiento** (una cuenta, un importe al Debe **o** al Haber).

### 3.3. Plan de cuentas

Usar PCGE vigente. Nivel mínimo habitual: **3 dígitos** (cuenta); lo recomendable en un ERP es 4 o más (subcuenta), coherente en todo el mes.

Cuentas típicas de una MYPE comercial:

| Código | Cuenta | Uso |
|---|---|---|
| 101 | Caja | Efectivo |
| 104 | Cuentas corrientes | Banco |
| 121 | Clientes | Por cobrar |
| 201 | Mercaderías | Inventario |
| 4011 | IGV por pagar | Débito fiscal |
| 4017 | IGV crédito fiscal | Compras |
| 421 | Proveedores | Por pagar |
| 501 | Capital | Patrimonio |
| 601 / 201 | Compras / mercaderías | Según método |
| 621 | Gastos de personal | Planilla |
| 63x | Servicios prestados por terceros | Alquiler, luz, etc. |
| 701 | Ventas | Ingresos |
| 791 | Cargas imputables a costos | Destino de compras |

El ERP debe tener un **plan de cuentas parametrizable**. No hardcodear solo estas.

---

## 4. Ejemplo de asientos (mes tipo)

Empresa comercial. IGV 18 %. Montos ilustrativos.

### Apertura (enero) — correlativo A1

| Cuenta | Debe | Haber |
|---|---:|---:|
| 101 Caja | 2 000.00 | |
| 104 Banco | 8 000.00 | |
| 201 Mercaderías | 15 000.00 | |
| 421 Proveedores | | 5 000.00 |
| 501 Capital | | 20 000.00 |
| **Totales** | **25 000.00** | **25 000.00** |

### Venta al contado S/ 1 180 (incluye IGV) — M1

Factura F001-00000015.

| Cuenta | Debe | Haber |
|---|---:|---:|
| 101 Caja | 1 180.00 | |
| 701 Ventas | | 1 000.00 |
| 4011 IGV | | 180.00 |

### Compra al crédito S/ 590 (incluye IGV) — M2

Factura del proveedor F002-00001000.

| Cuenta | Debe | Haber |
|---|---:|---:|
| 601 Compras | 500.00 | |
| 4017 IGV crédito | 90.00 | |
| 421 Proveedores | | 590.00 |

### Costo de ventas (si se usa inventario permanente o destinos) — M3

| Cuenta | Debe | Haber |
|---|---:|---:|
| 691 Costo de ventas | 400.00 | |
| 201 Mercaderías | | 400.00 |

### Pago a proveedor — M4

| Cuenta | Debe | Haber |
|---|---:|---:|
| 421 Proveedores | 590.00 | |
| 104 Banco | | 590.00 |

Regla: **un asiento = un CUO**. Varias filas (una por cuenta) comparten el mismo CUO.

---

## 5. Consolidación de operaciones

En el LDFS electrónico se puede consolidar **solo por día**, y solo si el sistema guarda el detalle para verificar cada documento.

En la práctica, para un ERP que ya emite electrónico:

- Lo más limpio es **un asiento por comprobante** (o por lote diario de caja, si hay decenas de boletas chicas).
- Si consolidas ventas del día: un asiento “Ventas del 15/09” con el total, y el detalle vive en el RVIE.
- Si el asiento es consolidado, **no** se llena el campo de dato estructurado / CAR.
- Si el asiento es individual (una factura), sí se vincula con ventas o compras.

Eso encaja con lo ya visto del Registro de Ventas: consolidar ventas en el libro de ventas es una cosa; consolidar el **asiento contable** del diario es otra. Ambas, si se hacen, son **diarias**.

---

## 6. Formato electrónico PLE (lo que el ERP debe generar)

El LDFS **no va por SIRE**. Se genera TXT, se valida en el **PLE** y se envía a SUNAT.

### 6.1. Archivos del mes

| Formato | Código | Archivo |
|---|---|---|
| 5.2 Libro Diario Simplificado | 050200 | `LERRRRRRRRRRRAAAAMM0005020000OIM1.TXT` |
| 5.4 Detalle del plan contable | 050400 | `LERRRRRRRRRRRAAAAMM0005040000OIM1.TXT` |

Nombre (posiciones):

```
LE + RUC(11) + AAAA + MM + 00 + 050200 + 00 + O + I + M + 1
```

- `O` = indicador de operaciones: `1` empresa operativa; `2` cierre del libro (ya no obligado); `0` baja de RUC
- `I` = contenido: `1` con información; `0` sin información
- `M` = moneda: `1` soles; `2` dólares

El 5.4 es **obligatorio en enero** y la primera vez que se genera el libro. El resto de meses puede ir vacío, salvo que cambie el plan de cuentas.

### 6.2. Campos del 5.2 (fila = una línea del asiento)

Separador: `|` (pipe). Una línea del TXT por cada cuenta del asiento.

Estructura de referencia (validar contra el PLE / Anexo 2 vigente):

| # | Campo | Notas |
|---|---|---|
| 1 | Periodo | `AAAAMM00` (ej. `20260900`) |
| 2 | CUO | Clave única del asiento en el ERP. No se repite. |
| 3 | Nº correlativo del asiento | Empieza con **A** (apertura), **M** (movimiento/ajuste) o **C** (cierre). Ej. `M00001` |
| 4 | Código de cuenta | Al máximo nivel usado, alineado al 5.4 |
| 5 | Código unidad de operación | Opcional (sucursal, local). Varios códigos con `&` |
| 6 | Glosa / descripción | Texto de la operación |
| 7 | Debe | ≥ 0. Excluyente con Haber |
| 8 | Haber | ≥ 0. Excluyente con Debe |
| 9–… | Fecha, moneda, estado, etc. | Según anexo vigente |
| … | Estado de la operación | `1` = del periodo; `8` = omitida anotada después; `9` = corrige una anterior |
| 20 | Dato estructurado / CAR | Obligatorio si el asiento **no** es consolidado y corresponde a compra o venta. Vacío si es consolidado. |

Estados para corregir:

- **8**: se omitió anotar; se informa después con el CUO de esa omisión.
- **9**: se corrige un asiento ya enviado; el CUO apunta al original.

### 6.3. Campo 20 (vínculo con ventas y compras)

Si el asiento detalla un comprobante (no está consolidado):

- Código del libro + campos de periodo/CUO del RVIE o RCE, separados por `&`, **o**
- **CAR** de 27 caracteres: RUC/doc emisor (11) + tipo (2) + serie (4) + número (10)

Códigos de libro de referencia:

- Ventas: `140100`
- Compras: `080100` / `080200`

Quien ya está en **módulo RVIE/RCE (SIRE)** usa las reglas de CAR del anexo actualizado (RS 112-2021 y ss.). Quien aún no, usa el dato estructurado clásico.

Este campo **debe coincidir** con lo declarado en RVIE/RCE. Si el ERP aceptó una propuesta SIRE, el diario tiene que usar los mismos comprobantes.

### 6.4. Cómo se envía

1. El ERP arma los TXT 5.2 y, si toca, 5.4.
2. Se validan con el **Programa de Libros Electrónicos (PLE)** de SUNAT.
3. El PLE genera el archivo de envío.
4. Se envía a SUNAT y se guarda la constancia.

No existe (al cierre de esta guía) un equivalente SIRE-API para el Diario Simplificado. Es PLE.

---

## 7. Plazo de atraso

Libro Diario de Formato Simplificado: **3 meses**, contados desde el primer día hábil del mes siguiente al de las operaciones (anexo de plazos de la RS 234-2006 y actualizaciones, p. ej. RS 257-2025/SUNAT).

Ejemplo: operaciones de septiembre 2026 → anotar a más tardar hacia diciembre 2026. No esperar al tope.

---

## 8. Cómo implementarlo en el ERP

### 8.1. Módulos

```
ERP
 ├── Facturación electrónica (SEE)     → XML + CDR
 ├── SIRE (RVIE + RCE)                 → propuesta mensual IGV
 └── Contabilidad
      ├── Plan de cuentas (PCGE)
      ├── Asientos (diario)
      └── Generador PLE 5.2 / 5.4
```

El asiento no se “inventa” en un Excel al cierre. Debe nacer de:

- la factura/boleta emitida,
- la compra registrada,
- tesorería (cobros/pagos),
- planilla,
- ajustes manuales.

### 8.2. Motor de asientos sugerido

Plantillas (document types):

| Origen | Asiento típico |
|---|---|
| Factura / boleta de venta | Dr Caja/Banco/Cliente — Cr Ventas — Cr IGV |
| Nota de crédito venta | Inverso |
| Factura de compra | Dr Compra/Gasto — Dr IGV crédito — Cr Proveedor |
| Pago / cobro | Dr/Cr Caja-Banco vs CxC/CxP |
| Destino de compras / costo | Dr 69 / 9x — Cr 20 / 61 |
| Planilla | Dr 62 — Cr 40 / 41 |
| Apertura | Saldos de balance |
| Cierre | Dr 70/75… Cr 90 / Resultado; cierre de gastos |

Cada plantilla genera un `account_move` con N líneas. El CUO = id interno estable (sin `/` ni espacios, o un correlativo propio).

### 8.3. Tablas mínimas

`account_account`  
plan PCGE, código SUNAT, naturaleza, elemento (activo, pasivo, patrimonio, gasto, ingreso)

`account_move`  
periodo, CUO, correlativo (A/M/C), fecha, glosa, estado (1/8/9), consolidado (sí/no), origen (venta, compra, tesorería, ajuste)

`account_move_line`  
cuenta, debe, haber, analítica / local

`ple_export_5_2`  
snapshot del TXT generado, hash, ticket PLE, constancia

### 8.4. Validaciones antes de exportar

- Debe = Haber por CUO
- Periodo `AAAAMM00`
- Correlativo empieza con A, M o C
- Una sola de Debe/Haber > 0 por línea
- Plan 5.4 cubre todas las cuentas usadas en el 5.2
- Si no es consolidado y es compra/venta: campo 20 / CAR presente y existente en RVIE o RCE
- No caracteres `|` `\` `/` en textos (regla PLE)
- Mes sin movimientos: archivo con indicador `I=0` o según regla del PLE para “sin información”

### 8.5. Relación con el RVIE que ya están construyendo

Flujo mensual coherente:

1. Emitir CPE (SEE).
2. Cerrar SIRE: aceptar/complementar RVIE y RCE.
3. Contabilizar (o ya está contabilizado en línea).
4. Generar PLE 5.2 con los mismos comprobantes.
5. Declarar 621.

No aceptar un RVIE de 100 facturas y un diario de 80. La conciliación SIRE debe alimentar o bloquear el cierre del diario.

---

## 9. Manual vs electrónico

- **Manual / hojas sueltas:** formato 5.2 en columnas (activos, pasivos, patrimonio, gastos, ingresos). Hay que legalizar según reglas vigentes si aún aplica al caso.
- **Electrónico (recomendado y, en muchos casos, obligatorio):** TXT 5.2 + 5.4 por PLE.

Si el contribuyente ya está obligado a libros electrónicos por ingresos (75 UIT en el periodo de evaluación de ventas/compras) o por ser PRICO, el Diario Simplificado también corre por PLE.

---

## 10. Errores frecuentes

- Llevar solo compras y ventas y olvidar el Diario Simplificado (obligatorio hasta 300 UIT en MYPE/General).
- Generar el 5.2 en SIRE (no existe ahí).
- Consolidar el mes entero en un solo asiento de ventas.
- Usar cuentas de 2 dígitos o un plan inventado que no se declara en el 5.4.
- No generar el 5.4 en enero.
- Asientos que no cuadran.
- Correlativo sin prefijo A/M/C.
- Campo 20 lleno en asientos consolidados, o vacío en asientos individuales de compras/ventas.
- Confundir LDFS (contabilidad) con RVIE (libro de ventas).

---

## 11. Checklist de cierre de mes

- [ ] Todos los CPE del mes tienen CDR / Resumen Diario
- [ ] RVIE y RCE generados en SIRE
- [ ] Todos los CPE contabilizados
- [ ] Cobros, pagos, planilla y ajustes ingresados
- [ ] Apertura solo en enero (o inicio)
- [ ] Cierre solo en diciembre
- [ ] Balance de comprobación: Debe = Haber
- [ ] TXT 5.2 validado en PLE
- [ ] TXT 5.4 en enero o si cambió el plan
- [ ] Constancia PLE archivada
- [ ] Plazo de 3 meses no vencido

---

## 12. Enlaces de partida

- Libros según régimen: https://www.gob.pe/1211-cuales-son-los-libros-contables-para-tu-negocio
- Orientación registros y libros: https://orientacion.sunat.gob.pe/registros-y-libros-1
- Estructuras PLE: https://www.gob.pe/institucion/sunat/informes-publicaciones/356712-estructura-de-los-libros-y-registros-electronicos-en-el-ple
- Art. 65 LIR (tramos 300 / 500 / 1 700 UIT)
- RS 234-2006/SUNAT (formatos e información mínima)
- RS 286-2009/SUNAT y anexos (PLE)
- RS 112-2021/SUNAT (vínculo Diario ↔ RVIE/CAR)

Descargar del portal SUNAT el Excel/PDF de estructuras PLE vigente y validar cada campo con la versión del PLE que use el cliente.

---

## 13. Relación con los documentos anteriores

| Documento | Cubre |
|---|---|
| `SUNAT-Libro-Ventas-RVIE-SIRE-ERP.md` | Agrupar ventas, SIRE/RVIE, API, emisión desde sistemas del contribuyente |
| Este archivo | Libro Diario Simplificado 5.2/5.4, asientos, PLE, enlace contable con RVIE |

En el ERP: SIRE cierra el IGV; el 5.2 cierra la contabilidad. Son complementarios, no intercambiables.
