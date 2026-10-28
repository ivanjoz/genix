package salestools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"app/core"
	"app/db"
	sales "app/sales/types"
	"genix-agent/harness"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/ivanjoz/genix-orm/dataframe"
)

// Two days of sale-summary rows: product 1 "Leche Gloria" sells both days, 2 "Leche Laive" once, 3
// "Pan" once. Sums are in sale-summary's order: amount, debt, 4 hidden, quantity, pending.
func withTestFrames(t *testing.T) (today int64, reads *int) {
	core.SetHistoricalUnix(time.Date(2026, 10, 27, 15, 0, 0, 0, time.UTC).Unix())
	t.Cleanup(func() { core.SetHistoricalUnix(0) })
	today = int64(core.FechaUnix())
	quantity := func(units int64) int64 { return units * dataframe.FloatScale }
	rows := []dataframe.Row{
		{Keys: []int64{today - 1}, RowValues: []int64{1, 1}, Sums: []int64{1000, 0, 0, 0, 0, 0, quantity(2), 0}},
		{Keys: []int64{today - 1}, RowValues: []int64{3, 1}, Sums: []int64{300, 0, 0, 0, 0, 0, quantity(3), 0}},
		{Keys: []int64{today}, RowValues: []int64{1, 1}, Sums: []int64{700, 0, 0, 0, 0, 0, quantity(1), 0}},
		{Keys: []int64{today}, RowValues: []int64{2, 1}, Sums: []int64{2500, 0, 0, 0, 0, 0, quantity(5), 0}},
	}
	reads = new(int)
	original := saleFrames
	saleFrames = func(companyID int32) []db.SQLFrame {
		frames := original(companyID)
		frames[0].Source = func(selection dataframe.Selection) ([]dataframe.Row, error) {
			*reads++
			return rows, nil
		}
		return frames
	}
	t.Cleanup(func() { saleFrames = original })

	productNames := map[int32]string{1: "Leche Gloria", 2: "Leche Laive", 3: "Pan"}
	originalCollection := recordCollections[sales.FrameSQLCollectionProducts]
	recordCollections[sales.FrameSQLCollectionProducts] = recordCollection{
		noun: originalCollection.noun,
		search: func(_ int32, name string) ([]recordMatch, error) {
			matches := []recordMatch{}
			for _, id := range []int32{1, 2, 3} {
				if strings.Contains(strings.ToLower(productNames[id]), strings.ToLower(name)) {
					matches = append(matches, recordMatch{ID: id, Name: productNames[id]})
				}
			}
			return matches, nil
		},
		namesByID: func(_ int32, ids []int32) (map[int32]string, error) { return productNames, nil },
	}
	t.Cleanup(func() { recordCollections[sales.FrameSQLCollectionProducts] = originalCollection })
	return today, reads
}

func runTool(t *testing.T, input frameSQLInput) (Answer, error) {
	t.Helper()
	return FrameSQLTool{CompanyID: 1, Language: "es"}.run(input, map[string][]int32{})
}

