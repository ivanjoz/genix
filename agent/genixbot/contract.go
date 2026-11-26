// Package genixbot is the runtime of genix-bot, the LFM3 model fine-tuned in genix-local-ai: its
// contract (system prompt, tools, user-message frame), the `hacer` grammar, the `calcular` evaluator
// and the turn loop. Everything genix-bot needs is in its weights, so the prompt is minimal: no
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
const ContractVersion = 2

// SystemPrompt is the whole system message genix-bot is trained and served with.
const SystemPrompt = `Eres Genix Bot, el asistente de Genix (ERP y tienda web para pequeñas empresas en Perú). Responde en español, breve, tuteando. Opera la página con las herramientas; usa solo ids y rutas que aparezcan en la página o en los hechos. Nunca guardes, envíes ni elimines sin una confirmación explícita del usuario en su último mensaje. Para cualquier cuenta usa calcular. Si no tienes un hecho que lo respalde, di que no está documentado.`

// Tool names, in Spanish by decision (genix-local-ai plan 08).
const (
	ToolIr       = "ir"
	ToolHacer    = "hacer"
	ToolHechos   = "hechos"
	ToolVer      = "ver"
	ToolCalcular = "calcular"
)

// Tools are the declarations sent with every call. llm.Tool keeps llama-server's key order (name,
// description, parameters), so the training render and the served prompt are the same tokens.
var Tools = []llm.Tool{
	contractTool(ToolIr, "Abre una ruta de Genix.", map[string]any{"ruta": stringParameter()}, []string{"ruta"}),
	contractTool(ToolHacer, "Ejecuta acciones en la página, una por línea: id método argumento.", map[string]any{"acciones": stringParameter()}, []string{"acciones"}),
	contractTool(ToolHechos, "Busca hechos documentados de una ruta y conceptos del negocio.", map[string]any{"consulta": stringParameter(), "ruta": stringParameter()}, []string{"consulta", "ruta"}),
	contractTool(ToolVer, "Lee la página actual.", map[string]any{}, []string{}),
	contractTool(ToolCalcular, "Calcula una expresión aritmética: números, + - * / ^ y paréntesis.", map[string]any{"expresion": stringParameter()}, []string{"expresion"}),
}

func contractTool(name, description string, properties map[string]any, required []string) llm.Tool {
	return llm.Tool{Type: "function", Function: llm.ToolFunction{
		Name: name, Description: description,
		Parameters: map[string]any{"type": "object", "properties": properties, "required": required},
	}}
}

func stringParameter() map[string]any { return map[string]any{"type": "string"} }

// UserMessage frames what the user typed with the route they are on, so the model decides whether it
// must navigate without calling `ver` first.
func UserMessage(route, text string) string {
	return "Ruta: " + route + "\n" + strings.TrimSpace(text)
}
