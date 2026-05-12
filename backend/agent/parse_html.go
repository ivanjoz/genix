package agent

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "source": true, "track": true,
	"wbr": true,
}

var droppedElements = map[string]bool{
	"script": true, "style": true, "noscript": true, "template": true,
}

// ParsePageHTML parses an HTML string and returns a cleaned, indented copy.
// It removes every attribute except aria-* and role, drops non-content tags
// (script/style/comments), deletes element subtrees that hold no text, and
// collapses chains of text-less wrappers so at most one wrapper sits between
// any two text-bearing layers.
// Elements with a data-id matching a known component (e.g. "Input:4",
// "SearchSelect:4") are replaced with a compact self-closing tag using the
// metadata from the supplied components slice.
func ParsePageHTML(input string, components []AgentComponentInfo) (string, error) {
	cm := make(map[string]AgentComponentInfo, len(components))
	for _, c := range components {
		cm[fmt.Sprintf("%s:%d", c.Type, c.ID)] = c
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(input))
	if err != nil {
		return "", err
	}
	root := doc.Get(0)
	if root == nil {
		return "", nil
	}

	dropNoise(root)
	dropMenu(root)
	cleanAttributes(root)
	for {
		removed := removeEmptyElements(root, cm)
		collapsed := collapseWrappers(root, cm)
		if !removed && !collapsed {
			break
		}
	}

	start := findElement(root, "body")
	if start == nil {
		start = root
	}

	var buf bytes.Buffer
	for c := start.FirstChild; c != nil; c = c.NextSibling {
		renderIndented(&buf, c, 0, cm)
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

func findElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if got := findElement(c, tag); got != nil {
			return got
		}
	}
	return nil
}

// dropMenu removes the side-menu DOM (desktop nav + mobile drawer) from the
// snapshot. The menu structure is exposed separately via GET /agent?get=menu
// so leaving it in the HTML would only repeat the same routes the agent
// already has access to via that endpoint. The frontend marks both root
// elements with `data-menu-root`.
func dropMenu(n *html.Node) {
	c := n.FirstChild
	for c != nil {
		next := c.NextSibling
		if c.Type == html.ElementNode && nodeAttr(c, "data-menu-root") != "" {
			n.RemoveChild(c)
		} else {
			dropMenu(c)
		}
		c = next
	}
}

func dropNoise(n *html.Node) {
	c := n.FirstChild
	for c != nil {
		next := c.NextSibling
		switch {
		case c.Type == html.CommentNode, c.Type == html.DoctypeNode:
			n.RemoveChild(c)
		case c.Type == html.ElementNode && droppedElements[c.Data]:
			n.RemoveChild(c)
		default:
			dropNoise(c)
		}
		c = next
	}
}

func cleanAttributes(n *html.Node) {
	if n.Type == html.ElementNode {
		kept := n.Attr[:0]
		for _, attr := range n.Attr {
			key := strings.ToLower(attr.Key)
			if key == "role" || key == "data-id" || key == "data-value" || key == "data-label" || key == "data-type" || key == "data-cell-type" || key == "data-selected" || strings.HasPrefix(key, "aria-") {
				kept = append(kept, attr)
			}
		}
		n.Attr = kept
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		cleanAttributes(c)
	}
}

