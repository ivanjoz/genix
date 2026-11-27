package datafunctions

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"app/db"
	sales "app/sales/types"

	"github.com/ivanjoz/genix-orm/dataframe"
)

// testToday is Friday 2026-11-27. "D-6..D" is November 21–27 (the ISO weeks of the 16th and the
// 23rd); its compared period is November 14–20.
var testToday = int16(time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC).Unix() / 86400)

var (
	testProducts   = map[int32]string{1: "Leche Gloria 400 g", 2: "Leche Gloria 1 L", 3: "Pan Francés", 4: "Arroz Costeño"}
	testClients    = map[int32]string{11: "Comercial Andina SAC", 12: "Bodega Rosita", 13: "Inversiones Sol"}
	testCategories = map[int32]string{21: "Lácteos", 22: "Panadería", 23: "Ofertas"}
	testSites      = map[int32]string{31: "Lima Centro", 32: "Arequipa"}
)

// testSale is one sale line of the test company, as both sale frames hold it.
type testSale struct {
	daysAgo                     int16
	siteID, clientID, productID int64
	amount, debt                int64
	quantity, pendingQuantity   int64
}

// The sales: November 26–27 (in "D-6..D") and November 19 (in its compared period). Product 3 sells
// many cheap units; product 4 and client 13 sold only on the 19th; client 0 is a sale without client.
var testSales = []testSale{
	{daysAgo: 0, siteID: 31, clientID: 11, productID: 1, amount: 1000, quantity: 2},
	{daysAgo: 0, siteID: 32, clientID: 12, productID: 2, amount: 2500, debt: 500, quantity: 5, pendingQuantity: 1},
	{daysAgo: 1, siteID: 31, clientID: 0, productID: 3, amount: 300, quantity: 8},
	{daysAgo: 1, siteID: 31, clientID: 11, productID: 1, amount: 700, debt: 700, quantity: 1, pendingQuantity: 1},
	{daysAgo: 8, siteID: 31, clientID: 12, productID: 4, amount: 4000, quantity: 4},
	{daysAgo: 8, siteID: 32, clientID: 13, productID: 1, amount: 1000, quantity: 2},
}

