import { formatTime } from '#libs/helpers.ts';
import { isProjectArchived, projectCode, type IProject } from '../project.ts';

// visibleProjects: the cards the Proyectos page shows. Archived projects only when asked; the
// filter text matches the code (P01), the name and the description.
export const visibleProjects = (projects: IProject[], filterText: string, showArchived: boolean): IProject[] =>
  projects.filter((project) => {
    if (project.ss === 0 || (isProjectArchived(project) && !showArchived)) { return false }
    if (!filterText) { return true }
    return [projectCode(project), project.Name, project.Description].join(' ').toLowerCase().includes(filterText)
  })

// projectDatesLabel: "01-03-2026 → 30-06-2026", one side when only one is set, '' with none.
export const projectDatesLabel = (project: IProject): string => {
  if (!project.StartDate && !project.EndDate) { return '' }
  const startLabel = project.StartDate ? formatTime(project.StartDate, 'd/m/Y') : '…'
  const endLabel = project.EndDate ? formatTime(project.EndDate, 'd/m/Y') : '…'
  return `${startLabel} → ${endLabel}`
}
