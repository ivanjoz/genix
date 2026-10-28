package agent

import (
	"fmt"
	"strings"
	"testing"
)

func TestPageToolResultKeepsSnapshotOutsideMetadataJSON(t *testing.T) {
	snapshot := `<Page><Button id="7" methods="click">Nuevo</Button></Page>`
	result := pageToolResult(map[string]any{"ok": true}, snapshot)

	if strings.Contains(result, `\u003c`) || !strings.Contains(result, pageSnapshotSection+snapshot) {
		t.Fatalf("snapshot must remain readable outside metadata JSON: %s", result)
	}
	metadata, extractedSnapshot, found := splitPageToolResult(result)
	if !found || metadata != `{"ok":true}` || extractedSnapshot != snapshot {
		t.Fatalf("unexpected page result split: found=%v metadata=%q snapshot=%q", found, metadata, extractedSnapshot)
	}
	if stripped := stripPageSnapshotFromToolResult(result); stripped != metadata {
		t.Fatalf("snapshot pruning changed metadata: %q", stripped)
	}
}

func TestToolResultErrorSummary(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{name: "top-level tool error", content: `{"error":"page actions are not allowed"}`, expected: "page actions are not allowed"},
		{name: "batch invocation error", content: `{"Results":[{"OK":false,"Error":"save failed"}]}`, expected: "save failed"},
		{name: "failed batch without detail", content: `{"Results":[{"OK":false}]}`, expected: "La acción no se completó."},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			summary, found := toolResultErrorSummary(testCase.content)
			if !found || summary != testCase.expected {
				t.Fatalf("toolResultErrorSummary() = %q, %t; want %q, true", summary, found, testCase.expected)
			}
		})
	}
	if summary, found := toolResultErrorSummary(`{"Results":[{"OK":true}]}`); found || summary != "" {
		t.Fatalf("successful result reported as error: %q, %t", summary, found)
	}
}

func TestTrimTableRowsToBytesRemovesLastRowsWithinLimit(t *testing.T) {
	var snapshot strings.Builder
	snapshot.WriteString("<Page>\n  <Table id=\"1\">\n    <table>\n      <thead><tr><th>Name</th></tr></thead>\n      <TableBody id=\"1\" methods=\"selectRow\">\n")
	for rowID := 1; rowID <= 40; rowID++ {
		fmt.Fprintf(&snapshot, "        <Row id=\"1:%d\">\n          <td>%s</td>\n        </Row>\n", rowID, strings.Repeat("x", 1_000))
	}
	snapshot.WriteString("      </TableBody>\n    </table>\n  </Table>\n</Page>\n")

	trimmed, trimmedRows := trimTableRowsToBytes(snapshot.String(), pageSnapshotMaxBytes)
	if len(trimmed) > pageSnapshotMaxBytes || trimmedRows == 0 {
		t.Fatalf("snapshot was not reduced to the byte limit: bytes=%d rows=%d", len(trimmed), trimmedRows)
	}
	if !strings.Contains(trimmed, `id="1:1"`) || strings.Contains(trimmed, `id="1:40"`) {
		t.Fatalf("expected earliest rows preserved and latest rows removed")
	}
	if strings.Count(trimmed, "<Row ") != strings.Count(trimmed, "</Row>") {
		t.Fatalf("row trimming left an incomplete tag:\n%s", trimmed)
	}
}

func TestLimitTableRowsKeepsFirstTwelvePerTable(t *testing.T) {
	var snapshot strings.Builder
	for tableID := 1; tableID <= 2; tableID++ {
		fmt.Fprintf(&snapshot, "<Table id=\"%d\"><table><TableBody id=\"%d\" methods=\"selectRow\">\n", tableID, tableID)
		for rowID := 1; rowID <= 15; rowID++ {
			fmt.Fprintf(&snapshot, "<Row id=\"%d:%d\"><td>row %d</td></Row>\n", tableID, rowID, rowID)
		}
		snapshot.WriteString("</TableBody></table></Table>\n")
	}

	limited, removedRows := limitTableRows(snapshot.String(), pageSnapshotMaxTableRows)
	if removedRows != 6 || strings.Count(limited, "<Row ") != 24 {
		t.Fatalf("unexpected table row limit: removed=%d remaining=%d", removedRows, strings.Count(limited, "<Row "))
	}
	for tableID := 1; tableID <= 2; tableID++ {
		if !strings.Contains(limited, fmt.Sprintf(`id="%d:12"`, tableID)) || strings.Contains(limited, fmt.Sprintf(`id="%d:13"`, tableID)) {
			t.Fatalf("table %d did not preserve only its first twelve rows", tableID)
		}
	}
}

func TestFindTableRowSpansIgnoresRowsOutsideTablesAndHandlesMultipleTables(t *testing.T) {
	snapshot := `<Row id="outside"/>
<TableBody id="1" methods="selectRow"><Row id="1:1">first</Row></TableBody>
<TableBody id="2" methods="selectRow"><Row id="2:1"/><Row id="2:2">last</Row></TableBody>`
	rows := findTableRowSpans(snapshot)
	if len(rows) != 3 {
		t.Fatalf("expected three rows inside tables, got %d", len(rows))
	}
	lastRow := snapshot[rows[2].start:rows[2].end]
	if !strings.Contains(lastRow, `id="2:2"`) {
		t.Fatalf("unexpected final row span: %q", lastRow)
	}
}
