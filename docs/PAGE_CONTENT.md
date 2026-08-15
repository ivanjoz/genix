# Contenido de la sección "Funcionalidades" — /welcome

Texto fuente de la página pública. Reemplaza las 6 tarjetas actuales por **8 secciones**, cada una
con su propia lista de funcionalidades.

**Leyenda de estado** (cada ítem lleva su marca; en la página el pendiente se pinta con un icono de
reloj + la etiqueta "Próximamente", nunca sólo con color):

- ✅ **Disponible** — implementado y usable hoy.
- ⏳ **Próximamente** — no está o está incompleto. Cuando ya existe una parte, el propio texto lo dice.

Notas de implementación:

- **Ya implementado** en `frontend/routes/welcome/features.ts` (cadenas `English|Spanish` para
  `T.svelte`) y `frontend/routes/welcome/+page.svelte`.
- La página muestra cada ítem como **etiqueta corta** de una línea, porque son ocho listas seguidas y
  el visitante las escanea. Este documento guarda la versión larga: el porqué, el matiz de lo que ya
  existe a medias y el detalle que no cabe en la página. Al agregar o quitar un ítem aquí hay que
  reflejarlo allá.
- El estado de cada ítem debe coincidir con el roadmap del `README.md`. Si cambia uno, cambia el otro.
- El bloque `roadmap` de la página se mantiene: aquí va el **qué hace el producto**, allá el **cuándo**.

---

## Encabezado de la sección

**Eyebrow:** Funcionalidades

**Título:** Todo lo que su negocio necesita, en un solo sistema.

**Bajada:** Genix cubre el día a día de una micro o pequeña empresa —vender, comprar, producir,
controlar el stock y la caja— y crece con usted hacia la planificación, la contabilidad y la venta en
línea. Estamos construyendo el producto a la vista: lo que ya funciona está marcado como disponible y
lo que viene, como próximamente.

---

## 1. Productos, servicios e insumos

**Bajada:** Un solo catálogo alimenta el mostrador, el almacén y la tienda en línea. Y si usted no
sólo revende sino que **fabrica**, aquí se mapea su proceso productivo: qué insumos consume cada
producto y cuánto le cuesta realmente producir una unidad.

- ✅ Catálogo con categorías, marcas, atributos y varias imágenes por producto.
- ✅ Varias presentaciones por producto, cada una con su propio precio y SKU.
- ✅ SKU por producto y por presentación, con control por lote y por número de serie individual.
- ✅ ¿Compra por caja y vende por unidad? Defina la sub-unidad y el sistema calcula precio, descuento
  y stock en la unidad menor.
- ✅ Catálogo de insumos y materiales, con stock mínimo y sus proveedores: precio, capacidad de
  abastecimiento y tiempo de entrega.
- ✅ Compra de insumos en la misma orden de compra que la mercadería.
- ✅ Carga y descarga masiva de productos desde Excel.
- ✅ Búsqueda instantánea por nombre y marca, con un motor pensado para el español.
- ⏳ Receta o ficha técnica del producto: qué insumos consume cada unidad y en qué cantidad. Un
  zapato lleva su cuero, su tela, su hilo y su pintura, cada uno con su merma esperada.
- ⏳ Costo de producción por unidad calculado solo: insumos al precio real de compra, mano de obra y
  costos indirectos prorrateados.
- ⏳ Orden de producción: descuenta los insumos del almacén y da de alta el producto terminado con su
  lote y su costo del día.
- ⏳ Etapas del proceso (corte, armado, pintado, acabado) con responsable, tiempo y costo por etapa,
  para ver dónde se va el dinero y dónde está el cuello de botella.
