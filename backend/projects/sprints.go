package projects

import (
	"encoding/json"
	"slices"

	"app/core"
	"app/db"
	"app/projects/types"
)

// Every sprint write runs in one transaction under the project's row lock (lockProjectWriteIn): the
// rules read the project's sprints and stories, decide, and write, with no other writer in between.
// Those reads see committed rows only, so a body that writes a sprint and then needs the project's
// sprints passes them along with its write applied (withWrittenSprints) instead of reading again.

// GetSprints is the delta of a project's sprints, with their stories' states.
func GetSprints(req *core.HandlerArgs) core.HandlerResponse {
	sprints := []types.Sprint{}
	query := db.Query(&sprints)
	query.CompanyID.Equals(req.User.CompanyID).ProjectID.Equals(req.GetQueryInt("project-id")).
		Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los sprints:", err)
	}
	return core.MakeResponse(req, &sprints)
}

// updateSprintsIn writes every field a sprint writer may change. ProjectID and Status go with any
// update: they are the delta index's keys.
func updateSprintsIn(transaction *db.Transaction, sprints []types.Sprint) error {
	sprintTable := db.TableOf[types.Sprint]()
	return db.UpdateIn(transaction, &sprints, sprintTable.ProjectID, sprintTable.Status, sprintTable.Name, sprintTable.Goal,
		sprintTable.StartDate, sprintTable.EndDate, sprintTable.State, sprintTable.Stories,
		sprintTable.ClosedAt, sprintTable.ClosedBy, sprintTable.UpdatedBy)
}

// updateStoriesStateIn writes a story's final state and the sprints that hold it.
func updateStoriesStateIn(transaction *db.Transaction, stories []types.Story) error {
	storyTable := db.TableOf[types.Story]()
	return db.UpdateIn(transaction, &stories, storyTable.ProjectID, storyTable.Status, storyTable.State, storyTable.StateDates,
		storyTable.LastSprintID, storyTable.SprintIDs, storyTable.UpdatedBy)
}

// withWrittenSprints is the project's active sprints once writtenSprints are applied: each replaces
// the stored one of its ID, and a deleted one drops out.
func withWrittenSprints(projectSprints []types.Sprint, writtenSprints ...types.Sprint) []types.Sprint {
	currentSprints := slices.Clone(projectSprints)
	for _, writtenSprint := range writtenSprints {
		currentSprints = slices.DeleteFunc(currentSprints, func(sprint types.Sprint) bool { return sprint.ID == writtenSprint.ID })
		if writtenSprint.Status == types.StatusActive {
			currentSprints = append(currentSprints, writtenSprint)
		}
	}
	return currentSprints
}

// syncStoriesFinalStateIn rewrites the final state of the stories a sprint change touched, from
// currentSprints: the project's sprints with the caller's own sprint writes applied. Every writer of
// a sprint calls it after its write, and the activities it writes are the change's history. The
// history entries of a story that is gone are dropped.
func syncStoriesFinalStateIn(transaction *db.Transaction, companyID int32, projectID int32, currentSprints []types.Sprint, storyIDs []int64, userID int32, activities []types.StoryActivity) error {
	if len(storyIDs) == 0 {
		return nil
	}
	projectStories, err := loadProjectStories(companyID, projectID)
	if err != nil {
		return err
	}
	storiesByID := map[int64]types.Story{}
	for _, story := range projectStories {
		storiesByID[story.ID] = story
	}
	today := core.FechaUnix()
	syncedStories := []types.Story{}
	for _, storyID := range storyIDs {
		story, storyExists := storiesByID[storyID]
		if !storyExists {
			activities = slices.DeleteFunc(activities, func(activity types.StoryActivity) bool { return activity.StoryID == storyID })
			continue
		}
		finalState, lastSprintID, sprintIDs := storyFinalState(storyID, currentSprints)
		setStoryState(&story, finalState, today)
		story.LastSprintID, story.SprintIDs, story.UpdatedBy = lastSprintID, sprintIDs, userID
		syncedStories = append(syncedStories, story)
	}
	if err := updateStoriesStateIn(transaction, syncedStories); err != nil {
		return err
	}
	return addStoryActivitiesIn(transaction, activities...)
}

type storyAssignee struct {
	StoryID    int64
	AssigneeID int32
}

