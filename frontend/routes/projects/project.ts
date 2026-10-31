// The agile projects' shared domain: the records the backend answers (backend/projects/types), their
// states and labels, and the rules the three pages (Proyectos, Backlog, Sprints) share. Pure: no
// Svelte, no fetch.

// The "Proyectos" access of backend/access.toml. View reads everything; the write level (what the
// backend asks of every POST) creates projects and works in the ones whose team it is on.
export const ACCESS_ID_PROJECTS = 39
export const ACCESS_LEVEL_WRITE = 2

export const STATUS_DELETED = 0
export const STATUS_ACTIVE = 1

export const PROJECT_ROLE_MEMBER = 1
export const PROJECT_ROLE_ADMIN = 2
export const PROJECT_ROLE_OPTIONS = [
  { id: PROJECT_ROLE_MEMBER, name: 'Member|Miembro' },
  { id: PROJECT_ROLE_ADMIN, name: 'Admin|Administrador' },
]

export const PROJECT_STATE_ACTIVE = 1
export const PROJECT_STATE_ARCHIVED = 2

export interface IProjectMember {
  UserID: number
  Role: number
}

export interface IProject {
  ID: number
  Name: string
  Description?: string
  StartDate?: number
  EndDate?: number
  Color: string
  Members: IProjectMember[]
  State: number
  CreatedBy?: number
  Created?: number
  ss: number
  upd?: number
}

// The colors a project or an epic may take: genix-ui Calendar's palette, as the backend accepts.
// Full class literals so Tailwind finds them.
export const PROJECT_COLORS: { id: string, name: string, swatchCss: string, softCss: string }[] = [
  { id: 'blue', name: 'Blue|Azul', swatchCss: 'bg-blue-solid', softCss: 'bg-blue-bg text-blue-fg' },
  { id: 'green', name: 'Green|Verde', swatchCss: 'bg-green-solid', softCss: 'bg-green-bg text-green-fg' },
  { id: 'red', name: 'Red|Rojo', swatchCss: 'bg-red-solid', softCss: 'bg-red-bg text-red-fg' },
  { id: 'amber', name: 'Amber|Ámbar', swatchCss: 'bg-amber-solid', softCss: 'bg-amber-bg text-amber-fg' },
  { id: 'purple', name: 'Purple|Morado', swatchCss: 'bg-purple-solid', softCss: 'bg-purple-bg text-purple-fg' },
  { id: 'teal', name: 'Teal|Turquesa', swatchCss: 'bg-teal-solid', softCss: 'bg-teal-bg text-teal-fg' },
  { id: 'pink', name: 'Pink|Rosado', swatchCss: 'bg-pink-solid', softCss: 'bg-pink-bg text-pink-fg' },
  { id: 'gray', name: 'Gray|Gris', swatchCss: 'bg-fg-subtle', softCss: 'bg-surface-muted text-fg' },
]

export const projectColor = (colorID: string) => PROJECT_COLORS.find((color) => color.id === colorID) || PROJECT_COLORS[0]

export const projectRole = (project: IProject, userID: number): number =>
  project.Members?.find((member) => member.UserID === userID)?.Role || 0

export const isProjectAdmin = (project: IProject, userID: number) => projectRole(project, userID) === PROJECT_ROLE_ADMIN

export const isProjectMember = (project: IProject, userID: number) => projectRole(project, userID) > 0

export const isProjectArchived = (project: IProject) => project.State === PROJECT_STATE_ARCHIVED

// canWorkInProject: the team writes in a project while it is not archived. The router also asks
// the Edit level, which every team member holds (the team picker only offers such users).
export const canWorkInProject = (project: IProject, userID: number) => isProjectMember(project, userID) && !isProjectArchived(project)

export const makeEmptyProject = (creatorUserID: number): IProject => ({
  ID: 0, Name: '', Description: '', Color: PROJECT_COLORS[0].id, State: PROJECT_STATE_ACTIVE, ss: STATUS_ACTIVE,
  Members: [{ UserID: creatorUserID, Role: PROJECT_ROLE_ADMIN }],
})

// makeProjectForm clones a project for its form, the team included: the form edits the copy, never
// the cached list row.
export const makeProjectForm = (project: IProject): IProject => ({
  ...project, Members: (project.Members || []).map((member) => ({ ...member })),
})

