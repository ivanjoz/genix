# Offline Product Search (Draft tecnico v0)

## Objetivo
Definir un indice binario compacto para busqueda offline de productos usando:
- `product_id` (uint16, lossy por overflow)
- `brand_id` (uint8)
- `category_ids` (1 a 4, uint8)
- fonemas de busqueda (cada fonema = uint8)
- numeros normalizados en 1 byte (uint8, lossy)

Este documento se enfoca en la estructura del archivo y el flujo de implementacion.

## Supuestos base
1. Catalogo aproximado: ~5k productos.
2. Diccionario de fonemas: maximo 250 entradas utiles.
3. Codigos reservados:
- `0x00`: separador de fin de registro.
4. Codigos de fonema validos: `0x01..0xFA` (ajustable segun tabla final).
5. Conversion lossy:
- `uint16`: `normalizedProductID = originalID % 65536`
- `uint8`: `normalizedValue = originalValue % 256`
6. Manejo de numeros:
- Los ceros `0` se eliminan de la frase original.
- Los digitos `1-9` tienen sus propios IDs de silaba.
- Se permiten maximo 2 digitos continuos (ej. `750` -> `75`).

## Estructura de un registro
Cada producto se guarda como un registro totalmente variable (sin padding):

```text
[Flags:1][ProductID:2][BrandID?][CategoryIDs:real_category_count][WordSepMask:mask_bytes][PhonemeStream:N][0x00]
```

- `BrandID?` existe solo si `has_brand=1`.
- `CategoryIDs` no reserva slots: se emiten exactamente `real_category_count` bytes.
- `WordSepMask` no es fijo: su tamano real es `mask_bytes`.

#### Byte 1: Flags
Bits (LSB -> MSB):
- bits 1-2: `encoded_category_count` en rango `0..3`; el valor real es `encoded_category_count + 1` (representa `1..4` categorias)
- bit 3: `has_brand` (1 = incluye marca, 0 = sin marca)
- bit 4: `phoneme_count_mode`
  - 0: contador por palabra en 2 bits (hasta 4 fonemas por palabra)
  - 1: contador por palabra en 4 bits (hasta 16 fonemas por palabra)
- bits 5-6: `word_sep_mask_bytes` (`0..3` => `1..4` bytes)
- bits 7-8: reservados para futuro

#### Bytes 2-3: Product ID (uint16)
- little-endian recomendado para Go por simplicidad.
- valor ya normalizado (lossy).

#### Campo opcional: Brand ID (uint8)
- solo se emite si `has_brand=1`.

#### Campo variable: Category IDs
- se emiten `real_category_count` bytes exactos (sin relleno), donde `real_category_count = encoded_category_count + 1`.

#### Campo variable: WordSepMask
- se emiten `word_sep_mask_bytes` bytes exactos (1..4).
- contiene la codificacion compacta de cantidades de fonemas por palabra.

## Layout del payload (fonemas)
Despues de `WordSepMask` se almacena contenido variable:

```text
[PhonemeStream:N][0x00]
```

- `WordSepMask`:
  - si `phoneme_count_mode=0`, cada palabra ocupa 2 bits (valor real `1..4`)
  - si `phoneme_count_mode=1`, cada palabra ocupa 4 bits (valor real `1..16`)
  - el stream de conteos define la segmentacion de palabras
  - el conteo incluye todos los bytes de la palabra en `PhonemeStream` (fonemas + digitos)
- `PhonemeStream`:
  - secuencia lineal de fonemas uint8 (letras o numeros)
  - ejemplo para `750ml` (sin ceros -> `75ml`): `[ID_7, ID_5, ID_ml]`
- `0x00` final: fin de registro

Nota: mantener `0x00` como separador exige que ningun codigo valido de fonema sea `0x00`.

