package projects

import (
	"encoding/json"
	"slices"

	"app/core"
	"app/db"
	"app/projects/types"
)

// A project's backlog is three delta-cached lists, one route per table (a delta response holds one
// table): the browser paints them from its cache and then merges only what changed. The project
// itself comes from the cached GET.projects-projects.

// GetEpics is the delta of a project's epics.
func GetEpics(req *core.HandlerArgs) core.HandlerResponse {
	epics := []types.Epic{}
	query := db.Query(&epics)
	query.CompanyID.Equals(req.User.CompanyID).ProjectID.Equals(req.GetQueryInt("project-id")).
		Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las épicas:", err)
	}
	return core.MakeResponse(req, &epics)
}

// GetStories is the delta of a project's stories: basic fields and final state.
func GetStories(req *core.HandlerArgs) core.HandlerResponse {
	stories := []types.Story{}
	query := db.Query(&stories)
	query.CompanyID.Equals(req.User.CompanyID).ProjectID.Equals(req.GetQueryInt("project-id")).
		Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las historias:", err)
	}
	return core.MakeResponse(req, &stories)
}

// GetStoriesByIDs serves genix-ui's cache-by-ids for stories of any project of the company: only the
// ones whose slot moved since the tick the client holds come back.
func GetStoriesByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID)
	if err != nil {
		return req.MakeErr(err)
	}
	partitions, err := db.QueryCachedIDs[types.Story](cachedIDsRequest)
	if err != nil {
		return req.MakeErr("Error al obtener las historias:", err)
	}
	return req.MakeResponse(core.HideCachedPartitions(partitions))
}

// GetAssignedStories is the delta of one person's stories across every project: only their IDs and
// projects, which the browser resolves through GET.projects-stories-ids.
func GetAssignedStories(req *core.HandlerArgs) core.HandlerResponse {
	assigneeID := req.GetQueryInt("user-id")
	if assigneeID <= 0 {
		return req.MakeErr("No se envió la persona.")
	}
	assigneeRows := []types.StoryAssignee{}
	query := db.Query(&assigneeRows)
	query.CompanyID.Equals(req.User.CompanyID).AssigneeID.Equals(assigneeID).Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las historias de la persona:", err)
	}
	return core.MakeResponse(req, &assigneeRows)
}

// saveStoryAssignees moves written stories in the per-person index. Called only when a story is new
// (its row is inserted, unassigned too, so every later change is an update) or its assignee or
// status changed: every write appends to the delta log.
func saveStoryAssignees(companyID int32, isNewStory bool, stories ...types.Story) error {
	if len(stories) == 0 {
		return nil
	}
	assigneeRows := make([]types.StoryAssignee, len(stories))
	for storyIndex, story := range stories {
		assigneeRows[storyIndex] = storyAssigneeRow(story)
		assigneeRows[storyIndex].CompanyID = companyID
	}
	if isNewStory {
		return db.Insert(&assigneeRows)
	}
	assigneeTable := db.TableOf[types.StoryAssignee]()
	return db.Update(&assigneeRows, assigneeTable.ProjectID, assigneeTable.AssigneeID, assigneeTable.Status)
}

// PostEpics saves the epics changed in the Epics tab of the project panel, all of one project: it
// creates and renames them (team members) and deletes them (admins). A deleted epic leaves its
// stories without one. It answers the saved epics.
func PostEpics(req *core.HandlerArgs) core.HandlerResponse {
	epics := []types.Epic{}
	if err := json.Unmarshal([]byte(*req.Body), &epics); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	projectID, hasDeletion, err := normalizeEpics(epics)
	if err != nil {
		return req.MakeErr(err)
	}
	requiredRole := types.ProjectRoleMember
	if hasDeletion {
		requiredRole = types.ProjectRoleAdmin
	}
	_, release, response, isDenied := openProjectWrite(req, projectID, requiredRole)
	if isDenied {
		return response
	}
	defer release()

	storedEpics := []types.Epic{}
	storedQuery := db.Query(&storedEpics)
	storedQuery.CompanyID.Equals(req.User.CompanyID).ProjectID.Equals(projectID).Status.Equals(types.StatusActive)
	if err := storedQuery.Exec(); err != nil {
		return req.MakeErr("Error al obtener las épicas:", err)
	}

	newEpics, editedEpics, deletedEpicIDs := []types.Epic{}, []types.Epic{}, []int64{}
	for _, epic := range epics {
		epic.CompanyID, epic.UpdatedBy = req.User.CompanyID, req.User.ID
		if epic.ID == 0 {
			epic.Status = types.StatusActive
			newEpics = append(newEpics, epic)
			continue
		}
		if !slices.ContainsFunc(storedEpics, func(storedEpic types.Epic) bool { return storedEpic.ID == epic.ID }) {
			return req.MakeErr("La épica", epic.Name, "no existe.")
		}
		if epic.Status == types.StatusDeleted {
			deletedEpicIDs = append(deletedEpicIDs, epic.ID)
		} else {
			epic.Status = types.StatusActive
		}
		editedEpics = append(editedEpics, epic)
	}
	if len(newEpics) > 0 {
		if err := db.Insert(&newEpics); err != nil {
			return req.MakeErr("Error al guardar las épicas:", err)
		}
	}
	if len(editedEpics) > 0 {
		epicTable := db.TableOf[types.Epic]()
		if err := db.Update(&editedEpics, epicTable.ProjectID, epicTable.Name, epicTable.Color, epicTable.Status, epicTable.UpdatedBy); err != nil {
			return req.MakeErr("Error al guardar las épicas:", err)
		}
	}
	if err := clearStoriesEpic(req.User.CompanyID, projectID, deletedEpicIDs, req.User.ID); err != nil {
		return req.MakeErr("Se eliminaron las épicas, pero no se pudieron actualizar sus historias:", err)
	}
	return req.MakeResponse(append(newEpics, editedEpics...))
}

