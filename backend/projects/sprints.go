package projects

import (
	"encoding/json"
	"slices"

	"app/core"
	"app/db"
	"app/projects/types"
)

// Every sprint write runs under the project lock (openProjectWrite): the rules read the project's
// sprints and stories, decide, and write, with no other writer in between.

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

// updateSprints writes every field a sprint writer may change. ProjectID and Status go with any
// update: they are the delta index's keys.
func updateSprints(sprints []types.Sprint) error {
	sprintTable := db.TableOf[types.Sprint]()
	return db.Update(&sprints, sprintTable.ProjectID, sprintTable.Status, sprintTable.Name, sprintTable.Goal,
		sprintTable.StartDate, sprintTable.EndDate, sprintTable.State, sprintTable.Stories,
		sprintTable.ClosedAt, sprintTable.ClosedBy, sprintTable.UpdatedBy)
}

// updateStoriesState writes a story's final state and the sprints that hold it.
func updateStoriesState(stories []types.Story) error {
	storyTable := db.TableOf[types.Story]()
	return db.Update(&stories, storyTable.ProjectID, storyTable.Status, storyTable.State, storyTable.StateDates,
		storyTable.LastSprintID, storyTable.SprintIDs, storyTable.UpdatedBy)
}

// syncStoriesFinalState rewrites the final state of the stories a sprint change touched, from the
// sprints as stored now: every writer of a sprint calls it after its own write. The history entries
// of a story that is gone are dropped. Runs under the project lock.
func syncStoriesFinalState(companyID int32, projectID int32, storyIDs []int64, userID int32, activities []types.StoryActivity) error {
	if len(storyIDs) == 0 {
		return nil
	}
	projectSprints, err := loadProjectSprints(companyID, projectID)
	if err != nil {
		return err
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
		finalState, lastSprintID, sprintIDs := storyFinalState(storyID, projectSprints)
		setStoryState(&story, finalState, today)
		story.LastSprintID, story.SprintIDs, story.UpdatedBy = lastSprintID, sprintIDs, userID
		syncedStories = append(syncedStories, story)
	}
	if len(syncedStories) > 0 {
		if err := updateStoriesState(syncedStories); err != nil {
			return err
		}
	}
	addStoryActivities(activities...)
	return nil
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
	sprint := request.Sprint
	isDeletion := sprint.ID != 0 && sprint.Status == types.StatusDeleted
	project, release, response, isDenied := openProjectWrite(req, sprint.ProjectID, types.ProjectRoleAdmin)
	if isDenied {
		return response
	}
	defer release()
	companyID := req.User.CompanyID

	projectSprints, err := loadProjectSprints(companyID, project.ID)
	if err != nil {
		return req.MakeErr("Error al obtener los sprints:", err)
	}
	if !isDeletion {
		if err := normalizeSprint(&sprint, projectSprints); err != nil {
			return req.MakeErr(err)
		}
	}
	storiesByID := map[int64]types.Story{}
	if request.StoryIDs != nil || len(request.StoryAssignees) > 0 {
		projectStories, err := loadProjectStories(companyID, project.ID)
		if err != nil {
			return req.MakeErr("Error al obtener las historias:", err)
		}
		for _, story := range projectStories {
			storiesByID[story.ID] = story
		}
	}
	for _, assignee := range request.StoryAssignees {
		if _, storyExists := storiesByID[assignee.StoryID]; !storyExists || (assignee.AssigneeID != 0 && !isProjectMember(project, assignee.AssigneeID)) {
			return req.MakeErr("El responsable debe ser parte del equipo del proyecto.")
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
				return req.MakeErr(err)
			}
		}
		if stateChanges, err = planSprintStoryStates(&sprint, request.StoryStates, now); err != nil {
			return req.MakeErr(err)
		}
		newSprints := []types.Sprint{sprint}
		if err := db.Insert(&newSprints); err != nil {
			return req.MakeErr("Error al guardar el sprint:", err)
		}
		sprint = newSprints[0]
	} else {
		storedIndex := slices.IndexFunc(projectSprints, func(projectSprint types.Sprint) bool { return projectSprint.ID == sprint.ID })
		if storedIndex < 0 {
			return req.MakeErr("El sprint no existe.")
		}
		stored := projectSprints[storedIndex]
		if !isSprintOpen(stored) {
			return req.MakeErr("Un sprint cerrado no se puede modificar.")
		}
		stored.UpdatedBy = req.User.ID
		if isDeletion {
			if stored.State != types.SprintStatePlanned {
				return req.MakeErr("Solo se puede eliminar un sprint planificado.")
			}
			for _, sprintStory := range stored.Stories {
				removedStoryIDs = append(removedStoryIDs, sprintStory.StoryID)
			}
			stored.Status = types.StatusDeleted
		} else {
			stored.Name, stored.Goal, stored.StartDate, stored.EndDate = sprint.Name, sprint.Goal, sprint.StartDate, sprint.EndDate
			if request.StoryIDs != nil {
				if addedStoryIDs, removedStoryIDs, err = planSprintStories(&stored, *request.StoryIDs, storiesByID, projectSprints, now); err != nil {
					return req.MakeErr(err)
				}
			}
			if stateChanges, err = planSprintStoryStates(&stored, request.StoryStates, now); err != nil {
				return req.MakeErr(err)
			}
		}
		if err := updateSprints([]types.Sprint{stored}); err != nil {
			return req.MakeErr("Error al guardar el sprint:", err)
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
	if err := syncStoriesFinalState(companyID, project.ID, slices.Compact(touchedStoryIDs), req.User.ID, sprintActivities); err != nil {
		return req.MakeErr("Se guardó el sprint, pero no el estado de sus historias:", err)
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
		if err := db.Update(&reassignedStories, storyTable.ProjectID, storyTable.Status, storyTable.AssigneeID, storyTable.UpdatedBy); err != nil {
			return req.MakeErr("Se guardó el sprint, pero no sus responsables:", err)
		}
		if err := saveStoryAssignees(companyID, false, reassignedStories...); err != nil {
			return req.MakeErr("Se guardó el sprint, pero no la lista de sus responsables:", err)
		}
		addStoryActivities(assigneeActivities...)
	}
	return req.MakeResponse(sprint)
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
	_, release, response, isDenied := openProjectWrite(req, request.ProjectID, types.ProjectRoleAdmin)
	if isDenied {
		return response
	}
	defer release()

	projectSprints, err := loadProjectSprints(req.User.CompanyID, request.ProjectID)
	if err != nil {
		return req.MakeErr("Error al obtener los sprints:", err)
	}
	sprintIndex := slices.IndexFunc(projectSprints, func(sprint types.Sprint) bool { return sprint.ID == request.SprintID })
	if sprintIndex < 0 {
		return req.MakeErr("El sprint no existe.")
	}
	startedSprint := projectSprints[sprintIndex]
	if err := startSprint(&startedSprint, projectSprints); err != nil {
		return req.MakeErr(err)
	}
	startedSprint.UpdatedBy = req.User.ID
	if err := updateSprints([]types.Sprint{startedSprint}); err != nil {
		return req.MakeErr("Error al iniciar el sprint:", err)
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
// between two cards of it. Moves are serialized by the project lock, so two people moving cards at
// once both land, one after the other.
func PostSprintMove(req *core.HandlerArgs) core.HandlerResponse {
	request := sprintMoveRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	reason, err := normalizeMoveReason(request.State, request.Reason)
	if err != nil {
		return req.MakeErr(err)
	}
	_, release, response, isDenied := openProjectWrite(req, request.ProjectID, types.ProjectRoleMember)
	if isDenied {
		return response
	}
	defer release()
	companyID := req.User.CompanyID

	projectSprints, err := loadProjectSprints(companyID, request.ProjectID)
	if err != nil {
		return req.MakeErr("Error al obtener los sprints:", err)
	}
	sprintIndex := slices.IndexFunc(projectSprints, func(sprint types.Sprint) bool { return sprint.ID == request.SprintID })
	if sprintIndex < 0 {
		return req.MakeErr("El sprint no existe.")
	}
	story, err := loadProjectStory(companyID, request.ProjectID, request.StoryID)
	if err != nil {
		return req.MakeErr("Error al obtener la historia:", err)
	}
	if story == nil {
		return req.MakeErr("La historia no existe.")
	}

	now := core.SUnixTime()
	movedSprint := projectSprints[sprintIndex]
	fromState, err := moveSprintStory(&movedSprint, request.StoryID, request.State, request.PreviousStoryID, request.NextStoryID, now)
	if err != nil {
		return req.MakeErr(err)
	}
	movedSprint.UpdatedBy = req.User.ID
	if err := updateSprints([]types.Sprint{movedSprint}); err != nil {
		return req.MakeErr("Error al guardar el movimiento:", err)
	}

	// A reorder inside the column only writes the sprint.
	if fromState != request.State {
		setStoryState(story, storyStateAfterCardMove(*story, request.SprintID, request.State), core.FechaUnix())
		story.UpdatedBy = req.User.ID
		if err := updateStoriesState([]types.Story{*story}); err != nil {
			return req.MakeErr("Se movió la tarjeta, pero no el estado de la historia:", err)
		}
		addStoryActivities(types.StoryActivity{
			CompanyID: companyID, ProjectID: request.ProjectID, StoryID: request.StoryID, Kind: types.ActivityKindStateChange, FromState: fromState,
			ToState: request.State, SprintID: request.SprintID, Text: reason, AuthorID: req.User.ID, Created: now,
		})
	}
	return req.MakeResponse(movedSprint)
}

type sprintCloseRequest struct {
	ProjectID  int32
	SprintID   int64
	CarryOvers []SprintCarryOver
}

// PostSprintClose closes the active sprint (admins). Its unfinished stories go on in the planned
// sprints chosen for them, keeping their state, or back to the backlog. The planned sprints are
// written first and skip what they already hold, so a close that failed half way can be retried.
func PostSprintClose(req *core.HandlerArgs) core.HandlerResponse {
	request := sprintCloseRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	_, release, response, isDenied := openProjectWrite(req, request.ProjectID, types.ProjectRoleAdmin)
	if isDenied {
		return response
	}
	defer release()
	companyID := req.User.CompanyID

	projectSprints, err := loadProjectSprints(companyID, request.ProjectID)
	if err != nil {
		return req.MakeErr("Error al obtener los sprints:", err)
	}
	closingIndex := slices.IndexFunc(projectSprints, func(sprint types.Sprint) bool { return sprint.ID == request.SprintID })
	if closingIndex < 0 {
		return req.MakeErr("El sprint no existe.")
	}
	closingSprint := projectSprints[closingIndex]
	closedSprint := closingSprint
	closedSprint.Stories = slices.Clone(closingSprint.Stories)
	now := core.SUnixTime()
	if err := closeSprint(&closedSprint, request.CarryOvers, projectSprints, req.User.ID, now); err != nil {
		return req.MakeErr(err)
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
		if err := carryStoriesIntoSprint(&projectSprint, closingSprint, carriedStoryIDs, now); err != nil {
			return req.MakeErr(err)
		}
		projectSprint.UpdatedBy = req.User.ID
		targetSprints = append(targetSprints, projectSprint)
	}
	if err := updateSprints(append(targetSprints, closedSprint)); err != nil {
		return req.MakeErr("Error al cerrar el sprint:", err)
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
	if err := syncStoriesFinalState(companyID, request.ProjectID, carriedStoryIDs, req.User.ID, activities); err != nil {
		return req.MakeErr("Se cerró el sprint, pero no el estado de sus historias:", err)
	}
	return req.MakeResponse(closedSprint)
}
