package datafunctions

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"app/core"
	"app/db"
	sales "app/sales/types"
)

// Function is one data function: data, not a hand-written handler. Its arguments are a subset of
// the conventions (argumentConventions), so a model fills an unseen function from its signature.
type Function struct {
	// Name is snake_case, unique and stable: training data references it.
	Name   string
	Domain string
	// Title is Spanish, at most 8 words; Gist completes it in the signature line, at most 12.
	Title string
	Gist  string
	// Description is what one row is, what the numbers count and what they do not count.
	Description string
	Arguments   []Argument
	// Examples are Spanish phrasings without instance values (names, numbers, dates): the search
	// index embeds them, and the model's query has none either.
	Examples []string
	// Access is the API permission the user needs.
	Access string
	// Message is the reply's sentence, with {periodo}, {comparado}, {filtros}, {orden} and {valor}.
	Message string
	run     func(call *functionCall) (shownResult, error)
}

type Argument struct {
	Name       string
	IsRequired bool
}

// arguments reads a declaration like ("products", "period?"): a trailing ? marks an optional one.
func arguments(names ...string) []Argument {
	declared := make([]Argument, len(names))
	for i, name := range names {
		declared[i] = Argument{Name: strings.TrimSuffix(name, "?"), IsRequired: !strings.HasSuffix(name, "?")}
	}
	return declared
}

// argumentConventions are every argument a function may declare, with one meaning across the catalog.
var argumentConventions = []string{"period", "compare_period", "products", "clients", "categories", "sites", "top", "rank_by"}

type recordArgument struct{ argument, collection, column, noun string }

// recordArguments are the record list arguments, in the order they are resolved and named, with
// their collection, their FrameSQL column, and their Spanish noun (singular|plural).
var recordArguments = []recordArgument{
	{"products", sales.FrameSQLCollectionProducts, "product_id", "producto|productos"},
	{"clients", sales.FrameSQLCollectionClients, "client_id", "cliente|clientes"},
	{"categories", sales.FrameSQLCollectionCategories, "category_id", "categoría|categorías"},
	{"sites", sales.FrameSQLCollectionSites, "site_id", "sede|sedes"},
}

// Signature is the function as `search` lists it: required arguments bare, optional ones with ?,
// rank_by's values inline, then the title and its gist.
func (function Function) Signature() string {
	declared := make([]string, len(function.Arguments))
	for i, argument := range function.Arguments {
		declared[i] = argument.Name
		if !argument.IsRequired {
			declared[i] += "?"
		}
		if argument.Name == "rank_by" {
			declared[i] += ": amount|quantity"
		}
	}
	return function.Name + "(" + strings.Join(declared, ", ") + ") | " + function.Title + ": " + function.Gist
}

// Catalog is the functions genix-bot can call.
type Catalog []Function

// Deps are what a function reads, injected: Genix passes the company's frames and records
// (DatabaseDeps); tests and genix-local-ai pass frames and records in memory.
type Deps struct {
	CompanyID int32
	Frames    func(companyID int32) []db.SQLFrame
	Records   map[string]RecordCollection
	// Today is the UnixDay periods resolve against.
	Today int16
}

// DatabaseDeps reads one company's sale frames and records at the effective clock's today.
func DatabaseDeps(companyID int32) Deps {
	return Deps{CompanyID: companyID, Frames: sales.SaleFrameSQLFrames, Records: DatabaseRecordCollections(), Today: core.FechaUnix()}
}

// Result is a call's outcome when it is not an error for the model: an Answer that ends the turn, or
// a Choice the user must make before the call can run (an ambiguous name).
type Result struct {
	Answer *Answer
	Choice *Choice
}

// dayRange is a resolved period, UnixDays inclusive.
type dayRange struct{ from, to int16 }

// functionCall is one call with its arguments read: periods resolved, names turned into IDs.
type functionCall struct {
	function      *Function
	deps          Deps
	period        dayRange
	comparePeriod dayRange
	// recordIDs are the record arguments' IDs by argument; a missing one is every record.
	recordIDs map[string][]int32
	// recordNames are their names, for the message.
	recordNames map[string][]string
	top         int
	rankBy      string
}

// shownResult is what a function shows: a FrameSQL result, or one built in Go over FrameSQL reads,
// with the view that lays it out.
type shownResult struct {
	result db.SQLResult
	table  TableInput
	chart  ChartInput
	// isValueOnly answers with the message alone (V): {valor} holds the numbers.
	isValueOnly bool
	value       string
	// isEmpty answers with emptyMessage and no table; a result without rows is empty too.
	isEmpty      bool
	emptyMessage string
	// note is a sentence after the message (categories overlap).
	note string
	// extraNames names record values the collections don't know (weekdays, "Otros"), per column.
	extraNames map[int]map[int32]string
}