// clearStoriesEpic takes deleted epics off their stories. Runs under the project lock.
func clearStoriesEpic(companyID int32, projectID int32, deletedEpicIDs []int64, userID int32) error {
	if len(deletedEpicIDs) == 0 {
		return nil
	}
	projectStories, err := loadProjectStories(companyID, projectID)
	if err != nil {
		return err
	}
	epiclessStories := []types.Story{}
	for _, story := range projectStories {
		if slices.Contains(deletedEpicIDs, story.EpicID) {
			story.EpicID, story.UpdatedBy = 0, userID
			epiclessStories = append(epiclessStories, story)
		}
	}
	if len(epiclessStories) == 0 {
		return nil
	}
	storyTable := db.TableOf[types.Story]()
	return db.Update(&epiclessStories, storyTable.ProjectID, storyTable.Status, storyTable.EpicID, storyTable.UpdatedBy)
}

type storyWithDetail struct {
	Story  types.Story
	Detail types.StoryDetail
}

// PostStory creates or edits a story with its detail (team members), or deletes it (admins). The
// number, rank, final state and counters never come from the body.
func PostStory(req *core.HandlerArgs) core.HandlerResponse {
	request := storyWithDetail{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	story, detail := request.Story, request.Detail
	companyID := req.User.CompanyID
	if story.ID != 0 && story.Status == types.StatusDeleted {
		_, release, response, isDenied := openProjectWrite(req, story.ProjectID, types.ProjectRoleAdmin)
		if isDenied {
			return response
		}
		defer release()
		return deleteStory(req, story.ProjectID, story.ID)
	}

	project, release, response, isDenied := openProjectWrite(req, story.ProjectID, types.ProjectRoleMember)
	if isDenied {
		return response
	}
	defer release()

	activeEpicIDs, err := loadProjectEpicIDs(companyID, project.ID)
	if err != nil {
		return req.MakeErr("Error al obtener las épicas:", err)
	}
	if err := normalizeStory(&story, project, activeEpicIDs); err != nil {
		return req.MakeErr(err)
	}
	if err := normalizeStoryDetail(&detail); err != nil {
		return req.MakeErr(err)
	}

	var storedStory *types.Story
	storedAttachments := []types.Attachment{}
	if story.ID != 0 {
		if storedStory, err = loadProjectStory(companyID, project.ID, story.ID); err != nil {
			return req.MakeErr("Error al obtener la historia:", err)
		}
		if storedStory == nil {
			return req.MakeErr("La historia no existe.")
		}
		storedDetails := []types.StoryDetail{}
		detailQuery := db.Query(&storedDetails)
		detailQuery.CompanyID.Equals(companyID).StoryID.Equals(story.ID)
		if err := detailQuery.Exec(); err != nil {
			return req.MakeErr("Error al obtener el detalle de la historia:", err)
		}
		if len(storedDetails) > 0 {
			storedAttachments = storedDetails[0].Attachments
		}
	}
	now := core.SUnixTime()
	attachments, removedAttachmentKeys, err := mergeAttachments(companyID, project.ID, storedAttachments, detail.Attachments, req.User.ID, now)
	if err != nil {
		return req.MakeErr(err)
	}

	previousAssigneeID := int32(0)
	if storedStory == nil {
		// The insert takes the ID, which numbers the story inside its project. Its creation time ranks
		// it after every story ranked so far, so it lands last in the backlog.
		story.ID, story.CompanyID, story.BacklogRank = 0, companyID, float64(now)
		story.State, story.LastSprintID, story.SprintIDs, story.CommentsCount, story.StateDates = types.StoryStateNotPlanned, 0, nil, 0, nil
		story.AttachmentsCount = int16(len(attachments))
		story.CreatedBy, story.Created, story.Status, story.UpdatedBy = req.User.ID, now, types.StatusActive, req.User.ID
		stories := []types.Story{story}
		if err := db.Insert(&stories); err != nil {
			return req.MakeErr("Error al guardar la historia:", err)
		}
		story = stories[0]
	} else {
		previousAssigneeID = storedStory.AssigneeID
		storedStory.Type, storedStory.Title, storedStory.Summary, storedStory.EpicID = story.Type, story.Title, story.Summary, story.EpicID
		storedStory.Priority, storedStory.AssigneeID, storedStory.EstimatedHours = story.Priority, story.AssigneeID, story.EstimatedHours
		storedStory.AttachmentsCount, storedStory.UpdatedBy = int16(len(attachments)), req.User.ID
		storyTable := db.TableOf[types.Story]()
		err := db.Update(&[]types.Story{*storedStory}, storyTable.ProjectID, storyTable.Status, storyTable.Type, storyTable.Title, storyTable.Summary, storyTable.EpicID,
			storyTable.Priority, storyTable.AssigneeID, storyTable.EstimatedHours, storyTable.AttachmentsCount, storyTable.UpdatedBy)
		if err != nil {
			return req.MakeErr("Error al guardar la historia:", err)
		}
		story = *storedStory
	}

	// A new assignee leaves a history line and moves the story in the per-person index.
	if story.AssigneeID != previousAssigneeID {
		addStoryActivities(types.StoryActivity{
			CompanyID: companyID, ProjectID: project.ID, StoryID: story.ID, Kind: types.ActivityKindAssigneeChange,
			AssigneeID: story.AssigneeID, AuthorID: req.User.ID, Created: now,
		})
	}
	if storedStory == nil || story.AssigneeID != previousAssigneeID {
		if err := saveStoryAssignees(companyID, storedStory == nil, story); err != nil {
			return req.MakeErr("Se guardó la historia, pero no en la lista de su responsable:", err)
		}
	}

	// The Documents page's copies go before the detail: if the detail then fails, saving again
	// recomputes the same changes from the stored detail. Written after it, a failure would leave
	// them out of step for good.
	documentChanges := storyDocumentChanges(project.ID, story.ID, storedAttachments, attachments, req.User.ID)
	if len(documentChanges) > 0 {
		if err := saveDocumentChanges(companyID, documentChanges); err != nil {
			return req.MakeErr("Se guardó la historia, pero no sus archivos:", err)
		}
		linkAttachmentDocuments(attachments, documentChanges)
	}

	detail.CompanyID, detail.ProjectID, detail.StoryID, detail.Attachments = companyID, project.ID, story.ID, attachments
	detail.Status, detail.UpdatedBy = types.StatusActive, req.User.ID
	if err := db.Insert(&[]types.StoryDetail{detail}); err != nil {
		return req.MakeErr("Se guardó la historia, pero no su detalle:", err)
	}
	deleteProjectFiles(removedAttachmentKeys)
	return req.MakeResponse(storyWithDetail{Story: story, Detail: detail})
}

// saveDocumentChanges writes the Documents page's rows of a story's files: new ones take their ID on
// insert (read back by linkAttachmentDocuments), removed ones are soft-deleted. The inserts go through
// db.Insert on the caller's own elements: InsertUpdateInclude writes copies, so the IDs it assigns
// never reach the caller.
func saveDocumentChanges(companyID int32, documents []types.Document) error {
	newDocuments, removedDocuments := []types.Document{}, []types.Document{}
	for _, document := range documents {
		document.CompanyID = companyID
		if document.ID == 0 {
			newDocuments = append(newDocuments, document)
		} else {
			removedDocuments = append(removedDocuments, document)
		}
	}
	if len(newDocuments) > 0 {
		if err := db.Insert(&newDocuments); err != nil {
			return err
		}
	}
	if len(removedDocuments) > 0 {
		documentTable := db.TableOf[types.Document]()
		if err := db.Update(&removedDocuments, documentTable.ProjectID, documentTable.Status, documentTable.UpdatedBy); err != nil {
			return err
		}
	}
	// The written rows back into the caller's slice, new ones first and holding their IDs: callers
	// match them by Key, not by position.
	copy(documents, slices.Concat(newDocuments, removedDocuments))
	return nil
}

// deleteStory soft-deletes a story and takes it off the open sprints. Closed sprints keep it: they
// are history. Its files stay, so the story can be restored. Runs under the project lock.
func deleteStory(req *core.HandlerArgs, projectID int32, storyID int64) core.HandlerResponse {
	companyID := req.User.CompanyID
	story, err := loadProjectStory(companyID, projectID, storyID)
	if err != nil {
		return req.MakeErr("Error al obtener la historia:", err)
	}
	if story == nil {
		return req.MakeErr("La historia no existe.")
	}
	story.Status, story.UpdatedBy = types.StatusDeleted, req.User.ID
	storyTable := db.TableOf[types.Story]()
	if err := db.Update(&[]types.Story{*story}, storyTable.ProjectID, storyTable.Status, storyTable.UpdatedBy); err != nil {
		return req.MakeErr("Error al eliminar la historia:", err)
	}
	if story.AssigneeID != 0 {
		if err := saveStoryAssignees(companyID, false, *story); err != nil {
			return req.MakeErr("Se eliminó la historia, pero sigue en la lista de su responsable:", err)
		}
	}

	projectSprints, err := loadProjectSprints(companyID, projectID)
	if err != nil {
		return req.MakeErr("Se eliminó la historia, pero no se pudieron leer los sprints:", err)
	}
	changedSprints := []types.Sprint{}
	for _, sprint := range projectSprints {
		if !isSprintOpen(sprint) || sprintStoryIndex(sprint, storyID) < 0 {
			continue
		}
		sprint.Stories = slices.DeleteFunc(sprint.Stories, func(sprintStory types.SprintStory) bool { return sprintStory.StoryID == storyID })
		sprint.UpdatedBy = req.User.ID
		changedSprints = append(changedSprints, sprint)
	}
	if len(changedSprints) > 0 {
		sprintTable := db.TableOf[types.Sprint]()
		if err := db.Update(&changedSprints, sprintTable.ProjectID, sprintTable.Status, sprintTable.Stories, sprintTable.UpdatedBy); err != nil {
			return req.MakeErr("Se eliminó la historia, pero no se pudo quitar de sus sprints:", err)
		}
	}
	return req.MakeResponse(storyWithDetail{Story: *story})
}

type storyRankRequest struct {
	ProjectID       int32
	StoryID         int64
	PreviousStoryID int64
	NextStoryID     int64
	// EpicID is the epic group the story was dropped in, 0 = no epic.
	EpicID int64
}

// PostStoryRank moves one story in the backlog: between two others, and into another epic group.
func PostStoryRank(req *core.HandlerArgs) core.HandlerResponse {
	request := storyRankRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	_, release, response, isDenied := openProjectWrite(req, request.ProjectID, types.ProjectRoleMember)
	if isDenied {
		return response
	}
	defer release()
	companyID := req.User.CompanyID

	if request.EpicID != 0 {
		activeEpicIDs, err := loadProjectEpicIDs(companyID, request.ProjectID)
		if err != nil {
			return req.MakeErr("Error al obtener las épicas:", err)
		}
		if !slices.Contains(activeEpicIDs, request.EpicID) {
			return req.MakeErr("La épica no existe.")
		}
	}

	// The moved story and its neighbours in one read. A drop into an empty group has no neighbours.
	readStoryIDs := []int64{request.StoryID}
	for _, neighbourID := range []int64{request.PreviousStoryID, request.NextStoryID} {
		if neighbourID != 0 && !slices.Contains(readStoryIDs, neighbourID) {
			readStoryIDs = append(readStoryIDs, neighbourID)
		}
	}
	readStories := []types.Story{}
	storiesQuery := db.Query(&readStories)
	storiesQuery.CompanyID.Equals(companyID).ID.In(readStoryIDs...)
	if err := storiesQuery.Exec(); err != nil {
		return req.MakeErr("Error al obtener las historias:", err)
	}

	var movedStory *types.Story
	var previousRank, nextRank float64
	var hasPrevious, hasNext bool
	for storyIndex, readStory := range readStories {
		if readStory.ProjectID != request.ProjectID {
			continue
		}
		if readStory.ID == request.StoryID {
			movedStory = &readStories[storyIndex]
			continue
		}
		if readStory.ID == request.PreviousStoryID {
			previousRank, hasPrevious = readStory.BacklogRank, true
		}
		if readStory.ID == request.NextStoryID {
			nextRank, hasNext = readStory.BacklogRank, true
		}
	}
	if movedStory == nil || movedStory.Status != types.StatusActive {
		return req.MakeErr("La historia no existe.")
	}
	movedStory.BacklogRank = rankBetween(previousRank, hasPrevious, nextRank, hasNext)
	movedStory.EpicID, movedStory.UpdatedBy = request.EpicID, req.User.ID
	storyTable := db.TableOf[types.Story]()
	if err := db.Update(&[]types.Story{*movedStory}, storyTable.ProjectID, storyTable.Status, storyTable.BacklogRank, storyTable.EpicID, storyTable.UpdatedBy); err != nil {
		return req.MakeErr("Error al mover la historia:", err)
	}
	return req.MakeResponse(*movedStory)
}

// GetStoryDetail is one story's detail, a list of at most one record answered as a full replace: it
// has no delta log (types.StoryDetailTable).
func GetStoryDetail(req *core.HandlerArgs) core.HandlerResponse {
	details := []types.StoryDetail{}
	query := db.Query(&details)
	query.CompanyID.Equals(req.User.CompanyID).StoryID.Equals(req.GetQueryInt64("story-id"))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el detalle de la historia:", err)
	}
	return req.MakeResponse(core.WithDeltaReplace(details))
}

