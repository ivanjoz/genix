package production

import (
	"app/core"
	"app/db"
	hr "app/hr/types"
	"app/production/types"
	"encoding/json"
	"fmt"
)

// Labor entries (docs/MANUFACTURING_PLAN.md §9.2): the hours each worker spends on an order, valued at
// their real cost per hour. Every write refreshes the order's recorded labor, which its next receipt
// absorbs.

// GetProductionLaborEntries is the delta of one order's labor entries ("Mano de obra" tab).
func GetProductionLaborEntries(req *core.HandlerArgs) core.HandlerResponse {
	orderID := int32(req.GetQueryInt("order-id"))
	if orderID <= 0 {
		return req.MakeErr("Se requiere la orden de producción.")
	}
	entries := []types.ProductionLaborEntry{}
	query := db.Query(&entries)
	query.Select().CompanyID.Equals(req.User.CompanyID).OrderID.Equals(orderID).Delta(req.GetDeltaCursor("records"))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las horas de la orden:", err)
	}
	return req.MakeResponse(map[string]any{"records": &entries})
}

// GetProductionLaborEntriesByPeriod is one month of every order's labor entries, for the "Horas vs
// planilla" report and the monthly adjustment (cache-groups by Period).
func GetProductionLaborEntriesByPeriod(req *core.HandlerArgs) core.HandlerResponse {
	period := int16(req.GetQueryInt("period"))
	if period <= 0 {
		return req.MakeErr("Se requiere el periodo.")
	}
	entries := []types.ProductionLaborEntry{}
	query := db.Query(&entries)
	query.Select().CompanyID.Equals(req.User.CompanyID).Period.Equals(period)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las horas del periodo:", err)
	}
	return req.MakeResponse(entries)
}

