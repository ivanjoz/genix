package hr

import (
	"encoding/json"
	"fmt"
	"strings"

	"app/core"
	"app/db"
	"app/hr/types"

	"golang.org/x/sync/errgroup"
)

// GetOrganization syncs departments and positions in one response, one delta cursor per key.
func GetOrganization(req *core.HandlerArgs) core.HandlerResponse {
	departments, positions := []types.Department{}, []types.Position{}
	errGroup := errgroup.Group{}
	errGroup.Go(func() error {
		query := db.Query(&departments)
		query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("Departments"), types.StatusActive)
		return query.Exec()
	})
	errGroup.Go(func() error {
		query := db.Query(&positions)
		query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("Positions"), types.StatusActive)
		return query.Exec()
	})
	if err := errGroup.Wait(); err != nil {
		return req.MakeErr("Error al obtener las áreas y cargos:", err)
	}
	return core.MakeResponse(req, &map[string]any{"Departments": departments, "Positions": positions})
}

// PostDepartment creates or edits a department; Status 0 deletes it, refused while it still has
// active employees or positions.
func PostDepartment(req *core.HandlerArgs) core.HandlerResponse {
	department := types.Department{}
	if err := json.Unmarshal([]byte(*req.Body), &department); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	department.Name = strings.TrimSpace(department.Name)

	if department.Status == types.StatusDeleted && department.ID > 0 {
		if err := ensureDepartmentUnused(companyID, department.ID); err != nil {
			return req.MakeErr(err)
		}
	} else {
		if len(department.Name) < 2 {
			return req.MakeErr("El nombre del área debe tener al menos 2 caracteres.")
		}
		if !types.HasOption(types.CostFunctionOptions, department.CostFunction) {
			return req.MakeErr("Seleccione la función de costo del área (administración, ventas o producción).")
		}
		if department.ParentID > 0 {
			if department.ParentID == department.ID {
				return req.MakeErr("Un área no puede depender de sí misma.")
			}
			if _, err := loadActiveDepartment(companyID, department.ParentID); err != nil {
				return req.MakeErr("El área superior:", err)
			}
		}
		if department.ManagerEmployeeID > 0 {
			if _, err := loadEmployee(companyID, department.ManagerEmployeeID); err != nil {
				return req.MakeErr("El responsable del área:", err)
			}
		}
		department.Status = types.StatusActive
	}

	department.CompanyID, department.UpdatedBy = companyID, req.User.ID
	records := []types.Department{department}
	if department.ID == 0 {
		records[0].Created, records[0].CreatedBy = core.SUnixTime(), req.User.ID
		if err := db.Upsert(&records); err != nil {
			return req.MakeErr("Error al guardar el área:", err)
		}
		return req.MakeResponse(records[0])
	}
	if _, err := loadDepartment(companyID, department.ID); err != nil {
		return req.MakeErr(err)
	}
	table := db.TableOf[types.Department]()
	if err := db.Update(&records, table.Name, table.ParentID, table.ManagerEmployeeID, table.CostFunction,
		table.Status, table.UpdatedBy); err != nil {
		return req.MakeErr("Error al guardar el área:", err)
	}
	return req.MakeResponse(records[0])
}