// nodeAttr returns the value of the named attribute on an element node, or "".
func nodeAttr(n *html.Node, key string) string {
	if n.Type != html.ElementNode {
		return ""
	}
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// nodeDataID returns the data-id attribute value for an element node, or "".
func nodeDataID(n *html.Node) string {
	if n.Type != html.ElementNode {
		return ""
	}
	for _, attr := range n.Attr {
		if attr.Key == "data-id" {
			return attr.Val
		}
	}
	return ""
}

// componentDataID returns dataID if the element matches a known component in
// cm, or "" otherwise. cm is keyed by "<Type>:<id>".
func componentDataID(n *html.Node, cm map[string]AgentComponentInfo) string {
	dataID := nodeDataID(n)
	if dataID == "" {
		return ""
	}
	if _, ok := cm[dataID]; !ok {
		return ""
	}
	return dataID
}

// markerComponentTypes lists data-id prefixes that don't have a registry
// handle of their own but should still be rendered as compact tags inside
// their parent (e.g. options inside a SearchCard, rows inside a Table).
var markerComponentTypes = map[string]bool{
	"Option":   true,
	"TableRow": true,
	"Button":   true,
}

// markerMethods lists the comma-separated method names available on each
// marker type, so the agent knows how to interact with them.
//
// TableRow's "select" surfaces the parent Table's row-click handler — the
// table is the only registered handle, but the agent addresses individual
// rows by their composite "<tableID>:<rowID>" id.
//
// Option's default is "remove" (SearchCard chips, CheckboxOptions); when the
// nearest agent ancestor exposes a "select" verb (PageViews, OptionsStrip,
// ArrowSteps) the marker is upgraded to "select" instead — see
// markerMethodsFor.
var markerMethods = map[string]string{
	"Option":   "remove",
	"Button":   "click",
	"TableRow": "select",
}

// markerMethodsFor returns the method hint for a marker, taking the nearest
// agent ancestor into account. Options live under different container styles
// — selection-strips ("select") and chip-removers ("remove") — and the
// marker should advertise the verb the agent will actually use.
func markerMethodsFor(typ string, n *html.Node, cm map[string]AgentComponentInfo) string {
	if typ != "Option" {
		return markerMethods[typ]
	}
	parent := nearestAgentAncestor(n, cm)
	if parent == nil {
		return markerMethods[typ]
	}
	for _, m := range parent.Methods {
		if m == "select" {
			return "select"
		}
	}
	return markerMethods[typ]
}

// nearestAgentAncestor walks up the DOM until it finds a registered agent
// component (skipping markers and cells, which are not in the registry).
// Returns nil when no registered ancestor exists in the path to the root.
func nearestAgentAncestor(n *html.Node, cm map[string]AgentComponentInfo) *AgentComponentInfo {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type != html.ElementNode {
			continue
		}
		dataID := nodeDataID(p)
		if dataID == "" {
			continue
		}
		if info, ok := cm[dataID]; ok {
			return &info
		}
	}
	return nil
}

// cellMethods lists the methods exposed on each cell type. The Table is the
// sole registered handle and routes these calls to the cell via its own
// setValueChild / searchChild / getOptionsChild internals; the bridge in
// http.go::resolveTarget rewrites setValue/search/getOptions on a composite
// id back to the *Child variant before invoking the table.
var cellMethods = map[string]string{
	"CellInput":  "setValue",
	"CellSelect": "search,select,getOptions",
	// CellClick lives on a <td>/<th> directly (not on an inner div), so the
	// rendering path keeps the cell tag and stamps id/methods on it. The
	// column position the cell occupies is preserved.
	"CellClick": "click",
}

// suppressedTableMethods are emitted on the cells/rows of a Table instead of
// on the Table element itself — surfacing them on the parent would mislead
// the agent into calling them with a child-relative id and no parent prefix.
var suppressedTableMethods = map[string]bool{
	"select":          true,
	"setValueChild":   true,
	"searchChild":     true,
	"getOptionsChild": true,
	"clickChild":      true,
}

// recurseContentTypes lists registered components whose inner HTML structure
// is preserved in the snapshot instead of self-closing the tag. Card wraps
// free-form user content (prices, stock counts, raw buttons, etc.) that the
// agent needs to read alongside its label. Empty wrappers are still pruned
// by the standard passes — only text-bearing children make it through.
var recurseContentTypes = map[string]bool{
	"Card":      true,
	"PageViews": true,
}

// structuralComponentTypes are registered components rendered as bare tags —
// no id, label, value, or methods on the parent — because the interactive
// surface lives on the children (e.g. PageViews's Option markers). The
// handle is still needed for routing, but the agent addresses the children
// directly via composite ids.
var structuralComponentTypes = map[string]bool{
	"PageViews": true,
}

// markerComponentType returns the marker type ("Option", "TableRow", "Button")
// for an element whose data-id uses a marker prefix, or "" otherwise.
func markerComponentType(n *html.Node) string {
	dataID := nodeDataID(n)
	if dataID == "" {
		return ""
	}
	colon := strings.IndexByte(dataID, ':')
	if colon < 0 {
		return ""
	}
	typ := dataID[:colon]
	if markerComponentTypes[typ] {
		return typ
	}
	return ""
}