- ⏳ Margen real por producto y por presentación: precio de venta contra costo de producción vigente.
- ⏳ Aviso cuando el alza del precio de un insumo deja un producto por debajo de su margen objetivo.
- ⏳ Explosión de materiales: cuánto cuero, tela e hilo comprar este mes según el plan de ventas.
- ⏳ Trazabilidad completa: de qué lote de insumo salió cada lote o serie de producto terminado.
- ⏳ Merma real contra merma esperada, por insumo y por orden de producción.
- ⏳ Versiones de la receta con su historial de costo, para comparar cuánto subió producir el mismo
  modelo respecto al año pasado.
- ⏳ Servicios: venda reparación, mantenimiento o instalación desde el mismo punto de venta, sin
  stock pero con su costo y su margen.
- ⏳ Producción tercerizada: el trabajo del taller externo entra como un costo más de la orden.
- ⏳ Combos y productos compuestos que descuentan sus componentes al venderse.

## 2. Clientes y CRM

**Bajada:** El cliente no termina en la boleta. Genix tendrá un CRM completo: con quién habló, qué le
ofreció, qué le compró y cuánto le debe, todo sobre la misma base del ERP.

- ✅ Clientes y proveedores: persona natural o jurídica, RUC o DNI, correo y ubicación geográfica.
- ✅ Búsqueda de clientes por nombre o número de documento.
- ✅ Cree el cliente en el mismo punto de venta, sin salir del cobro.
- ✅ Reporte de ventas filtrado por cliente, producto, fecha y estado.
- ⏳ Ficha del cliente en una sola vista: historial de compras, ticket promedio, última compra,
  productos preferidos y saldo pendiente.
- ⏳ Contactos y personas dentro de cada empresa cliente, con su cargo y sus datos de contacto.
- ⏳ Bitácora de interacciones: llamadas, visitas, correos y mensajes, con la próxima acción anotada.
- ⏳ Agenda de reuniones y recordatorios de seguimiento por responsable.
- ⏳ Embudo de oportunidades con etapas, monto estimado y probabilidad de cierre.
- ⏳ La oportunidad se convierte en cotización y la cotización en pedido, sin volver a digitar nada.
- ⏳ Cartera asignada por vendedor, con su avance contra la meta.
- ⏳ Segmentación de clientes por zona, giro y frecuencia de compra, con lista de precios por
  segmento.
- ⏳ Cuentas por cobrar y cobranza: saldo por cliente, vencimientos, antigüedad de la deuda y
  recordatorio automático. La venta ya registra el monto adeudado y su fecha de vencimiento; falta el
  reporte y el aviso.
- ⏳ Línea de crédito por cliente, con aviso o bloqueo al superarla.
- ⏳ Aviso de cliente inactivo: quién dejó de comprar y desde cuándo.
- ⏳ Campañas y envíos masivos por correo o WhatsApp a un segmento.
- ⏳ Postventa: reclamos, garantías y devoluciones con su seguimiento.
- ⏳ Fidelización: puntos, descuentos por volumen y cupones por cliente.
- ⏳ Portal del cliente para consultar sus pedidos, comprobantes y saldo.

## 3. Punto de venta y ventas

**Bajada:** Cobre en el mostrador en segundos y quede con la venta, el stock y la caja ya cuadrados.
Después revise qué se vendió, a quién y cuánto queda por cobrar.

- ✅ Punto de venta con búsqueda de productos, carrito, selección de caja y almacén, y asignación o
  creación del cliente en el mismo momento.
- ✅ Cobro y entrega se registran por separado: pendiente de pago, pendiente de entrega o finalizado.
- ✅ Venta al crédito: la orden guarda el monto adeudado y la fecha de vencimiento.
- ✅ El stock se descuenta por almacén, lote y serie al confirmar la venta.
- ✅ Reporte de ventas con filtros por rango de fechas, cliente, producto y estado, más búsqueda libre
  sobre el resultado.
