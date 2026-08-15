package agent

import (
	"sort"
	"strings"

	"app/agent/llm"
)

const (
	pageSnapshotMaxBytes     = 30_000
	pageSnapshotMaxTableRows = 12
	pageSnapshotSection      = "\n=== PAGE SNAPSHOT ===\n"
)

type snapshotSpan struct {
	start int
	end   int
}

type snapshotTableRows struct {
	rows []snapshotSpan
}

// pageToolResult keeps machine-readable metadata separate from the raw page
// text, avoiding an escaped HTML string inside another JSON string.
func pageToolResult(metadata any, snapshot string) string {
	return llm.PageSnapshotGrammar + toolJSON(metadata) + pageSnapshotSection + snapshot
}

func splitPageToolResult(content string) (metadata, snapshot string, found bool) {
	if !strings.HasPrefix(content, llm.PageSnapshotGrammar) {
		return "", "", false
	}
	metadata, snapshot, found = strings.Cut(content[len(llm.PageSnapshotGrammar):], pageSnapshotSection)
	return strings.TrimSpace(metadata), snapshot, found
}

func pageHTML(page *PageContent) string {
	if page == nil {
		return ""
	}
	return page.HTML
}

// limitTableRows keeps the first rows of every compact <TableBody> and removes
// complete trailing rows independently for each table.
func limitTableRows(snapshot string, maximumRows int) (string, int) {
	if maximumRows < 0 {
		return snapshot, 0
	}
	var excessRows []snapshotSpan
	for _, table := range findSnapshotTables(snapshot) {
		if len(table.rows) > maximumRows {
			excessRows = append(excessRows, table.rows[maximumRows:]...)
		}
	}
	return removeSnapshotSpans(snapshot, excessRows), len(excessRows)
}

// trimTableRowsToBytes removes complete rows from the end of identified
// tables, preserving the surrounding table and its headers.
func trimTableRowsToBytes(snapshot string, maximumBytes int) (string, int) {
	if maximumBytes <= 0 || len(snapshot) <= maximumBytes {
		return snapshot, 0
	}
	rows := findTableRowSpans(snapshot)
	trimmedRows := 0
	for index := len(rows) - 1; index >= 0 && len(snapshot) > maximumBytes; index-- {
		row := rows[index]
		snapshot = snapshot[:row.start] + snapshot[row.end:]
		trimmedRows++
	}
	return snapshot, trimmedRows
}

// findTableRowSpans recognizes compact <TableBody> containers and returns
// only complete <Row> elements nested inside them.
func findTableRowSpans(snapshot string) []snapshotSpan {
	var rows []snapshotSpan
	for _, table := range findSnapshotTables(snapshot) {
		rows = append(rows, table.rows...)
	}
	sort.Slice(rows, func(left, right int) bool { return rows[left].start < rows[right].start })
	return rows
}

// findSnapshotTables associates each compact Row with its TableBody.
func findSnapshotTables(snapshot string) []snapshotTableRows {
	tables := make([]snapshotTableRows, 0)
	tableStack := make([]int, 0)
	rowDepth := 0
	rowStart := -1
	rowTable := -1
	for cursor := 0; cursor < len(snapshot); {
		relativeStart := strings.IndexByte(snapshot[cursor:], '<')
		if relativeStart < 0 {
			break
		}
		tagStart := cursor + relativeStart
		relativeEnd := strings.IndexByte(snapshot[tagStart:], '>')
		if relativeEnd < 0 {
			break
		}
		tagEnd := tagStart + relativeEnd + 1
		name, closing, selfClosing := snapshotTag(snapshot[tagStart:tagEnd])

		switch name {
		case "TableBody":
			if closing {
				if len(tableStack) > 0 {
					tableStack = tableStack[:len(tableStack)-1]
				}
			} else if !selfClosing {
				tables = append(tables, snapshotTableRows{})
				tableStack = append(tableStack, len(tables)-1)
			}
		case "Row":
			if len(tableStack) == 0 {
				break
			}
			activeTable := tableStack[len(tableStack)-1]
			if closing {
				if rowDepth > 0 {
					rowDepth--
					if rowDepth == 0 && rowStart >= 0 && rowTable >= 0 {
						tables[rowTable].rows = append(tables[rowTable].rows, expandSnapshotLine(snapshot, snapshotSpan{start: rowStart, end: tagEnd}))
						rowStart = -1
						rowTable = -1
					}
				}
			} else if selfClosing {
				tables[activeTable].rows = append(tables[activeTable].rows, expandSnapshotLine(snapshot, snapshotSpan{start: tagStart, end: tagEnd}))
			} else {
				if rowDepth == 0 {
					rowStart = tagStart
					rowTable = activeTable
				}
				rowDepth++
			}
		}
		cursor = tagEnd
	}
	return tables
}

func removeSnapshotSpans(snapshot string, spans []snapshotSpan) string {
	sort.Slice(spans, func(left, right int) bool { return spans[left].start < spans[right].start })
	for index := len(spans) - 1; index >= 0; index-- {
		span := spans[index]
		snapshot = snapshot[:span.start] + snapshot[span.end:]
	}
	return snapshot
}

func snapshotTag(rawTag string) (name string, closing, selfClosing bool) {
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(rawTag, "<"), ">"))
	if inner == "" || strings.HasPrefix(inner, "!") || strings.HasPrefix(inner, "?") {
		return "", false, false
	}
	closing = strings.HasPrefix(inner, "/")
	inner = strings.TrimSpace(strings.TrimPrefix(inner, "/"))
	selfClosing = strings.HasSuffix(inner, "/")
	inner = strings.TrimSpace(strings.TrimSuffix(inner, "/"))
	if boundary := strings.IndexAny(inner, " \t\r\n"); boundary >= 0 {
		inner = inner[:boundary]
	}
	return inner, closing, selfClosing
}

func expandSnapshotLine(snapshot string, span snapshotSpan) snapshotSpan {
	lineStart := strings.LastIndexByte(snapshot[:span.start], '\n') + 1
	if strings.TrimSpace(snapshot[lineStart:span.start]) == "" {
		span.start = lineStart
	}
	if span.end < len(snapshot) && snapshot[span.end] == '\n' {
		span.end++
	}
	return span
}
