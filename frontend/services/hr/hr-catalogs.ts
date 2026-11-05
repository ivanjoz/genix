// HR catalogs, copied from backend/hr/types/catalogs.go. The backend validates against the same
// lists; run `go run ./scripts sync_struct_interfaces` after changing them there.

export interface IHROption {
  id: number
  /** SUNAT / T-Registro code, converted at the export boundary. Never sent. */
  code?: string
  name: string
}

//CATALOG:hr.CostFunctionOptions
export const COST_FUNCTION_OPTIONS: IHROption[] = [
  { id: 1, name: 'Administration|Administración' },
  { id: 2, name: 'Sales|Ventas' },
  { id: 3, name: 'Production|Producción' },
]

//CATALOG:hr.SexOptions
export const SEX_OPTIONS: IHROption[] = [
  { id: 1, code: '1', name: 'Male|Masculino' },
  { id: 2, code: '2', name: 'Female|Femenino' },
]

//CATALOG:hr.PensionSystemOptions
export const PENSION_SYSTEM_OPTIONS: IHROption[] = [
  { id: 1, name: 'ONP|ONP' },
  { id: 2, name: 'AFP Habitat|AFP Habitat' },
  { id: 3, name: 'AFP Integra|AFP Integra' },
  { id: 4, name: 'AFP Prima|AFP Prima' },
  { id: 5, name: 'AFP Profuturo|AFP Profuturo' },
]

//CATALOG:hr.AfpCommissionTypeOptions
export const AFP_COMMISSION_TYPE_OPTIONS: IHROption[] = [
  { id: 1, name: 'On flow|Sobre el flujo' },
  { id: 2, name: 'Mixed|Mixta' },
]

//CATALOG:hr.ContractTypeOptions
export const CONTRACT_TYPE_OPTIONS: IHROption[] = [
  { id: 1, code: '01', name: 'Indefinite|Plazo indeterminado' },
  { id: 2, code: '02', name: 'Part time|Tiempo parcial' },
  { id: 3, code: '03', name: 'Start of activity|Inicio o incremento de actividad' },
  { id: 4, code: '04', name: 'Market needs|Necesidades del mercado' },
  { id: 7, code: '07', name: 'Replacement|Suplencia' },
  { id: 9, code: '09', name: 'Specific work or service|Obra determinada o servicio específico' },
  { id: 10, code: '10', name: 'Intermittent|Intermitente' },
  { id: 11, code: '11', name: 'Seasonal|De temporada' },
]

//CATALOG:hr.TerminationReasonOptions
export const TERMINATION_REASON_OPTIONS: IHROption[] = [
  { id: 1, code: '01', name: 'Resignation|Renuncia' },
  { id: 3, code: '03', name: 'Dismissal|Despido' },
  { id: 5, code: '05', name: 'Retirement|Jubilación' },
  { id: 7, code: '07', name: 'Death|Fallecimiento' },
  { id: 8, code: '08', name: 'End of contract|Término de contrato' },
  { id: 9, code: '09', name: 'Mutual agreement|Mutuo disenso' },
]

//CATALOG:hr.RelationshipOptions
export const RELATIONSHIP_OPTIONS: IHROption[] = [
  { id: 1, name: 'Spouse|Cónyuge' },
  { id: 2, name: 'Cohabitant|Concubino(a)' },
  { id: 3, name: 'Child|Hijo(a)' },
]

//CATALOG:hr.BankOptions
export const BANK_OPTIONS: IHROption[] = [
  { id: 1, name: 'BCP|BCP' },
  { id: 2, name: 'BBVA|BBVA' },
  { id: 3, name: 'Interbank|Interbank' },
  { id: 4, name: 'Scotiabank|Scotiabank' },
  { id: 5, name: 'Banco de la Nación|Banco de la Nación' },
  { id: 6, name: 'BanBif|BanBif' },
  { id: 7, name: 'Banco Pichincha|Banco Pichincha' },
]

// Payroll catalogs (backend/hr/types/legal_concepts.go, payroll.go, employee_records.go,
// legal_parameters.go).

