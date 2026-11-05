package hr

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"app/core"
	coreTypes "app/core/types"
	crm "app/crm/types"
	"app/db"
	"app/hr/types"
)

var dniPattern = regexp.MustCompile(`^\d{8}$`)

// GetEmployees syncs active and terminated employees: a terminated worker stays listed because his
// payslips, contracts and profit sharing still refer to him. A deleted one arrives as a removed ID.
func GetEmployees(req *core.HandlerArgs) core.HandlerResponse {
	employees := []types.Employee{}
	query := db.Query(&employees)
	query.Select().CompanyID.Equals(req.User.CompanyID).
		Delta(req.GetDeltaCursor(), types.StatusActive, types.EmployeeStatusTerminated)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el personal:", err)
	}
	return core.MakeResponse(req, &employees)
}

// GetEmployeesByIDs resolves employee labels for any fact that carries an EmployeeID.
func GetEmployeesByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	partitions, err := db.QueryCachedIDs[types.Employee](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener el personal.", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}

// PostEmployee creates or edits an employee. Status 0 deletes him; terminating is done by setting
// TerminationDate and TerminationReason, which keeps the record.
func PostEmployee(req *core.HandlerArgs) core.HandlerResponse {
	employee := types.Employee{}
	if err := json.Unmarshal([]byte(*req.Body), &employee); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	isDelete := employee.Status == types.StatusDeleted && employee.ID > 0
	if !isDelete {
		if err := validateEmployee(companyID, &employee); err != nil {
			return req.MakeErr(err)
		}
		employee.Status = types.StatusActive // SelfParse turns it into terminated when due
	}

	employee.CompanyID, employee.UpdatedBy = companyID, req.User.ID
	// The dependents summary is the server's: only a dependent save writes it.
	employee.DependentsCount, employee.FamilyAllowance = 0, nil
	records := []types.Employee{employee}
	if employee.ID == 0 {
		records[0].Created, records[0].CreatedBy = core.SUnixTime(), req.User.ID
		if err := db.Insert(&records); err != nil {
			return req.MakeErr("Error al guardar el trabajador:", err)
		}
		return req.MakeResponse(records[0])
	}
	if _, err := loadEmployee(companyID, employee.ID); err != nil {
		return req.MakeErr(err)
	}
	table := db.TableOf[types.Employee]()
	err := db.Update(&records, table.UserID, table.IdentityDocType, table.DocumentNumber, table.FirstName,
		table.LastName, table.SecondLastName, table.BirthDate, table.Sex, table.CountryID, table.Address,
		table.CityID, table.Email, table.Phone, table.DepartmentID, table.PositionID, table.ManagerEmployeeID, table.HireDate,
		table.TerminationDate, table.TerminationReason, table.Status, table.UpdatedBy)
	if err != nil {
		return req.MakeErr("Error al guardar el trabajador:", err)
	}
	return req.MakeResponse(records[0])
}

// validateEmployee checks every field the payroll, T-Registro and the labels rely on, and
// normalizes the text fields in place.
func validateEmployee(companyID int32, employee *types.Employee) error {
	employee.DocumentNumber = strings.TrimSpace(employee.DocumentNumber)
	employee.FirstName = strings.TrimSpace(employee.FirstName)
	employee.LastName = strings.TrimSpace(employee.LastName)
	employee.SecondLastName = strings.TrimSpace(employee.SecondLastName)
	employee.Email = strings.TrimSpace(employee.Email)

	if !crm.IsIdentityDocType(employee.IdentityDocType) {
		return fmt.Errorf("seleccione el tipo de documento de identidad")
	}
	if employee.IdentityDocType == crm.IdentityDocDNI && !dniPattern.MatchString(employee.DocumentNumber) {
		return fmt.Errorf("el DNI debe tener 8 dígitos")
	}
	if len(employee.DocumentNumber) < 6 {
		return fmt.Errorf("el número de documento es inválido")
	}
	if employee.FirstName == "" || employee.LastName == "" {
		return fmt.Errorf("ingrese los nombres y el apellido paterno")
	}
	if !types.HasOption(types.SexOptions, employee.Sex) {
		return fmt.Errorf("seleccione el sexo (lo exige T-Registro)")
	}
	if employee.BirthDate <= 0 {
		return fmt.Errorf("ingrese la fecha de nacimiento")
	}
	if employee.HireDate <= 0 {
		return fmt.Errorf("ingrese la fecha de ingreso")
	}
	if employee.HireDate <= employee.BirthDate {
		return fmt.Errorf("la fecha de ingreso debe ser posterior a la de nacimiento")
	}
	if employee.CountryID == 0 {
		employee.CountryID = 604 // Peru
	}

	// Termination: both fields or neither.
	if employee.TerminationDate > 0 || employee.TerminationReason > 0 {
		if !types.HasOption(types.TerminationReasonOptions, employee.TerminationReason) {
			return fmt.Errorf("seleccione el motivo del cese")
		}
		if employee.TerminationDate < employee.HireDate {
			return fmt.Errorf("la fecha de cese no puede ser anterior a la de ingreso")
		}
	}

	if _, err := loadActiveDepartment(companyID, employee.DepartmentID); err != nil {
		return err
	}
	if employee.PositionID > 0 {
		position, err := loadPosition(companyID, employee.PositionID)
		if err != nil {
			return err
		}
		if position.Status != types.StatusActive {
			return fmt.Errorf("el cargo \"%s\" fue eliminado", position.Name)
		}
		if position.DepartmentID != employee.DepartmentID {
			return fmt.Errorf("el cargo \"%s\" no pertenece al área seleccionada", position.Name)
		}
	}
	if err := ensureManagerValid(companyID, employee.ID, employee.ManagerEmployeeID); err != nil {
		return err
	}
	if employee.UserID > 0 {
		if err := ensureCompanyUser(companyID, employee.UserID); err != nil {
			return err
		}
	}
	return ensureDocumentNumberFree(companyID, employee.ID, employee.DocumentNumber)
}

// ensureManagerValid refuses a boss who is the employee himself, is gone, or reports (directly or
// through others) to the employee: that would close a loop in the org chart.
func ensureManagerValid(companyID, employeeID, managerID int32) error {
	if managerID == 0 {
		return nil
	}
	if managerID == employeeID {
		return fmt.Errorf("un trabajador no puede ser su propio jefe inmediato")
	}
	employees, err := loadCompanyEmployees(companyID)
	if err != nil {
		return err
	}
	managerByEmployee := map[int32]int32{}
	for _, employee := range employees {
		if employee.Status == types.StatusDeleted {
			continue
		}
		managerByEmployee[employee.ID] = employee.ManagerEmployeeID
	}
	if _, exists := managerByEmployee[managerID]; !exists {
		return fmt.Errorf("el jefe inmediato %d no existe", managerID)
	}
	if employeeID > 0 {
		for ancestorID, visited := managerID, 0; ancestorID != 0 && visited <= len(managerByEmployee); ancestorID, visited = managerByEmployee[ancestorID], visited+1 {
			if ancestorID == employeeID {
				return fmt.Errorf("el jefe inmediato elegido ya depende de este trabajador: se formaría un ciclo")
			}
		}
	}
	for _, employee := range employees {
		if employee.ID == managerID && employee.Status != types.StatusActive {
			return fmt.Errorf("el jefe inmediato %s %s está cesado", employee.FirstName, employee.LastName)
		}
	}
	return nil
}

// ensureDocumentNumberFree refuses a document number another live employee of the company holds.
func ensureDocumentNumberFree(companyID, employeeID int32, documentNumber string) error {
	employees := []types.Employee{}
	query := db.Query(&employees)
	query.CompanyID.Equals(companyID).DocumentNumber.Equals(documentNumber)
	if err := query.Exec(); err != nil {
		return fmt.Errorf("error al verificar el documento: %v", err)
	}
	for _, employee := range employees {
		if employee.ID != employeeID && employee.Status != types.StatusDeleted {
			return fmt.Errorf("el documento %s ya está registrado para %s %s", documentNumber, employee.FirstName, employee.LastName)
		}
	}
	return nil
}

func ensureCompanyUser(companyID, userID int32) error {
	users := []coreTypes.User{}
	query := db.Query(&users)
	query.CompanyID.Equals(companyID).ID.Equals(userID)
	if err := query.Exec(); err != nil {
		return fmt.Errorf("error al leer el usuario: %v", err)
	}
	if len(users) == 0 || users[0].Status == types.StatusDeleted {
		return fmt.Errorf("el usuario %d no existe en la empresa", userID)
	}
	return nil
}

func loadEmployee(companyID, employeeID int32) (types.Employee, error) {
	employees := []types.Employee{}
	query := db.Query(&employees)
	query.CompanyID.Equals(companyID).ID.Equals(employeeID)
	if err := query.Exec(); err != nil {
		return types.Employee{}, fmt.Errorf("error al leer el trabajador: %v", err)
	}
	if len(employees) == 0 || employees[0].Status == types.StatusDeleted {
		return types.Employee{}, fmt.Errorf("el trabajador %d no existe", employeeID)
	}
	return employees[0], nil
}

// loadCompanyEmployees reads every employee of the company: the deletion guards need the whole set,
// and an SME's staff fits in one partition read.
func loadCompanyEmployees(companyID int32) ([]types.Employee, error) {
	employees := []types.Employee{}
	query := db.Query(&employees)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer el personal: %v", err)
	}
	return employees, nil
}