## Reglas de tokenizacion y fonemas
1. Normalizar texto:
- minusculas
- quitar tildes/diacriticos
- eliminar todos los ceros `0`
- colapsar espacios
2. Tokenizar por espacios y separadores comunes.
3. Para cada token:
- Procesar contenido mezclado (letras y numeros):
  - Letras: convertir a secuencia de fonemas del diccionario (silabas).
  - Numeros: convertir a secuencia de IDs de digitos (1-9).
  - Regla de truncamiento numerico: si hay una secuencia de numeros, tomar maximo los primeros 2 digitos (ej. `123` -> `12`).
4. Truncar tokens extremos para respetar limites de mascara por palabra.

## Diccionario de fonemas (250 entradas)
Se recomienda archivo versionado adicional:

```text
phoneme_dict_v1.bin
[version:1][count:1][entries...]
entry = [code:1][len:1][utf8_bytes:len]
```

Reglas:
- codigos estables entre builds (no renumerar sin cambio de version)
- reservar bloques:
  - `0x00` separador
  - `0x01..0x09` digitos (segun definicion en silabas.go)
  - `0xFB..0xFF` control/futuro

## Pipeline de indexacion
1. Cargar productos (ID, marca, categorias, nombre, atributos buscables).
2. Normalizar y tokenizar texto fuente por producto (eliminando ceros).
3. Mapear a fonemas + IDs numericos.
4. Construir registro variable: header + `WordSepMask` + `PhonemeStream`.
5. Escribir registro secuencial en `products_search.idx`.
6. Opcional: generar offset table (`product_offsets.idx`) para acceso directo por posicion.

## Pipeline de busqueda offline
1. Normalizar query de entrada igual que en indexacion (sin ceros).
2. Convertir query a stream de fonemas/numeros.
3. Escanear registros:
- filtrar primero por categoria/marca desde header
- luego hacer matching de stream (exacto o subsecuencia)
4. Rankear resultados:
- coincidencia completa de tokens
- bonus por prefijo

## Separacion de palabras por mask (tu ejemplo)
Si la mask representa conteos `[2,4,2,3]`, entonces los rangos en `PhonemeStream` son:
- palabra 1: indices `[0,2)` (fonemas 1..2)
- palabra 2: indices `[2,6)` (fonemas 3..6)
- palabra 3: indices `[6,8)` (fonemas 7..8)
- palabra 4: indices `[8,11)` (fonemas 9..11)

Nota: el modelo correcto es por longitudes acumuladas (sin solapar).

Ejemplo de numero embebido:
- token original: `100ml`
- token procesado: `1ml` (ceros eliminados)
- stream posible: `[ID_1, ID_ml]`
- conteo de esa palabra en mask: `2`

## Pseudocodigo Go (encoder)
```go
// EncodeProductSearchRecord serializa un producto a formato binario v0.
func EncodeProductSearchRecord(p ProductIndexInput, phonemeDictionary PhonemeDict) ([]byte, error) {
    // 1) Normalizaciones lossy para mantener formato compacto fijo.
    normalizedProductID := uint16(p.ProductID % 65536)
    normalizedBrandID := uint8(p.BrandID % 256)

    // 2) Tokenizacion y conversion a fonemas/numero con reglas uniformes.
    // TokenizeAndNormalizeSpanish debe eliminar '0's.
    tokens := TokenizeAndNormalizeSpanish(p.SearchText) 
    phonemeWords, err := EncodeTokens(tokens, phonemeDictionary)
    if err != nil {
        return nil, err
    }

    // 3) Seleccion de modo de mascara segun max fonemas por palabra.
    maxPhonemesInWord := MaxPhonemesPerWord(phonemeWords)
    usesFourBitWordCounts := maxPhonemesInWord > 4

    // 4) Construccion de cabecera variable sin bytes de relleno.
    wordLengthCounts := BuildWordLengthCounts(phonemeWords)
    packedWordMask, err := PackWordLengthCounts(wordLengthCounts, usesFourBitWordCounts)
    if err != nil {
        return nil, err
    }
    flags := BuildFlagsByte(len(p.CategoryIDs), p.BrandID != 0, usesFourBitWordCounts, len(packedWordMask))

    header := make([]byte, 0, 1+2+1+len(p.CategoryIDs)+len(packedWordMask))
    header = append(header, flags)
    productIDBytes := make([]byte, 2)
    binary.LittleEndian.PutUint16(productIDBytes, normalizedProductID)
    header = append(header, productIDBytes...)
    if p.BrandID != 0 {
        header = append(header, normalizedBrandID)
    }
    header = append(header, NormalizeCategoryIDsLossy(p.CategoryIDs)...)
    header = append(header, packedWordMask...)

    // 5) Stream final de fonemas.
    phonemeStream := FlattenEncodedTokens(phonemeWords)

    // 6) Registro final: header variable + stream + separador final.
    out := append(header, phonemeStream...)
    out = append(out, 0x00)
    return out, nil
}
```

