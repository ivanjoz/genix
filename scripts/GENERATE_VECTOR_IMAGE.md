# generate_vector_image

Genera imágenes **vectoriales (SVG)** para los assets del sitio con los modelos Recraft a través de
OpenRouter. El recorrido operativo — cómo escribir el prompt, cómo revisar el resultado y dónde
colocar el archivo — está en la skill `create-vector-image`.

```bash
cd scripts
go run . generate_vector_image -prompt "minimal flat line icon of a shopping cart, single accent color"
```

Escribe en `tmp/vector/<slug del prompt>.svg` (ignorado por git) e imprime cada archivo generado y el
costo real de la llamada en USD.

## Flags

| Flag | Default | Descripción |
| --- | --- | --- |
| `-prompt` | — | requerido |
| `-model` | primera entrada de `[[image_models]]` | coincidencia por subcadena (`-model pro`) |
| `-out` | `tmp/vector/<slug>.svg` | ruta destino explícita |
| `-aspect` | `1:1` | relación de aspecto |
| `-n` | `1` | variantes en una sola llamada; la segunda en adelante lleva sufijo `_2`, `_3` |

## Configuración

Lee `agent.openrouter_key` y la tabla `[[image_models]]` de `config.toml` (o de `GENIX_CONFIG_FILE`,
que define `deploy.sh`). El orden del array decide el default:

```toml
[[image_models]]
id       = "recraft/recraft-v4.1-vector"
provider = "openrouter"

[[image_models]]
id       = "recraft/recraft-v4.1-pro-vector"
provider = "openrouter"
```

`[[image_models]]` es una tabla **aparte de `[[models]]`** a propósito: aquel array es el registro de
modelos de chat del agente y su primera entrada decide el default del selector. Estos modelos no son
de chat — su `supported_parameters` viene vacío y devuelven un archivo, no un mensaje — así que
meterlos ahí los ofrecería en el selector del agente y los rutearía a un endpoint que no soportan.
El backend no lee esta tabla; sólo la lee este script.

## Costo

Se cobra por token de imagen, no por token de texto: **~$0.08 USD** por imagen con
`recraft-v4.1-vector` y **~$0.30 USD** con `recraft-v4.1-pro-vector`. `-n` multiplica el costo. Por
eso el script imprime `usage.cost` en cada corrida.

## Endpoint

`POST https://openrouter.ai/api/v1/images`, distinto de `/chat/completions`. La respuesta trae
`data[].b64_json` con el archivo en base64 y `data[].media_type` (`image/svg+xml`), más `usage.cost`.
