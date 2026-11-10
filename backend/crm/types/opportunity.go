package types

import "app/db"

const (
	OpportunityOutcomeOpen int8 = 0
	OpportunityOutcomeWon  int8 = 1
	OpportunityOutcomeLost int8 = 2
)

// Opportunity Status is derived from Outcome by SelfParse: it is the last key of the board's
// delta index, so closing an opportunity (or moving it to another pipeline) makes it leave the
// board's subset and reach the browser as a removed ID.
const (
	OpportunityStatusOpen   int8 = 1
	OpportunityStatusClosed int8 = 2
)

// CrmOpportunity sits in exactly one pipeline and one stage; won / lost is an outcome, not a stage.
type CrmOpportunity struct {
	db.TableStruct[CrmOpportunityTable, CrmOpportunity]
	CompanyID         int32  `cb:"1"`
	ID                int32  `cb:"2"`
	PipelineID        int16  `cb:"3"`
	StageID           int16  `cb:"4"`
	Title             string `cb:"5"`
	ClientID          int32  `cb:"6"`
	OwnerUserID       int32  `cb:"7"`  // assigned sales rep
	Amount            int32  `cb:"8"`  // company currency, same scale as SaleOrder amounts
	Probability       int8   `cb:"9"`  // starts at the stage default, user can override
	ExpectedCloseDate int16  `cb:"10"` // UnixDay
	Outcome           int8   `cb:"11"`
	LostReasonID      int16  `cb:"12"`
	LostNote          string `cb:"13"`
	ClosedTime        int32  `cb:"14"` // SUnixTime
	StageEnteredTime  int32  `cb:"15"` // SUnixTime; drives the "stale" flag
	// Where the opportunity was before its last move, denormalized from the stage log so the
	// card can show "from <pipeline> · <stage>" without reading the log. 0 = never moved.
	PreviousPipelineID int16                       `cb:"16"`
	PreviousStageID    int16                       `cb:"17"`
	CustomValues       []CrmCustomFieldValue       `cb:"18"`
	Summary            []CrmOpportunityStageSumary `cb:"26"`
	Created            int32                       `cb:"19"`
	CreatedBy          int32                       `cb:"20"`
	Status             int8                        `json:"ss" cb:"ss,21"`
	Updated            int32                       `json:"upd" cb:"upd,22"`
	Version            int16                       `json:"ver" cb:"ver,23"`
	UpdatedBy          int32                       `cb:"24"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,25"`
}

type CrmOpportunityTable struct {
	db.TableStruct[CrmOpportunityTable, CrmOpportunity]
	CompanyID          db.Col[*CrmOpportunityTable, int32]
	ID                 db.Col[*CrmOpportunityTable, int32]
	PipelineID         db.Col[*CrmOpportunityTable, int16]
	StageID            db.Col[*CrmOpportunityTable, int16]
	Title              db.Col[*CrmOpportunityTable, string]
	ClientID           db.Col[*CrmOpportunityTable, int32]
	OwnerUserID        db.Col[*CrmOpportunityTable, int32]
	Amount             db.Col[*CrmOpportunityTable, int32]
	Probability        db.Col[*CrmOpportunityTable, int8]
	ExpectedCloseDate  db.Col[*CrmOpportunityTable, int16]
	Outcome            db.Col[*CrmOpportunityTable, int8]
	LostReasonID       db.Col[*CrmOpportunityTable, int16]
	LostNote           db.Col[*CrmOpportunityTable, string]
	ClosedTime         db.Col[*CrmOpportunityTable, int32]
	StageEnteredTime   db.Col[*CrmOpportunityTable, int32]
	PreviousPipelineID db.Col[*CrmOpportunityTable, int16]
	PreviousStageID    db.Col[*CrmOpportunityTable, int16]
	CustomValues       db.Col[*CrmOpportunityTable, []CrmCustomFieldValue]
	Created            db.Col[*CrmOpportunityTable, int32]
	CreatedBy          db.Col[*CrmOpportunityTable, int32]
	Status             db.Col[*CrmOpportunityTable, int8]
	Updated            db.Col[*CrmOpportunityTable, int32]
	Version            db.Col[*CrmOpportunityTable, int16]
	UpdatedBy          db.Col[*CrmOpportunityTable, int32]
}

type CrmOpportunityStageSumary struct {
	PipelineID int16 `json:"p" cb:"1"`
	StageID    int16 `json:"s" cb:"2"`
	DaysPassed   int16 `json:"d" cb:"3"`
}

func (e *CrmOpportunity) SelfParse() {
	e.Status = OpportunityStatusOpen
	if e.Outcome != OpportunityOutcomeOpen {
		e.Status = OpportunityStatusClosed
	}
}

func (e CrmOpportunityTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        66,
		Name:      "crm_opportunities",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{1, 2}},
			{Col: e.PipelineID, Min: 1, Max: 1<<15 - 1},
		},
		Indexes: []db.Index{
			// One board = one pinned PipelineID; Status (the last key) lists the open cards.
			{Type: db.TypeDelta, Keys: db.Cols(e.PipelineID, e.Status)},
			// Lets a pipeline edit find the opportunities still sitting on a stage.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.PipelineID)},
		},
	}
}

// CrmOpportunityMovePayload is the body of POST.crm-opportunity-move.
type CrmOpportunityMovePayload struct {
	ID         int32
	PipelineID int16
	StageID    int16
}

// CrmOpportunityClosePayload is the body of POST.crm-opportunity-close. Outcome 0 reopens.
type CrmOpportunityClosePayload struct {
	ID           int32
	Outcome      int8
	LostReasonID int16
	LostNote     string
}