## Pseudocodigo Go (decoder + match)
```go
// MatchRecordAgainstQuery evalua si un registro indexado coincide con una query codificada.
func MatchRecordAgainstQuery(record []byte, query EncodedQuery) (bool, error) {
    // 1) Parsear cabecera variable para filtros baratos (marca/categoria).
    parsedHeader, phonemeStreamStart, err := ParseVariableHeader(record)
    if err != nil {
        return false, err
    }

    if !HeaderPassesFilters(parsedHeader, query.BrandFilter, query.CategoryFilters) {
        return false, nil
    }

    // 2) Reconstruir palabras usando la mascara de separacion y el stream lineal.
    recordStream, err := DecodePhonemeStream(record[phonemeStreamStart:], parsedHeader.WordLengthCounts)
    if err != nil {
        return false, err
    }

    // 3) Matching de subsecuencia/prefijo segun modo de busqueda.
    return ContainsPhonemeSubsequence(recordStream, query.Stream), nil
}
```

## Ejemplo de serializacion (conceptual)
Producto:
- ID: `70000` -> `4464` (`70000 % 65536`)
- Marca: `12`
- Categorias: `[3, 9]`
- Texto: `Vino de 750ml`

Posible stream (ceros eliminados -> `vino de 75ml`):
- `vino` -> `[0x21, 0x34]` (ejemplos IDs)
- `de` -> `[0x18]`
- `75` -> `[ID_7, ID_5]` (IDs especificos de numeros)
- `ml` -> `[0x62]`

Resultado conceptual:

```text
[Flags][ProductID][Brand][Cat1][Cat2][Mask][0x21 0x34 0x18 ID_7 ID_5 0x62][0x00]
```

## Validaciones obligatorias
- rechazar fonema `0x00` en diccionario
- rechazar colision de codigos reservados
- validar que `encoded_category_count` este en rango 0..3 y que `real_category_count = encoded_category_count + 1`
- si `has_brand=0`, no serializar byte de marca
- validar que `word_sep_mask_bytes` coincida exactamente con bytes emitidos
- validar que la suma de conteos en mask sea igual al total de fonemas en stream
- registrar metricas de truncamiento (tokens/fonemas)

## Logging recomendado (debug)
- durante build:
  - producto original -> stream codificado
  - flags calculados
  - bytes finales por registro
- durante query:
  - query normalizada
  - stream query
  - motivo de descarte (categoria, marca, no-match stream)

## Riesgos y decisiones abiertas
1. Ambiguedad por overflow lossy (aceptado por tamano de catalogo).
2. Necesidad de congelar diccionario por version para evitar incompatibilidad.
3. Definir estrategia de ranking final (exact vs fuzzy) segun UX.
4. Confirmar si el separador `0x00` se usa solo fin de registro o tambien entre secciones.

## Siguientes pasos recomendados
1. Implementar `phoneme_dict_v1.json` fuente y generador a binario.
2. Implementar encoder/decoder con golden tests de bytes.
3. Crear benchmark con 5k, 20k y 50k productos.
4. Medir recall/precision de busqueda para ajustar tokenizacion fonetica en espanol.
