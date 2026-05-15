package agent

import (
	"strings"
	"testing"
)

func TestParsePageHTML_StripsAttributesExceptAriaAndRole(t *testing.T) {
	in := `<div class="x" id="y" role="button" aria-label="ok" onclick="boom()">hi</div>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `role="button"`) || !strings.Contains(out, `aria-label="ok"`) {
		t.Fatalf("expected role + aria-label preserved, got:\n%s", out)
	}
	for _, banned := range []string{"class=", "id=", "onclick="} {
		if strings.Contains(out, banned) {
			t.Fatalf("expected %q stripped, got:\n%s", banned, out)
		}
	}
}

func TestParsePageHTML_DropsEmptyChains(t *testing.T) {
	in := `<section><div><div><div></div></div></div></section>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty output, got:\n%s", out)
	}
}

func TestParsePageHTML_CollapsesWrapperChain(t *testing.T) {
	in := `<div><div><div><div>hello</div></div></div></div>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Expect at most one wrapper between the outer text-bearing context
	// (the implicit body) and the innermost element with text.
	if strings.Count(out, "<div>") > 2 {
		t.Fatalf("expected at most 2 <div> wrappers, got:\n%s", out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected text preserved, got:\n%s", out)
	}
}

func TestParsePageHTML_KeepsOneWrapperBetweenTextLayers(t *testing.T) {
	in := `<div>hello<div><div><div>world</div></div></div></div>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hello") || !strings.Contains(out, "world") {
		t.Fatalf("expected both text nodes preserved, got:\n%s", out)
	}
	// Between hello (in outer div) and world (innermost) only one wrapper allowed.
	// Outer div + 1 wrapper + innermost div = 3 opening div tags total.
	if strings.Count(out, "<div") != 3 {
		t.Fatalf("expected exactly 3 <div tags, got %d:\n%s", strings.Count(out, "<div"), out)
	}
}

func TestParsePageHTML_DropsScriptsAndComments(t *testing.T) {
	in := `<div>keep<script>evil()</script><!-- gone --><style>.x{}</style></div>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"script", "style", "evil", "gone"} {
		if strings.Contains(out, banned) {
			t.Fatalf("expected %q stripped, got:\n%s", banned, out)
		}
	}
	if !strings.Contains(out, "keep") {
		t.Fatalf("expected text preserved, got:\n%s", out)
	}
}

func TestParsePageHTML_IndentsNestedContent(t *testing.T) {
	in := `<section><h1>Title</h1><p>Body text</p></section>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n  <h1>") {
		t.Fatalf("expected indented children, got:\n%s", out)
	}
}

func TestParsePageHTML_TableSuppressesRoutingMethods(t *testing.T) {
	in := `
<div data-id="Table:38">
  <table>
    <tbody>
      <tr data-id="TableRow:38:100"><td>Sara Quintana</td></tr>
    </tbody>
  </table>
</div>`
	components := []AgentComponentInfo{{
		ID:      38,
		Type:    "Table",
		Methods: []string{"select", "setValueChild", "searchChild", "getOptionsChild"},
	}}
	out, err := ParsePageHTML(in, components)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<Table id="38"`) {
		t.Fatalf("expected Table tag, got:\n%s", out)
	}
	// Methods belong on the row, not the table.
	if strings.Contains(out, `<Table id="38" methods=`) {
		t.Fatalf("expected no methods= on Table, got:\n%s", out)
	}
	if !strings.Contains(out, `<TableRow id="38:100"`) {
		t.Fatalf("expected composite TableRow id, got:\n%s", out)
	}
	if !strings.Contains(out, `methods="select"`) {
		t.Fatalf("expected TableRow methods=\"select\", got:\n%s", out)
	}
}

func TestParsePageHTML_KeepsEmptyCellsForColumnAlignment(t *testing.T) {
	// A row with one missing column — the empty <td> must be preserved so the
	// agent can still tell which column the populated cells belong to.
	in := `
<table>
  <thead>
    <tr><th>A</th><th>B</th><th>C</th></tr>
  </thead>
  <tbody>
    <tr><td>row1-a</td><td></td><td>row1-c</td></tr>
  </tbody>
</table>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(out, "<td>") != 3 {
		t.Fatalf("expected 3 <td> tags preserved, got:\n%s", out)
	}
	if !strings.Contains(out, "<td></td>") {
		t.Fatalf("expected empty <td></td> retained, got:\n%s", out)
	}
}