func TestFrameSQLRanksProductsWithNamesAndAnInlineBar(t *testing.T) {
	withTestFrames(t)
	answer, err := runTool(t, frameSQLInput{
		Dataset: "sale_summary", Message: "Los productos más vendidos.",
		SQL:   `SELECT product_id, SUM(amount) WHERE date IN PERIOD("D-6..D") GROUP BY product_id SORT BY amount DESC LIMIT 10`,
		Table: frameSQLTableInput{Columns: []string{"product_id", "amount"}, InlineBar: "amount"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(answer.Tables) != 1 || len(answer.Charts) != 0 {
		t.Fatalf("expected one table and no chart: %+v", answer)
	}
	products, amounts := answer.Tables[0].Columns[0], answer.Tables[0].Columns[1]
	if products.Label != "Producto" || products.Values[0] != "Leche Laive" || products.Values[1] != "Leche Gloria" {
		t.Fatalf("unexpected product column: %+v", products)
	}
	// Money leaves the tool in currency units.
	if amounts.Format != "money" || amounts.Values[0] != 25.0 || amounts.Values[1] != 17.0 || !amounts.InlineBar {
		t.Fatalf("unexpected amount column: %+v", amounts)
	}
	if !strings.Contains(answer.Summary, "[2] Leche Laive") {
		t.Fatalf("the summary doesn't name the first records: %s", answer.Summary)
	}
}

func TestFrameSQLDrawsADailyLineWithTheMissingDayFilled(t *testing.T) {
	today, _ := withTestFrames(t)
	answer, err := runTool(t, frameSQLInput{
		Dataset: "sale_summary", Message: "Unidades de pan por día.",
		SQL:   `SELECT date, SUM(quantity) WHERE date IN PERIOD("D-2..D") AND product_id = "pan" GROUP BY date`,
		Chart: frameSQLChartInput{Type: "line", X: "date", Y: []string{"quantity"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(answer.Charts) != 1 || len(answer.Tables) != 0 {
		t.Fatalf("expected only a chart: %+v", answer)
	}
	days, units := answer.Charts[0].Columns[0], answer.Charts[0].Columns[1]
	if len(days.Values) != 3 || days.Values[2] != formatUnixDay(today) || days.Format != "date" {
		t.Fatalf("the days of the period are not every point: %+v", days)
	}
	if units.Values[0] != 0.0 || units.Values[1] != 3.0 || units.Values[2] != 0.0 || units.Type != "line" {
		t.Fatalf("unexpected series: %+v", units)
	}
}

func TestFrameSQLChecksTheViewBeforeReading(t *testing.T) {
	_, reads := withTestFrames(t)
	for _, input := range []frameSQLInput{
		{Dataset: "sale_summary", Message: "x", SQL: `SELECT product_id, SUM(amount) WHERE date IN PERIOD("D") GROUP BY product_id`,
			Chart: frameSQLChartInput{X: "product_id", Y: []string{"units"}}},
		{Dataset: "sale_summary", Message: "x", SQL: `SELECT product_id, SUM(amount) WHERE date IN PERIOD("D") GROUP BY product_id`,
			Table: frameSQLTableInput{Columns: []string{"amount:id"}}},
		{Dataset: "sale_summary", Message: "", SQL: `SELECT SUM(amount) WHERE date IN PERIOD("D")`},
		{Dataset: "sale_summary", Message: "x", SQL: `SELECT SUM(sold_units) WHERE date IN PERIOD("D")`},
	} {
		if _, err := runTool(t, input); err == nil {
			t.Fatalf("accepted %+v", input)
		}
	}
	if *reads != 0 {
		t.Fatalf("an invalid call read the frame %d times", *reads)
	}
}

func TestFrameSQLAsksForAnAmbiguousNameAndUsesThePick(t *testing.T) {
	withTestFrames(t)
	input := frameSQLInput{
		Dataset: "sale_summary", Message: "Ventas de leche.",
		SQL: `SELECT SUM(amount) WHERE date IN PERIOD("D-6..D") AND product_id = "leche"`,
	}
	_, err := runTool(t, input)
	var needsChoice *needsChoiceError
	if !errors.As(err, &needsChoice) {
		t.Fatalf("expected a question, got %v", err)
	}
	choices := needsChoice.question.Choices
	if len(choices) != 3 || choices[0].ID != "1" || choices[2].ID != includeAllChoiceID {
		t.Fatalf("unexpected choices: %+v", choices)
	}

	ids, isOffered := pickedIDs(pendingChoice{Choices: choices}, "2")
	if !isOffered {
		t.Fatal("an offered choice was refused")
	}
	if _, isOffered := pickedIDs(pendingChoice{Choices: choices}, "3"); isOffered {
		t.Fatal("a record that wasn't offered was accepted")
	}
	answer, err := FrameSQLTool{CompanyID: 1, Language: "es"}.run(input, map[string][]int32{needsChoice.key: ids})
	if err != nil {
		t.Fatal(err)
	}
	if amount := answer.Tables[0].Columns[0].Values[0]; amount != 25.0 {
		t.Fatalf("the pick was not applied: amount %v", amount)
	}
}

func TestResolvePeriod(t *testing.T) {
	today := int16(time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC).Unix() / 86400)
	day := func(year int, month time.Month, dayOfMonth int) int16 {
		return int16(time.Date(year, month, dayOfMonth, 0, 0, 0, 0, time.UTC).Unix() / 86400)
	}
	for period, want := range map[string][2]int16{
		"D-6..D":        {today - 6, today},
		"M-1":           {day(2026, 9, 1), day(2026, 9, 30)},
		"M":             {day(2026, 10, 1), today},
		"S=1,Y=2026":    {day(2025, 12, 29), day(2026, 1, 4)},
		"D=31":          {day(2026, 8, 31), day(2026, 8, 31)},
		"2026-02-03..D": {day(2026, 2, 3), today},
	} {
		fromDay, toDay, err := resolvePeriod(period, today, 400)
		if err != nil || fromDay != want[0] || toDay != want[1] {
			t.Errorf("%s: got %d..%d %v, want %d..%d", period, fromDay, toDay, err, want[0], want[1])
		}
	}
	for _, period := range []string{"D+1", "Y-3..D", "M=13"} {
		if _, _, err := resolvePeriod(period, today, 400); err == nil {
			t.Errorf("%s was accepted", period)
		}
	}
}

// The tool inside the harness: an ambiguous name pauses the turn with a choice, and the resumed
// call answers with the pick, ending the turn without another model call.
func TestFrameSQLPausesAndResumesInsideTheHarness(t *testing.T) {
	withTestFrames(t)
	arguments, _ := json.Marshal(frameSQLInput{
		Dataset: "sale_summary", Message: "Ventas de leche.",
		SQL: `SELECT SUM(amount) WHERE date IN PERIOD("D-6..D") AND product_id = "leche"`,
	})
	scripted := &scriptedModel{replies: []*schema.Message{schema.AssistantMessage("", []schema.ToolCall{{
		ID: "call-1", Function: schema.FunctionCall{Name: ToolName, Arguments: string(arguments)},
	}})}}
	request := harness.TurnRequest{
		CheckPointID: "sales1", Store: &memoryStore{checkPoints: map[string][]byte{}}, Model: scripted, Instruction: "x",
		Messages: []*schema.Message{schema.UserMessage("ventas de leche")},
		Tools:    []tool.BaseTool{FrameSQLTool{CompanyID: 1, Language: "es"}},
		EndsTurn: func(toolName, toolResult string) bool {
			return toolName == ToolName && !strings.HasPrefix(toolResult, `{"error"`)
		},
		MaxIterations: 4,
	}
	paused := harness.RunTurn(context.Background(), request)
	question, isChoice := paused.Interrupt.Info.(harness.ChoiceQuestion)
	if paused.Err != nil || !isChoice || len(question.Choices) != 3 {
		t.Fatalf("expected a choice pause, got %+v", paused)
	}

	request.Messages = nil
	request.Resume = &harness.TurnResume{InterruptID: paused.Interrupt.InterruptID, Answer: "all"}
	resumed := harness.RunTurn(context.Background(), request)
	if resumed.Err != nil || resumed.EndToolName != ToolName {
		t.Fatalf("the resume did not answer: %+v", resumed)
	}
	answer := Answer{}
	if err := json.Unmarshal([]byte(resumed.EndToolResult), &answer); err != nil || answer.Tables[0].Columns[0].Values[0] != 42.0 {
		t.Fatalf("unexpected answer %s (%v)", resumed.EndToolResult, err)
	}
	if scripted.calls != 1 {
		t.Fatalf("model calls = %d, want 1", scripted.calls)
	}
}

type scriptedModel struct {
	replies []*schema.Message
	calls   int
}

func (scripted *scriptedModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	scripted.calls++
	return scripted.replies[scripted.calls-1], nil
}

func (scripted *scriptedModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	panic("not used")
}

func (scripted *scriptedModel) WithTools([]*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return scripted, nil
}

type memoryStore struct{ checkPoints map[string][]byte }

func (store *memoryStore) Get(_ context.Context, id string) ([]byte, bool, error) {
	content, found := store.checkPoints[id]
	return content, found, nil
}

func (store *memoryStore) Set(_ context.Context, id string, content []byte) error {
	store.checkPoints[id] = content
	return nil
}

// The golden questions are the reference calls a model should write (testdata/golden.jsonl): each
// must still prepare on the real sale frames and plan its table and chart, so a change to the frames
// or the grammar can't silently break them.
func TestGoldenCallsPrepareOnTheSaleFrames(t *testing.T) {
	content, err := os.ReadFile("testdata/golden.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		golden := struct {
			Question string
			Call     frameSQLInput
		}{}
		if err := json.Unmarshal([]byte(line), &golden); err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		statement, err := db.PrepareFrameSQL(golden.Call.Dataset, golden.Call.SQL, sales.SaleFrameSQLFrames(1), db.SQLOptions{MaxRows: MaxTableRows})
		if err == nil {
			_, err = planFrameSQLView(statement.Columns(), golden.Call.Table, golden.Call.Chart)
		}
		if err != nil {
			t.Errorf("%q: %v", golden.Question, err)
		}
	}
}
