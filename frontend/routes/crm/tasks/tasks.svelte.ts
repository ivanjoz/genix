import { todayUnixDay } from '#libs/helpers.ts'
import { security } from '#libs/ui-runtime.svelte.ts'
import { CrmActivityTypesService } from '#services/crm/crm-activity.svelte.ts'
import { CrmEmailTemplatesService } from '#services/crm/crm-email-template.svelte.ts'
import { CrmPipelinesService } from '#services/crm/crm-pipeline.svelte.ts'
import { CrmTeamsService } from '#services/crm/crm-team.svelte.ts'
import {
  CrmTasksService, completeCrmActivityTask, decideCrmApproval, dismissCrmTask, type ICrmTask,
} from '#services/crm/crm-task.svelte.ts'
import { UsuariosService } from '../../security/users-profiles/users-profiles.svelte'
import { activityTypeOptions } from '../pipeline/pipeline.ts'
import { groupClosedTasksByDay, groupOpenTasks, teamMembersLedBy } from './tasks.ts'

export type TasksView = 'mine' | 'team' | 'done'
/** Team view filter value that shows every member's tasks. */
export const ALL_TEAM_MEMBERS = 0

/**
 * State of the tasks page. Each user's open tasks and each user's closed tasks are their own delta
 * list: the page opens only the lists on screen (mine, one team member, or my closed ones).
 */
export class CrmTasksState {
  userID = security.getUserInfo()?.ID || 0
  today = todayUnixDay()
  users = new UsuariosService()
  teams = new CrmTeamsService(true)
  pipelines = new CrmPipelinesService(true)
  activityTypes = new CrmActivityTypesService(true)
  emailTemplates = new CrmEmailTemplatesService(true)

  view = $state<TasksView>('mine')
  /** The team member whose tasks the leader reads (read-only); ALL_TEAM_MEMBERS reads every member's at once. */
  teamMemberID = $state(ALL_TEAM_MEMBERS)
  mineOpen = new CrmTasksService(this.userID)
  /** One open-task list per team member, opened the first time the member is shown. */
  teamOpenByMember = $state(new Map<number, CrmTasksService>())
  mineClosed = $state<CrmTasksService | undefined>()

  teamMemberIDs = $derived(teamMembersLedBy(this.userID, this.teams.records))
  activityTypeOptions = $derived(activityTypeOptions(this.activityTypes.records))
  /** The team members whose lists the team view shows. */
  shownMemberIDs = $derived(this.teamMemberID === ALL_TEAM_MEMBERS ? this.teamMemberIDs : [this.teamMemberID])
  /** Several sellers' tasks share the screen: each card names its seller. */
  showsSeller = $derived(this.view === 'team' && this.teamMemberID === ALL_TEAM_MEMBERS)
  openGroups = $derived(groupOpenTasks(this.view === 'team'
    ? this.shownMemberIDs.flatMap((memberID) => this.teamOpenByMember.get(memberID)?.records ?? [])
    : this.mineOpen.records, this.today))
  closedDays = $derived(groupClosedTasksByDay(this.mineClosed?.records ?? []))

  constructor() {
    this.mineOpen.fetch()
  }

  /** Opens the list of the chosen view the first time it is needed. */
  selectView(view: TasksView) {
    this.view = view
    if (view === 'done' && !this.mineClosed) {
      this.mineClosed = new CrmTasksService(this.userID, true)
      this.mineClosed.fetch()
    }
    if (view === 'team') this.selectTeamMember(this.teamMemberID)
  }

  /** Shows one member's tasks, or every member's (ALL_TEAM_MEMBERS), opening the lists not held yet. */
  selectTeamMember(memberID: number) {
    this.teamMemberID = memberID
    const missingMemberIDs = this.shownMemberIDs.filter((shownMemberID) => !this.teamOpenByMember.has(shownMemberID))
    if (!missingMemberIDs.length) return
    const openedLists = new Map(this.teamOpenByMember)
    for (const missingMemberID of missingMemberIDs) {
      const memberList = new CrmTasksService(missingMemberID)
      memberList.fetch()
      openedLists.set(missingMemberID, memberList)
    }
    this.teamOpenByMember = openedLists
  }

  userName(userID: number) {
    const user = this.users.usuariosMap.get(userID)
    return user ? `${user.FirstName} ${user.LastName}`.trim() : `#${userID}`
  }

  stageName(task: ICrmTask) {
    return this.pipelines.recordsMap.get(task.PipelineID)?.Stages.find((stage) => stage.ID === task.StageID)?.Name ?? ''
  }

  /** The stage template that generated the task: it names the activity type or email template to use. */
  templateOf(task: ICrmTask) {
    return this.pipelines.recordsMap.get(task.PipelineID)?.Stages
      .find((stage) => stage.ID === task.StageID)?.TaskTemplates.find((stageTemplate) => stageTemplate.ID === task.TemplateID)
  }

  /** The assignee drops a template task. */
  dismissTask = (task: ICrmTask) => dismissCrmTask(this.mineOpen, task.ID)

  /** The assignee logs the activity the task asks for, which closes it. */
  completeActivityTask = (task: ICrmTask, typeID: number, text: string) =>
    completeCrmActivityTask(this.mineOpen, { OpportunityID: task.OpportunityID, TypeID: typeID, Text: text }, task.ID)

  decideApproval = (task: ICrmTask, approve: boolean, note: string) => decideCrmApproval(this.mineOpen, task.ID, approve, note)
}
