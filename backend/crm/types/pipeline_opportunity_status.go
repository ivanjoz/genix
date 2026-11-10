package types

import "app/db"

// Outcome of a stage visit. 0..2 mirror the opportunity's Outcome; VisitOutcomeLeft closes the
// pipeline's card when the opportunity moves to another pipeline.
const VisitOutcomeLeft int8 = 3

// CrmPipelineOpportunityStatus is the path of one opportunity through one pipeline: the stages it
// visited, in order, with won / lost / left as dated visits too. An opportunity that crosses
// pipelines has one row in each. It is what the timeline (Gantt) report reads instead of the stage
// log: one small row per card, synced per pipeline by delta, filtered by StartDate / LastDate in
// the browser and joined there with crm_opportunities (title, owner).
type CrmPipelineOpportunityStatus struct {
	db.TableStruct[CrmPipelineOpportunityStatusTable, CrmPipelineOpportunityStatus]
	CompanyID     int32 `cb:"1"`
	ID            int32 `cb:"2"`
	PipelineID    int16 `cb:"3"`
	OpportunityID int32 `cb:"4"`
	StartDate     int16 `cb:"5"` // UnixDay it entered the pipeline (the first time)
	LastDate      int16 `cb:"6"` // UnixDay of its last change in the pipeline
	// Visits in order; a visit lasts until the next one starts.
	Visits    []CrmOpportunityStageVisit `cb:"7"`
	Status    int8                       `json:"ss" cb:"ss,8"`
	Updated   int32                      `json:"upd" cb:"upd,9"`
	Version   int16                      `json:"ver" cb:"ver,10"`
	UpdatedBy int32                      `cb:"11"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,12"`
}

// CrmOpportunityStageVisit is one state the opportunity entered in the row's pipeline.
type CrmOpportunityStageVisit struct {
	StageID int16 `cb:"1"`
	Outcome int8  `cb:"2"` // 0 open, 1 won, 2 lost, VisitOutcomeLeft
	// Days since the row's StartDate: small numbers, which colbin stores in fewer bytes.
	DaysPassed int16 `cb:"3"`
}

type CrmPipelineOpportunityStatusTable struct {
	db.TableStruct[CrmPipelineOpportunityStatusTable, CrmPipelineOpportunityStatus]
	CompanyID     db.Col[*CrmPipelineOpportunityStatusTable, int32]
	ID            db.Col[*CrmPipelineOpportunityStatusTable, int32]
	PipelineID    db.Col[*CrmPipelineOpportunityStatusTable, int16]
	OpportunityID db.Col[*CrmPipelineOpportunityStatusTable, int32]
	StartDate     db.Col[*CrmPipelineOpportunityStatusTable, int16]
	LastDate      db.Col[*CrmPipelineOpportunityStatusTable, int16]
	Visits        db.Col[*CrmPipelineOpportunityStatusTable, []CrmOpportunityStageVisit]
	Status        db.Col[*CrmPipelineOpportunityStatusTable, int8]
	Updated       db.Col[*CrmPipelineOpportunityStatusTable, int32]
	Version       db.Col[*CrmPipelineOpportunityStatusTable, int16]
	UpdatedBy     db.Col[*CrmPipelineOpportunityStatusTable, int32]
}

func (e CrmPipelineOpportunityStatusTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        84,
		Name:      "crm_pipeline_opportunity_status",
		Partition: e.CompanyID,
		// A delta index needs one integer key; (PipelineID, OpportunityID) is unique by the writer.
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{1}}},
		Indexes: []db.Index{
			// The timeline syncs one pipeline's rows; Status is always 1 (a row is never deleted).
			{Type: db.TypeDelta, Keys: db.Cols(e.PipelineID, e.Status)},
			// The writer finds an opportunity's rows; the by-IDs route checks its pipelines.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.OpportunityID)},
		},
	}
}
