// Plain module so pure files can import it without the Svelte runtime. Mirrors types.CrmGoalMetric*.
export const GOAL_METRIC = { WON_AMOUNT: 1, WON_COUNT: 2, NEW_OPPORTUNITIES: 3, ACTIVITIES: 4 } as const

/** Mirrors types.GoalPeriod*: the unit a pipeline sets goals and forecasts in: month (YYMM), week, or day (a UnixDay;
 * the grid then shows a month or a week of days at a time). */
export const GOAL_PERIOD = { MONTHLY: 1, WEEKLY: 2, DAILY_BY_MONTH: 3, DAILY_BY_WEEK: 4 } as const
