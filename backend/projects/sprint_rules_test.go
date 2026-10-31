package projects

import (
	"slices"
	"testing"

	"app/projects/types"
)

func activeStory(storyID int64, finalState int8) types.Story {
	return types.Story{ID: storyID, Title: "Story", State: finalState, Status: types.StatusActive}
}

func TestNormalizeSprint(t *testing.T) {
	projectSprints := []types.Sprint{{ID: 1, Status: types.StatusActive}, {ID: 2, Status: types.StatusDeleted}}
	sprint := types.Sprint{StartDate: 100, EndDate: 113}
	if err := normalizeSprint(&sprint, projectSprints); err != nil {
		t.Fatal(err)
	}
	if sprint.Name != "Sprint 2" {
		t.Errorf("an unnamed sprint is numbered after the existing ones, got %q", sprint.Name)
	}
	if normalizeSprint(&types.Sprint{StartDate: 100, EndDate: 99}, nil) == nil {
		t.Errorf("a sprint cannot end before it starts")
	}
	if normalizeSprint(&types.Sprint{EndDate: 99}, nil) == nil {
		t.Errorf("a sprint needs a start date")
	}
}

func TestPlanSprintStories(t *testing.T) {
	storiesByID := map[int64]types.Story{
		10: activeStory(10, types.StoryStateNotPlanned),
		11: activeStory(11, types.StoryStateInDevelopment),
		12: activeStory(12, types.StoryStateFinished),
		13: activeStory(13, types.StoryStatePending),
	}
	otherOpenSprint := types.Sprint{ID: 2, Name: "Sprint 2", Status: types.StatusActive, State: types.SprintStatePlanned,
		Stories: []types.SprintStory{{StoryID: 13, State: types.StoryStatePending}}}
	sprint := types.Sprint{ID: 1, Status: types.StatusActive, State: types.SprintStateActive,
		Stories: []types.SprintStory{{StoryID: 11, State: types.StoryStateInDevelopment, Rank: 4}, {StoryID: 14, State: types.StoryStatePending, Rank: 2}}}
	projectSprints := []types.Sprint{sprint, otherOpenSprint}

	addedStoryIDs, removedStoryIDs, err := planSprintStories(&sprint, []int64{11, 10}, storiesByID, projectSprints, 500)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(addedStoryIDs, []int64{10}) || !slices.Equal(removedStoryIDs, []int64{14}) {
		t.Errorf("added %v, removed %v", addedStoryIDs, removedStoryIDs)
	}
	addedStory := sprint.Stories[sprintStoryIndex(sprint, 10)]
	if addedStory.State != types.StoryStatePending || addedStory.Rank != 5 || addedStory.AddedAt != 500 {
		t.Errorf("a planned story joins Pending at the end: %+v", addedStory)
	}
	if sprint.Stories[sprintStoryIndex(sprint, 11)].State != types.StoryStateInDevelopment {
		t.Errorf("a story already in the sprint keeps its state")
	}

	if _, _, err := planSprintStories(&sprint, []int64{12}, storiesByID, projectSprints, 500); err == nil {
		t.Errorf("a finished story can't be planned again")
	}
	if _, _, err := planSprintStories(&sprint, []int64{13}, storiesByID, projectSprints, 500); err == nil {
		t.Errorf("a story in another open sprint can't join")
	}
	if _, _, err := planSprintStories(&sprint, []int64{99}, storiesByID, projectSprints, 500); err == nil {
		t.Errorf("a story of another project can't join")
	}
	closedSprint := types.Sprint{Status: types.StatusActive, State: types.SprintStateClosed}
	if _, _, err := planSprintStories(&closedSprint, []int64{10}, storiesByID, projectSprints, 500); err == nil {
		t.Errorf("a closed sprint is read only")
	}
}

