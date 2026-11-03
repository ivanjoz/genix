package types

import "app/db"

// Planned → Active → Closed. One active sprint per project; a closed sprint is read only.
const (
	SprintStatePlanned int8 = 1
	SprintStateActive  int8 = 2
	SprintStateClosed  int8 = 3
)

// SprintStory is one story on a sprint's board, with its state inside this sprint.
type SprintStory struct {
	StoryID int64 `cb:"1"`
	State   int8  `cb:"2"`
	// Rank orders the cards of a column ascending.
	Rank float64 `cb:"3"`
	// AddedAt / StartedAt / DoneAt are SUnixTimes. AddedAt decides the story's final state (the
	// sprint it was most recently added to); DoneAt feeds the calendar and the burndown.
	AddedAt   int32 `cb:"4"`
	StartedAt int32 `cb:"5"`
	DoneAt    int32 `cb:"6"`
	// CarriedToSprintID: when the sprint closed unfinished, the sprint it went on in (0 = back to
	// the backlog). Its board shows it "Not completed".
	CarriedToSprintID int64 `cb:"7"`
}

type Sprint struct {
	db.TableStruct[SprintTable, Sprint]
	CompanyID int32  `cb:"1"`
	ID        int64  `cb:"2"`
	ProjectID int32  `cb:"3"`
	Name      string `cb:"4"`
	Goal      string `cb:"5"`
	// StartDate / EndDate are UnixDays.
	StartDate int16         `cb:"6"`
	EndDate   int16         `cb:"7"`
	State     int8          `cb:"8"`
	Stories   []SprintStory `cb:"9"`
	// ClosedAt is a SUnixTime.
	ClosedAt  int32   `cb:"10"`
	ClosedBy  int32   `cb:"11"`
	UpdatedBy int32   `cb:"12"`
	Status    int8    `json:"ss" cb:"ss,13"`
	Updated   int32   `json:"upd" cb:"upd,14"`
	Version   int16   `json:"ver" cb:"ver,15"`
	Metadata  *string `json:"mdt" cb:"mdt,16"`
}

type SprintTable struct {
	db.TableStruct[SprintTable, Sprint]
	CompanyID db.Col[*SprintTable, int32]
	ID        db.Col[*SprintTable, int64]
	ProjectID db.Col[*SprintTable, int32]
	Name      db.Col[*SprintTable, string]
	Goal      db.Col[*SprintTable, string]
	StartDate db.Col[*SprintTable, int16]
	EndDate   db.Col[*SprintTable, int16]
	State     db.Col[*SprintTable, int8]
	Stories   db.Col[*SprintTable, []SprintStory]
	ClosedAt  db.Col[*SprintTable, int32]
	ClosedBy  db.Col[*SprintTable, int32]
	UpdatedBy db.Col[*SprintTable, int32]
	Status    db.Col[*SprintTable, int8]
	Updated   db.Col[*SprintTable, int32]
	Version   db.Col[*SprintTable, int16]
}

// The board's moves rewrite this record under the project lock, so two people moving cards at once
// never lose a move. The delta index pinned on ProjectID syncs the browser cache per project.
func (e SprintTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                63,
		Name:              "projects_sprint",
		Partition:         e.CompanyID,
		Keys:              db.Cols(e.ID),
		KeyIntPacking:     db.Cols(e.ProjectID.Size(63-projectSequenceBits), e.Autoincrement(0)),
		AutoincrementPart: e.ProjectID,
		FixedValues:       []db.FixedValues{{Col: e.Status, Values: statusValues}},
		// The delta view also serves the plain reads by ProjectID and Status; a read without Status fans out over its FixedValues.
		Indexes: []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.ProjectID, e.Status)}},
	}
}
