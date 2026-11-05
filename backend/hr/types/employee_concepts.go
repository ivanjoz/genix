package types

import "app/db"

// EmployeeConcept is a company concept paid or deducted every month of a range (a fixed bonus,
// movilidad, a judicial withholding): what applies to an employee without belonging to a date.
// What happened on a date is an EmployeeRecord. ToPeriod 0 = no end.
type EmployeeConcept struct {
	db.TableStruct[EmployeeConceptTable, EmployeeConcept]
	CompanyID  int32  `cb:"1"`
	ID         int32  `cb:"2"`
	EmployeeID int32  `cb:"3"`
	ConceptID  int32  `cb:"4"`
	Amount     int32  `cb:"5"` // cents per month
	FromPeriod int16  `cb:"6"`
	ToPeriod   int16  `cb:"7"`
	Note       string `cb:"8"`
	Status     int8   `json:"ss" cb:"ss,9"`
	Updated    int32  `json:"upd" cb:"upd,10"`
	Version    int16  `json:"ver" cb:"ver,11"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,12"`
	UpdatedBy int32   `cb:"13"`
}

type EmployeeConceptTable struct {
	db.TableStruct[EmployeeConceptTable, EmployeeConcept]
	CompanyID  db.Col[*EmployeeConceptTable, int32]
	ID         db.Col[*EmployeeConceptTable, int32]
	EmployeeID db.Col[*EmployeeConceptTable, int32]
	ConceptID  db.Col[*EmployeeConceptTable, int32]
	Amount     db.Col[*EmployeeConceptTable, int32]
	FromPeriod db.Col[*EmployeeConceptTable, int16]
	ToPeriod   db.Col[*EmployeeConceptTable, int16]
	Note       db.Col[*EmployeeConceptTable, string]
	Status     db.Col[*EmployeeConceptTable, int8]
	Updated    db.Col[*EmployeeConceptTable, int32]
	Version    db.Col[*EmployeeConceptTable, int16]
	UpdatedBy  db.Col[*EmployeeConceptTable, int32]
}

func (e EmployeeConceptTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          75,
		Name:        "hr_employee_concepts",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.EmployeeID, e.Status)}},
	}
}

// AppliesTo reports whether the recurring concept is paid in a period.
func (e EmployeeConcept) AppliesTo(period int16) bool {
	return e.Status == StatusActive && e.FromPeriod <= period && (e.ToPeriod == 0 || period <= e.ToPeriod)
}