// cellComponentType returns the cell type ("CellInput" / "CellSelect")
// for an element whose data-id is "<tableID>:<cellID>" pointing to a
// registered Table or CardList component. Cells don't have a registry
// handle of their own; the parent container is the agent handle and
// dispatches by cellID. The type is read from the `data-cell-type`
// attribute the cell stamps on its root.
func cellComponentType(n *html.Node, cm map[string]AgentComponentInfo) string {
	dataID := nodeDataID(n)
	if dataID == "" {
		return ""
	}
	colon := strings.IndexByte(dataID, ':')
	if colon < 0 {
		return ""
	}
	tablePart := dataID[:colon]
	if _, ok := cm["Table:"+tablePart]; !ok {
		if _, ok := cm["CardList:"+tablePart]; !ok {
			return ""
		}
	}
	return nodeAttr(n, "data-cell-type")
}

// isAgentElement reports whether the element should be preserved in the
// rendered output (registered component, marker, or cell-of-Table).
func isAgentElement(n *html.Node, cm map[string]AgentComponentInfo) bool {
	return componentDataID(n, cm) != "" || markerComponentType(n) != "" || cellComponentType(n, cm) != ""
}

// inlineText flattens n's subtree into a single normalised text string when
// it is pure chrome around text — no agent elements, no void elements, just
// nested wrappers and text nodes. Returns ok=false if any descendant carries
// meaning beyond text, signaling that the caller must render the children
// recursively. Used to compress `<th><div>ID</div></th>` into `<th>ID</th>`
// and `<Option><span>foo</span></Option>` into `<Option>foo</Option>`.
func inlineText(n *html.Node, cm map[string]AgentComponentInfo) (string, bool) {
	var out strings.Builder
	if !appendInlineText(n, &out, cm) {
		return "", false
	}
	return collapseWhitespace(out.String()), true
}

func appendInlineText(n *html.Node, out *strings.Builder, cm map[string]AgentComponentInfo) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			out.WriteString(c.Data)
		case html.ElementNode:
			if voidElements[c.Data] || isAgentElement(c, cm) {
				return false
			}
			if !appendInlineText(c, out, cm) {
				return false
			}
		}
	}
	return true
}

func isProtectedShell(n *html.Node) bool {
	return n.Type == html.ElementNode && (n.Data == "html" || n.Data == "body")
}

func hasTextDescendant(n *html.Node, cm map[string]AgentComponentInfo) bool {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data) != ""
	}
	if n.Type == html.ElementNode {
		if voidElements[n.Data] || isAgentElement(n, cm) {
			return true
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if hasTextDescendant(c, cm) {
			return true
		}
	}
	return false
}

func removeEmptyElements(n *html.Node, cm map[string]AgentComponentInfo) bool {
	changed := false
	c := n.FirstChild
	for c != nil {
		next := c.NextSibling
		if c.Type == html.ElementNode && !voidElements[c.Data] {
			if removeEmptyElements(c, cm) {
				changed = true
			}
			// Table cells (<td>/<th>, plus ARIA gridcell/columnheader) are
			// structural: their position in the row defines the column, so an
			// empty cell still carries meaning when its siblings are populated.
			// Whole-empty rows are still pruned because the parent <tr> then
			// has no text descendants and gets dropped on its own pass.
			if !isProtectedShell(c) && !isCellElement(c) && !hasTextDescendant(c, cm) {
				n.RemoveChild(c)
				changed = true
			}
		}
		c = next
	}
	return changed
}

func hasOwnText(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && strings.TrimSpace(c.Data) != "" {
			return true
		}
	}
	return false
}

func onlyElementChild(n *html.Node) *html.Node {
	var only *html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		if only != nil {
			return nil
		}
		only = c
	}
	return only
}

func isWrapper(n *html.Node, cm map[string]AgentComponentInfo) bool {
	if n == nil || n.Type != html.ElementNode {
		return false
	}
	if voidElements[n.Data] || isProtectedShell(n) || isAgentElement(n, cm) {
		return false
	}
	if hasOwnText(n) {
		return false
	}
	return onlyElementChild(n) != nil
}

func collapseWrappers(n *html.Node, cm map[string]AgentComponentInfo) bool {
	changed := false
	c := n.FirstChild
	for c != nil {
		next := c.NextSibling
		if collapseWrappers(c, cm) {
			changed = true
		}
		c = next
	}
	if !isWrapper(n, cm) {
		return changed
	}
	for {
		child := onlyElementChild(n)
		if !isWrapper(child, cm) {
			return changed
		}
		grand := onlyElementChild(child)
		if grand == nil {
			return changed
		}
		child.RemoveChild(grand)
		n.InsertBefore(grand, child)
		n.RemoveChild(child)
		changed = true
	}
}

