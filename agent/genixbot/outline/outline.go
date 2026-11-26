// Package outline renders a Genix UI page as an indented outline for small models.
//
// It reads two inputs of one page state: the compact snapshot produced by genix-agent's
// ParsePageHTML (component types, ids, labels, values, options, rows) and the raw DOM the browser
// sent (only for table header structure, which the snapshot drops: colspan/rowspan).
//
// Output rules (see COMPACT_FORMAT.md in this directory):
//   - one line per registered component: `Type id | label = "value"`; hierarchy by indentation;
//   - layout tags (div, span, header, table parts) are transparent; their text becomes `"text"` lines;
//   - unregistered buttons, inputs and images are dropped (the agent cannot act on them);
//   - methods are omitted when they are the type's default;
//   - tables print flattened leaf headers once and positional rows, capped at MaxRows;
//   - focus mode: while a floating layer is open (raw DOM data-layer-id), every other container is
//     collapsed to one "en segundo plano" line;
//   - " *" marks the selected/active item; indentation is one space per level (cheapest in LFM3 tokens).
package outline

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// Options tune the rendering; zero values use the defaults below.
type Options struct {
	Route   string // shown in the first line
	MaxRows int    // rows printed per table (default 5; the model filters the table to see others)
	// OptionLabels holds each Select's inline options as the frontend registered them
	// (component id → "[id] label" list). The snapshot rewrites "English|Español" option labels
	// to "English/Español", which can no longer be split safely, so the originals win.
	OptionLabels map[string][]string
}

// Component is the part of a registered component handle (genix-agent's AgentComponentInfo) the
// outline needs: its inline options with their original "English|Español" labels.
type Component struct {
	ID      int
	Options []ComponentOption
}

// ComponentOption mirrors genix-agent's AgentOption.
type ComponentOption struct {
	ID    any
	Value any
}

// OptionLabelsFromComponents builds Options.OptionLabels from the registered components.
func OptionLabelsFromComponents(components []Component) map[string][]string {
	labels := map[string][]string{}
	for _, component := range components {
		for _, option := range component.Options {
			labels[fmt.Sprint(component.ID)] = append(labels[fmt.Sprint(component.ID)],
				fmt.Sprintf("[%v] %s", option.ID, spanishLabel(fmt.Sprint(option.Value))))
		}
	}
	return labels
}

type node struct {
	tag      string
	attrs    map[string]string
	children []*node
	text     string // for text nodes (tag == "")
}

// defaultMethods are the verbs a model learns per type; only deviations are printed.
var defaultMethods = map[string]string{
	"Input": "setValue", "FilterInput": "setValue", "DateInput": "setValue", "CellInput": "setValue",
	"Button": "click", "Card": "click", "Checkbox": "click",
	"Select": "search,select,getOptions", "CellSelect": "search,select,getOptions",
	"MultiSelect": "search,select,remove,getOptions",
	"Layer":       "open,close", "LayerStatic": "open,close", "ButtonLayer": "click,open,close",
	"Option": "select", "TableBody": "selectRow", "ImageUploader": "click,deleted",
	"CheckboxOptions": "select,remove",
}

// valueTypes always print a value, `= ""` when empty, so the model sees what is still unfilled.
var valueTypes = map[string]bool{
	"Input": true, "FilterInput": true, "DateInput": true, "CellInput": true,
	"Select": true, "CellSelect": true, "MultiSelect": true, "Checkbox": true,
}

// shellTypes are app-wide overlays that only matter when they hold something.
var shellTypes = map[string]bool{"TopLayerSelector": true, "TopLayerDatePicker": true}

// longTextLine is the length above which a text block is split into its items.
const longTextLine = 160

var unspacedBilingualSeparator = regexp.MustCompile(`(\S)\|(\S)`)

