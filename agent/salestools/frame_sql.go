package salestools

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"app/core"
	"app/db"
	sales "app/sales/types"
	"genix-agent/harness"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
)

// ToolName is the sales FrameSQL tool as the model calls it.
const ToolName = "sales_frame_sql"

// saleFrames are the frames the tool reads; tests swap in frames over rows in memory.
var saleFrames = sales.SaleFrameSQLFrames

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
	Dataset string             `json:"dataset"`
	SQL     string             `json:"sql"`
	Message string             `json:"message"`
	Table   frameSQLTableInput `json:"table"`
	Chart   frameSQLChartInput `json:"chart"`
}

type frameSQLTableInput struct {
	Columns   []string `json:"columns"`
	SortBy    []string `json:"sort_by"`
	InlineBar string   `json:"inline_bar"`
}

type frameSQLChartInput struct {
	Type     string   `json:"type"`
	X        string   `json:"x"`
	Y        []string `json:"y"`
	Y2       []string `json:"y2"`
	SeriesBy string   `json:"series_by"`
}

// saleDatasets describe the sale frames for the model: what one row is and what its numbers count.
var saleDatasets = map[string]struct{ title, description string }{
	"sale-summary": {"Sales per day and product",
		"One row per product sold on a day: the amount sold, what is still unpaid, and the units sold and still undelivered. Cancelled sales are left out."},
	"sale-client-product": {"Sales per day, client and product",
		"One row per client and product on a day, with the same sums as sale_summary. Use it only to group or filter by client; sales without a client have client_id 0."},
	// The category views repeat a product's sales under each of its categories (framesql.View).
	"sale-category": {"Sales per day, product and category",
		"sale_summary with each row repeated under every category of its product (category_id 0: no category). Use it only to group or filter by category: a product in two categories counts in both, so the categories add up to more than the total sales; take totals from sale_summary."},
	"sale-client-category": {"Sales per day, client, product and category",
		"sale_client_product with each row repeated under every category of its product (category_id 0: no category). Use it only to group or filter by client and category together; the categories add up to more than the total sales."},
}