const (
	defaultTop     = 10
	maxTop         = 50
	salesAccess    = "GET.sale-summary"
	noSalesMessage = "No hay ventas en ese periodo."
)

// Run runs one function. Errors are written in Spanish for the model, which fixes its call or tells
// the user; a Result is the turn's end.
func (catalog Catalog) Run(_ context.Context, deps Deps, name string, callArguments map[string]any) (Result, error) {
	index := slices.IndexFunc(catalog, func(function Function) bool { return function.Name == name })
	if index < 0 {
		return Result{}, fmt.Errorf("no existe la función «%s»: búscala con search", name)
	}
	call, choice, err := newFunctionCall(&catalog[index], deps, callArguments)
	if err != nil || choice != nil {
		return Result{Choice: choice}, err
	}
	shown, err := call.function.run(call)
	if errors.Is(err, db.ErrFrameNotBuilt) {
		return Result{}, errors.New("los datos de ventas se están reconstruyendo tras un cambio y no se pueden leer por unos 20 minutos: díselo al usuario")
	}
	if err != nil {
		return Result{}, err
	}
	answer, err := call.answer(shown, callArguments)
	if err != nil {
		return Result{}, err
	}
	return Result{Answer: &answer}, nil
}

// newFunctionCall reads the arguments against the function's declaration and the conventions. A
// name that needs the user's pick returns the Choice instead.
func newFunctionCall(function *Function, deps Deps, callArguments map[string]any) (*functionCall, *Choice, error) {
	declaredNames := make([]string, len(function.Arguments))
	for i, argument := range function.Arguments {
		declaredNames[i] = argument.Name
		if _, isGiven := callArguments[argument.Name]; argument.IsRequired && !isGiven {
			return nil, nil, fmt.Errorf("%s necesita «%s»", function.Name, argument.Name)
		}
	}
	for argumentName := range callArguments {
		if !slices.Contains(declaredNames, argumentName) {
			return nil, nil, fmt.Errorf("%s no recibe «%s»; sus argumentos son: %s", function.Name, argumentName, strings.Join(declaredNames, ", "))
		}
	}

	call := &functionCall{function: function, deps: deps, recordIDs: map[string][]int32{}, recordNames: map[string][]string{}, top: defaultTop, rankBy: "amount"}
	periodText, err := stringArgument(callArguments, "period", DefaultPeriod)
	if err != nil {
		return nil, nil, err
	}
	if call.period, err = resolveDayRange(periodText, deps.Today); err != nil {
		return nil, nil, fmt.Errorf("period: %w", err)
	}
	// Without compare_period, the period of the same length right before.
	periodLength := call.period.to - call.period.from + 1
	call.comparePeriod = dayRange{from: call.period.from - periodLength, to: call.period.from - 1}
	if comparePeriodText, err := stringArgument(callArguments, "compare_period", ""); err != nil {
		return nil, nil, err
	} else if comparePeriodText != "" {
		if call.comparePeriod, err = resolveDayRange(comparePeriodText, deps.Today); err != nil {
			return nil, nil, fmt.Errorf("compare_period: %w", err)
		}
	}
	if topValue, isGiven := callArguments["top"]; isGiven {
		top, isNumber := topValue.(float64)
		if topInteger, isInteger := topValue.(int); isInteger {
			top, isNumber = float64(topInteger), true
		}
		if !isNumber || top != math.Trunc(top) || top < 1 || top > maxTop {
			return nil, nil, fmt.Errorf("top es un número entero de 1 a %d, no %v", maxTop, topValue)
		}
		call.top = int(top)
	}
	if call.rankBy, err = stringArgument(callArguments, "rank_by", "amount"); err != nil {
		return nil, nil, err
	}
	// Rule Q: units only rank products, the only function declaring rank_by.
	if call.rankBy != "amount" && call.rankBy != "quantity" {
		return nil, nil, fmt.Errorf("rank_by es amount o quantity, no %q", call.rankBy)
	}

	for _, record := range recordArguments {
		names, err := stringListArgument(callArguments, record.argument)
		if err != nil {
			return nil, nil, err
		}
		if len(names) == 0 {
			continue
		}
		ids, err := ResolveNames(deps.Records, deps.CompanyID, "es", record.collection, names, nil)
		var needsChoice *NeedsChoiceError
		if errors.As(err, &needsChoice) {
			choice := needsChoice.Choice
			choice.Argument = record.argument
			return nil, &choice, nil
		}
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", record.argument, err)
		}
		namesByID, err := deps.Records[record.collection].NamesByID(deps.CompanyID, ids)
		if err != nil {
			return nil, nil, err
		}
		call.recordIDs[record.argument] = ids
		for _, id := range ids {
			call.recordNames[record.argument] = append(call.recordNames[record.argument], namesByID[id])
		}
	}
	return call, nil, nil
}