// Render converts one page state to the compact outline.
func Render(snapshotHTML, rawHTML string, options Options) (string, error) {
	if options.MaxRows <= 0 {
		options.MaxRows = 5
	}
	root, err := parseSnapshot(snapshotHTML)
	if err != nil {
		return "", err
	}
	renderer := &renderer{options: options, tableHeaders: tableHeadersFromRaw(rawHTML), focusedLayers: floatingLayersFromRaw(rawHTML)}
	title := pageTitle(root)
	// The route takes the id slot: "Page /crm/customers | Clientes".
	pageLine := "Page"
	if options.Route != "" {
		pageLine += " " + options.Route
	}
	if title != "" {
		pageLine += " | " + escapeCell(title)
	}
	renderer.lines = append(renderer.lines, pageLine)
	renderer.cardPositions = cardPositions(root)
	renderer.renderChildren(root, 0)
	return strings.Join(renderer.lines, "\n") + "\n", nil
}

type renderer struct {
	options      Options
	tableHeaders map[string][]string
	lines        []string
	// enclosingLabel is the label of the component being expanded; a text line that only repeats
	// it (a layer title rendered again inside the layer) is skipped.
	enclosingLabel string
	// focusedLayers are the open floating layers; when any exists, containers outside them collapse.
	focusedLayers map[string]bool
	insideFocus   bool
	// cardPositions: each Card's index and group size among the Cards of its nearest component
	// ancestor, so long card grids (point of sale) are capped like table rows.
	cardPositions map[*node][2]int
}

func cardPositions(root *node) map[*node][2]int {
	positions := map[*node][2]int{}
	var walk func(n *node, group *[]*node)
	walk = func(n *node, group *[]*node) {
		for _, child := range n.children {
			if child.tag == "Card" {
				*group = append(*group, child)
			}
			if isComponent(child) {
				own := []*node{}
				walk(child, &own)
				record(positions, own)
			} else {
				walk(child, group)
			}
		}
	}
	top := []*node{}
	walk(root, &top)
	record(positions, top)
	return positions
}

func record(positions map[*node][2]int, group []*node) {
	for index, card := range group {
		positions[card] = [2]int{index, len(group)}
	}
}

func (r *renderer) emitText(depth int, text string) {
	if text != "" && text != r.enclosingLabel {
		r.emit(depth, "Text | "+escapeCell(text))
	}
}

func (r *renderer) emit(depth int, line string) {
	r.lines = append(r.lines, strings.Repeat(" ", depth)+line)
}

func (r *renderer) renderChildren(parent *node, depth int) {
	for _, child := range parent.children {
		r.renderNode(child, depth)
	}
}

func (r *renderer) renderNode(n *node, depth int) {
	if n.tag == "" {
		r.emitText(depth, cleanText(n.text))
		return
	}
	if isDropped(n) {
		return
	}
	if n.tag == "table" {
		r.renderPlainTable(n, depth)
		return
	}
	if isComponent(n) {
		r.renderComponent(n, depth)
		return
	}
	// A layout element without components inside collapses to one text line ("73 registros").
	if !containsComponent(n) && len(findAll(n, "table")) == 0 {
		r.renderTextBlock(n, depth)
		return
	}
	r.renderChildren(n, depth)
}

func (r *renderer) renderComponent(n *node, depth int) {
	componentType := n.tag
	if shellTypes[componentType] && len(n.children) == 0 && n.attrs["value"] == "" {
		return
	}
	switch componentType {
	case "Table":
		r.renderTable(n, depth)
		return
	case "TableBody":
		// Reached only for a body outside a Table wrapper; rows are rendered by renderTable.
		r.renderTable(n, depth)
		return
	case "OptionsStrip", "PageViews":
		r.renderOptionGroup(n, depth)
		return
	}

	if position, isCard := r.cardPositions[n]; isCard && position[0] >= r.options.MaxRows {
		if position[0] == position[1]-1 {
			r.emit(depth, fmt.Sprintf("More | +%d tarjetas", position[1]-r.options.MaxRows))
		}
		return
	}
	if r.collapsesInFocus(n) {
		r.emit(depth, componentType+" "+n.attrs["id"]+" | en segundo plano"+backgroundSummary(n))
		return
	}
	enteringFocus := r.focusedLayers[n.attrs["id"]] && strings.HasPrefix(componentType, "Layer")

	line := componentType + " " + n.attrs["id"]
	label := spanishLabel(n.attrs["label"])
	ownText := ""
	if !containsComponent(n) {
		ownText = cleanText(innerText(n))
	}
	if label == "" {
		label, ownText = ownText, ""
	}
	// Cards repeat their label as visible text and show some figures twice ("Stock 327 327").
	ownText = dropRepeatedTokens(strings.TrimSpace(strings.TrimPrefix(ownText, cleanText(label))))
	if label != "" {
		line += " | " + label
	}
	if ownText != "" && ownText != label {
		line += " | " + ownText
	}
	if _, selected := n.attrs["selected"]; selected {
		line += " *"
	}
	line += valueSuffix(n)
	line += r.optionsSuffix(n)
	if methods := n.attrs["methods"]; methods != "" && methods != defaultMethods[componentType] {
		line += " {" + methods + "}"
	}
	r.emit(depth, line)
	if containsComponent(n) {
		outerLabel, outerFocus := r.enclosingLabel, r.insideFocus
		r.enclosingLabel = label
		r.insideFocus = r.insideFocus || enteringFocus
		r.renderChildren(n, depth+1)
		r.enclosingLabel, r.insideFocus = outerLabel, outerFocus
	}
}