// memorySource reads rows as a frame's files would: Keys[0] in the range, later Keys pinned.
func memorySource(rows []dataframe.Row) db.SQLSource {
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

// testFrames are the sale frames over testSales and an in-memory category dictionary.
func testFrames(int32) []db.SQLFrame {
	summaryRows, clientProductRows := []dataframe.Row{}, []dataframe.Row{}
	for _, sale := range testSales {
		sums := []int64{sale.amount, sale.debt, sale.quantity * dataframe.FloatScale, sale.pendingQuantity * dataframe.FloatScale}
		day := int64(testToday - sale.daysAgo)
		summaryRows = append(summaryRows, dataframe.Row{Keys: []int64{day, sale.siteID}, RowValues: []int64{sale.productID}, Sums: sums})
		clientProductRows = append(clientProductRows, dataframe.Row{Keys: []int64{day}, RowValues: []int64{sale.clientID, sale.productID}, Sums: sums})
	}
	categoryRows := []dataframe.Row{}
	for _, categoryProduct := range [][2]int64{{21, 1}, {21, 2}, {22, 3}, {23, 1}, {23, 4}} {
		categoryRows = append(categoryRows, dataframe.Row{Keys: []int64{0}, RowValues: categoryProduct[:], Sums: []int64{1}})
	}
	return sales.SaleFrameSQLFramesOver(sales.SaleFrameSQLSources{
		SaleSummary: memorySource(summaryRows), SaleClientProduct: memorySource(clientProductRows), ProductCategories: memorySource(categoryRows),
	})
}

// testDeps are the test company in memory; searches counts the text searches.
func testDeps() (Deps, *int) {
	searches := new(int)
	collection := func(noun string, names map[int32]string, details map[int32]string) RecordCollection {
		ids := slices.Sorted(func(yield func(int32) bool) {
			for id := range names {
				if !yield(id) {
					return
				}
			}
		})
		return RecordCollection{
			Noun: noun,
			// A record matches when it holds one of the name's words: classifyMatches decides.
			Search: func(_ int32, name string) ([]RecordMatch, error) {
				*searches++
				matches := []RecordMatch{}
				for _, id := range ids {
					if slices.ContainsFunc(normalizedWords(name), func(word string) bool { return bestWordMatch(word, normalizedWords(names[id])) != wordMatchNone }) {
						matches = append(matches, RecordMatch{ID: id, Name: names[id], Detail: details[id]})
					}
				}
				return matches, nil
			},
			NamesByID: func(_ int32, requested []int32) (map[int32]string, error) {
				found := map[int32]string{}
				for _, id := range requested {
					if names[id] != "" {
						found[id] = names[id]
					}
				}
				return found, nil
			},
		}
	}
	return Deps{CompanyID: 1, Frames: testFrames, Today: testToday, Records: map[string]RecordCollection{
		sales.FrameSQLCollectionProducts:   collection("product|producto", testProducts, nil),
		sales.FrameSQLCollectionClients:    collection("client|cliente", testClients, map[int32]string{11: "20123456789", 12: "45678912", 13: "20987654321"}),
		sales.FrameSQLCollectionCategories: collection("category|categoría", testCategories, nil),
		sales.FrameSQLCollectionSites:      collection("site|sede", testSites, nil),
	}}, searches
}

func runFunction(t *testing.T, deps Deps, name, argumentsJSON string) (Result, error) {
	t.Helper()
	callArguments := map[string]any{}
	if err := json.Unmarshal([]byte(argumentsJSON), &callArguments); err != nil {
		t.Fatalf("%s: %v", argumentsJSON, err)
	}
	return Functions.Run(context.Background(), deps, name, callArguments)
}

// renderAnswer is the answer as lines: the message, each table row as "a | b", each chart column as
// "Label: a, b".
func renderAnswer(answer Answer) string {
	renderValue := func(value any) string {
		switch typed := value.(type) {
		case nil:
			return "-"
		case float64:
			return strconv.FormatFloat(typed, 'f', -1, 64)
		}
		return value.(string)
	}
	lines := []string{answer.Message}
	for _, table := range answer.Tables {
		for row := range table.Columns[0].Values {
			cells := []string{}
			for _, column := range table.Columns {
				cells = append(cells, renderValue(column.Values[row]))
			}
			lines = append(lines, strings.Join(cells, " | "))
		}
	}
	for _, chart := range answer.Charts {
		for _, column := range chart.Columns {
			values := []string{}
			for _, value := range column.Values {
				values = append(values, renderValue(value))
			}
			lines = append(lines, column.Label+": "+strings.Join(values, ", "))
		}
	}
	return strings.Join(lines, "\n")
}

// Every function of the catalog, run on the test company, with the numbers it must show.
func TestEveryFunctionShowsItsNumbers(t *testing.T) {
	const lastWeek = `"period":"D-6..D"`
	cases := []struct {
		function, arguments string
		want                []string
	}{
		{"sales_total", `{}`, []string{"Total vendido del 14 al 27 de noviembre de 2026: S/ 95.00."}},
		{"sales_total", `{` + lastWeek + `,"products":["pan francés"]}`, []string{"Total vendido del 21 al 27 de noviembre de 2026 (producto: Pan Francés): S/ 3.00 y 8 unidades."}},
		{"sales_by_day", `{` + lastWeek + `}`, []string{"S/ 45.00 en total.", "Fecha: 2026-11-21, 2026-11-22, 2026-11-23, 2026-11-24, 2026-11-25, 2026-11-26, 2026-11-27", "Monto: 0, 0, 0, 0, 0, 10, 35"}},
		{"sales_by_week", `{` + lastWeek + `}`, []string{"Semana: 2026-11-16, 2026-11-23", "Monto: 0, 45"}},
		{"sales_by_month", `{"period":"M"}`, []string{"Ventas por mes del 1 al 27 de noviembre de 2026: S/ 95.00 en total.", "Mes: 2026-11-01", "Monto: 95"}},
		{"average_daily_sales", `{` + lastWeek + `}`, []string{": S/ 6.43 por día (S/ 45.00 en 7 días)."}},
		{"sales_by_weekday", `{` + lastWeek + `}`, []string{"Día: lunes, martes, miércoles, jueves, viernes, sábado, domingo", "Promedio: 0, 0, 0, 10, 35, 0, 0"}},
		{"sales_by_product", `{` + lastWeek + `}`, []string{"ordenados por monto.", "Leche Gloria 1 L | 25 | 5\nLeche Gloria 400 g | 17 | 3\nPan Francés | 3 | 8"}},
		{"sales_by_product", `{` + lastWeek + `,"rank_by":"quantity","top":2}`, []string{"ordenados por unidades.", "Pan Francés | 3 | 8\nLeche Gloria 1 L | 25 | 5"}},
		{"sales_by_client", `{` + lastWeek + `}`, []string{"Bodega Rosita | 25\nComercial Andina SAC | 17\n(ninguno) | 3"}},
		{"sales_by_category", `{` + lastWeek + `}`, []string{"las categorías suman más que el total", "Lácteos | 42\nOfertas | 17\nPanadería | 3"}},
		{"sales_by_site", `{` + lastWeek + `}`, []string{"S/ 45.00 en total.", "Sede: Arequipa, Lima Centro", "Monto: 25, 20"}},
		{"sales_share_by_product", `{` + lastWeek + `,"top":2}`, []string{"de un total de S/ 45.00.", "Leche Gloria 1 L | 25 | 55.6\nLeche Gloria 400 g | 17 | 37.8\nOtros | 3 | 6.7"}},
		{"sales_share_by_site", `{` + lastWeek + `}`, []string{"Arequipa | 25 | 55.6\nLima Centro | 20 | 44.4"}},
		{"sales_by_product_and_day", `{` + lastWeek + `,"products":["#2","pan francés"]}`, []string{"(productos: Leche Gloria 1 L, Pan Francés)", "Leche Gloria 1 L: 0, 0, 0, 0, 0, 0, 25", "Pan Francés: 0, 0, 0, 0, 0, 3, 0"}},
		{"sales_by_product_and_week", `{` + lastWeek + `,"products":["#2","pan francés"]}`, []string{"Leche Gloria 1 L: 0, 25", "Pan Francés: 0, 3"}},
		{"sales_by_product_and_month", `{"period":"M","products":["#1"]}`, []string{"Mes: 2026-11-01", "Leche Gloria 400 g: 27"}},
		{"sales_by_client_and_month", `{"period":"M","clients":["comercial andina"]}`, []string{"(cliente: Comercial Andina SAC)", "Comercial Andina SAC: 17"}},
		{"sales_by_category_and_week", `{` + lastWeek + `,"categories":["lácteos","ofertas"]}`, []string{"Lácteos: 0, 42", "Ofertas: 0, 17"}},
		{"sales_by_site_and_week", `{` + lastWeek + `}`, []string{"Lima Centro: 0, 20", "Arequipa: 0, 25"}},
		{"sales_vs_previous_period", `{` + lastWeek + `}`, []string{"Ventas del 21 al 27 de noviembre de 2026 comparadas con las del 14 al 20 de noviembre de 2026: S/ 45.00 frente a S/ 50.00, S/ 5.00 menos (-10.0 %)."}},
		{"sales_by_product_vs_previous_period", `{` + lastWeek + `}`, []string{
			"Leche Gloria 1 L | 25 | 0 | -\nLeche Gloria 400 g | 17 | 10 | 70\nPan Francés | 3 | 0 | -\nArroz Costeño | 0 | 40 | -100",
			"21/11–27/11/2026: 25, 17, 3, 0", "14/11–20/11/2026: 0, 10, 0, 40"}},
		{"sales_by_client_vs_previous_period", `{` + lastWeek + `}`, []string{"Bodega Rosita | 25 | 40 | -37.5\nComercial Andina SAC | 17 | 0 | -\n(ninguno) | 3 | 0 | -\nInversiones Sol | 0 | 10 | -100"}},
		{"sales_by_category_vs_previous_period", `{` + lastWeek + `}`, []string{"Lácteos | 42 | 10 | 320\nOfertas | 17 | 50 | -66\nPanadería | 3 | 0 | -"}},
		{"sales_by_site_vs_previous_period", `{` + lastWeek + `}`, []string{"Arequipa | 25 | 10 | 150\nLima Centro | 20 | 40 | -50"}},
		{"products_stopped_selling", `{` + lastWeek + `}`, []string{"Productos vendidos del 14 al 20 de noviembre de 2026 que no se vendieron del 21 al 27 de noviembre de 2026.", "\nArroz Costeño | 40"}},
		{"clients_stopped_buying", `{` + lastWeek + `}`, []string{"\nInversiones Sol | 10"}},
		{"unpaid_sales_total", `{` + lastWeek + `}`, []string{"Por cobrar de las ventas del 21 al 27 de noviembre de 2026: S/ 12.00."}},
		{"unpaid_sales_by_client", `{` + lastWeek + `}`, []string{"S/ 12.00 en total.", "Comercial Andina SAC | 7\nBodega Rosita | 5"}},
		{"unpaid_sales_by_week", `{` + lastWeek + `}`, []string{"Por cobrar: 0, 12"}},
		{"unpaid_sales_by_site", `{` + lastWeek + `}`, []string{"Sede: Lima Centro, Arequipa", "Por cobrar: 7, 5"}},
		{"undelivered_by_product", `{` + lastWeek + `}`, []string{"Leche Gloria 400 g | 1\nLeche Gloria 1 L | 1"}},
		{"undelivered_by_client_and_product", `{` + lastWeek + `}`, []string{"Comercial Andina SAC | Leche Gloria 400 g | 1\nBodega Rosita | Leche Gloria 1 L | 1"}},
		{"undelivered_by_site_and_product", `{` + lastWeek + `}`, []string{"Lima Centro | Leche Gloria 400 g | 1\nArequipa | Leche Gloria 1 L | 1"}},
		{"undelivered_by_product_and_week", `{` + lastWeek + `,"products":["#1","#2"]}`, []string{"Leche Gloria 400 g: 0, 1", "Leche Gloria 1 L: 0, 1"}},
	}
	deps, _ := testDeps()
	tested := map[string]bool{}
	for _, testCase := range cases {
		tested[testCase.function] = true
		result, err := runFunction(t, deps, testCase.function, testCase.arguments)
		if err != nil || result.Answer == nil {
			t.Errorf("%s %s: %v %+v", testCase.function, testCase.arguments, err, result)
			continue
		}
		rendered := renderAnswer(*result.Answer)
		for _, want := range testCase.want {
			if !strings.Contains(rendered, want) {
				t.Errorf("%s %s: missing %q in:\n%s", testCase.function, testCase.arguments, want, rendered)
			}
		}
	}
	for _, function := range Functions {
		if !tested[function.Name] {
			t.Errorf("%s has no expected numbers", function.Name)
		}
	}
}

func TestTheSummaryNamesTheFirstRecordsWithTheirValues(t *testing.T) {
	deps, _ := testDeps()
	result, err := runFunction(t, deps, "sales_by_product", `{"period":"D-6..D","top":2}`)
	if err != nil {
		t.Fatal(err)
	}
	want := `sales_by_product {"period":"D-6..D","top":2} | 2026-11-21..2026-11-27 | filas 2 | #2 Leche Gloria 1 L · Monto 25.00 · Cantidad 5; #1 Leche Gloria 400 g · Monto 17.00 · Cantidad 3`
	if result.Answer.Summary != want {
		t.Fatalf("summary:\n%s\nwant:\n%s", result.Answer.Summary, want)
	}
}

func TestAnEmptyPeriodSaysSoWithoutATable(t *testing.T) {
	deps, _ := testDeps()
	for name, wantEnd := range map[string]string{
		"sales_by_product":       "No hay ventas en ese periodo.",
		"sales_total":            "No hay ventas en ese periodo.",
		"undelivered_by_product": "No hay unidades por entregar de las ventas de ese periodo.",
	} {
		result, err := runFunction(t, deps, name, `{"period":"D-4..D-2"}`)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(result.Answer.Message, wantEnd) || len(result.Answer.Tables)+len(result.Answer.Charts) > 0 {
			t.Errorf("%s: %+v", name, result.Answer)
		}
	}
}

// Rule F: no dataset has client and site together. Rule Q: units rank only products, and a sales
// total shows units only for the products named.
func TestRulesFAndQ(t *testing.T) {
	deps, _ := testDeps()
	_, err := runFunction(t, deps, "sales_total", `{"clients":["comercial andina"],"sites":["lima"]}`)
	if err == nil || !strings.Contains(err.Error(), "no hay datos de ventas por cliente y sede a la vez") {
		t.Errorf("rule F: %v", err)
	}
	for name, arguments := range map[string]string{
		"sales_by_client":  `{"rank_by":"quantity"}`,
		"sales_by_product": `{"rank_by":"units"}`,
	} {
		if _, err := runFunction(t, deps, name, arguments); err == nil {
			t.Errorf("rule Q: %s %s was accepted", name, arguments)
		}
	}
	result, err := runFunction(t, deps, "sales_total", `{"period":"D-6..D","sites":["lima"]}`)
	if err != nil || strings.Contains(result.Answer.Message, "unidades") || !strings.HasSuffix(result.Answer.Message, "(sede: Lima Centro): S/ 20.00.") {
		t.Errorf("rule Q: a total without products: %v %+v", err, result.Answer)
	}
}

func TestArgumentErrorsGoBackToTheModel(t *testing.T) {
	deps, _ := testDeps()
	for _, testCase := range []struct{ function, arguments, wantError string }{
		{"sales_by_day", `{"top":5}`, "sales_by_day no recibe «top»; sus argumentos son: period, products, clients, categories, sites"},
		{"sales_by_product_and_day", `{}`, "sales_by_product_and_day necesita «products»"},
		{"sales_by_product", `{"top":0}`, "top es un número entero de 1 a 50"},
		{"sales_by_product", `{"top":2.5}`, "top es un número entero de 1 a 50"},
		{"sales_by_product", `{"top":"5"}`, "top es un número entero de 1 a 50"},
		{"sales_total", `{"products":"pan"}`, "products es una lista de nombres"},
		{"sales_total", `{"products":["queso"]}`, "products: no hay producto con el nombre «queso»"},
		{"sales_total", `{"categories":["lácteos","ofertas"]}`, "aquí va una sola categoría"},
		{"sales_forecast", `{}`, "no existe la función «sales_forecast»"},
		// Period errors.
		{"sales_total", `{"period":"D+1"}`, `period: "D+1" no es un periodo`},
		{"sales_total", `{"period":"Y-3..D"}`, "este reporte lee como máximo 400"},
		{"sales_total", `{"period":"Y=2027"}`, "empieza después de hoy"},
		{"sales_vs_previous_period", `{"compare_period":"M=13"}`, "compare_period: \"M=13\" tiene un valor fuera de rango"},
	} {
		_, err := runFunction(t, deps, testCase.function, testCase.arguments)
		if err == nil || !strings.Contains(err.Error(), testCase.wantError) {
			t.Errorf("%s %s: got %v, want %q", testCase.function, testCase.arguments, err, testCase.wantError)
		}
	}
	// A single category filters exactly: product 1 is in Ofertas, sold S/ 17 in the last 7 days.
	result, err := runFunction(t, deps, "sales_total", `{"period":"D-6..D","categories":["ofertas"]}`)
	if err != nil || !strings.HasSuffix(result.Answer.Message, ": S/ 17.00.") {
		t.Errorf("one category: %v %+v", err, result)
	}
}

// An ambiguous name ends the call with a choice; the pick comes back as #<id> and never resolves
// again.
func TestAnAmbiguousNameAsksAndThePickRunsWithoutASearch(t *testing.T) {
	deps, searches := testDeps()
	result, err := runFunction(t, deps, "sales_total", `{"period":"D-6..D","products":["leche gloria"]}`)
	if err != nil || result.Choice == nil || result.Answer != nil {
		t.Fatalf("expected a choice: %v %+v", err, result)
	}
	choice := *result.Choice
	if choice.Argument != "products" || choice.Name != "leche gloria" || choice.Question != "¿Qué producto es «leche gloria»?" {
		t.Fatalf("unexpected choice: %+v", choice)
	}
	wantOptions := []ChoiceOption{{"1", "Leche Gloria 400 g"}, {"2", "Leche Gloria 1 L"}, {IncludeAllChoiceID, "Incluir todos"}}
	if !slices.Equal(choice.Options, wantOptions) {
		t.Fatalf("options: %+v", choice.Options)
	}
	if summary := choice.Summary(); summary != "Pregunta | ¿Qué producto es «leche gloria»? | #1 Leche Gloria 400 g; #2 Leche Gloria 1 L" {
		t.Fatalf("summary: %s", summary)
	}
	if records, isOffered := choice.PickedRecords(IncludeAllChoiceID); !isOffered || !slices.Equal(records, []string{"#1", "#2"}) {
		t.Fatalf("include all: %v %v", records, isOffered)
	}
	if _, isOffered := choice.PickedRecords("4"); isOffered {
		t.Fatal("a record that wasn't offered was accepted")
	}
	pickedRecords, _ := choice.PickedRecords("2")
	encodedPick, _ := json.Marshal(pickedRecords)

	searchesBefore := *searches
	result, err = runFunction(t, deps, "sales_total", `{"period":"D-6..D","products":`+string(encodedPick)+`}`)
	if err != nil || !strings.HasSuffix(result.Answer.Message, "(producto: Leche Gloria 1 L): S/ 25.00 y 5 unidades.") {
		t.Fatalf("the pick: %v %+v", err, result)
	}
	if *searches != searchesBefore {
		t.Fatal("a picked record was searched again")
	}
	if _, err := runFunction(t, deps, "sales_total", `{"products":["#99"]}`); err == nil || !strings.Contains(err.Error(), "«#99»") {
		t.Fatalf("an unknown pick: %v", err)
	}
}

// Each golden question of the sales tool maps to a function, but the documented gap.
func TestGoldenQuestionsMapToFunctions(t *testing.T) {
	goldenCalls := map[string]struct{ function, arguments string }{
		"¿Cuáles fueron los 10 productos más vendidos este mes?":                  {"sales_by_product", `{"period":"M","top":10}`},
		"¿Cuánto vendí ayer?":                                                     {"sales_total", `{"period":"D-1"}`},
		"ventas por día de los últimos 30 días":                                   {"sales_by_day", `{"period":"D-29..D"}`},
		"ventas semanales de los últimos 3 meses con las unidades":                {"sales_by_week", `{"period":"M-2..M"}`},
		"¿Cuánto vendí por mes este año?":                                         {"sales_by_month", `{"period":"Y"}`},
		"compara las ventas diarias de leche gloria y pan francés del mes pasado": {"sales_by_product_and_day", `{"period":"M-1","products":["leche gloria","pan francés"]}`},
		"¿Quiénes son mis 5 mejores clientes del último trimestre?":               {"sales_by_client", `{"period":"M-2..M","top":5}`},
		"¿Qué productos le vendí a Comercial Andina este año?":                    {"sales_by_product", `{"period":"Y","clients":["Comercial Andina"]}`},
		"¿Cuánto me deben los clientes por las ventas de este mes?":               {"unpaid_sales_by_client", `{"period":"M"}`},
		"productos pendientes de entrega de la semana pasada":                     {"undelivered_by_product", `{"period":"S-1"}`},
		"top 10 products by units sold last week":                                 {"sales_by_product", `{"period":"S-1","top":10,"rank_by":"quantity"}`},
		"¿Cuál es la categoría más vendida de la semana pasada?":                  {"sales_by_category", `{"period":"S-1"}`},
	}
	documentedGaps := map[string]string{
		"¿Cuál es el ticket promedio diario por producto en los últimos 30 días?": "no data function: the frames have no number of sales (plan 13 §2)",
	}
	content, err := os.ReadFile("../salestools/testdata/golden.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	deps, _ := testDeps()
	mappedCount := 0
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		golden := struct{ Question string }{}
		if err := json.Unmarshal([]byte(line), &golden); err != nil {
			t.Fatal(err)
		}
		call, isMapped := goldenCalls[golden.Question]
		if _, isGap := documentedGaps[golden.Question]; isGap == isMapped {
			t.Errorf("%q must be mapped to a function or be a documented gap", golden.Question)
			continue
		}
		if !isMapped {
			continue
		}
		mappedCount++
		// The ambiguous "leche gloria" ends with a choice, which is the call working.
		if result, err := runFunction(t, deps, call.function, call.arguments); err != nil || (result.Answer == nil && result.Choice == nil) {
			t.Errorf("%q → %s %s: %v", golden.Question, call.function, call.arguments, err)
		}
	}
	if mappedCount != 12 {
		t.Errorf("%d golden questions map to a function, want 12", mappedCount)
	}
}

// The catalog's specs: what search shows and the training data references must hold.
func TestCatalogSpecs(t *testing.T) {
	recordWords := []string{}
	for _, names := range []map[int32]string{testProducts, testClients, testCategories, testSites} {
		for _, name := range names {
			recordWords = append(recordWords, normalizedWords(name)...)
		}
	}
	// Examples are embedded for search, and the model's query has no instance values: no digits, no
	// record names, no dates.
	dateWords := regexp.MustCompile(`(?i)\b(ayer|hoy|anteayer|pasad[oa]s?|enero|febrero|marzo|abril|mayo|junio|julio|agosto|septiembre|octubre|noviembre|diciembre)\b`)
	if len(Functions) != 33 {
		t.Errorf("%d functions, plan 13 §5 has 33", len(Functions))
	}
	names := map[string]bool{}
	for _, function := range Functions {
		if names[function.Name] || !regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`).MatchString(function.Name) {
			t.Errorf("%s: name repeated or not snake_case", function.Name)
		}
		names[function.Name] = true
		switch {
		case function.Domain != "sales" || function.Access != salesAccess || function.Description == "" || function.Message == "" || function.run == nil:
			t.Errorf("%s: incomplete spec", function.Name)
		case len(strings.Fields(function.Title)) > 8 || len(strings.Fields(function.Gist)) > 12 || strings.Contains(function.Gist, ":"):
			t.Errorf("%s: title over 8 words or gist over 12 or with a colon", function.Name)
		case !strings.Contains(function.Description, "anuladas"):
			t.Errorf("%s: the description doesn't say annulled sales are left out", function.Name)
		case strings.HasPrefix(function.Name, "unpaid_") && !strings.Contains(function.Description, "deuda total"):
			t.Errorf("%s: the description doesn't say it is not the client's whole debt", function.Name)
		case strings.Contains(function.Name, "category") && !strings.Contains(function.Description, "más que el total"):
			t.Errorf("%s: the description doesn't say categories overlap", function.Name)
		case len(function.Examples) < 8:
			t.Errorf("%s: %d examples, at least 8", function.Name, len(function.Examples))
		}
		for _, argument := range function.Arguments {
			if !slices.Contains(argumentConventions, argument.Name) {
				t.Errorf("%s: argument %q is not a convention", function.Name, argument.Name)
			}
		}
		for _, example := range function.Examples {
			hasRecordWord := slices.ContainsFunc(normalizedWords(example), func(word string) bool { return slices.Contains(recordWords, word) })
			if strings.ContainsAny(example, "0123456789") || hasRecordWord || dateWords.MatchString(example) {
				t.Errorf("%s: the example %q has an instance value", function.Name, example)
			}
		}
	}
	if signature := Functions[6].Signature(); signature != "sales_by_product(period?, clients?, categories?, sites?, top?, rank_by?: amount|quantity) | Ventas por producto: ranking de productos por monto o por unidades vendidas" {
		t.Errorf("signature: %s", signature)
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
		fromDay, toDay, err := ResolvePeriod(period, today, 400, "es")
		if err != nil || fromDay != want[0] || toDay != want[1] {
			t.Errorf("%s: got %d..%d %v, want %d..%d", period, fromDay, toDay, err, want[0], want[1])
		}
	}
	for _, period := range []string{"D+1", "Y-3..D", "M=13"} {
		if _, _, err := ResolvePeriod(period, today, 400, "es"); err == nil {
			t.Errorf("%s was accepted", period)
		}
	}
	for period, want := range map[[2]int16]string{
		{today, today}:                       "del 27 de octubre de 2026",
		{day(2026, 9, 1), day(2026, 9, 30)}:  "de septiembre de 2026",
		{day(2025, 1, 1), day(2025, 12, 31)}: "de 2025",
		{day(2026, 9, 28), today}:            "del 28 de septiembre al 27 de octubre de 2026",
		{day(2025, 12, 29), day(2026, 1, 4)}: "del 29 de diciembre de 2025 al 4 de enero de 2026",
	} {
		if words := spanishPeriodWords(period[0], period[1]); words != want {
			t.Errorf("%v: %q, want %q", period, words, want)
		}
	}
}
