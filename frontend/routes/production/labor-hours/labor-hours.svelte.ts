import { notifyFailure } from '@genix/ui/notify'
import { tr } from '#core/store.svelte.ts'
import { extractError, GET } from '#libs/ui-runtime.svelte.ts'
import { ContractsAgreementsService } from '#services/hr/contracts-agreements.svelte.ts'
import { EmployeeRecordsYearService } from '#services/hr/employee-records.svelte.ts'
import { yearOfPeriod } from '#services/hr/employee-records.ts'
import { ProductionWorkersService } from '#services/production/production-settings.svelte.ts'
import { OrganizationService } from '#services/hr/organization.svelte.ts'
import { PayslipsService } from '#services/hr/payroll.svelte.ts'
import { ProductionLaborTypesService } from '#services/production/production-settings.svelte.ts'
import { buildLaborHoursReport, type IProductionLaborEntry } from './labor-hours.ts'

/** The month's labor entries: a plain list (not delta), read once per period and kept in memory. */
const laborEntriesByPeriod = new Map<number, IProductionLaborEntry[]>()

const getLaborEntriesOfPeriod = async (period: number): Promise<IProductionLaborEntry[]> => {
  if (!laborEntriesByPeriod.has(period)) {
    const entries = await GET({ route: `production-labor-entries-period?period=${period}` }) as IProductionLaborEntry[]
    laborEntriesByPeriod.set(period, entries || [])
  }
  return laborEntriesByPeriod.get(period)!
}

/** Everything the "Horas vs planilla" report reads, and the report of the selected month once all arrived. */
export class LaborHoursReader {
  employeesService = new ProductionWorkersService(true)
  organizationService = new OrganizationService(true)
  agreementsService = new ContractsAgreementsService(true)
  laborTypesService = new ProductionLaborTypesService(true)

  period = $state(0)
  entries: IProductionLaborEntry[] | null = $state(null)
  payslipsService: PayslipsService | undefined = $state()
  recordsService: EmployeeRecordsYearService | undefined = $state()
  // The pinned services fetch on construction; one per period / year is kept while the page lives.
  private payslipsByPeriod = new Map<number, PayslipsService>()
  private recordsByYear = new Map<number, EmployeeRecordsYearService>()

  isLoading = $derived(this.entries === null || !this.payslipsService?.hasFetched || !this.recordsService?.hasFetched
    || !this.employeesService.hasFetched || !this.organizationService.hasFetched || !this.agreementsService.hasFetched)

  report = $derived(this.isLoading ? null : buildLaborHoursReport({
    period: this.period,
    entries: this.entries || [],
    employees: this.employeesService.records,
    departments: this.organizationService.Departments,
    agreements: this.agreementsService.records,
    payslips: this.payslipsService?.records || [],
    records: this.recordsService?.records || [],
  }))

  async read(period: number): Promise<void> {
    this.period = period
    this.entries = null
    if (!this.payslipsByPeriod.has(period)) this.payslipsByPeriod.set(period, new PayslipsService(period))
    this.payslipsService = this.payslipsByPeriod.get(period)
    const year = yearOfPeriod(period)
    if (!this.recordsByYear.has(year)) this.recordsByYear.set(year, new EmployeeRecordsYearService(year))
    this.recordsService = this.recordsByYear.get(year)

    try {
      const entries = await getLaborEntriesOfPeriod(period)
      // A late answer for a month left behind is dropped.
      if (period === this.period) this.entries = entries
    } catch (error) {
      console.error('[labor-hours] entries load error', { period, error })
      notifyFailure(extractError(error) || tr('The hours of the month could not be read.|No se pudieron leer las horas del mes.'))
    }
  }
}