func renderIndented(buf *bytes.Buffer, n *html.Node, depth int, cm map[string]AgentComponentInfo) {
	indent := strings.Repeat("  ", depth)
	switch n.Type {
	case html.TextNode:
		text := collapseWhitespace(n.Data)
		if text == "" {
			return
		}
		buf.WriteString(indent)
		buf.WriteString(html.EscapeString(text))
		buf.WriteByte('\n')
	case html.ElementNode:
		// Replace known component elements with a compact tag. Container components
		// (those that hold further registered components inside) are rendered with
		// their children so the agent still sees the nested handles. Markers
		// (Option/TableRow/Button) follow the same shape but use the marker type
		// from the data-id directly since they have no registry entry.
		if dataID := nodeDataID(n); dataID != "" {
			if c, ok := cm[dataID]; ok {
				renderComponent(buf, indent, dataID, c, n, depth, cm)
				return
			}
			if markerComponentType(n) != "" {
				renderMarker(buf, indent, n, depth, cm)
				return
			}
			// Inner-div cells (CellInput / CellSelect) collapse to a self-
			// closing tag. Cells whose hit-target is the <td>/<th> itself
			// (CellClick) keep the cell tag and pick up id/methods further
			// below so the column position stays visible.
			if cellType := cellComponentType(n, cm); cellType != "" && !isCellElement(n) {
				renderCell(buf, indent, cellType, dataID, n)
				return
			}
		}

		hasElementChild := false
		var directText strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				hasElementChild = true
			} else if c.Type == html.TextNode {
				directText.WriteString(c.Data)
			}
		}
		agentCellID, agentCellMethods := agentCellAttrsForElement(n, cm)
		buf.WriteString(indent)
		buf.WriteByte('<')
		buf.WriteString(n.Data)
		if agentCellID != "" {
			buf.WriteString(` id="`)
			buf.WriteString(html.EscapeString(agentCellID))
			buf.WriteByte('"')
			if agentCellMethods != "" {
				buf.WriteString(` methods="`)
				buf.WriteString(agentCellMethods)
				buf.WriteByte('"')
			}
		}
		for _, attr := range n.Attr {
			// data-id / data-cell-type are emitted as the agent-facing id /
			// methods on cell-on-td elements; skip the raw form so we don't
			// print both.
			if agentCellID != "" && (attr.Key == "data-id" || attr.Key == "data-cell-type") {
				continue
			}
			buf.WriteByte(' ')
			buf.WriteString(attr.Key)
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(attr.Val))
			buf.WriteByte('"')
		}
		if voidElements[n.Data] {
			buf.WriteString(" />\n")
			return
		}
		buf.WriteByte('>')
		if !hasElementChild {
			text := collapseWhitespace(directText.String())
			buf.WriteString(html.EscapeString(text))
			buf.WriteString("</")
			buf.WriteString(n.Data)
			buf.WriteString(">\n")
			return
		}
		// Table cells (<th>/<td>, plus the ARIA-equivalent <div role="cell">
		// / role="columnheader" used by grid-style tables) are commonly
		// wrapped in a chrome element for styling. When there's nothing
		// meaningful inside beyond that wrapped text, fold it onto one line
		// for a cleaner agent view.
		if isCellElement(n) {
			if text, ok := inlineText(n, cm); ok {
				buf.WriteString(html.EscapeString(text))
				buf.WriteString("</")
				buf.WriteString(n.Data)
				buf.WriteString(">\n")
				return
			}
			// When the cell holds a single agent element (e.g. a CellInput
			// or CellSelect inside a <td>), keep open tag + agent + close tag
			// on one line so the agent view stays compact:
			//   <td><CellInput id="46" value="1"/></td>
			if child := singleAgentChild(n, cm); child != nil {
				var tmp bytes.Buffer
				renderIndented(&tmp, child, 0, cm)
				inner := strings.TrimRight(tmp.String(), "\n")
				buf.WriteString(inner)
				buf.WriteString("</")
				buf.WriteString(n.Data)
				buf.WriteString(">\n")
				return
			}
		}
		buf.WriteByte('\n')
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderIndented(buf, c, depth+1, cm)
		}
		buf.WriteString(indent)
		buf.WriteString("</")
		buf.WriteString(n.Data)
		buf.WriteString(">\n")
	}
}

