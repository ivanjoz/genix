package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"app/db"
	sales "app/sales/types"
	"genix-agent/datafunctions"
	"genix-agent/genixbot"
	"genix-agent/llm"

	"github.com/ivanjoz/genix-orm/dataframe"
)

func TestActionError(t *testing.T) {
	cases := map[string]string{
		"unknown handle id: 566":                           genixbot.ErrorUnknownID,
		"handle 12 (Checkbox) does not implement setValue": "Checkbox no tiene el método setValue",
		"opción 9 no encontrada":                           "opción 9 no encontrada",
	}
	for browserError, want := range cases {
		if got := actionError(browserError); got != want {
			t.Errorf("actionError(%q) = %q, want %q", browserError, got, want)
		}
	}
}

func TestOptionsText(t *testing.T) {
	cases := map[string]string{
		`[]`:   "sin resultados",
		`null`: "",
		`[{"ID":1,"Value":"A"},{"ID":2,"Value":"B"}]`: "[1] A, [2] B",
		`[{"ID":1,"Value":"A"},{"ID":2,"Value":"A"},{"ID":3,"Value":"A"},{"ID":4,"Value":"A"},{"ID":5,"Value":"A"},{"ID":6,"Value":"A"},{"ID":7,"Value":"A"},{"ID":8,"Value":"A"},{"ID":9,"Value":"A"},{"ID":10,"Value":"A"},{"ID":11,"Value":"A"},{"ID":12,"Value":"A"}]`: "[1] A, [2] A, [3] A, [4] A, [5] A, [6] A, [7] A, [8] A, [9] A, [10] A (+2 más)",
	}
	for value, want := range cases {
		if got := optionsText(json.RawMessage(value)); got != want {
			t.Errorf("optionsText(%s) = %q, want %q", value, got, want)
		}
	}
}

// genixBotTestDeps: two sales of two "Leche Gloria" on 2026-11-27 and one of "Pan Francés"; "pan"
// is ambiguous too (Pan Integral sold nothing).
func genixBotTestDeps() datafunctions.Deps {
	today := int16(time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC).Unix() / 86400)
	productNames := map[int32]string{1: "Leche Gloria 400 g", 2: "Leche Gloria 1 L", 3: "Pan Francés", 4: "Pan Integral"}
	summaryRows := []dataframe.Row{}
	for productID, amount := range map[int64]int64{1: 1000, 2: 2500, 3: 300} {
		summaryRows = append(summaryRows, dataframe.Row{Keys: []int64{int64(today), 31}, RowValues: []int64{productID}, Sums: []int64{amount, 0, 2 * dataframe.FloatScale, 0}})
	}
	source := func(rows []dataframe.Row) db.SQLSource {
		return func(selection dataframe.Selection) ([]dataframe.Row, error) {
			selected := []dataframe.Row{}
			for _, row := range rows {
				if row.Keys[0] >= selection.FromKey && row.Keys[0] <= selection.ToKey && slices.Equal(row.Keys[1:1+len(selection.PinnedKeys)], selection.PinnedKeys) {
					selected = append(selected, row)
				}
			}
			return selected, nil
		}
	}
	frames := func(int32) []db.SQLFrame {
		return sales.SaleFrameSQLFramesOver(sales.SaleFrameSQLSources{SaleSummary: source(summaryRows), SaleClientProduct: source(nil), ProductCategories: source(nil)})
	}
	// A record matches when its name holds every word of the query.
	collection := func(noun string, names map[int32]string) datafunctions.RecordCollection {
		return datafunctions.RecordCollection{Noun: noun,
			Search: func(_ int32, name string) ([]datafunctions.RecordMatch, error) {
				matches := []datafunctions.RecordMatch{}
				for id := int32(1); id <= int32(len(names)); id++ {
					if !slices.ContainsFunc(strings.Fields(strings.ToLower(name)), func(word string) bool { return !strings.Contains(strings.ToLower(names[id]), word) }) {
						matches = append(matches, datafunctions.RecordMatch{ID: id, Name: names[id]})
					}
				}
				return matches, nil
			},
			NamesByID: func(int32, []int32) (map[int32]string, error) { return names, nil },
		}
	}
	return datafunctions.Deps{CompanyID: 1, Frames: frames, Today: today, Records: map[string]datafunctions.RecordCollection{
		sales.FrameSQLCollectionProducts:   collection("product|producto", productNames),
		sales.FrameSQLCollectionClients:    collection("client|cliente", map[int32]string{}),
		sales.FrameSQLCollectionCategories: collection("category|categoría", map[int32]string{}),
		sales.FrameSQLCollectionSites:      collection("site|sede", map[int32]string{31: "Lima Centro"}),
	}}
}

type scriptedGenixBotModel struct{ replies []llm.Message }

func (model *scriptedGenixBotModel) Next(context.Context, []llm.Message, []llm.Tool) (llm.Message, error) {
	reply := model.replies[0]
	model.replies = model.replies[1:]
	return reply, nil
}

type salesSearch struct{}

func (salesSearch) Search(context.Context, string, string, string) (string, error) {
	return "Funciones\n" + datafunctions.Functions[0].Signature(), nil
}

