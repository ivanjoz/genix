package genixbot

import (
	"context"
	"encoding/json"
	"errors"
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

func toolCall(name string, arguments map[string]any) llm.Message {
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
		toolCall(ToolRead, nil),
		toolCall(ToolAct, map[string]any{"actions": "134 click"}),
		toolCall(ToolAct, map[string]any{"actions": "139 setValue Cristian Castro"}),
		{Role: "assistant", Content: "Llené Nombre. ¿Guardo el cliente?"},
	}}
	page := &formPage{current: listOutline}
	result, err := RunTurn(context.Background(), model, page, nil, nil, nil, "/crm/customers", "agrega el cliente Cristian Castro", nil)
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
	if strings.Join(result.Actions, " · ") != "act 134 click · act 139 setValue Cristian Castro" {
		t.Fatalf("actions %v", result.Actions)
	}
}

// The reasoning of a step is sent back on the later steps of the turn: it is the turn's plan.
func TestRunTurnKeepsTheStepReasoningWithinTheTurn(t *testing.T) {
	firstStep := toolCall(ToolRead, nil)
	firstStep.ReasoningContent = "Pide crear un cliente y estoy en Clientes: miro la página antes de llenar."
	model := &scriptedModel{replies: []llm.Message{firstStep, {Role: "assistant", Content: "¿Cuál es su RUC?"}}}
	if _, err := RunTurn(context.Background(), model, &formPage{current: listOutline}, nil, nil, nil, "/crm/customers", "agrega un cliente", nil); err != nil {
		t.Fatal(err)
	}
	if got := model.prompts[1][2].ReasoningContent; got != firstStep.ReasoningContent {
		t.Fatalf("second prompt reasoning %q", got)
	}
}