// collapsesInFocus: with a floating layer open, a container that neither is nor holds a focused
// layer is background. Leaf components (buttons, filters) stay, so the model can still act on them.
func (r *renderer) collapsesInFocus(n *node) bool {
	if len(r.focusedLayers) == 0 || r.insideFocus || r.focusedLayers[n.attrs["id"]] || !containsComponent(n) {
		return false
	}
	for _, descendant := range allDescendants(n) {
		if r.focusedLayers[descendant.attrs["id"]] {
			return false
		}
	}
	return true
}

// backgroundSummary keeps what the model may need to come back to: tables and their row counts.
func backgroundSummary(n *node) string {
	parts := []string{}
	for _, table := range findAll(n, "Table") {
		parts = append(parts, fmt.Sprintf("Table %s (%d filas)", table.attrs["id"], len(findAll(table, "Row"))))
	}
	if len(parts) == 0 {
		return ""
	}
	return ": " + strings.Join(parts, ", ")
}

// renderTextBlock prints a component-free block as one text line, or, when it is a long list of
// items (a product picker), one line per item capped like table rows.
func (r *renderer) renderTextBlock(n *node, depth int) {
	text := cleanText(innerText(n))
	items := []*node{}
	for _, child := range n.children {
		if cleanText(innerText(child)) != "" {
			items = append(items, child)
		}
	}
	if len(text) <= longTextLine || len(items) == 0 {
		r.emitText(depth, text)
		return
	}
	// Wrapper chains (div > div > list) are walked down to the element that holds the items.
	if len(items) == 1 {
		if items[0].tag == "" {
			r.emitText(depth, text)
			return
		}
		r.renderTextBlock(items[0], depth)
		return
	}
	for index, item := range items {
		if index == r.options.MaxRows {
			r.emit(depth, fmt.Sprintf("More | +%d elementos", len(items)-index))
			return
		}
		r.renderTextBlock(item, depth)
	}
}

// renderPlainTable shows an unregistered <table>: readable, but its rows cannot be selected, so
// they carry no ids.
func (r *renderer) renderPlainTable(n *node, depth int) {
	headers := []string{}
	for _, th := range findAll(n, "th") {
		headers = append(headers, escapeCell(cleanText(innerText(th))))
	}
	r.emit(depth, "Table | "+strings.Join(headers, " | "))
	bodyRows := []*node{}
	for _, tr := range findAll(n, "tr") {
		if cells := directCells(tr); len(cells) > 0 && cells[0].tag != "th" {
			bodyRows = append(bodyRows, tr)
		}
	}
	rows := 0
	for _, tr := range bodyRows {
		cells := directCells(tr)
		if rows == r.options.MaxRows {
			r.emit(depth+1, fmt.Sprintf("More | +%d filas", len(bodyRows)-rows))
			break
		}
		values := []string{}
		for _, cell := range cells {
			values = append(values, renderCell(cell))
		}
		r.emit(depth+1, "Row | "+strings.Join(values, " | "))
		rows++
	}
	if rows == 0 {
		r.emit(depth+1, "Text | sin filas")
	}
}

