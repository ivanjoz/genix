package hr

import (
	"encoding/json"
	"fmt"
	"strings"

	"app/core"
	"app/db"
	"app/hr/types"
)

// GetContractsAgreements syncs every contract version of the company. Served only by the payroll access:
// it carries salaries, pension and bank data.
func GetContractsAgreements(req *core.HandlerArgs) core.HandlerResponse {
	contracts := []types.ContractAgreement{}
	query := db.Query(&contracts)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los contratos:", err)
	}
	return core.MakeResponse(req, &contracts)
}

// PostContractAgreement creates or edits a contract version; Status 0 deletes it.
func PostContractAgreement(req *core.HandlerArgs) core.HandlerResponse {
	contract := types.ContractAgreement{}
	if err := json.Unmarshal([]byte(*req.Body), &contract); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	employee, err := loadEmployee(companyID, contract.EmployeeID)
	if err != nil {
		return req.MakeErr(err)
	}
	if !(contract.Status == types.StatusDeleted && contract.ID > 0) {
		if err := validateContractAgreement(companyID, employee, &contract); err != nil {
			return req.MakeErr(err)
		}
		contract.Status = types.StatusActive
	}

	contract.CompanyID, contract.UpdatedBy = companyID, req.User.ID
	records := []types.ContractAgreement{contract}
	if contract.ID == 0 {
		records[0].Created, records[0].CreatedBy = core.SUnixTime(), req.User.ID
		if err := db.Upsert(&records); err != nil {
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
	table := db.TableOf[types.ContractAgreement]()
	err = db.Update(&records, table.EmployeeID, table.ValidFrom, table.ContractType, table.StartDate, table.EndDate,
		table.PayAmount, table.DailyHours, table.PensionSystem, table.CUSPP, table.AfpCommissionType,
		table.HasEPS, table.SalaryBankID, table.SalaryAccount, table.CtsBankID, table.CtsAccount,
		table.PayBasis, table.PayDayOfWeek, table.Status, table.UpdatedBy)
	if err != nil {
		return req.MakeErr("Error al guardar el contrato:", err)
	}
	return req.MakeResponse(records[0])
}

// validateContractAgreement checks the dates, the pay and the pension and bank data the payroll engine
// and the bank files will read, and clears the fields that do not apply. The employee's EngagementType
// decides the shape (docs/MANUFACTURING_PLAN.md §9.5): a payroll contract is monthly with its T-Registro
// type and pension; an agreement off payroll leaves every payroll field empty, is hourly for a fee
// receipt and monthly, weekly or hourly for a temporary worker.
func validateContractAgreement(companyID int32, employee types.Employee, agreement *types.ContractAgreement) error {
	isPayroll := employee.EngagementType == types.EngagementPayroll
	switch {
	case isPayroll && agreement.PayBasis != types.PayBasisMonthly:
		return fmt.Errorf("un contrato de planilla se paga por mes")
	case employee.EngagementType == types.EngagementFeeReceipt && agreement.PayBasis != types.PayBasisHourly:
		return fmt.Errorf("un trabajador por recibo por honorarios se paga por hora")
	case !types.HasOption(types.PayBasisOptions, agreement.PayBasis):
		return fmt.Errorf("seleccione la forma de pago (mensual, semanal o por hora)")
	}

	if agreement.StartDate <= 0 {
		return fmt.Errorf("ingrese la fecha de inicio")
	}
	if agreement.StartDate < employee.HireDate {
		return fmt.Errorf("el contrato o acuerdo no puede empezar antes de la fecha de ingreso del trabajador")
	}
	if isPayroll {
		if !types.HasOption(types.ContractTypeOptions, agreement.ContractType) {
			return fmt.Errorf("seleccione el tipo de contrato")
		}
		// Only an indefinite contract may be open-ended.
		if agreement.ContractType == types.ContractTypeIndefinite {
			agreement.EndDate = 0
		} else if agreement.EndDate < agreement.StartDate {
			return fmt.Errorf("un contrato a plazo fijo necesita una fecha de fin posterior a la de inicio")
		}
	} else if agreement.EndDate != 0 && agreement.EndDate < agreement.StartDate {
		return fmt.Errorf("la fecha de fin del acuerdo es anterior a la de inicio")
	}
	if agreement.ValidFrom == 0 {
		agreement.ValidFrom = agreement.StartDate
	}
	if agreement.ValidFrom < agreement.StartDate || (agreement.EndDate > 0 && agreement.ValidFrom > agreement.EndDate) {
		return fmt.Errorf("la vigencia de esta versión debe caer dentro de sus fechas")
	}
	if agreement.PayAmount <= 0 {
		return fmt.Errorf("ingrese el monto de pago")
	}
	if agreement.PayBasis == types.PayBasisWeekly {
		if agreement.PayDayOfWeek < 1 || agreement.PayDayOfWeek > 7 {
			return fmt.Errorf("seleccione el día de pago de la semana")
		}
	} else {
		agreement.PayDayOfWeek = 0
	}
	// The daily hours turn a monthly or weekly pay into an hourly cost.
	if agreement.PayBasis != types.PayBasisHourly && (agreement.DailyHours <= 0 || agreement.DailyHours > 12) {
		return fmt.Errorf("las horas diarias deben estar entre 1 y 12")
	}
	if agreement.DailyHours < 0 || agreement.DailyHours > 12 {
		return fmt.Errorf("las horas diarias deben estar entre 0 y 12")
	}

	if isPayroll {
		if !types.HasOption(types.PensionSystemOptions, agreement.PensionSystem) {
			return fmt.Errorf("seleccione el sistema de pensiones (ONP o AFP)")
		}
		if types.IsAfp(agreement.PensionSystem) {
			agreement.CUSPP = strings.ToUpper(strings.TrimSpace(agreement.CUSPP))
			if len(agreement.CUSPP) != 12 {
				return fmt.Errorf("el CUSPP de la AFP debe tener 12 caracteres")
			}
			if !types.HasOption(types.AfpCommissionTypeOptions, agreement.AfpCommissionType) {
				return fmt.Errorf("seleccione el tipo de comisión de la AFP (flujo o mixta)")
			}
		} else {
			agreement.CUSPP, agreement.AfpCommissionType = "", 0
		}
		if err := validateBankAccount(agreement.CtsBankID, &agreement.CtsAccount, "CTS"); err != nil {
			return err
		}
	} else {
		// Off payroll: no T-Registro type, pension, EPS or CTS.
		agreement.ContractType, agreement.PensionSystem, agreement.CUSPP, agreement.AfpCommissionType = 0, 0, "", 0
		agreement.HasEPS, agreement.CtsBankID, agreement.CtsAccount = false, 0, ""
	}
	if err := validateBankAccount(agreement.SalaryBankID, &agreement.SalaryAccount, "sueldo"); err != nil {
		return err
	}

	// One version per employee and start date: two would make "the version valid on a date" ambiguous.
	stored, err := loadEmployeeContracts(companyID, agreement.EmployeeID)
	if err != nil {
		return err
	}
	for _, other := range stored {
		if other.ID != agreement.ID && other.Status == types.StatusActive && other.ValidFrom == agreement.ValidFrom {
			return fmt.Errorf("ya existe una versión vigente desde esa fecha")
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

func loadEmployeeContracts(companyID, employeeID int32) ([]types.ContractAgreement, error) {
	contracts := []types.ContractAgreement{}
	query := db.Query(&contracts)
	query.CompanyID.Equals(companyID).EmployeeID.Equals(employeeID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los contratos: %v", err)
	}
	return contracts, nil
}

func hasContract(contracts []types.ContractAgreement, contractID int32) bool {
	for _, contract := range contracts {
		if contract.ID == contractID {
			return true
		}
	}
	return false
}
