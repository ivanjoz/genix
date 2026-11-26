package outline

import (
	"regexp"
	"strings"
)

// Delta reports what changed between two outlines of the same page, for the tool result of an
// action: "~" a line whose content changed (a filled field, a new selection), "+" a new line,
// "-" a line that disappeared (only its "Type id" key for component lines).
//
// It returns the full next outline instead (full=true) when the page itself changed (different
// Page line key) or when more than maxChangedShare of the lines changed (a layer opened), because
// then the delta is not shorter and loses the hierarchy.
//
//	Delta /crm/customers
//	~ Input 79 | Nombre = "Pedro Pascal"
//	~ Select 82 | Tipo Persona = [2] Empresa | opciones: [1] Persona, [2] Empresa
func Delta(previous, next string) (outline string, full bool) {
	const maxChangedShare = 0.5
	previousLines, nextLines := outlineLines(previous), outlineLines(next)
	if len(previousLines) == 0 || len(nextLines) == 0 || lineKey(previousLines[0]) != lineKey(nextLines[0]) {
		return next, true
	}

	previousByKey := map[string]string{}
	for _, line := range previousLines {
		previousByKey[lineKey(line)] = line
	}
	nextKeys := map[string]bool{}
	changes := []string{}
	for _, line := range nextLines {
		key := lineKey(line)
		nextKeys[key] = true
		before, existed := previousByKey[key]
		switch {
		case !existed:
			changes = append(changes, "+ "+line)
		case before != line:
			changes = append(changes, "~ "+line)
		}
	}
	for _, line := range previousLines {
		if key := lineKey(line); !nextKeys[key] {
			changes = append(changes, "- "+key)
		}
	}
	if float64(len(changes)) > maxChangedShare*float64(len(nextLines)) {
		return next, true
	}
	header := "Delta" + strings.TrimPrefix(lineKey(nextLines[0]), "Page")
	if len(changes) == 0 {
		return header + "\nText | sin cambios\n", false
	}
	return header + "\n" + strings.Join(changes, "\n") + "\n", false
}

// componentKeyPattern matches "Type id" at the start of a line: a numeric/composite Genix id, or the
// route of the Page line.
var componentKeyPattern = regexp.MustCompile(`^([A-Z][A-Za-z]*) ([0-9][0-9:]*|/\S*)(?: |$)`)

// lineKey identifies a line across two outlines: "Type id" for components, rows and the page; the
// whole line for id-less lines (Text, More, read-only rows), which can only appear or disappear.
func lineKey(line string) string {
	if match := componentKeyPattern.FindStringSubmatch(line); match != nil {
		return match[1] + " " + match[2]
	}
	return line
}

func outlineLines(outline string) []string {
	lines := []string{}
	for _, line := range strings.Split(outline, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}
