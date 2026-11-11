package crm

import (
	"encoding/json"
	"fmt"

	"app/core"
	"app/crm/types"
	"app/db"
)

// GetCrmTasks syncs the tasks of ONE user: ?user-id= (the caller when absent) and ?closed=1 for the
// closed ones. Each combination is its own delta list, so a user downloads only their open tasks and
// a closed task leaves that list as a removed ID. A team leader may read their members' lists.
func GetCrmTasks(req *core.HandlerArgs) core.HandlerResponse {
	assigneeUserID := int32(req.GetQueryInt("user-id"))
	if assigneeUserID == 0 {
		assigneeUserID = req.User.ID
	}
	if assigneeUserID != req.User.ID {
		teams, err := loadActiveTeams(req.User.CompanyID)
		if err != nil {
			return req.MakeErr(err)
		}
		if !types.CanSeeTasksOf(req.User.ID, assigneeUserID, teams) {
			return req.MakeErr("Solo puede ver las tareas de su equipo.")
		}
	}
	taskStatus := types.TaskOpen
	if req.GetQueryInt("closed") == 1 {
		taskStatus = types.TaskClosed
	}

	tasks := []types.CrmTask{}
	query := db.Query(&tasks)
	query.Select().CompanyID.Equals(req.User.CompanyID).AssignedUserID.Equals(assigneeUserID).
		Delta(req.GetDeltaCursor(), taskStatus)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las tareas:", err)
	}
	if err := attachOpportunityCards(req.User.CompanyID, tasks); err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(tasks)
}

// attachOpportunityCards fills in what each task card shows of its opportunity: title, amount and
// client name. Removed IDs of the delta carry no OpportunityID and are left as they are.
func attachOpportunityCards(companyID int32, tasks []types.CrmTask) error {
	opportunityIDs := core.SliceSet[int32]{}
	for _, task := range tasks {
		opportunityIDs.AddIf(task.OpportunityID)
	}
	if opportunityIDs.IsEmpty() {
		return nil
	}
	opportunities := []types.CrmOpportunity{}
	opportunityQuery := db.Query(&opportunities)
	opportunityQuery.Select(opportunityQuery.ID, opportunityQuery.Title, opportunityQuery.ClientID, opportunityQuery.Amount).
		CompanyID.Equals(companyID).ID.In(opportunityIDs.Values...)
	if err := opportunityQuery.Exec(); err != nil {
		return fmt.Errorf("error al obtener las oportunidades de las tareas: %v", err)
	}

	clientIDs := core.SliceSet[int32]{}
	for _, opportunity := range opportunities {
		clientIDs.AddIf(opportunity.ClientID)
	}
	clientNames := map[int32]string{}
	if !clientIDs.IsEmpty() {
		clients := []types.ClientProvider{}
		clientQuery := db.Query(&clients)
		clientQuery.Select(clientQuery.ID, clientQuery.Name).CompanyID.Equals(companyID).ID.In(clientIDs.Values...)
		if err := clientQuery.Exec(); err != nil {
			return fmt.Errorf("error al obtener los clientes de las tareas: %v", err)
		}
		for _, client := range clients {
			clientNames[client.ID] = client.Name
		}
	}

	opportunitiesByID := core.SliceToMapK(opportunities, func(opportunity types.CrmOpportunity) int32 { return opportunity.ID })
	for index := range tasks {
		opportunity, found := opportunitiesByID[tasks[index].OpportunityID]
		if !found {
			continue
		}
		tasks[index].OpportunityTitle = opportunity.Title
		tasks[index].OpportunityAmount = opportunity.Amount
		tasks[index].ClientName = clientNames[opportunity.ClientID]
	}
	return nil
}

// touchPendingTasks re-stamps the open tasks of an opportunity so the tasks delta resends them with
// the opportunity's new title, client or amount. Closed tasks keep what they showed.
func touchPendingTasks(transaction *db.Transaction, companyID, opportunityID, userID int32) error {
	pendingTasks, err := loadPendingTasks(companyID, opportunityID)
	if err != nil || len(pendingTasks) == 0 {
		return err
	}
	for index := range pendingTasks {
		pendingTasks[index].UpdatedBy = userID
	}
	table := db.TableOf[types.CrmTask]()
	return db.UpdateIn(transaction, &pendingTasks, table.UpdatedBy)
}

