package datafunctions

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"app/db"
)

// The shapes the sales functions share: a total, a line over time, a ranking, a line per record, and
// the ones FrameSQL can't express alone, computed in Go over its reads (an average over calendar days,
// weekday averages, shares, comparisons, records that stopped selling).

// emptyMessages say what an empty answer lacks, by the measure it reads.
var emptyMessages = map[string]string{
	"amount":           noSalesMessage,
	"debt_amount":      "No hay montos por cobrar de las ventas de ese periodo.",
	"pending_quantity": "No hay unidades por entregar de las ventas de ese periodo.",
}

// timeGroupNames are the result column names of the time groups.
var timeGroupNames = map[string]string{"date": "date", "WEEK(date)": "week", "MONTH(date)": "month"}

// total answers with the measure's sum alone (V). Rule Q: the units of a sale total only make sense
// for the products named.
func (call *functionCall) total(measure string) (shownResult, error) {
	selects := "SUM(" + measure + ")"
	hasUnits := measure == "amount" && len(call.recordIDs["products"]) > 0
	if hasUnits {
		selects += ", SUM(quantity)"
	}
	result, err := call.read(frameRead{Select: selects})
	if err != nil {
		return shownResult{}, err
	}
	totalValue := result.Columns[0].Values[0]
	value := formatMeasure(measure, totalValue)
	if hasUnits {
		value += " y " + formatQuantity(result.Columns[1].Values[0]) + " unidades"
	}
	return shownResult{result: result, isValueOnly: true, value: value, isEmpty: totalValue == 0, emptyMessage: emptyMessages[measure]}, nil
}

// timeline is one series of the measure per day, week or month, every period of the read drawn.
func (call *functionCall) timeline(timeGroup, measure, chartType string) (shownResult, error) {
	result, err := call.read(frameRead{Select: timeGroup + ", SUM(" + measure + ")", GroupBy: timeGroup})
	if err != nil {
		return shownResult{}, err
	}
	totalValue := columnSum(result, 1)
	return shownResult{result: result, chart: ChartInput{Type: chartType, X: timeGroupNames[timeGroup], Y: []string{measure}},
		value: formatMeasure(measure, totalValue), isEmpty: totalValue == 0, emptyMessage: emptyMessages[measure]}, nil
}

// timelinePerRecord is one series per record (at most 12) of the measure per day, week or month.
func (call *functionCall) timelinePerRecord(timeGroup, recordColumn, measure, chartType string) (shownResult, error) {
	result, err := call.read(frameRead{Select: timeGroup + ", " + recordColumn + ", SUM(" + measure + ")", GroupBy: timeGroup + ", " + recordColumn})
	if err != nil {
		return shownResult{}, err
	}
	totalValue := columnSum(result, 2)
	return shownResult{result: result, chart: ChartInput{Type: chartType, X: timeGroupNames[timeGroup], Y: []string{measure}, SeriesBy: recordColumn},
		isEmpty: totalValue == 0, emptyMessage: emptyMessages[measure]}, nil
}

// ranking lists the records by sortMeasure, largest first, with an inline bar. What is unpaid or
// undelivered leaves out the records that owe nothing.
func (call *functionCall) ranking(recordColumns string, measures []string, sortMeasure string, limit int) (shownResult, error) {
	selects := recordColumns
	for _, measure := range measures {
		selects += ", SUM(" + measure + ")"
	}
	result, err := call.read(frameRead{Select: selects, GroupBy: recordColumns, SortBy: sortMeasure + " DESC", Limit: limit})
	if err != nil {
		return shownResult{}, err
	}
	sortIndex := slices.IndexFunc(result.Columns, func(column db.SQLResultColumn) bool { return column.Name == sortMeasure })
	if sortMeasure != "amount" && sortMeasure != "quantity" {
		result = keepRows(result, func(row int) bool { return result.Columns[sortIndex].Values[row] != 0 })
	}
	return shownResult{result: result, table: TableInput{Columns: append(strings.Split(recordColumns, ", "), measures...), InlineBar: sortMeasure},
		value: formatMeasure(sortMeasure, columnSum(result, sortIndex)), emptyMessage: emptyMessages[sortMeasure]}, nil
}