// validateProject mirrors backend/projects normalizeProject: the backend checks again.
export const validateProject = (project: IProject): string => {
  if (!(project.Name || '').trim() || project.Name.trim().length > 120) {
    return 'The project needs a name of up to 120 characters.|El proyecto necesita un nombre de hasta 120 caracteres.'
  }
  if (project.StartDate && project.EndDate && project.StartDate > project.EndDate) {
    return 'The project cannot end before it starts.|El proyecto no puede terminar antes de empezar.'
  }
  if (!project.Members.some((member) => member.Role === PROJECT_ROLE_ADMIN)) {
    return 'The team needs at least one admin.|El equipo necesita al menos un administrador.'
  }
  return ''
}

// sortProjects: active ones first, then by name.
export const sortProjects = (projects: IProject[]): IProject[] =>
  [...projects].sort((left, right) =>
    Number(isProjectArchived(left)) - Number(isProjectArchived(right)) || left.Name.localeCompare(right.Name))


// ------------------------------------------------------------------------------ backlog records

export interface IEpic {
  ProjectID: number
  ID: number
  Name: string
  Color: string
  ss: number
  upd?: number
}

export const makeEmptyEpic = (projectID: number): IEpic => ({ ProjectID: projectID, ID: 0, Name: '', Color: PROJECT_COLORS[0].id, ss: STATUS_ACTIVE })

// changedEpics: what the project panel's Epics tab sends on save. Its forms hold the stored epics
// (a removed one with ss 0) and the new ones (ID 0); only new, renamed, recolored or removed
// epics are written.
export const changedEpics = (storedEpics: IEpic[], epicForms: IEpic[]): IEpic[] => {
  const storedEpicsByID = new Map(storedEpics.map((epic) => [epic.ID, epic]))
  return epicForms.filter((epicForm) => {
    const storedEpic = storedEpicsByID.get(epicForm.ID)
    return !storedEpic || storedEpic.Name !== epicForm.Name || storedEpic.Color !== epicForm.Color || storedEpic.ss !== epicForm.ss
  })
}

// validateEpics mirrors backend/projects normalizeEpics: every kept epic is named.
export const validateEpics = (epics: IEpic[]): string =>
  epics.some((epic) => epic.ss !== STATUS_DELETED && (!epic.Name.trim() || epic.Name.trim().length > 80))
    ? 'Every epic needs a name of up to 80 characters.|Cada épica necesita un nombre de hasta 80 caracteres.'
    : ''

export const STORY_STATE_NOT_PLANNED = 0
export const STORY_STATE_PENDING = 1
export const STORY_STATE_IN_DEVELOPMENT = 2
export const STORY_STATE_STOPPED = 3
export const STORY_STATE_IN_REVIEW = 4
export const STORY_STATE_FINISHED = 5

// The labels and badge colors of a story's state. Full class literals so Tailwind finds them.
export const STORY_STATES: Record<number, { name: string, badgeCss: string }> = {
  [STORY_STATE_NOT_PLANNED]: { name: 'Not planned|Sin planificar', badgeCss: 'bg-surface-muted text-fg-muted' },
  [STORY_STATE_PENDING]: { name: 'Pending|Pendiente', badgeCss: 'bg-blue-bg-strong text-blue-fg' },
  [STORY_STATE_IN_DEVELOPMENT]: { name: 'In development|En desarrollo', badgeCss: 'bg-accent-bg-strong text-accent-fg' },
  [STORY_STATE_STOPPED]: { name: 'Blocked|Bloqueado', badgeCss: 'bg-red-bg-strong text-red-fg' },
  [STORY_STATE_IN_REVIEW]: { name: 'In review|En revisión', badgeCss: 'bg-amber-bg-strong text-amber-fg' },
  [STORY_STATE_FINISHED]: { name: 'Finished|Finalizado', badgeCss: 'bg-green-bg-strong text-green-fg' },
}

// The API omits a zero State: a story never planned arrives without it.
export const storyState = (state: number | undefined) => STORY_STATES[state || STORY_STATE_NOT_PLANNED] || STORY_STATES[STORY_STATE_NOT_PLANNED]

export const STORY_TYPE_STORY = 1
export const STORY_TYPE_BUG = 2
export const STORY_TYPE_TASK = 3
export const STORY_TYPES: { id: number, name: string, icon: string, iconCss: string }[] = [
  { id: STORY_TYPE_STORY, name: 'Story|Historia', icon: 'icon-[fa--bookmark]', iconCss: 'text-green-solid' },
  { id: STORY_TYPE_BUG, name: 'Bug|Bug', icon: 'icon-[fa--bug]', iconCss: 'text-red-solid' },
  { id: STORY_TYPE_TASK, name: 'Task|Tarea', icon: 'icon-[fa--check-square]', iconCss: 'text-blue-solid' },
]
export const storyType = (typeID: number) => STORY_TYPES.find((type) => type.id === typeID) || STORY_TYPES[0]

