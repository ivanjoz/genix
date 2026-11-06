package sample_records

import (
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"strings"

	"app/core"
	crm "app/crm/types"
	"app/hr"
	"app/hr/types"
)

// The staff of the demo company, written by the HR seed (seed_hr.go): departments with their cost
// function, positions, employees with invented Peruvian identities, contract versions (some with a
// raise), dependents and a couple of terminated workers. The mix covers the payroll engine's
// branches: ONP and the four AFPs, flow and mixed commission, EPS, part-time, fixed-term contracts,
// children that grant the asignación familiar and children that no longer do.
//
// Everything goes through the production handlers of app/hr, so the run also exercises their
// validation.

const (
	hrDataActorID int32 = 1
	// minimumWageCents is the RMV from 2026-10-01 (S/ 1,230): no seeded salary goes below it.
	minimumWageCents int32 = 123000
)

type hrDataDepartment struct {
	Name         string
	CostFunction int8
	Positions    []hrDataPosition
}

// hrDataPosition carries the salary band of the job, in soles, and how many people hold it.
// ReportsTo names the position whose holders are the boss of this one ("" = top of the company);
// the holders are shared out evenly. A boss position must be listed before the ones reporting to it.
type hrDataPosition struct {
	Name      string
	MinSalary int32
	MaxSalary int32
	Headcount int
	ReportsTo string
}

// The org chart has four levels (general manager → area managers → team heads → staff), plus a
// fifth on one sales branch (a cashier's assistant).
var hrDataDepartments = []hrDataDepartment{
	{Name: "Gerencia General", CostFunction: types.CostFunctionAdministration, Positions: []hrDataPosition{
		{Name: "Gerente General", MinSalary: 9000, MaxSalary: 12000, Headcount: 1},
	}},
	{Name: "Administración y Finanzas", CostFunction: types.CostFunctionAdministration, Positions: []hrDataPosition{
		{Name: "Gerente de Administración y Finanzas", MinSalary: 6500, MaxSalary: 8000, Headcount: 1, ReportsTo: "Gerente General"},
		{Name: "Contador", MinSalary: 4200, MaxSalary: 5500, Headcount: 1, ReportsTo: "Gerente de Administración y Finanzas"},
		{Name: "Jefe de Tesorería", MinSalary: 3800, MaxSalary: 4800, Headcount: 1, ReportsTo: "Gerente de Administración y Finanzas"},
		{Name: "Asistente Contable", MinSalary: 1800, MaxSalary: 2400, Headcount: 2, ReportsTo: "Contador"},
		{Name: "Asistente Administrativo", MinSalary: 1500, MaxSalary: 2200, Headcount: 2, ReportsTo: "Jefe de Tesorería"},
	}},
	{Name: "Ventas", CostFunction: types.CostFunctionSales, Positions: []hrDataPosition{
		{Name: "Gerente Comercial", MinSalary: 6500, MaxSalary: 8000, Headcount: 1, ReportsTo: "Gerente General"},
		{Name: "Jefe de Tienda", MinSalary: 2800, MaxSalary: 3600, Headcount: 2, ReportsTo: "Gerente Comercial"},
		{Name: "Cajero", MinSalary: 1230, MaxSalary: 1500, Headcount: 6, ReportsTo: "Jefe de Tienda"},
		{Name: "Reponedor", MinSalary: 1230, MaxSalary: 1400, Headcount: 4, ReportsTo: "Jefe de Tienda"},
		// One cashier coaches an assistant: a store team that mixes leaves and a small sub-team.
		{Name: "Asistente de Ventas", MinSalary: 1230, MaxSalary: 1350, Headcount: 1, ReportsTo: "Cajero"},
	}},
	{Name: "Almacén y Logística", CostFunction: types.CostFunctionProduction, Positions: []hrDataPosition{
		{Name: "Gerente de Logística", MinSalary: 6000, MaxSalary: 7500, Headcount: 1, ReportsTo: "Gerente General"},
		{Name: "Jefe de Almacén", MinSalary: 2600, MaxSalary: 3200, Headcount: 1, ReportsTo: "Gerente de Logística"},
		{Name: "Jefe de Transporte", MinSalary: 2600, MaxSalary: 3200, Headcount: 1, ReportsTo: "Gerente de Logística"},
		{Name: "Almacenero", MinSalary: 1300, MaxSalary: 1700, Headcount: 4, ReportsTo: "Jefe de Almacén"},
		{Name: "Chofer Repartidor", MinSalary: 1600, MaxSalary: 2100, Headcount: 2, ReportsTo: "Jefe de Transporte"},
	}},
}