- ✅ Gráficos de ventas por producto (en monto o cantidad) y resumen diario.
- ✅ Proyección de ventas: cantidad base por producto y curva de estacionalidad de 52 semanas.
- ✅ Costos de envío configurados por departamento, provincia y distrito.
- ⏳ Resumen semanal y mensual, con comparativo contra el periodo anterior.
- ⏳ Facturación electrónica SUNAT: boleta, factura y notas de crédito y débito, con envío y CDR. El
  formato UBL 2.1 ya está construido; falta la firma digital, el envío y la consulta.
- ⏳ Libro de ventas y de compras exportable para su contador.
- ⏳ Cotizaciones y su conversión a pedido en un clic.
- ⏳ Devoluciones y anulaciones con reposición automática de stock.
- ⏳ Promociones: descuentos por volumen, combos, cupones y precios por temporada.
- ⏳ Impresión de ticket en impresora térmica y envío del comprobante por WhatsApp o correo.
- ⏳ Lectura de código de barras y conexión con balanza en el mostrador.
- ⏳ Metas y comisiones por vendedor.
- ⏳ Venta sin internet en el mostrador, con sincronización al reconectarse.

## 4. Logística y compras

**Bajada:** Sepa qué tiene, dónde lo tiene y cuándo hay que reponerlo, sin cuadernos paralelos.

- ✅ Sedes y almacenes, con un editor visual del layout de cada almacén.
- ✅ Stock por almacén con control de lote y número de serie.
- ✅ Entradas y salidas manuales de stock, con su reporte de movimientos.
- ✅ Órdenes de compra: creación, seguimiento y recepción de la mercadería contra la orden.
- ✅ Gestión de compras y de proveedores.
- ✅ Suministros y materiales, separados del producto terminado.
- ✅ Planificación de reposición apoyada en la proyección de ventas.
- ⏳ Sugerencia automática de compra por punto de reorden, stock de seguridad y tiempo de entrega del
  proveedor.
- ⏳ Kardex valorizado y costo promedio ponderado por producto.
- ⏳ Transferencias entre almacenes con confirmación de recepción.
- ⏳ Toma de inventario físico y conteo cíclico, con ajuste por diferencias.
- ⏳ Alertas de stock mínimo y de lotes próximos a vencer.
- ⏳ Evaluación de proveedores: precios históricos, cumplimiento y tiempos de entrega.
- ⏳ Guía de remisión electrónica.
- ⏳ Stock de insumos consumido por las órdenes de producción, junto al de la mercadería.

## 5. Caja y finanzas

**Bajada:** El dinero que entra y sale queda registrado donde ocurre, y desde ahí sale la foto de la
salud del negocio.

- ✅ Cajas y bancos con ingresos, egresos y transferencias entre cuentas.
- ✅ Cuadre y conciliación de caja al cierre.
- ✅ Reporte de movimientos de caja y banco.
- ✅ Gastos puntuales, gastos fijos programados y pagos parciales.
- ✅ El cobro de una venta genera su movimiento de caja, sin doble digitación.
- ⏳ Flujo de caja proyectado: cruza la proyección de ventas, las cuentas por cobrar, las cuentas por
  pagar y los gastos programados para anticipar la falta de liquidez.
- ⏳ Cuentas por pagar: saldo por proveedor, vencimientos y programación de pagos.
- ⏳ Estado de resultados mensual automático: ingresos, costo de ventas (con el costo de producción
  real), gastos y utilidad.
- ⏳ Balance y estados financieros.
- ⏳ Centros de costo y clasificación de gastos por categoría.
- ⏳ Conciliación bancaria importando el extracto del banco.
- ⏳ Presupuesto anual y comparativo de presupuesto contra lo real.
- ⏳ Resumen mensual de impuestos para el contador.
- ⏳ Activos fijos y depreciación.
- ⏳ Multimoneda con tipo de cambio diario.

## 6. Comercio electrónico y página web

**Bajada:** Su tienda en línea se arma con IA sobre el mismo catálogo del ERP: lo que actualiza en el
sistema es lo que ve su cliente.