func renderComponent(buf *bytes.Buffer, indent, dataID string, c AgentComponentInfo, n *html.Node, depth int, cm map[string]AgentComponentInfo) {
	colon := strings.IndexByte(dataID, ':')
	typ := dataID[:colon]
	id := dataID[colon+1:]
	value := nodeAttr(n, "data-value")

	buf.WriteString(indent)
	buf.WriteByte('<')
	buf.WriteString(html.EscapeString(typ))
	structural := structuralComponentTypes[typ]
	if !structural {
		buf.WriteString(` id="`)
		buf.WriteString(html.EscapeString(id))
		buf.WriteByte('"')
		if c.Label != "" {
			buf.WriteString(` label="`)
			buf.WriteString(html.EscapeString(c.Label))
			buf.WriteByte('"')
		}
		if value != "" {
			buf.WriteString(` value="`)
			buf.WriteString(html.EscapeString(value))
			buf.WriteByte('"')
		}
		methods := c.Methods
		if typ == "Table" || typ == "CardList" {
			// Table/CardList row/cell-routing methods belong on the inner TableRow /
			// cell tags, not on the container itself. Drop them here.
			filtered := methods[:0:0]
			for _, m := range methods {
				if !suppressedTableMethods[m] {
					filtered = append(filtered, m)
				}
			}
			methods = filtered
		}
		if len(methods) > 0 {
			buf.WriteString(` methods="`)
			buf.WriteString(html.EscapeString(strings.Join(methods, ",")))
			buf.WriteByte('"')
		}
	}

	// Recurse when there are registered components or markers nested inside —
	// Layer/Modal/ButtonLayer hold further handles, Table holds TableRow markers,
	// SearchCard holds Option markers, etc. Leaf components self-close, except
	// for recurseContentTypes (e.g. Card) whose HTML body is preserved verbatim.
	if !hasAgentDescendant(n, cm) && !recurseContentTypes[typ] {
		buf.WriteString("/>\n")
		return
	}
	buf.WriteString(">\n")
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		renderIndented(buf, child, depth+1, cm)
	}
	buf.WriteString(indent)
	buf.WriteString("</")
	buf.WriteString(html.EscapeString(typ))
	buf.WriteString(">\n")
}

// renderMarker emits a compact tag for an Option / TableRow / Button element.
// Markers don't have a registry handle, but they carry id, optional value and
// optional selected state on the DOM node, plus the inner text/children that
// the agent uses to identify the row or option.
func renderMarker(buf *bytes.Buffer, indent string, n *html.Node, depth int, cm map[string]AgentComponentInfo) {
	dataID := nodeDataID(n)
	colon := strings.IndexByte(dataID, ':')
	typ := dataID[:colon]
	id := dataID[colon+1:]
	value := nodeAttr(n, "data-value")
	selected := nodeAttr(n, "data-selected") == "true"
	ariaLabel := nodeAttr(n, "aria-label")

	buf.WriteString(indent)
	buf.WriteByte('<')
	buf.WriteString(html.EscapeString(typ))
	buf.WriteString(` id="`)
	buf.WriteString(html.EscapeString(id))
	buf.WriteByte('"')
	if value != "" {
		buf.WriteString(` value="`)
		buf.WriteString(html.EscapeString(value))
		buf.WriteByte('"')
	}
	if ariaLabel != "" {
		buf.WriteString(` aria-label="`)
		buf.WriteString(html.EscapeString(ariaLabel))
		buf.WriteByte('"')
	}
	if selected {
		buf.WriteString(` selected`)
	}
	if methods := markerMethodsFor(typ, n, cm); methods != "" {
		buf.WriteString(` methods="`)
		buf.WriteString(methods)
		buf.WriteByte('"')
	}

	// Inspect children: when the marker only contains text we can keep it on
	// one line; element children get a recursive render so nested markers /
	// components stay visible.
	hasElementChild := false
	var directText strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			hasElementChild = true
		} else if c.Type == html.TextNode {
			directText.WriteString(c.Data)
		}
	}

	if !hasElementChild {
		text := collapseWhitespace(directText.String())
		if text == "" {
			buf.WriteString("/>\n")
			return
		}
		buf.WriteByte('>')
		buf.WriteString(html.EscapeString(text))
		buf.WriteString("</")
		buf.WriteString(html.EscapeString(typ))
		buf.WriteString(">\n")
		return
	}

	// Chrome around plain text — fold children into a single inline text run
	// (e.g. <Option><span>foo</span></Option> -> <Option>foo</Option>). Skip
	// TableRow: rows preserve their cell structure so the agent can read each
	// column.
	if typ != "TableRow" {
		if text, ok := inlineText(n, cm); ok {
			if text == "" {
				buf.WriteString("/>\n")
				return
			}
			buf.WriteByte('>')
			buf.WriteString(html.EscapeString(text))
			buf.WriteString("</")
			buf.WriteString(html.EscapeString(typ))
			buf.WriteString(">\n")
			return
		}
	}

	buf.WriteString(">\n")
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		renderIndented(buf, child, depth+1, cm)
	}
	buf.WriteString(indent)
	buf.WriteString("</")
	buf.WriteString(html.EscapeString(typ))
	buf.WriteString(">\n")
}

