package types

import "app/db"

// ProductionLaborEntry is time a worker spent on a production order, of one labor type
// (docs/MANUFACTURING_PLAN.md §9.2, M9, M18). The same worker may log different types on different
// days. Amount is the cost the order's next receipt absorbs: Minutes at the worker's real cost per
// hour, capped for a Planilla worker so their month never books more than their estimated payroll cost.
type ProductionLaborEntry struct {
	db.TableStruct[ProductionLaborEntryTable, ProductionLaborEntry]
	CompanyID int32 `cb:"1"`
	ID        int32 `cb:"2"`
	OrderID   int32 `cb:"3"`
	// EmployeeID is an hr_employees row, on payroll or not (§9.5).
	EmployeeID  int32 `cb:"4"`
	LaborTypeID int32 `cb:"5"`
	Date        int16 `cb:"6"`
	// Period is the YYMM of Date: the monthly cap and the "Horas vs planilla" report read by it.
	Period  int16 `cb:"7"`
	Minutes int32 `cb:"8"`
	// CostPerHour is the worker's real rate when logged, PEN cents.
	CostPerHour int32 `cb:"9"`
	// Amount is the valued cost after the monthly cap, PEN cents.
	Amount    int32 `cb:"10"`
	Status    int8  `json:"ss" cb:"ss,11"`
	Created   int32 `cb:"12"`
	CreatedBy int32 `cb:"13"`
	Updated   int32 `json:"upd" cb:"upd,14"`
	UpdatedBy int32 `cb:"15"`
	Version   int16 `json:"ver" cb:"ver,16"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,17"`
}

type ProductionLaborEntryTable struct {
	db.TableStruct[ProductionLaborEntryTable, ProductionLaborEntry]
	CompanyID   db.Col[*ProductionLaborEntryTable, int32]
	ID          db.Col[*ProductionLaborEntryTable, int32]
	OrderID     db.Col[*ProductionLaborEntryTable, int32]
	EmployeeID  db.Col[*ProductionLaborEntryTable, int32]
	LaborTypeID db.Col[*ProductionLaborEntryTable, int32]
	Date        db.Col[*ProductionLaborEntryTable, int16]
	Period      db.Col[*ProductionLaborEntryTable, int16]
	Minutes     db.Col[*ProductionLaborEntryTable, int32]
	CostPerHour db.Col[*ProductionLaborEntryTable, int32]
	Amount      db.Col[*ProductionLaborEntryTable, int32]
	Status      db.Col[*ProductionLaborEntryTable, int8]
	Created     db.Col[*ProductionLaborEntryTable, int32]
	CreatedBy   db.Col[*ProductionLaborEntryTable, int32]
	Updated     db.Col[*ProductionLaborEntryTable, int32]
	UpdatedBy   db.Col[*ProductionLaborEntryTable, int32]
	Version     db.Col[*ProductionLaborEntryTable, int16]
}

func (e ProductionLaborEntryTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          103,
		Name:        "production_labor_entry",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 1}},
		Indexes: []db.Index{
			// The order's "Mano de obra" tab, with OrderID pinned by an equality.
			{Type: db.TypeDelta, Keys: db.Cols(e.OrderID)},
			// The monthly cap, the "Horas vs planilla" report and the adjustment: one group per month.
			{Type: db.TypeCacheGroup, Keys: db.Cols(e.Period)},
			// The cap reads the employee's other entries of the month.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.EmployeeID, e.Period.Size(16))},
		},
	}
}

// CappedLaborAmount is what an entry books (§9.2):
//
//	min(Minutes × CostPerHour ÷ 60, monthly cap − what the employee's other entries of the month booked)
//
// never below 0. A worker off payroll has no cap (monthlyCap < 0).
func CappedLaborAmount(minutes int32, costPerHour int32, monthlyCap int64, otherEntriesAmount int64) int32 {
	amount := (int64(minutes)*int64(costPerHour) + 30) / 60
	if monthlyCap >= 0 {
		amount = min(amount, max(monthlyCap-otherEntriesAmount, 0))
	}
	return int32(amount)
}

// ProductionOrderLaborRecorded rebuilds an order's recorded labor from its active entries: the
// minutes and cost of each planned type (a type logged but not planned is added), and the total the
// receipts absorb.
func ProductionOrderLaborRecorded(planned []ProductionOrderLabor, entries []ProductionLaborEntry) ([]ProductionOrderLabor, int64) {
	labor := make([]ProductionOrderLabor, len(planned))
	indexByTypeID := map[int32]int{}
	for index, orderLabor := range planned {
		orderLabor.RecordedMinutes, orderLabor.RecordedCost = 0, 0
		labor[index] = orderLabor
		indexByTypeID[orderLabor.LaborTypeID] = index
	}
	recordedAmount := int64(0)
	for _, entry := range entries {
		if entry.Status != 1 {
			continue
		}
		index, isKnown := indexByTypeID[entry.LaborTypeID]
		if !isKnown {
			labor = append(labor, ProductionOrderLabor{LaborTypeID: entry.LaborTypeID})
			index = len(labor) - 1
			indexByTypeID[entry.LaborTypeID] = index
		}
		labor[index].RecordedMinutes += entry.Minutes
		labor[index].RecordedCost += entry.Amount
		recordedAmount += int64(entry.Amount)
	}
	return labor, recordedAmount
}
