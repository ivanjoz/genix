package types

import "app/db"

const (
	StatusDeleted int8 = 0
	StatusActive  int8 = 1
)

// AccesoIDProjects is the "Proyectos" access of backend/access.toml. A user holding it at the write
// level (2+) can create projects and join a project's team.
const AccesoIDProjects int32 = 39

// statusValues are the Status values every table of the module holds: Delta() enumerates them.
var statusValues = []int64{int64(StatusDeleted), int64(StatusActive)}

// A project's team: members work on the backlog and the board; admins also delete stories, edit
// the project and its team, and run the sprints.
const (
	ProjectRoleMember int8 = 1
	ProjectRoleAdmin  int8 = 2
)

// An archived project is read only and hidden by default.
const (
	ProjectStateActive   int8 = 1
	ProjectStateArchived int8 = 2
)

type ProjectMember struct {
	UserID int32 `cb:"1"`
	Role   int8  `cb:"2"`
}

// Project is one agile project. Every user with the projects access sees it; only its Members
// write in it.
type Project struct {
	db.TableStruct[ProjectTable, Project]
	CompanyID int32 `cb:"1"`
	// ID is also the project's code: P01, P02...
	ID          int32  `cb:"2"`
	Name        string `cb:"3"`
	Description string `cb:"4"`
	// StartDate / EndDate are UnixDays, 0 = not set.
	StartDate int16           `cb:"5"`
	EndDate   int16           `cb:"6"`
	Color     string          `cb:"7"`
	Members   []ProjectMember `cb:"8"`
	State     int8            `cb:"9"`
	CreatedBy int32           `cb:"11"`
	// Created is a SUnixTime.
	Created   int32   `cb:"12"`
	UpdatedBy int32   `cb:"13"`
	Status    int8    `json:"ss" cb:"ss,14"`
	Updated   int32   `json:"upd" cb:"upd,15"`
	Version   int16   `json:"ver" cb:"ver,16"`
	Metadata  *string `json:"mdt" cb:"mdt,17"`
}

type ProjectTable struct {
	db.TableStruct[ProjectTable, Project]
	CompanyID   db.Col[*ProjectTable, int32]
	ID          db.Col[*ProjectTable, int32]
	Name        db.Col[*ProjectTable, string]
	Description db.Col[*ProjectTable, string]
	StartDate   db.Col[*ProjectTable, int16]
	EndDate     db.Col[*ProjectTable, int16]
	Color       db.Col[*ProjectTable, string]
	Members     db.Col[*ProjectTable, []ProjectMember]
	State       db.Col[*ProjectTable, int8]
	CreatedBy   db.Col[*ProjectTable, int32]
	Created     db.Col[*ProjectTable, int32]
	UpdatedBy   db.Col[*ProjectTable, int32]
	Status      db.Col[*ProjectTable, int8]
	Updated     db.Col[*ProjectTable, int32]
	Version     db.Col[*ProjectTable, int16]
}

// The project list is small and read whole: the company's partition, synced through the delta index.
func (e ProjectTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          57,
		Name:        "projects_project",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: statusValues}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}

// Epic is a tag that groups stories in the backlog. A story has one epic or none.
type Epic struct {
	db.TableStruct[EpicTable, Epic]
	CompanyID int32   `cb:"1"`
	ID        int64   `cb:"2"`
	ProjectID int32   `cb:"3"`
	Name      string  `cb:"4"`
	Color     string  `cb:"5"`
	UpdatedBy int32   `cb:"6"`
	Status    int8    `json:"ss" cb:"ss,7"`
	Updated   int32   `json:"upd" cb:"upd,8"`
	Version   int16   `json:"ver" cb:"ver,9"`
	Metadata  *string `json:"mdt" cb:"mdt,10"`
}

type EpicTable struct {
	db.TableStruct[EpicTable, Epic]
	CompanyID db.Col[*EpicTable, int32]
	ID        db.Col[*EpicTable, int64]
	ProjectID db.Col[*EpicTable, int32]
	Name      db.Col[*EpicTable, string]
	Color     db.Col[*EpicTable, string]
	UpdatedBy db.Col[*EpicTable, int32]
	Status    db.Col[*EpicTable, int8]
	Updated   db.Col[*EpicTable, int32]
	Version   db.Col[*EpicTable, int16]
}

// Every record of a project lives in the company's partition with its ProjectID as a column: the
// delta index pinned on ProjectID keeps one change log per project, so the browser syncs one
// project at a time (GET.projects-epics?project-id=).
func (e EpicTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          58,
		Name:        "projects_epic",
		Partition:   e.CompanyID,
		Keys:              db.Cols(e.ID),
		KeyIntPacking:     db.Cols(e.ProjectID.Size(63-projectSequenceBits), e.Autoincrement(0)),
		AutoincrementPart: e.ProjectID,
		FixedValues: []db.FixedValues{{Col: e.Status, Values: statusValues}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.ProjectID, e.Status)}},
	}
}
