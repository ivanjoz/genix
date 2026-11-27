package datafunctions

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"app/db"
)

// A FrameSQL result becomes an answer's table and chart through a view: which result columns the
// table shows and in which order, and which ones the chart draws. The sales tool takes the view from
// the model's call; a data function declares it.

// TableInput is the table of a view, by result column names.
type TableInput struct {
	Columns   []string `json:"columns"`
	SortBy    []string `json:"sort_by"`
	InlineBar string   `json:"inline_bar"`
}

// ChartInput is the chart of a view, by result column names; X empty for none.
type ChartInput struct {
	Type     string   `json:"type"`
	X        string   `json:"x"`
	Y        []string `json:"y"`
	Y2       []string `json:"y2"`
	SeriesBy string   `json:"series_by"`
}

// View is where the result's columns go: the table's and the chart's, as result column indexes.
type View struct {
	// tableColumns is empty when no table shows.
	tableColumns []columnRef
	sortBy       []sortKey
	// inlineBar is the position in tableColumns of the column drawn as bars, -1 for none.
	inlineBar int
	chart     *chartView
}

// columnRef is a result column, and for a record column whether it shows the ID (column:id).
type columnRef struct {
	index   int
	showsID bool
}

type sortKey struct {
	columnRef
	isDescending bool
}

type chartView struct {
	// chartType is "bar" or "line", "" for the renderer's choice.
	chartType string
	x         int
	// seriesBy is -1 when the series are the y and y2 columns.
	seriesBy int
	y, y2    []int
}

// PlanView checks the table and the chart against the statement's columns. Without a chart the table
// always shows, with every column when Columns is empty; with a chart, only when Columns lists some.
func PlanView(columns []db.SQLResultColumn, tableInput TableInput, chartInput ChartInput) (View, error) {
	view := View{inlineBar: -1}
	if chartInput.X != "" {
		chartView, err := planChart(columns, chartInput)
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
		ref, err := columnRefOf(columns, "table.columns", text)
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
		ref, err := columnRefOf(columns, "table.sort_by", words[0])
		if err != nil {
			return view, err
		}
		view.sortBy = append(view.sortBy, sortKey{columnRef: ref, isDescending: isDescending})
	}
	if tableInput.InlineBar != "" {
		index, err := columnIndex(columns, "table.inline_bar", tableInput.InlineBar)
		if err != nil {
			return view, err
		}
		view.inlineBar = slices.IndexFunc(view.tableColumns, func(ref columnRef) bool { return ref.index == index })
		if view.inlineBar < 0 || columns[index].IsGroup {
			return view, fmt.Errorf("table.inline_bar: `%s` must be a value column of table.columns", tableInput.InlineBar)
		}
	}
	return view, nil
}

func planChart(columns []db.SQLResultColumn, chartInput ChartInput) (chartView, error) {
	planned := chartView{seriesBy: -1}
	switch chartInput.Type {
	case "bar", "line":
		planned.chartType = chartInput.Type
	case "auto", "":
	default:
		return planned, fmt.Errorf("chart.type is auto, bar or line, not %q", chartInput.Type)
	}
	var err error
	if planned.x, err = columnIndex(columns, "chart.x", chartInput.X); err != nil {
		return planned, err
	}
	if !columns[planned.x].IsGroup {
		return planned, fmt.Errorf("chart.x: `%s` is not a GROUP BY column", chartInput.X)
	}
	if chartInput.SeriesBy != "" {
		if planned.seriesBy, err = columnIndex(columns, "chart.series_by", chartInput.SeriesBy); err != nil {
			return planned, err
		}
		if !columns[planned.seriesBy].IsGroup || planned.seriesBy == planned.x {
			return planned, fmt.Errorf("chart.series_by: `%s` must be a GROUP BY column other than x", chartInput.SeriesBy)
		}
		if len(chartInput.Y) != 1 || len(chartInput.Y2) > 0 {
			return planned, errors.New("chart.series_by takes exactly one y and no y2")
		}
	}
	for _, axis := range []struct {
		argument string
		names    []string
		indexes  *[]int
	}{{"chart.y", chartInput.Y, &planned.y}, {"chart.y2", chartInput.Y2, &planned.y2}} {
		for _, name := range axis.names {
			index, err := columnIndex(columns, axis.argument, name)
			if err != nil {
				return planned, err
			}
			if columns[index].IsGroup {
				return planned, fmt.Errorf("%s: `%s` is a GROUP BY column, not a value", axis.argument, name)
			}
			*axis.indexes = append(*axis.indexes, index)
		}
	}
	if len(planned.y) == 0 {
		return planned, errors.New("chart.y: name the value columns to draw")
	}
	// The bars of one scale draw stacked: money beside units there would read as one sum.
	moneyColumnCount := 0
	for _, index := range planned.y {
		if columns[index].Kind == db.SQLCents {
			moneyColumnCount++
		}
	}
	if moneyColumnCount > 0 && moneyColumnCount < len(planned.y) {
		return planned, errors.New("chart.y: money and other numbers can't share a scale, the bars would stack them as one sum: put the non-money columns in y2")
	}
	if seriesCount := len(planned.y) + len(planned.y2); seriesCount > MaxChartSeries {
		return planned, fmt.Errorf("chart: %d series, at most %d", seriesCount, MaxChartSeries)
	}
	// Any other group would give x several points.
	for i, column := range columns {
		if column.IsGroup && i != planned.x && i != planned.seriesBy {
			return planned, fmt.Errorf("chart: the result has a row per `%s` too: set series_by to it, or leave it out of GROUP BY", column.Name)
		}
	}
	return planned, nil
}

