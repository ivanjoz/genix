package genixbot

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"genix-agent/llm"
)

const listOutline = `Page /crm/customers | Clientes
Button 134 | Nuevo cliente
`

const formOutline = `Page /crm/customers | Clientes
Layer 136 | Nuevo Cliente
 Button 137 | Guardar
 Button 138 | Cerrar
 Input 139 | Nombre = ""
 Input 144 | Email = ""
 Input 147 | RUC = ""
`

// scriptedModel answers with the next message of its script and records every prompt it got.
type scriptedModel struct {
	replies []llm.Message
	prompts [][]llm.Message
}

func (model *scriptedModel) Next(_ context.Context, messages []llm.Message, _ []llm.Tool) (llm.Message, error) {
	model.prompts = append(model.prompts, append([]llm.Message(nil), messages...))
	reply := model.replies[0]
	model.replies = model.replies[1:]
	return reply, nil
}

func toolCall(name string, arguments map[string]string) llm.Message {
	encoded, _ := json.Marshal(arguments)
	return llm.Message{Role: "assistant", ToolCalls: []llm.ToolCall{{ID: name, Type: "function",
		Function: llm.ToolCallFunction{Name: name, Arguments: string(encoded)}}}}
}

// formPage: the customers list; clicking 134 opens the form, setValue fills an Input.
type formPage struct {
	current string
	invoked []Action
}

func (page *formPage) Outline(context.Context) (string, error) { return page.current, nil }
func (page *formPage) Navigate(context.Context, string) (string, error) {
	page.current = listOutline
	return page.current, nil
}
func (page *formPage) Invoke(_ context.Context, actions []Action) ([]ActionResult, string, error) {
	results := []ActionResult{}
	for _, action := range actions {
		page.invoked = append(page.invoked, action)
		switch {
		case action.ID == "134":
			page.current = formOutline
		case action.Method == "setValue":
			page.current = strings.Replace(page.current, "Input "+action.ID+" | Nombre = \"\"", "Input "+action.ID+" | Nombre = \""+action.Argument+"\"", 1)
		}
		results = append(results, ActionResult{})
	}
	return results, page.current, nil
}

