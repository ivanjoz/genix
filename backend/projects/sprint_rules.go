package projects

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"app/projects/types"
)

// SprintCarryOver is what happens to one unfinished story when its sprint closes: it goes on in
// another open sprint, or back to the backlog (ToSprintID 0).
type SprintCarryOver struct {
	StoryID    int64
	ToSprintID int64
}

func isSprintOpen(sprint types.Sprint) bool {
	return sprint.Status == types.StatusActive && sprint.State != types.SprintStateClosed
}

func sprintStoryIndex(sprint types.Sprint, storyID int64) int {
	return slices.IndexFunc(sprint.Stories, func(sprintStory types.SprintStory) bool { return sprintStory.StoryID == storyID })
}

// normalizeSprint validates the fields an admin edits. An empty name takes "Sprint N".
func normalizeSprint(sprint *types.Sprint, projectSprints []types.Sprint) error {
	sprint.Name = strings.TrimSpace(sprint.Name)
	sprint.Goal = strings.TrimSpace(sprint.Goal)
	if sprint.Name == "" {
		activeSprintsCount := 0
		for _, projectSprint := range projectSprints {
			if projectSprint.Status == types.StatusActive {
				activeSprintsCount++
			}
		}
		sprint.Name = fmt.Sprintf("Sprint %d", activeSprintsCount+1)
	}
	if len(sprint.Name) > 80 {
		return errors.New("El nombre del sprint puede tener hasta 80 caracteres.")
	}
	if len(sprint.Goal) > 500 {
		return errors.New("El objetivo del sprint puede tener hasta 500 caracteres.")
	}
	if sprint.StartDate <= 0 || sprint.EndDate < sprint.StartDate {
		return errors.New("El sprint necesita una fecha de inicio y una de fin posterior.")
	}
	return nil
}

// planSprintStories makes the open sprint hold exactly storyIDs. A story joins as Pending at the end
// of its column; a finished story, or one already in another open sprint, can't join.
func planSprintStories(sprint *types.Sprint, storyIDs []int64, storiesByID map[int64]types.Story, projectSprints []types.Sprint, now int32) (addedStoryIDs []int64, removedStoryIDs []int64, err error) {
	if !isSprintOpen(*sprint) {
		return nil, nil, errors.New("Un sprint cerrado no se puede modificar.")
	}
	plannedStoryIDs := map[int64]bool{}
	for _, storyID := range storyIDs {
		plannedStoryIDs[storyID] = true
	}

	keptSprintStories := []types.SprintStory{}
	maxRank := 0.0
	for _, sprintStory := range sprint.Stories {
		if !plannedStoryIDs[sprintStory.StoryID] {
			removedStoryIDs = append(removedStoryIDs, sprintStory.StoryID)
			continue
		}
		keptSprintStories = append(keptSprintStories, sprintStory)
		maxRank = max(maxRank, sprintStory.Rank)
	}

	for _, storyID := range storyIDs {
		if sprintStoryIndex(types.Sprint{Stories: keptSprintStories}, storyID) >= 0 {
			continue
		}
		story, storyExists := storiesByID[storyID]
		if !storyExists || story.Status != types.StatusActive {
			return nil, nil, fmt.Errorf("La historia %[1]d no existe en este proyecto.", storyID)
		}
		if story.State == types.StoryStateFinished {
			return nil, nil, fmt.Errorf("La historia %[1]q ya está finalizada.", story.Title)
		}
		for _, projectSprint := range projectSprints {
			if projectSprint.ID != sprint.ID && isSprintOpen(projectSprint) && sprintStoryIndex(projectSprint, storyID) >= 0 {
				return nil, nil, fmt.Errorf("La historia %[1]q ya está en %[2]s.", story.Title, projectSprint.Name)
			}
		}
		maxRank++
		keptSprintStories = append(keptSprintStories, types.SprintStory{StoryID: storyID, State: types.StoryStatePending, Rank: maxRank, AddedAt: now})
		addedStoryIDs = append(addedStoryIDs, storyID)
	}
	sprint.Stories = keptSprintStories
	return addedStoryIDs, removedStoryIDs, nil
}

