package types

import (
	"errors"
	"fmt"
	"strings"

	"app/core"
)

const maxTaskNoteLength = 500

// ---------------------------------------------------------------------------------------------
// Stage templates -> tasks (plan §5). Entering a stage proposes its tasks; they never run alone.
// ---------------------------------------------------------------------------------------------

// StageEntryTasks returns the pending tasks to insert when the opportunity enters the stage: one
// per template, assigned to the seller or the template's fixed user. A template that already has a
// pending task of this opportunity in the stage is skipped, so stepping out and back into a stage
// does not duplicate what is still waiting.
func StageEntryTasks(opportunity CrmOpportunity, pipeline CrmPipeline, stageID int16, pendingTasks []CrmTask, now int32) []CrmTask {
	stage, isActive := FindActiveStage(pipeline, stageID)
	if !isActive {
		return nil
	}
	today := core.SUnixTimeToUnixDay(now)
	tasks := []CrmTask{}
	for _, template := range stage.TaskTemplates {
		if hasPendingTemplateTask(pendingTasks, pipeline.ID, stageID, template.ID) {
			continue
		}
		assignedUserID := opportunity.OwnerUserID
		if template.AssignTo == TaskAssignUser {
			assignedUserID = template.AssignUserID
		}
		tasks = append(tasks, CrmTask{
			CompanyID: opportunity.CompanyID, OpportunityID: opportunity.ID, PipelineID: pipeline.ID, StageID: stageID,
			Kind: template.Kind, TemplateID: template.ID, Title: template.Title, AssignedUserID: assignedUserID,
			DueDate: today + template.DueDays, TaskStatus: TaskStatusPending, Created: now,
		})
	}
	return tasks
}

func hasPendingTemplateTask(pendingTasks []CrmTask, pipelineID, stageID, templateID int16) bool {
	for _, task := range pendingTasks {
		if task.Kind != TaskKindApproval && task.TaskStatus == TaskStatusPending &&
			task.PipelineID == pipelineID && task.StageID == stageID && task.TemplateID == templateID {
			return true
		}
	}
	return false
}

// TasksLeftBehind returns the pending template tasks an opportunity no longer owes after it ends in
// the target pipeline and stage: the ones of another pipeline, of a stage deeper than the target
// (it stepped back), or of a stage that no longer exists. targetStageID 0 means it left every stage
// (lost), so all of them. Approvals are not template tasks and are never cancelled here.
func TasksLeftBehind(pendingTasks []CrmTask, target CrmPipeline, targetStageID int16) []CrmTask {
	targetPosition := StagePosition(target, targetStageID)
	leftBehind := []CrmTask{}
	for _, task := range pendingTasks {
		if task.Kind == TaskKindApproval || task.TaskStatus != TaskStatusPending {
			continue
		}
		taskPosition := StagePosition(target, task.StageID)
		if targetStageID == 0 || task.PipelineID != target.ID || taskPosition == 0 || taskPosition > targetPosition {
			leftBehind = append(leftBehind, task)
		}
	}
	return leftBehind
}

// ---------------------------------------------------------------------------------------------
// Approvals (plan §4.2)
// ---------------------------------------------------------------------------------------------

// ApprovalGate says who must approve a move of the opportunity into the target stage: 0 when it
// needs nobody. A forward move cannot skip a stage that has an approver (P1); a backward move
// never needs approval (P2); entering a gated stage from another pipeline does (the stages it
// never sat in are not "skipped").
func ApprovalGate(opportunity CrmOpportunity, source, target CrmPipeline, targetStageID int16) (int32, error) {
	targetStage, isActive := FindActiveStage(target, targetStageID)
	if !isActive {
		return 0, fmt.Errorf("La etapa de destino no existe o está desactivada.")
	}
	if opportunity.PipelineID == target.ID {
		currentPosition, targetPosition := StagePosition(source, opportunity.StageID), StagePosition(target, targetStageID)
		if targetPosition <= currentPosition {
			return 0, nil
		}
		position := int8(0)
		for _, stage := range target.Stages {
			if stage.Status != CrmStatusActive {
				continue
			}
			position++
			if position > currentPosition && position < targetPosition && stage.ApproverUserID != 0 {
				return 0, fmt.Errorf("Debe pasar primero por la etapa %q, que requiere aprobación.", stage.Name)
			}
		}
	}
	return targetStage.ApproverUserID, nil
}

