package types

import (
	"app/core"
	"app/db"
)

// What a stage log row records. A close, a reopen and a field edit keep the stage, so their
// From/To stage are equal; Kind tells them apart from a move without comparing fields.
const (
	StageLogKindCreated int8 = 1
	StageLogKindMoved   int8 = 2
	StageLogKindClosed  int8 = 3 // won, lost or reopened
	StageLogKindEdited  int8 = 4 // amount, expected close date or owner changed
)

// CrmOpportunityStageLog is append-only: one row per change of the opportunity's reportable state,
// written in the same transaction as the opportunity update.
//
// Every row carries the full state BEFORE and AFTER the change (pipeline, stage, funnel position,
// outcome, owner, amount, expected close date). A report for a period therefore aggregates that
// period's rows alone, without reading earlier rows of the same opportunity, which may sit in
// another month's cache group. See docs/CRM_REPORTS_QUOTES_PLAN.md §2 and §8.1.
type CrmOpportunityStageLog struct {
	db.TableStruct[CrmOpportunityStageLogTable, CrmOpportunityStageLog]
	CompanyID      int32 `cb:"1"`
	ID             int32 `cb:"2"`
	OpportunityID  int32 `cb:"3"`
	FromPipelineID int16 `cb:"4"`
	FromStageID    int16 `cb:"5"`
	ToPipelineID   int16 `cb:"6"`
	ToStageID      int16 `cb:"7"`
	ToOutcome      int8  `cb:"8"`
	Time           int32 `cb:"9"` // SUnixTime
	// UserID is who acted: audit only. Reports attribute to the owners (decision E1).
	UserID int32 `cb:"10"`
	Kind   int8  `cb:"11"`
	// Period is the YYMM of Time: the step key of the reports' cache groups.
	Period int16 `cb:"12"`
	// 1-based depth of the stage among the pipeline's active stages at the time of the change, so a
	// later reorder does not rewrite history (decision E2). 0 = no state (before a creation).
	FromPosition          int8  `cb:"13"`
	ToPosition            int8  `cb:"14"`
	FromOutcome           int8  `cb:"15"`
	FromOwnerUserID       int32 `cb:"16"`
	ToOwnerUserID         int32 `cb:"17"`
	FromAmount            int32 `cb:"18"`
	ToAmount              int32 `cb:"19"`
	FromExpectedCloseDate int16 `cb:"20"` // UnixDay
	ToExpectedCloseDate   int16 `cb:"21"` // UnixDay
	// LostReasonID is the reason of the after-state when it is lost; feeds the lost-reasons report.
	LostReasonID int16 `cb:"22"`
}

// PipelineIDs are the pipelines the row touches (one, or two for a transfer). It feeds the
// computed column the reports' cache groups are keyed by, so a pipeline's report also receives
// the rows that moved opportunities into or out of it.
func (e *CrmOpportunityStageLog) PipelineIDs() []int32 {
	pipelineIDs := []int32{int32(e.ToPipelineID)}
	if e.FromPipelineID != 0 && e.FromPipelineID != e.ToPipelineID {
		pipelineIDs = append(pipelineIDs, int32(e.FromPipelineID))
	}
	return pipelineIDs
}

