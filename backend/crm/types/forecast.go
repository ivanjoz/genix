package types

import "app/db"

// Forecast categories of an open stage (plan §3.3). Won is always "closed" and lost "omitted":
// those are outcomes, not stages, so they are not selectable.
const (
	ForecastCategoryPipeline int8 = 1
	ForecastCategoryBestCase int8 = 2
	ForecastCategoryCommit   int8 = 3
)

// CrmForecastSubmission is what a seller (or a team leader) declared for one month. The computed
// forecast changes every day, so each submission is a new row and the report can show how the
// forecast evolved against what finally happened (plan §3.4).
type CrmForecastSubmission struct {
	db.TableStruct[CrmForecastSubmissionTable, CrmForecastSubmission]
	CompanyID  int32 `cb:"1"`
	ID         int32 `cb:"2"`
	PipelineID int16 `cb:"3"`
	UserID     int32 `cb:"4"` // the seller, or the team leader for a team submission
	TeamID     int16 `cb:"5"` // 0 = individual submission
	Period     int16 `cb:"6"` // the goal period being forecast (GoalPeriodType: YYMM month, week or day)
	// Week is the core.WeekOfDay of the period's first day: the step key of the submissions' cache
	// groups, so a monthly, weekly or daily period is fetched as the weeks it spans.
	Week int16 `cb:"20"`
	// Computed by the server at submission time, never trusted from the client.
	ComputedCommit   int32 `cb:"7"`
	ComputedBestCase int32 `cb:"8"`
	ComputedPipeline int32 `cb:"9"`
	ComputedClosed   int32 `cb:"10"`
	// Declared by the seller or leader; defaults to the computed values.
	CommitAmount   int32  `cb:"11"`
	BestCaseAmount int32  `cb:"12"`
	Note           string `cb:"13"`
	Created        int32  `cb:"14"` // SUnixTime
	Status         int8   `json:"ss" cb:"ss,15"`
	Updated        int32  `json:"upd" cb:"upd,16"`
	Version        int16  `json:"ver" cb:"ver,17"`
	UpdatedBy      int32  `cb:"18"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,19"`
}

type CrmForecastSubmissionTable struct {
	db.TableStruct[CrmForecastSubmissionTable, CrmForecastSubmission]
	CompanyID        db.Col[*CrmForecastSubmissionTable, int32]
	ID               db.Col[*CrmForecastSubmissionTable, int32]
	PipelineID       db.Col[*CrmForecastSubmissionTable, int16]
	UserID           db.Col[*CrmForecastSubmissionTable, int32]
	TeamID           db.Col[*CrmForecastSubmissionTable, int16]
	Period           db.Col[*CrmForecastSubmissionTable, int16]
	Week             db.Col[*CrmForecastSubmissionTable, int16]
	ComputedCommit   db.Col[*CrmForecastSubmissionTable, int32]
	ComputedBestCase db.Col[*CrmForecastSubmissionTable, int32]
	ComputedPipeline db.Col[*CrmForecastSubmissionTable, int32]
	ComputedClosed   db.Col[*CrmForecastSubmissionTable, int32]
	CommitAmount     db.Col[*CrmForecastSubmissionTable, int32]
	BestCaseAmount   db.Col[*CrmForecastSubmissionTable, int32]
	Note             db.Col[*CrmForecastSubmissionTable, string]
	Created          db.Col[*CrmForecastSubmissionTable, int32]
	Status           db.Col[*CrmForecastSubmissionTable, int8]
	Updated          db.Col[*CrmForecastSubmissionTable, int32]
	Version          db.Col[*CrmForecastSubmissionTable, int16]
	UpdatedBy        db.Col[*CrmForecastSubmissionTable, int32]
}

func (e CrmForecastSubmissionTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          80,
		Name:        "crm_forecast_submissions",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{1}}},
		// The Forecast page reads the submissions of one pipeline and period as the weeks the period
		// spans: one cache group per week, append-only, so a past week is downloaded once.
		Indexes: []db.Index{{Type: db.TypeCacheGroup, Keys: db.Cols(e.Week, e.PipelineID)}},
	}
}

// CrmForecastSubmissionPayload is the body of POST.crm-forecast-submission. Only the declared
// amounts and the note come from the client.
type CrmForecastSubmissionPayload struct {
	PipelineID     int16
	Period         int16
	TeamID         int16 // 0 = the caller's own forecast; otherwise a team the caller leads
	CommitAmount   int32
	BestCaseAmount int32
	Note           string
}
