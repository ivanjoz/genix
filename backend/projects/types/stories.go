package types

import "app/db"

// The state of a story inside a sprint. The board has 4 columns: Stopped and InReview share the
// third one (Stopped on top, in red). StoryStateNotPlanned is only a final state: the story was
// never in a sprint.
const (
	StoryStateNotPlanned    int8 = 0
	StoryStatePending       int8 = 1
	StoryStateInDevelopment int8 = 2
	StoryStateStopped       int8 = 3
	StoryStateInReview      int8 = 4
	StoryStateFinished      int8 = 5
)

const (
	StoryTypeStory int8 = 1
	StoryTypeBug   int8 = 2
	StoryTypeTask  int8 = 3
)

const (
	StoryPriorityLow      int8 = 1
	StoryPriorityMedium   int8 = 2
	StoryPriorityHigh     int8 = 3
	StoryPriorityCritical int8 = 4
)

// Story holds the fields the backlog and the boards show: the backlog GET returns every story of
// the project. The long content lives in StoryDetail, read when the story is opened.
type Story struct {
	db.TableStruct[StoryTable, Story]
	CompanyID int32 `cb:"1"`
	// ID packs ProjectID and the project's own story sequence (StoryTable.GetSchema): the sequence
	// is the story's number inside its project, shown as H<number> (StoryNumber).
	ID        int64  `cb:"2"`
	ProjectID int32  `cb:"3"`
	Type      int8   `cb:"5"`
	Title     string `cb:"6"`
	Summary   string `cb:"7"`
	// EpicID 0 = no epic.
	EpicID   int64 `cb:"8"`
	Priority int8  `cb:"9"`
	// AssigneeID is the team member who develops it, 0 = unassigned.
	AssigneeID     int32   `cb:"10"`
	EstimatedHours float32 `cb:"11"`
	// BacklogRank orders the backlog ascending. A new story takes its number, so it lands last.
	BacklogRank float64 `cb:"12"`
	// State is the final state: the story's state in the sprint it was most recently added to
	// (storyFinalState). Derived data, written with every sprint change that touches the story.
	State        int8    `cb:"13"`
	LastSprintID int64   `cb:"14"`
	SprintIDs    []int64 `cb:"15"`
	// The counters let the cards show them without reading the detail.
	CommentsCount    int16 `cb:"16"`
	AttachmentsCount int16 `cb:"17"`
	// StateDates keeps the last day the final State entered each sprint state (setStoryState).
	StateDates []StoryStateDate `cb:"18"`
	CreatedBy  int32            `cb:"19"`
	// Created is a SUnixTime.
	Created   int32   `cb:"20"`
	UpdatedBy int32   `cb:"21"`
	Status    int8    `json:"ss" cb:"ss,22"`
	Updated   int32   `json:"upd" cb:"upd,23"`
	Version   int16   `json:"ver" cb:"ver,24"`
	Metadata  *string `json:"mdt" cb:"mdt,25"`
}

// StoryStateDate: the story was last put in State on Date, a UnixDay.
type StoryStateDate struct {
	State int8  `cb:"1"`
	Date  int16 `cb:"2"`
}

type StoryTable struct {
	db.TableStruct[StoryTable, Story]
	CompanyID        db.Col[*StoryTable, int32]
	ID               db.Col[*StoryTable, int64]
	ProjectID        db.Col[*StoryTable, int32]
	Type             db.Col[*StoryTable, int8]
	Title            db.Col[*StoryTable, string]
	Summary          db.Col[*StoryTable, string]
	EpicID           db.Col[*StoryTable, int64]
	Priority         db.Col[*StoryTable, int8]
	AssigneeID       db.Col[*StoryTable, int32]
	EstimatedHours   db.Col[*StoryTable, float32]
	BacklogRank      db.Col[*StoryTable, float64]
	State            db.Col[*StoryTable, int8]
	LastSprintID     db.Col[*StoryTable, int64]
	SprintIDs        db.ColSlice[*StoryTable, int64]
	CommentsCount    db.Col[*StoryTable, int16]
	AttachmentsCount db.Col[*StoryTable, int16]
	StateDates       db.Col[*StoryTable, []StoryStateDate]
	CreatedBy        db.Col[*StoryTable, int32]
	Created          db.Col[*StoryTable, int32]
	UpdatedBy        db.Col[*StoryTable, int32]
	Status           db.Col[*StoryTable, int8]
	Updated          db.Col[*StoryTable, int32]
	Version          db.Col[*StoryTable, int16]
}