type sprintSaveRequest struct {
	Sprint types.Sprint
	// StoryIDs is the sprint's whole plan; nil leaves the stories as they are.
	StoryIDs *[]int64
	// StoryAssignees are the assignees set while planning (a story dropped on a person).
	StoryAssignees []storyAssignee
	// StoryStates are the states set while planning (a story dropped on a state's group).
	StoryStates []SprintStoryStateChange
}

// PostSprint is the sprint's edit mode (admins): create or edit it with its plan, or delete it
// while it is still planned.
func PostSprint(req *core.HandlerArgs) core.HandlerResponse {
	request := sprintSaveRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	var savedSprint types.Sprint
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var err error
		savedSprint, err = saveSprintIn(transaction, req, request)
		return err
	})
	if err != nil {
		return makeProjectErr(req, err)
	}
	return req.MakeResponse(savedSprint)
}

// saveSprintIn creates, edits or deletes a sprint with its plan under the project's row lock.
func saveSprintIn(transaction *db.Transaction, req *core.HandlerArgs, request sprintSaveRequest) (types.Sprint, error) {
	sprint := request.Sprint
	isDeletion := sprint.ID != 0 && sprint.Status == types.StatusDeleted
	project, err := lockProjectWriteIn(transaction, req, sprint.ProjectID, types.ProjectRoleAdmin)
	if err != nil {
		return types.Sprint{}, err
	}
	companyID := req.User.CompanyID

	projectSprints, err := loadProjectSprints(companyID, project.ID)
	if err != nil {
		return types.Sprint{}, core.Err("Error al obtener los sprints:", err)
	}
	if !isDeletion {
		if err := normalizeSprint(&sprint, projectSprints); err != nil {
			return types.Sprint{}, err
		}
	}
	storiesByID := map[int64]types.Story{}
	if request.StoryIDs != nil || len(request.StoryAssignees) > 0 {
		projectStories, err := loadProjectStories(companyID, project.ID)
		if err != nil {
			return types.Sprint{}, core.Err("Error al obtener las historias:", err)
		}
		for _, story := range projectStories {
			storiesByID[story.ID] = story
		}
	}
	for _, assignee := range request.StoryAssignees {
		if _, storyExists := storiesByID[assignee.StoryID]; !storyExists || (assignee.AssigneeID != 0 && !isProjectMember(project, assignee.AssigneeID)) {
			return types.Sprint{}, core.Err("El responsable debe ser parte del equipo del proyecto.")
		}
	}
	now := core.SUnixTime()

	var addedStoryIDs, removedStoryIDs []int64
	var stateChanges []SprintStoryStateChange
	if sprint.ID == 0 {
		sprint.CompanyID, sprint.ProjectID = companyID, project.ID
		sprint.State, sprint.Stories, sprint.Status, sprint.UpdatedBy = types.SprintStatePlanned, nil, types.StatusActive, req.User.ID
		if request.StoryIDs != nil {
			if addedStoryIDs, removedStoryIDs, err = planSprintStories(&sprint, *request.StoryIDs, storiesByID, projectSprints, now); err != nil {
				return types.Sprint{}, err
			}
		}
		if stateChanges, err = planSprintStoryStates(&sprint, request.StoryStates, now); err != nil {
			return types.Sprint{}, err
		}
		newSprints := []types.Sprint{sprint}
		if err := db.InsertIn(transaction, &newSprints); err != nil {
			return types.Sprint{}, core.Err("Error al guardar el sprint:", err)
		}
		sprint = newSprints[0]
	} else {
		storedIndex := slices.IndexFunc(projectSprints, func(projectSprint types.Sprint) bool { return projectSprint.ID == sprint.ID })
		if storedIndex < 0 {
			return types.Sprint{}, core.Err("El sprint no existe.")
		}
		stored := projectSprints[storedIndex]
		// The rules change the sprint's stories in place: the copy must not share them with
		// projectSprints, which stays the stored state until withWrittenSprints applies this write.
		stored.Stories = slices.Clone(stored.Stories)
		if !isSprintOpen(stored) {
			return types.Sprint{}, core.Err("Un sprint cerrado no se puede modificar.")
		}
		stored.UpdatedBy = req.User.ID
		if isDeletion {
			if stored.State != types.SprintStatePlanned {
				return types.Sprint{}, core.Err("Solo se puede eliminar un sprint planificado.")
			}
			for _, sprintStory := range stored.Stories {
				removedStoryIDs = append(removedStoryIDs, sprintStory.StoryID)
			}
			stored.Status = types.StatusDeleted
		} else {
			stored.Name, stored.Goal, stored.StartDate, stored.EndDate = sprint.Name, sprint.Goal, sprint.StartDate, sprint.EndDate
			if request.StoryIDs != nil {
				if addedStoryIDs, removedStoryIDs, err = planSprintStories(&stored, *request.StoryIDs, storiesByID, projectSprints, now); err != nil {
					return types.Sprint{}, err
				}
			}
			if stateChanges, err = planSprintStoryStates(&stored, request.StoryStates, now); err != nil {
				return types.Sprint{}, err
			}
		}
		if err := updateSprintsIn(transaction, []types.Sprint{stored}); err != nil {
			return types.Sprint{}, core.Err("Error al guardar el sprint:", err)
		}
		sprint = stored
	}

	sprintActivity := func(storyID int64, kind int8) types.StoryActivity {
		return types.StoryActivity{
			CompanyID: companyID, ProjectID: project.ID, StoryID: storyID, Kind: kind, SprintID: sprint.ID, AuthorID: req.User.ID, Created: now,
		}
	}
	sprintActivities := []types.StoryActivity{}
	for _, storyID := range addedStoryIDs {
		sprintActivities = append(sprintActivities, sprintActivity(storyID, types.ActivityKindAddedToSprint))
	}
	for _, storyID := range removedStoryIDs {
		sprintActivities = append(sprintActivities, sprintActivity(storyID, types.ActivityKindRemovedFromSprint))
	}
	touchedStoryIDs := slices.Concat(addedStoryIDs, removedStoryIDs)
	for _, stateChange := range stateChanges {
		stateActivity := sprintActivity(stateChange.StoryID, types.ActivityKindStateChange)
		stateActivity.FromState, stateActivity.ToState, stateActivity.Text = stateChange.fromState, stateChange.State, stateChange.Reason
		sprintActivities = append(sprintActivities, stateActivity)
		touchedStoryIDs = append(touchedStoryIDs, stateChange.StoryID)
	}
	// A story added and given a state is synced once.
	slices.Sort(touchedStoryIDs)
	if err := syncStoriesFinalStateIn(transaction, companyID, project.ID, withWrittenSprints(projectSprints, sprint),
		slices.Compact(touchedStoryIDs), req.User.ID, sprintActivities); err != nil {
		return types.Sprint{}, core.Err("Error al guardar el estado de las historias del sprint:", err)
	}

	assigneeActivities := []types.StoryActivity{}
	reassignedStories := []types.Story{}
	for _, assignee := range request.StoryAssignees {
		story := storiesByID[assignee.StoryID]
		if story.AssigneeID == assignee.AssigneeID {
			continue
		}
		story.AssigneeID, story.UpdatedBy = assignee.AssigneeID, req.User.ID
		reassignedStories = append(reassignedStories, story)
		assigneeActivity := sprintActivity(assignee.StoryID, types.ActivityKindAssigneeChange)
		assigneeActivity.SprintID, assigneeActivity.AssigneeID = 0, assignee.AssigneeID
		assigneeActivities = append(assigneeActivities, assigneeActivity)
	}
	if len(reassignedStories) > 0 {
		storyTable := db.TableOf[types.Story]()
		if err := db.UpdateIn(transaction, &reassignedStories, storyTable.ProjectID, storyTable.Status, storyTable.AssigneeID, storyTable.UpdatedBy); err != nil {
			return types.Sprint{}, core.Err("Error al guardar los responsables del sprint:", err)
		}
		if err := saveStoryAssigneesIn(transaction, companyID, false, reassignedStories...); err != nil {
			return types.Sprint{}, core.Err("Error al guardar la lista de los responsables:", err)
		}
		if err := addStoryActivitiesIn(transaction, assigneeActivities...); err != nil {
			return types.Sprint{}, core.Err("Error al guardar la actividad de las historias:", err)
		}
	}
	return sprint, nil
}

