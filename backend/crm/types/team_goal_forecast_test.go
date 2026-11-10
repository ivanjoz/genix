package types

import (
	"testing"
	"time"

	"app/core"
)

func TestApplyTeamEdit(t *testing.T) {
	active := map[int32]bool{1: true, 2: true, 3: true}
	others := []CrmTeam{{ID: 9, Name: "North", MemberUserIDs: []int32{3}, Status: CrmStatusActive}}

	saved, err := ApplyTeamEdit(CrmTeam{Name: " South ", LeaderUserID: 1, MemberUserIDs: []int32{1, 2}}, active, others)
	if err != nil || saved.Name != "South" || saved.Status != CrmStatusActive {
		t.Fatalf("valid team rejected: %v %+v", err, saved)
	}

	rejected := map[string]CrmTeam{
		"no name":          {LeaderUserID: 1, MemberUserIDs: []int32{1}},
		"no members":       {Name: "X", LeaderUserID: 1},
		"leader outside":   {Name: "X", LeaderUserID: 2, MemberUserIDs: []int32{1}},
		"duplicate member": {Name: "X", LeaderUserID: 1, MemberUserIDs: []int32{1, 1}},
		"inactive member":  {Name: "X", LeaderUserID: 7, MemberUserIDs: []int32{7}},
		"in another team":  {Name: "X", LeaderUserID: 3, MemberUserIDs: []int32{3}},
		"same name":        {Name: "North", LeaderUserID: 1, MemberUserIDs: []int32{1}},
	}
	for name, team := range rejected {
		if _, err := ApplyTeamEdit(team, active, others); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}

	// Editing a team does not collide with itself.
	if _, err := ApplyTeamEdit(CrmTeam{ID: 9, Name: "North", LeaderUserID: 3, MemberUserIDs: []int32{3}}, active, others); err != nil {
		t.Errorf("a team collided with itself: %v", err)
	}
}

func TestPlanGoalSave(t *testing.T) {
	stored := []CrmGoal{{ID: 5, PipelineID: 1, UserID: 2, Period: 2601, Metric: CrmGoalMetricWonAmount, Target: 100, Status: CrmStatusActive}}
	payload := []CrmGoal{
		{PipelineID: 1, UserID: 2, Period: 2601, Metric: CrmGoalMetricWonAmount, Target: 0},  // clears the stored cell
		{PipelineID: 1, UserID: 2, Period: 2602, Metric: CrmGoalMetricWonAmount, Target: 50}, // new
		{PipelineID: 1, UserID: 2, Period: 2603, Metric: CrmGoalMetricWonAmount, Target: 0},  // new and empty: dropped
	}
	inserts, updates, err := PlanGoalSave(payload, stored)
	if err != nil || len(inserts) != 1 || len(updates) != 1 {
		t.Fatalf("unexpected plan: %v inserts=%d updates=%d", err, len(inserts), len(updates))
	}
	if updates[0].ID != 5 || updates[0].Status != CrmStatusDeleted {
		t.Errorf("the cleared cell should soft-delete the stored goal: %+v", updates[0])
	}
	if _, _, err := PlanGoalSave([]CrmGoal{payload[1], payload[1]}, nil); err == nil {
		t.Error("a repeated key must be rejected")
	}
}

func TestComputeForecast(t *testing.T) {
	pipeline := CrmPipeline{ID: 1, Stages: []CrmPipelineStage{
		{ID: 1, ForecastCategory: ForecastCategoryPipeline}, {ID: 2, ForecastCategory: ForecastCategoryBestCase}, {ID: 3, ForecastCategory: ForecastCategoryCommit},
	}}
	// Expected close in January 2026 (period 2601) and a day in February.
	january := core.PeriodFirstDay(2601) + 10
	february := core.PeriodFirstDay(2602) + 3
	wonTime := int32((int64(january)*86400 - 1e9) / 2)
	opportunities := []CrmOpportunity{
		{PipelineID: 1, OwnerUserID: 1, StageID: 1, Amount: 10, ExpectedCloseDate: january},
		{PipelineID: 1, OwnerUserID: 1, StageID: 2, Amount: 20, ExpectedCloseDate: january},
		{PipelineID: 1, OwnerUserID: 1, StageID: 3, Amount: 30, ExpectedCloseDate: january},
		{PipelineID: 1, OwnerUserID: 1, StageID: 3, Amount: 99, ExpectedCloseDate: february}, // other month
		{PipelineID: 1, OwnerUserID: 2, StageID: 3, Amount: 7, ExpectedCloseDate: january},   // other owner
		{PipelineID: 1, OwnerUserID: 1, Outcome: OpportunityOutcomeWon, Amount: 40, ClosedTime: wonTime},
		{PipelineID: 2, OwnerUserID: 1, StageID: 3, Amount: 500, ExpectedCloseDate: january}, // other pipeline
	}
	got := ComputeForecast(opportunities, pipeline, 2601, map[int32]bool{1: true})
	want := ComputedForecast{Commit: 30, BestCase: 20, Pipeline: 10, Closed: 40}
	if got != want {
		t.Errorf("forecast = %+v, want %+v", got, want)
	}
}