// GetCrmOpportunityApproval returns the approval task an opportunity waits on (none, or one), for
// users who can see its pipeline. The requester is not the task's assignee, so the tasks list does
// not reach them: the opportunity panel reads it from here.
func GetCrmOpportunityApproval(req *core.HandlerArgs) core.HandlerResponse {
	opportunity, err := authorizeOpportunityView(req, int32(req.GetQueryInt("opportunity-id")))
	if err != nil {
		return req.MakeErr(err)
	}
	tasks := []types.CrmTask{}
	if opportunity.PendingApprovalTaskID == 0 {
		return req.MakeResponse(tasks)
	}
	query := db.Query(&tasks)
	query.CompanyID.Equals(req.User.CompanyID).ID.Equals(opportunity.PendingApprovalTaskID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener la aprobación:", err)
	}
	return req.MakeResponse(tasks)
}

// PostCrmTask acts on one task without producing anything: the assignee dismisses a template task,
// or the requester withdraws an approval request. Doing the task is done through the action's own
// route (an activity: POST.crm-opportunity-activity with the TaskID), which closes it in the same
// transaction; an approval is decided with POST.crm-task-decide.
func PostCrmTask(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmTaskActionPayload{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID, userID := req.User.CompanyID, req.User.ID
	task, err := loadTask(companyID, payload.ID)
	if err != nil {
		return req.MakeErr(err)
	}

	var closedTask types.CrmTask
	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		now := core.SUnixTime()
		switch payload.Action {
		case types.TaskActionDismiss:
			lockedTask, err := lockTask(transaction, companyID, task.ID)
			if err != nil {
				return err
			}
			if closedTask, err = types.PlanTaskDismissal(lockedTask, payload.Note, userID, now); err != nil {
				return err
			}
			return updateClosedTask(transaction, closedTask)
		case types.TaskActionCancelRequest:
			// The opportunity is locked first, as in every path that changes both.
			opportunity, err := lockOpportunity(transaction, companyID, task.OpportunityID)
			if err != nil {
				return err
			}
			lockedTask, err := lockTask(transaction, companyID, task.ID)
			if err != nil {
				return err
			}
			if closedTask, err = types.PlanApprovalCancel(lockedTask, opportunity, userID, now); err != nil {
				return err
			}
			if err := updateClosedTask(transaction, closedTask); err != nil {
				return err
			}
			return clearPendingApproval(transaction, opportunity, userID)
		}
		return fmt.Errorf("La acción %d no es válida.", payload.Action)
	})
	if saveError != nil {
		return req.MakeErr("Error al guardar la tarea:", saveError)
	}
	return req.MakeResponse(closedTask)
}

// PostCrmTaskDecide lets an approval's approver approve or reject it. Approving runs the move the
// request was holding (as the approver, who need not be a member of the pipeline); rejecting leaves
// the opportunity where it is. Either way a note, when given, reaches the opportunity's timeline.
func PostCrmTaskDecide(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmTaskDecisionPayload{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID, userID := req.User.CompanyID, req.User.ID
	task, err := loadTask(companyID, payload.ID)
	if err != nil {
		return req.MakeErr(err)
	}

	var decidedTask types.CrmTask
	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		now := core.SUnixTime()
		opportunity, err := lockOpportunity(transaction, companyID, task.OpportunityID)
		if err != nil {
			return err
		}
		lockedTask, err := lockTask(transaction, companyID, task.ID)
		if err != nil {
			return err
		}
		if decidedTask, err = types.PlanApprovalDecision(lockedTask, opportunity, payload.Approve, payload.Note, userID, now); err != nil {
			return err
		}
		if err := updateClosedTask(transaction, decidedTask); err != nil {
			return err
		}

		if payload.Approve {
			source, err := loadPipeline(companyID, opportunity.PipelineID)
			if err != nil {
				return err
			}
			target := source
			if lockedTask.PipelineID != source.ID {
				if target, err = loadPipeline(companyID, lockedTask.PipelineID); err != nil {
					return err
				}
			}
			moved, stageLog, err := types.PlanApprovedMove(opportunity, lockedTask, source, target, userID, now)
			if err != nil {
				return err
			}
			if err := persistStageChange(transaction, moved, stageLog, target, userID, now); err != nil {
				return err
			}
		} else if err := clearPendingApproval(transaction, opportunity, userID); err != nil {
			// A rejection never needs the target pipeline to still exist: it only releases the card.
			return err
		}

		// A rejection always explains itself on the timeline; an approval does when it has a note.
		if payload.Approve && decidedTask.ResultNote == "" {
			return nil
		}
		stageName := stageNameOf(companyID, lockedTask.PipelineID, lockedTask.StageID)
		return insertTimelineNote(transaction, opportunity, types.ApprovalTimelineText(payload.Approve, stageName, decidedTask.ResultNote), userID, now)
	})
	if saveError != nil {
		return req.MakeErr("Error al decidir la aprobación:", saveError)
	}
	return req.MakeResponse(decidedTask)
}