//CATALOG:hr.ConceptKindOptions
export const CONCEPT_KIND_OPTIONS: IHROption[] = [
  { id: 1, name: 'Earning|Ingreso' },
  { id: 2, name: 'Deduction|Descuento' },
  { id: 3, name: 'Employer contribution|Aporte del empleador' },
  { id: 4, name: 'Provision|Provisión' },
]

//CATALOG:hr.IncomeTaxOptions
export const INCOME_TAX_OPTIONS: IHROption[] = [
  { id: 0, name: 'Not taxed|Inafecto' },
  { id: 1, name: 'Ordinary (projected)|Ordinaria (proyectada)' },
  { id: 2, name: 'Extraordinary|Extraordinaria' },
]

//CATALOG:hr.InputTypeOptions
export const INPUT_TYPE_OPTIONS: IHROption[] = [
  { id: 1, name: 'Unpaid absence (days)|Inasistencias (días)' },
  { id: 2, name: 'Overtime 25% (hours)|Horas extras 25% (horas)' },
  { id: 3, name: 'Overtime 35% (hours)|Horas extras 35% (horas)' },
  { id: 4, name: 'Concept amount|Monto de concepto' },
  { id: 5, name: 'Extraordinary payment|Pago extraordinario' },
]

//CATALOG:hr.TimeOffTypeOptions
export const TIME_OFF_TYPE_OPTIONS: IHROption[] = [
  { id: 30, code: '23', name: 'Vacation|Vacaciones' },
  { id: 31, code: '20', name: 'Medical leave (paid by the company)|Descanso médico (pagado por la empresa)' },
  { id: 32, code: '21', name: 'Medical leave (EsSalud subsidy)|Descanso médico (subsidio EsSalud)' },
  { id: 33, code: '22', name: 'Maternity|Maternidad' },
  { id: 34, code: '05', name: 'Unpaid leave|Licencia sin goce' },
]

//CATALOG:hr.AdvanceTypeOptions
export const ADVANCE_TYPE_OPTIONS: IHROption[] = [
  { id: 10, name: 'Weekly|Semanal' },
  { id: 11, name: 'Biweekly|Quincenal' },
  { id: 12, name: 'Other|Otro' },
]

export interface ILegalParameterOption {
  id: number
  name: string
  /** "cents" or "bp" (basis points: 900 = 9 %). */
  unit: string
}

//CATALOG:hr.LegalParameterOptions
export const LEGAL_PARAMETER_OPTIONS: ILegalParameterOption[] = [
  { id: 1, name: 'Minimum wage (RMV)|Remuneración mínima vital (RMV)', unit: 'cents' },
  { id: 2, name: 'Tax unit (UIT)|Unidad impositiva tributaria (UIT)', unit: 'cents' },
  { id: 3, name: 'EsSalud rate|Tasa EsSalud', unit: 'bp' },
  { id: 4, name: 'EsSalud rate with EPS|Tasa EsSalud con EPS', unit: 'bp' },
  { id: 5, name: 'ONP rate|Tasa ONP', unit: 'bp' },
  { id: 6, name: 'Max. insurable salary (AFP premium)|Remuneración máxima asegurable', unit: 'cents' },
  { id: 7, name: 'Family allowance (% of RMV)|Asignación familiar (% de la RMV)', unit: 'bp' },
  { id: 8, name: 'Gratificación bonus (Ley 30334)|Bonificación extraordinaria (Ley 30334)', unit: 'bp' },
  { id: 9, name: 'Gratificación bonus with EPS|Bonificación extraordinaria con EPS', unit: 'bp' },
]

/** A legal payroll concept and its treatment (plan §4.1). */
export interface ILegalConcept {
  id: number
  name: string
  kind: number
  plameCode?: string
  isRemunerative?: boolean
  affectsEssalud?: boolean
  affectsPension?: boolean
  incomeTax?: number
  computableForBenefits?: boolean
  expenseAccountCode?: number
  payableAccountCode?: number
}