// asBars shows a ranking of one record column as bars instead of a table.
func asBars(shown shownResult, recordColumn, measure string) shownResult {
	shown.table, shown.chart = TableInput{}, ChartInput{Type: "bar", X: recordColumn, Y: []string{measure}}
	return shown
}

// averageDaily divides the period's sales by its calendar days: FrameSQL's AVG would skip the days
// without sales.
func (call *functionCall) averageDaily() (shownResult, error) {
	result, err := call.read(frameRead{Select: "SUM(amount)"})
	if err != nil {
		return shownResult{}, err
	}
	totalAmount := result.Columns[0].Values[0]
	days := float64(call.period.to - call.period.from + 1)
	average := math.Round(totalAmount / days)
	shown := oneRow(
		valueColumn("average", "Promedio diario", db.SQLCents, average),
		valueColumn("amount", "Total", db.SQLCents, totalAmount),
		valueColumn("days", "Días", db.SQLInteger, days),
	)
	return shownResult{result: shown, isValueOnly: true, isEmpty: totalAmount == 0,
		value: fmt.Sprintf("%s por día (%s en %d días)", formatSoles(average), formatSoles(totalAmount), int(days))}, nil
}

// spanishWeekdays are the weekdays from Monday, as weekdayAverages numbers them from 1.
var spanishWeekdays = []string{"lunes", "martes", "miércoles", "jueves", "viernes", "sábado", "domingo"}

// weekdayAverages averages the sales of each weekday over every such day of the period, the ones
// without sales included.
func (call *functionCall) weekdayAverages() (shownResult, error) {
	result, err := call.read(frameRead{Select: "date, SUM(amount)", GroupBy: "date"})
	if err != nil {
		return shownResult{}, err
	}
	amountByDay := map[int16]float64{}
	for row := range result.RowCount {
		amountByDay[int16(result.Columns[0].Values[row])] = result.Columns[1].Values[row]
	}
	var amountSums, dayCounts [7]float64
	for day := call.period.from; day <= call.period.to; day++ {
		// UnixDay 0 is a Thursday: Monday is 0.
		weekday := (int(day) + 3) % 7
		dayCounts[weekday]++
		amountSums[weekday] += amountByDay[day]
	}
	weekdays, averages := []float64{}, []float64{}
	weekdayNames := map[int32]string{}
	for weekday, dayCount := range dayCounts {
		if dayCount > 0 {
			weekdays = append(weekdays, float64(weekday+1))
			averages = append(averages, math.Round(amountSums[weekday]/dayCount))
			weekdayNames[int32(weekday+1)] = spanishWeekdays[weekday]
		}
	}
	shown := oneRow(groupColumn("weekday", "Día", weekdays), valueColumn("average", "Promedio", db.SQLCents, averages...))
	return shownResult{result: shown, chart: ChartInput{Type: "bar", X: "weekday", Y: []string{"average"}},
		isEmpty: columnSum(result, 1) == 0, extraNames: map[int]map[int32]string{0: weekdayNames}}, nil
}

// otherRecordsID is the "Otros" row of a share: every record past the top.
const otherRecordsID = -1

// share is each record's part of the total sales. The total is its own read: a ranking stops at its
// limit, and the rest shows as "Otros".
func (call *functionCall) share(recordColumn string, limit int) (shownResult, error) {
	ranked, err := call.read(frameRead{Select: recordColumn + ", SUM(amount)", GroupBy: recordColumn, SortBy: "amount DESC", Limit: limit})
	if err != nil {
		return shownResult{}, err
	}
	totalResult, err := call.read(frameRead{Select: "SUM(amount)"})
	if err != nil {
		return shownResult{}, err
	}
	totalAmount := totalResult.Columns[0].Values[0]
	recordIDs, amounts := slices.Clone(ranked.Columns[0].Values), slices.Clone(ranked.Columns[1].Values)
	if rest := totalAmount - columnSum(ranked, 1); rest >= 1 {
		recordIDs, amounts = append(recordIDs, otherRecordsID), append(amounts, rest)
	}
	shares := make([]float64, len(amounts))
	for i, amount := range amounts {
		shares[i] = math.Round(amount/totalAmount*1000) / 10
	}
	shown := oneRow(withValues(ranked.Columns[0], recordIDs), withValues(ranked.Columns[1], amounts),
		valueColumn("share", "% del total", db.SQLDecimal, shares...))
	return shownResult{result: shown, table: TableInput{Columns: []string{recordColumn, "amount", "share"}},
		value: formatSoles(totalAmount), isEmpty: totalAmount == 0, extraNames: map[int]map[int32]string{0: {otherRecordsID: "Otros"}}}, nil
}