var (
	hrDataMaleNames   = []string{"José", "Luis", "Carlos", "Jorge", "Miguel", "Juan", "Pedro", "César", "Víctor", "Raúl", "Diego", "Renzo", "Kevin", "Jhon", "Wilmer"}
	hrDataFemaleNames = []string{"María", "Rosa", "Ana", "Carmen", "Lucía", "Milagros", "Patricia", "Gladys", "Sofía", "Valeria", "Karina", "Yesenia", "Flor", "Luz", "Elena"}
	hrDataSurnames    = []string{"Quispe", "Flores", "Sánchez", "Rodríguez", "García", "Huamán", "Mamani", "Chávez", "Rojas", "Vásquez", "Ramírez", "Torres", "Mendoza", "Castillo", "Gutiérrez", "Paredes", "Salazar", "Ccori", "Condori", "Espinoza"}
	hrDataStreets     = []string{"Av. Perú", "Jr. Huánuco", "Av. Túpac Amaru", "Calle Los Pinos", "Av. Universitaria", "Jr. Ayacucho", "Av. Próceres"}
	// hrDataAfps are the four AFP ids; ONP is drawn separately.
	hrDataAfps  = []int8{types.PensionSystemHabitat, types.PensionSystemIntegra, types.PensionSystemPrima, types.PensionSystemProfuturo}
	hrDataBanks = []int8{types.BankBCP, types.BankBCP, types.BankBBVA, types.BankInterbank, types.BankScotiabank}
)

type hrDataGenerator struct {
	random          *mrand.Rand
	today           int16
	usedDocuments   map[string]bool
	contractsCount  int
	dependentsCount int
	terminatedCount int
	// holdersByPosition lists who holds each position, so a position's boss is found by name.
	holdersByPosition map[string][]int32
}

// isBossPosition: somebody reports to the position. Its holders are never terminated, so every
// chain of bosses ends at an active person.
func isBossPosition(positionName string) bool {
	for _, departmentSeed := range hrDataDepartments {
		for _, positionSeed := range departmentSeed.Positions {
			if positionSeed.ReportsTo == positionName {
				return true
			}
		}
	}
	return false
}

// seedHRStaff writes the departments, positions and employees as they stand on `today`: hire dates,
// raises and the few terminations all fall before it. The caller checks the company has no staff.
func seedHRStaff(seed int64, today int16) error {
	generator := hrDataGenerator{random: mrand.New(mrand.NewSource(seed)), today: today, usedDocuments: map[string]bool{},
		holdersByPosition: map[string][]int32{}}
	employeesCount := 0
	for _, departmentSeed := range hrDataDepartments {
		department := types.Department{}
		if err := callHRHandler("POST.hr-departments", hr.PostDepartment, nil,
			types.Department{Name: departmentSeed.Name, CostFunction: departmentSeed.CostFunction}, &department); err != nil {
			return err
		}
		for _, positionSeed := range departmentSeed.Positions {
			position := types.Position{}
			if err := callHRHandler("POST.hr-positions", hr.PostPosition, nil,
				types.Position{Name: positionSeed.Name, DepartmentID: department.ID}, &position); err != nil {
				return err
			}
			for holderIndex := range positionSeed.Headcount {
				// Holders are shared out evenly among the bosses of the position they report to.
				var managerID int32
				if bosses := generator.holdersByPosition[positionSeed.ReportsTo]; len(bosses) > 0 {
					managerID = bosses[holderIndex%len(bosses)]
				}
				if err := generator.seedEmployee(department, position, positionSeed, managerID); err != nil {
					return fmt.Errorf("falló el trabajador %d: %w", employeesCount+1, err)
				}
				employeesCount++
			}
		}
	}
	core.Log("SeedHR:: personal", "trabajadores", employeesCount, "cesados", generator.terminatedCount,
		"contratos", generator.contractsCount, "derechohabientes", generator.dependentsCount)
	return nil
}