// requestApproval creates the approval task of a move into a gated stage and parks the opportunity
// on it. The approver must still be an active user, or the task would wait on nobody.
func requestApproval(transaction *db.Transaction, opportunity types.CrmOpportunity, target types.CrmPipeline,
	targetStageID int16, approverUserID, requesterID, now int32) (types.CrmOpportunity, error) {

	activeUserIDs, err := loadActiveUserIDs(opportunity.CompanyID, []int32{approverUserID})
	if err != nil {
		return types.CrmOpportunity{}, err
	}
	if !activeUserIDs[approverUserID] {
		return types.CrmOpportunity{}, fmt.Errorf("El aprobador de la etapa está inactivo; pida a un administrador que lo cambie.")
	}
	tasks := []types.CrmTask{types.NewApprovalTask(opportunity, target, targetStageID, requesterID, now)}
	if err := insertTasks(transaction, tasks, requesterID); err != nil {
		return types.CrmOpportunity{}, err
	}
	opportunity.PendingApprovalTaskID, opportunity.UpdatedBy = tasks[0].ID, requesterID
	records := []types.CrmOpportunity{opportunity}
	table := db.TableOf[types.CrmOpportunity]()
	if err := db.UpdateIn(transaction, &records, table.PendingApprovalTaskID, table.UpdatedBy); err != nil {
		return types.CrmOpportunity{}, err
	}
	return records[0], nil
}

// stageNameOf is the name of a stage for a message; empty when the pipeline or stage is gone.
func stageNameOf(companyID int32, pipelineID, stageID int16) string {
	pipeline, err := loadPipeline(companyID, pipelineID)
	if err != nil {
		return ""
	}
	stage, _ := types.FindActiveStage(pipeline, stageID)
	return stage.Name
}

// persistStageChange writes a move that happens now: the opportunity, its log row, the tasks it
// leaves behind and the ones its new stage proposes. The caller holds the transaction.
func persistStageChange(transaction *db.Transaction, moved types.CrmOpportunity, stageLog types.CrmOpportunityStageLog,
	target types.CrmPipeline, userID, now int32) error {

	records := []types.CrmOpportunity{moved}
	table := db.TableOf[types.CrmOpportunity]()
	if err := db.UpdateIn(transaction, &records, table.PipelineID, table.StageID, table.Probability, table.StageEnteredTime,
		table.PreviousPipelineID, table.PreviousStageID, table.PendingApprovalTaskID, table.UpdatedBy); err != nil {
		return err
	}
	if err := insertStageLogs(transaction, moved.CompanyID, []types.CrmOpportunityStageLog{stageLog}); err != nil {
		return err
	}
	return syncStageTasks(transaction, moved, target, moved.StageID, userID, now)
}