func TestMoveSprintStory(t *testing.T) {
	sprint := types.Sprint{Status: types.StatusActive, State: types.SprintStateActive, Stories: []types.SprintStory{
		{StoryID: 10, State: types.StoryStatePending, Rank: 1},
		{StoryID: 11, State: types.StoryStateInDevelopment, Rank: 1},
		{StoryID: 12, State: types.StoryStateInDevelopment, Rank: 2},
	}}
	fromState, err := moveSprintStory(&sprint, 10, types.StoryStateInDevelopment, 11, 12, 500)
	if err != nil || fromState != types.StoryStatePending {
		t.Fatalf("move: %v, from %d", err, fromState)
	}
	movedStory := sprint.Stories[0]
	if movedStory.State != types.StoryStateInDevelopment || movedStory.Rank != 1.5 || movedStory.StartedAt != 500 {
		t.Errorf("the card lands between 11 and 12 and starts its development: %+v", movedStory)
	}
	if _, err := moveSprintStory(&sprint, 10, types.StoryStateFinished, 12, 0, 600); err != nil || sprint.Stories[0].DoneAt != 600 || sprint.Stories[0].StartedAt != 500 {
		t.Errorf("finishing stamps DoneAt and keeps StartedAt: %+v", sprint.Stories[0])
	}
	if _, err := moveSprintStory(&sprint, 10, types.StoryStateInReview, 0, 0, 700); err != nil || sprint.Stories[0].DoneAt != 0 {
		t.Errorf("reopening clears DoneAt: %+v", sprint.Stories[0])
	}
	if _, err := moveSprintStory(&sprint, 99, types.StoryStatePending, 0, 0, 700); err == nil {
		t.Errorf("a story off the board can't move")
	}
	if _, err := moveSprintStory(&sprint, 10, 9, 0, 0, 700); err == nil {
		t.Errorf("an unknown state is refused")
	}
	plannedSprint := types.Sprint{Status: types.StatusActive, State: types.SprintStatePlanned, Stories: []types.SprintStory{{StoryID: 10}}}
	if _, err := moveSprintStory(&plannedSprint, 10, types.StoryStateInDevelopment, 0, 0, 700); err == nil {
		t.Errorf("only the active sprint's board moves")
	}
}

