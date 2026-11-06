import { describe, expect, it } from 'vitest'
import { periodFirstDay } from '#routes/hr/payroll/payroll.ts'
import { RecordStatus, type IEmployeeRecord } from '#services/hr/employee-records.ts'
import { LegalConcept } from '#services/hr/hr-catalogs.ts'
import type { IConcept } from '#services/hr/payroll.svelte.ts'
import {
  employeeEntries, employeeLoanInstallments, entryConceptOptions, entryInput, entryMonthOptions, isEditableRecord,
  makeEmptyEntry,
} from './employee-records.ts'

const BONUS_CONCEPT_ID = 201

const record = (ID: number, ConceptID: number, Period: number, ss: number = RecordStatus.Active) =>
  ({ ID, ConceptID, Period, ss } as IEmployeeRecord)

describe('entryMonthOptions / makeEmptyEntry', () => {
  it('offers the twelve months of the date year', () => {
    const options = entryMonthOptions(periodFirstDay(2610) + 26)
    expect(options.map((option) => option.id)).toEqual([2601, 2602, 2603, 2604, 2605, 2606, 2607, 2608, 2609, 2610, 2611, 2612])
    expect(options[9].name).toBe('OCT')
  })

  it('pays a new record in the month of its date', () => {
    expect(makeEmptyEntry(7, periodFirstDay(2611) + 4)).toMatchObject({ EmployeeID: 7, Period: 2611, Date: periodFirstDay(2611) + 4 })
  })
})

describe('employeeEntries', () => {
  it('keeps live Registros only (company and hand-registered legal concepts), newest month first', () => {
    const entries = employeeEntries([
      record(1, LegalConcept.Absence, 2609),
      record(2, LegalConcept.Overtime25, 2611),
      record(3, BONUS_CONCEPT_ID, 2610, RecordStatus.Settled),
      record(4, BONUS_CONCEPT_ID, 2611, RecordStatus.Cancelled),
      record(5, LegalConcept.Loan, 2611),
      record(6, LegalConcept.Vacation, 2611),
      record(7, LegalConcept.AdvanceWeekly, 2611),
    ])
    expect(entries.map((entry) => entry.ID)).toEqual([2, 3, 1])
  })
})

describe('entryConceptOptions / entryInput', () => {
  it('lists overtime and absences first, then the company concepts', () => {
    const options = entryConceptOptions([{ ID: BONUS_CONCEPT_ID, Name: 'Bono' } as IConcept])
    expect(options.map((option) => option.ID)).toEqual([LegalConcept.Overtime25, LegalConcept.Overtime35, LegalConcept.Absence, BONUS_CONCEPT_ID])
  })

  it('asks days, hours or an amount by concept', () => {
    expect(entryInput(LegalConcept.Absence)).toBe('days')
    expect(entryInput(LegalConcept.Overtime35)).toBe('hours')
    expect(entryInput(BONUS_CONCEPT_ID)).toBe('amount')
    expect(entryInput(0)).toBeUndefined()
  })
})

describe('employeeLoanInstallments / isEditableRecord', () => {
  it('keeps live installments, oldest month first', () => {
    const installments = employeeLoanInstallments([
      record(1, LegalConcept.LoanInstallment, 2612),
      record(2, LegalConcept.LoanInstallment, 2610, RecordStatus.Settled),
      record(3, LegalConcept.LoanInstallment, 2611, RecordStatus.Cancelled),
    ])
    expect(installments.map((installment) => installment.ID)).toEqual([2, 1])
  })

  it('locks records a closed payslip settled', () => {
    expect(isEditableRecord(record(1, LegalConcept.Absence, 2611))).toBe(true)
    expect(isEditableRecord(record(1, LegalConcept.Absence, 2611, RecordStatus.Settled))).toBe(false)
  })
})