// syncStageTasks cancels the pending template tasks the opportunity left behind and proposes the
// ones of the stage it now sits in. stageID 0 means it left every stage (lost): nothing is proposed.
func syncStageTasks(transaction *db.Transaction, opportunity types.CrmOpportunity, pipeline types.CrmPipeline,
	stageID int16, userID, now int32) error {

	pendingTasks, err := loadPendingTasks(opportunity.CompanyID, opportunity.ID)
	if err != nil {
		return err
	}
	leftBehind := types.TasksLeftBehind(pendingTasks, pipeline, stageID)
	for _, task := range leftBehind {
		if err := updateClosedTask(transaction, types.CancelTask(task, userID, now)); err != nil {
			return err
		}
	}
	if stageID == 0 {
		return nil
	}
	stillPending := []types.CrmTask{}
	for _, task := range pendingTasks {
		if !containsTask(leftBehind, task.ID) {
			stillPending = append(stillPending, task)
		}
	}
	return insertTasks(transaction, types.StageEntryTasks(opportunity, pipeline, stageID, stillPending, now), userID)
}

func containsTask(tasks []types.CrmTask, taskID int32) bool {
	for _, task := range tasks {
		if task.ID == taskID {
			return true
		}
	}
	return false
}

func insertTasks(transaction *db.Transaction, tasks []types.CrmTask, userID int32) error {
	if len(tasks) == 0 {
		return nil
	}
	for index := range tasks {
		tasks[index].UpdatedBy = userID
	}
	return db.InsertIn(transaction, &tasks)
}

// updateClosedTask writes the fields closing a task changes.
func updateClosedTask(transaction *db.Transaction, closedTask types.CrmTask) error {
	records := []types.CrmTask{closedTask}
	table := db.TableOf[types.CrmTask]()
	return db.UpdateIn(transaction, &records, table.TaskStatus, table.ResultNote, table.ResultRefID,
		table.DoneTime, table.DoneBy, table.Status, table.UpdatedBy)
}

// clearPendingApproval releases an opportunity that waited on an approval.
func clearPendingApproval(transaction *db.Transaction, opportunity types.CrmOpportunity, userID int32) error {
	opportunity.PendingApprovalTaskID, opportunity.UpdatedBy = 0, userID
	records := []types.CrmOpportunity{opportunity}
	table := db.TableOf[types.CrmOpportunity]()
	return db.UpdateIn(transaction, &records, table.PendingApprovalTaskID, table.UpdatedBy)
}

// insertTimelineNote adds a system-written note to the opportunity's timeline, authored by the user.
func insertTimelineNote(transaction *db.Transaction, opportunity types.CrmOpportunity, text string, userID, now int32) error {
	note, err := types.PlanActivity(types.CrmActivity{OpportunityID: opportunity.ID, TypeID: types.ActivityTypeNote, Text: text},
		opportunity.CompanyID, opportunity.PipelineID, false, userID, now)
	if err != nil {
		return err
	}
	records := []types.CrmActivity{note}
	return db.InsertIn(transaction, &records)
}

func loadTask(companyID, taskID int32) (types.CrmTask, error) {
	tasks := []types.CrmTask{}
	query := db.Query(&tasks)
	query.CompanyID.Equals(companyID).ID.Equals(taskID)
	if err := query.Exec(); err != nil {
		return types.CrmTask{}, fmt.Errorf("error al leer la tarea: %v", err)
	}
	if len(tasks) == 0 {
		return types.CrmTask{}, fmt.Errorf("La tarea %d no existe.", taskID)
	}
	return tasks[0], nil
}

// lockTask reads the stored task under lock, so two decisions on the same task are serialized.
func lockTask(transaction *db.Transaction, companyID, taskID int32) (types.CrmTask, error) {
	records := []types.CrmTask{{CompanyID: companyID, ID: taskID}}
	if err := db.LockByKeys(transaction, &records); err != nil {
		return types.CrmTask{}, err
	}
	if len(records) == 0 {
		return types.CrmTask{}, fmt.Errorf("La tarea %d no existe.", taskID)
	}
	return records[0], nil
}

// loadPendingTasks returns the opportunity's tasks that are still pending.
func loadPendingTasks(companyID, opportunityID int32) ([]types.CrmTask, error) {
	tasks := []types.CrmTask{}
	query := db.Query(&tasks)
	query.CompanyID.Equals(companyID).OpportunityID.Equals(opportunityID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer las tareas de la oportunidad: %v", err)
	}
	pendingTasks := make([]types.CrmTask, 0, len(tasks))
	for _, task := range tasks {
		if task.TaskStatus == types.TaskStatusPending {
			pendingTasks = append(pendingTasks, task)
		}
	}
	return pendingTasks, nil
}
