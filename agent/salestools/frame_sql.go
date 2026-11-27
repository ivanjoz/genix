// Package salestools is the agent's sales FrameSQL tool (backend/genix-orm/dataframe/framesql/
// FRAMESQL.md §6–7): the model writes one SQL statement over a sale DataFrame and how to show its
// result, and the tool's answer — a message, a table, a chart — is the turn's reply, with no other
// model call. Ported from berryapps' agenttools. The period notation, the record names and the
// answer's table and chart are datafunctions', shared with genix-bot's data functions.
package salestools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"app/core"
	"app/db"
	sales "app/sales/types"
	"genix-agent/datafunctions"
	"genix-agent/harness"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
)

// ToolName is the sales FrameSQL tool as the model calls it.
const ToolName = "sales_frame_sql"

// saleFrames and recordCollections are what the tool reads; tests swap in frames and records in memory.
var (
	saleFrames        = sales.SaleFrameSQLFrames
	recordCollections = datafunctions.DatabaseRecordCollections()
)

func init() {
	// The interrupt state travels in the checkpoint, gob-encoded.
	schema.RegisterName[pendingChoice]("genix.salestools.pendingChoice")
}

// FrameSQLTool reads one company's sale frames, in the language of the turn's answer.
type FrameSQLTool struct {
	CompanyID int32
	// Language is "es" or "en": headers, notes and choice questions.
	Language string
}

type frameSQLInput struct {
	Dataset string                   `json:"dataset"`
	SQL     string                   `json:"sql"`
	Message string                   `json:"message"`
	Table   datafunctions.TableInput `json:"table"`
	Chart   datafunctions.ChartInput `json:"chart"`
}

// pendingChoice is the tool's interrupt state: the names the user already picked, and the question
// asked now. A resumed call validates the answer against Choices: the browser echoes it back, and
// is never trusted with an ID it wasn't offered.
type pendingChoice struct {
	// Picks are record IDs by name key (collection:normalized words).
	Picks   map[string][]int32
	Key     string
	Choices []datafunctions.ChoiceOption
}

// choiceQuestion is a choice as the harness pauses the turn with it.
func choiceQuestion(question string, options []datafunctions.ChoiceOption) harness.ChoiceQuestion {
	choiceQuestion := harness.ChoiceQuestion{Question: question}
	for _, option := range options {
		choiceQuestion.Choices = append(choiceQuestion.Choices, harness.Choice(option))
	}
	return choiceQuestion
}

// saleDatasets describe the sale frames for the model: what one row is and what its numbers count.
var saleDatasets = map[string]struct{ title, description string }{
	"sale-summary": {"Sales per day, site and product",
		"One row per product sold on a day in a site (sede): the amount sold, what is still unpaid, and the units sold and still undelivered. Cancelled sales are left out; site_id 0 is a sale with no warehouse. Use it for totals and to group or filter by site."},
	"sale-client-product": {"Sales per day, client and product",
		"One row per client and product on a day, with the same sums as sale_summary. Use it only to group or filter by client; sales without a client have client_id 0."},
	// The category views repeat a product's sales under each of its categories (framesql.View).
	"sale-category": {"Sales per day, site, product and category",
		"sale_summary with each row repeated under every category of its product (category_id 0: no category). Use it only to group or filter by category: a product in two categories counts in both, so the categories add up to more than the total sales; take totals from sale_summary."},
	"sale-client-category": {"Sales per day, client, product and category",
		"sale_client_product with each row repeated under every category of its product (category_id 0: no category). Use it only to group or filter by client and category together; the categories add up to more than the total sales."},
}

