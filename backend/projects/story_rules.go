package projects

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"app/core"
	"app/projects/types"
)

// MaxAttachmentBytes caps one attached file. The file travels base64 in the upload request, and a
// Lambda payload caps at 6 MB, so 4 MB (5.4 MB encoded) is what fits with the request around it.
const MaxAttachmentBytes = 4 * 1024 * 1024

const maxAttachmentsPerEntry = 30

// attachmentFileTokenLength is the unguessable part of a file's public link: 16 base36 characters
// are 82 bits.
const attachmentFileTokenLength = 16

// attachmentFolder is the project's folder in the CDN bucket. The company is in it: project IDs
// are per company, so the folder alone must not let one company's record name another's file.
func attachmentFolder(companyID int32, projectID int32) string {
	return fmt.Sprintf("projects/c%d/p%d", companyID, projectID)
}

// attachmentObjectKey is where an uploaded file lives: in its project's folder. The random token
// makes the public link unguessable; the file name keeps the link readable.
func attachmentObjectKey(companyID int32, projectID int32, randomToken string, fileName string) string {
	return attachmentFolder(companyID, projectID) + "/" + randomToken + "/" + core.SafeFileName(fileName)
}

var projectFileKeyPattern = regexp.MustCompile(`^projects/c([0-9]+)/p([0-9]+)/[0-9a-z]{16}/[0-9A-Za-z._-]{1,100}$`)

// isProjectFileKey: a project only takes the files uploaded to its own folder, so no record can list
// (and later delete) another project's file.
func isProjectFileKey(companyID int32, projectID int32, objectKey string) bool {
	keyParts := projectFileKeyPattern.FindStringSubmatch(objectKey)
	return len(keyParts) == 3 && keyParts[1] == strconv.Itoa(int(companyID)) && keyParts[2] == strconv.Itoa(int(projectID))
}

// normalizeStory validates the fields a team member edits. The epic and the assignee must belong
// to the story's project.
func normalizeStory(story *types.Story, project types.Project, projectEpicIDs []int64) error {
	story.Title = strings.TrimSpace(story.Title)
	story.Summary = strings.TrimSpace(story.Summary)
	if story.Title == "" || len(story.Title) > 200 {
		return errors.New("La historia necesita un título de hasta 200 caracteres.")
	}
	if len(story.Summary) > 500 {
		return errors.New("La descripción corta puede tener hasta 500 caracteres.")
	}
	if story.Type < types.StoryTypeStory || story.Type > types.StoryTypeTask {
		return errors.New("El tipo de historia no es válido.")
	}
	if story.Priority < types.StoryPriorityLow || story.Priority > types.StoryPriorityCritical {
		return errors.New("La prioridad no es válida.")
	}
	// Estimates go in half hours.
	if story.EstimatedHours < 0 || story.EstimatedHours > 2000 || math.Mod(float64(story.EstimatedHours)*2, 1) != 0 {
		return errors.New("Las horas estimadas deben estar entre 0 y 2000, en pasos de 0.5.")
	}
	if story.EpicID != 0 && !slices.Contains(projectEpicIDs, story.EpicID) {
		return errors.New("La épica no pertenece a este proyecto.")
	}
	if story.AssigneeID != 0 && !isProjectMember(project, story.AssigneeID) {
		return errors.New("El responsable debe ser parte del equipo del proyecto.")
	}
	return nil
}

// storyAssigneeRow is the story's row in the per-person index: active while someone holds the
// active story. Unassigned or deleted, it leaves its last holder's list.
func storyAssigneeRow(story types.Story) types.StoryAssignee {
	assigneeRow := types.StoryAssignee{StoryID: story.ID, ProjectID: story.ProjectID, AssigneeID: story.AssigneeID, Status: types.StatusDeleted}
	if story.AssigneeID != 0 && story.Status == types.StatusActive {
		assigneeRow.Status = types.StatusActive
	}
	return assigneeRow
}

// normalizeStoryDetail validates the long content. A large column slows every read of its row, so the
// description is bounded and pasted images (data URLs) go as attachments instead.
func normalizeStoryDetail(detail *types.StoryDetail) error {
	if len(detail.Description) > 100_000 {
		return errors.New("La descripción es demasiado larga.")
	}
	if strings.Contains(detail.Description, "data:image") {
		return errors.New("No se permiten imágenes pegadas en la descripción: adjúntalas como archivos.")
	}
	checklist := make([]types.ChecklistItem, 0, len(detail.Checklist))
	for _, item := range detail.Checklist {
		item.Text = strings.TrimSpace(item.Text)
		if item.Text == "" {
			continue
		}
		if len(item.Text) > 300 {
			return errors.New("Un ítem del checklist puede tener hasta 300 caracteres.")
		}
		checklist = append(checklist, item)
	}
	if len(checklist) > 100 {
		return errors.New("El checklist puede tener hasta 100 ítems.")
	}
	detail.Checklist = checklist
	return nil
}