// GetStoryActivity is the delta of one story's timeline: comments and history.
func GetStoryActivity(req *core.HandlerArgs) core.HandlerResponse {
	activities := []types.StoryActivity{}
	query := db.Query(&activities)
	query.CompanyID.Equals(req.User.CompanyID).StoryID.Equals(req.GetQueryInt64("story-id")).
		Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener la actividad de la historia:", err)
	}
	return core.MakeResponse(req, &activities)
}

type storyCommentRequest struct {
	ProjectID   int32
	StoryID     int64
	Text        string
	Attachments []types.Attachment
}

// PostStoryComment adds a comment, with its files, to a story's timeline. Only the team comments.
func PostStoryComment(req *core.HandlerArgs) core.HandlerResponse {
	request := storyCommentRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	_, release, response, isDenied := openProjectWrite(req, request.ProjectID, types.ProjectRoleMember)
	if isDenied {
		return response
	}
	defer release()
	companyID := req.User.CompanyID

	now := core.SUnixTime()
	attachments, _, err := mergeAttachments(companyID, request.ProjectID, nil, request.Attachments, req.User.ID, now)
	if err != nil {
		return req.MakeErr(err)
	}
	commentText, err := normalizeCommentText(request.Text, len(attachments))
	if err != nil {
		return req.MakeErr(err)
	}
	story, err := loadProjectStory(companyID, request.ProjectID, request.StoryID)
	if err != nil {
		return req.MakeErr("Error al obtener la historia:", err)
	}
	if story == nil {
		return req.MakeErr("La historia no existe.")
	}
	story.CommentsCount++
	storyTable := db.TableOf[types.Story]()
	if err := db.Update(&[]types.Story{*story}, storyTable.ProjectID, storyTable.Status, storyTable.CommentsCount); err != nil {
		return req.MakeErr("Error al guardar el comentario:", err)
	}

	comments := []types.StoryActivity{{
		CompanyID: companyID, ProjectID: request.ProjectID, StoryID: request.StoryID, Kind: types.ActivityKindComment,
		Text: commentText, Attachments: attachments, AuthorID: req.User.ID, Created: now, Status: types.StatusActive,
	}}
	if err := db.Insert(&comments); err != nil {
		return req.MakeErr("Error al guardar el comentario:", err)
	}
	// The Documents page's copies of its files name the comment, so they go once it has its ID; the
	// comment then records their IDs.
	if documents := commentDocuments(comments[0], req.User.ID); len(documents) > 0 {
		if err := saveDocumentChanges(companyID, documents); err != nil {
			return req.MakeErr("Se guardó el comentario, pero no sus archivos:", err)
		}
		linkAttachmentDocuments(comments[0].Attachments, documents)
		activityTable := db.TableOf[types.StoryActivity]()
		if err := db.Update(&comments, activityTable.StoryID, activityTable.Status, activityTable.Attachments); err != nil {
			return req.MakeErr("Se guardó el comentario, pero no el enlace a sus archivos:", err)
		}
	}
	return req.MakeResponse(comments[0])
}

// addStoryActivities writes history entries. Best effort: the change they record is already saved,
// so a failure is logged and never fails the request.
func addStoryActivities(activities ...types.StoryActivity) {
	if len(activities) == 0 {
		return
	}
	for activityIndex := range activities {
		activities[activityIndex].Status = types.StatusActive
	}
	if err := db.Insert(&activities); err != nil {
		core.Log("Error al guardar la actividad de la historia", err)
	}
}