// NewApprovalTask is the task a gated move waits on, assigned to the stage's approver. It carries
// the target pipeline and stage; the opportunity does not move until the approver decides.
func NewApprovalTask(opportunity CrmOpportunity, target CrmPipeline, targetStageID int16, requesterID, now int32) CrmTask {
	targetStage, _ := FindActiveStage(target, targetStageID)
	return CrmTask{
		CompanyID: opportunity.CompanyID, OpportunityID: opportunity.ID, PipelineID: target.ID, StageID: targetStageID,
		Kind: TaskKindApproval, AssignedUserID: targetStage.ApproverUserID, DueDate: core.SUnixTimeToUnixDay(now),
		TaskStatus: TaskStatusPending, Created: now, CreatedBy: requesterID,
	}
}

// PlanApprovalDecision closes an approval task as approved or rejected. Only its approver decides,
// and only while the opportunity still waits on it. An approved task is followed by PlanApprovedMove.
func PlanApprovalDecision(task CrmTask, opportunity CrmOpportunity, approve bool, note string, userID, now int32) (CrmTask, error) {
	if task.Kind != TaskKindApproval {
		return CrmTask{}, errors.New("La tarea no es una aprobación.")
	}
	if task.TaskStatus != TaskStatusPending || opportunity.PendingApprovalTaskID != task.ID {
		return CrmTask{}, errors.New("La solicitud de aprobación ya no está vigente.")
	}
	if task.AssignedUserID != userID {
		return CrmTask{}, errors.New("Solo el aprobador puede decidir esta solicitud.")
	}
	note, err := normalizeTaskNote(note)
	if err != nil {
		return CrmTask{}, err
	}
	taskStatus := TaskStatusRejected
	if approve {
		taskStatus = TaskStatusApproved
	}
	return closeTask(task, taskStatus, note, userID, now), nil
}

// PlanApprovalCancel withdraws an approval request. Only whoever asked can.
func PlanApprovalCancel(task CrmTask, opportunity CrmOpportunity, userID, now int32) (CrmTask, error) {
	if task.Kind != TaskKindApproval || task.TaskStatus != TaskStatusPending || opportunity.PendingApprovalTaskID != task.ID {
		return CrmTask{}, errors.New("La solicitud de aprobación ya no está vigente.")
	}
	if task.CreatedBy != userID {
		return CrmTask{}, errors.New("Solo quien pidió la aprobación puede cancelarla.")
	}
	return closeTask(task, TaskStatusCancelled, "", userID, now), nil
}

// PlanApprovedMove is the move an approver's OK releases. The requester's rights were checked when
// the request was made, and the approver need not be a member of the pipeline (approving gives no
// board access), so only the state is checked again: still open, still waiting on this task, the
// target stage still active and its required data complete. userID is the approver, who becomes the
// stage log row's author.
func PlanApprovedMove(opportunity CrmOpportunity, task CrmTask, source, target CrmPipeline, userID, now int32) (CrmOpportunity, CrmOpportunityStageLog, error) {
	if opportunity.Outcome != OpportunityOutcomeOpen || opportunity.PendingApprovalTaskID != task.ID {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, errors.New("La oportunidad ya no espera esta aprobación.")
	}
	stage, err := validateMoveTarget(opportunity, target, task.StageID)
	if err != nil {
		return CrmOpportunity{}, CrmOpportunityStageLog{}, err
	}
	moved, stageLog := applyMove(opportunity, source, target, stage, userID, now)
	return moved, stageLog, nil
}

// ApprovalTimelineText is the entry an approver's decision leaves on the opportunity's timeline.
func ApprovalTimelineText(approved bool, stageName, note string) string {
	text := fmt.Sprintf("Rechazó el paso a la etapa «%s».", stageName)
	if approved {
		text = fmt.Sprintf("Aprobó el paso a la etapa «%s».", stageName)
	}
	if note != "" {
		text += " " + note
	}
	return text
}

// ---------------------------------------------------------------------------------------------
// Template tasks
// ---------------------------------------------------------------------------------------------

// PlanTaskDismissal closes a template task the assignee decides not to do. Approvals are decided,
// not dismissed.
func PlanTaskDismissal(task CrmTask, note string, userID, now int32) (CrmTask, error) {
	if task.Kind == TaskKindApproval {
		return CrmTask{}, errors.New("Una aprobación se aprueba o se rechaza, no se descarta.")
	}
	if task.TaskStatus != TaskStatusPending {
		return CrmTask{}, errors.New("La tarea ya está cerrada.")
	}
	if task.AssignedUserID != userID {
		return CrmTask{}, errors.New("Solo la persona asignada puede descartar la tarea.")
	}
	note, err := normalizeTaskNote(note)
	if err != nil {
		return CrmTask{}, err
	}
	return closeTask(task, TaskStatusDismissed, note, userID, now), nil
}

