# Plan: pestaña `Data` para inspección y limpieza de caché

## Objetivo

Agregar una nueva pestaña `Data` dentro de `HeaderConfig.svelte` para mostrar el estado del caché local y permitir su limpieza.

La vista debe listar, en dos columnas visuales por fila:

1. nombre lógico de la ruta base
2. detalle con:
   - ruta API / query shape
   - tamaño en MB si existe un metadato confiable
   - en caso contrario, cantidad de registros

También debe existir un botón para eliminar el caché local, tanto memoria como `IndexedDB`.

## Hallazgos

### 1. Dónde vive el menú actual

- El menú actual está en [frontend/domain-components/HeaderConfig.svelte](/home/ivanjoz/projects/genix/frontend/domain-components/HeaderConfig.svelte:1).
- Hoy solo tiene dos tabs: `Usuario` y `Config.`.
- `ButtonLayer` ya abre ese contenido dentro del dropdown del header.

### 2. Qué metadatos existen hoy para delta cache

En [frontend/libs/cache/delta-cache.idb.ts](/home/ivanjoz/projects/genix/frontend/libs/cache/delta-cache.idb.ts:1):

- existe la tabla `cacheRoutes` con metadatos por ruta
- existe la tabla `cacheRecords` con filas por registro
- ya se puede contar registros por ruta usando el índice virtual `cR`
- ya existe `listEnvironmentCacheStats`, pero solo agrupa por `module`

En [frontend/libs/cache/delta-cache.types.ts](/home/ivanjoz/projects/genix/frontend/libs/cache/delta-cache.types.ts:1):

- cada `ICacheRouteRow` guarda `fetchedRecordsCount` y `fetchedBytes`

Limitación importante:

- `fetchedBytes` no representa el tamaño actual persistido en `IndexedDB`
- es una métrica acumulada de tráfico recibido por fetch/delta
- por tanto no sirve como “MB ocupados actualmente” por ruta

Conclusión:

- sin leer toda la data, sí es posible obtener por ruta:
  - nombre de ruta
  - ruta API
  - cantidad de registros
- sin leer toda la data, no es posible obtener un tamaño exacto actual en MB por ruta con el diseño actual

### 3. Qué metadatos existen hoy para group cache

En [frontend/libs/cache/group-cache.idb.ts](/home/ivanjoz/projects/genix/frontend/libs/cache/group-cache.idb.ts:1):

- solo existe la tabla `groupRows`
- el índice `[queryShape+id+upc]` permite leer metadata de frescura sin cargar `records`
- no existe tamaño persistido por `queryShape`
- tampoco existe conteo agregado por `queryShape` expuesto por helper

Conclusión:

- sin leer toda la data, sí es posible obtener por `queryShape`:
  - la ruta base derivada desde `queryShape`
  - el `queryShape` completo
  - cantidad de grupos/filas usando índice `queryShape`
- sin leer toda la data, no es posible obtener MB exactos por `queryShape` con el diseño actual

## Decisión técnica

Para esta primera versión:

1. mostrar `MB` solo si existe un metadato confiable y actual
2. como hoy ese metadato no existe ni para delta ni para group cache, mostrar fallback por cantidad de registros
3. no leer snapshots completos para calcular bytes, porque eso rompe el requisito de inspección barata

Esto deja la UI útil y rápida sin introducir una lectura masiva de `IndexedDB`.

## Implementación propuesta

### Fase 1. Helpers de inspección baratos

Agregar helpers nuevos para listar entradas de caché sin cargar payloads completos:

#### Delta cache

Crear un helper que devuelva por ruta:

- `baseRoute`
- `apiRoute`
- `module`
- `recordsCount`
- `sizeMB?: number`
- `source: 'delta'`

Fuente de datos:

- `cacheRoutes.toArray()`
- `countRouteRecords(routeRow)` por cada ruta

Notas:

- `baseRoute` puede derivarse de `route.split('?')[0]`
- `sizeMB` quedará `undefined` por ahora para evitar mentir con `fetchedBytes`

#### Group cache

Agregar helper en `group-cache.idb.ts` que devuelva por `queryShape`:

- `baseRoute`
- `apiRoute`
- `recordsCount`
- `sizeMB?: number`
- `source: 'group'`

Fuente de datos:

- recorrer claves de `groupRows` por índice `queryShape` o por `toCollection().primaryKeys()`
- agrupar sin deserializar `records`

Notas:

- `baseRoute` se obtiene de la primera parte de `queryShape` separada por `|`
- `recordsCount` será la cantidad de buckets/rows por `queryShape`

### Fase 2. Limpieza de caché

Agregar helpers explícitos para borrar:

#### Delta cache

- reutilizar `clearEnvironmentCache`
- añadir limpieza de memoria en el mismo flujo
- asegurar logs `console.debug` / `console.warn`

#### Group cache

Agregar helper para:

- vaciar `groupRows`
- cerrar/eliminar la base `Dexie`
- limpiar la instancia cacheada en `groupCacheDatabasesByName`

Idealmente exponer:

- `clearGroupCache(): Promise<number>`

Donde el retorno sea cantidad de filas o query shapes eliminados para poder mostrar feedback.

### Fase 3. UI en `HeaderConfig.svelte`

Agregar tercera pestaña:

- `Data`

Estado local nuevo:

- lista de filas de cache
- loading de lectura
- loading de limpieza
- posible mensaje de error

UI mínima:

- tabla/lista de dos columnas
- columna 1: `baseRoute`
- columna 2: `apiRoute` + `MB` o `records`
- botón `Eliminar caché`
- botón opcional `Recargar`

Reglas visuales:

- mantener el estilo actual del dropdown
- no agregar más columnas
- si no hay datos, mostrar estado vacío claro

### Fase 4. Recarga y feedback

Cuando el usuario entre a `Data`:

- cargar stats una vez
- permitir recargar manualmente

Cuando elimine caché:

- borrar delta cache
- borrar group cache
- refrescar la lista
- mostrar confirmación con `Notify`

## Cambios esperados en archivos

- [frontend/domain-components/HeaderConfig.svelte](/home/ivanjoz/projects/genix/frontend/domain-components/HeaderConfig.svelte:1)
- [frontend/libs/cache/delta-cache.idb.ts](/home/ivanjoz/projects/genix/frontend/libs/cache/delta-cache.idb.ts:1)
- [frontend/libs/cache/group-cache.idb.ts](/home/ivanjoz/projects/genix/frontend/libs/cache/group-cache.idb.ts:1)
- posiblemente un helper nuevo compartido si conviene normalizar tipos de filas de debug

## Riesgos

1. `fetchedBytes` puede parecer tentador, pero no refleja tamaño actual persistido.
2. En group cache, “records” significará buckets cacheados, no necesariamente cantidad total de filas de negocio dentro de cada bucket.
3. Si más adelante quieres MB reales por ruta sin leer toda la data, habrá que guardar metadatos de bytes al escribir cada fila o cada snapshot.

## Siguiente paso recomendado

Implementar la Fase 1 y Fase 2 primero, porque definen el contrato de datos y la limpieza real. Luego conectar la pestaña `Data`.
