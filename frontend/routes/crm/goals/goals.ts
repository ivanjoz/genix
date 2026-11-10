import { GOAL_METRIC, GOAL_PERIOD } from '#services/crm/crm-goal-metric.ts'
import type { ICrmGoal } from '#services/crm/crm-goal.svelte.ts'
import type { ICrmTeam } from '#services/crm/crm-team.svelte.ts'
import { periodFirstDay, periodLastDay, periodOfUnixDay } from '../reports/reports.ts'

// Pure logic of the goals grid (plan §3.2): sellers grouped by team × 12 months, for one pipeline
// and one metric. Draft edits live in a map keyed by the cell and are sent together.

/** Money goals are shown in currency units but stored in the opportunities' amount scale (x100). */
export const AMOUNT_SCALE = 100
export const isMoneyMetric = (metric: number) => metric === GOAL_METRIC.WON_AMOUNT

/** A goal period of a pipeline: YYMM when monthly, a week since the epoch when weekly. */
export interface GoalPeriod {
  period: number
  /** First UnixDay (a Monday for a week). */
  firstDay: number
}

// Weeks are counted in whole Monday-based weeks since the epoch (a Thursday): backend core.WeekOfDay.
export const weekOfDay = (unixDay: number) => Math.floor((unixDay + 3) / 7)
export const weekFirstDay = (week: number) => week * 7 - 3

/** Daily pipelines store one goal per UnixDay; the grid shows a month or a week of them at a time. */
export const isDailyPeriodType = (periodType: number) =>
  periodType === GOAL_PERIOD.DAILY_BY_MONTH || periodType === GOAL_PERIOD.DAILY_BY_WEEK

export const periodOfDay = (periodType: number, unixDay: number) => {
  if (isDailyPeriodType(periodType)) return unixDay
  return periodType === GOAL_PERIOD.WEEKLY ? weekOfDay(unixDay) : periodOfUnixDay(unixDay)
}

/** First and last UnixDay of a goal period. */
export const goalPeriodDays = (periodType: number, period: number): [number, number] => {
  if (isDailyPeriodType(periodType)) return [period, period]
  return periodType === GOAL_PERIOD.WEEKLY ? [weekFirstDay(period), weekFirstDay(period) + 6] : [periodFirstDay(period), periodLastDay(period)]
}

const daysBetween = (firstDay: number, lastDay: number): GoalPeriod[] =>
  Array.from({ length: lastDay - firstDay + 1 }, (_, index) => ({ period: firstDay + index, firstDay: firstDay + index }))

/** The weeks whose Monday falls in a calendar year (YY). */
const weeksOfYear = (year: number): GoalPeriod[] => {
  const firstDayOfYear = periodFirstDay(year * 100 + 1)
  const lastDayOfYear = periodLastDay(year * 100 + 12)
  const weeks: GoalPeriod[] = []
  for (let week = weekOfDay(firstDayOfYear); weekFirstDay(week) <= lastDayOfYear; week++) {
    if (weekFirstDay(week) >= firstDayOfYear) weeks.push({ period: week, firstDay: weekFirstDay(week) })
  }
  return weeks
}

/** The periods of a calendar year (YY): its 12 months, its weeks, or every day. */
export const periodsOfYear = (periodType: number, year: number): GoalPeriod[] => {
  if (isDailyPeriodType(periodType)) return daysBetween(periodFirstDay(year * 100 + 1), periodLastDay(year * 100 + 12))
  if (periodType === GOAL_PERIOD.WEEKLY) return weeksOfYear(year)
  return Array.from({ length: 12 }, (_, index) => ({ period: year * 100 + index + 1, firstDay: periodFirstDay(year * 100 + index + 1) }))
}

/**
 * What the goals grid shows at once. Monthly and weekly pipelines show the whole year; a daily one
 * shows the days of one month (DAILY_BY_MONTH) or of one week (DAILY_BY_WEEK), picked by scopeIndex
 * (the month's or the week's position in the year, from 0).
 */
export const periodsOfScope = (periodType: number, year: number, scopeIndex: number): GoalPeriod[] => {
  if (periodType === GOAL_PERIOD.DAILY_BY_MONTH) {
    const period = year * 100 + Math.min(Math.max(scopeIndex, 0), 11) + 1
    return daysBetween(periodFirstDay(period), periodLastDay(period))
  }
  if (periodType === GOAL_PERIOD.DAILY_BY_WEEK) {
    const weeks = weeksOfYear(year)
    const week = weeks[Math.min(Math.max(scopeIndex, 0), weeks.length - 1)]
    return daysBetween(week.firstDay, week.firstDay + 6)
  }
  return periodsOfYear(periodType, year)
}