type sprintActionRequest struct {
	ProjectID int32
	SprintID  int64
}

// PostSprintStart makes a planned sprint the active one (admins).
func PostSprintStart(req *core.HandlerArgs) core.HandlerResponse {
	request := sprintActionRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	var startedSprint types.Sprint
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		if _, err := lockProjectWriteIn(transaction, req, request.ProjectID, types.ProjectRoleAdmin); err != nil {
			return err
		}
		projectSprints, err := loadProjectSprints(req.User.CompanyID, request.ProjectID)
		if err != nil {
			return core.Err("Error al obtener los sprints:", err)
		}
		sprintIndex := slices.IndexFunc(projectSprints, func(sprint types.Sprint) bool { return sprint.ID == request.SprintID })
		if sprintIndex < 0 {
			return core.Err("El sprint no existe.")
		}
		startedSprint = projectSprints[sprintIndex]
		if err := startSprint(&startedSprint, projectSprints); err != nil {
			return err
		}
		startedSprint.UpdatedBy = req.User.ID
		if err := updateSprintsIn(transaction, []types.Sprint{startedSprint}); err != nil {
			return core.Err("Error al iniciar el sprint:", err)
		}
		return nil
	})
	if err != nil {
		return makeProjectErr(req, err)
	}
	return req.MakeResponse(startedSprint)
}