// frameSQLGrammar is the language as the model needs it (the parser's errors teach the rest).
const frameSQLGrammar = `SELECT item, ... WHERE condition AND ... [GROUP BY ...] [SORT BY item [DESC], ...] [LIMIT n]
- No FROM: the statement reads the dataset argument.
- An item is named after its column: SUM(amount) and AVG(amount) are amount, COUNT(*) is count, WEEK(date) is week, MONTH(date) is month; name any other expression with AS (SUM(amount) / COUNT(*) AS ticket). SORT BY, the table and the chart use these names; two items of one name need AS on one.
- WHERE must bound date: date IN PERIOD("<period>"), at most 400 days.
- product_id, client_id, category_id and site_id take = value or IN (value, ...). Write a record's name in quotes exactly as the user wrote it, never shortened or widened to find similar records (product_id IN ("name 1", "name 2")): the tool looks it up and asks the user when it is ambiguous.
- Names and periods go in double quotes: an apostrophe inside needs nothing ("O'Brien"), a double quote is written twice ("Pantalla 24""").
- SUM(col) adds a summed column. COUNT(*) counts the dataset's rows in each group (with a day key, the days a product has sales), never sales or units. AVG(col) is SUM(col) / COUNT(*). + - * / combine them. No MIN, MAX, HAVING, OR, joins or subqueries.
- GROUP BY date, site_id, product_id, client_id, category_id, WEEK(date) (its Monday) or MONTH(date) (its 1st). A SELECT column that is not an aggregate must be in GROUP BY.
- Money is in cents and the answer formats it: never divide it by 100. Quantities are units with their fraction.
- LIMIT at most 400. Without SORT BY, rows come by group ascending.
`

// frameSQLExamples are two calls the model can adapt: a ranking with inline bars, and a weekly line
// per named product.
const frameSQLExamples = `{"dataset":"sale_summary","sql":"SELECT product_id, SUM(amount) WHERE date IN PERIOD(\"D-29..D\") GROUP BY product_id SORT BY amount DESC LIMIT 10","message":"Los 10 productos más vendidos de los últimos 30 días.","table":{"columns":["product_id","amount"],"sort_by":[],"inline_bar":"amount"},"chart":{"type":"auto","x":"","y":[],"y2":[],"series_by":""}}
{"dataset":"sale_summary","sql":"SELECT WEEK(date), product_id, SUM(quantity) WHERE date IN PERIOD(\"M-2..M\") AND product_id IN (\"name 1\", \"name 2\") GROUP BY WEEK(date), product_id","message":"Unidades vendidas por semana de name 1 y name 2 en los últimos tres meses.","table":{"columns":[],"sort_by":[],"inline_bar":""},"chart":{"type":"line","x":"week","y":["quantity"],"y2":[],"series_by":"product_id"}}
`