func TestParsePageHTML_DropsFullyEmptyRows(t *testing.T) {
	// A row whose every cell is empty has no text descendants at all and
	// gets pruned, taking its cells with it. Only the populated row's content
	// survives — everything else is wrapper chrome the collapser flattens.
	in := `
<table>
  <thead><tr><th>A</th><th>B</th></tr></thead>
  <tbody>
    <tr><td>kept</td><td>val</td></tr>
    <tr><td></td><td></td></tr>
  </tbody>
</table>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "<td></td>") {
		t.Fatalf("expected empty cells from the dropped row to be gone, got:\n%s", out)
	}
	// Two cells from the populated row + two header cells = 4 cell tags total.
	cellCount := strings.Count(out, "<td>") + strings.Count(out, "<th>")
	if cellCount != 4 {
		t.Fatalf("expected 4 cell tags (2 header + 2 populated row), got %d:\n%s", cellCount, out)
	}
}

func TestParsePageHTML_CellClickStaysOnTd(t *testing.T) {
	// onCellClick puts the agent handle on the <td> itself so the column slot
	// stays visible in the row. id/methods replace the raw data-* attrs.
	in := `
<div data-id="Table:11">
  <table>
    <tbody>
      <tr data-id="TableRow:11:100">
        <td>Producto</td>
        <td data-id="11:201" data-cell-type="CellClick">5</td>
        <td>6</td>
      </tr>
    </tbody>
  </table>
</div>`
	components := []AgentComponentInfo{{
		ID:      11,
		Type:    "Table",
		Methods: []string{"select", "clickChild"},
	}}
	out, err := ParsePageHTML(in, components)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<td id="11:201" methods="click">5</td>`) {
		t.Fatalf("expected <td id=\"11:201\" methods=\"click\">5</td>, got:\n%s", out)
	}
	// Plain cells should NOT pick up agent attrs.
	if strings.Contains(out, `<td id=`) && !strings.Contains(out, `<td id="11:201"`) {
		t.Fatalf("plain cells should not have id, got:\n%s", out)
	}
	// Raw data-* form is replaced.
	if strings.Contains(out, "data-cell-type") || strings.Contains(out, `data-id="11:201"`) {
		t.Fatalf("expected data-* attrs replaced by id/methods, got:\n%s", out)
	}
	// Table itself must not advertise clickChild.
	if strings.Contains(out, "clickChild") {
		t.Fatalf("expected clickChild suppressed on Table, got:\n%s", out)
	}
}

func TestParsePageHTML_DropsMenuRoots(t *testing.T) {
	in := `
<div data-menu-root="true" role="navigation" aria-label="Main navigation">
  <button data-id="MenuHeader:1">CONFIG</button>
  <button data-id="MenuOption:5" data-label="Cambios" data-value="/x">Cambios</button>
</div>
<div data-menu-root="true" role="dialog" aria-modal="true">
  <button data-id="MenuOption:6" data-label="Compras" data-value="/y">Compras</button>
</div>
<main>kept content</main>`
	out, err := ParsePageHTML(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "kept content") {
		t.Fatalf("expected non-menu content preserved, got:\n%s", out)
	}
	for _, banned := range []string{"MenuHeader", "MenuOption", "Main navigation", "Cambios", "Compras"} {
		if strings.Contains(out, banned) {
			t.Fatalf("expected %q stripped from menu drop, got:\n%s", banned, out)
		}
	}
}

