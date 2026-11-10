package types

import "app/db"

// Goal metrics (plan §3.2). The IDs are persisted; never renumber them.
const (
	CrmGoalMetricWonAmount        int8 = 1
	CrmGoalMetricWonCount         int8 = 2
	CrmGoalMetricNewOpportunities int8 = 3
	CrmGoalMetricActivities       int8 = 4
)

// CrmGoal is the target of one seller, in one pipeline, for one month and one metric. Target is in
// the amount scale of the opportunities for the money metric and a plain count otherwise.
type CrmGoal struct {
	db.TableStruct[CrmGoalTable, CrmGoal]
	CompanyID  int32 `cb:"1"`
	ID         int32 `cb:"2"`
	PipelineID int16 `cb:"3"`
	UserID     int32 `cb:"4"`
	Period     int16 `cb:"5"` // YYMM
	Metric     int8  `cb:"6"`
	Target     int32 `cb:"7"`
	Status     int8  `json:"ss" cb:"ss,8"`
	Updated    int32 `json:"upd" cb:"upd,9"`
	Version    int16 `json:"ver" cb:"ver,10"`
	UpdatedBy  int32 `cb:"11"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,12"`
}

type CrmGoalTable struct {
	db.TableStruct[CrmGoalTable, CrmGoal]
	CompanyID  db.Col[*CrmGoalTable, int32]
	ID         db.Col[*CrmGoalTable, int32]
	PipelineID db.Col[*CrmGoalTable, int16]
	UserID     db.Col[*CrmGoalTable, int32]
	Period     db.Col[*CrmGoalTable, int16]
	Metric     db.Col[*CrmGoalTable, int8]
	Target     db.Col[*CrmGoalTable, int32]
	Status     db.Col[*CrmGoalTable, int8]
	Updated    db.Col[*CrmGoalTable, int32]
	Version    db.Col[*CrmGoalTable, int16]
	UpdatedBy  db.Col[*CrmGoalTable, int32]
}

func (e CrmGoalTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          79,
		Name:        "crm_goals",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		// Sellers x months x metrics per pipeline stays small; the whole table syncs.
		Indexes: []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}

// CrmGoalsPayload is the body of POST.crm-goal: the cells the grid changed. A Target of 0 clears
// the cell (soft delete).
type CrmGoalsPayload struct {
	Goals []CrmGoal
}
