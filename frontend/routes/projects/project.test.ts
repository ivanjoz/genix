import { describe, expect, test } from 'bun:test';
import {
  PROJECT_ROLE_ADMIN, PROJECT_ROLE_MEMBER, PROJECT_STATE_ARCHIVED, STATUS_ACTIVE, STATUS_DELETED, canWorkInProject, changedEpics,
  makeEmptyEpic, makeEmptyProject, makeProjectForm, makeUserLabel, projectCode, projectRole, sortProjects, storyCode, teamCandidates,
  validateEpics, validateProject, type IProject,
} from './project.ts';
import { makeEmptyStory } from './story/story.ts';

const makeProject = (overrides: Partial<IProject> = {}): IProject => ({
  ...makeEmptyProject(1), ID: 1, Name: 'Fertirriego', ...overrides,
})

describe('codes', () => {
  test('projects and stories go by their number, with at least two digits', () => {
    expect(projectCode(makeProject({ ID: 1 }))).toBe('P01')
    expect(projectCode(makeProject({ ID: 123 }))).toBe('P123')
    expect(storyCode({ ...makeEmptyStory(1), ID: 3 * 2 ** 23 + 55 })).toBe('H55')
    expect(storyCode({ ...makeEmptyStory(1), ID: 3 * 2 ** 23 + 7 })).toBe('H07')
  })
})

describe('validateProject', () => {
  test('a named project with an admin is valid', () => {
    expect(validateProject(makeProject())).toBe('')
    expect(validateProject(makeProject({ Name: ' ' }))).not.toBe('')
  })
  test('the team keeps an admin', () => {
    expect(validateProject(makeProject({ Members: [{ UserID: 1, Role: PROJECT_ROLE_MEMBER }] }))).not.toBe('')
  })
  test('a project cannot end before it starts', () => {
    expect(validateProject(makeProject({ StartDate: 20, EndDate: 10 }))).not.toBe('')
  })
})

describe('project roles', () => {
  test('the creator is the first admin', () => {
    expect(projectRole(makeEmptyProject(7), 7)).toBe(PROJECT_ROLE_ADMIN)
  })
  test('the team works only while the project is not archived', () => {
    expect(canWorkInProject(makeProject(), 1)).toBe(true)
    expect(canWorkInProject(makeProject(), 2)).toBe(false)
    expect(canWorkInProject(makeProject({ State: PROJECT_STATE_ARCHIVED }), 1)).toBe(false)
  })
})

test('makeProjectForm copies the team, so the form never edits the cached row', () => {
  const project = makeProject()
  const projectForm = makeProjectForm(project)
  projectForm.Members[0].Role = PROJECT_ROLE_MEMBER
  expect(project.Members[0].Role).toBe(PROJECT_ROLE_ADMIN)
})

test('sortProjects lists active projects first, then by name', () => {
  const sorted = sortProjects([
    makeProject({ ID: 1, Name: 'Zeta' }), makeProject({ ID: 2, Name: 'Alfa', State: PROJECT_STATE_ARCHIVED }), makeProject({ ID: 3, Name: 'Beta' }),
  ])
  expect(sorted.map((project) => project.ID)).toEqual([3, 1, 2])
})

describe('epics of the project panel', () => {
  const storedEpics = [
    { ProjectID: 1, ID: 1, Name: 'Catálogo', Color: 'blue', ss: STATUS_ACTIVE },
    { ProjectID: 1, ID: 2, Name: 'Pagos', Color: 'green', ss: STATUS_ACTIVE },
  ]
  test('only new, edited and removed epics are saved', () => {
    const epicForms = [
      { ...storedEpics[0] },
      { ...storedEpics[1], ss: STATUS_DELETED },
      { ...makeEmptyEpic(1), Name: 'Reportes' },
    ]
    expect(changedEpics(storedEpics, epicForms).map((epic) => epic.ID)).toEqual([2, 0])
    expect(changedEpics(storedEpics, [{ ...storedEpics[0], Color: 'red' }])).toHaveLength(1)
  })
  test('kept epics need a name; removed ones do not', () => {
    expect(validateEpics([makeEmptyEpic(1)])).not.toBe('')
    expect(validateEpics([{ ...makeEmptyEpic(1), ID: 3, ss: STATUS_DELETED }])).toBe('')
  })
})

describe('team candidates', () => {
  // Grant words of access 39 (Proyectos): [0, 157] = level 2, [0, 156] = level 1, [0, 158, 1] = level 3 with a sub-access.
  const makeUser = (ID: number, AccesosComputed: string | null, AccesosSubComputed: string | null = null) => ({
    ID, User: `user${ID}`, FirstName: '', LastName: '', AccesosComputed, AccesosSubComputed,
  })
  test('only users holding the projects access at the write level, in either grant blob', () => {
    const candidates = teamCandidates([
      makeUser(2, 'AJ0='), makeUser(3, 'AJw='), makeUser(4, null, 'AJ4B'), makeUser(5, null),
    ])
    expect(candidates.map((candidate) => candidate.ID)).toEqual([2, 4])
  })
  test('user 1 holds every access; the label falls back to the login handle', () => {
    expect(teamCandidates([makeUser(1, null)])).toEqual([{ ID: 1, Name: 'user1' }])
    expect(makeUserLabel({ User: 'ana', FirstName: 'Ana', LastName: 'Ruiz' })).toBe('Ana Ruiz')
  })
})