// moveSprintStory moves a card of the active sprint to a state and between two cards of its column
// (0 = none).
func moveSprintStory(sprint *types.Sprint, storyID int64, toState int8, previousStoryID int64, nextStoryID int64, now int32) (fromState int8, err error) {
	if sprint.State != types.SprintStateActive {
		return 0, errors.New("Solo se puede cambiar el tablero del sprint activo.")
	}
	if toState < types.StoryStatePending || toState > types.StoryStateFinished {
		return 0, errors.New("El estado no es válido.")
	}
	storyIndex := sprintStoryIndex(*sprint, storyID)
	if storyIndex < 0 {
		return 0, errors.New("La historia no está en este sprint.")
	}
	previousIndex, nextIndex := sprintStoryIndex(*sprint, previousStoryID), sprintStoryIndex(*sprint, nextStoryID)
	var previousRank, nextRank float64
	if previousIndex >= 0 {
		previousRank = sprint.Stories[previousIndex].Rank
	}
	if nextIndex >= 0 {
		nextRank = sprint.Stories[nextIndex].Rank
	}

	sprintStory := &sprint.Stories[storyIndex]
	fromState = sprintStory.State
	sprintStory.Rank = rankBetween(previousRank, previousIndex >= 0 && previousIndex != storyIndex, nextRank, nextIndex >= 0 && nextIndex != storyIndex)
	setSprintStoryState(sprintStory, toState, now)
	return fromState, nil
}

// setSprintStoryState: StartedAt keeps the first time the story went into development; DoneAt the
// last time it finished, cleared when it is reopened.
func setSprintStoryState(sprintStory *types.SprintStory, toState int8, now int32) {
	sprintStory.State = toState
	if toState == types.StoryStateInDevelopment && sprintStory.StartedAt == 0 {
		sprintStory.StartedAt = now
	}
	sprintStory.DoneAt = 0
	if toState == types.StoryStateFinished {
		sprintStory.DoneAt = now
	}
}

// SprintStoryStateChange is a state set while planning: a story dropped on a state's group.
type SprintStoryStateChange struct {
	StoryID int64
	State   int8
	Reason  string
	// The state it had in the sprint, filled by planSprintStoryStates for the history line.
	fromState int8
}

// planSprintStoryStates sets the states chosen while planning on the sprint's stories, run after
// planSprintStories so a story just added can take one. Blocked needs its reason, like on the board.
// It returns the changes that did change a state.
func planSprintStoryStates(sprint *types.Sprint, stateChanges []SprintStoryStateChange, now int32) ([]SprintStoryStateChange, error) {
	appliedChanges := []SprintStoryStateChange{}
	for _, stateChange := range stateChanges {
		if stateChange.State < types.StoryStatePending || stateChange.State > types.StoryStateFinished {
			return nil, errors.New("El estado no es válido.")
		}
		storyIndex := sprintStoryIndex(*sprint, stateChange.StoryID)
		if storyIndex < 0 {
			return nil, errors.New("La historia no está en este sprint.")
		}
		sprintStory := &sprint.Stories[storyIndex]
		if sprintStory.State == stateChange.State {
			continue
		}
		reason, err := normalizeMoveReason(stateChange.State, stateChange.Reason)
		if err != nil {
			return nil, err
		}
		stateChange.Reason, stateChange.fromState = reason, sprintStory.State
		setSprintStoryState(sprintStory, stateChange.State, now)
		appliedChanges = append(appliedChanges, stateChange)
	}
	return appliedChanges, nil
}

// normalizeMoveReason: moving a card to Stopped needs the reason, kept in the story's history.
func normalizeMoveReason(toState int8, reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if toState == types.StoryStateStopped && reason == "" {
		return "", errors.New("Indica por qué se detiene la historia.")
	}
	if len(reason) > 1000 {
		return "", errors.New("El motivo puede tener hasta 1000 caracteres.")
	}
	return reason, nil
}