// frameSQLGrammar is the language as the model needs it (the parser's errors teach the rest).
const frameSQLGrammar = `SELECT item, ... WHERE condition AND ... [GROUP BY ...] [SORT BY item [DESC], ...] [LIMIT n]
- No FROM: the statement reads the dataset argument.
- An item is named after its column: SUM(amount) and AVG(amount) are amount, COUNT(*) is count, WEEK(date) is week, MONTH(date) is month; name any other expression with AS (SUM(amount) / COUNT(*) AS ticket). SORT BY, the table and the chart use these names; two items of one name need AS on one.
- WHERE must bound date: date IN PERIOD("<period>"), at most 400 days.
- product_id, client_id and category_id take = value or IN (value, ...). Write a record's name in quotes exactly as the user wrote it, never shortened or widened to find similar records (product_id IN ("name 1", "name 2")): the tool looks it up and asks the user when it is ambiguous.
- Names and periods go in double quotes: an apostrophe inside needs nothing ("O'Brien"), a double quote is written twice ("Pantalla 24""").
- SUM(col) adds a summed column. COUNT(*) counts the dataset's rows in each group (with a day key, the days a product has sales), never sales or units. AVG(col) is SUM(col) / COUNT(*). + - * / combine them. No MIN, MAX, HAVING, OR, joins or subqueries.
- GROUP BY date, product_id, client_id, category_id, WEEK(date) (its Monday) or MONTH(date) (its 1st). A SELECT column that is not an aggregate must be in GROUP BY.
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
	notation.WriteString("One SQL statement over the dataset, without FROM, that totals, ranks or averages its values by day, week, month, product, client or product category.\n")
	notation.WriteString("The datasets hold sums per day and row; a row whose sums are all 0 is not stored.\n")
	for _, frame := range frames {
		dataset := saleDatasets[frame.Name]
		datasetNames = append(datasetNames, frameSQLName(frame.Name))
		fmt.Fprintf(&notation, "- %s: %s. %s\n  Keys %s; Rows %s; Sums %s\n", frameSQLName(frame.Name), dataset.title, dataset.description,
			describeColumns(frame.Keys), describeColumns(frame.Rows), describeColumns(frame.Sums))
	}
	notation.WriteString(frameSQLGrammar)
	notation.WriteString("Example calls:\n" + frameSQLExamples)
	notation.WriteString("PERIOD() takes the period notation. " + periodNotation)

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
			kindLabel, _, _ = strings.Cut(recordCollections[column.Collection].noun, "|")
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
		ids, isOffered := pickedIDs(pending, answer)
		if !isResumeTarget || !hasAnswer || !isOffered {
			// The answer isn't one of the choices: ask the same question again.
			core.Log("agent.salestools choice not offered key::", pending.Key, " answer::", answer)
			return "", tool.StatefulInterrupt(ctx, harness.ChoiceQuestion{
				Question: localized(frameSQLTool.Language, "Elige una de las opciones.", "Pick one of the options."), Choices: pending.Choices,
			}, pending)
		}
		picks[pending.Key] = ids
	}

	answer, err := frameSQLTool.run(input, picks)
	var needsChoice *needsChoiceError
	if errors.As(err, &needsChoice) {
		return "", tool.StatefulInterrupt(ctx, needsChoice.question, pendingChoice{Picks: picks, Key: needsChoice.key, Choices: needsChoice.question.Choices})
	}
	if err != nil {
		return "", err
	}
	encodedAnswer, err := json.Marshal(answer)
	return string(encodedAnswer), err
}

func (frameSQLTool FrameSQLTool) run(input frameSQLInput, picks map[string][]int32) (Answer, error) {
	language := frameSQLTool.Language
	if strings.TrimSpace(input.Message) == "" {
		return Answer{}, errors.New("message is empty: write the sentence the user reads above the result")
	}
	statement, err := db.PrepareFrameSQL(input.Dataset, input.SQL, saleFrames(frameSQLTool.CompanyID), db.SQLOptions{MaxRows: MaxTableRows})
	if err != nil {
		return Answer{}, err
	}
	// The table and the chart are checked before the read: a name lookup may ask the user, and the
	// call must not fail once they answered.
	view, err := planFrameSQLView(statement.Columns(), input.Table, input.Chart)
	if err != nil {
		return Answer{}, err
	}
	result, err := statement.Run(db.SQLResolvers{
		Period: func(text string) (int64, int64, error) {
			fromDay, toDay, err := resolvePeriod(text, core.FechaUnix(), db.FrameMaxDays)
			return int64(fromDay), int64(toDay), err
		},
		Names: func(column db.SQLColumn, names []string) ([]int64, error) {
			ids, err := resolveNames(frameSQLTool.CompanyID, language, column.Collection, names, picks)
			ids64 := make([]int64, len(ids))
			for i, id := range ids {
				ids64[i] = int64(id)
			}
			return ids64, err
		},
	})
	if errors.Is(err, db.ErrFrameNotBuilt) {
		return Answer{}, errors.New("the dataset is being rebuilt after a change and can't be read for about 20 minutes: tell the user to try again later")
	}
	if err != nil {
		return Answer{}, err
	}
	summary := fmt.Sprintf("Ran FrameSQL on %s (%d rows): %s", input.Dataset, result.RowCount, input.SQL)
	if result.RowCount == 0 {
		return Answer{Message: input.Message + "\n\n" + localized(language, "La consulta no encontró datos.", "The query found no data."), Summary: summary}, nil
	}
	namesByColumn, err := frameSQLRecordNames(frameSQLTool.CompanyID, result)
	if err != nil {
		return Answer{}, fmt.Errorf("load the record names: %w", err)
	}

	answer := Answer{Message: input.Message, Summary: summary + frameSQLFirstRecords(result, namesByColumn)}
	// A statement without LIMIT stops at the table's rows: say so, or the user reads a partial total as all of it.
	if result.Truncated && result.RowCount == MaxTableRows {
		answer.Message += "\n\n" + fmt.Sprintf(localized(language, "Se muestran los primeros %d grupos.", "Showing the first %d groups."), result.RowCount)
	}
	if view.chart != nil {
		chart, err := frameSQLChart(language, *view.chart, result, namesByColumn)
		if err != nil {
			return Answer{}, err
		}
		answer.Charts = []Chart{chart}
	}
	if len(view.tableColumns) > 0 {
		answer.Tables = []Table{frameSQLTable(language, view, result, namesByColumn)}
	}
	return answer, nil
}

// frameSQLView is where the result's columns go: the table's and the chart's, as result column indexes.
type frameSQLView struct {
	// tableColumns is empty when no table shows.
	tableColumns []frameSQLColumnRef
	sortBy       []frameSQLSortKey
	// inlineBar is the position in tableColumns of the column drawn as bars, -1 for none.
	inlineBar int
	chart     *frameSQLChartView
}

// frameSQLColumnRef is a result column, and for a record column whether it shows the ID (column:id).
type frameSQLColumnRef struct {
	index   int
	showsID bool
}

type frameSQLSortKey struct {
	frameSQLColumnRef
	isDescending bool
}

type frameSQLChartView struct {
	// chartType is "bar" or "line", "" for the renderer's choice.
	chartType string
	x         int
	// seriesBy is -1 when the series are the y and y2 columns.
	seriesBy int
	y, y2    []int
}

// planFrameSQLView checks the table and the chart against the statement's columns.
func planFrameSQLView(columns []db.SQLResultColumn, tableInput frameSQLTableInput, chartInput frameSQLChartInput) (frameSQLView, error) {
	view := frameSQLView{inlineBar: -1}
	if chartInput.X != "" {
		chartView, err := planFrameSQLChart(columns, chartInput)
		if err != nil {
			return view, err
		}
		view.chart = &chartView
	}
	columnTexts := tableInput.Columns
	if len(columnTexts) == 0 {
		if view.chart != nil {
			return view, nil
		}
		for _, column := range columns {
			columnTexts = append(columnTexts, column.Name)
		}
	}
	if len(columnTexts) > MaxTableColumns {
		return view, fmt.Errorf("table.columns: %d columns, at most %d", len(columnTexts), MaxTableColumns)
	}
	for _, text := range columnTexts {
		ref, err := frameSQLColumnRefOf(columns, "table.columns", text)
		if err != nil {
			return view, err
		}
		view.tableColumns = append(view.tableColumns, ref)
	}
	for _, text := range tableInput.SortBy {
		words := strings.Fields(text)
		isDescending := len(words) == 2 && strings.EqualFold(words[1], "DESC")
		if len(words) == 0 || len(words) > 2 || (len(words) == 2 && !isDescending && !strings.EqualFold(words[1], "ASC")) {
			return view, fmt.Errorf("table.sort_by: write \"column\" or \"column DESC\", not %q", text)
		}
		ref, err := frameSQLColumnRefOf(columns, "table.sort_by", words[0])
		if err != nil {
			return view, err
		}
		view.sortBy = append(view.sortBy, frameSQLSortKey{frameSQLColumnRef: ref, isDescending: isDescending})
	}
	if tableInput.InlineBar != "" {
		index, err := frameSQLColumnIndex(columns, "table.inline_bar", tableInput.InlineBar)
		if err != nil {
			return view, err
		}
		view.inlineBar = slices.IndexFunc(view.tableColumns, func(ref frameSQLColumnRef) bool { return ref.index == index })
		if view.inlineBar < 0 || columns[index].IsGroup {
			return view, fmt.Errorf("table.inline_bar: `%s` must be a value column of table.columns", tableInput.InlineBar)
		}
	}
	return view, nil
}

func planFrameSQLChart(columns []db.SQLResultColumn, chartInput frameSQLChartInput) (frameSQLChartView, error) {
	chartView := frameSQLChartView{seriesBy: -1}
	switch chartInput.Type {
	case "bar", "line":
		chartView.chartType = chartInput.Type
	case "auto", "":
	default:
		return chartView, fmt.Errorf("chart.type is auto, bar or line, not %q", chartInput.Type)
	}
	var err error
	if chartView.x, err = frameSQLColumnIndex(columns, "chart.x", chartInput.X); err != nil {
		return chartView, err
	}
	if !columns[chartView.x].IsGroup {
		return chartView, fmt.Errorf("chart.x: `%s` is not a GROUP BY column", chartInput.X)
	}
	if chartInput.SeriesBy != "" {
		if chartView.seriesBy, err = frameSQLColumnIndex(columns, "chart.series_by", chartInput.SeriesBy); err != nil {
			return chartView, err
		}
		if !columns[chartView.seriesBy].IsGroup || chartView.seriesBy == chartView.x {
			return chartView, fmt.Errorf("chart.series_by: `%s` must be a GROUP BY column other than x", chartInput.SeriesBy)
		}
		if len(chartInput.Y) != 1 || len(chartInput.Y2) > 0 {
			return chartView, errors.New("chart.series_by takes exactly one y and no y2")
		}
	}
	for _, axis := range []struct {
		argument string
		names    []string
		indexes  *[]int
	}{{"chart.y", chartInput.Y, &chartView.y}, {"chart.y2", chartInput.Y2, &chartView.y2}} {
		for _, name := range axis.names {
			index, err := frameSQLColumnIndex(columns, axis.argument, name)
			if err != nil {
				return chartView, err
			}
			if columns[index].IsGroup {
				return chartView, fmt.Errorf("%s: `%s` is a GROUP BY column, not a value", axis.argument, name)
			}
			*axis.indexes = append(*axis.indexes, index)
		}
	}
	if len(chartView.y) == 0 {
		return chartView, errors.New("chart.y: name the value columns to draw")
	}
	// The bars of one scale draw stacked: money beside units there would read as one sum.
	moneyColumnCount := 0
	for _, index := range chartView.y {
		if columns[index].Kind == db.SQLCents {
			moneyColumnCount++
		}
	}
	if moneyColumnCount > 0 && moneyColumnCount < len(chartView.y) {
		return chartView, errors.New("chart.y: money and other numbers can't share a scale, the bars would stack them as one sum: put the non-money columns in y2")
	}
	if seriesCount := len(chartView.y) + len(chartView.y2); seriesCount > MaxChartSeries {
		return chartView, fmt.Errorf("chart: %d series, at most %d", seriesCount, MaxChartSeries)
	}
	// Any other group would give x several points.
	for i, column := range columns {
		if column.IsGroup && i != chartView.x && i != chartView.seriesBy {
			return chartView, fmt.Errorf("chart: the result has a row per `%s` too: set series_by to it, or leave it out of GROUP BY", column.Name)
		}
	}
	return chartView, nil
}

// frameSQLColumnIndex finds a result column by its name, in any case.
func frameSQLColumnIndex(columns []db.SQLResultColumn, argument, name string) (int, error) {
	index := slices.IndexFunc(columns, func(column db.SQLResultColumn) bool { return strings.EqualFold(column.Name, strings.TrimSpace(name)) })
	if index < 0 {
		columnNames := make([]string, len(columns))
		for i, column := range columns {
			columnNames[i] = column.Name
		}
		return -1, fmt.Errorf("%s: the result has no column `%s`; its columns are: %s", argument, name, strings.Join(columnNames, ", "))
	}
	return index, nil
}

// frameSQLColumnRefOf reads "column" or, on a record column, "column:id".
func frameSQLColumnRefOf(columns []db.SQLResultColumn, argument, text string) (frameSQLColumnRef, error) {
	name, suffix, hasSuffix := strings.Cut(text, ":")
	index, err := frameSQLColumnIndex(columns, argument, name)
	if err != nil {
		return frameSQLColumnRef{}, err
	}
	if hasSuffix && (!strings.EqualFold(strings.TrimSpace(suffix), "id") || columns[index].Kind != db.SQLRef) {
		return frameSQLColumnRef{}, fmt.Errorf("%s: `%s`: only a record column takes :id", argument, text)
	}
	return frameSQLColumnRef{index: index, showsID: hasSuffix}, nil
}

// frameSQLRecordNames names the IDs of each record column of the result (nil for the other columns).
func frameSQLRecordNames(companyID int32, result db.SQLResult) ([]map[int32]string, error) {
	namesByColumn := make([]map[int32]string, len(result.Columns))
	for i, column := range result.Columns {
		collection, isKnown := recordCollections[column.Collection]
		if column.Kind != db.SQLRef || !isKnown {
			continue
		}
		ids := []int32{}
		for _, value := range column.Values {
			if id := int32(value); id != 0 && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			continue
		}
		names, err := collection.namesByID(companyID, ids)
		if err != nil {
			return nil, err
		}
		namesByColumn[i] = names
	}
	return namesByColumn, nil
}

// frameSQLHeader is a column's header in the user's language: the frame column's label
// ("Amount|Monto"), what a week, a month or a count is, else the name as written (an expression).
func frameSQLHeader(language string, column db.SQLResultColumn) string {
	englishLabel, spanishLabel, isBilingual := strings.Cut(column.Label, "|")
	label := column.Label
	if isBilingual {
		label = localized(language, spanishLabel, englishLabel)
	}
	switch {
	case column.Period == "week":
		return localized(language, "Semana", "Week")
	case column.Period == "month":
		return localized(language, "Mes", "Month")
	case column.Aggregate == "COUNT":
		return localized(language, "Cantidad", "Count")
	case label == "":
		return column.Name
	case column.Aggregate == "AVG":
		return localized(language, label+" promedio", "Average "+label)
	}
	return label
}

// frameSQLCell is one value as the table and the chart show it: a day as its date, a record by name
// (column:id by ID; ID 0 is a sale without one), any other group as text (a code, not a quantity),
// money in currency units, and nil for no value (a division by zero).
func frameSQLCell(language string, column db.SQLResultColumn, value float64, names map[int32]string, showsID bool) any {
	switch {
	case math.IsNaN(value):
		return nil
	case column.Kind == db.SQLDay:
		return formatUnixDay(int64(value))
	case column.Kind == db.SQLRef && !showsID && value == 0:
		return localized(language, "(ninguno)", "(none)")
	case column.Kind == db.SQLRef && !showsID:
		return cmp.Or(names[int32(value)], fmt.Sprintf("#%d", int64(value)))
	case column.IsGroup || column.Kind == db.SQLRef:
		return strconv.FormatInt(int64(value), 10)
	case column.Kind == db.SQLCents:
		return value / 100
	}
	return value
}

// frameSQLFormat is how the renderer formats a column's cells (frameSQLCell).
func frameSQLFormat(column db.SQLResultColumn) string {
	switch {
	case column.Kind == db.SQLDay:
		return "date"
	case column.IsGroup || column.Kind == db.SQLRef:
		return "text"
	case column.Kind == db.SQLCents:
		return "money"
	}
	return "number"
}

// frameSQLTable lays out the table: the result's rows in sort_by's order, then the statement's.
func frameSQLTable(language string, view frameSQLView, result db.SQLResult, namesByColumn []map[int32]string) Table {
	// A record sorts by its name, anything else by its number; no value sorts last either way.
	sortValue := func(key frameSQLSortKey, row int) any {
		column := result.Columns[key.index]
		if column.Kind == db.SQLRef && !key.showsID {
			return frameSQLCell(language, column, column.Values[row], namesByColumn[key.index], false)
		}
		if math.IsNaN(column.Values[row]) {
			return nil
		}
		return column.Values[row]
	}
	rowOrder := make([]int, result.RowCount)
	for row := range rowOrder {
		rowOrder[row] = row
	}
	slices.SortStableFunc(rowOrder, func(a, b int) int {
		for _, key := range view.sortBy {
			if order := compareSortValues(sortValue(key, a), sortValue(key, b), key.isDescending); order != 0 {
				return order
			}
		}
		return 0
	})

	table := Table{}
	for position, ref := range view.tableColumns {
		column := result.Columns[ref.index]
		tableColumn := TableColumn{Label: frameSQLHeader(language, column), Format: frameSQLFormat(column), InlineBar: position == view.inlineBar}
		if ref.showsID {
			tableColumn.Label += " ID"
		}
		for _, row := range rowOrder {
			tableColumn.Values = append(tableColumn.Values, frameSQLCell(language, column, column.Values[row], namesByColumn[ref.index], ref.showsID))
		}
		table.Columns = append(table.Columns, tableColumn)
	}
	return table
}

func compareSortValues(a, b any, isDescending bool) int {
	switch {
	case a == nil || b == nil:
		return cmp.Compare(boolRank(a == nil), boolRank(b == nil))
	case isDescending:
		a, b = b, a
	}
	if aText, isText := a.(string); isText {
		return strings.Compare(strings.ToLower(aText), strings.ToLower(b.(string)))
	}
	return cmp.Compare(a.(float64), b.(float64))
}

func boolRank(isTrue bool) int {
	if isTrue {
		return 1
	}
	return 0
}

// frameSQLChart draws the result. The points are x's values: days ascending, and every period of the
// read when the result holds every group; records in the result's order (a ranking). A point
// without a row is 0 on a SUM or a COUNT, no value otherwise.
func frameSQLChart(language string, chartView frameSQLChartView, result db.SQLResult, namesByColumn []map[int32]string) (Chart, error) {
	xColumn := result.Columns[chartView.x]
	rowByPoint := map[[2]float64]int{}
	var xValues, seriesValues []float64
	for row := range result.RowCount {
		point := [2]float64{xColumn.Values[row]}
		if chartView.seriesBy >= 0 {
			point[1] = result.Columns[chartView.seriesBy].Values[row]
		}
		// A group the statement didn't SELECT gives x several rows.
		if _, isRepeated := rowByPoint[point]; isRepeated {
			return Chart{}, fmt.Errorf("chart: `%s` has several rows per point: GROUP BY only the chart's x and series_by", xColumn.Name)
		}
		rowByPoint[point] = row
		if !slices.Contains(xValues, point[0]) {
			xValues = append(xValues, point[0])
		}
		if chartView.seriesBy >= 0 && !slices.Contains(seriesValues, point[1]) {
			seriesValues = append(seriesValues, point[1])
		}
	}
	if xColumn.Kind == db.SQLDay {
		slices.Sort(xValues)
		if xColumn.Period != "" && !result.Truncated {
			xValues = xValues[:0]
			for _, periodStart := range db.FramePeriodStarts(xColumn.Period, result.FromKey, result.ToKey) {
				xValues = append(xValues, float64(periodStart))
			}
		}
	}

	xChartColumn := ChartColumn{Label: frameSQLHeader(language, xColumn), Axis: "x"}
	if xColumn.Kind == db.SQLDay {
		xChartColumn.Format = "date"
	}
	for _, xValue := range xValues {
		xChartColumn.Values = append(xChartColumn.Values, frameSQLCell(language, xColumn, xValue, namesByColumn[chartView.x], false))
	}
	chart := Chart{Columns: []ChartColumn{xChartColumn}}
	addSeries := func(label, axis string, valueColumn db.SQLResultColumn, seriesValue float64) {
		series := ChartColumn{Label: label, Axis: axis, Type: chartView.chartType, Format: "number"}
		if valueColumn.Kind == db.SQLCents {
			series.Format = "money"
		}
		for _, xValue := range xValues {
			row, hasRow := rowByPoint[[2]float64{xValue, seriesValue}]
			switch {
			case hasRow:
				series.Values = append(series.Values, frameSQLCell(language, valueColumn, valueColumn.Values[row], nil, false))
			case valueColumn.Aggregate == "SUM" || valueColumn.Aggregate == "COUNT":
				series.Values = append(series.Values, 0.0)
			default:
				series.Values = append(series.Values, nil)
			}
		}
		chart.Columns = append(chart.Columns, series)
	}

	if chartView.seriesBy < 0 {
		for _, index := range chartView.y {
			addSeries(frameSQLHeader(language, result.Columns[index]), "y", result.Columns[index], 0)
		}
		for _, index := range chartView.y2 {
			addSeries(frameSQLHeader(language, result.Columns[index]), "y2", result.Columns[index], 0)
		}
		return chart, nil
	}
	seriesColumn := result.Columns[chartView.seriesBy]
	if len(seriesValues) > MaxChartSeries {
		return Chart{}, fmt.Errorf("chart.series_by: `%s` has %d values, at most %d: filter it in WHERE", seriesColumn.Name, len(seriesValues), MaxChartSeries)
	}
	for _, seriesValue := range seriesValues {
		seriesLabel := fmt.Sprint(frameSQLCell(language, seriesColumn, seriesValue, namesByColumn[chartView.seriesBy], false))
		addSeries(seriesLabel, "y", result.Columns[chartView.y[0]], seriesValue)
	}
	return chart, nil
}

// frameSQLFirstRecords is the summary's tail: the first records of the result, so a follow-up can
// name them.
func frameSQLFirstRecords(result db.SQLResult, namesByColumn []map[int32]string) string {
	for i, column := range result.Columns {
		if column.Kind != db.SQLRef {
			continue
		}
		labels := []string{}
		for _, value := range column.Values[:min(result.RowCount, 10)] {
			labels = append(labels, fmt.Sprintf("[%d] %s", int32(value), frameSQLCell("en", column, value, namesByColumn[i], false)))
		}
		return "; first " + column.Name + ": " + strings.Join(labels, ", ")
	}
	return ""
}