/** The scope choices of a daily pipeline: month or week indexes of the year; empty otherwise. */
export const scopeCount = (periodType: number, year: number) => {
  if (periodType === GOAL_PERIOD.DAILY_BY_MONTH) return 12
  return periodType === GOAL_PERIOD.DAILY_BY_WEEK ? weeksOfYear(year).length : 0
}

/** The scope index holding a UnixDay, so the grid opens on today's month or week. */
export const scopeIndexOfDay = (periodType: number, year: number, unixDay: number) => {
  if (periodType === GOAL_PERIOD.DAILY_BY_MONTH) return Math.min(Math.max(periodOfUnixDay(unixDay) - year * 100 - 1, 0), 11)
  const weeks = weeksOfYear(year)
  return Math.max(weeks.findIndex((week) => weekOfDay(week.firstDay) === weekOfDay(unixDay)), 0)
}

export const cellKey = (userID: number, period: number) => `${userID}|${period}`

/** Stored target of one cell, in display units. */
export const storedTarget = (goals: ICrmGoal[], pipelineID: number, metric: number, userID: number, period: number) => {
  const goal = goals.find((candidate) => candidate.PipelineID === pipelineID && candidate.Metric === metric
    && candidate.UserID === userID && candidate.Period === period)
  return (goal?.Target ?? 0) / (isMoneyMetric(metric) ? AMOUNT_SCALE : 1)
}

export interface GoalGridRow {
  /** 0 for a team header row. */
  userID: number
  teamID: number
  label: string
  isTeamRow: boolean
}

/**
 * Rows grouped by team: each team's header row (its sum) followed by its members that work in the
 * pipeline, then the sellers with no team. Teams without members in the pipeline are left out.
 */
export const goalGridRows = (
  teams: ICrmTeam[], memberUserIDs: number[], userName: (userID: number) => string,
): GoalGridRow[] => {
  const rows: GoalGridRow[] = []
  const placed = new Set<number>()
  for (const team of teams) {
    const teamMembers = team.MemberUserIDs.filter((userID) => memberUserIDs.includes(userID))
    if (!teamMembers.length) continue
    rows.push({ userID: 0, teamID: team.ID, label: team.Name, isTeamRow: true })
    for (const userID of teamMembers) {
      placed.add(userID)
      rows.push({ userID, teamID: team.ID, label: userName(userID), isTeamRow: false })
    }
  }
  for (const userID of memberUserIDs.filter((candidate) => !placed.has(candidate))) {
    rows.push({ userID, teamID: 0, label: userName(userID), isTeamRow: false })
  }
  return rows
}

/** The value of a cell: the draft when edited, otherwise the stored target. */
export const cellValue = (
  draft: Map<string, number>, goals: ICrmGoal[], pipelineID: number, metric: number, userID: number, period: number,
) => draft.get(cellKey(userID, period)) ?? storedTarget(goals, pipelineID, metric, userID, period)

/** Sum of a team's members for one month: what its header row shows (read-only). */
export const teamTotal = (
  row: GoalGridRow, rows: GoalGridRow[], valueOf: (userID: number) => number,
) => rows.filter((candidate) => !candidate.isTeamRow && candidate.teamID === row.teamID)
  .reduce((total, candidate) => total + valueOf(candidate.userID), 0)

/** Draft cells for "fill row": the same value for every period of the year. */
export const fillRow = (draft: Map<string, number>, userID: number, periods: GoalPeriod[], value: number) => {
  const next = new Map(draft)
  for (const { period } of periods) next.set(cellKey(userID, period), value)
  return next
}

/** Draft cells for "copy previous year": last year's stored targets, period by period (week n to week n). */
export const copyPreviousYear = (
  goals: ICrmGoal[], pipelineID: number, metric: number, userIDs: number[], periods: GoalPeriod[], previousPeriods: GoalPeriod[],
) => {
  const next = new Map<string, number>()
  for (const userID of userIDs) {
    periods.forEach(({ period }, index) => {
      if (!previousPeriods[index]) return
      const previous = storedTarget(goals, pipelineID, metric, userID, previousPeriods[index].period)
      if (previous > 0) next.set(cellKey(userID, period), previous)
    })
  }
  return next
}

/** The cells to send: only the ones whose draft differs from the stored value. */
export const changedGoalCells = (
  draft: Map<string, number>, goals: ICrmGoal[], pipelineID: number, metric: number,
): Partial<ICrmGoal>[] => {
  const scale = isMoneyMetric(metric) ? AMOUNT_SCALE : 1
  const changed: Partial<ICrmGoal>[] = []
  for (const [key, value] of draft) {
    const [userID, period] = key.split('|').map(Number)
    if (value === storedTarget(goals, pipelineID, metric, userID, period)) continue
    changed.push({ PipelineID: pipelineID, UserID: userID, Period: period, Metric: metric, Target: Math.max(0, Math.round(value * scale)) })
  }
  return changed
}