// startSprint makes a planned sprint the project's active one. Only one is active at a time.
func startSprint(sprint *types.Sprint, projectSprints []types.Sprint) error {
	if sprint.State != types.SprintStatePlanned {
		return errors.New("Solo un sprint planificado puede iniciarse.")
	}
	for _, projectSprint := range projectSprints {
		if projectSprint.ID != sprint.ID && projectSprint.Status == types.StatusActive && projectSprint.State == types.SprintStateActive {
			return fmt.Errorf("%[1]s sigue activo: ciérralo primero.", projectSprint.Name)
		}
	}
	sprint.State = types.SprintStateActive
	return nil
}

// closeSprint closes the active sprint. Every unfinished story needs exactly one carry-over, to a
// planned sprint of the project or back to the backlog. The unfinished stories stay on this board
// with their state at close: it shows them "Not completed".
func closeSprint(sprint *types.Sprint, carryOvers []SprintCarryOver, projectSprints []types.Sprint, userID int32, now int32) error {
	if sprint.State != types.SprintStateActive {
		return errors.New("Solo se puede cerrar el sprint activo.")
	}
	carryOverByStoryID := map[int64]SprintCarryOver{}
	for _, carryOver := range carryOvers {
		if _, isRepeated := carryOverByStoryID[carryOver.StoryID]; isRepeated {
			return errors.New("Una historia tiene más de un destino.")
		}
		storyIndex := sprintStoryIndex(*sprint, carryOver.StoryID)
		if storyIndex < 0 || sprint.Stories[storyIndex].State == types.StoryStateFinished {
			return errors.New("Solo se trasladan las historias no culminadas del sprint.")
		}
		if carryOver.ToSprintID != 0 {
			isPlannedSprint := slices.ContainsFunc(projectSprints, func(projectSprint types.Sprint) bool {
				return projectSprint.ID == carryOver.ToSprintID && projectSprint.Status == types.StatusActive && projectSprint.State == types.SprintStatePlanned
			})
			if !isPlannedSprint {
				return errors.New("Las historias no culminadas solo pueden pasar a un sprint planificado.")
			}
		}
		carryOverByStoryID[carryOver.StoryID] = carryOver
	}
	for storyIndex, sprintStory := range sprint.Stories {
		if sprintStory.State == types.StoryStateFinished {
			continue
		}
		carryOver, isDecided := carryOverByStoryID[sprintStory.StoryID]
		if !isDecided {
			return errors.New("Elige a dónde va cada historia no culminada.")
		}
		sprint.Stories[storyIndex].CarriedToSprintID = carryOver.ToSprintID
	}
	sprint.State, sprint.ClosedAt, sprint.ClosedBy = types.SprintStateClosed, now, userID
	return nil
}

// carryStoriesIntoSprint adds the closing sprint's unfinished stories to the planned sprint they
// go on in, keeping their state. A story already there is skipped, so a retried close is harmless.
func carryStoriesIntoSprint(targetSprint *types.Sprint, closingSprint types.Sprint, storyIDs []int64, now int32) error {
	if targetSprint.Status != types.StatusActive || targetSprint.State != types.SprintStatePlanned {
		return errors.New("Las historias no culminadas solo pueden pasar a un sprint planificado.")
	}
	maxRank := 0.0
	for _, sprintStory := range targetSprint.Stories {
		maxRank = max(maxRank, sprintStory.Rank)
	}
	for _, storyID := range storyIDs {
		closingIndex := sprintStoryIndex(closingSprint, storyID)
		if closingIndex < 0 || sprintStoryIndex(*targetSprint, storyID) >= 0 {
			continue
		}
		maxRank++
		closingSprintStory := closingSprint.Stories[closingIndex]
		targetSprint.Stories = append(targetSprint.Stories, types.SprintStory{
			StoryID: storyID, State: closingSprintStory.State, Rank: maxRank, AddedAt: now, StartedAt: closingSprintStory.StartedAt,
		})
	}
	return nil
}