func TestPlanForecastSubmission(t *testing.T) {
	computed := ComputedForecast{Commit: 30, BestCase: 20}
	submission, err := PlanForecastSubmission(CrmForecastSubmissionPayload{PipelineID: 1, Period: 2601}, GoalPeriodMonthly, computed, 4, PipelineLevelEdit, CrmTeam{}, 100)
	if err != nil || submission.CommitAmount != 30 || submission.BestCaseAmount != 20 || submission.UserID != 4 {
		t.Fatalf("defaults to the computed amounts: %v %+v", err, submission)
	}
	team := CrmTeam{ID: 2, LeaderUserID: 4, Status: CrmStatusActive}
	if _, err := PlanForecastSubmission(CrmForecastSubmissionPayload{PipelineID: 1, Period: 2601, TeamID: 2}, GoalPeriodMonthly, computed, 5, PipelineLevelEdit, team, 100); err == nil {
		t.Error("only the leader may submit for the team")
	}
	if _, err := PlanForecastSubmission(CrmForecastSubmissionPayload{PipelineID: 1, Period: 2601}, GoalPeriodMonthly, computed, 4, 0, CrmTeam{}, 100); err == nil {
		t.Error("a non-member must be refused")
	}
	if _, err := PlanForecastSubmission(CrmForecastSubmissionPayload{PipelineID: 1, Period: 2613}, GoalPeriodMonthly, computed, 4, PipelineLevelEdit, CrmTeam{}, 100); err == nil {
		t.Error("an invalid period must be refused")
	}
}

func TestWeeklyGoalPeriods(t *testing.T) {
	// 2026-11-09 is a Monday; its week runs to Sunday the 15th.
	monday := int16(time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC).Unix() / 86400)
	week := GoalPeriodOfDay(GoalPeriodWeekly, monday)
	if GoalPeriodOfDay(GoalPeriodWeekly, monday+6) != week || GoalPeriodOfDay(GoalPeriodWeekly, monday+7) != week+1 || GoalPeriodOfDay(GoalPeriodWeekly, monday-1) != week-1 {
		t.Fatalf("a week must run Monday to Sunday: %d", week)
	}
	if first, last := GoalPeriodDays(GoalPeriodWeekly, week); first != monday || last != monday+6 {
		t.Errorf("week days = %d..%d, want %d..%d", first, last, monday, monday+6)
	}
	if IsValidGoalPeriod(GoalPeriodWeekly, 2610) != true || IsValidGoalPeriod(GoalPeriodMonthly, 2613) || !IsValidGoalPeriod(GoalPeriodMonthly, 2612) {
		t.Error("period validity depends on the period type")
	}

	// A weekly pipeline forecasts the opportunities expected inside the week.
	pipeline := CrmPipeline{ID: 1, GoalPeriodType: GoalPeriodWeekly, Stages: []CrmPipelineStage{{ID: 1, ForecastCategory: ForecastCategoryCommit}}}
	opportunities := []CrmOpportunity{
		{PipelineID: 1, StageID: 1, Amount: 5, ExpectedCloseDate: monday + 3},
		{PipelineID: 1, StageID: 1, Amount: 9, ExpectedCloseDate: monday + 8},
	}
	if got := ComputeForecast(opportunities, pipeline, week, nil); got.Commit != 5 {
		t.Errorf("weekly commit = %d, want 5", got.Commit)
	}
	goals := map[int16]int8{1: GoalPeriodWeekly}
	if err := ValidateGoal(CrmGoal{PipelineID: 1, UserID: 1, Period: 2610, Metric: 1}, goals, map[int32]bool{1: true}); err != nil {
		t.Errorf("a week is a valid period of a weekly pipeline: %v", err)
	}
	if err := ValidateGoal(CrmGoal{PipelineID: 1, UserID: 1, Period: 2601, Metric: 1}, map[int16]int8{1: GoalPeriodMonthly}, map[int32]bool{1: true}); err != nil {
		t.Errorf("YYMM is valid for monthly: %v", err)
	}
}

func TestDailyGoalPeriods(t *testing.T) {
	day := int16(time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC).Unix() / 86400)
	for _, periodType := range []int8{GoalPeriodDailyByMonth, GoalPeriodDailyByWeek} {
		if GoalPeriodOfDay(periodType, day) != day {
			t.Errorf("type %d: a daily period is the day itself", periodType)
		}
		if first, last := GoalPeriodDays(periodType, day); first != day || last != day {
			t.Errorf("type %d: days = %d..%d", periodType, first, last)
		}
		if !IsValidGoalPeriod(periodType, day) || IsValidGoalPeriod(periodType, 2611) {
			t.Errorf("type %d: validity", periodType)
		}
	}
	pipeline := CrmPipeline{ID: 1, GoalPeriodType: GoalPeriodDailyByWeek, Stages: []CrmPipelineStage{{ID: 1, ForecastCategory: ForecastCategoryCommit}}}
	opportunities := []CrmOpportunity{
		{PipelineID: 1, StageID: 1, Amount: 5, ExpectedCloseDate: day},
		{PipelineID: 1, StageID: 1, Amount: 9, ExpectedCloseDate: day + 1},
	}
	if got := ComputeForecast(opportunities, pipeline, day, nil); got.Commit != 5 {
		t.Errorf("daily commit = %d, want 5", got.Commit)
	}
}