func TestRunTurnGuardsSaveUntilTheUserConfirmed(t *testing.T) {
	unconfirmedSave := &scriptedModel{replies: []llm.Message{
		toolCall(ToolAct, map[string]any{"actions": "137 click"}),
		{Role: "assistant", Content: "No pude guardar."},
	}}
	page := &formPage{current: formOutline}
	result, err := RunTurn(context.Background(), unconfirmedSave, page, nil, nil, nil, "/crm/customers", "guárdalo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps[1].Content != "Error | 137: requiere confirmación del usuario" || len(page.invoked) != 0 {
		t.Fatalf("unconfirmed save ran: %q %v", result.Steps[1].Content, page.invoked)
	}

	// After an answer that asked, a bare "sí" confirms: the save runs and answers in full (new turn).
	asked := []CompletedTurn{{Route: "/crm/customers", UserText: "agrega a Pedro", Answer: "Llené Nombre. ¿Guardo el cliente?"}}
	confirmedSave := &scriptedModel{replies: []llm.Message{
		toolCall(ToolAct, map[string]any{"actions": "! 137 click"}),
		{Role: "assistant", Content: "Listo, guardé el cliente."},
	}}
	result, err = RunTurn(context.Background(), confirmedSave, page, nil, nil, asked, "/crm/customers", "sí", nil)
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

// Without a question before, "!" counts only when the last message asks for the final action itself.
func TestRunTurnAcceptsAFinalActionTheUserAskedFor(t *testing.T) {
	for text, wantsSave := range map[string]bool{"guárdalo": true, "ENVÍALA ya": true, "anula el pedido": true, "crea el cliente Raul": false} {
		model := &scriptedModel{replies: []llm.Message{
			toolCall(ToolAct, map[string]any{"actions": "! 137 click"}),
			{Role: "assistant", Content: "Listo."},
		}}
		page := &formPage{current: formOutline}
		result, err := RunTurn(context.Background(), model, page, nil, nil, nil, "/crm/customers", text, nil)
		if err != nil {
			t.Fatal(err)
		}
		if saved := len(page.invoked) == 1; saved != wantsSave {
			t.Errorf("%q: saved %v, want %v (%q)", text, saved, wantsSave, result.Steps[1].Content)
		}
		if !wantsSave && result.Steps[1].Content != "Error | 137: requiere confirmación del usuario" {
			t.Errorf("%q: %q", text, result.Steps[1].Content)
		}
	}
}

func TestAsksForFinalActionReadsEveryFormOfTheVerbs(t *testing.T) {
	asks := []string{"guárdalo", "Grábalo", "envíala por correo", "ENVÍALA", "elimínalos", "bórrala", "anúlalo", "emite la factura",
		"emítela", "págale", "que le paguen", "cóbrale", "entrégaselo", "guardar", "quiero enviarla", "anulando la venta"}
	for _, text := range asks {
		if !AsksForFinalAction(text) {
			t.Errorf("%q must ask for the final action", text)
		}
	}
	doesNotAsk := []string{"crea el cliente", "regístralo", "abre la página de pagos", "registra el cobro", "muéstrame el borrador",
		"¿cuánto cuesta el envío?", "ventas anuladas", "sí", "ok", "¿cuál es la entrega pendiente?", "entrega el pedido", "mis entregas"}
	for _, text := range doesNotAsk {
		if AsksForFinalAction(text) {
			t.Errorf("%q must not ask for the final action", text)
		}
	}
}

// The bad session of 2026-11-25: an "ok" meant for a navigation offer closed the open form and threw
// the name away. Closing a form that has a save button needs "!" like saving does.
func TestRunTurnGuardsClosingAForm(t *testing.T) {
	unconfirmedClose := &scriptedModel{replies: []llm.Message{
		toolCall(ToolAct, map[string]any{"actions": "138 click"}),
		{Role: "assistant", Content: "¿Cierro el formulario sin guardar?"},
	}}
	page := &formPage{current: formOutline}
	result, err := RunTurn(context.Background(), unconfirmedClose, page, nil, nil, nil, "/crm/customers", "ok", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps[1].Content != "Error | 138: requiere confirmación del usuario" || len(page.invoked) != 0 {
		t.Fatalf("unconfirmed close ran: %q %v", result.Steps[1].Content, page.invoked)
	}
	// Without a save button on the page, a close button is an ordinary click.
	page = &formPage{current: "Page /crm/customers | Clientes\nLayer 136 | Detalle\n Button 138 | Cerrar\n"}
	plainClose := &scriptedModel{replies: []llm.Message{
		toolCall(ToolAct, map[string]any{"actions": "138 click"}),
		{Role: "assistant", Content: "Listo, lo cerré."},
	}}
	if _, err := RunTurn(context.Background(), plainClose, page, nil, nil, nil, "/crm/customers", "cierra eso", nil); err != nil || len(page.invoked) != 1 {
		t.Fatalf("plain close: %v %v", err, page.invoked)
	}
}

func TestMessagesReplaysOlderTurnsByTheirFirstLines(t *testing.T) {
	formTurn := CompletedTurn{Route: "/crm/customers", UserText: "crea el cliente Raul", Answer: "¿Lo guardo?", Steps: []llm.Message{
		toolCall(ToolAct, map[string]any{"actions": "134 click"}),
		{Role: "tool", Content: strings.TrimSpace(formOutline), ToolCallID: ToolAct},
	}}
	offerTurn := CompletedTurn{Route: "/crm/customers", UserText: "mis top 3 productos?", Answer: "¿Te abro Gráficos de Ventas?", Steps: []llm.Message{
		toolCall(ToolRead, nil),
		{Role: "tool", Content: strings.TrimSpace(formOutline), ToolCallID: ToolRead},
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

func TestParseActionsReadsTheActGrammar(t *testing.T) {
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

func TestRunTurnStopsAFailingLoop(t *testing.T) {
	repeat := toolCall(ToolAct, map[string]any{"actions": "! 999"})
	model := &scriptedModel{replies: []llm.Message{repeat, repeat, repeat, repeat}}
	result, err := RunTurn(context.Background(), model, &formPage{current: formOutline}, nil, nil, nil, "/crm/customers", "guárdalo", nil)
	if err != nil {
		t.Fatal(err)
	}
	// The same call failed twice: the third model call gets no tools and its tool call is ignored.
	if len(model.prompts) != 3 || !strings.HasPrefix(result.Answer, "No pude hacerlo: línea \"! 999\"") {
		t.Fatalf("calls %d, answer %q", len(model.prompts), result.Answer)
	}
}

const salesSearchResult = `Funciones
sales_total(period?, products?, clients?, categories?, sites?) | Total de ventas: monto vendido en un periodo
sales_by_product(period?, top?, rank_by?: amount|quantity) | Ventas por producto: monto y unidades, ranking`

// fakeSearch lists the sales functions and records the route of every search.
type fakeSearch struct{ routes []string }

func (search *fakeSearch) Search(_ context.Context, _, _, route string) (string, error) {
	search.routes = append(search.routes, route)
	return salesSearchResult, nil
}

// fakeFunctions answers every call with result, or err, and records the calls.
type fakeFunctions struct {
	result FunctionResult
	err    error
	calls  []string
}

func (functions *fakeFunctions) Call(_ context.Context, name string, arguments map[string]any) (FunctionResult, error) {
	encoded, _ := json.Marshal(arguments)
	functions.calls = append(functions.calls, name+" "+string(encoded))
	return functions.result, functions.err
}

var salesAnswer = FunctionResult{Answer: &FunctionAnswer{Message: "Vendiste S/ 45.00 esta semana.", Summary: "sales_total | period S | filas 1 | total 45.00", Display: "tabla"}}

// open → search → call: the search gets the route open reached, and the call's answer ends the turn
// without another model call. A call after it in the same reply never runs and is not stored.
func TestRunTurnEndsWithTheFunctionAnswer(t *testing.T) {
	callStep := toolCall(ToolCall, map[string]any{"function": "sales_total", "arguments": map[string]any{"period": "S"}})
	callStep.ToolCalls = append(callStep.ToolCalls, toolCall(ToolCalc, map[string]any{"expression": "1+1"}).ToolCalls...)
	model := &scriptedModel{replies: []llm.Message{
		toolCall(ToolOpen, map[string]any{"route": "/sales/sale_orders"}),
		toolCall(ToolSearch, map[string]any{"facts": "", "functions": "ventas totales de un periodo"}),
		callStep,
	}}
	search, functions := &fakeSearch{}, &fakeFunctions{result: salesAnswer}
	result, err := RunTurn(context.Background(), model, &formPage{current: listOutline}, search, functions, nil, "/crm/customers", "¿cuánto vendí esta semana?", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.prompts) != 3 || result.Answer != salesAnswer.Answer.Message || result.Display != "tabla" || result.Choice != nil {
		t.Fatalf("calls %d, result %+v", len(model.prompts), result)
	}
	if strings.Join(search.routes, ",") != "/sales/sale_orders" || strings.Join(functions.calls, ",") != `sales_total {"period":"S"}` {
		t.Fatalf("search routes %v, calls %v", search.routes, functions.calls)
	}
	if len(result.Steps) != 6 || len(result.Steps[4].ToolCalls) != 1 || result.Steps[5].Content != salesAnswer.Answer.Summary {
		t.Fatalf("steps %+v", result.Steps)
	}
	if strings.Join(result.Actions, " · ") != "open /sales/sale_orders · call sales_total" {
		t.Fatalf("actions %v", result.Actions)
	}
}

// A function no search of this turn listed is an error, even one the catalog has; so is a function
// error. Both go back to the model, which answers.
func TestRunTurnReturnsCallErrorsToTheModel(t *testing.T) {
	salesTotal := toolCall(ToolCall, map[string]any{"function": "sales_total", "arguments": map[string]any{}})
	model := &scriptedModel{replies: []llm.Message{salesTotal, {Role: "assistant", Content: "No pude."}}}
	functions := &fakeFunctions{result: salesAnswer}
	result, err := RunTurn(context.Background(), model, &formPage{}, &fakeSearch{}, functions, nil, "/", "¿cuánto vendí?", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps[1].Content != "Error | sales_total: "+ErrorSearchFirst || len(functions.calls) != 0 || result.Answer != "No pude." {
		t.Fatalf("call without search: %q %v %q", result.Steps[1].Content, functions.calls, result.Answer)
	}

	model = &scriptedModel{replies: []llm.Message{
		toolCall(ToolSearch, map[string]any{"facts": "", "functions": "ventas totales"}),
		salesTotal,
		{Role: "assistant", Content: "No encontré ese producto."},
	}}
	functions = &fakeFunctions{err: errors.New("no hay producto con el nombre «lehce»")}
	result, err = RunTurn(context.Background(), model, &formPage{}, &fakeSearch{}, functions, nil, "/", "¿cuánto vendí de lehce?", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps[3].Content != "Error | no hay producto con el nombre «lehce»" || result.Answer != "No encontré ese producto." {
		t.Fatalf("function error: %q %q", result.Steps[3].Content, result.Answer)
	}
}

// An ambiguous name ends the turn with the choice: the reply is its question, the tool message its
// Pregunta line.
func TestRunTurnEndsWithAChoice(t *testing.T) {
	choice := &FunctionChoice{Question: "¿Qué producto es «leche gloria»?", Summary: "Pregunta | ¿Qué producto es «leche gloria»? | #1 Leche Gloria 400 g; #2 Leche Gloria 1 L",
		Options: []ChoiceOption{{ID: "1", Label: "Leche Gloria 400 g"}, {ID: "2", Label: "Leche Gloria 1 L"}, {ID: "all", Label: "Incluir todos"}}}
	model := &scriptedModel{replies: []llm.Message{
		toolCall(ToolSearch, map[string]any{"facts": "", "functions": "ventas totales"}),
		toolCall(ToolCall, map[string]any{"function": "sales_total", "arguments": map[string]any{"products": []string{"leche gloria"}}}),
	}}
	result, err := RunTurn(context.Background(), model, &formPage{}, &fakeSearch{}, &fakeFunctions{result: FunctionResult{Choice: choice}}, nil, "/", "¿cuánto vendí de leche gloria?", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Choice != choice || result.Answer != choice.Question || result.Display != nil || result.Steps[3].Content != choice.Summary {
		t.Fatalf("result %+v", result)
	}
}

// The next turn replays the data answer's Summary as the call's tool message, then the reply.
func TestMessagesReplaysTheFunctionSummary(t *testing.T) {
	dataTurn := CompletedTurn{Route: "/", UserText: "¿cuánto vendí esta semana?", Answer: salesAnswer.Answer.Message, Steps: []llm.Message{
		toolCall(ToolSearch, map[string]any{"facts": "", "functions": "ventas totales"}),
		{Role: "tool", Content: salesSearchResult, ToolCallID: ToolSearch},
		toolCall(ToolCall, map[string]any{"function": "sales_total", "arguments": map[string]any{"period": "S"}}),
		{Role: "tool", Content: salesAnswer.Answer.Summary, ToolCallID: ToolCall},
	}}
	messages := Messages([]CompletedTurn{dataTurn}, "/", "¿y la semana pasada?")
	if messages[5].Content != salesAnswer.Answer.Summary || messages[6].Content != salesAnswer.Answer.Message || messages[7].Content != "Ruta: /\n¿y la semana pasada?" {
		t.Fatalf("messages %+v", messages)
	}
}
