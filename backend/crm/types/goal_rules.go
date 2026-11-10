package types

import (
	"fmt"

	"app/core"
)

// GoalPeriodOfDay is the goal period a UnixDay falls in: its YYMM month, its core.WeekOfDay week,
// or the day itself for the daily types.
func GoalPeriodOfDay(periodType int8, unixDay int16) int16 {
	switch periodType {
	case GoalPeriodWeekly:
		return core.WeekOfDay(unixDay)
	case GoalPeriodDailyByMonth, GoalPeriodDailyByWeek:
		return unixDay
	}
	return core.PeriodOfDay(unixDay)
}

// GoalPeriodDays is the first and last UnixDay of a goal period.
func GoalPeriodDays(periodType int8, period int16) (firstDay, lastDay int16) {
	switch periodType {
	case GoalPeriodWeekly:
		return core.WeekFirstDay(period), core.WeekLastDay(period)
	case GoalPeriodDailyByMonth, GoalPeriodDailyByWeek:
		return period, period
	}
	return core.PeriodFirstDay(period), core.PeriodLastDay(period)
}

// IsValidGoalPeriod: a YYMM month, a week since the epoch (from 2008 to well past 2100), or a
// UnixDay from 2008 on.
func IsValidGoalPeriod(periodType int8, period int16) bool {
	switch periodType {
	case GoalPeriodWeekly:
		return period >= 2000 && period <= 6000
	case GoalPeriodDailyByMonth, GoalPeriodDailyByWeek:
		return period >= 14000
	}
	return period >= 100 && period%100 >= 1 && period%100 <= 12
}

// ValidateGoal checks one cell of the goals grid. Target 0 is valid: it clears the cell.
// goalPeriodTypeByPipelineID holds the period type of every active pipeline; others are unknown.
func ValidateGoal(goal CrmGoal, goalPeriodTypeByPipelineID map[int16]int8, activeUserIDs map[int32]bool) error {
	periodType, isActive := goalPeriodTypeByPipelineID[goal.PipelineID]
	if !isActive {
		return fmt.Errorf("El pipeline %d no existe.", goal.PipelineID)
	}
	if !activeUserIDs[goal.UserID] {
		return fmt.Errorf("El usuario %d no existe o está inactivo.", goal.UserID)
	}
	if goal.Metric < CrmGoalMetricWonAmount || goal.Metric > CrmGoalMetricActivities {
		return fmt.Errorf("La métrica %d no es válida.", goal.Metric)
	}
	if !IsValidGoalPeriod(periodType, goal.Period) {
		return fmt.Errorf("El periodo %d no es válido para este pipeline.", goal.Period)
	}
	if goal.Target < 0 {
		return fmt.Errorf("La meta no puede ser negativa.")
	}
	return nil
}

// GoalKey identifies a goal cell: one per pipeline, seller, month and metric.
type GoalKey struct {
	PipelineID int16
	UserID     int32
	Period     int16
	Metric     int8
}

func (goal CrmGoal) Key() GoalKey {
	return GoalKey{PipelineID: goal.PipelineID, UserID: goal.UserID, Period: goal.Period, Metric: goal.Metric}
}

// PlanGoalSave turns the grid's changed cells into the records to insert and update. A stored
// cell with the same key is updated in place (a zero Target soft-deletes it); a new cell with a
// zero Target is dropped. A key repeated inside the payload is an error.
func PlanGoalSave(payload []CrmGoal, stored []CrmGoal) (inserts, updates []CrmGoal, err error) {
	storedByKey := map[GoalKey]CrmGoal{}
	for _, goal := range stored {
		storedByKey[goal.Key()] = goal
	}
	seenKeys := map[GoalKey]bool{}
	for _, goal := range payload {
		if seenKeys[goal.Key()] {
			return nil, nil, fmt.Errorf("La meta del usuario %d está repetida.", goal.UserID)
		}
		seenKeys[goal.Key()] = true
		existing, exists := storedByKey[goal.Key()]
		switch {
		case exists:
			existing.Target = goal.Target
			existing.Status = CrmStatusActive
			if goal.Target == 0 {
				existing.Status = CrmStatusDeleted
			}
			updates = append(updates, existing)
		case goal.Target > 0:
			goal.ID, goal.Status = 0, CrmStatusActive
			inserts = append(inserts, goal)
		}
	}
	return inserts, updates, nil
}
