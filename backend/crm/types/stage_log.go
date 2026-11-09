package types

import "app/db"

// CrmOpportunityStageLog is append-only: one row per change (created, stage move, pipeline move,
// won, lost, reopened). It is written in the same transaction as the opportunity update and
// feeds the conversion and time-in-stage reports.
type CrmOpportunityStageLog struct {
	db.TableStruct[CrmOpportunityStageLogTable, CrmOpportunityStageLog]
	CompanyID      int32 `cb:"1"`
	ID             int32 `cb:"2"`
	OpportunityID  int32 `cb:"3"`
	FromPipelineID int16 `cb:"4"`
	FromStageID    int16 `cb:"5"`
	ToPipelineID   int16 `cb:"6"`
	ToStageID      int16 `cb:"7"`
	Outcome        int8  `cb:"8"` // outcome after the change
	Time           int32 `cb:"9"` // SUnixTime
	UserID         int32 `cb:"10"`
}

type CrmOpportunityStageLogTable struct {
	db.TableStruct[CrmOpportunityStageLogTable, CrmOpportunityStageLog]
	CompanyID      db.Col[*CrmOpportunityStageLogTable, int32]
	ID             db.Col[*CrmOpportunityStageLogTable, int32]
	OpportunityID  db.Col[*CrmOpportunityStageLogTable, int32]
	FromPipelineID db.Col[*CrmOpportunityStageLogTable, int16]
	FromStageID    db.Col[*CrmOpportunityStageLogTable, int16]
	ToPipelineID   db.Col[*CrmOpportunityStageLogTable, int16]
	ToStageID      db.Col[*CrmOpportunityStageLogTable, int16]
	Outcome        db.Col[*CrmOpportunityStageLogTable, int8]
	Time           db.Col[*CrmOpportunityStageLogTable, int32]
	UserID         db.Col[*CrmOpportunityStageLogTable, int32]
}

func (e CrmOpportunityStageLogTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        67,
		Name:      "crm_opportunity_stage_log",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		// History shown in the opportunity panel. The by-month report index comes with the
		// reports phase.
		Indexes: []db.Index{{Type: db.TypeLocalIndex, Keys: db.Cols(e.OpportunityID)}},
	}
}