func TestParsePageHTML_CardPreservesChildren(t *testing.T) {
	in := `
<div>
  <div data-id="Card:19" aria-label="Zumo de naranja">
    <div>
      <span>Zumo de naranja</span>
      <span>(Botella 75cl)</span>
    </div>
    <div class="empty-separator"></div>
    <button>2</button>
    <button>5</button>
    <div>7.50</div>
  </div>
</div>`
	components := []AgentComponentInfo{{
		ID:      19,
		Type:    "Card",
		Label:   "Zumo de naranja",
		Methods: []string{"click"},
	}}
	out, err := ParsePageHTML(in, components)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<Card id="19"`) {
		t.Fatalf("expected Card tag, got:\n%s", out)
	}
	if strings.Contains(out, `<Card id="19" label="Zumo de naranja" methods="click"/>`) {
		t.Fatalf("expected Card NOT to self-close, got:\n%s", out)
	}
	// Structure preserved: <span> and <button> tags survive.
	for _, needle := range []string{"<span>", "<button>"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected %q inside Card, got:\n%s", needle, out)
		}
	}
	// Text content preserved.
	for _, needle := range []string{"Zumo de naranja", "Botella 75cl", "2", "5", "7.50"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected %q in Card body, got:\n%s", needle, out)
		}
	}
	// Empty separator div should be pruned by removeEmptyElements.
	if strings.Contains(out, "<div></div>") {
		t.Fatalf("expected empty <div></div> to be pruned, got:\n%s", out)
	}
	if !strings.Contains(out, "</Card>") {
		t.Fatalf("expected closing </Card>, got:\n%s", out)
	}
}

func TestParsePageHTML_PageViewsOptionsExposeSelect(t *testing.T) {
	// PageViews renders as a structural wrapper (no id, no methods). Its
	// option buttons use composite ids "<pageViewsID>:<optionID>" so the
	// agent can address each option individually while routing flows
	// through the PageViews handle's `select` method.
	in := `
<div data-id="PageViews:7">
  <button data-id="Option:7:1" aria-label="Ventas" data-selected="true">Ventas</button>
  <button data-id="Option:7:2" aria-label="Configuración">Configuración</button>
</div>`
	components := []AgentComponentInfo{{
		ID:      7,
		Type:    "PageViews",
		Methods: []string{"select"},
	}}
	out, err := ParsePageHTML(in, components)
	if err != nil {
		t.Fatal(err)
	}
	// PageViews tag carries no id/methods/label.
	if strings.Contains(out, `<PageViews id=`) || strings.Contains(out, `<PageViews methods=`) {
		t.Fatalf("expected bare <PageViews> tag, got:\n%s", out)
	}
	if !strings.Contains(out, "<PageViews>") {
		t.Fatalf("expected <PageViews> opening tag, got:\n%s", out)
	}
	if !strings.Contains(out, "</PageViews>") {
		t.Fatalf("expected closing </PageViews>, got:\n%s", out)
	}
	// Composite Option ids surface for routing.
	if !strings.Contains(out, `<Option id="7:1" aria-label="Ventas" selected methods="select">Ventas</Option>`) {
		t.Fatalf("expected selected Option with composite id and methods=\"select\", got:\n%s", out)
	}
	if !strings.Contains(out, `<Option id="7:2" aria-label="Configuración" methods="select">Configuración</Option>`) {
		t.Fatalf("expected unselected Option with composite id and methods=\"select\", got:\n%s", out)
	}
}

func TestParsePageHTML_OptionDefaultsToRemoveOutsideSelectParent(t *testing.T) {
	// SearchCard chip is an Option marker whose parent exposes "remove" — the
	// marker should keep its default "remove" verb.
	in := `
<div data-id="SearchCard:4">
  <button data-id="Option:11">Foo</button>
</div>`
	components := []AgentComponentInfo{{
		ID:      4,
		Type:    "SearchCard",
		Methods: []string{"remove"},
	}}
	out, err := ParsePageHTML(in, components)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `methods="remove"`) {
		t.Fatalf("expected Option to keep methods=\"remove\", got:\n%s", out)
	}
}

func TestParsePageHTML_CellMethodsOnCellTags(t *testing.T) {
	in := `
<div data-id="Table:38">
  <table>
    <tbody>
      <tr data-id="TableRow:38:100">
        <td>
          <div data-id="38:101" data-cell-type="CellInput" data-value="Sara" data-type="text"></div>
        </td>
        <td>
          <div data-id="38:102" data-cell-type="CellSelect" data-value="[1] Persona"></div>
        </td>
      </tr>
    </tbody>
  </table>
</div>`
	components := []AgentComponentInfo{{
		ID:      38,
		Type:    "Table",
		Methods: []string{"select", "setValueChild", "searchChild", "getOptionsChild"},
	}}
	out, err := ParsePageHTML(in, components)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<CellInput id="38:101"`) {
		t.Fatalf("expected CellInput tag, got:\n%s", out)
	}
	if !strings.Contains(out, `<CellInput id="38:101" value="Sara" type="text" methods="setValue"/>`) {
		t.Fatalf("expected CellInput with methods=\"setValue\", got:\n%s", out)
	}
	if !strings.Contains(out, `<CellSelect id="38:102" value="[1] Persona" methods="search,select,getOptions"/>`) {
		t.Fatalf("expected CellSelect with methods=\"search,select,getOptions\", got:\n%s", out)
	}
}