type sprintMoveRequest struct {
	ProjectID       int32
	SprintID        int64
	StoryID         int64
	State           int8
	PreviousStoryID int64
	NextStoryID     int64
	Reason          string
}

// PostSprintMove moves a card of the active sprint's board (team members): to another column, and
// between two cards of it. Moves are serialized by the project's row lock, so two people moving
// cards at once both land, one after the other.
func PostSprintMove(req *core.HandlerArgs) core.HandlerResponse {
	request := sprintMoveRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	reason, err := normalizeMoveReason(request.State, request.Reason)
	if err != nil {
		return req.MakeErr(err)
	}
	var movedSprint types.Sprint
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		var err error
		movedSprint, err = moveSprintCardIn(transaction, req, request, reason)
		return err
	})
	if err != nil {
		return makeProjectErr(req, err)
	}
	return req.MakeResponse(movedSprint)
}

// moveSprintCardIn moves a card of a sprint's board under the project's row lock.
func moveSprintCardIn(transaction *db.Transaction, req *core.HandlerArgs, request sprintMoveRequest, reason string) (types.Sprint, error) {
	if _, err := lockProjectWriteIn(transaction, req, request.ProjectID, types.ProjectRoleMember); err != nil {
		return types.Sprint{}, err
	}
	companyID := req.User.CompanyID

	projectSprints, err := loadProjectSprints(companyID, request.ProjectID)
	if err != nil {
		return types.Sprint{}, core.Err("Error al obtener los sprints:", err)
	}
	sprintIndex := slices.IndexFunc(projectSprints, func(sprint types.Sprint) bool { return sprint.ID == request.SprintID })
	if sprintIndex < 0 {
		return types.Sprint{}, core.Err("El sprint no existe.")
	}
	story, err := loadProjectStory(companyID, request.ProjectID, request.StoryID)
	if err != nil {
		return types.Sprint{}, core.Err("Error al obtener la historia:", err)
	}
	if story == nil {
		return types.Sprint{}, core.Err("La historia no existe.")
	}

	now := core.SUnixTime()
	movedSprint := projectSprints[sprintIndex]
	fromState, err := moveSprintStory(&movedSprint, request.StoryID, request.State, request.PreviousStoryID, request.NextStoryID, now)
	if err != nil {
		return types.Sprint{}, err
	}
	movedSprint.UpdatedBy = req.User.ID
	if err := updateSprintsIn(transaction, []types.Sprint{movedSprint}); err != nil {
		return types.Sprint{}, core.Err("Error al guardar el movimiento:", err)
	}

	// A reorder inside the column only writes the sprint.
	if fromState != request.State {
		setStoryState(story, storyStateAfterCardMove(*story, request.SprintID, request.State), core.FechaUnix())
		story.UpdatedBy = req.User.ID
		if err := updateStoriesStateIn(transaction, []types.Story{*story}); err != nil {
			return types.Sprint{}, core.Err("Error al guardar el estado de la historia:", err)
		}
		if err := addStoryActivitiesIn(transaction, types.StoryActivity{
			CompanyID: companyID, ProjectID: request.ProjectID, StoryID: request.StoryID, Kind: types.ActivityKindStateChange, FromState: fromState,
			ToState: request.State, SprintID: request.SprintID, Text: reason, AuthorID: req.User.ID, Created: now,
		}); err != nil {
			return types.Sprint{}, core.Err("Error al guardar la actividad de la historia:", err)
		}
	}
	return movedSprint, nil
}