// agentCellAttrsForElement returns the id/methods attrs to print for a
// cell-on-td (CellClick): a <td>/<th> whose `data-cell-type` matches an
// entry in cellMethods and whose parent Table is registered. Returns empty
// strings for any other element.
func agentCellAttrsForElement(n *html.Node, cm map[string]AgentComponentInfo) (id, methods string) {
	if !isCellElement(n) {
		return "", ""
	}
	cellType := cellComponentType(n, cm)
	if cellType == "" {
		return "", ""
	}
	m, ok := cellMethods[cellType]
	if !ok {
		return "", ""
	}
	return nodeDataID(n), m
}

// isCellElement reports whether n is treated as a table cell for the
// inline-fold logic: a native <td>/<th> or any element carrying the
// equivalent ARIA role ("cell", "columnheader", "rowheader", "gridcell").
// Grid tables built from <div>s rely on the role to be addressable.
func isCellElement(n *html.Node) bool {
	if n == nil || n.Type != html.ElementNode {
		return false
	}
	if n.Data == "td" || n.Data == "th" {
		return true
	}
	switch nodeAttr(n, "role") {
	case "cell", "gridcell", "columnheader", "rowheader":
		return true
	}
	return false
}

// singleAgentChild returns the lone agent-element child of n (registered
// component or marker) iff n has exactly one element child, that child is an
// agent element, and there is no non-whitespace text directly under n.
// Returns nil otherwise. Used by the table-cell renderer to keep
// <td><CellInput .../></td> on a single line.
func singleAgentChild(n *html.Node, cm map[string]AgentComponentInfo) *html.Node {
	var only *html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			if strings.TrimSpace(c.Data) != "" {
				return nil
			}
		case html.ElementNode:
			if !isAgentElement(c, cm) {
				return nil
			}
			if only != nil {
				return nil
			}
			only = c
		}
	}
	return only
}

// renderCell emits a compact, self-closing tag for a CellInput /
// CellSelect under a registered Table. The cell carries the composite
// "<tableID>:<cellID>" id verbatim so the agent can reuse it when invoking
// the Table's setValueChild / select / searchChild / getOptionsChild.
func renderCell(buf *bytes.Buffer, indent, typ, dataID string, n *html.Node) {
	value := nodeAttr(n, "data-value")
	label := nodeAttr(n, "data-label")
	dataType := nodeAttr(n, "data-type")

	buf.WriteString(indent)
	buf.WriteByte('<')
	buf.WriteString(html.EscapeString(typ))
	buf.WriteString(` id="`)
	buf.WriteString(html.EscapeString(dataID))
	buf.WriteByte('"')
	if label != "" {
		buf.WriteString(` label="`)
		buf.WriteString(html.EscapeString(label))
		buf.WriteByte('"')
	}
	if value != "" {
		buf.WriteString(` value="`)
		buf.WriteString(html.EscapeString(value))
		buf.WriteByte('"')
	}
	if dataType != "" && dataType != "other" {
		buf.WriteString(` type="`)
		buf.WriteString(html.EscapeString(dataType))
		buf.WriteByte('"')
	}
	if methods := cellMethods[typ]; methods != "" {
		buf.WriteString(` methods="`)
		buf.WriteString(methods)
		buf.WriteByte('"')
	}
	buf.WriteString("/>\n")
}

func hasAgentDescendant(n *html.Node, cm map[string]AgentComponentInfo) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		if isAgentElement(c, cm) {
			return true
		}
		if hasAgentDescendant(c, cm) {
			return true
		}
	}
	return false
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	leading := true
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if leading {
				continue
			}
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		leading = false
		prevSpace = false
		b.WriteRune(r)
	}
	out := b.String()
	return strings.TrimRight(out, " ")
}
