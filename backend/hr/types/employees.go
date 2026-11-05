package types

import (
	"slices"
	"time"

	"app/db"
)

// Employee is the label-safe record of a worker: identity and where he sits in the company. It
// carries no salary, pension or bank data on purpose — those live in Contract, behind the payroll
// access — because hr-employees-ids may be granted to other modules that only need a name, and a
// cached route ships whole records (plan §3.0, rule 2).
type Employee struct {
	db.TableStruct[EmployeeTable, Employee]
	CompanyID int32 `cb:"1"`
	ID        int32 `cb:"2"`
	// UserID optionally links the worker to his login account (0 = no account).
	UserID          int32  `cb:"3"`
	IdentityDocType int8   `cb:"4"`
	DocumentNumber  string `cb:"5"`
	FirstName       string `cb:"6"`
	// LastName is the paternal surname, SecondLastName the maternal one (T-Registro asks both).
	LastName       string `cb:"7"`
	SecondLastName string `cb:"8"`
	BirthDate      int16  `cb:"9"` // UnixDay
	Sex            int8   `cb:"10"`
	CountryID      int32  `cb:"11"` // nationality, 604 = Peru
	Address        string `cb:"12"`
	CityID         int32  `cb:"13"` // district ubigeo, as in sites
	Email          string `cb:"14"`
	Phone          string `cb:"15"`
	DepartmentID   int32  `cb:"16"`
	PositionID     int32  `cb:"17"`
	HireDate       int16  `cb:"18"` // UnixDay
	// TerminationDate and TerminationReason are set together; with them the employee is terminated.
	TerminationDate   int16 `cb:"19"`
	TerminationReason int8  `cb:"20"`
	// Status: 1 active · 2 terminated · 0 deleted. Derived in SelfParse from TerminationDate.
	Status  int8  `json:"ss" cb:"ss,21"`
	Updated int32 `json:"upd" cb:"upd,22"`
	Version int16 `json:"ver" cb:"ver,23"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,24"`
	UpdatedBy int32   `cb:"25"`
	Created   int32   `cb:"26"`
	CreatedBy int32   `cb:"27"`
	// A summary of the dependents, rewritten on every dependent save (SetDependentsSummary): the
	// payroll and the employee list read it without loading the dependents. FamilyAllowance holds
	// the days a child gives the right to the asignación familiar.
	DependentsCount int8       `cb:"28"`
	FamilyAllowance []DayRange `cb:"29"`
	// ManagerEmployeeID is the immediate boss (0 = reports to nobody): the edge the org chart draws.
	ManagerEmployeeID int32 `cb:"30"`
}

// DayRange is a span of UnixDays, both included.
type DayRange struct {
	From int16 `cb:"1"`
	To   int16 `cb:"2"`
}

type EmployeeTable struct {
	db.TableStruct[EmployeeTable, Employee]
	CompanyID         db.Col[*EmployeeTable, int32]
	ID                db.Col[*EmployeeTable, int32]
	UserID            db.Col[*EmployeeTable, int32]
	IdentityDocType   db.Col[*EmployeeTable, int8]
	DocumentNumber    db.Col[*EmployeeTable, string]
	FirstName         db.Col[*EmployeeTable, string]
	LastName          db.Col[*EmployeeTable, string]
	SecondLastName    db.Col[*EmployeeTable, string]
	BirthDate         db.Col[*EmployeeTable, int16]
	Sex               db.Col[*EmployeeTable, int8]
	CountryID         db.Col[*EmployeeTable, int32]
	Address           db.Col[*EmployeeTable, string]
	CityID            db.Col[*EmployeeTable, int32]
	Email             db.Col[*EmployeeTable, string]
	Phone             db.Col[*EmployeeTable, string]
	DepartmentID      db.Col[*EmployeeTable, int32]
	PositionID        db.Col[*EmployeeTable, int32]
	HireDate          db.Col[*EmployeeTable, int16]
	TerminationDate   db.Col[*EmployeeTable, int16]
	TerminationReason db.Col[*EmployeeTable, int8]
	Status            db.Col[*EmployeeTable, int8]
	Updated           db.Col[*EmployeeTable, int32]
	Version           db.Col[*EmployeeTable, int16]
	UpdatedBy         db.Col[*EmployeeTable, int32]
	Created           db.Col[*EmployeeTable, int32]
	CreatedBy         db.Col[*EmployeeTable, int32]
	DependentsCount   db.Col[*EmployeeTable, int8]
	FamilyAllowance   db.Col[*EmployeeTable, []DayRange]
	ManagerEmployeeID db.Col[*EmployeeTable, int32]
}

// SetDependentsSummary rewrites the employee's dependents summary from all his dependents.
func (e *Employee) SetDependentsSummary(dependents []Dependent) {
	e.DependentsCount, e.FamilyAllowance = 0, FamilyAllowanceRanges(dependents)
	for _, dependent := range dependents {
		if dependent.Status == StatusActive {
			e.DependentsCount++
		}
	}
}

// HasFamilyAllowanceOn: a child gives the worker the asignación familiar on that day.
func (e Employee) HasFamilyAllowanceOn(unixDay int16) bool {
	for _, dayRange := range e.FamilyAllowance {
		if dayRange.From <= unixDay && unixDay <= dayRange.To {
			return true
		}
	}
	return false
}