type CrmOpportunityStageLogTable struct {
	db.TableStruct[CrmOpportunityStageLogTable, CrmOpportunityStageLog]
	CompanyID             db.Col[*CrmOpportunityStageLogTable, int32]
	ID                    db.Col[*CrmOpportunityStageLogTable, int32]
	OpportunityID         db.Col[*CrmOpportunityStageLogTable, int32]
	FromPipelineID        db.Col[*CrmOpportunityStageLogTable, int16]
	FromStageID           db.Col[*CrmOpportunityStageLogTable, int16]
	ToPipelineID          db.Col[*CrmOpportunityStageLogTable, int16]
	ToStageID             db.Col[*CrmOpportunityStageLogTable, int16]
	ToOutcome             db.Col[*CrmOpportunityStageLogTable, int8]
	Time                  db.Col[*CrmOpportunityStageLogTable, int32]
	UserID                db.Col[*CrmOpportunityStageLogTable, int32]
	Kind                  db.Col[*CrmOpportunityStageLogTable, int8]
	Period                db.Col[*CrmOpportunityStageLogTable, int16]
	FromPosition          db.Col[*CrmOpportunityStageLogTable, int8]
	ToPosition            db.Col[*CrmOpportunityStageLogTable, int8]
	FromOutcome           db.Col[*CrmOpportunityStageLogTable, int8]
	FromOwnerUserID       db.Col[*CrmOpportunityStageLogTable, int32]
	ToOwnerUserID         db.Col[*CrmOpportunityStageLogTable, int32]
	FromAmount            db.Col[*CrmOpportunityStageLogTable, int32]
	ToAmount              db.Col[*CrmOpportunityStageLogTable, int32]
	FromExpectedCloseDate db.Col[*CrmOpportunityStageLogTable, int16]
	ToExpectedCloseDate   db.Col[*CrmOpportunityStageLogTable, int16]
	LostReasonID          db.Col[*CrmOpportunityStageLogTable, int16]
	// PipelineIDs has no field in CrmOpportunityStageLog: GetSchema computes it.
	PipelineIDs db.Col[*CrmOpportunityStageLogTable, []int32]
}

func (e CrmOpportunityStageLogTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        67,
		Name:      "crm_opportunity_stage_log",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		Computed: []db.ComputedCol{
			{Col: e.PipelineIDs, From: db.Cols(e.FromPipelineID, e.ToPipelineID), Value: db.Compute((*CrmOpportunityStageLog).PipelineIDs)},
		},
		Indexes: []db.Index{
			// History shown in the opportunity panel: one group per opportunity, so the browser
			// downloads it again only after a new change of that opportunity. It is also the index
			// the server's OpportunityID reads use.
			{Type: db.TypeCacheGroup, Keys: db.Cols(e.OpportunityID)},
			// Reports: one group per month and pipeline. The log is append-only and always written
			// with the current time, so a closed month never changes and is downloaded once.
			{Type: db.TypeCacheGroup, Keys: db.Cols(e.Period, e.PipelineIDs)},
		},
	}
}

// StagePosition is the 1-based depth of a stage among the pipeline's active stages, in board
// order; 0 when the stage is not active in it.
func StagePosition(pipeline CrmPipeline, stageID int16) int8 {
	position := int8(0)
	for _, stage := range pipeline.Stages {
		if stage.Status != CrmStatusActive {
			continue
		}
		position++
		if stage.ID == stageID {
			return position
		}
	}
	return 0
}

// BuildStageLogRow is the one constructor of a log row. before is the stored opportunity (zero
// for a creation) in beforePipeline; after is the record being written, in afterPipeline.
func BuildStageLogRow(kind int8, before CrmOpportunity, beforePipeline CrmPipeline, after CrmOpportunity, afterPipeline CrmPipeline,
	userID, now int32) CrmOpportunityStageLog {

	row := CrmOpportunityStageLog{
		CompanyID: after.CompanyID, OpportunityID: after.ID, Kind: kind, Time: now, UserID: userID,
		Period:       core.PeriodOfDay(core.SUnixTimeToUnixDay(now)),
		ToPipelineID: after.PipelineID, ToStageID: after.StageID, ToPosition: StagePosition(afterPipeline, after.StageID),
		ToOutcome: after.Outcome, ToOwnerUserID: after.OwnerUserID, ToAmount: after.Amount,
		ToExpectedCloseDate: after.ExpectedCloseDate,
	}
	if after.Outcome == OpportunityOutcomeLost {
		row.LostReasonID = after.LostReasonID
	}
	if kind == StageLogKindCreated {
		return row
	}
	row.FromPipelineID, row.FromStageID = before.PipelineID, before.StageID
	row.FromPosition = StagePosition(beforePipeline, before.StageID)
	row.FromOutcome, row.FromOwnerUserID, row.FromAmount = before.Outcome, before.OwnerUserID, before.Amount
	row.FromExpectedCloseDate = before.ExpectedCloseDate
	return row
}

// EditChangesReportedState: an edit is logged only when it changes what reports read.
func EditChangesReportedState(before, after CrmOpportunity) bool {
	return before.Amount != after.Amount || before.OwnerUserID != after.OwnerUserID ||
		before.ExpectedCloseDate != after.ExpectedCloseDate
}
