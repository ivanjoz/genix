# PLAN — Meta AI como proveedor de modelos del agente

Objetivo: que `backend/agent` hable con **Meta Model API** en lugar de OpenRouter,
eligiendo el proveedor con el flag `providers.model` de config.toml y tomando
`agent.default_model` como modelo por defecto.

## Hechos del proveedor Meta (verificados en ai.developer.meta.com/docs)

| | Meta Model API | OpenRouter |
|---|---|---|
| Endpoint | `https://api.meta.ai/v1/chat/completions` | `https://openrouter.ai/api/v1/chat/completions` |
| Auth | `Authorization: Bearer <agent.meta_key>` | `Authorization: Bearer <agent.openrouter_key>` |
| Razonamiento | `reasoning_effort: "minimal"\|"low"\|"medium"\|"high"\|"xhigh"` (string plano; `"none"` devuelve **400** en Muse Spark) | objeto `reasoning: {effort, max_tokens, exclude, enabled}` |
| Ruteo de proveedor | no existe | objeto `provider: {order, allow_fallbacks}` |
| Headers extra | ninguno | `HTTP-Referer`, `X-Title` (analítica) |
| Respuesta | `choices[].message`, `usage.{prompt,completion,total}_tokens` | idéntico |
| Modelos | `muse-spark-1.1`, `muse-spark-1.2`, `muse-spark-1.2-contributor` (1M de contexto) | registro actual |

Conclusión: los tipos de wire actuales (`Message`, `Tool`, `ToolCall`, `Choice`,
`Usage`) sirven **sin cambios** para ambos. Sólo difieren endpoint, headers y la
forma en que se serializa el presupuesto de razonamiento.

## Cambios

### 1. `backend/core/security.go` — `EnvStruct`
- **Añadir** `providers.model` (`"meta"|"openrouter"`, vacío ⇒ `meta`).
- **Añadir** `agent.meta_key`.
- **Añadir** `agent.default_model` (modelo por defecto, agnóstico del proveedor).
- **Eliminar** `OPENROUTER_MODEL` — lo reemplaza `DEFAULT_MODEL` (sin retrocompatibilidad,
  regla del proyecto).
- **Conservar** `agent.openrouter_key` para cuando `providers.model=openrouter`.

### 2. `backend/agent/llm/openrouter.go` → renombrar a `client.go`
El paquete deja de ser "cliente de OpenRouter" y pasa a ser "cliente de chat
completions multi-proveedor".

- `ProviderMeta` / `ProviderOpenRouter` como constantes; `ActiveProvider()` lee
  `core.Env.MODEL_PROVIDER` (vacío ⇒ `ProviderOpenRouter`).
- `Client` gana el campo `Provider`; `NewClient()` resuelve **la clave del
  proveedor activo** (`agent.meta_key` u `agent.openrouter_key`) y falla en el arranque con un
  mensaje que nombra la variable que falta.
- `DefaultModelID()` = `core.Env.DEFAULT_MODEL` si está seteado, si no el default
  compilado del proveedor activo (`muse-spark-1.2-contributor` / `openai/gpt-5.6-luna`).
  `OPENROUTER_MODEL` desaparece.
- `ChatRequest` gana `ReasoningEffort string \`json:"reasoning_effort,omitempty"\``.
  Antes de serializar, `Chat()` adapta el request al proveedor activo:
  - **meta**: traduce `Reasoning` → `ReasoningEffort` y anula `Reasoning` + `Provider`
    (campos que Meta no conoce). `Enabled:false` ⇒ `"minimal"`, **no** `"none"`
    (Muse Spark rechaza `"none"` con 400). `MaxTokens`/`Exclude` no tienen
    equivalente: se ignoran.
  - **openrouter**: anula `ReasoningEffort` y mantiene el objeto anidado.
- Endpoint y headers salen de una tablita por proveedor; los headers de analítica
  de OpenRouter sólo se envían a OpenRouter.
- `ProviderOptions` → renombrar a `OpenRouterRouting` (con tag json `provider`)
  para que no se confunda con `providers.model`; `pinnedProvider` →
  `pinnedOpenRouterProvider`.
- Logs y errores dejan de decir "openrouter" y usan el proveedor activo.

### 3. `backend/agent/llm/models.go` — registro
- `ModelConfig` gana `Provider string` (qué proveedor sirve ese modelo).
- Entradas nuevas de Meta: `muse-spark-1.1`, `muse-spark-1.2`,
  `muse-spark-1.2-contributor`, todas con `Reasoning{Effort:"medium", Exclude:true}`
  (el `Exclude` se ignora en Meta, pero mantiene la entrada válida si el modelo
  se sirviera vía OpenRouter).
- `ListModels()` **filtra por proveedor activo**: el desplegable del frontend sólo
  ofrece modelos que la clave configurada puede realmente invocar.
  `HeaderConfig.svelte:71` ya descarta un hash guardado que no esté en la lista y
  cae al default, así que la selección vieja de OpenRouter se auto-corrige sin
  cambios en el frontend.
- `LookupModel` sigue devolviendo config vacía para ids desconocidos (pasan tal cual).

### 4. Ajustes de llamadores (sólo texto/comentarios y mensajes de error)
- `backend/agent/chat_loop.go`: comentario `OPENROUTER_MODEL` → `DEFAULT_MODEL`;
  error `"openrouter chat"` → neutral; comentario de `getLLMClient`.
- `backend/agent/webpage/loop.go`: el bloque de "requisitos del modelo" y los
  comentarios de presupuesto de razonamiento mencionan hy3/OpenRouter; se
  actualizan a Muse Spark. **La lógica no cambia**: sigue dejando `Reasoning` nil
  en el loop principal y fijándolo en subagentes/crítico.
- `backend/agent/llm/prompts.go`: comentario sobre `tool_choice:"required"`.
- `backend/agent/llm/openrouter_test.go` → `client_test.go`: el skip mira la clave
  del proveedor activo.

### 5. Config y documentación
- `config.example.toml`: añadir `providers.model`, `agent.meta_key`, `agent.default_model`,
  `agent.openrouter_key` con comentarios al estilo del archivo.
- `config.1.toml` / `config.toml` (ignorados por git): quitar
  `OPENROUTER_MODEL`, que ya no se lee.
- `backend/agent/AGENTIC_LOOP_DESIGN.md`: la sección de credenciales pasa a
  `providers.model` + `agent.meta_key`/`agent.openrouter_key` + `agent.default_model`.

## Verificación
- `go build ./...` y `go vet ./agent/...` en `backend/`.
- Test en vivo existente (`client_test.go`) contra `muse-spark-1.2-contributor`
  con una tool para confirmar que `tool_calls` vuelve bien y que
  `reasoning_effort` es aceptado.

## Decisiones confirmadas
1. `providers.model` vacío ⇒ **`openrouter`**: ningún despliegue existente se rompe
   por omisión, y Meta se activa explícitamente con `providers.model=meta`.
2. `ListModels()` **filtra por proveedor activo**.
3. `OPENROUTER_MODEL` se **elimina**; `DEFAULT_MODEL` es el único override.
4. `Enabled:false` (subagentes del page builder) ⇒ `reasoning_effort:"minimal"`
   en Meta, porque `"none"` da 400. Es lo más cercano a "sin razonamiento".
