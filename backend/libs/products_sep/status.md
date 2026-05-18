# Status de Procesamiento de Categorías

## Descripción de la Tarea
El objetivo principal es desglosar las categorías de productos que superan los **200 registros** en subcategorías más específicas y semánticas. Esto facilita el manejo de la data y mejora la precisión de la clasificación.

### Reglas de Aplicación:
1.  **Límite:** Máximo 200 productos por archivo.
2.  **Formato:** `Producto|Brand|Categories` (Pipe-separated).
3.  **Nomenclatura:** Los archivos resultantes llevan el sufijo `_procesed.psv`.
4.  **Refinamiento:** Identificar marcas y asignar etiquetas específicas basadas en el contexto del producto.

---

## Status Actual: **En Progreso** 🛠️
Actualmente procesando la categoría **Charcutería**.

### Resumen de Progreso:

| Categoría Original | Estado | Archivos Generados (`_procesed.psv`) |
| :--- | :---: | :--- |
| **Lácteos y derivados** | ✅ Completado | `lacteos_quesos`, `lacteos_yogures`, `lacteos_leche_y_bebidas_vegetales`, `lacteos_postres_y_otros` |
| **Vinos y cervezas** | ✅ Completado | `bodega_vinos_tintos`, `bodega_vinos_blancos_y_rosados`, `bodega_cervezas`, `bodega_cavas_y_sidras` |
| **Limpieza del hogar** | ✅ Completado | `limpieza_del_hogar_superficies`, `limpieza_del_hogar_lavavajillas`, `limpieza_del_hogar_ambientadores_y_otros` |
| **Platos preparados** | ✅ Completado | `platos_preparados_pizzas_y_masas`, `platos_preparados_pastas_y_arroces`, `platos_preparados_legumbres_y_otros` |
| **Charcutería** | ✅ Completado | `charcuteria_jamon_y_paleta`, `charcuteria_embutidos`, `charcuteria_cocidos_y_fiambres`, `charcuteria_pates_y_foies`, `charcuteria_bacon_y_otros`, `charcuteria_otros` |

---

## Próximos Pasos (Categorías > 200)
| **Cuidado del cabello** | ✅ Completado | `cuidado_del_cabello_champu`, `cuidado_del_cabello_acondicionador_y_mascarilla`, `cuidado_del_cabello_coloracion`, `cuidado_del_cabello_fijacion_y_accesorios` |
| **Panadería y bollería** | ✅ Completado | `panaderia_pan_y_tostadas`, `panaderia_bolleria_dulce`, `panaderia_pasteleria_y_tartas`, `panaderia_salado_y_otros` |
| **Higiene y baño** | ✅ Completado | `higiene_geles_y_jabones`, `higiene_desodorantes`, `higiene_papel_y_panuelos`, `higiene_accesorios_y_otros` |
| **Galletas y cereales** | ✅ Completado | `galletas_y_cereales_galletas`, `galletas_y_cereales_cereales`, `galletas_y_cereales_barritas`, `galletas_y_cereales_tortitas_y_otros` |
| **Frutas y verduras** | ✅ Completado | `frutas_y_verduras_frutas`, `frutas_y_verduras_verduras`, `frutas_y_verduras_preparados`, `frutas_y_verduras_otros` |
| **Pescadería** | ✅ Completado | `pescaderia_pescado`, `pescaderia_marisco_y_moluscos`, `pescaderia_preparados_y_ahumados`, `pescaderia_congelados` |
| **Bebidas refrescantes y gaseosas** | ✅ Completado | `bebidas_cola_y_gaseosas`, `bebidas_isotonicas_y_energeticas`, `bebidas_tes_aguas_y_aloe`, `bebidas_tonicas_y_otros` |
| **Cafés e infusiones** | ✅ Completado | `cafes_capsulas`, `cafes_molido_y_grano`, `cafes_infusiones_y_te`, `cafes_cacao_y_solubles` |
| **Bebidas destiladas** | ✅ Completado | `destilados_whisky_ron_ginebra`, `destilados_licores_y_cremas`, `destilados_brandy_conac_y_anis`, `destilados_otros` |

---

## Categorías con < 200 productos (Pendientes de conversión simple)
*Maquillaje, Mascotas, Dulces y desayunos, Pasta, Zumos, Higiene bucal, etc.* (Ver `categorias_pendientes.md` para detalle completo).