func TestParsePageHTML_SelectRendersOptionsCountWithoutInlineOptions(t *testing.T) {
	// Long option lists: the handle carries options-count but no inline
	// options. The agent uses search / getOptions to drill down.
	in := `
<div data-id="Select:9" data-value="[3] PEN" data-label="Moneda" data-type="other" data-options-count="120">
  <input />
</div>`
	components := []AgentComponentInfo{{
		ID:      9,
		Type:    "Select",
		Label:   "Moneda",
		Methods: []string{"search", "select", "getOptions"},
	}}
	out, err := ParsePageHTML(in, components)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `options-count="120"`) {
		t.Fatalf("expected options-count attribute, got:\n%s", out)
	}
	if strings.Contains(out, `options="`) {
		t.Fatalf("expected no inline options for long list, got:\n%s", out)
	}
}

func TestParsePageHTML_SelectRendersInlineOptionsWhenAttached(t *testing.T) {
	// Short option lists: the frontend probe attaches Options to the
	// AgentComponentInfo and the renderer inlines them with [id] label|… .
	in := `
<div data-id="Select:9" data-value="" data-label="Moneda" data-type="other" data-options-count="3">
  <input />
</div>`
	components := []AgentComponentInfo{{
		ID:      9,
		Type:    "Select",
		Label:   "Moneda",
		Methods: []string{"search", "select", "getOptions"},
		Options: []AgentOption{
			{ID: 1, Value: "PEN"},
			{ID: 2, Value: "USD"},
			{ID: 3, Value: "EUR"},
		},
	}}
	out, err := ParsePageHTML(in, components)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `options-count="3"`) {
		t.Fatalf("expected options-count=3, got:\n%s", out)
	}
	if !strings.Contains(out, `options="[1] PEN|[2] USD|[3] EUR"`) {
		t.Fatalf("expected inline options attribute, got:\n%s", out)
	}
}

func TestParsePageHTML_OptionsStripIsStructural(t *testing.T) {
	// OptionsStrip is structural: the parent tag carries no id/methods/label —
	// the agent addresses each Option by its composite "<stripID>:<optID>" id
	// and the strip's select method strips the parent prefix.
	in := `
<div data-id="OptionsStrip:30">
  <button data-id="Option:30:1" data-selected="true">Productos</button>
  <button data-id="Option:30:2">Categorías</button>
  <button data-id="Option:30:3">Marcas</button>
</div>`
	components := []AgentComponentInfo{{
		ID:      30,
		Type:    "OptionsStrip",
		Methods: []string{"select"},
	}}
	out, err := ParsePageHTML(in, components)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, `<OptionsStrip id=`) || strings.Contains(out, `<OptionsStrip methods=`) {
		t.Fatalf("expected bare <OptionsStrip> tag without id/methods, got:\n%s", out)
	}
	if !strings.Contains(out, "<OptionsStrip>") || !strings.Contains(out, "</OptionsStrip>") {
		t.Fatalf("expected open/close <OptionsStrip> tags, got:\n%s", out)
	}
	if !strings.Contains(out, `<Option id="30:1" selected methods="select">Productos</Option>`) {
		t.Fatalf("expected selected Option with composite id and methods=\"select\", got:\n%s", out)
	}
	if !strings.Contains(out, `<Option id="30:2" methods="select">Categorías</Option>`) {
		t.Fatalf("expected unselected Option with composite id and methods=\"select\", got:\n%s", out)
	}
}

func TestParsePageHTML_SelectInlineOptionsEscapeSeparator(t *testing.T) {
	// Pipes in labels collide with the option separator — they get rewritten
	// to "/" so the inline list stays unambiguous for the agent parser.
	in := `<div data-id="Select:9" data-options-count="2"><input /></div>`
	components := []AgentComponentInfo{{
		ID:      9,
		Type:    "Select",
		Methods: []string{"select"},
		Options: []AgentOption{
			{ID: 1, Value: "Foo|Bar"},
			{ID: 2, Value: "Baz"},
		},
	}}
	out, err := ParsePageHTML(in, components)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `options="[1] Foo/Bar|[2] Baz"`) {
		t.Fatalf("expected pipe in label escaped to /, got:\n%s", out)
	}
}