// Info is the tool's schema. The datasets, the grammar and the period notation reach the model
// through the sql argument's description.
func (frameSQLTool FrameSQLTool) Info(context.Context) (*schema.ToolInfo, error) {
	frames := saleFrames(frameSQLTool.CompanyID)
	datasetNames := []any{}
	var notation strings.Builder
	notation.WriteString("One SQL statement over the dataset, without FROM, that totals, ranks or averages its values by day, week, month, site, product, client or product category.\n")
	notation.WriteString("The datasets hold sums per day and row; a row whose sums are all 0 is not stored.\n")
	for _, frame := range frames {
		dataset := saleDatasets[frame.Name]
		datasetNames = append(datasetNames, frameSQLName(frame.Name))
		fmt.Fprintf(&notation, "- %s: %s. %s\n  Keys %s; Rows %s; Sums %s\n", frameSQLName(frame.Name), dataset.title, dataset.description,
			describeColumns(frame.Keys), describeColumns(frame.Rows), describeColumns(frame.Sums))
	}
	notation.WriteString(frameSQLGrammar)
	notation.WriteString("Example calls:\n" + frameSQLExamples)
	notation.WriteString("PERIOD() takes the period notation. " + datafunctions.PeriodNotation)

	stringList := func(description string) *jsonschema.Schema {
		return &jsonschema.Schema{Type: "array", Items: &jsonschema.Schema{Type: "string"}, Description: description}
	}
	strictObject := func(description string, properties map[string]*jsonschema.Schema, order []string) *jsonschema.Schema {
		object := &jsonschema.Schema{Type: "object", Description: description, Properties: jsonschema.NewProperties(), Required: order, AdditionalProperties: jsonschema.FalseSchema}
		for _, name := range order {
			object.Properties.Set(name, properties[name])
		}
		return object
	}
	parameters := strictObject("", map[string]*jsonschema.Schema{
		"dataset": {Type: "string", Enum: datasetNames, Description: "The dataset the statement reads: its FROM."},
		"sql":     {Type: "string", Description: notation.String()},
		"message": {Type: "string", Description: "The sentence the user reads above the result, in the user's language: what it shows, for which period and records. The table and the chart show the numbers."},
		"table": strictObject("The result's table. Without a chart it always shows, with every column when columns is empty; with a chart, only when columns lists some.", map[string]*jsonschema.Schema{
			"columns":    stringList("The result's columns to show, in order. A record column shows the record's name; column:id shows its ID."),
			"sort_by":    stringList(`The rows' order, each "column" or "column DESC" (a record column sorts by name). Empty keeps the statement's order.`),
			"inline_bar": {Type: "string", Description: "A value column of the table drawn as a bar under each row, to read a ranking at a glance. Empty for none."},
		}, []string{"columns", "sort_by", "inline_bar"}),
		"chart": strictObject("The result's chart; x empty for none.", map[string]*jsonschema.Schema{
			"type":      {Type: "string", Enum: []any{"auto", "bar", "line"}, Description: "auto draws records as ranked bars and days as lines."},
			"x":         {Type: "string", Description: "The GROUP BY column along the chart: days, weeks, months or records. Empty for no chart."},
			"y":         stringList("The value columns drawn against x."),
			"y2":        stringList("Value columns on a second scale, when their size differs from y's (units beside money). Usually empty."),
			"series_by": {Type: "string", Description: "A second GROUP BY column whose values are one series each (a line per product over the days), at most 12. Takes exactly one y and no y2. Empty for none."},
		}, []string{"type", "x", "y", "y2", "series_by"}),
	}, []string{"dataset", "sql", "message", "table", "chart"})

	return &schema.ToolInfo{
		Name: ToolName,
		Desc: "Answers a question about sales with one SQL statement over the sale datasets: totals, rankings, averages and trends " +
			"per day, week, month, product, client or product category, shown as a table, a chart or both. Its answer ends the turn: call it alone, " +
			"after nothing else, when the user asks for sales figures.",
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(parameters),
	}, nil
}

// frameSQLName is a frame's name as the model writes it.
func frameSQLName(frameName string) string { return strings.ReplaceAll(frameName, "-", "_") }

func describeColumns(columns []db.SQLColumn) string {
	described := []string{}
	for _, column := range columns {
		if column.Hidden {
			continue
		}
		kindLabel := map[db.SQLKind]string{db.SQLDay: "day", db.SQLCents: "money", db.SQLInteger: "number", db.SQLQuantity: "units"}[column.Kind]
		if column.Kind == db.SQLRef {
			kindLabel, _, _ = strings.Cut(recordCollections[column.Collection].Noun, "|")
		}
		described = append(described, column.Name+" ("+kindLabel+")")
	}
	return strings.Join(described, ", ")
}