// seedEmployee creates one worker with his contract history and family.
func (generator *hrDataGenerator) seedEmployee(department types.Department, position types.Position, positionSeed hrDataPosition, managerID int32) error {
	random := generator.random
	sex := types.SexMale
	firstName := hrDataMaleNames[random.Intn(len(hrDataMaleNames))]
	if random.Intn(2) == 0 {
		sex, firstName = types.SexFemale, hrDataFemaleNames[random.Intn(len(hrDataFemaleNames))]
	}
	// Dates are UnixDay int16, so a birth before 1970 can't be stored: keep the oldest under 54.
	ageYears := 19 + random.Intn(35)
	birthDate := generator.today - int16(ageYears*365+random.Intn(365))
	// Hired between 2 months and 8 years ago, never before turning 18.
	hireDate := generator.today - int16(60+random.Intn(8*365))
	if minimumHire := birthDate + 18*365; hireDate < minimumHire {
		hireDate = minimumHire
	}

	employee := types.Employee{
		IdentityDocType: crm.IdentityDocDNI, DocumentNumber: generator.uniqueDNI(),
		FirstName: firstName, LastName: generator.pickSurname(), SecondLastName: generator.pickSurname(),
		BirthDate: birthDate, Sex: sex, CountryID: 604,
		Address:      fmt.Sprintf("%s %d", hrDataStreets[random.Intn(len(hrDataStreets))], 100+random.Intn(2400)),
		Phone:        fmt.Sprintf("9%08d", random.Intn(100000000)),
		DepartmentID: department.ID, PositionID: position.ID, HireDate: hireDate, ManagerEmployeeID: managerID,
	}
	employee.Email = strings.ToLower(fmt.Sprintf("%s.%s@demo.genix.pe", asciiOnly(firstName), asciiOnly(employee.LastName)))
	if err := callHRHandler("POST.hr-employees", hr.PostEmployee, nil, employee, &employee); err != nil {
		return err
	}
	generator.holdersByPosition[position.Name] = append(generator.holdersByPosition[position.Name], employee.ID)

	if err := generator.seedContracts(employee, positionSeed); err != nil {
		return err
	}
	if err := generator.seedDependents(employee, ageYears); err != nil {
		return err
	}

	// One worker in twelve has already left: resignation or the end of his fixed-term contract.
	if !isBossPosition(position.Name) && random.Intn(12) == 0 && generator.today-hireDate > 120 {
		employee.TerminationDate = generator.today - int16(10+random.Intn(90))
		employee.TerminationReason = types.TerminationResignation
		if random.Intn(2) == 0 {
			employee.TerminationReason = types.TerminationContractEnd
		}
		if err := callHRHandler("POST.hr-employees", hr.PostEmployee, nil, employee, &employee); err != nil {
			return err
		}
		generator.terminatedCount++
	}
	return nil
}

// seedContracts writes the first contract at the hire date and, for workers with more than a year,
// a raise one year later: two versions the payroll must pick between by date.
func (generator *hrDataGenerator) seedContracts(employee types.Employee, positionSeed hrDataPosition) error {
	random := generator.random
	salaryCents := (positionSeed.MinSalary + random.Int31n(positionSeed.MaxSalary-positionSeed.MinSalary+1)) * 100
	salaryCents = max(salaryCents/1000*1000, minimumWageCents) // round to S/ 10

	contract := types.Contract{
		EmployeeID: employee.ID, StartDate: employee.HireDate, MonthlySalary: salaryCents, DailyHours: 8,
		ContractType: types.ContractTypeIndefinite,
	}
	// Recent hires are on fixed-term contracts of a year.
	if generator.today-employee.HireDate < 400 && random.Intn(3) > 0 {
		contract.ContractType, contract.EndDate = types.ContractTypeMarketNeeds, employee.HireDate+364
	}
	// A few part-timers (under 4 hours: no CTS, reduced vacations) on the shop floor.
	if positionSeed.MaxSalary <= 1500 && random.Intn(8) == 0 {
		contract.ContractType, contract.EndDate, contract.DailyHours = types.ContractTypePartTime, employee.HireDate+364, 3
		contract.MonthlySalary = minimumWageCents / 2
	}

	// One in four is in the ONP; the rest in an AFP, mostly on the mixed commission.
	contract.PensionSystem = types.PensionSystemONP
	if random.Intn(4) > 0 {
		contract.PensionSystem = hrDataAfps[random.Intn(len(hrDataAfps))]
		contract.CUSPP = generator.cuspp(employee)
		contract.AfpCommissionType = types.AfpCommissionMixed
		if random.Intn(4) == 0 {
			contract.AfpCommissionType = types.AfpCommissionFlow
		}
	}
	contract.HasEPS = salaryCents >= 300000 && random.Intn(3) == 0
	contract.SalaryBankID = hrDataBanks[random.Intn(len(hrDataBanks))]
	contract.SalaryAccount = fmt.Sprintf("%03d-%08d-0-%02d", 190+random.Intn(9), random.Intn(100000000), random.Intn(100))
	contract.CtsBankID = hrDataBanks[random.Intn(len(hrDataBanks))]
	contract.CtsAccount = fmt.Sprintf("%03d-%08d-1-%02d", 190+random.Intn(9), random.Intn(100000000), random.Intn(100))

	if err := callHRHandler("POST.hr-contracts", hr.PostContract, nil, contract, &contract); err != nil {
		return err
	}
	generator.contractsCount++

	if contract.ContractType != types.ContractTypeIndefinite || generator.today-employee.HireDate < 400 || random.Intn(2) == 0 {
		return nil
	}
	raise := contract
	raise.ID, raise.ValidFrom = 0, employee.HireDate+365
	raise.MonthlySalary = (contract.MonthlySalary * int32(105+random.Intn(8)) / 100) / 1000 * 1000
	if err := callHRHandler("POST.hr-contracts", hr.PostContract, nil, raise, &raise); err != nil {
		return err
	}
	generator.contractsCount++
	return nil
}