// projectSequenceBits is the per-project sequence's slot of the ID of every record of a project
// (stories, epics, sprints, documents, activity): 8,388,607 of each per project. The ProjectID slot
// above it is wide only to leave these bits to the sequence; an ID stays below 2^53 (exact as a JS
// number) while project IDs stay below 2^30.
const projectSequenceBits = 23

// StoryNumber is the story's number inside its project: the sequence slot of its ID.
func StoryNumber(storyID int64) int32 {
	return int32(storyID & (1<<projectSequenceBits - 1))
}

// The ID's sequence runs per project (AutoincrementPart), so each project numbers its own stories
// 1, 2, 3... The delta index pinned on ProjectID syncs the browser cache per project. CacheByIDs backs
// GET.projects-stories-ids: stories by ID from any number of projects (the per-person report).
func (e StoryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                59,
		Name:              "projects_story",
		Partition:         e.CompanyID,
		Keys:              db.Cols(e.ID),
		KeyIntPacking:     db.Cols(e.ProjectID.Size(63-projectSequenceBits), e.Autoincrement(0)),
		AutoincrementPart: e.ProjectID,
		UseListAsDefault:  true,
		CacheByIDs:        true,
		FixedValues:       []db.FixedValues{{Col: e.Status, Values: statusValues}},
		// The delta view also serves the plain reads by ProjectID and Status; a read without Status fans out over its FixedValues.
		Indexes: []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.ProjectID, e.Status)}},
	}
}

// StoryAssignee is a story's current assignee, one row per story: the per-person index of stories
// across every project. Active while someone holds an active story.
type StoryAssignee struct {
	db.TableStruct[StoryAssigneeTable, StoryAssignee]
	CompanyID  int32   `cb:"1"`
	StoryID    int64   `cb:"2"`
	ProjectID  int32   `cb:"3"`
	AssigneeID int32   `cb:"4"`
	Status     int8    `json:"ss" cb:"ss,5"`
	Updated    int32   `json:"upd" cb:"upd,6"`
	Version    int16   `json:"ver" cb:"ver,7"`
	Metadata   *string `json:"mdt" cb:"mdt,8"`
}

type StoryAssigneeTable struct {
	db.TableStruct[StoryAssigneeTable, StoryAssignee]
	CompanyID  db.Col[*StoryAssigneeTable, int32]
	StoryID    db.Col[*StoryAssigneeTable, int64]
	ProjectID  db.Col[*StoryAssigneeTable, int32]
	AssigneeID db.Col[*StoryAssigneeTable, int32]
	Status     db.Col[*StoryAssigneeTable, int8]
	Updated    db.Col[*StoryAssigneeTable, int32]
	Version    db.Col[*StoryAssigneeTable, int16]
}

// The delta index pinned on AssigneeID keeps one change log per person: when a story changes
// hands, the ORM writes a deleted entry in the previous person's log and the row in the new one's,
// so both browsers' caches follow with a single write.
func (e StoryAssigneeTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          60,
		Name:        "projects_story_assignee",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.StoryID),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: statusValues}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.AssigneeID, e.Status)}},
	}
}

// Attachment is a file in the CDN bucket, public behind an unguessable key (attachmentObjectKey).
type Attachment struct {
	Key         string `cb:"1"`
	Name        string `cb:"2"`
	ContentType string `cb:"3"`
	Size        int32  `cb:"4"`
	UploadedBy  int32  `cb:"5"`
	// UploadedAt is a SUnixTime.
	UploadedAt int32 `cb:"6"`
	// DocumentID is the file's row on the Documents page (Document.ID), set by the backend once
	// that row is written.
	DocumentID int64 `cb:"7"`
}

type ChecklistItem struct {
	Text string `cb:"1"`
	Done bool   `cb:"2"`
}

// StoryDetail is the long content of a story, one per story, read when the story is opened.
type StoryDetail struct {
	db.TableStruct[StoryDetailTable, StoryDetail]
	CompanyID int32 `cb:"1"`
	StoryID   int64 `cb:"2"`
	ProjectID int32 `cb:"3"`
	// Description is the HTMLEditor's HTML.
	Description string `cb:"4"`
	// Checklist holds the acceptance criteria.
	Checklist   []ChecklistItem `cb:"5"`
	Attachments []Attachment    `cb:"6"`
	UpdatedBy   int32           `cb:"7"`
	Status      int8            `json:"ss" cb:"ss,8"`
	Updated     int32           `json:"upd" cb:"upd,9"`
}