func resolveDayRange(period string, today int16) (dayRange, error) {
	fromDay, toDay, err := ResolvePeriod(period, today, db.FrameMaxDays, "es")
	return dayRange{from: fromDay, to: toDay}, err
}

func stringArgument(callArguments map[string]any, name, defaultValue string) (string, error) {
	value, isGiven := callArguments[name]
	if !isGiven {
		return defaultValue, nil
	}
	text, isText := value.(string)
	if !isText || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("%s es un texto, no %v", name, value)
	}
	return strings.TrimSpace(text), nil
}

// stringListArgument reads a record list: names exactly as the user wrote them, or "#<id>" picks.
func stringListArgument(callArguments map[string]any, name string) ([]string, error) {
	value, isGiven := callArguments[name]
	if !isGiven {
		return nil, nil
	}
	listError := fmt.Errorf(`%s es una lista de nombres: ["nombre 1", "nombre 2"]`, name)
	var items []any
	switch typed := value.(type) {
	case []any:
		items = typed
	case []string:
		for _, text := range typed {
			items = append(items, text)
		}
	default:
		return nil, listError
	}
	names := make([]string, len(items))
	for i, item := range items {
		text, isText := item.(string)
		if !isText || strings.TrimSpace(text) == "" {
			return nil, listError
		}
		names[i] = text
	}
	return names, nil
}

// frameRead is one FrameSQL statement of a function, without its WHERE: read adds the period and
// the call's record filters.
type frameRead struct {
	Select  string
	GroupBy string
	SortBy  string
	Limit   int
	// Period is the call's period when zero.
	Period dayRange
}

// read runs a statement on the dataset its columns need. Rule F: no dataset has client and site
// together. A product in two categories is in both category views' rows, so a filter of several
// categories would count it twice: it takes one category unless the statement groups by category.
func (call *functionCall) read(statement frameRead) (db.SQLResult, error) {
	groups := strings.Split(statement.GroupBy, ", ")
	usesColumn := func(column string) bool {
		recordIndex := slices.IndexFunc(recordArguments, func(record recordArgument) bool { return record.column == column })
		return slices.Contains(groups, column) || len(call.recordIDs[recordArguments[recordIndex].argument]) > 0
	}
	if usesColumn("client_id") && usesColumn("site_id") {
		return db.SQLResult{}, errors.New("no hay datos de ventas por cliente y sede a la vez: quita clients o sites")
	}
	if len(call.recordIDs["categories"]) > 1 && !slices.Contains(groups, "category_id") {
		return db.SQLResult{}, errors.New("categories: aquí va una sola categoría, porque un producto en dos categorías se contaría dos veces")
	}
	dataset := "sale_summary"
	switch {
	case usesColumn("category_id") && usesColumn("client_id"):
		dataset = "sale_client_category"
	case usesColumn("category_id"):
		dataset = "sale_category"
	case usesColumn("client_id"):
		dataset = "sale_client_product"
	}

	period := cmp.Or(statement.Period, call.period)
	sql := fmt.Sprintf("SELECT %s WHERE date BETWEEN %d AND %d", statement.Select, period.from, period.to)
	for _, record := range recordArguments {
		if ids := call.recordIDs[record.argument]; len(ids) > 0 {
			idTexts := make([]string, len(ids))
			for i, id := range ids {
				idTexts[i] = strconv.Itoa(int(id))
			}
			sql += " AND " + record.column + " IN (" + strings.Join(idTexts, ", ") + ")"
		}
	}
	if statement.GroupBy != "" {
		sql += " GROUP BY " + statement.GroupBy
	}
	if statement.SortBy != "" {
		sql += " SORT BY " + statement.SortBy
	}
	sql += " LIMIT " + strconv.Itoa(cmp.Or(statement.Limit, MaxTableRows))
	prepared, err := db.PrepareFrameSQL(dataset, sql, call.deps.Frames(call.deps.CompanyID), db.SQLOptions{MaxRows: MaxTableRows})
	if err != nil {
		return db.SQLResult{}, fmt.Errorf("%s: %s: %w", call.function.Name, sql, err)
	}
	return prepared.Run(db.SQLResolvers{})
}