// columnIndex finds a result column by its name, in any case.
func columnIndex(columns []db.SQLResultColumn, argument, name string) (int, error) {
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

// columnRefOf reads "column" or, on a record column, "column:id".
func columnRefOf(columns []db.SQLResultColumn, argument, text string) (columnRef, error) {
	name, suffix, hasSuffix := strings.Cut(text, ":")
	index, err := columnIndex(columns, argument, name)
	if err != nil {
		return columnRef{}, err
	}
	if hasSuffix && (!strings.EqualFold(strings.TrimSpace(suffix), "id") || columns[index].Kind != db.SQLRef) {
		return columnRef{}, fmt.Errorf("%s: `%s`: only a record column takes :id", argument, text)
	}
	return columnRef{index: index, showsID: hasSuffix}, nil
}

// RecordNames names the IDs of each record column of the result (nil for the other columns).
func RecordNames(collections map[string]RecordCollection, companyID int32, result db.SQLResult) ([]map[int32]string, error) {
	namesByColumn := make([]map[int32]string, len(result.Columns))
	for i, column := range result.Columns {
		collection, isKnown := collections[column.Collection]
		if column.Kind != db.SQLRef || !isKnown {
			continue
		}
		ids := []int32{}
		for _, value := range column.Values {
			if id := int32(value); id > 0 && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			continue
		}
		names, err := collection.NamesByID(companyID, ids)
		if err != nil {
			return nil, err
		}
		namesByColumn[i] = names
	}
	return namesByColumn, nil
}

// RenderView lays the result out as the view's table and chart.
func RenderView(language string, view View, result db.SQLResult, namesByColumn []map[int32]string) ([]Table, []Chart, error) {
	var tables []Table
	var charts []Chart
	if view.chart != nil {
		chart, err := buildChart(language, *view.chart, result, namesByColumn)
		if err != nil {
			return nil, nil, err
		}
		charts = []Chart{chart}
	}
	if len(view.tableColumns) > 0 {
		tables = []Table{buildTable(language, view, result, namesByColumn)}
	}
	return tables, charts, nil
}

// columnHeader is a column's header in the user's language: the frame column's label
// ("Amount|Monto"), what a week, a month or a count is, else the name as written (an expression).
func columnHeader(language string, column db.SQLResultColumn) string {
	englishLabel, spanishLabel, isBilingual := strings.Cut(column.Label, "|")
	label := column.Label
	if isBilingual {
		label = Localized(language, spanishLabel, englishLabel)
	}
	switch {
	case column.Period == "week":
		return Localized(language, "Semana", "Week")
	case column.Period == "month":
		return Localized(language, "Mes", "Month")
	case column.Aggregate == "COUNT":
		return Localized(language, "Cantidad", "Count")
	case label == "":
		return column.Name
	case column.Aggregate == "AVG":
		return Localized(language, label+" promedio", "Average "+label)
	}
	return label
}

// cellValue is one value as the table and the chart show it: a day as its date, a record by name
// (column:id by ID; ID 0 is a sale without one), any other group as text (a code, not a quantity),
// money in currency units, and nil for no value (a division by zero).
func cellValue(language string, column db.SQLResultColumn, value float64, names map[int32]string, showsID bool) any {
	switch {
	case math.IsNaN(value):
		return nil
	case column.Kind == db.SQLDay:
		return FormatUnixDay(int64(value))
	case column.Kind == db.SQLRef && !showsID && value == 0:
		return Localized(language, "(ninguno)", "(none)")
	case column.Kind == db.SQLRef && !showsID:
		return cmp.Or(names[int32(value)], fmt.Sprintf("#%d", int64(value)))
	case column.IsGroup || column.Kind == db.SQLRef:
		return strconv.FormatInt(int64(value), 10)
	case column.Kind == db.SQLCents:
		return value / 100
	}
	return value
}

// cellFormat is how the renderer formats a column's cells (cellValue).
func cellFormat(column db.SQLResultColumn) string {
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

// buildTable lays out the table: the result's rows in sort_by's order, then the statement's.
func buildTable(language string, view View, result db.SQLResult, namesByColumn []map[int32]string) Table {
	// A record sorts by its name, anything else by its number; no value sorts last either way.
	sortValue := func(key sortKey, row int) any {
		column := result.Columns[key.index]
		if column.Kind == db.SQLRef && !key.showsID {
			return cellValue(language, column, column.Values[row], namesByColumn[key.index], false)
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
		tableColumn := TableColumn{Label: columnHeader(language, column), Format: cellFormat(column), InlineBar: position == view.inlineBar}
		if ref.showsID {
			tableColumn.Label += " ID"
		}
		for _, row := range rowOrder {
			tableColumn.Values = append(tableColumn.Values, cellValue(language, column, column.Values[row], namesByColumn[ref.index], ref.showsID))
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

// buildChart draws the result. The points are x's values: days ascending, and every period of the
// read when the result holds every group; records in the result's order (a ranking). A point
// without a row is 0 on a SUM or a COUNT, no value otherwise.
func buildChart(language string, planned chartView, result db.SQLResult, namesByColumn []map[int32]string) (Chart, error) {
	xColumn := result.Columns[planned.x]
	rowByPoint := map[[2]float64]int{}
	var xValues, seriesValues []float64
	for row := range result.RowCount {
		point := [2]float64{xColumn.Values[row]}
		if planned.seriesBy >= 0 {
			point[1] = result.Columns[planned.seriesBy].Values[row]
		}
		// A group the statement didn't SELECT gives x several rows.
		if _, isRepeated := rowByPoint[point]; isRepeated {
			return Chart{}, fmt.Errorf("chart: `%s` has several rows per point: GROUP BY only the chart's x and series_by", xColumn.Name)
		}
		rowByPoint[point] = row
		if !slices.Contains(xValues, point[0]) {
			xValues = append(xValues, point[0])
		}
		if planned.seriesBy >= 0 && !slices.Contains(seriesValues, point[1]) {
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

	xChartColumn := ChartColumn{Label: columnHeader(language, xColumn), Axis: "x"}
	if xColumn.Kind == db.SQLDay {
		xChartColumn.Format = "date"
	}
	for _, xValue := range xValues {
		xChartColumn.Values = append(xChartColumn.Values, cellValue(language, xColumn, xValue, namesByColumn[planned.x], false))
	}
	chart := Chart{Columns: []ChartColumn{xChartColumn}}
	addSeries := func(label, axis string, valueColumn db.SQLResultColumn, seriesValue float64) {
		series := ChartColumn{Label: label, Axis: axis, Type: planned.chartType, Format: "number"}
		if valueColumn.Kind == db.SQLCents {
			series.Format = "money"
		}
		for _, xValue := range xValues {
			row, hasRow := rowByPoint[[2]float64{xValue, seriesValue}]
			switch {
			case hasRow:
				series.Values = append(series.Values, cellValue(language, valueColumn, valueColumn.Values[row], nil, false))
			case valueColumn.Aggregate == "SUM" || valueColumn.Aggregate == "COUNT":
				series.Values = append(series.Values, 0.0)
			default:
				series.Values = append(series.Values, nil)
			}
		}
		chart.Columns = append(chart.Columns, series)
	}

	if planned.seriesBy < 0 {
		for _, index := range planned.y {
			addSeries(columnHeader(language, result.Columns[index]), "y", result.Columns[index], 0)
		}
		for _, index := range planned.y2 {
			addSeries(columnHeader(language, result.Columns[index]), "y2", result.Columns[index], 0)
		}
		return chart, nil
	}
	seriesColumn := result.Columns[planned.seriesBy]
	if len(seriesValues) > MaxChartSeries {
		return Chart{}, fmt.Errorf("chart.series_by: `%s` has %d values, at most %d: filter it in WHERE", seriesColumn.Name, len(seriesValues), MaxChartSeries)
	}
	for _, seriesValue := range seriesValues {
		seriesLabel := fmt.Sprint(cellValue(language, seriesColumn, seriesValue, namesByColumn[planned.seriesBy], false))
		addSeries(seriesLabel, "y", result.Columns[planned.y[0]], seriesValue)
	}
	return chart, nil
}

// FirstRecords is a summary's tail: the first records of the result, so a follow-up can name them.
func FirstRecords(result db.SQLResult, namesByColumn []map[int32]string) string {
	for i, column := range result.Columns {
		if column.Kind != db.SQLRef {
			continue
		}
		labels := []string{}
		for _, value := range column.Values[:min(result.RowCount, 10)] {
			labels = append(labels, fmt.Sprintf("[%d] %s", int32(value), cellValue("en", column, value, namesByColumn[i], false)))
		}
		return "; first " + column.Name + ": " + strings.Join(labels, ", ")
	}
	return ""
}