func TestPlanSprintStoryStates(t *testing.T) {
	sprint := types.Sprint{Status: types.StatusActive, State: types.SprintStatePlanned, Stories: []types.SprintStory{
		{StoryID: 10, State: types.StoryStatePending},
		{StoryID: 11, State: types.StoryStateInReview},
	}}
	appliedChanges, err := planSprintStoryStates(&sprint, []SprintStoryStateChange{
		{StoryID: 10, State: types.StoryStateStopped, Reason: " Waiting for the lab "},
		{StoryID: 11, State: types.StoryStateInReview},
	}, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(appliedChanges) != 1 || appliedChanges[0].fromState != types.StoryStatePending || appliedChanges[0].Reason != "Waiting for the lab" {
		t.Errorf("only the story whose state changed is applied, with its trimmed reason: %+v", appliedChanges)
	}
	if sprint.Stories[0].State != types.StoryStateStopped {
		t.Errorf("a planned sprint takes the state: %+v", sprint.Stories[0])
	}
	if _, err := planSprintStoryStates(&sprint, []SprintStoryStateChange{{StoryID: 11, State: types.StoryStateStopped}}, 500); err == nil {
		t.Errorf("Blocked needs a reason")
	}
	if _, err := planSprintStoryStates(&sprint, []SprintStoryStateChange{{StoryID: 99, State: types.StoryStatePending}}, 500); err == nil {
		t.Errorf("a story off the plan can't take a state")
	}
}

func TestNormalizeMoveReason(t *testing.T) {
	if _, err := normalizeMoveReason(types.StoryStateStopped, "  "); err == nil {
		t.Errorf("Stopped needs a reason")
	}
	if reason, err := normalizeMoveReason(types.StoryStateStopped, " Waiting for the lab "); err != nil || reason != "Waiting for the lab" {
		t.Errorf("reason = %q, err = %v", reason, err)
	}
	if _, err := normalizeMoveReason(types.StoryStateInReview, ""); err != nil {
		t.Errorf("other states need no reason")
	}
}

func TestStartSprint(t *testing.T) {
	activeSprint := types.Sprint{ID: 1, Name: "Sprint 1", Status: types.StatusActive, State: types.SprintStateActive}
	plannedSprint := types.Sprint{ID: 2, Status: types.StatusActive, State: types.SprintStatePlanned}
	if startSprint(&plannedSprint, []types.Sprint{activeSprint, plannedSprint}) == nil {
		t.Errorf("only one sprint is active at a time")
	}
	activeSprint.State = types.SprintStateClosed
	if err := startSprint(&plannedSprint, []types.Sprint{activeSprint, plannedSprint}); err != nil || plannedSprint.State != types.SprintStateActive {
		t.Errorf("the planned sprint starts once the other closed: %v", err)
	}
	if startSprint(&plannedSprint, nil) == nil {
		t.Errorf("an active sprint can't start again")
	}
}

func TestCloseSprint(t *testing.T) {
	newClosingSprint := func() types.Sprint {
		return types.Sprint{ID: 1, Status: types.StatusActive, State: types.SprintStateActive, Stories: []types.SprintStory{
			{StoryID: 10, State: types.StoryStateFinished},
			{StoryID: 11, State: types.StoryStateInDevelopment, StartedAt: 300},
			{StoryID: 12, State: types.StoryStateStopped},
		}}
	}
	nextSprint := types.Sprint{ID: 2, Status: types.StatusActive, State: types.SprintStatePlanned, Stories: []types.SprintStory{{StoryID: 20, Rank: 3}}}
	projectSprints := []types.Sprint{newClosingSprint(), nextSprint}

	sprint := newClosingSprint()
	if closeSprint(&sprint, []SprintCarryOver{{StoryID: 11, ToSprintID: 2}}, projectSprints, 7, 900) == nil {
		t.Errorf("every unfinished story needs a destination")
	}
	sprint = newClosingSprint()
	if closeSprint(&sprint, []SprintCarryOver{{StoryID: 10, ToSprintID: 2}, {StoryID: 11}, {StoryID: 12}}, projectSprints, 7, 900) == nil {
		t.Errorf("a finished story is not carried over")
	}
	sprint = newClosingSprint()
	if closeSprint(&sprint, []SprintCarryOver{{StoryID: 11, ToSprintID: 1}, {StoryID: 12}}, projectSprints, 7, 900) == nil {
		t.Errorf("the destination must be a planned sprint")
	}

	sprint = newClosingSprint()
	carryOvers := []SprintCarryOver{{StoryID: 11, ToSprintID: 2}, {StoryID: 12}}
	if err := closeSprint(&sprint, carryOvers, projectSprints, 7, 900); err != nil {
		t.Fatal(err)
	}
	if sprint.State != types.SprintStateClosed || sprint.ClosedAt != 900 || sprint.ClosedBy != 7 || sprint.Stories[1].CarriedToSprintID != 2 {
		t.Errorf("closed sprint = %+v", sprint)
	}
	if sprint.Stories[1].State != types.StoryStateInDevelopment {
		t.Errorf("the closed board keeps the state at close")
	}

	if err := carryStoriesIntoSprint(&nextSprint, newClosingSprint(), []int64{11}, 900); err != nil {
		t.Fatal(err)
	}
	if err := carryStoriesIntoSprint(&nextSprint, newClosingSprint(), []int64{11}, 950); err != nil || len(nextSprint.Stories) != 2 {
		t.Errorf("a retried carry-over adds nothing twice: %+v", nextSprint.Stories)
	}
	carriedStory := nextSprint.Stories[1]
	if carriedStory.State != types.StoryStateInDevelopment || carriedStory.Rank != 4 || carriedStory.AddedAt != 900 || carriedStory.StartedAt != 300 {
		t.Errorf("a carried story keeps its state and goes last: %+v", carriedStory)
	}
}
