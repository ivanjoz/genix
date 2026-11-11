package types

import "app/db"

const (
	// CrmStatusDeleted is a soft delete: history keeps the labels of stages and pipelines.
	CrmStatusDeleted int8 = 0
	CrmStatusActive  int8 = 1
)

// Pipeline member levels, ordered: each level can do everything the lower ones can.
const (
	PipelineLevelView int8 = 1
	PipelineLevelEdit int8 = 2
	PipelineLevelFull int8 = 3
)

// Goal period types: the unit a pipeline sets its goals and forecast in. The two daily types
// both store one goal per day (the Period is a UnixDay); they differ in how the goals grid groups
// the days: a month at a time or a week at a time. The IDs are persisted; never renumber them.
const (
	GoalPeriodMonthly      int8 = 1
	GoalPeriodWeekly       int8 = 2
	GoalPeriodDailyByMonth int8 = 3
	GoalPeriodDailyByWeek  int8 = 4
)

// CrmPipeline is one kanban board. Its stages are embedded (stable IDs, one atomic save per
// reorder) and its Members decide who can see and work in it; see docs/CRM_PLAN.md §2.
type CrmPipeline struct {
	db.TableStruct[CrmPipelineTable, CrmPipeline]
	CompanyID   int32  `cb:"1"`
	ID          int16  `cb:"2"`
	Name        string `cb:"3"`
	Description string `cb:"4"`
	Color       string `cb:"5"`
	SortOrder   int16  `cb:"6"`
	// Board columns in display order. Stage IDs are stable and allocated by the server
	// (max+1 inside the pipeline); opportunities and the stage log reference them.
	Stages []CrmPipelineStage `cb:"7"`
	// Custom fields this pipeline shows, in display order.
	Fields []CrmPipelineField `cb:"8"`
	// Written ONLY by POST.crm-pipeline-members, never by POST.crm-pipeline, so the settings
	// page and the access page cannot overwrite each other.
	Members   []CrmPipelineMember `cb:"9"`
	Status    int8                `json:"ss" cb:"ss,10"`
	Updated   int32               `json:"upd" cb:"upd,11"`
	Version   int16               `json:"ver" cb:"ver,12"`
	UpdatedBy int32               `cb:"13"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,14"`
	// GoalPeriodType is one of the GoalPeriod* constants: the unit of this pipeline's goals and
	// forecast submissions (a CrmGoal.Period is a YYMM month, a core.WeekOfDay week or a UnixDay).
	GoalPeriodType int8 `cb:"15"`
}

type CrmPipelineStage struct {
	ID          int16  `cb:"1"`
	Name        string `cb:"2"`
	Color       string `cb:"3"`
	Probability int8   `cb:"4"` // default % for opportunities entering the stage
	StaleDays   int16  `cb:"5"` // 0 = never flagged
	Status      int8   `cb:"6"` // soft delete; blocked while the stage holds open opportunities
	// ForecastCategory is ForecastCategoryPipeline / BestCase / Commit: how the forecast counts
	// the open opportunities sitting in this stage.
	ForecastCategory int8 `cb:"7"`
	// RequiredStandardFields are the standard fields (StandardField*) an opportunity must have to
	// enter this stage or any later one by board position, so skipping the stage does not skip the rule.
	RequiredStandardFields []int8 `cb:"8"`
	// ApproverUserID is the user who must approve a move into this stage (0 = no approval): the
	// move waits as an approval task instead of happening (plan §4.2).
	ApproverUserID int32 `cb:"9"`
	// TaskTemplates become pending tasks of the opportunity each time it enters the stage. They
	// never run on their own: a user executes or dismisses them from the Tasks page (plan §5).
	TaskTemplates []CrmStageTaskTemplate `cb:"10"`
}

// CrmStageTaskTemplate is one task a stage proposes when an opportunity enters it: log an activity,
// or send an email template. "Send quote" joins with quotes.
type CrmStageTaskTemplate struct {
	ID              int16  `cb:"1"` // stable inside the stage, allocated by the server
	Kind            int8   `cb:"2"` // TaskKindActivity or TaskKindSendEmail
	Title           string `cb:"3"`
	ActivityTypeID  int16  `cb:"4"` // TaskKindActivity: the activity type asked for (system or company type)
	DueDays         int16  `cb:"5"` // due = day of entering + DueDays; 0 = the same day
	AssignTo        int8   `cb:"6"` // TaskAssignOwner or TaskAssignUser
	AssignUserID    int32  `cb:"7"` // when AssignTo is TaskAssignUser
	EmailTemplateID int16  `cb:"8"` // TaskKindSendEmail: the template the email starts from
}

// Standard opportunity fields a stage can require (CrmPipelineStage.RequiredStandardFields). The
// custom-field catalog only holds custom fields, so these two need their own list. Persisted IDs.
const (
	StandardFieldAmount            int8 = 1
	StandardFieldExpectedCloseDate int8 = 2
)

type CrmPipelineField struct {
	FieldID int16 `cb:"1"`
	// RequiredFromStageID: 0 = optional; otherwise the field is required to enter that stage or any
	// later one by board position. The pipeline's first stage means "required from creation".
	RequiredFromStageID int16 `cb:"2"`
}

type CrmPipelineMember struct {
	UserID int32 `cb:"1"`
	Level  int8  `cb:"2"`
}

// CrmPipelineMembersPayload is the body of POST.crm-pipeline-members.
type CrmPipelineMembersPayload struct {
	PipelineID int16
	Members    []CrmPipelineMember
}

type CrmPipelineTable struct {
	db.TableStruct[CrmPipelineTable, CrmPipeline]
	CompanyID      db.Col[*CrmPipelineTable, int32]
	ID             db.Col[*CrmPipelineTable, int16]
	Name           db.Col[*CrmPipelineTable, string]
	Description    db.Col[*CrmPipelineTable, string]
	Color          db.Col[*CrmPipelineTable, string]
	SortOrder      db.Col[*CrmPipelineTable, int16]
	Stages         db.Col[*CrmPipelineTable, []CrmPipelineStage]
	Fields         db.Col[*CrmPipelineTable, []CrmPipelineField]
	Members        db.Col[*CrmPipelineTable, []CrmPipelineMember]
	GoalPeriodType db.Col[*CrmPipelineTable, int8]
	Status         db.Col[*CrmPipelineTable, int8]
	Updated        db.Col[*CrmPipelineTable, int32]
	Version        db.Col[*CrmPipelineTable, int16]
	UpdatedBy      db.Col[*CrmPipelineTable, int32]
}

func (e CrmPipelineTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          65,
		Name:        "crm_pipelines",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		// A handful of pipelines per company: the whole table syncs to every CRM user.
		Indexes: []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}