func TestRunTurnFillsTheFormAndAnswersWithDeltas(t *testing.T) {
	model := &scriptedModel{replies: []llm.Message{
		toolCall(ToolVer, nil),
		toolCall(ToolHacer, map[string]string{"acciones": "134 click"}),
		toolCall(ToolHacer, map[string]string{"acciones": "139 setValue Cristian Castro"}),
		{Role: "assistant", Content: "Llené Nombre. ¿Guardo el cliente?"},
	}}
	page := &formPage{current: listOutline}
	result, err := RunTurn(context.Background(), model, page, nil, nil, "/crm/customers", "agrega el cliente Cristian Castro", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer != "Llené Nombre. ¿Guardo el cliente?" {
		t.Fatalf("answer %q", result.Answer)
	}
	firstPrompt := model.prompts[0]
	if len(firstPrompt) != 2 || firstPrompt[0].Content != SystemPrompt ||
		firstPrompt[1].Content != "Ruta: /crm/customers\nagrega el cliente Cristian Castro" {
		t.Fatalf("first prompt %+v", firstPrompt)
	}
	// The click changed the whole page: full outline. The fill changed one line: a delta.
	if got := result.Steps[3].Content; got != strings.TrimSpace(formOutline) {
		t.Fatalf("click result %q", got)
	}
	if got := result.Steps[5].Content; got != "Delta /crm/customers\n~ Input 139 | Nombre = \"Cristian Castro\"" {
		t.Fatalf("fill result %q", got)
	}
	if strings.Join(result.Actions, " · ") != "hacer 134 click · hacer 139 setValue Cristian Castro" {
		t.Fatalf("actions %v", result.Actions)
	}
}

func TestRunTurnGuardsSaveUntilTheUserConfirmed(t *testing.T) {
	unconfirmedSave := &scriptedModel{replies: []llm.Message{
		toolCall(ToolHacer, map[string]string{"acciones": "137 click"}),
		{Role: "assistant", Content: "No pude guardar."},
	}}
	page := &formPage{current: formOutline}
	result, err := RunTurn(context.Background(), unconfirmedSave, page, nil, nil, "/crm/customers", "guárdalo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps[1].Content != "Error | 137: requiere confirmación del usuario" || len(page.invoked) != 0 {
		t.Fatalf("unconfirmed save ran: %q %v", result.Steps[1].Content, page.invoked)
	}

	// "!" only counts after an answer that asked; then the save runs and answers in full (new turn).
	asked := []CompletedTurn{{Route: "/crm/customers", UserText: "agrega a Pedro", Answer: "Llené Nombre. ¿Guardo el cliente?"}}
	confirmedSave := &scriptedModel{replies: []llm.Message{
		toolCall(ToolHacer, map[string]string{"acciones": "! 137 click"}),
		{Role: "assistant", Content: "Listo, guardé el cliente."},
	}}
	result, err = RunTurn(context.Background(), confirmedSave, page, nil, asked, "/crm/customers", "sí, guárdalo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.invoked) != 1 || !page.invoked[0].Confirmed || result.Steps[1].Content != strings.TrimSpace(formOutline) {
		t.Fatalf("confirmed save: %v %q", page.invoked, result.Steps[1].Content)
	}
	// The history is replayed whole before the new message.
	prompt := confirmedSave.prompts[0]
	if prompt[1].Content != "Ruta: /crm/customers\nagrega a Pedro" || prompt[2].Content != "Llené Nombre. ¿Guardo el cliente?" {
		t.Fatalf("history %+v", prompt)
	}
}

// The bad session of 2026-11-25: an "ok" meant for a navigation offer closed the open form and threw
// the name away. Closing a form that has a save button needs "!" like saving does.
func TestRunTurnGuardsClosingAForm(t *testing.T) {
	unconfirmedClose := &scriptedModel{replies: []llm.Message{
		toolCall(ToolHacer, map[string]string{"acciones": "138 click"}),
		{Role: "assistant", Content: "¿Cierro el formulario sin guardar?"},
	}}
	page := &formPage{current: formOutline}
	result, err := RunTurn(context.Background(), unconfirmedClose, page, nil, nil, "/crm/customers", "ok", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps[1].Content != "Error | 138: requiere confirmación del usuario" || len(page.invoked) != 0 {
		t.Fatalf("unconfirmed close ran: %q %v", result.Steps[1].Content, page.invoked)
	}
	// Without a save button on the page, a close button is an ordinary click.
	page = &formPage{current: "Page /crm/customers | Clientes\nLayer 136 | Detalle\n Button 138 | Cerrar\n"}
	plainClose := &scriptedModel{replies: []llm.Message{
		toolCall(ToolHacer, map[string]string{"acciones": "138 click"}),
		{Role: "assistant", Content: "Listo, lo cerré."},
	}}
	if _, err := RunTurn(context.Background(), plainClose, page, nil, nil, "/crm/customers", "cierra eso", nil); err != nil || len(page.invoked) != 1 {
		t.Fatalf("plain close: %v %v", err, page.invoked)
	}
}

func TestMessagesReplaysOlderTurnsByTheirFirstLines(t *testing.T) {
	formTurn := CompletedTurn{Route: "/crm/customers", UserText: "crea el cliente Raul", Answer: "¿Lo guardo?", Steps: []llm.Message{
		toolCall(ToolHacer, map[string]string{"acciones": "134 click"}),
		{Role: "tool", Content: strings.TrimSpace(formOutline), ToolCallID: ToolHacer},
	}}
	offerTurn := CompletedTurn{Route: "/crm/customers", UserText: "mis top 3 productos?", Answer: "¿Te abro Gráficos de Ventas?", Steps: []llm.Message{
		toolCall(ToolVer, nil),
		{Role: "tool", Content: strings.TrimSpace(formOutline), ToolCallID: ToolVer},
	}}
	messages := Messages([]CompletedTurn{formTurn, offerTurn}, "/crm/customers", "ok")
	if got := messages[3].Content; got != "Page /crm/customers | Clientes" {
		t.Fatalf("older turn's tool result %q", got)
	}
	if got := messages[7].Content; got != strings.TrimSpace(formOutline) {
		t.Fatalf("last turn's tool result %q", got)
	}
	// Compacting the prompt never edits the stored history.
	if formTurn.Steps[1].Content != strings.TrimSpace(formOutline) {
		t.Fatal("Messages changed the history it was given")
	}
	history := make([]CompletedTurn, HistoryTurns+2)
	if got := len(Messages(history, "/crm/customers", "ok")); got != 1+HistoryTurns*2+1 {
		t.Fatalf("%d messages: the history must be cut to %d turns", got, HistoryTurns)
	}
}

func TestParseActionsReadsTheHacerGrammar(t *testing.T) {
	actions, err := ParseActions("79 setValue Pedro Pascal\n82 select 2\n! 77 click\n75 selectRow 75:300")
	if err != nil {
		t.Fatal(err)
	}
	if actions[0].Argument != "Pedro Pascal" || actions[1].Argument != "2" || !actions[2].Confirmed || actions[3].Argument != "75:300" {
		t.Fatalf("%+v", actions)
	}
	if _, err := ParseActions("guardar el formulario"); err == nil {
		t.Fatal("free text must not parse")
	}
}