func (r *renderer) renderOptionGroup(n *node, depth int) {
	parts := []string{}
	for _, option := range findAll(n, "Option") {
		text := cleanText(innerText(option))
		if text == "" {
			text = spanishLabel(option.attrs["aria-label"])
		}
		part := "[" + option.attrs["id"] + "] " + text
		if _, selected := option.attrs["selected"]; selected {
			part += " *"
		}
		parts = append(parts, part)
	}
	r.emit(depth, n.tag+" | "+strings.Join(parts, " · "))
}

func (r *renderer) renderTable(n *node, depth int) {
	tableID := n.attrs["id"]
	headers := r.tableHeaders[tableID]
	if len(headers) == 0 {
		for _, th := range findAll(n, "th") {
			headers = append(headers, cleanText(innerText(th)))
		}
	}
	line := "Table " + tableID
	if len(headers) > 0 {
		line += " | " + strings.Join(escapeCells(headers), " | ")
	}
	r.emit(depth, line)

	rows := findAll(n, "Row")
	for index, row := range rows {
		if index == r.options.MaxRows {
			r.emit(depth+1, fmt.Sprintf("More | +%d filas", len(rows)-index))
			break
		}
		cells := []string{}
		for _, cell := range directCells(row) {
			cells = append(cells, renderCell(cell))
		}
		rowLine := "Row " + row.attrs["id"]
		if _, selected := row.attrs["selected"]; selected {
			rowLine += " *"
		}
		r.emit(depth+1, rowLine+" | "+strings.Join(cells, " | "))
	}
	if len(rows) == 0 {
		r.emit(depth+1, "Text | sin filas")
	}
}

// renderCell shows plain text, or the editable component inside the cell.
func renderCell(cell *node) string {
	for _, child := range allDescendants(cell) {
		if isComponent(child) {
			return "{" + child.tag + " " + child.attrs["id"] + valueSuffix(child) + "}"
		}
	}
	return escapeCell(cleanText(innerText(cell)))
}

func valueSuffix(n *node) string {
	value, hasValue := n.attrs["value"]
	if !valueTypes[n.tag] {
		if hasValue && value != "" && value != "closed" {
			return " [" + localizeState(value) + "]"
		}
		return ""
	}
	if n.tag == "Checkbox" {
		if value == "1" || value == "true" {
			return " = sí"
		}
		return " = no"
	}
	value = strings.TrimSpace(value)
	// Selected options arrive as "[id] text"; keep the id so the model can reason about it.
	if strings.HasPrefix(value, "[") {
		if idEnd := strings.Index(value, "] "); idEnd > 0 {
			return " = " + value[:idEnd+2] + spanishLabel(value[idEnd+2:])
		}
		return " = " + value
	}
	return " = " + quote(value)
}

func (r *renderer) optionsSuffix(n *node) string {
	count := n.attrs["options-count"]
	if registered := r.options.OptionLabels[n.attrs["id"]]; len(registered) > 0 && n.attrs["options"] != "" {
		return " | opciones: " + strings.Join(registered, ", ")
	}
	if list := n.attrs["options"]; list != "" {
		options := strings.Split(list, "|")
		for index := range options {
			options[index] = strings.TrimSpace(options[index])
		}
		return " | opciones: " + strings.Join(options, ", ")
	}
	if count != "" && count != "0" {
		return " | " + count + " opciones"
	}
	return ""
}

func localizeState(value string) string {
	switch value {
	case "open":
		return "abierto"
	case "empty":
		return "vacío"
	}
	return value
}

// ── table headers from the raw DOM ──────────────────────────────────────────

