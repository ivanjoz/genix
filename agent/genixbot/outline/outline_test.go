package outline

import (
	"fmt"
	"strings"
	"testing"
)

// The stock-changes table: three headers span two rows, two group headers span sub-columns.
func TestFlattenHeaderGridJoinsGroupedHeaders(t *testing.T) {
	rows := [][]HeaderCell{
		{
			{Text: "Producto", ColSpan: 1, RowSpan: 2}, {Text: "Presentación", ColSpan: 1, RowSpan: 2},
			{Text: "Seriales", ColSpan: 1, RowSpan: 2}, {Text: "Costo Unitario", ColSpan: 2, RowSpan: 1},
			{Text: "Stock", ColSpan: 3, RowSpan: 1},
		},
		{
			{Text: "Consolidado", ColSpan: 1, RowSpan: 1}, {Text: "Nuevo", ColSpan: 1, RowSpan: 1},
			{Text: "Simple", ColSpan: 1, RowSpan: 1}, {Text: "Loteado", ColSpan: 1, RowSpan: 1},
			{Text: "Total", ColSpan: 1, RowSpan: 1},
		},
	}
	got := strings.Join(FlattenHeaderGrid(rows), " | ")
	want := "Producto | Presentación | Seriales | Costo Unitario › Consolidado | Costo Unitario › Nuevo | Stock › Simple | Stock › Loteado | Stock › Total"
	if got != want {
		t.Fatalf("headers\n got: %s\nwant: %s", got, want)
	}
}

