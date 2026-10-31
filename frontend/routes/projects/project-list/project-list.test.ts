import { expect, test } from 'bun:test';
import { PROJECT_STATE_ARCHIVED, makeEmptyProject, type IProject } from '../project.ts';
import { projectDatesLabel, visibleProjects } from './project-list.ts';

const makeProject = (overrides: Partial<IProject>): IProject => ({ ...makeEmptyProject(1), ...overrides })

test('visibleProjects hides archived and deleted projects unless asked, and filters by code, name and description', () => {
  const projects = [
    makeProject({ ID: 1, Name: 'Fertirriego' }),
    makeProject({ ID: 2, Name: 'Old one', State: PROJECT_STATE_ARCHIVED }),
    makeProject({ ID: 3, Name: 'Deleted', ss: 0 }),
    makeProject({ ID: 4, Name: 'Packing', Description: 'Calibrado de fruta' }),
  ]
  expect(visibleProjects(projects, '', false).map((project) => project.ID)).toEqual([1, 4])
  expect(visibleProjects(projects, '', true).map((project) => project.ID)).toEqual([1, 2, 4])
  expect(visibleProjects(projects, 'fer', false).map((project) => project.ID)).toEqual([1])
  expect(visibleProjects(projects, 'p04', false).map((project) => project.ID)).toEqual([4])
  expect(visibleProjects(projects, 'calibrado', false).map((project) => project.ID)).toEqual([4])
})

test('projectDatesLabel shows what is set', () => {
  expect(projectDatesLabel(makeProject({}))).toBe('')
  expect(projectDatesLabel(makeProject({ StartDate: 20000 }))).toMatch(/→ …$/)
})