type StoryDetailTable struct {
	db.TableStruct[StoryDetailTable, StoryDetail]
	CompanyID   db.Col[*StoryDetailTable, int32]
	StoryID     db.Col[*StoryDetailTable, int64]
	ProjectID   db.Col[*StoryDetailTable, int32]
	Description db.Col[*StoryDetailTable, string]
	Checklist   db.Col[*StoryDetailTable, []ChecklistItem]
	Attachments db.Col[*StoryDetailTable, []Attachment]
	UpdatedBy   db.Col[*StoryDetailTable, int32]
	Status      db.Col[*StoryDetailTable, int8]
	Updated     db.Col[*StoryDetailTable, int32]
}

// Keyed by the story itself, so it has no delta index (a delta log ID cannot be its own pinned
// key): GET.projects-story-detail reads the one row and answers it as a full replace.
func (e StoryDetailTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        61,
		Name:      "projects_story_detail",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.StoryID),
	}
}

// What a story's activity entry records. Comments and the history share one timeline.
const (
	ActivityKindComment           int8 = 1
	ActivityKindStateChange       int8 = 2 // FromState → ToState in SprintID; Text = the reason of a Stopped
	ActivityKindAssigneeChange    int8 = 3 // AssigneeID, 0 = unassigned
	ActivityKindAddedToSprint     int8 = 4 // SprintID
	ActivityKindRemovedFromSprint int8 = 5 // SprintID
	ActivityKindCarriedOver       int8 = 6 // from SprintID to ToSprintID, 0 = back to the backlog
)

// StoryActivity is one entry of a story's timeline: a comment or an event of its history. Entries
// are never edited.
type StoryActivity struct {
	db.TableStruct[StoryActivityTable, StoryActivity]
	CompanyID   int32        `cb:"1"`
	ID          int64        `cb:"2"`
	ProjectID   int32        `cb:"3"`
	StoryID     int64        `cb:"4"`
	Kind        int8         `cb:"5"`
	Text        string       `cb:"6"`
	FromState   int8         `cb:"7"`
	ToState     int8         `cb:"8"`
	SprintID    int64        `cb:"9"`
	ToSprintID  int64        `cb:"10"`
	AssigneeID  int32        `cb:"11"`
	Attachments []Attachment `cb:"12"`
	AuthorID    int32        `cb:"13"`
	// Created is a SUnixTime.
	Created  int32   `cb:"14"`
	Status   int8    `json:"ss" cb:"ss,15"`
	Updated  int32   `json:"upd" cb:"upd,16"`
	Version  int16   `json:"ver" cb:"ver,17"`
	Metadata *string `json:"mdt" cb:"mdt,18"`
}

type StoryActivityTable struct {
	db.TableStruct[StoryActivityTable, StoryActivity]
	CompanyID   db.Col[*StoryActivityTable, int32]
	ID          db.Col[*StoryActivityTable, int64]
	ProjectID   db.Col[*StoryActivityTable, int32]
	StoryID     db.Col[*StoryActivityTable, int64]
	Kind        db.Col[*StoryActivityTable, int8]
	Text        db.Col[*StoryActivityTable, string]
	FromState   db.Col[*StoryActivityTable, int8]
	ToState     db.Col[*StoryActivityTable, int8]
	SprintID    db.Col[*StoryActivityTable, int64]
	ToSprintID  db.Col[*StoryActivityTable, int64]
	AssigneeID  db.Col[*StoryActivityTable, int32]
	Attachments db.Col[*StoryActivityTable, []Attachment]
	AuthorID    db.Col[*StoryActivityTable, int32]
	Created     db.Col[*StoryActivityTable, int32]
	Status      db.Col[*StoryActivityTable, int8]
	Updated     db.Col[*StoryActivityTable, int32]
	Version     db.Col[*StoryActivityTable, int16]
}

// The IDs come from the table's autoincrement (a delta log needs one integer key). The delta index
// pinned on StoryID syncs the browser cache per story (GET.projects-story-activity).
func (e StoryActivityTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                62,
		Name:              "projects_story_activity",
		Partition:         e.CompanyID,
		Keys:              db.Cols(e.ID),
		KeyIntPacking:     db.Cols(e.ProjectID.Size(63-projectSequenceBits), e.Autoincrement(0)),
		AutoincrementPart: e.ProjectID,
		FixedValues:       []db.FixedValues{{Col: e.Status, Values: statusValues}},
		Indexes:           []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.StoryID, e.Status)}},
	}
}