func TestSpanishLabelKeepsSpanishHalf(t *testing.T) {
	cases := map[string]string{
		"Name|Nombre": "Nombre",
		"Department | Province | District|Departamento | Provincia | Distrito": "Departamento / Provincia / Distrito",
		"Opens the side layer to create a new client|cliente.":                 "Opens the side layer to create a new cliente.",
		"Email":         "Email",
		"Nuevo Cliente": "Nuevo Cliente",
	}
	for input, want := range cases {
		if got := spanishLabel(input); got != want {
			t.Errorf("spanishLabel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRenderFormShowsEmptyValuesAndRows(t *testing.T) {
	snapshot := `<div><header><div><span>Clientes</span></div></header>
<Layer id="76" label="Nuevo Cliente" methods="open,close">
  <div><span>Nuevo Cliente</span></div>
  <Button id="77" label="Guardar" methods="click"/>
  <Input id="79" label="Name|Nombre" methods="setValue"/>
  <Select id="82" label="Person Type|Tipo Persona" value="[1] Person|Persona" methods="search,select,getOptions" options-count="2" options="[1] Persona|[2] Empresa"/>
  <Checkbox id="93" label="Agente de retención" value="0" methods="click"/>
</Layer>
<Table id="75"><table><thead><tr><th>ID</th><th>Nombre</th></tr></thead>
<TableBody id="75" methods="selectRow"><Row id="75:100"><td>189</td><td>Sabor | Criollo</td></Row></TableBody></table></Table></div>`
	got, err := Render(snapshot, "", Options{Route: "/crm/customers"})
	if err != nil {
		t.Fatal(err)
	}
	want := `Page /crm/customers | Clientes
Layer 76 | Nuevo Cliente
 Button 77 | Guardar
 Input 79 | Nombre = ""
 Select 82 | Tipo Persona = [1] Persona | opciones: [1] Persona, [2] Empresa
 Checkbox 93 | Agente de retención = no
Table 75 | ID | Nombre
 Row 75:100 | 189 | Sabor / Criollo
`
	if got != want {
		t.Fatalf("render\n got:\n%s\nwant:\n%s", got, want)
	}
}

// With a floating layer open (raw DOM data-layer-id), the list layer behind it collapses to one line.
func TestFocusModeCollapsesBackgroundLayers(t *testing.T) {
	snapshot := `<div><Layer id="71" methods="open,close">
  <FilterInput id="72" label="Buscar" methods="setValue"/>
  <Table id="75"><table><TableBody id="75" methods="selectRow"><Row id="75:100"><td>1</td></Row><Row id="75:200"><td>2</td></Row></TableBody></table></Table>
</Layer>
<Layer id="76" label="Nuevo Cliente" methods="open,close"><Input id="79" label="Nombre" methods="setValue"/></Layer></div>`
	raw := `<div data-id="Layer:71"></div><div data-id="Layer:76" data-layer-id="11"></div>`
	got, err := Render(snapshot, raw, Options{Route: "/crm/customers"})
	if err != nil {
		t.Fatal(err)
	}
	want := `Page /crm/customers
Layer 71 | en segundo plano: Table 75 (2 filas)
Layer 76 | Nuevo Cliente
 Input 79 | Nombre = ""
`
	if got != want {
		t.Fatalf("render\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDeltaReportsChangedLinesOrFallsBackToFull(t *testing.T) {
	before := "Page /crm/customers | Clientes\nLayer 76 | Nuevo Cliente\n Input 79 | Nombre = \"\"\n Input 84 | Email = \"\"\n Input 87 | RUC = \"\"\n Button 77 | Guardar\n"
	after := "Page /crm/customers | Clientes\nLayer 76 | Nuevo Cliente\n Input 79 | Nombre = \"Pedro\"\n Input 84 | Email = \"\"\n Input 87 | RUC = \"\"\n Button 77 | Guardar\n Text | Falta el RUC\n"
	got, full := Delta(before, after)
	want := "Delta /crm/customers\n~ Input 79 | Nombre = \"Pedro\"\n+ Text | Falta el RUC\n"
	if full || got != want {
		t.Fatalf("delta (full=%v)\n got:\n%s\nwant:\n%s", full, got, want)
	}
	other := "Page /finance/loans | Préstamos\nButton 5 | Nuevo\n"
	if _, full := Delta(after, other); !full {
		t.Fatal("a different page must return the full outline")
	}
}

func TestMethods(t *testing.T) {
	cases := map[string]string{
		`Checkbox 12 | Activo = sí`:                  "click",
		`Select 5 | Moneda = [1] Soles | 2 opciones`: "search,select,getOptions",
		`TableBody 40 | {selectRow,click}`:           "selectRow,click",
		`Text | sin filas`:                           "",
	}
	for line, want := range cases {
		if got := strings.Join(Methods(line), ","); got != want {
			t.Errorf("Methods(%q) = %q, want %q", line, got, want)
		}
	}
}

// An editable grid registers its cells, not its rows: the rows still print, without an id.
func TestRenderEditableGridRows(t *testing.T) {
	snapshot := `<div><Table id="158"><table><thead><tr><th>#</th><th>Vence</th><th>Total</th><th></th></tr></thead><tbody>
<tr><td>1</td><td><CellInput id="158:102" value="01-01-2027" type="text" methods="setValue"/></td><td>3,499.09</td><td><Button id="159" label="Removes the installment." methods="click"/></td></tr>
</tbody></table></Table></div>`
	got, err := Render(snapshot, "", Options{Route: "/finance/loans"})
	if err != nil {
		t.Fatal(err)
	}
	want := "Page /finance/loans\nTable 158 | # | Vence | Total | \n Row | 1 | {CellInput 158:102 = \"01-01-2027\"} | 3,499.09 | {Button 159}\n"
	if got != want {
		t.Fatalf("render\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderShowsToasts(t *testing.T) {
	raw := `<div aria-live="polite"><button class="notify-toast bg-red-solid"><i></i> <span>Debe ingresar un email válido.</span> <span class="notify-toast-timer"></span></button></div>` +
		`<div data-id="Button:37" aria-label="Guardar">Guardar</div>`
	snapshot := `<div aria-live="polite"></div><Button id="37">Guardar</Button>`
	got, err := Render(snapshot, raw, Options{Route: "/crm/customers"})
	if err != nil {
		t.Fatal(err)
	}
	want := "Page /crm/customers\nToast | Debe ingresar un email válido.\nButton 37 | Guardar\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestRenderCountsTheRowsOfAVirtualizedTable(t *testing.T) {
	rows := ""
	for index := 1; index <= 6; index++ {
		rows += fmt.Sprintf(`<Row id="211:%d00"><td>%d</td></Row>`, index, index)
	}
	got, err := Render(`<Table id="211" rows-count="4120">`+rows+`</Table>`, "", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "More | +4115 filas") {
		t.Fatalf("want the held rows counted, got\n%s", got)
	}
}