type sprintCloseRequest struct {
	ProjectID  int32
	SprintID   int64
	CarryOvers []SprintCarryOver
}

// PostSprintClose closes the active sprint (admins). Its unfinished stories go on in the planned
// sprints chosen for them, keeping their state, or back to the backlog. The whole close commits in
// one transaction.
func PostSprintClose(req *core.HandlerArgs) core.HandlerResponse {
	request := sprintCloseRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	var closedSprint types.Sprint
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var err error
		closedSprint, err = closeSprintIn(transaction, req, request)
		return err
	})
	if err != nil {
		return makeProjectErr(req, err)
	}
	return req.MakeResponse(closedSprint)
}

// closeSprintIn closes the active sprint and carries its unfinished stories, under the project's row
// lock.
func closeSprintIn(transaction *db.Transaction, req *core.HandlerArgs, request sprintCloseRequest) (types.Sprint, error) {
	if _, err := lockProjectWriteIn(transaction, req, request.ProjectID, types.ProjectRoleAdmin); err != nil {
		return types.Sprint{}, err
	}
	companyID := req.User.CompanyID

	projectSprints, err := loadProjectSprints(companyID, request.ProjectID)
	if err != nil {
		return types.Sprint{}, core.Err("Error al obtener los sprints:", err)
	}
	closingIndex := slices.IndexFunc(projectSprints, func(sprint types.Sprint) bool { return sprint.ID == request.SprintID })
	if closingIndex < 0 {
		return types.Sprint{}, core.Err("El sprint no existe.")
	}
	closingSprint := projectSprints[closingIndex]
	closedSprint := closingSprint
	closedSprint.Stories = slices.Clone(closingSprint.Stories)
	now := core.SUnixTime()
	if err := closeSprint(&closedSprint, request.CarryOvers, projectSprints, req.User.ID, now); err != nil {
		return types.Sprint{}, err
	}
	closedSprint.UpdatedBy = req.User.ID

	carriedStoryIDsBySprintID := map[int64][]int64{}
	for _, carryOver := range request.CarryOvers {
		if carryOver.ToSprintID != 0 {
			carriedStoryIDsBySprintID[carryOver.ToSprintID] = append(carriedStoryIDsBySprintID[carryOver.ToSprintID], carryOver.StoryID)
		}
	}
	targetSprints := []types.Sprint{}
	for _, projectSprint := range projectSprints {
		carriedStoryIDs, isTarget := carriedStoryIDsBySprintID[projectSprint.ID]
		if !isTarget {
			continue
		}
		projectSprint.Stories = slices.Clone(projectSprint.Stories)
		if err := carryStoriesIntoSprint(&projectSprint, closingSprint, carriedStoryIDs, now); err != nil {
			return types.Sprint{}, err
		}
		projectSprint.UpdatedBy = req.User.ID
		targetSprints = append(targetSprints, projectSprint)
	}
	writtenSprints := append(targetSprints, closedSprint)
	if err := updateSprintsIn(transaction, writtenSprints); err != nil {
		return types.Sprint{}, core.Err("Error al cerrar el sprint:", err)
	}

	carriedStoryIDs := []int64{}
	activities := []types.StoryActivity{}
	for _, carryOver := range request.CarryOvers {
		carriedStoryIDs = append(carriedStoryIDs, carryOver.StoryID)
		activities = append(activities, types.StoryActivity{
			CompanyID: companyID, ProjectID: request.ProjectID, StoryID: carryOver.StoryID, Kind: types.ActivityKindCarriedOver,
			SprintID: request.SprintID, ToSprintID: carryOver.ToSprintID, AuthorID: req.User.ID, Created: now,
		})
	}
	if err := syncStoriesFinalStateIn(transaction, companyID, request.ProjectID, withWrittenSprints(projectSprints, writtenSprints...),
		carriedStoryIDs, req.User.ID, activities); err != nil {
		return types.Sprint{}, core.Err("Error al guardar el estado de las historias del sprint:", err)
	}
	return closedSprint, nil
}