// compareTotals sets the period's sales beside the compared period's.
func (call *functionCall) compareTotals() (shownResult, error) {
	current, err := call.read(frameRead{Select: "SUM(amount)"})
	if err != nil {
		return shownResult{}, err
	}
	previous, err := call.read(frameRead{Select: "SUM(amount)", Period: call.comparePeriod})
	if err != nil {
		return shownResult{}, err
	}
	currentAmount, previousAmount := current.Columns[0].Values[0], previous.Columns[0].Values[0]
	change := percentChange(currentAmount, previousAmount)
	shown := oneRow(
		valueColumn("period_amount", periodLabel(call.period), db.SQLCents, currentAmount),
		valueColumn("compare_amount", periodLabel(call.comparePeriod), db.SQLCents, previousAmount),
		valueColumn("difference", "Diferencia", db.SQLCents, currentAmount-previousAmount),
		valueColumn("change", "Variación %", db.SQLDecimal, change),
	)
	value := fmt.Sprintf("%s frente a %s", formatSoles(currentAmount), formatSoles(previousAmount))
	if difference := currentAmount - previousAmount; difference >= 0 {
		value += ", " + formatSoles(difference) + " más"
	} else {
		value += ", " + formatSoles(-difference) + " menos"
	}
	if !math.IsNaN(change) {
		value += fmt.Sprintf(" (%+.1f %%)", change)
	}
	return shownResult{result: shown, isValueOnly: true, value: value, isEmpty: currentAmount == 0 && previousAmount == 0}, nil
}

// compareByRecord sets each record's sales in the period beside the compared period's: the records
// of either period, by the period's sales, then the compared one's.
func (call *functionCall) compareByRecord(recordColumn string, limit int) (shownResult, error) {
	statement := frameRead{Select: recordColumn + ", SUM(amount)", GroupBy: recordColumn, SortBy: "amount DESC"}
	current, err := call.read(statement)
	if err != nil {
		return shownResult{}, err
	}
	statement.Period = call.comparePeriod
	previous, err := call.read(statement)
	if err != nil {
		return shownResult{}, err
	}
	amountsByID := map[float64]*[2]float64{}
	recordIDs := []float64{}
	for side, result := range []db.SQLResult{current, previous} {
		for row := range result.RowCount {
			id := result.Columns[0].Values[row]
			if amountsByID[id] == nil {
				amountsByID[id] = &[2]float64{}
				recordIDs = append(recordIDs, id)
			}
			amountsByID[id][side] = result.Columns[1].Values[row]
		}
	}
	recordIDs = recordIDs[:min(len(recordIDs), limit)]
	currentAmounts, previousAmounts, changes := make([]float64, len(recordIDs)), make([]float64, len(recordIDs)), make([]float64, len(recordIDs))
	for i, id := range recordIDs {
		currentAmounts[i], previousAmounts[i] = amountsByID[id][0], amountsByID[id][1]
		changes[i] = percentChange(currentAmounts[i], previousAmounts[i])
	}
	shown := oneRow(withValues(current.Columns[0], recordIDs),
		valueColumn("period_amount", periodLabel(call.period), db.SQLCents, currentAmounts...),
		valueColumn("compare_amount", periodLabel(call.comparePeriod), db.SQLCents, previousAmounts...),
		valueColumn("change", "Variación %", db.SQLDecimal, changes...),
	)
	return shownResult{result: shown,
		table: TableInput{Columns: []string{recordColumn, "period_amount", "compare_amount", "change"}},
		chart: ChartInput{Type: "bar", X: recordColumn, Y: []string{"period_amount", "compare_amount"}}}, nil
}

