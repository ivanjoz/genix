package types

import (
	"fmt"

	"app/core"
)

// ComputedForecast is the forecast of a set of sellers for one month, in opportunity amounts.
type ComputedForecast struct {
	Commit   int32
	BestCase int32
	Pipeline int32
	// Closed is what was already won in the month.
	Closed int32
}

// ComputeForecast sums, for the opportunities of the owners (all when ownerIDs is empty): the open
// ones expected to close in the goal period (the pipeline's month or week) by the forecast
// category of their stage, and the ones already won in that period.
func ComputeForecast(opportunities []CrmOpportunity, pipeline CrmPipeline, period int16, ownerIDs map[int32]bool) ComputedForecast {
	periodType := pipeline.GoalPeriodType
	categoryByStageID := map[int16]int8{}
	for _, stage := range pipeline.Stages {
		categoryByStageID[stage.ID] = stage.ForecastCategory
	}
	forecast := ComputedForecast{}
	for _, opportunity := range opportunities {
		if opportunity.PipelineID != pipeline.ID || (len(ownerIDs) > 0 && !ownerIDs[opportunity.OwnerUserID]) {
			continue
		}
		switch opportunity.Outcome {
		case OpportunityOutcomeWon:
			if GoalPeriodOfDay(periodType, core.SUnixTimeToUnixDay(opportunity.ClosedTime)) == period {
				forecast.Closed += opportunity.Amount
			}
		case OpportunityOutcomeOpen:
			if GoalPeriodOfDay(periodType, opportunity.ExpectedCloseDate) != period {
				continue
			}
			switch categoryByStageID[opportunity.StageID] {
			case ForecastCategoryCommit:
				forecast.Commit += opportunity.Amount
			case ForecastCategoryBestCase:
				forecast.BestCase += opportunity.Amount
			default:
				forecast.Pipeline += opportunity.Amount
			}
		}
	}
	return forecast
}

// PlanForecastSubmission validates the payload and builds the row to append. Declared amounts
// default to the computed ones. A team submission needs the caller to lead that team; an
// individual one needs the caller to be a member of the pipeline.
func PlanForecastSubmission(payload CrmForecastSubmissionPayload, periodType int8, computed ComputedForecast,
	callerID int32, callerLevel int8, team CrmTeam, now int32) (CrmForecastSubmission, error) {

	if !IsValidGoalPeriod(periodType, payload.Period) {
		return CrmForecastSubmission{}, fmt.Errorf("El periodo %d no es válido para este pipeline.", payload.Period)
	}
	if payload.CommitAmount < 0 || payload.BestCaseAmount < 0 {
		return CrmForecastSubmission{}, fmt.Errorf("Los montos del forecast no pueden ser negativos.")
	}
	if payload.TeamID != 0 && (team.ID != payload.TeamID || team.Status != CrmStatusActive || team.LeaderUserID != callerID) {
		return CrmForecastSubmission{}, fmt.Errorf("Solo el líder del equipo puede enviar el forecast del equipo.")
	}
	if callerLevel < PipelineLevelView {
		return CrmForecastSubmission{}, fmt.Errorf("No tiene acceso a este pipeline.")
	}
	submission := CrmForecastSubmission{
		PipelineID: payload.PipelineID, UserID: callerID, TeamID: payload.TeamID, Period: payload.Period,
		ComputedCommit: computed.Commit, ComputedBestCase: computed.BestCase,
		ComputedPipeline: computed.Pipeline, ComputedClosed: computed.Closed,
		CommitAmount: payload.CommitAmount, BestCaseAmount: payload.BestCaseAmount,
		Note: payload.Note, Created: now, Status: CrmStatusActive, UpdatedBy: callerID,
	}
	if payload.CommitAmount == 0 && payload.BestCaseAmount == 0 {
		submission.CommitAmount, submission.BestCaseAmount = computed.Commit, computed.BestCase
	}
	return submission, nil
}