- ✅ Constructor visual por secciones, con más de 20 plantillas, vista previa móvil y componentes
  propios: grilla de productos, sliders, íconos y efectos de imagen.
- ✅ Creación y edición de páginas con IA, que respeta el contenido que usted ya escribió.
- ✅ Tienda con catálogo, buscador, categorías y fichas de producto sincronizadas con el ERP.
- ✅ Carrito e interfaz de pago con Culqi.
- ✅ Publicación con dominio propio.
- ✅ Galería de imágenes con optimización automática.
- ⏳ El pedido de la tienda se registra en el ERP: hoy el checkout confirma la compra, pero todavía no
  crea la orden.
- ⏳ Cuenta del cliente en la tienda: registro, acceso, historial de pedidos y seguimiento del envío.
- ⏳ Cálculo del costo de envío en el checkout según la dirección. La configuración por distrito ya
  existe; falta aplicarla al carrito.
- ⏳ Más medios de pago: Yape, Plin, transferencia y pago contra entrega.
- ⏳ Cupones y campañas de descuento en la tienda.
- ⏳ Venta por WhatsApp: catálogo compartible y pedido que entra al mismo flujo de ventas.
- ⏳ Correos automáticos de la tienda: confirmación, despacho y carrito abandonado.
- ⏳ SEO por producto: metadatos, sitemap y datos estructurados.
- ⏳ Reseñas y valoraciones de productos.
- ⏳ Publicación del catálogo en redes y marketplaces.

## 7. Asistente de IA integrado

**Bajada:** Un asistente que no sólo responde: abre la página, filtra, llena el formulario y le deja
el trabajo hecho.

- ✅ Chat dentro del ERP que navega y opera las páginas por usted: busca, filtra y completa
  formularios.
- ✅ Construcción y edición de la página web con IA, con control de calidad del diseño y del
  contenido.
- ✅ Generación de imágenes e íconos vectoriales para su tienda.
- ✅ API para que herramientas externas de IA operen el ERP.
- ✅ Registro del consumo de IA por usuario.
- ⏳ Respuestas en streaming: hoy la respuesta aparece completa al terminar.
- ⏳ Cuotas y presupuesto de consumo por empresa y por usuario.
- ⏳ Preguntas en lenguaje natural sobre sus datos —"¿cuánto vendí la semana pasada?"— con su gráfico
  y su exportación.
- ⏳ Avisos proactivos: quiebre de stock, caída de ventas, gasto fuera de lo normal, cliente que dejó
  de comprar.
- ⏳ Asistente por WhatsApp para consultar stock y registrar pedidos.
- ⏳ Registro de compras y gastos tomando una foto del comprobante.
- ⏳ Sugerencias de precio y de reposición a partir de su propio historial.

## 8. Seguridad, datos y operación

**Bajada:** La información es del negocio, no de la plataforma. Puede llevársela cuando quiera y
correr Genix donde quiera.

- ✅ Usuarios y perfiles de acceso por módulo y por acción.
- ✅ Aislamiento de datos por empresa: multiempresa real desde el primer día.
- ✅ Copia de seguridad completa que usted mismo genera, descarga y restaura.
- ✅ Tareas programadas y panel de servidor.
- ✅ En la nube, en su propio servidor o como un solo binario en una PC del local.
- ✅ Interfaz en español e inglés.
- ✅ Código abierto con licencia GPL v3.
- ⏳ Bitácora de auditoría consultable: quién cambió qué y cuándo. El registro por usuario ya se
  guarda en el servidor; falta la pantalla para revisarlo.
- ⏳ Verificación en dos pasos y políticas de contraseña.
- ⏳ Respaldo automático programado y restauración selectiva por tabla.
- ⏳ Aplicación móvil para consultar y vender fuera del local.
- ⏳ Avisos del negocio por correo y WhatsApp.
- ⏳ Migración asistida desde su sistema actual o desde Excel.