// PlanActivityTaskDone closes a task because the assignee did what it asked for: logged the activity
// (TaskKindActivity) or sent the email, which is logged as an activity too (TaskKindSendEmail). It
// runs in the same transaction as the activity, so the task closes exactly when it is done. doneKind
// is what the caller did: a hand-written note must not close a "send email" task.
func PlanActivityTaskDone(task CrmTask, activity CrmActivity, doneKind int8, userID, now int32) (CrmTask, error) {
	if task.TaskStatus != TaskStatusPending {
		return CrmTask{}, errors.New("La tarea ya está cerrada.")
	}
	if task.Kind != doneKind {
		if task.Kind == TaskKindSendEmail {
			return CrmTask{}, errors.New("Esta tarea se completa enviando el correo.")
		}
		return CrmTask{}, errors.New("La tarea no es una actividad pendiente.")
	}
	if task.AssignedUserID != userID {
		return CrmTask{}, errors.New("Solo la persona asignada puede completar la tarea.")
	}
	if task.OpportunityID != activity.OpportunityID {
		return CrmTask{}, errors.New("La actividad debe registrarse en la oportunidad de la tarea.")
	}
	done := closeTask(task, TaskStatusDone, "", userID, now)
	done.ResultRefID = int64(activity.ID)
	return done, nil
}

// CancelTask closes a pending task because the opportunity moved on (plan P3).
func CancelTask(task CrmTask, userID, now int32) CrmTask {
	return closeTask(task, TaskStatusCancelled, "", userID, now)
}

func closeTask(task CrmTask, taskStatus int8, note string, userID, now int32) CrmTask {
	task.TaskStatus, task.ResultNote, task.DoneTime, task.DoneBy, task.UpdatedBy = taskStatus, note, now, userID, userID
	task.SelfParse()
	return task
}

func normalizeTaskNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if len(note) > maxTaskNoteLength {
		return "", fmt.Errorf("La nota puede tener hasta %d caracteres.", maxTaskNoteLength)
	}
	return note, nil
}

// ---------------------------------------------------------------------------------------------
// Who sees whose tasks
// ---------------------------------------------------------------------------------------------

