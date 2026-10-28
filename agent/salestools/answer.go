// Package salestools is the agent's sales FrameSQL tool (backend/genix-orm/dataframe/framesql/
// FRAMESQL.md §6–7): the model writes one SQL statement over a sale DataFrame and how to show its
// result, and the tool's answer — a message, a table, a chart — is the turn's reply, with no other
// model call. Ported from berryapps' agenttools.
package salestools

import "time"

// Answer is the tool's result when it answered. The agent sends it as the turn's reply; Summary is
// what later turns read, since they never see the table.
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

// formatUnixDay renders a UnixDay as "YYYY-MM-DD". The day number is already a calendar date, so it
// is read in UTC: no offset shifts it.
func formatUnixDay(unixDay int64) string {
	return time.Unix(unixDay*86400, 0).UTC().Format(time.DateOnly)
}