// PostPosition creates or edits a position; Status 0 deletes it, refused while employees hold it.
func PostPosition(req *core.HandlerArgs) core.HandlerResponse {
	position := types.Position{}
	if err := json.Unmarshal([]byte(*req.Body), &position); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	position.Name = strings.TrimSpace(position.Name)

	if position.Status == types.StatusDeleted && position.ID > 0 {
		employees, err := loadCompanyEmployees(companyID)
		if err != nil {
			return req.MakeErr(err)
		}
		for _, employee := range employees {
			if employee.PositionID == position.ID && employee.Status == types.StatusActive {
				return req.MakeErr(fmt.Sprintf("El cargo está asignado a %s %s; reasígnelo antes de eliminarlo.",
					employee.FirstName, employee.LastName))
			}
		}
	} else {
		if len(position.Name) < 2 {
			return req.MakeErr("El nombre del cargo debe tener al menos 2 caracteres.")
		}
		if _, err := loadActiveDepartment(companyID, position.DepartmentID); err != nil {
			return req.MakeErr("El área del cargo:", err)
		}
		position.Status = types.StatusActive
	}

	position.CompanyID, position.UpdatedBy = companyID, req.User.ID
	records := []types.Position{position}
	if position.ID == 0 {
		records[0].Created, records[0].CreatedBy = core.SUnixTime(), req.User.ID
		if err := db.Upsert(&records); err != nil {
			return req.MakeErr("Error al guardar el cargo:", err)
		}
		return req.MakeResponse(records[0])
	}
	if _, err := loadPosition(companyID, position.ID); err != nil {
		return req.MakeErr(err)
	}
	table := db.TableOf[types.Position]()
	if err := db.Update(&records, table.Name, table.DepartmentID, table.OccupationCode, table.Status, table.UpdatedBy); err != nil {
		return req.MakeErr("Error al guardar el cargo:", err)
	}
	return req.MakeResponse(records[0])
}

// ensureDepartmentUnused refuses to delete a department that active employees, positions or child
// departments still point to: their labels would stop resolving.
func ensureDepartmentUnused(companyID, departmentID int32) error {
	employees, err := loadCompanyEmployees(companyID)
	if err != nil {
		return err
	}
	for _, employee := range employees {
		if employee.DepartmentID == departmentID && employee.Status == types.StatusActive {
			return fmt.Errorf("el área tiene personal activo (%s %s); reasígnelo antes de eliminarla", employee.FirstName, employee.LastName)
		}
	}
	positions := []types.Position{}
	query := db.Query(&positions)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return fmt.Errorf("error al leer los cargos: %v", err)
	}
	for _, position := range positions {
		if position.DepartmentID == departmentID && position.Status == types.StatusActive {
			return fmt.Errorf("el área tiene el cargo \"%s\"; elimínelo o muévalo antes", position.Name)
		}
	}
	departments := []types.Department{}
	departmentQuery := db.Query(&departments)
	departmentQuery.CompanyID.Equals(companyID)
	if err := departmentQuery.Exec(); err != nil {
		return fmt.Errorf("error al leer las áreas: %v", err)
	}
	for _, department := range departments {
		if department.ParentID == departmentID && department.Status == types.StatusActive {
			return fmt.Errorf("el área \"%s\" depende de ella; muévala antes", department.Name)
		}
	}
	return nil
}

func loadDepartment(companyID, departmentID int32) (types.Department, error) {
	departments := []types.Department{}
	query := db.Query(&departments)
	query.CompanyID.Equals(companyID).ID.Equals(departmentID)
	if err := query.Exec(); err != nil {
		return types.Department{}, fmt.Errorf("error al leer el área: %v", err)
	}
	if len(departments) == 0 {
		return types.Department{}, fmt.Errorf("el área %d no existe", departmentID)
	}
	return departments[0], nil
}

func loadActiveDepartment(companyID, departmentID int32) (types.Department, error) {
	if departmentID <= 0 {
		return types.Department{}, fmt.Errorf("debe seleccionar un área")
	}
	department, err := loadDepartment(companyID, departmentID)
	if err == nil && department.Status != types.StatusActive {
		err = fmt.Errorf("el área \"%s\" fue eliminada", department.Name)
	}
	return department, err
}

func loadPosition(companyID, positionID int32) (types.Position, error) {
	positions := []types.Position{}
	query := db.Query(&positions)
	query.CompanyID.Equals(companyID).ID.Equals(positionID)
	if err := query.Exec(); err != nil {
		return types.Position{}, fmt.Errorf("error al leer el cargo: %v", err)
	}
	if len(positions) == 0 {
		return types.Position{}, fmt.Errorf("el cargo %d no existe", positionID)
	}
	return positions[0], nil
}