// InvokableRun runs the statement and returns the Answer as JSON. A name that needs the user's pick
// pauses the turn; the resumed call runs the same arguments with the pick in its state.
func (frameSQLTool FrameSQLTool) InvokableRun(ctx context.Context, argumentsJSON string, _ ...tool.Option) (string, error) {
	input := frameSQLInput{}
	if err := json.Unmarshal([]byte(argumentsJSON), &input); err != nil {
		return "", fmt.Errorf("the arguments are not valid JSON: %w", err)
	}

	picks := map[string][]int32{}
	if wasInterrupted, hasState, pending := tool.GetInterruptState[pendingChoice](ctx); wasInterrupted && hasState {
		picks = pending.Picks
		isResumeTarget, hasAnswer, answer := tool.GetResumeContext[string](ctx)
		ids, isOffered := datafunctions.PickedIDs(pending.Choices, answer)
		if !isResumeTarget || !hasAnswer || !isOffered {
			// The answer isn't one of the choices: ask the same question again.
			core.Log("agent.salestools choice not offered key::", pending.Key, " answer::", answer)
			question := datafunctions.Localized(frameSQLTool.Language, "Elige una de las opciones.", "Pick one of the options.")
			return "", tool.StatefulInterrupt(ctx, choiceQuestion(question, pending.Choices), pending)
		}
		picks[pending.Key] = ids
	}

	answer, err := frameSQLTool.run(input, picks)
	var needsChoice *datafunctions.NeedsChoiceError
	if errors.As(err, &needsChoice) {
		return "", tool.StatefulInterrupt(ctx, choiceQuestion(needsChoice.Choice.Question, needsChoice.Choice.Options),
			pendingChoice{Picks: picks, Key: needsChoice.Key, Choices: needsChoice.Choice.Options})
	}
	if err != nil {
		return "", err
	}
	encodedAnswer, err := json.Marshal(answer)
	return string(encodedAnswer), err
}

func (frameSQLTool FrameSQLTool) run(input frameSQLInput, picks map[string][]int32) (datafunctions.Answer, error) {
	language := frameSQLTool.Language
	if strings.TrimSpace(input.Message) == "" {
		return datafunctions.Answer{}, errors.New("message is empty: write the sentence the user reads above the result")
	}
	statement, err := db.PrepareFrameSQL(input.Dataset, input.SQL, saleFrames(frameSQLTool.CompanyID), db.SQLOptions{MaxRows: datafunctions.MaxTableRows})
	if err != nil {
		return datafunctions.Answer{}, err
	}
	// The table and the chart are checked before the read: a name lookup may ask the user, and the
	// call must not fail once they answered.
	view, err := datafunctions.PlanView(statement.Columns(), input.Table, input.Chart)
	if err != nil {
		return datafunctions.Answer{}, err
	}
	result, err := statement.Run(db.SQLResolvers{
		Period: func(text string) (int64, int64, error) {
			fromDay, toDay, err := datafunctions.ResolvePeriod(text, core.FechaUnix(), db.FrameMaxDays, "en")
			return int64(fromDay), int64(toDay), err
		},
		Names: func(column db.SQLColumn, names []string) ([]int64, error) {
			ids, err := datafunctions.ResolveNames(recordCollections, frameSQLTool.CompanyID, language, column.Collection, names, picks)
			ids64 := make([]int64, len(ids))
			for i, id := range ids {
				ids64[i] = int64(id)
			}
			return ids64, err
		},
	})
	if errors.Is(err, db.ErrFrameNotBuilt) {
		return datafunctions.Answer{}, errors.New("the dataset is being rebuilt after a change and can't be read for about 20 minutes: tell the user to try again later")
	}
	if err != nil {
		return datafunctions.Answer{}, err
	}
	summary := fmt.Sprintf("Ran FrameSQL on %s (%d rows): %s", input.Dataset, result.RowCount, input.SQL)
	if result.RowCount == 0 {
		return datafunctions.Answer{Message: input.Message + "\n\n" + datafunctions.Localized(language, "La consulta no encontró datos.", "The query found no data."), Summary: summary}, nil
	}
	namesByColumn, err := datafunctions.RecordNames(recordCollections, frameSQLTool.CompanyID, result)
	if err != nil {
		return datafunctions.Answer{}, fmt.Errorf("load the record names: %w", err)
	}

	answer := datafunctions.Answer{Message: input.Message, Summary: summary + datafunctions.FirstRecords(result, namesByColumn)}
	// A statement without LIMIT stops at the table's rows: say so, or the user reads a partial total as all of it.
	if result.Truncated && result.RowCount == datafunctions.MaxTableRows {
		answer.Message += "\n\n" + fmt.Sprintf(datafunctions.Localized(language, "Se muestran los primeros %d grupos.", "Showing the first %d groups."), result.RowCount)
	}
	answer.Tables, answer.Charts, err = datafunctions.RenderView(language, view, result, namesByColumn)
	return answer, err
}
