import { describe, expect, it } from 'vitest'
import type { IContract } from '#services/hr/contracts.svelte.ts'
import type { IEmployee } from '#services/hr/employees.svelte.ts'
import {
  buildEmployeeTree, completedYears, contractValidOn, employeeFullName, hasFamilyAllowanceOn, showsDepartmentInOrgChart,
} from './employees.ts'

describe('buildEmployeeTree', () => {
  const employee = (ID: number, FirstName: string, ManagerEmployeeID: number, ss = 1) =>
    ({ ID, FirstName, LastName: '', SecondLastName: '', ManagerEmployeeID, ss } as IEmployee)

  it('nests by immediate boss and sorts by name', () => {
    const tree = buildEmployeeTree([employee(1, 'Gerente', 0), employee(3, 'Zoe', 1), employee(2, 'Ana', 1), employee(4, 'Beto', 3)])
    expect(tree.map((node) => node.item.FirstName)).toEqual(['Gerente'])
    expect(tree[0].children.map((node) => node.item.FirstName)).toEqual(['Ana', 'Zoe'])
    expect(tree[0].children[1].children[0].item.FirstName).toBe('Beto')
  })

  it('makes roots of people whose boss is gone and cuts a cycle', () => {
    const tree = buildEmployeeTree([
      employee(1, 'A', 9), employee(2, 'B', 3), employee(3, 'C', 2), employee(9, 'Terminated', 0, 2),
    ])
    // A's boss is terminated; B and C point at each other: B becomes a root and the cycle is cut under C.
    expect(tree.map((node) => node.item.FirstName)).toEqual(['A', 'B'])
    expect(tree[1].children[0].item.FirstName).toBe('C')
    expect(tree[1].children[0].children).toEqual([])
  })
})

describe('showsDepartmentInOrgChart', () => {
  const employee = (DepartmentID: number) => ({ DepartmentID } as IEmployee)

  it('names the department on a root and where it differs from the boss', () => {
    expect(showsDepartmentInOrgChart(employee(1), undefined)).toBe(true)
    expect(showsDepartmentInOrgChart(employee(2), employee(1))).toBe(true)
    expect(showsDepartmentInOrgChart(employee(1), employee(1))).toBe(false)
  })
})

const unixDayOf =(isoDate: string) => Math.floor(Date.parse(isoDate) / 86400000)

const makeContract = (ID: number, EmployeeID: number, ValidFrom: string, ss = 1) =>
  ({ ID, EmployeeID, ValidFrom: unixDayOf(ValidFrom), ss } as IContract)

describe('employeeFullName', () => {
  it('lists surnames first and skips the missing ones', () => {
    expect(employeeFullName({ FirstName: 'Ana', LastName: 'Quispe', SecondLastName: 'Mamani' })).toBe('Quispe Mamani, Ana')
    expect(employeeFullName({ FirstName: 'Ana', LastName: 'Quispe', SecondLastName: '' })).toBe('Quispe, Ana')
  })
})

describe('contractValidOn', () => {
  const contracts = [
    makeContract(1, 7, '2025-01-01'),
    makeContract(2, 7, '2026-03-01'),
    makeContract(3, 7, '2026-06-01', 0), // deleted version never applies
    makeContract(4, 8, '2026-04-01'), // another employee
  ]

  it('picks the latest version that already started', () => {
    expect(contractValidOn(contracts, 7, unixDayOf('2026-02-28'))?.ID).toBe(1)
    expect(contractValidOn(contracts, 7, unixDayOf('2026-03-01'))?.ID).toBe(2)
    expect(contractValidOn(contracts, 7, unixDayOf('2026-07-01'))?.ID).toBe(2)
  })

  it('is undefined before the first version', () => {
    expect(contractValidOn(contracts, 7, unixDayOf('2024-12-31'))).toBeUndefined()
  })
})

describe('completedYears', () => {
  it('counts a year only on the anniversary', () => {
    expect(completedYears(unixDayOf('2008-11-05'), unixDayOf('2026-11-04'))).toBe(17)
    expect(completedYears(unixDayOf('2008-11-05'), unixDayOf('2026-11-05'))).toBe(18)
  })
})

describe('hasFamilyAllowanceOn', () => {
  // The ranges come from the server (Employee.FamilyAllowance); both ends are included.
  const employee = { FamilyAllowance: [{ From: unixDayOf('2020-01-01'), To: unixDayOf('2026-11-04') }] }

  it('reads the day inside a range, ends included', () => {
    expect(hasFamilyAllowanceOn(employee, unixDayOf('2026-11-04'))).toBe(true)
    expect(hasFamilyAllowanceOn(employee, unixDayOf('2026-11-05'))).toBe(false)
    expect(hasFamilyAllowanceOn(employee, unixDayOf('2019-12-31'))).toBe(false)
  })

  it('has none without ranges', () => {
    expect(hasFamilyAllowanceOn({ FamilyAllowance: [] }, unixDayOf('2026-11-04'))).toBe(false)
  })
})