// answer writes the reply: the message from the spec's template, the table and the chart, and the
// Summary later turns read.
func (call *functionCall) answer(shown shownResult, callArguments map[string]any) (Answer, error) {
	rankWords := map[string]string{"amount": "por monto", "quantity": "por unidades"}[call.rankBy]
	placeholders := strings.NewReplacer("{periodo}", spanishPeriodWords(call.period.from, call.period.to),
		"{comparado}", spanishPeriodWords(call.comparePeriod.from, call.comparePeriod.to),
		"{filtros}", call.filterWords(), "{orden}", rankWords, "{valor}", shown.value)
	message := placeholders.Replace(call.function.Message)

	encodedArguments, _ := json.Marshal(callArguments)
	summary := fmt.Sprintf("%s %s | %s..%s", call.function.Name, encodedArguments, FormatUnixDay(int64(call.period.from)), FormatUnixDay(int64(call.period.to)))
	if slices.ContainsFunc(call.function.Arguments, func(argument Argument) bool { return argument.Name == "compare_period" }) {
		summary += fmt.Sprintf(" vs %s..%s", FormatUnixDay(int64(call.comparePeriod.from)), FormatUnixDay(int64(call.comparePeriod.to)))
	}
	result := shown.result
	if shown.isEmpty || result.RowCount == 0 {
		return Answer{Message: message + "\n\n" + placeholders.Replace(cmp.Or(shown.emptyMessage, noSalesMessage)), Summary: summary + " | sin datos"}, nil
	}
	if shown.note != "" {
		message += " " + shown.note
	}
	namesByColumn, err := RecordNames(call.deps.Records, call.deps.CompanyID, result)
	if err != nil {
		return Answer{}, fmt.Errorf("load the record names: %w", err)
	}
	for columnIndex, extraNames := range shown.extraNames {
		mergedNames := map[int32]string{}
		maps.Copy(mergedNames, namesByColumn[columnIndex])
		maps.Copy(mergedNames, extraNames)
		namesByColumn[columnIndex] = mergedNames
	}
	answer := Answer{Message: message, Summary: summary + fmt.Sprintf(" | filas %d | ", result.RowCount) + summarizeRows(result, namesByColumn)}
	if shown.isValueOnly {
		return answer, nil
	}
	// A chart needs every point: a cut result would draw a wrong line.
	if shown.chart.X != "" && result.Truncated {
		return Answer{}, errors.New("hay demasiados datos para el gráfico: acorta el periodo o pide menos registros")
	}
	if result.Truncated && result.RowCount == MaxTableRows {
		answer.Message += fmt.Sprintf("\n\nSe muestran los primeros %d.", result.RowCount)
	}
	view, err := PlanView(result.Columns, shown.table, shown.chart)
	if err != nil {
		return Answer{}, fmt.Errorf("%s: %w", call.function.Name, err)
	}
	answer.Tables, answer.Charts, err = RenderView("es", view, result, namesByColumn)
	return answer, err
}

// filterWords names the record arguments for the message: " (producto: Leche Gloria; sede: Lima)".
func (call *functionCall) filterWords() string {
	filters := []string{}
	for _, record := range recordArguments {
		names := call.recordNames[record.argument]
		if len(names) == 0 {
			continue
		}
		singular, plural, _ := strings.Cut(record.noun, "|")
		filters = append(filters, map[bool]string{true: singular, false: plural}[len(names) == 1]+": "+strings.Join(names, ", "))
	}
	if len(filters) == 0 {
		return ""
	}
	return " (" + strings.Join(filters, "; ") + ")"
}

// summarizeRows is the Summary's tail: the first rows with their values, records as "#id Name" so
// a follow-up can call again with them.
func summarizeRows(result db.SQLResult, namesByColumn []map[int32]string) string {
	rows := []string{}
	for row := range min(result.RowCount, 10) {
		cells := []string{}
		for i, column := range result.Columns {
			value := column.Values[row]
			switch {
			case column.Kind == db.SQLRef && value > 0 && namesByColumn[i][int32(value)] != "" && column.Collection != "":
				cells = append(cells, fmt.Sprintf("#%d %s", int32(value), namesByColumn[i][int32(value)]))
			case column.IsGroup || column.Kind == db.SQLRef:
				cells = append(cells, fmt.Sprint(cellValue("es", column, value, namesByColumn[i], false)))
			case math.IsNaN(value):
				cells = append(cells, columnHeader("es", column)+" -")
			case column.Kind == db.SQLCents:
				cells = append(cells, columnHeader("es", column)+" "+strconv.FormatFloat(value/100, 'f', 2, 64))
			default:
				cells = append(cells, columnHeader("es", column)+" "+strconv.FormatFloat(math.Round(value*100)/100, 'f', -1, 64))
			}
		}
		rows = append(rows, strings.Join(cells, " · "))
	}
	return strings.Join(rows, "; ")
}
