package types

import "app/db"

// Department is an area of the company. Its CostFunction decides where the payroll cost of its
// people lands in the income statement by function.
type Department struct {
	db.TableStruct[DepartmentTable, Department]
	CompanyID int32  `cb:"1"`
	ID        int32  `cb:"2"`
	Name      string `cb:"3"`
	// ParentID is the department it reports to (0 = top level). Feeds the org chart.
	ParentID int32 `cb:"4"`
	// ManagerEmployeeID is the employee heading it (0 = not set).
	ManagerEmployeeID int32 `cb:"5"`
	CostFunction      int8  `cb:"6"`
	Status            int8  `json:"ss" cb:"ss,7"`
	Updated           int32 `json:"upd" cb:"upd,8"`
	Version           int16 `json:"ver" cb:"ver,9"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,10"`
	UpdatedBy int32   `cb:"11"`
	Created   int32   `cb:"12"`
	CreatedBy int32   `cb:"13"`
}

type DepartmentTable struct {
	db.TableStruct[DepartmentTable, Department]
	CompanyID         db.Col[*DepartmentTable, int32]
	ID                db.Col[*DepartmentTable, int32]
	Name              db.Col[*DepartmentTable, string]
	ParentID          db.Col[*DepartmentTable, int32]
	ManagerEmployeeID db.Col[*DepartmentTable, int32]
	CostFunction      db.Col[*DepartmentTable, int8]
	Status            db.Col[*DepartmentTable, int8]
	Updated           db.Col[*DepartmentTable, int32]
	Version           db.Col[*DepartmentTable, int16]
	UpdatedBy         db.Col[*DepartmentTable, int32]
	Created           db.Col[*DepartmentTable, int32]
	CreatedBy         db.Col[*DepartmentTable, int32]
}

// A small list read whole on page load: one delta slice of the active rows (route hr-org).
func (e DepartmentTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          70,
		Name:        "hr_departments",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}

// Position is a job title inside a department.
type Position struct {
	db.TableStruct[PositionTable, Position]
	CompanyID    int32  `cb:"1"`
	ID           int32  `cb:"2"`
	Name         string `cb:"3"`
	DepartmentID int32  `cb:"4"`
	// OccupationCode is SUNAT's occupation code for T-Registro (optional until the export exists).
	OccupationCode string `cb:"5"`
	Status         int8   `json:"ss" cb:"ss,6"`
	Updated        int32  `json:"upd" cb:"upd,7"`
	Version        int16  `json:"ver" cb:"ver,8"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,9"`
	UpdatedBy int32   `cb:"10"`
	Created   int32   `cb:"11"`
	CreatedBy int32   `cb:"12"`
}

type PositionTable struct {
	db.TableStruct[PositionTable, Position]
	CompanyID      db.Col[*PositionTable, int32]
	ID             db.Col[*PositionTable, int32]
	Name           db.Col[*PositionTable, string]
	DepartmentID   db.Col[*PositionTable, int32]
	OccupationCode db.Col[*PositionTable, string]
	Status         db.Col[*PositionTable, int8]
	Updated        db.Col[*PositionTable, int32]
	Version        db.Col[*PositionTable, int16]
	UpdatedBy      db.Col[*PositionTable, int32]
	Created        db.Col[*PositionTable, int32]
	CreatedBy      db.Col[*PositionTable, int32]
}

// Read whole with the departments, in the same hr-org response.
func (e PositionTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          71,
		Name:        "hr_positions",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}
