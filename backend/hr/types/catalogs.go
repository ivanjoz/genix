// Catalogs of the HR module. Each list is written once here: the save handlers validate against
// it, and the //CATALOG: markers in frontend/services/hr/hr-catalogs.ts copy it into the pickers
// (`go run ./scripts sync_struct_interfaces`).
//
// The SUNAT/T-Registro codes are kept as Code so the PLAME and T-Registro exports (plan phase 6)
// can convert at that boundary; the stored value is always the int ID, like the identity docs.
// The T-Registro codes below must be checked against the official tables before those exports ship.

package types

const (
	StatusDeleted int8 = 0
	StatusActive  int8 = 1
)

// An employee who left keeps every record: he is terminated, never deleted, so his payslips,
// contracts and profit sharing of the year still resolve.
const EmployeeStatusTerminated int8 = 2

// HROption is one entry of an HR catalog. Name carries the app's "EN|ES" bilingual form.
type HROption struct {
	ID   int8   `cb:"1"`
	Code string `cb:"2"`
	Name string `cb:"3"`
}

// CostFunction tells where a department's payroll cost belongs in the income statement by
// function. Stamped on each payslip (plan §5.5).
const (
	CostFunctionAdministration int8 = 1
	CostFunctionSales          int8 = 2
	CostFunctionProduction     int8 = 3
)

var CostFunctionOptions = []HROption{
	{ID: CostFunctionAdministration, Name: "Administration|Administración"},
	{ID: CostFunctionSales, Name: "Sales|Ventas"},
	{ID: CostFunctionProduction, Name: "Production|Producción"},
}

const (
	SexMale   int8 = 1
	SexFemale int8 = 2
)

var SexOptions = []HROption{
	{ID: SexMale, Code: "1", Name: "Male|Masculino"},
	{ID: SexFemale, Code: "2", Name: "Female|Femenino"},
}

// PensionSystem: the worker is in the public system (ONP) or in one of the four AFPs. The AFP
// rates themselves are a dated table (plan §3.3), not part of this catalog.
const (
	PensionSystemONP       int8 = 1
	PensionSystemHabitat   int8 = 2
	PensionSystemIntegra   int8 = 3
	PensionSystemPrima     int8 = 4
	PensionSystemProfuturo int8 = 5
)

var PensionSystemOptions = []HROption{
	{ID: PensionSystemONP, Name: "ONP|ONP"},
	{ID: PensionSystemHabitat, Name: "AFP Habitat|AFP Habitat"},
	{ID: PensionSystemIntegra, Name: "AFP Integra|AFP Integra"},
	{ID: PensionSystemPrima, Name: "AFP Prima|AFP Prima"},
	{ID: PensionSystemProfuturo, Name: "AFP Profuturo|AFP Profuturo"},
}

// IsAfp reports whether the pension system is private: only then CUSPP and commission type apply.
func IsAfp(pensionSystem int8) bool {
	return pensionSystem >= PensionSystemHabitat && pensionSystem <= PensionSystemProfuturo
}

// AfpCommissionType: commission on the monthly flow, or the mixed scheme (0% on flow since 2023,
// charged on the balance instead).
const (
	AfpCommissionFlow  int8 = 1
	AfpCommissionMixed int8 = 2
)

var AfpCommissionTypeOptions = []HROption{
	{ID: AfpCommissionFlow, Name: "On flow|Sobre el flujo"},
	{ID: AfpCommissionMixed, Name: "Mixed|Mixta"},
}

// ContractType follows T-Registro's "tipo de contrato". Only the indefinite one may omit EndDate.
const (
	ContractTypeIndefinite    int8 = 1
	ContractTypePartTime      int8 = 2
	ContractTypeActivityStart int8 = 3
	ContractTypeMarketNeeds   int8 = 4
	ContractTypeSeasonal      int8 = 11
	ContractTypeSpecificWork  int8 = 9
	ContractTypeReplacement   int8 = 7
	ContractTypeIntermittent  int8 = 10
)