// tableHeadersFromRaw flattens every registered table's <thead> into leaf column names, joining
// grouped headers: a "Stock" cell spanning "Simple | Total" yields "Stock › Simple", "Stock › Total".
func tableHeadersFromRaw(rawHTML string) map[string][]string {
	headersByTable := map[string][]string{}
	document, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return headersByTable
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if dataID := htmlAttr(n, "data-id"); strings.HasPrefix(dataID, "Table:") {
				if thead := findHTML(n, "thead"); thead != nil {
					headersByTable[strings.TrimPrefix(dataID, "Table:")] = FlattenHeaderGrid(headerRows(thead))
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	return headersByTable
}

// floatingLayersFromRaw finds the open floating layers: Genix renders them with data-layer-id
// (the page's own Layer containers and the fixed LayerStatic panels carry none).
func floatingLayersFromRaw(rawHTML string) map[string]bool {
	layers := map[string]bool{}
	document, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return layers
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && htmlAttr(n, "data-layer-id") != "" {
			if dataID := htmlAttr(n, "data-id"); strings.HasPrefix(dataID, "Layer:") {
				layers[strings.TrimPrefix(dataID, "Layer:")] = true
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	return layers
}

// HeaderCell is one <th> with its spans.
type HeaderCell struct {
	Text             string
	ColSpan, RowSpan int
}

func headerRows(thead *html.Node) [][]HeaderCell {
	rows := [][]HeaderCell{}
	for tr := thead.FirstChild; tr != nil; tr = tr.NextSibling {
		if tr.Type != html.ElementNode || tr.Data != "tr" {
			continue
		}
		row := []HeaderCell{}
		for th := tr.FirstChild; th != nil; th = th.NextSibling {
			if th.Type != html.ElementNode || (th.Data != "th" && th.Data != "td") {
				continue
			}
			row = append(row, HeaderCell{
				Text: cleanText(htmlText(th)), ColSpan: atoiOrOne(htmlAttr(th, "colspan")), RowSpan: atoiOrOne(htmlAttr(th, "rowspan")),
			})
		}
		rows = append(rows, row)
	}
	return rows
}

// FlattenHeaderGrid lays the header cells on a grid (honoring colspan/rowspan) and returns one name
// per leaf column, made of the distinct non-empty labels stacked above it.
func FlattenHeaderGrid(rows [][]HeaderCell) []string {
	grid := map[[2]int]string{}
	columnCount := 0
	for rowIndex, row := range rows {
		column := 0
		for _, cell := range row {
			for hasCell(grid, rowIndex, column) {
				column++
			}
			for rowOffset := 0; rowOffset < cell.RowSpan; rowOffset++ {
				for colOffset := 0; colOffset < cell.ColSpan; colOffset++ {
					text := cell.Text
					if text == "" {
						text = " " // marks the slot as taken without contributing a label
					}
					grid[[2]int{rowIndex + rowOffset, column + colOffset}] = text
				}
			}
			column += cell.ColSpan
			columnCount = max(columnCount, column)
		}
	}
	headers := make([]string, columnCount)
	for column := 0; column < columnCount; column++ {
		parts := []string{}
		for rowIndex := range rows {
			text := strings.TrimSpace(grid[[2]int{rowIndex, column}])
			if text != "" && (len(parts) == 0 || parts[len(parts)-1] != text) {
				parts = append(parts, text)
			}
		}
		headers[column] = strings.Join(parts, " › ")
	}
	return headers
}

func hasCell(grid map[[2]int]string, row, column int) bool {
	_, taken := grid[[2]int{row, column}]
	return taken
}

// ── snapshot parsing ────────────────────────────────────────────────────────

// parseSnapshot reads the snapshot as lenient XML: Go's HTML parser would lowercase the component
// tags and move <TableBody>/<Row> out of <table> (foster parenting).
func parseSnapshot(snapshotHTML string) (*node, error) {
	decoder := xml.NewDecoder(strings.NewReader(snapshotHTML))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	root := &node{tag: "#root"}
	stack := []*node{root}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse snapshot: %w", err)
		}
		parent := stack[len(stack)-1]
		switch typed := token.(type) {
		case xml.StartElement:
			element := &node{tag: typed.Name.Local, attrs: map[string]string{}}
			for _, attribute := range typed.Attr {
				element.attrs[attribute.Name.Local] = attribute.Value
			}
			parent.children = append(parent.children, element)
			stack = append(stack, element)
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if strings.TrimSpace(string(typed)) != "" {
				parent.children = append(parent.children, &node{text: string(typed)})
			}
		}
	}
	return root, nil
}

func isComponent(n *node) bool {
	return n.tag != "" && unicode.IsUpper(rune(n.tag[0]))
}

// isDropped removes what the agent cannot act on and what only repeats app chrome.
func isDropped(n *node) bool {
	switch n.tag {
	case "header", "img", "button", "input", "i", "svg":
		return true
	}
	_, live := n.attrs["aria-live"]
	return live
}

func containsComponent(n *node) bool {
	for _, child := range n.children {
		if isComponent(child) || (child.tag != "" && !isDropped(child) && containsComponent(child)) {
			return true
		}
	}
	return false
}

func pageTitle(root *node) string {
	for _, header := range findAll(root, "header") {
		for _, span := range findAll(header, "span") {
			if text := cleanText(innerText(span)); text != "" && text != "Cargando..." {
				return text
			}
		}
	}
	return ""
}

func innerText(n *node) string {
	if n.tag == "" {
		return n.text
	}
	if isDropped(n) {
		return ""
	}
	parts := []string{}
	for _, child := range n.children {
		if text := strings.TrimSpace(innerText(child)); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func findAll(n *node, tag string) []*node {
	found := []*node{}
	for _, descendant := range allDescendants(n) {
		if descendant.tag == tag {
			found = append(found, descendant)
		}
	}
	return found
}

func allDescendants(n *node) []*node {
	result := []*node{}
	for _, child := range n.children {
		result = append(result, child)
		result = append(result, allDescendants(child)...)
	}
	return result
}

func directCells(row *node) []*node {
	cells := []*node{}
	for _, child := range row.children {
		if child.tag == "td" || child.tag == "th" {
			cells = append(cells, child)
		}
	}
	return cells
}

// ── text helpers ────────────────────────────────────────────────────────────

// spanishLabel keeps the Spanish half of "English|Español" labels (an unspaced "|"), and turns the
// remaining spaced separators into "/" so they never clash with the outline's " | ".
func spanishLabel(label string) string {
	label = strings.TrimSpace(label)
	// The match is "<char>|<char>": the Spanish half starts right after the "|".
	if location := unspacedBilingualSeparator.FindStringIndex(label); location != nil {
		english, spanish := label[:location[0]+1], label[location[0]+2:]
		if strings.Contains(english, " ") && !strings.Contains(strings.TrimSpace(spanish), " ") {
			// One word swapped inside a sentence: "create a new client|cliente." → "create a new cliente."
			label = english[:strings.LastIndex(english, " ")+1] + spanish
		} else {
			label = strings.TrimSpace(spanish)
		}
	}
	return strings.ReplaceAll(label, " | ", " / ")
}

// dropRepeatedTokens removes a word that repeats the one before it.
func dropRepeatedTokens(text string) string {
	words := strings.Fields(text)
	kept := []string{}
	for index, word := range words {
		if index == 0 || word != words[index-1] {
			kept = append(kept, word)
		}
	}
	return strings.Join(kept, " ")
}

func cleanText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func quote(text string) string {
	return `"` + strings.ReplaceAll(text, `"`, `\"`) + `"`
}

func escapeCell(text string) string {
	return strings.ReplaceAll(text, "|", "/")
}

func escapeCells(values []string) []string {
	escaped := make([]string, len(values))
	for index, value := range values {
		escaped[index] = escapeCell(value)
	}
	return escaped
}

func htmlAttr(n *html.Node, key string) string {
	for _, attribute := range n.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

func findHTML(n *html.Node, tag string) *html.Node {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tag {
			return child
		}
		if found := findHTML(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func htmlText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	parts := []string{}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		parts = append(parts, htmlText(child))
	}
	return strings.Join(parts, " ")
}

func atoiOrOne(value string) int {
	number := 0
	for _, character := range value {
		if character < '0' || character > '9' {
			return 1
		}
		number = number*10 + int(character-'0')
	}
	if number < 1 {
		return 1
	}
	return number
}

var methodsSuffix = regexp.MustCompile(`\{([a-zA-Z,]+)\}$`)

// Methods lists what a rendered outline line's component accepts: its printed `{m1,m2}` or its
// type's defaults.
func Methods(line string) []string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	if match := methodsSuffix.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
		return strings.Split(match[1], ",")
	}
	if defaults := defaultMethods[fields[0]]; defaults != "" {
		return strings.Split(defaults, ",")
	}
	return nil
}