func genixBotToolCall(name, arguments string) llm.Message {
	return llm.Message{Role: "assistant", ToolCalls: []llm.ToolCall{{ID: name, Type: "function", Function: llm.ToolCallFunction{Name: name, Arguments: arguments}}}}
}

// A turn ends on the choice card; a click runs the stored call again with the pick, with no model
// call, and the steps read as if the model had called with the pick. A second ambiguous name asks
// again; "Incluir todos" takes every candidate.
func TestResumeFunctionCallRunsThePickAndRewritesTheSteps(t *testing.T) {
	functions := genixBotFunctions{deps: genixBotTestDeps(), accesses: genixBotAccesses()}
	runChoiceTurn := func(arguments string) genixbot.TurnResult {
		t.Helper()
		model := &scriptedGenixBotModel{replies: []llm.Message{
			genixBotToolCall(genixbot.ToolSearch, `{"facts":"","functions":"ventas totales"}`),
			genixBotToolCall(genixbot.ToolCall, `{"function":"sales_total","arguments":`+arguments+`}`),
		}}
		result, err := genixbot.RunTurn(context.Background(), model, nil, salesSearch{}, functions, nil, "/", "¿cuánto vendí?", nil)
		if err != nil || result.Choice == nil {
			t.Fatalf("expected a choice: %v %+v", err, result)
		}
		return result
	}

	asked := runChoiceTurn(`{"period":"D","products":["leche gloria"]}`)
	reply := genixBotReply(asked.Answer, "", 1, asked.Display, asked.Choice)
	if reply.Message != "¿Qué producto es «leche gloria»?" || reply.Pause == nil || reply.Pause.CheckPointID != "" || len(reply.Pause.Choices) != 3 {
		t.Fatalf("choice reply %+v", reply)
	}
	result, steps, err := resumeFunctionCall(context.Background(), functions, asked.Steps, "2")
	if err != nil || result.Answer == nil {
		t.Fatalf("the pick: %v %+v", err, result)
	}
	if got := steps[2].ToolCalls[0].Function.Arguments; got != `{"function":"sales_total","arguments":{"period":"D","products":["#2"]}}` {
		t.Fatalf("rewritten call %s", got)
	}
	if steps[3].Content != result.Answer.Summary || !strings.HasSuffix(result.Answer.Message, "(producto: Leche Gloria 1 L): S/ 25.00 y 2 unidades.") {
		t.Fatalf("answer %+v, tool message %q", result.Answer, steps[3].Content)
	}
	if !strings.HasPrefix(asked.Steps[3].Content, "Pregunta |") {
		t.Fatal("the stored steps were changed in place")
	}
	// A data answer's tables and charts reach the reply.
	display := datafunctions.Answer{Message: "m", Tables: []datafunctions.Table{{}}, Charts: []datafunctions.Chart{{}}}
	if reply := genixBotReply("m", "", 1, display, nil); len(reply.Tables) != 1 || len(reply.Charts) != 1 || reply.Pause != nil {
		t.Fatalf("data reply %+v", reply)
	}

	// Two ambiguous names: the first pick asks about the second, with the first pick already in the call.
	asked = runChoiceTurn(`{"period":"D","products":["leche gloria","pan"]}`)
	result, steps, err = resumeFunctionCall(context.Background(), functions, asked.Steps, "1")
	if err != nil || result.Choice == nil || result.Choice.Question != "¿Qué producto es «pan»?" || steps[3].Content != result.Choice.Summary {
		t.Fatalf("second question: %v %+v", err, result)
	}
	result, steps, err = resumeFunctionCall(context.Background(), functions, steps, datafunctions.IncludeAllChoiceID)
	if err != nil || result.Answer == nil {
		t.Fatalf("include all: %v %+v", err, result)
	}
	if got := steps[2].ToolCalls[0].Function.Arguments; got != `{"function":"sales_total","arguments":{"period":"D","products":["#1","#3","#4"]}}` {
		t.Fatalf("rewritten call %s", got)
	}
	if !strings.HasSuffix(result.Answer.Message, "(productos: Leche Gloria 400 g, Pan Francés, Pan Integral): S/ 13.00 y 4 unidades.") {
		t.Fatalf("include all answer %q", result.Answer.Message)
	}

	if _, _, err := resumeFunctionCall(context.Background(), functions, asked.Steps, "7"); err == nil {
		t.Fatal("an answer that was not offered was accepted")
	}
	if _, _, err := resumeFunctionCall(context.Background(), functions, steps, "1"); !errors.Is(err, errNoPendingChoice) {
		t.Fatalf("a turn that already answered resumed: %v", err)
	}
}

// genix-bot offers every data function to any signed-in user: true only while their accesses are
// unmapped in access.toml.
func TestDataFunctionAccessesAreUnmapped(t *testing.T) {
	accessCatalog, err := os.ReadFile("../backend/access.toml")
	if err != nil {
		t.Fatal(err)
	}
	for _, access := range genixBotAccesses() {
		if regexp.MustCompile(`backend_apis\s*=\s*"[^"]*` + regexp.QuoteMeta(access) + `\b`).Match(accessCatalog) {
			t.Fatalf("%s is mapped to an access now: read the user's grants in genixBotAccesses", access)
		}
	}
}
