package types

import "app/db"

// Task kinds. "Send quote" (2) joins with quotes, so its number stays free.
const (
	TaskKindApproval  int8 = 1
	TaskKindSendEmail int8 = 3
	TaskKindActivity  int8 = 4
)

// CrmTask.TaskStatus. Pending is the only open state; every other one closes the task.
const (
	TaskStatusPending   int8 = 1
	TaskStatusDone      int8 = 2
	TaskStatusDismissed int8 = 3
	TaskStatusApproved  int8 = 4
	TaskStatusRejected  int8 = 5
	TaskStatusCancelled int8 = 6
)

// CrmTask.Status is derived from TaskStatus by SelfParse: it is the last key of the assignee's
// delta index, so closing a task makes it leave the open list as a removed ID.
const (
	TaskOpen   int8 = 1
	TaskClosed int8 = 2
)

// CrmStageTaskTemplate.AssignTo
const (
	TaskAssignOwner int8 = 1 // the opportunity's seller when it enters the stage
	TaskAssignUser  int8 = 2 // a fixed user
)

// CrmTaskActionPayload is the body of POST.crm-task.
type CrmTaskActionPayload struct {
	ID     int32
	Action int8 // TaskAction*
	Note   string
}

// CrmTaskActionPayload.Action
const (
	TaskActionDismiss       int8 = 1 // the assignee drops a template task
	TaskActionCancelRequest int8 = 2 // the requester withdraws an approval request
)

// CrmTaskDecisionPayload is the body of POST.crm-task-decide.
type CrmTaskDecisionPayload struct {
	ID      int32
	Approve bool
	Note    string
}

// CrmTask is something a user has to do for an opportunity: approve a move, or do what a stage
// template proposed. Tasks are proposed, never run automatically (plan §5).
type CrmTask struct {
	db.TableStruct[CrmTaskTable, CrmTask]
	CompanyID     int32 `cb:"1"`
	ID            int32 `cb:"2"`
	OpportunityID int32 `cb:"3"`
	// PipelineID and StageID: the stage that generated the task. For an approval they are the
	// move's target, which may be in another pipeline.
	PipelineID     int16  `cb:"4"`
	StageID        int16  `cb:"5"`
	Kind           int8   `cb:"6"` // TaskKind*
	TemplateID     int16  `cb:"7"` // CrmStageTaskTemplate.ID; 0 for an approval
	Title          string `cb:"8"` // empty for an approval: the screen composes it from the stage
	AssignedUserID int32  `cb:"9"`
	DueDate        int16  `cb:"10"` // UnixDay
	TaskStatus     int8   `cb:"11"` // TaskStatus*
	ResultNote     string `cb:"12"`
	ResultRefID    int64  `cb:"13"` // what executing it produced: the activity ID
	Created        int32  `cb:"14"` // SUnixTime
	DoneTime       int32  `cb:"15"`
	DoneBy         int32  `cb:"16"`
	// CreatedBy is who asked: the requester of an approval (who may cancel it); 0 for a template task.
	CreatedBy int32   `cb:"17"`
	Status    int8    `json:"ss" cb:"ss,18"`
	Updated   int32   `json:"upd" cb:"upd,19"`
	Version   int16   `json:"ver" cb:"ver,20"`
	UpdatedBy int32   `cb:"21"`
	Metadata  *string `json:"mdt" cb:"mdt,22"`
	// What the task card shows of its opportunity, attached on read (not stored). Editing the
	// opportunity's title, client or amount bumps its open tasks' Updated so the delta resends them;
	// renaming the client does not.
	OpportunityTitle  string `db:"-" cb:"23"`
	ClientName        string `db:"-" cb:"24"`
	OpportunityAmount int32  `db:"-" cb:"25"`
}

// EditChangesTaskCard tells whether an opportunity edit changes what its tasks show.
func EditChangesTaskCard(before, after CrmOpportunity) bool {
	return before.Title != after.Title || before.ClientID != after.ClientID || before.Amount != after.Amount
}

type CrmTaskTable struct {
	db.TableStruct[CrmTaskTable, CrmTask]
	CompanyID      db.Col[*CrmTaskTable, int32]
	ID             db.Col[*CrmTaskTable, int32]
	OpportunityID  db.Col[*CrmTaskTable, int32]
	PipelineID     db.Col[*CrmTaskTable, int16]
	StageID        db.Col[*CrmTaskTable, int16]
	Kind           db.Col[*CrmTaskTable, int8]
	TemplateID     db.Col[*CrmTaskTable, int16]
	Title          db.Col[*CrmTaskTable, string]
	AssignedUserID db.Col[*CrmTaskTable, int32]
	DueDate        db.Col[*CrmTaskTable, int16]
	TaskStatus     db.Col[*CrmTaskTable, int8]
	ResultNote     db.Col[*CrmTaskTable, string]
	ResultRefID    db.Col[*CrmTaskTable, int64]
	Created        db.Col[*CrmTaskTable, int32]
	DoneTime       db.Col[*CrmTaskTable, int32]
	DoneBy         db.Col[*CrmTaskTable, int32]
	CreatedBy      db.Col[*CrmTaskTable, int32]
	Status         db.Col[*CrmTaskTable, int8]
	Updated        db.Col[*CrmTaskTable, int32]
	Version        db.Col[*CrmTaskTable, int16]
	UpdatedBy      db.Col[*CrmTaskTable, int32]
}

func (e *CrmTask) SelfParse() {
	e.Status = TaskOpen
	if e.TaskStatus != TaskStatusPending {
		e.Status = TaskClosed
	}
}

func (e CrmTaskTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        89,
		Name:      "crm_tasks",
		Partition: e.CompanyID,
		// The opportunity panel resolves the approval a card waits on by ID (GET.crm-task-ids).
		CacheByIDs: true,
		Keys:       db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{1, 2}},
			{Col: e.AssignedUserID, Min: 1, Max: 1<<24 - 1},
		},
		Indexes: []db.Index{
			// One user's list = one pinned AssignedUserID; Status (the last key) picks open or closed.
			{Type: db.TypeDelta, Keys: db.Cols(e.AssignedUserID, e.Status)},
			// The pending tasks of one opportunity: cancelled when it leaves the stage, and not
			// created twice when it re-enters one.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.OpportunityID)},
		},
	}
}