// GetDependents syncs one employee's dependents (?employee-id=), a delta slice pinned by EmployeeID.
func GetDependents(req *core.HandlerArgs) core.HandlerResponse {
	employeeID := req.GetQueryInt("employee-id")
	if employeeID <= 0 {
		return req.MakeErr("Debe enviar employee-id.")
	}
	dependents := []types.Dependent{}
	query := db.Query(&dependents)
	query.Select().CompanyID.Equals(req.User.CompanyID).EmployeeID.Equals(employeeID).
		Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los derechohabientes:", err)
	}
	return core.MakeResponse(req, &dependents)
}

// PostDependent creates or edits a dependent; Status 0 deletes it.
func PostDependent(req *core.HandlerArgs) core.HandlerResponse {
	dependent := types.Dependent{}
	if err := json.Unmarshal([]byte(*req.Body), &dependent); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	employee, err := loadEmployee(companyID, dependent.EmployeeID)
	if err != nil {
		return req.MakeErr(err)
	}
	if !(dependent.Status == types.StatusDeleted && dependent.ID > 0) {
		dependent.FirstName = strings.TrimSpace(dependent.FirstName)
		dependent.LastName = strings.TrimSpace(dependent.LastName)
		if !types.HasOption(types.RelationshipOptions, dependent.Relationship) {
			return req.MakeErr("Seleccione el vínculo familiar.")
		}
		if dependent.FirstName == "" || dependent.LastName == "" {
			return req.MakeErr("Ingrese los nombres y el apellido paterno del derechohabiente.")
		}
		if !crm.IsIdentityDocType(dependent.IdentityDocType) || len(strings.TrimSpace(dependent.DocumentNumber)) < 6 {
			return req.MakeErr("Ingrese el tipo y número de documento del derechohabiente.")
		}
		// A child's birth date decides the asignación familiar.
		if dependent.Relationship == types.RelationshipChild && dependent.BirthDate <= 0 {
			return req.MakeErr("Ingrese la fecha de nacimiento del hijo(a).")
		}
		if dependent.ValidFrom <= 0 {
			return req.MakeErr("Ingrese desde cuándo se registra el derechohabiente.")
		}
		if dependent.ValidTo > 0 && dependent.ValidTo < dependent.ValidFrom {
			return req.MakeErr("La fecha de baja no puede ser anterior a la de alta.")
		}
		dependent.Status = types.StatusActive
	}

	dependent.CompanyID, dependent.UpdatedBy = companyID, req.User.ID
	records := []types.Dependent{dependent}
	if dependent.ID == 0 {
		records[0].Created, records[0].CreatedBy = core.SUnixTime(), req.User.ID
		if err := db.Insert(&records); err != nil {
			return req.MakeErr("Error al guardar el derechohabiente:", err)
		}
	} else {
		stored := []types.Dependent{}
		query := db.Query(&stored)
		query.CompanyID.Equals(companyID).ID.Equals(dependent.ID)
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al leer el derechohabiente:", err)
		}
		if len(stored) == 0 || stored[0].EmployeeID != dependent.EmployeeID {
			return req.MakeErr("El derechohabiente no existe para este trabajador.")
		}
		table := db.TableOf[types.Dependent]()
		if err := db.Update(&records, table.EmployeeID, table.Relationship, table.IdentityDocType, table.DocumentNumber,
			table.FirstName, table.LastName, table.SecondLastName, table.BirthDate, table.IsStudying,
			table.ValidFrom, table.ValidTo, table.Status, table.UpdatedBy); err != nil {
			return req.MakeErr("Error al guardar el derechohabiente:", err)
		}
	}
	if err := saveDependentsSummary(employee, req.User.ID); err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(records[0])
}

// saveDependentsSummary rewrites the employee's dependents count and asignación familiar ranges
// from all his dependents, so the payroll and the employee list never read the dependents.
func saveDependentsSummary(employee types.Employee, userID int32) error {
	dependents := []types.Dependent{}
	query := db.Query(&dependents)
	query.CompanyID.Equals(employee.CompanyID)
	if err := query.Exec(); err != nil {
		return fmt.Errorf("error al leer los derechohabientes: %v", err)
	}
	employee.SetDependentsSummary(slices.DeleteFunc(dependents, func(dependent types.Dependent) bool {
		return dependent.EmployeeID != employee.ID
	}))
	employee.UpdatedBy = userID
	employees := []types.Employee{employee}
	table := db.TableOf[types.Employee]()
	if err := db.Update(&employees, table.DependentsCount, table.FamilyAllowance, table.UpdatedBy); err != nil {
		return fmt.Errorf("error al actualizar el trabajador: %v", err)
	}
	return nil
}