// CanSeeTasksOf: a user sees their own tasks, and a team leader sees the tasks of their team's
// members (read only: acting stays with the assignee). teams are the company's active teams.
func CanSeeTasksOf(viewerUserID, assigneeUserID int32, teams []CrmTeam) bool {
	if viewerUserID == assigneeUserID {
		return true
	}
	for _, team := range teams {
		if team.Status != CrmStatusActive || team.LeaderUserID != viewerUserID {
			continue
		}
		for _, memberUserID := range team.MemberUserIDs {
			if memberUserID == assigneeUserID {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------------------------
// Stage definition checks (called by ApplyPipelineEdit and the handler)
// ---------------------------------------------------------------------------------------------

// normalizeTaskTemplates validates a stage's templates and allocates the IDs of the new ones
// (max+1 inside the stage). The user and activity-type references are checked by
// ValidatePipelineReferences, which needs the database.
func normalizeTaskTemplates(templates []CrmStageTaskTemplate, storedTemplates []CrmStageTaskTemplate, stageName string) ([]CrmStageTaskTemplate, error) {
	storedIDs := map[int16]bool{}
	nextTemplateID := int16(1)
	for _, storedTemplate := range storedTemplates {
		storedIDs[storedTemplate.ID] = true
		nextTemplateID = max(nextTemplateID, storedTemplate.ID+1)
	}
	seenIDs := map[int16]bool{}
	saved := make([]CrmStageTaskTemplate, 0, len(templates))
	for _, template := range templates {
		template.Title = strings.TrimSpace(template.Title)
		if template.Title == "" || len(template.Title) > 120 {
			return nil, fmt.Errorf("Toda tarea de la etapa %q necesita un título de hasta 120 caracteres.", stageName)
		}
		// Each kind keeps only its own reference, so a template switched from one kind to the other
		// carries no stale ID the reference check would trip on.
		switch template.Kind {
		case TaskKindActivity:
			template.EmailTemplateID = 0
		case TaskKindSendEmail:
			if template.EmailTemplateID <= 0 {
				return nil, fmt.Errorf("La tarea %q necesita una plantilla de correo.", template.Title)
			}
			template.ActivityTypeID = ActivityTypeEmail
		default:
			return nil, fmt.Errorf("El tipo de tarea %d de la etapa %q no es válido (3 enviar correo, 4 actividad).", template.Kind, stageName)
		}
		if template.DueDays < 0 || template.DueDays > 365 {
			return nil, fmt.Errorf("Los días de plazo de la tarea %q deben estar entre 0 y 365.", template.Title)
		}
		if template.AssignTo != TaskAssignOwner && template.AssignTo != TaskAssignUser {
			return nil, fmt.Errorf("La asignación de la tarea %q no es válida (1 responsable, 2 usuario fijo).", template.Title)
		}
		if template.AssignTo == TaskAssignOwner {
			template.AssignUserID = 0
		} else if template.AssignUserID <= 0 {
			return nil, fmt.Errorf("La tarea %q necesita un usuario asignado.", template.Title)
		}
		if template.ID == 0 {
			template.ID = nextTemplateID
			nextTemplateID++
		} else if !storedIDs[template.ID] {
			return nil, fmt.Errorf("La tarea %d no existe en la etapa %q.", template.ID, stageName)
		}
		if seenIDs[template.ID] {
			return nil, fmt.Errorf("La tarea %d está repetida en la etapa %q.", template.ID, stageName)
		}
		seenIDs[template.ID] = true
		saved = append(saved, template)
	}
	return saved, nil
}

// PipelineReferences are the users, activity types and email templates a pipeline's stages may point at.
type PipelineReferences struct {
	ActiveUserIDs          map[int32]bool
	ActiveActivityTypeIDs  map[int16]bool // the company's own types; the system ones are always valid
	ActiveEmailTemplateIDs map[int16]bool
}

// PipelineReferenceIDs lists the users, company activity types and email templates a pipeline's
// active stages name, so the handler can load only those.
func PipelineReferenceIDs(pipeline CrmPipeline) (userIDs []int32, activityTypeIDs []int16, emailTemplateIDs []int16) {
	for _, stage := range pipeline.Stages {
		if stage.Status != CrmStatusActive {
			continue
		}
		if stage.ApproverUserID != 0 {
			userIDs = append(userIDs, stage.ApproverUserID)
		}
		for _, template := range stage.TaskTemplates {
			if template.AssignTo == TaskAssignUser {
				userIDs = append(userIDs, template.AssignUserID)
			}
			if template.ActivityTypeID > ActivityTypeSystemMaxID {
				activityTypeIDs = append(activityTypeIDs, template.ActivityTypeID)
			}
			if template.EmailTemplateID > 0 {
				emailTemplateIDs = append(emailTemplateIDs, template.EmailTemplateID)
			}
		}
	}
	return userIDs, activityTypeIDs, emailTemplateIDs
}

// ValidatePipelineReferences checks that every approver, fixed assignee and activity type a
// pipeline's active stages name exists and is active.
func ValidatePipelineReferences(pipeline CrmPipeline, references PipelineReferences) error {
	for _, stage := range pipeline.Stages {
		if stage.Status != CrmStatusActive {
			continue
		}
		if stage.ApproverUserID < 0 || (stage.ApproverUserID != 0 && !references.ActiveUserIDs[stage.ApproverUserID]) {
			return fmt.Errorf("El aprobador de la etapa %q no existe o está inactivo.", stage.Name)
		}
		for _, template := range stage.TaskTemplates {
			if template.AssignTo == TaskAssignUser && !references.ActiveUserIDs[template.AssignUserID] {
				return fmt.Errorf("El usuario asignado a la tarea %q no existe o está inactivo.", template.Title)
			}
			isSystemType := isSystemActivityType(template.ActivityTypeID)
			if !isSystemType && !references.ActiveActivityTypeIDs[template.ActivityTypeID] {
				return fmt.Errorf("El tipo de actividad de la tarea %q no existe.", template.Title)
			}
			if template.Kind == TaskKindSendEmail && !references.ActiveEmailTemplateIDs[template.EmailTemplateID] {
				return fmt.Errorf("La plantilla de correo de la tarea %q no existe.", template.Title)
			}
		}
	}
	return nil
}