export const STORY_PRIORITY_MEDIUM = 2
export const STORY_PRIORITIES: { id: number, name: string, icon: string, iconCss: string }[] = [
  { id: 1, name: 'Low|Baja', icon: 'icon-[fa--angle-double-down]', iconCss: 'text-fg-subtle' },
  { id: 2, name: 'Medium|Media', icon: 'icon-[fa--minus]', iconCss: 'text-amber-solid' },
  { id: 3, name: 'High|Alta', icon: 'icon-[fa--angle-double-up]', iconCss: 'text-orange-solid' },
  { id: 4, name: 'Critical|Crítica', icon: 'icon-[fa--exclamation-circle]', iconCss: 'text-red-solid' },
]
export const storyPriority = (priorityID: number) => STORY_PRIORITIES.find((priority) => priority.id === priorityID) || STORY_PRIORITIES[1]

export interface IStory {
  ProjectID: number
  // Packs ProjectID and the project's own story sequence (storyNumber).
  ID: number
  Type: number
  Title: string
  Summary?: string
  EpicID?: number
  Priority: number
  AssigneeID?: number
  EstimatedHours?: number
  BacklogRank: number
  // The final state: the one in the sprint it was most recently added to (computed by the backend).
  State: number
  LastSprintID?: number
  SprintIDs?: number[]
  // The last day (UnixDay) the story entered each state.
  StateDates?: { State: number, Date: number }[]
  CommentsCount?: number
  AttachmentsCount?: number
  CreatedBy?: number
  Created?: number
  ss: number
  upd?: number
}

// rankBetween mirrors backend/projects rankBetween (backlog and board ranks): the midpoint of the
// neighbours, or one step past the only one.
export const rankBetween = (previousRank: number | null, nextRank: number | null): number => {
  if (previousRank !== null && nextRank !== null) { return (previousRank + nextRank) / 2 }
  if (previousRank !== null) { return previousRank + 1 }
  if (nextRank !== null) { return nextRank - 1 }
  return 1
}

export interface ISprintStory {
  StoryID: number
  State: number
  Rank: number
  AddedAt: number
  StartedAt?: number
  DoneAt?: number
  CarriedToSprintID?: number
}

export const SPRINT_STATE_PLANNED = 1
export const SPRINT_STATE_ACTIVE = 2
export const SPRINT_STATE_CLOSED = 3

export interface ISprint {
  ProjectID: number
  ID: number
  Name: string
  Goal?: string
  StartDate: number
  EndDate: number
  State: number
  Stories?: ISprintStory[]
  ClosedAt?: number
  ClosedBy?: number
  ss: number
  upd?: number
}

// The codes are the autoincrement numbers, at least two digits: a project is P01 (its ID), a story
// H07 (its number inside the project). The page always shows the project's stories together, so
// the story code needs no project prefix.
export const projectCode = (project: IProject) => `P${String(project.ID).padStart(2, '0')}`

// STORY_NUMBER_SLOT mirrors backend/projects/types projectSequenceBits: a story ID is
// ProjectID * 2^23 + its number inside the project. Modulo, not a bitmask: JS bit operators are 32-bit.
const STORY_NUMBER_SLOT = 2 ** 23
export const storyNumber = (storyID: number) => storyID % STORY_NUMBER_SLOT
export const storyCode = (story: IStory) => `H${String(storyNumber(story.ID)).padStart(2, '0')}`

export const isSprintOpen = (sprint: ISprint) => sprint.ss !== 0 && sprint.State !== SPRINT_STATE_CLOSED

// storySprintNote is the sprint line of a story in the backlog: the open sprint it is in, or the
// closed sprint it came back from unfinished ("Not completed in Sprint 3").
export const storySprintNote = (story: IStory, sprintsByID: Map<number, ISprint>): { text: string, isNotCompleted: boolean } | null => {
  const lastSprint = story.LastSprintID ? sprintsByID.get(story.LastSprintID) : undefined
  if (!lastSprint) { return null }
  if (isSprintOpen(lastSprint)) { return { text: lastSprint.Name, isNotCompleted: false } }
  if (story.State !== STORY_STATE_FINISHED) {
    return { text: `Not completed in ${lastSprint.Name}|No culminada en ${lastSprint.Name}`, isNotCompleted: true }
  }
  return { text: lastSprint.Name, isNotCompleted: false }
}

export const makeUserLabel =(user: { Username: string, FirstName?: string, LastName?: string }): string =>
  [user.FirstName, user.LastName].filter(Boolean).join(' ').trim() || user.Username