// PostProductionLaborEntry logs, edits or (ss = 0) deletes one entry of a released or in-progress
// order. The worker's real cost per hour on the entry's day values it (§9.2, §9.5): a Planilla worker's
// month never books more than their estimated payroll cost; a worker off payroll is uncapped, at their
// agreement's rate or, without one, the labor type's.
func PostProductionLaborEntry(req *core.HandlerArgs) core.HandlerResponse {
	entry := types.ProductionLaborEntry{}
	if err := json.Unmarshal([]byte(*req.Body), &entry); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	isDelete := entry.Status == 0
	if !isDelete {
		if entry.Date <= 0 || entry.Date > core.FechaUnix() {
			return req.MakeErr("La fecha de las horas no es válida.")
		}
		if entry.Minutes <= 0 || entry.Minutes > 24*60 {
			return req.MakeErr("Las horas de un día deben estar entre 1 minuto y 24 horas.")
		}
	}
	var savedEntry types.ProductionLaborEntry

	err := db.RunTransaction(func(transaction *db.Transaction) error {
		// Entries are locked out once the order finishes (§1): every hour is absorbed by a receipt.
		order, err := lockProductionOrderIn(transaction, companyID, entry.OrderID,
			types.ProductionOrderStatusReleased, types.ProductionOrderStatusInProgress)
		if err != nil {
			return err
		}
		orderEntries := []types.ProductionLaborEntry{}
		orderEntriesQuery := db.Query(&orderEntries)
		orderEntriesQuery.Select().CompanyID.Equals(companyID).OrderID.Equals(order.ID)
		if err := orderEntriesQuery.Exec(); err != nil {
			return core.Err("Error al obtener las horas de la orden:", err)
		}
		var stored *types.ProductionLaborEntry
		for index := range orderEntries {
			if orderEntries[index].ID == entry.ID && entry.ID > 0 {
				stored = &orderEntries[index]
			}
		}
		if entry.ID > 0 && stored == nil {
			return core.Err("El registro de horas no existe en esta orden.")
		}
		if entry.ID <= 0 && isDelete {
			return core.Err("No se puede eliminar un registro de horas que no se ha guardado.")
		}

		nowTime := core.SUnixTime()
		if isDelete {
			entry = *stored
			entry.Status = 0
		} else {
			if err := valueLaborEntry(companyID, &entry); err != nil {
				return err
			}
			entry.Status = 1
		}
		entry.CompanyID, entry.Updated, entry.UpdatedBy = companyID, nowTime, req.User.ID
		records := []types.ProductionLaborEntry{entry}
		table := db.TableOf[types.ProductionLaborEntry]()
		if stored == nil {
			records[0].ID, records[0].Created, records[0].CreatedBy = 0, nowTime, req.User.ID
			err = db.InsertIn(transaction, &records)
		} else {
			err = db.UpdateIn(transaction, &records, table.EmployeeID, table.LaborTypeID, table.Date, table.Period, table.Minutes,
				table.CostPerHour, table.Amount, table.Status, table.OrderID, table.Updated, table.UpdatedBy)
		}
		if err != nil {
			return core.Err("Error al guardar las horas:", err)
		}
		savedEntry = records[0]

		// The order's recorded labor is rebuilt from its entries, this write included.
		replaced := false
		for index := range orderEntries {
			if orderEntries[index].ID == savedEntry.ID {
				orderEntries[index], replaced = savedEntry, true
			}
		}
		if !replaced {
			orderEntries = append(orderEntries, savedEntry)
		}
		order.Labor, order.LaborRecordedAmount = types.ProductionOrderLaborRecorded(order.Labor, orderEntries)
		order.Status = types.ProductionOrderStatusInProgress
		order.Updated, order.UpdatedBy = nowTime, req.User.ID
		orders := []types.ProductionOrder{*order}
		orderTable := db.TableOf[types.ProductionOrder]()
		if err := db.UpdateIn(transaction, &orders, orderTable.Labor, orderTable.LaborRecordedAmount, orderTable.Status,
			orderTable.Updated, orderTable.UpdatedBy); err != nil {
			return core.Err("Error al actualizar la mano de obra de la orden:", err)
		}
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(savedEntry)
}

// valueLaborEntry checks the entry's labor type and stamps its Period, CostPerHour and capped Amount.
func valueLaborEntry(companyID int32, entry *types.ProductionLaborEntry) error {
	laborTypes := []types.ProductionLaborType{}
	laborTypeQuery := db.Query(&laborTypes)
	laborTypeQuery.Select(laborTypeQuery.ID, laborTypeQuery.Name, laborTypeQuery.CostPerHour, laborTypeQuery.Status).
		CompanyID.Equals(companyID).ID.Equals(entry.LaborTypeID)
	if err := laborTypeQuery.Exec(); err != nil {
		return core.Err("Error al obtener el tipo de trabajo:", err)
	}
	if len(laborTypes) == 0 || laborTypes[0].Status != 1 {
		return core.Err(fmt.Sprintf("El tipo de trabajo %v no existe o no está activo.", entry.LaborTypeID))
	}
	hourCost, err := hr.LoadWorkerHourCost(companyID, entry.EmployeeID, entry.Date)
	if err != nil {
		return err
	}
	entry.Period = core.PeriodOfDay(entry.Date)
	entry.CostPerHour = hourCost.CostPerHour
	if !hourCost.HasAgreement {
		entry.CostPerHour = laborTypes[0].CostPerHour
	}

	// The month's other entries of the worker, in any order, count against the cap.
	otherEntriesAmount := int64(0)
	if hourCost.MonthlyCap >= 0 {
		monthEntries := []types.ProductionLaborEntry{}
		monthQuery := db.Query(&monthEntries)
		monthQuery.Select(monthQuery.ID, monthQuery.Amount, monthQuery.Status).
			CompanyID.Equals(companyID).EmployeeID.Equals(entry.EmployeeID).Period.Equals(entry.Period)
		if err := monthQuery.Exec(); err != nil {
			return core.Err("Error al obtener las horas del trabajador en el mes:", err)
		}
		for _, monthEntry := range monthEntries {
			if monthEntry.Status == 1 && monthEntry.ID != entry.ID {
				otherEntriesAmount += int64(monthEntry.Amount)
			}
		}
	}
	entry.Amount = types.CappedLaborAmount(entry.Minutes, entry.CostPerHour, hourCost.MonthlyCap, otherEntriesAmount)
	return nil
}

// GetProductionWorkers is the delta of the company's workers as production needs them: who can log
// hours on an order and who the "Horas vs planilla" report lists (§9.2, §9.5). Only the name, the
// engagement, the department and the employment dates: no pay, document or contact data.
func GetProductionWorkers(req *core.HandlerArgs) core.HandlerResponse {
	employees := []hr.Employee{}
	query := db.Query(&employees)
	query.Select(query.ID, query.FirstName, query.LastName, query.SecondLastName, query.EngagementType, query.DepartmentID,
		query.HireDate, query.TerminationDate, query.Status, query.Updated, query.Version).
		CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), hr.StatusActive, hr.EmployeeStatusTerminated)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los trabajadores:", err)
	}
	return req.MakeResponse(employees)
}