var ContractTypeOptions = []HROption{
	{ID: ContractTypeIndefinite, Code: "01", Name: "Indefinite|Plazo indeterminado"},
	{ID: ContractTypePartTime, Code: "02", Name: "Part time|Tiempo parcial"},
	{ID: ContractTypeActivityStart, Code: "03", Name: "Start of activity|Inicio o incremento de actividad"},
	{ID: ContractTypeMarketNeeds, Code: "04", Name: "Market needs|Necesidades del mercado"},
	{ID: ContractTypeReplacement, Code: "07", Name: "Replacement|Suplencia"},
	{ID: ContractTypeSpecificWork, Code: "09", Name: "Specific work or service|Obra determinada o servicio específico"},
	{ID: ContractTypeIntermittent, Code: "10", Name: "Intermittent|Intermitente"},
	{ID: ContractTypeSeasonal, Code: "11", Name: "Seasonal|De temporada"},
}

// TerminationReason follows T-Registro's "motivo de baja".
const (
	TerminationResignation int8 = 1
	TerminationDismissal   int8 = 3
	TerminationContractEnd int8 = 8
	TerminationRetirement  int8 = 5
	TerminationDeath       int8 = 7
	TerminationMutual      int8 = 9
)

var TerminationReasonOptions = []HROption{
	{ID: TerminationResignation, Code: "01", Name: "Resignation|Renuncia"},
	{ID: TerminationDismissal, Code: "03", Name: "Dismissal|Despido"},
	{ID: TerminationRetirement, Code: "05", Name: "Retirement|Jubilación"},
	{ID: TerminationDeath, Code: "07", Name: "Death|Fallecimiento"},
	{ID: TerminationContractEnd, Code: "08", Name: "End of contract|Término de contrato"},
	{ID: TerminationMutual, Code: "09", Name: "Mutual agreement|Mutuo disenso"},
}

// Relationship of a dependent (EsSalud "derechohabiente"). A child earns the worker the
// asignación familiar while a minor, or while studying up to 24 (plan §3.1).
const (
	RelationshipSpouse     int8 = 1
	RelationshipCohabitant int8 = 2
	RelationshipChild      int8 = 3
)

var RelationshipOptions = []HROption{
	{ID: RelationshipSpouse, Name: "Spouse|Cónyuge"},
	{ID: RelationshipCohabitant, Name: "Cohabitant|Concubino(a)"},
	{ID: RelationshipChild, Name: "Child|Hijo(a)"},
}

// Banks for salary and CTS deposits. Each bank's mass-payment file is its own layout (phase 6).
const (
	BankBCP        int8 = 1
	BankBBVA       int8 = 2
	BankInterbank  int8 = 3
	BankScotiabank int8 = 4
	BankNacion     int8 = 5
	BankBanBif     int8 = 6
	BankPichincha  int8 = 7
)

var BankOptions = []HROption{
	{ID: BankBCP, Name: "BCP|BCP"},
	{ID: BankBBVA, Name: "BBVA|BBVA"},
	{ID: BankInterbank, Name: "Interbank|Interbank"},
	{ID: BankScotiabank, Name: "Scotiabank|Scotiabank"},
	{ID: BankNacion, Name: "Banco de la Nación|Banco de la Nación"},
	{ID: BankBanBif, Name: "BanBif|BanBif"},
	{ID: BankPichincha, Name: "Banco Pichincha|Banco Pichincha"},
}

// HasOption reports whether id is one of the catalog's entries.
func HasOption(options []HROption, id int8) bool {
	for _, option := range options {
		if option.ID == id {
			return true
		}
	}
	return false
}

// Employee.EngagementType values (docs/MANUFACTURING_PLAN.md §9.5, M28).
const (
	EngagementPayroll      int8 = 1 // Planilla
	EngagementFeeReceipt   int8 = 2 // Recibo por honorarios (4ta categoría)
	EngagementTemporaryOff int8 = 3 // Temporal sin planilla
)

var EngagementTypeOptions = []HROption{
	{ID: EngagementPayroll, Name: "Payroll|Planilla"},
	{ID: EngagementFeeReceipt, Name: "Fee receipt|Recibo por honorarios"},
	{ID: EngagementTemporaryOff, Name: "Temporary, off payroll|Temporal sin planilla"},
}

// ContractAgreement.PayBasis values.
const (
	PayBasisMonthly int8 = 1
	PayBasisWeekly  int8 = 2
	PayBasisHourly  int8 = 3
)

var PayBasisOptions = []HROption{
	{ID: PayBasisMonthly, Name: "Monthly|Mensual"},
	{ID: PayBasisWeekly, Name: "Weekly|Semanal"},
	{ID: PayBasisHourly, Name: "Hourly|Por hora"},
}