// mergeAttachments takes the attachments the client sends and returns what to store: a stored one
// keeps its stored metadata, a new one is validated (a file of the project's folder) and stamped.
// removedKeys are the stored files the client dropped, to delete from the bucket.
func mergeAttachments(companyID int32, projectID int32, storedAttachments []types.Attachment, incomingAttachments []types.Attachment, userID int32, now int32) ([]types.Attachment, []string, error) {
	if len(incomingAttachments) > maxAttachmentsPerEntry {
		return nil, nil, errors.New("Se pueden adjuntar hasta 30 archivos.")
	}
	storedByKey := map[string]types.Attachment{}
	for _, storedAttachment := range storedAttachments {
		storedByKey[storedAttachment.Key] = storedAttachment
	}
	mergedAttachments := []types.Attachment{}
	keptKeys := map[string]bool{}
	for _, attachment := range incomingAttachments {
		if keptKeys[attachment.Key] {
			continue
		}
		keptKeys[attachment.Key] = true
		if storedAttachment, isStored := storedByKey[attachment.Key]; isStored {
			mergedAttachments = append(mergedAttachments, storedAttachment)
			continue
		}
		attachment.Name = strings.TrimSpace(attachment.Name)
		if !isProjectFileKey(companyID, projectID, attachment.Key) || attachment.Name == "" || len(attachment.Name) > 200 ||
			attachment.Size <= 0 || attachment.Size > MaxAttachmentBytes || len(attachment.ContentType) > 100 {
			return nil, nil, errors.New("Un archivo adjunto no es válido.")
		}
		// A new file has no document row yet: the caller writes it and links it (linkAttachmentDocuments).
		attachment.UploadedBy, attachment.UploadedAt, attachment.DocumentID = userID, now, 0
		mergedAttachments = append(mergedAttachments, attachment)
	}
	removedKeys := []string{}
	for _, storedAttachment := range storedAttachments {
		if !keptKeys[storedAttachment.Key] {
			removedKeys = append(removedKeys, storedAttachment.Key)
		}
	}
	return mergedAttachments, removedKeys, nil
}

// setStoryState sets the story's final state. Entering a sprint state stamps its day in StateDates,
// overwriting the previous time it was in that state; going back to not planned stamps nothing.
func setStoryState(story *types.Story, toState int8, day int16) {
	if story.State == toState {
		return
	}
	story.State = toState
	if toState == types.StoryStateNotPlanned {
		return
	}
	stateDateIndex := slices.IndexFunc(story.StateDates, func(stateDate types.StoryStateDate) bool { return stateDate.State == toState })
	if stateDateIndex < 0 {
		story.StateDates = append(story.StateDates, types.StoryStateDate{State: toState, Date: day})
	} else {
		story.StateDates[stateDateIndex].Date = day
	}
}

// storyStateAfterCardMove: a card move changes neither the sprints holding the story nor when it
// joined them, so its final state (storyFinalState) follows the card only when the moved sprint is
// the story's latest one.
func storyStateAfterCardMove(story types.Story, sprintID int64, toState int8) int8 {
	if story.LastSprintID == sprintID {
		return toState
	}
	return story.State
}

// storyFinalState is the story's state in the sprint it was most recently added to (ties: the
// highest sprint ID), with that sprint and every sprint holding it. Never planned → not planned.
func storyFinalState(storyID int64, sprints []types.Sprint) (finalState int8, lastSprintID int64, sprintIDs []int64) {
	var latestAddedAt int32 = -1
	for _, sprint := range sprints {
		if sprint.Status != types.StatusActive {
			continue
		}
		for _, sprintStory := range sprint.Stories {
			if sprintStory.StoryID != storyID {
				continue
			}
			sprintIDs = append(sprintIDs, sprint.ID)
			if sprintStory.AddedAt > latestAddedAt || (sprintStory.AddedAt == latestAddedAt && sprint.ID > lastSprintID) {
				latestAddedAt, lastSprintID, finalState = sprintStory.AddedAt, sprint.ID, sprintStory.State
			}
		}
	}
	slices.Sort(sprintIDs)
	return finalState, lastSprintID, sprintIDs
}

// rankBetween is the rank of an item dropped between two neighbours: their midpoint, or one step
// past the only neighbour. Only the moved item is rewritten.
func rankBetween(previousRank float64, hasPrevious bool, nextRank float64, hasNext bool) float64 {
	switch {
	case hasPrevious && hasNext:
		return (previousRank + nextRank) / 2
	case hasPrevious:
		return previousRank + 1
	case hasNext:
		return nextRank - 1
	default:
		return 1
	}
}

// normalizeCommentText validates a comment: text, files or both.
func normalizeCommentText(text string, attachmentsCount int) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" && attachmentsCount == 0 {
		return "", errors.New("El comentario está vacío.")
	}
	if len(text) > 5000 {
		return "", errors.New("Un comentario puede tener hasta 5000 caracteres.")
	}
	return text, nil
}