// stopped lists the records with sales in the compared period and none in the period, with what they
// had before. A sale without a client is not a client that stopped.
func (call *functionCall) stopped(recordColumn, amountLabel string) (shownResult, error) {
	before, err := call.read(frameRead{Select: recordColumn + ", SUM(amount)", GroupBy: recordColumn, SortBy: "amount DESC", Period: call.comparePeriod})
	if err != nil {
		return shownResult{}, err
	}
	now, err := call.read(frameRead{Select: recordColumn + ", COUNT(*)", GroupBy: recordColumn})
	if err != nil {
		return shownResult{}, err
	}
	if now.Truncated {
		return shownResult{}, errors.New("hay demasiados registros con ventas en el periodo para compararlos: acorta el periodo")
	}
	soldNow := now.Columns[0].Values
	shown := keepRows(before, func(row int) bool {
		id := before.Columns[0].Values[row]
		return id > 0 && !slices.Contains(soldNow, id)
	})
	shown.Columns[1].Label = amountLabel + " " + periodLabel(call.comparePeriod)
	return shownResult{result: shown, table: TableInput{Columns: []string{recordColumn, "amount"}}}, nil
}

// percentChange is the change from previous to current in percent, one decimal; no value without a
// previous amount.
func percentChange(current, previous float64) float64 {
	if previous == 0 {
		return math.NaN()
	}
	return math.Round((current-previous)/previous*1000) / 10
}

// oneRow builds a result in Go from its columns, all of one length.
func oneRow(columns ...db.SQLResultColumn) db.SQLResult {
	return db.SQLResult{Columns: columns, RowCount: len(columns[0].Values)}
}

func valueColumn(name, label string, kind db.SQLKind, values ...float64) db.SQLResultColumn {
	return db.SQLResultColumn{Name: name, Label: label, Kind: kind, Values: values}
}

// groupColumn is a group the collections don't name (a weekday): Result.extraNames names it.
func groupColumn(name, label string, values []float64) db.SQLResultColumn {
	return db.SQLResultColumn{Name: name, Label: label, Kind: db.SQLRef, IsGroup: true, Values: values}
}

func withValues(column db.SQLResultColumn, values []float64) db.SQLResultColumn {
	column.Values = values
	return column
}

// keepRows is the result with only the rows keep accepts.
func keepRows(result db.SQLResult, keep func(row int) bool) db.SQLResult {
	kept := result
	kept.Columns = slices.Clone(result.Columns)
	for i := range kept.Columns {
		kept.Columns[i].Values = nil
	}
	kept.RowCount = 0
	for row := range result.RowCount {
		if !keep(row) {
			continue
		}
		for i, column := range result.Columns {
			kept.Columns[i].Values = append(kept.Columns[i].Values, column.Values[row])
		}
		kept.RowCount++
	}
	return kept
}

func columnSum(result db.SQLResult, columnIndex int) float64 {
	sum := 0.0
	for _, value := range result.Columns[columnIndex].Values {
		if !math.IsNaN(value) {
			sum += value
		}
	}
	return sum
}

// periodLabel is a period as a column header: "01/11–27/11/2026", "27/11/2026".
func periodLabel(period dayRange) string {
	from, to := time.Unix(int64(period.from)*86400, 0).UTC(), time.Unix(int64(period.to)*86400, 0).UTC()
	switch {
	case period.from == period.to:
		return to.Format("02/01/2006")
	case from.Year() == to.Year():
		return from.Format("02/01") + "–" + to.Format("02/01/2006")
	}
	return from.Format("02/01/2006") + "–" + to.Format("02/01/2006")
}

// formatMeasure writes a measure's value for a message: money in soles, quantities in units.
func formatMeasure(measure string, value float64) string {
	if measure == "amount" || measure == "debt_amount" {
		return formatSoles(value)
	}
	return formatQuantity(value) + " unidades"
}

// formatSoles writes cents as Peru writes soles: "S/ 1,234.50".
func formatSoles(cents float64) string {
	return "S/ " + withThousands(strconv.FormatFloat(cents/100, 'f', 2, 64))
}

func formatQuantity(quantity float64) string {
	return withThousands(strconv.FormatFloat(math.Round(quantity*100)/100, 'f', -1, 64))
}

// withThousands puts commas between the thousands of a formatted number.
func withThousands(number string) string {
	sign, digits := "", number
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}
	integerPart, fraction, hasFraction := strings.Cut(digits, ".")
	for i := len(integerPart) - 3; i > 0; i -= 3 {
		integerPart = integerPart[:i] + "," + integerPart[i:]
	}
	if hasFraction {
		return sign + integerPart + "." + fraction
	}
	return sign + integerPart
}
