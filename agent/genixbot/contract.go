// Package genixbot is the runtime of genix-bot, the LFM3 model fine-tuned in genix-local-ai: its
// contract (system prompt, tools, user-message frame), the `act` grammar, the `calc` evaluator and
// the turn loop. Everything genix-bot needs is in its weights, so the prompt is minimal: no
// discovery planner, no SystemPromptChat, no snapshot grammar — pages are compact outlines
// (outline.Render).
//
// The training data imports this package (genix-local-ai/generators), and the evaluation runs this
// same loop (genix-local-ai cmd/eval-loop), so training, evaluation and production share one contract.
// Changing any string here changes what the model sees: bump ContractVersion and regenerate the data.
package genixbot

import (
	"strings"

	"genix-agent/llm"
)

// ContractName selects this runtime for a [[models]] entry (`contract = "genix-bot"` in config.toml).
const ContractName = "genix-bot"

// ContractVersion is stamped on every training sample; a change to the prompt, the tools or the
// user-message frame bumps it and the affected datasets are regenerated (never mixed).
const ContractVersion = 4

// SystemPrompt is the whole system message genix-bot is trained and served with.
const SystemPrompt = `Eres Genix Bot, el asistente de Genix (ERP y tienda web para pequeñas empresas en Perú). Responde en español, breve, tuteando. Antes de actuar, decide qué pide el usuario: datos de su negocio → search de funciones y call; una acción que esta página no hace → search de hechos y funciones antes de abrir otra página; qué es algo, si Genix lo tiene o cómo se hace → search de hechos y responde solo con ellos; ir a una página → open. Usa solo ids y rutas que aparezcan en la página o en los hechos. Guarda, envía o elimina solo si el usuario lo pidió explícitamente en su último mensaje; si no, pregúntale. Para cualquier cálculo usa calc. Si no tienes un hecho que lo respalde, di que no está documentado.`

// Tool names, in English by decision (genix-local-ai plan 11 §2); their descriptions are Spanish.
const (
	ToolOpen   = "open"
	ToolAct    = "act"
	ToolRead   = "read"
	ToolSearch = "search"
	ToolCall   = "call"
	ToolCalc   = "calc"
)

// Tools are the declarations sent with every call. llm.Tool keeps llama-server's key order (name,
// description, parameters), so the training render and the served prompt are the same tokens. Data
// functions are not declared here: `search` lists them and `call` runs them, so the catalog grows
// without changing the contract.
var Tools = []llm.Tool{
	contractTool(ToolOpen, "Abre una ruta de Genix.", map[string]any{"route": stringParameter()}, []string{"route"}),
	contractTool(ToolAct, "Ejecuta acciones en la página, una por línea: id método argumento.", map[string]any{"actions": stringParameter()}, []string{"actions"}),
	contractTool(ToolRead, "Lee la página actual.", map[string]any{}, []string{}),
	contractTool(ToolSearch, "Busca hechos de Genix (facts) y funciones de datos (functions); pide ambos en la misma llamada cuando no sabes cuál responde, y deja vacío el que no necesites.",
		map[string]any{"facts": stringParameter(), "functions": stringParameter()}, []string{"facts", "functions"}),
	contractTool(ToolCall, "Ejecuta una función de datos que search mostró en este turno; su respuesta, con tabla o gráfico, termina el turno.",
		map[string]any{"function": stringParameter(), "arguments": map[string]any{"type": "object"}}, []string{"function", "arguments"}),
	contractTool(ToolCalc, "Calcula una expresión aritmética: números, + - * / ^ y paréntesis.", map[string]any{"expression": stringParameter()}, []string{"expression"}),
}

func contractTool(name, description string, properties map[string]any, required []string) llm.Tool {
	return llm.Tool{Type: "function", Function: llm.ToolFunction{
		Name: name, Description: description,
		Parameters: map[string]any{"type": "object", "properties": properties, "required": required},
	}}
}

func stringParameter() map[string]any { return map[string]any{"type": "string"} }

// UserMessage frames what the user typed with the route they are on, so the model decides whether it
// must navigate without calling `read` first.
func UserMessage(route, text string) string {
	return "Ruta: " + route + "\n" + strings.TrimSpace(text)
}