// FamilyAllowanceRanges are the days a child gives the right to the asignación familiar (Ley
// 25129): while under 18, or under 24 while studying higher education, within the days the child is
// registered (ValidFrom–ValidTo). Overlapping and adjacent ranges are merged.
func FamilyAllowanceRanges(dependents []Dependent) []DayRange {
	ranges := []DayRange{}
	for _, dependent := range dependents {
		if dependent.Status != StatusActive || dependent.Relationship != RelationshipChild {
			continue
		}
		ageLimit := 18
		if dependent.IsStudying {
			ageLimit = 24
		}
		dayRange := DayRange{From: max(dependent.BirthDate, dependent.ValidFrom), To: birthdayDay(dependent.BirthDate, ageLimit) - 1}
		if dependent.ValidTo > 0 {
			dayRange.To = min(dayRange.To, dependent.ValidTo)
		}
		if dayRange.From <= dayRange.To {
			ranges = append(ranges, dayRange)
		}
	}
	slices.SortFunc(ranges, func(left, right DayRange) int { return int(left.From) - int(right.From) })
	merged := []DayRange{}
	for _, dayRange := range ranges {
		if last := len(merged) - 1; last >= 0 && dayRange.From <= merged[last].To+1 {
			merged[last].To = max(merged[last].To, dayRange.To)
			continue
		}
		merged = append(merged, dayRange)
	}
	return merged
}

// birthdayDay is the UnixDay someone born on birthDate turns years old (29 Feb → 1 Mar).
func birthdayDay(birthDate int16, years int) int16 {
	birth := time.Unix(int64(birthDate)*86400, 0).UTC()
	return int16(birth.AddDate(years, 0, 0).Unix() / 86400)
}

// SelfParse keeps Status in step with the termination: a worker with a TerminationDate is
// terminated. A deleted record (0) stays deleted.
func (e *Employee) SelfParse() {
	if e.Status == StatusDeleted {
		return
	}
	if e.TerminationDate > 0 {
		e.Status = EmployeeStatusTerminated
	} else {
		e.Status = StatusActive
	}
}

// Two caches: the employees page syncs active + terminated through the delta slice, and every
// fact that carries an EmployeeID (payslips, advances, time off, other modules) resolves the name
// through the by-IDs route.
func (e EmployeeTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          72,
		Name:        "hr_employees",
		Partition:   e.CompanyID,
		CacheByIDs:  true,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1, 2}}},
		Indexes: []db.Index{
			// The save handler refuses a document number already registered in the company.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.DocumentNumber)},
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// Dependent is a family member of a worker: the EsSalud "derechohabiente" T-Registro asks for, and
// the fact the asignación familiar is derived from (a child who is a minor, or studying up to 24).
type Dependent struct {
	db.TableStruct[DependentTable, Dependent]
	CompanyID       int32  `cb:"1"`
	ID              int32  `cb:"2"`
	EmployeeID      int32  `cb:"3"`
	Relationship    int8   `cb:"4"`
	IdentityDocType int8   `cb:"5"`
	DocumentNumber  string `cb:"6"`
	FirstName       string `cb:"7"`
	LastName        string `cb:"8"`
	SecondLastName  string `cb:"9"`
	BirthDate       int16  `cb:"10"` // UnixDay
	IsStudying      bool   `cb:"11"`
	// ValidFrom / ValidTo bound the period the dependent counts (UnixDays, ValidTo 0 = open).
	ValidFrom int16 `cb:"12"`
	ValidTo   int16 `cb:"13"`
	Status    int8  `json:"ss" cb:"ss,14"`
	Updated   int32 `json:"upd" cb:"upd,15"`
	Version   int16 `json:"ver" cb:"ver,16"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,17"`
	UpdatedBy int32   `cb:"18"`
	Created   int32   `cb:"19"`
	CreatedBy int32   `cb:"20"`
}

type DependentTable struct {
	db.TableStruct[DependentTable, Dependent]
	CompanyID       db.Col[*DependentTable, int32]
	ID              db.Col[*DependentTable, int32]
	EmployeeID      db.Col[*DependentTable, int32]
	Relationship    db.Col[*DependentTable, int8]
	IdentityDocType db.Col[*DependentTable, int8]
	DocumentNumber  db.Col[*DependentTable, string]
	FirstName       db.Col[*DependentTable, string]
	LastName        db.Col[*DependentTable, string]
	SecondLastName  db.Col[*DependentTable, string]
	BirthDate       db.Col[*DependentTable, int16]
	IsStudying      db.Col[*DependentTable, bool]
	ValidFrom       db.Col[*DependentTable, int16]
	ValidTo         db.Col[*DependentTable, int16]
	Status          db.Col[*DependentTable, int8]
	Updated         db.Col[*DependentTable, int32]
	Version         db.Col[*DependentTable, int16]
	UpdatedBy       db.Col[*DependentTable, int32]
	Created         db.Col[*DependentTable, int32]
	CreatedBy       db.Col[*DependentTable, int32]
}

// One delta slice per employee (EmployeeID pinned by the query, like stories by project): the
// dependents are downloaded only when that employee's panel is opened.
func (e DependentTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          83,
		Name:        "hr_employee_dependents",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.EmployeeID, e.Status)}},
	}
}
