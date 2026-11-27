// Package datafunctions is the catalog of data functions genix-bot finds with `search` and runs with
// `call` (genix-local-ai plan 13): each function is a spec whose arguments follow one set of
// conventions, and whose answer — a message, a table, a chart — is the turn's reply.
//
// It also holds what the Eino sales tool (salestools) shares with the catalog, in one copy: the
// period notation and its resolver, record-name resolution, and the Answer with its table and chart
// builders.
package datafunctions

import "time"

// Answer is a data answer: the agent sends it as the turn's reply; Summary is what later turns
// read, since they never see the table.
type Answer struct {
	Message string
	Summary string
	Tables  []Table `json:",omitempty"`
	Charts  []Chart `json:",omitempty"`
}

// Limits of what one answer shows, matched by the chat's renderer (frontend/core/agent/agent-answer.ts).
const (
	MaxTableColumns = 12
	MaxTableRows    = 400
	MaxChartSeries  = 12
)

// Table travels as columns: every column holds one value per row. Values are display units (soles,
// not cents): a table is a view, never a stored record.
type Table struct {
	Columns []TableColumn
}

type TableColumn struct {
	Label string
	// Format is "text", "number", "money" or "date".
	Format string
	// Values: text holds strings, date "YYYY-MM-DD" strings, number and money numbers; nil is no value.
	Values []any
	// InlineBar draws each row's value as a bar under the row, against the column's largest value.
	InlineBar bool `json:",omitempty"`
}

// Chart travels as columns too: the x column names the points and every other column is a series
// drawn against it.
type Chart struct {
	Columns []ChartColumn
}

type ChartColumn struct {
	Label string
	// Axis is "x", "y" or "y2" (a series on its own scale, like units beside money).
	Axis   string
	Values []any
	// Type is "bar" or "line" on a series; empty lets the renderer decide.
	Type string `json:",omitempty"`
	// Format is "number", "money" or "date"; empty is "number".
	Format string `json:",omitempty"`
}

// FormatUnixDay renders a UnixDay as "YYYY-MM-DD". The day number is already a calendar date, so it
// is read in UTC: no offset shifts it.
func FormatUnixDay(unixDay int64) string {
	return time.Unix(unixDay*86400, 0).UTC().Format(time.DateOnly)
}

// Localized picks the text of the answer's language: "en" or Spanish.
func Localized(language, spanish, english string) string {
	if language == "en" {
		return english
	}
	return spanish
}