//CATALOG:hr.LegalConcepts
export const LEGAL_CONCEPTS: ILegalConcept[] = [
  { id: 1, name: 'Basic salary|Remuneración básica', kind: 1, plameCode: '0121', isRemunerative: true, affectsEssalud: true, affectsPension: true, incomeTax: 1, computableForBenefits: true, expenseAccountCode: 6211 },
  { id: 2, name: 'Family allowance|Asignación familiar', kind: 1, plameCode: '0201', isRemunerative: true, affectsEssalud: true, affectsPension: true, incomeTax: 1, computableForBenefits: true, expenseAccountCode: 6211 },
  { id: 3, name: 'Overtime 25%|Horas extras 25%', kind: 1, plameCode: '0105', isRemunerative: true, affectsEssalud: true, affectsPension: true, incomeTax: 1, computableForBenefits: true, expenseAccountCode: 6211 },
  { id: 4, name: 'Overtime 35%|Horas extras 35%', kind: 1, plameCode: '0106', isRemunerative: true, affectsEssalud: true, affectsPension: true, incomeTax: 1, computableForBenefits: true, expenseAccountCode: 6211 },
  { id: 5, name: 'Vacation pay|Remuneración vacacional', kind: 1, plameCode: '0118', isRemunerative: true, affectsEssalud: true, affectsPension: true, incomeTax: 1, expenseAccountCode: 6215 },
  { id: 6, name: 'Gratificación|Gratificación', kind: 1, plameCode: '0406', isRemunerative: true, incomeTax: 1, expenseAccountCode: 6214 },
  { id: 7, name: 'Gratificación bonus (Ley 30334)|Bonificación extraordinaria (Ley 30334)', kind: 1, plameCode: '0312', incomeTax: 1, expenseAccountCode: 6214 },
  { id: 8, name: 'CTS|CTS', kind: 1, plameCode: '0904', expenseAccountCode: 629, payableAccountCode: 415 },
  { id: 9, name: 'Profit sharing|Participación en utilidades', kind: 1, plameCode: '0910', incomeTax: 2, expenseAccountCode: 622 },
  { id: 10, name: 'Dismissal indemnity|Indemnización por despido', kind: 1, plameCode: '0501', expenseAccountCode: 623 },
  { id: 20, name: 'AFP fund contribution|Aporte obligatorio AFP', kind: 2, plameCode: '0608', payableAccountCode: 417 },
  { id: 21, name: 'AFP insurance premium|Prima de seguro AFP', kind: 2, plameCode: '0606', payableAccountCode: 417 },
  { id: 22, name: 'AFP commission|Comisión AFP', kind: 2, plameCode: '0601', payableAccountCode: 417 },
  { id: 23, name: 'ONP|ONP (SNP)', kind: 2, plameCode: '0607', payableAccountCode: 4032 },
  { id: 24, name: 'Income tax 5th category|Renta de quinta categoría', kind: 2, plameCode: '0605', payableAccountCode: 40173 },
  { id: 25, name: 'Advances paid|Adelantos pagados', kind: 2, plameCode: '0701', payableAccountCode: 1412 },
  { id: 26, name: 'Loan installment|Cuota de préstamo', kind: 2, plameCode: '0706', payableAccountCode: 1411 },
  { id: 40, name: 'EsSalud|EsSalud', kind: 3, plameCode: '0804', expenseAccountCode: 6271, payableAccountCode: 4031 },
  { id: 50, name: 'Gratificación provision|Provisión de gratificación', kind: 4, expenseAccountCode: 6214, payableAccountCode: 4114 },
  { id: 51, name: 'Gratificación bonus provision|Provisión de bonificación extraordinaria', kind: 4, expenseAccountCode: 6214, payableAccountCode: 4114 },
  { id: 52, name: 'CTS provision|Provisión de CTS', kind: 4, expenseAccountCode: 629, payableAccountCode: 415 },
  { id: 53, name: 'Vacation provision|Provisión de vacaciones', kind: 4, expenseAccountCode: 6215, payableAccountCode: 4115 },
]