// seedDependents gives about half of the workers over 25 a family: a spouse and one to three children,
// some already adults (a few still studying, so they keep granting the asignación familiar).
func (generator *hrDataGenerator) seedDependents(employee types.Employee, ageYears int) error {
	random := generator.random
	if ageYears < 25 || random.Intn(2) == 0 {
		return nil
	}
	dependents := []types.Dependent{}
	if random.Intn(3) > 0 {
		spouseSex := types.SexFemale
		if employee.Sex == types.SexFemale {
			spouseSex = types.SexMale
		}
		dependents = append(dependents, types.Dependent{
			Relationship: types.RelationshipSpouse, FirstName: generator.pickFirstName(spouseSex),
			LastName: generator.pickSurname(), SecondLastName: generator.pickSurname(),
			BirthDate: employee.BirthDate + int16(random.Intn(1800)-900),
		})
	}
	maximumChildAge := min(ageYears-20, 28)
	for range 1 + random.Intn(3) {
		childAge := random.Intn(maximumChildAge + 1)
		dependents = append(dependents, types.Dependent{
			Relationship: types.RelationshipChild, FirstName: generator.pickFirstName(int8(1 + random.Intn(2))),
			LastName: employee.LastName, SecondLastName: generator.pickSurname(),
			BirthDate:  generator.today - int16(childAge*365+random.Intn(365)),
			IsStudying: childAge >= 18 && childAge < 24 && random.Intn(2) == 0,
		})
	}
	for _, dependent := range dependents {
		dependent.EmployeeID, dependent.IdentityDocType = employee.ID, crm.IdentityDocDNI
		dependent.DocumentNumber, dependent.ValidFrom = generator.uniqueDNI(), employee.HireDate
		if err := callHRHandler("POST.hr-dependents", hr.PostDependent, nil, dependent, nil); err != nil {
			return err
		}
		generator.dependentsCount++
	}
	return nil
}

func (generator *hrDataGenerator) uniqueDNI() string {
	for {
		documentNumber := fmt.Sprintf("%08d", 10000000+generator.random.Intn(80000000))
		if !generator.usedDocuments[documentNumber] {
			generator.usedDocuments[documentNumber] = true
			return documentNumber
		}
	}
}

func (generator *hrDataGenerator) pickSurname() string {
	return hrDataSurnames[generator.random.Intn(len(hrDataSurnames))]
}

func (generator *hrDataGenerator) pickFirstName(sex int8) string {
	if sex == types.SexFemale {
		return hrDataFemaleNames[generator.random.Intn(len(hrDataFemaleNames))]
	}
	return hrDataMaleNames[generator.random.Intn(len(hrDataMaleNames))]
}

// cuspp builds a 12-character AFP code shaped like the real ones: digits, then initials.
func (generator *hrDataGenerator) cuspp(employee types.Employee) string {
	initials := strings.ToUpper(asciiOnly(employee.FirstName)[:1] + asciiOnly(employee.LastName)[:1] + asciiOnly(employee.SecondLastName)[:1])
	return fmt.Sprintf("%06d%s%03d", generator.random.Intn(1000000), initials, generator.random.Intn(1000))
}

// asciiOnly drops the accents of a Spanish name, for e-mails and codes.
func asciiOnly(text string) string {
	return strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ñ", "N").Replace(text)
}

// callHRHandler runs a production handler of app/hr as the demo company's admin.
func callHRHandler(route string, handler func(*core.HandlerArgs) core.HandlerResponse, query map[string]string, payload any, target any) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body := string(bodyBytes)
	method, _, _ := strings.Cut(route, ".")
	request := core.HandlerArgs{
		Body: &body, Query: query, Route: route, Method: method,
		// The actor holds the payroll access (41) with its sub-accesses 2–4 (bit subID−1): calculate,
		// close and pay, which the run handlers check.
		User: &core.UsuarioToken{CompanyID: sampleCompanyID, ID: hrDataActorID, SubAccesos: map[int32]uint16{41: 0b1110}},
	}
	response := handler(&request)
	if response.StatusCode != 200 {
		return fmt.Errorf("%s respondió %d: %s", route, response.StatusCode, response.Error)
	}
	if target == nil || response.Body == nil {
		return nil
	}
	return json.Unmarshal(*response.Body, target)
}
