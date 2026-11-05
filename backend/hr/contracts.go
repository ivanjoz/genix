package hr

import (
	"encoding/json"
	"fmt"
	"strings"

	"app/core"
	"app/db"
	"app/hr/types"
)

// GetContracts syncs every contract version of the company. Served only by the payroll access:
// it carries salaries, pension and bank data.
func GetContracts(req *core.HandlerArgs) core.HandlerResponse {
	contracts := []types.Contract{}
	query := db.Query(&contracts)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los contratos:", err)
	}
	return core.MakeResponse(req, &contracts)
}

// PostContract creates or edits a contract version; Status 0 deletes it.
func PostContract(req *core.HandlerArgs) core.HandlerResponse {
	contract := types.Contract{}
	if err := json.Unmarshal([]byte(*req.Body), &contract); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	employee, err := loadEmployee(companyID, contract.EmployeeID)
	if err != nil {
		return req.MakeErr(err)
	}
	if !(contract.Status == types.StatusDeleted && contract.ID > 0) {
		if err := validateContract(companyID, employee, &contract); err != nil {
			return req.MakeErr(err)
		}
		contract.Status = types.StatusActive
	}

	contract.CompanyID, contract.UpdatedBy = companyID, req.User.ID
	records := []types.Contract{contract}
	if contract.ID == 0 {
		records[0].Created, records[0].CreatedBy = core.SUnixTime(), req.User.ID
		if err := db.Insert(&records); err != nil {
			return req.MakeErr("Error al guardar el contrato:", err)
		}
		return req.MakeResponse(records[0])
	}
	stored, err := loadEmployeeContracts(companyID, contract.EmployeeID)
	if err != nil {
		return req.MakeErr(err)
	}
	if !hasContract(stored, contract.ID) {
		return req.MakeErr("El contrato no existe para este trabajador.")
	}
	table := db.TableOf[types.Contract]()
	err = db.Update(&records, table.EmployeeID, table.ValidFrom, table.ContractType, table.StartDate, table.EndDate,
		table.MonthlySalary, table.DailyHours, table.PensionSystem, table.CUSPP, table.AfpCommissionType,
		table.HasEPS, table.SalaryBankID, table.SalaryAccount, table.CtsBankID, table.CtsAccount,
		table.Status, table.UpdatedBy)
	if err != nil {
		return req.MakeErr("Error al guardar el contrato:", err)
	}
	return req.MakeResponse(records[0])
}

// validateContract checks the dates, the salary and the pension and bank data the payroll engine
// and the bank files will read, and clears the fields that do not apply.
func validateContract(companyID int32, employee types.Employee, contract *types.Contract) error {
	if !types.HasOption(types.ContractTypeOptions, contract.ContractType) {
		return fmt.Errorf("seleccione el tipo de contrato")
	}
	if contract.StartDate <= 0 {
		return fmt.Errorf("ingrese la fecha de inicio del contrato")
	}
	if contract.StartDate < employee.HireDate {
		return fmt.Errorf("el contrato no puede empezar antes de la fecha de ingreso del trabajador")
	}
	// Only an indefinite contract may be open-ended.
	if contract.ContractType == types.ContractTypeIndefinite {
		contract.EndDate = 0
	} else if contract.EndDate < contract.StartDate {
		return fmt.Errorf("un contrato a plazo fijo necesita una fecha de fin posterior a la de inicio")
	}
	if contract.ValidFrom == 0 {
		contract.ValidFrom = contract.StartDate
	}
	if contract.ValidFrom < contract.StartDate || (contract.EndDate > 0 && contract.ValidFrom > contract.EndDate) {
		return fmt.Errorf("la vigencia de esta versión debe caer dentro de las fechas del contrato")
	}
	if contract.MonthlySalary <= 0 {
		return fmt.Errorf("ingrese la remuneración mensual")
	}
	if contract.DailyHours <= 0 || contract.DailyHours > 12 {
		return fmt.Errorf("las horas diarias deben estar entre 1 y 12")
	}

	if !types.HasOption(types.PensionSystemOptions, contract.PensionSystem) {
		return fmt.Errorf("seleccione el sistema de pensiones (ONP o AFP)")
	}
	if types.IsAfp(contract.PensionSystem) {
		contract.CUSPP = strings.ToUpper(strings.TrimSpace(contract.CUSPP))
		if len(contract.CUSPP) != 12 {
			return fmt.Errorf("el CUSPP de la AFP debe tener 12 caracteres")
		}
		if !types.HasOption(types.AfpCommissionTypeOptions, contract.AfpCommissionType) {
			return fmt.Errorf("seleccione el tipo de comisión de la AFP (flujo o mixta)")
		}
	} else {
		contract.CUSPP, contract.AfpCommissionType = "", 0
	}

	if err := validateBankAccount(contract.SalaryBankID, &contract.SalaryAccount, "sueldo"); err != nil {
		return err
	}
	if err := validateBankAccount(contract.CtsBankID, &contract.CtsAccount, "CTS"); err != nil {
		return err
	}

	// One version per employee and start date: two would make "the version valid on a date" ambiguous.
	stored, err := loadEmployeeContracts(companyID, contract.EmployeeID)
	if err != nil {
		return err
	}
	for _, other := range stored {
		if other.ID != contract.ID && other.Status == types.StatusActive && other.ValidFrom == contract.ValidFrom {
			return fmt.Errorf("ya existe una versión del contrato vigente desde esa fecha")
		}
	}
	return nil
}

// validateBankAccount accepts no bank (the worker is paid in cash) or a known bank with an account.
func validateBankAccount(bankID int8, account *string, purpose string) error {
	*account = strings.TrimSpace(*account)
	if bankID == 0 {
		*account = ""
		return nil
	}
	if !types.HasOption(types.BankOptions, bankID) {
		return fmt.Errorf("el banco de la cuenta de %s no es válido", purpose)
	}
	if len(*account) < 8 {
		return fmt.Errorf("ingrese el número de cuenta de %s", purpose)
	}
	return nil
}

func loadEmployeeContracts(companyID, employeeID int32) ([]types.Contract, error) {
	contracts := []types.Contract{}
	query := db.Query(&contracts)
	query.CompanyID.Equals(companyID).EmployeeID.Equals(employeeID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los contratos: %v", err)
	}
	return contracts, nil
}

func hasContract(contracts []types.Contract, contractID int32) bool {
	for _, contract := range contracts {
		if contract.ID == contractID {
			return true
		}
	}
	return false
}
