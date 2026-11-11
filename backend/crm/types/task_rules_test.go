package types

import (
	"testing"

	"app/core"
)

// gatedPipeline has four active stages; stage 3 needs approval from user 20.
func gatedPipeline(id int16) CrmPipeline {
	pipeline := CrmPipeline{ID: id, Status: CrmStatusActive, Members: []CrmPipelineMember{{UserID: 7, Level: PipelineLevelEdit}}}
	for stageID := int16(1); stageID <= 4; stageID++ {
		pipeline.Stages = append(pipeline.Stages, CrmPipelineStage{ID: stageID, Name: "E", Probability: 10 * int8(stageID), Status: CrmStatusActive})
	}
	pipeline.Stages[2].ApproverUserID = 20
	return pipeline
}

func TestStageEntryTasksAssignsAndDoesNotDuplicate(t *testing.T) {
	pipeline := gatedPipeline(1)
	pipeline.Stages[1].TaskTemplates = []CrmStageTaskTemplate{
		{ID: 1, Kind: TaskKindActivity, Title: "Llamar", ActivityTypeID: ActivityTypeCall, DueDays: 2, AssignTo: TaskAssignOwner},
		{ID: 2, Kind: TaskKindActivity, Title: "Visita", ActivityTypeID: ActivityTypeVisit, AssignTo: TaskAssignUser, AssignUserID: 9},
	}
	opportunity := CrmOpportunity{CompanyID: 1, ID: 5, OwnerUserID: 7}
	now := int32(100_000_000)

	tasks := StageEntryTasks(opportunity, pipeline, 2, nil, now)
	today := core.SUnixTimeToUnixDay(now)
	if len(tasks) != 2 || tasks[0].AssignedUserID != 7 || tasks[0].DueDate != today+2 || tasks[1].AssignedUserID != 9 || tasks[1].DueDate != today {
		t.Fatalf("tasks of the stage: %+v", tasks)
	}
	if tasks[0].TaskStatus != TaskStatusPending || tasks[0].PipelineID != 1 || tasks[0].StageID != 2 || tasks[0].TemplateID != 1 || tasks[0].OpportunityID != 5 {
		t.Fatalf("task fields: %+v", tasks[0])
	}

	// Re-entering a stage whose task is still waiting does not propose it twice.
	again := StageEntryTasks(opportunity, pipeline, 2, tasks[:1], now)
	if len(again) != 1 || again[0].TemplateID != 2 {
		t.Fatalf("only the template with no pending task is proposed: %+v", again)
	}
	if len(StageEntryTasks(opportunity, pipeline, 3, nil, now)) != 0 || len(StageEntryTasks(opportunity, pipeline, 9, nil, now)) != 0 {
		t.Fatal("a stage with no templates, or a missing one, proposes nothing")
	}
}

func TestTasksLeftBehind(t *testing.T) {
	pipeline := gatedPipeline(1)
	pending := []CrmTask{
		{ID: 1, Kind: TaskKindActivity, PipelineID: 1, StageID: 1, TaskStatus: TaskStatusPending},
		{ID: 2, Kind: TaskKindActivity, PipelineID: 1, StageID: 3, TaskStatus: TaskStatusPending},
		{ID: 3, Kind: TaskKindActivity, PipelineID: 2, StageID: 1, TaskStatus: TaskStatusPending},
		{ID: 4, Kind: TaskKindApproval, PipelineID: 1, StageID: 4, TaskStatus: TaskStatusPending},
		{ID: 5, Kind: TaskKindActivity, PipelineID: 1, StageID: 99, TaskStatus: TaskStatusPending},
		{ID: 6, Kind: TaskKindActivity, PipelineID: 1, StageID: 4, TaskStatus: TaskStatusDone},
	}
	idsOf := func(tasks []CrmTask) []int32 {
		ids := []int32{}
		for _, task := range tasks {
			ids = append(ids, task.ID)
		}
		return ids
	}
	// Stepping back to stage 2 drops stage 3's task, the other pipeline's and the one of a stage that is gone.
	if got := idsOf(TasksLeftBehind(pending, pipeline, 2)); len(got) != 3 || got[0] != 2 || got[1] != 3 || got[2] != 5 {
		t.Fatalf("stepping back: %v", got)
	}
	// Moving forward keeps what is earlier.
	if got := idsOf(TasksLeftBehind(pending, pipeline, 4)); len(got) != 2 || got[0] != 3 || got[1] != 5 {
		t.Fatalf("moving forward: %v", got)
	}
	// Lost leaves every stage: every template task goes, never an approval or a closed task.
	if got := idsOf(TasksLeftBehind(pending, pipeline, 0)); len(got) != 4 {
		t.Fatalf("lost: %v", got)
	}
}

func TestApprovalGate(t *testing.T) {
	pipeline := gatedPipeline(1)
	inStage := func(stageID int16) CrmOpportunity { return CrmOpportunity{PipelineID: 1, StageID: stageID} }

	if approver, err := ApprovalGate(inStage(2), pipeline, pipeline, 3); err != nil || approver != 20 {
		t.Fatalf("a forward move into the gated stage needs its approver: %d %v", approver, err)
	}
	if approver, err := ApprovalGate(inStage(1), pipeline, pipeline, 2); err != nil || approver != 0 {
		t.Fatalf("an ungated stage needs nobody: %d %v", approver, err)
	}
	if _, err := ApprovalGate(inStage(1), pipeline, pipeline, 4); err == nil {
		t.Fatal("going from 1 to 4 skips the gate of stage 3")
	}
	if approver, err := ApprovalGate(inStage(4), pipeline, pipeline, 3); err != nil || approver != 0 {
		t.Fatalf("a backward move into a gated stage needs no approval: %d %v", approver, err)
	}
	// From another pipeline, entering the gated stage needs approval; the stages never sat in are not skipped.
	other := CrmOpportunity{PipelineID: 2, StageID: 1}
	if approver, err := ApprovalGate(other, CrmPipeline{ID: 2}, pipeline, 3); err != nil || approver != 20 {
		t.Fatalf("entering a gated stage from another pipeline: %d %v", approver, err)
	}
	if _, err := ApprovalGate(inStage(1), pipeline, pipeline, 9); err == nil {
		t.Fatal("a missing target stage is refused")
	}
}

func TestApprovalLifecycle(t *testing.T) {
	pipeline := gatedPipeline(1)
	opportunity := CrmOpportunity{CompanyID: 1, ID: 5, PipelineID: 1, StageID: 2, OwnerUserID: 7, Amount: 50}
	task := NewApprovalTask(opportunity, pipeline, 3, 7, 1000)
	task.ID = 44
	opportunity.PendingApprovalTaskID = 44
	if task.Kind != TaskKindApproval || task.AssignedUserID != 20 || task.StageID != 3 || task.PipelineID != 1 || task.CreatedBy != 7 || task.TaskStatus != TaskStatusPending {
		t.Fatalf("approval task: %+v", task)
	}

	// Nothing moves or closes while the request waits.
	if _, _, err := PlanOpportunityMove(opportunity, pipeline, pipeline, 1, 7, 2000); err == nil {
		t.Fatal("a card waiting on an approval cannot be moved")
	}
	if _, _, err := PlanOpportunityClose(opportunity, pipeline, CrmOpportunityClosePayload{Outcome: OpportunityOutcomeWon}, false, 7, 2000); err == nil {
		t.Fatal("a card waiting on an approval cannot be closed")
	}

	if _, err := PlanApprovalDecision(task, opportunity, true, "", 7, 2000); err == nil {
		t.Fatal("only the approver decides")
	}
	if _, err := PlanApprovalDecision(task, CrmOpportunity{ID: 5}, true, "", 20, 2000); err == nil {
		t.Fatal("a request the card no longer waits on cannot be decided")
	}
	approved, err := PlanApprovalDecision(task, opportunity, true, "  ok  ", 20, 2000)
	if err != nil || approved.TaskStatus != TaskStatusApproved || approved.ResultNote != "ok" || approved.DoneBy != 20 || approved.Status != TaskClosed {
		t.Fatalf("approval: %+v %v", approved, err)
	}
	rejected, _ := PlanApprovalDecision(task, opportunity, false, "", 20, 2000)
	if rejected.TaskStatus != TaskStatusRejected {
		t.Fatalf("rejection: %+v", rejected)
	}

	// The approved move runs as the approver, who is not a member of the pipeline.
	moved, stageLog, err := PlanApprovedMove(opportunity, task, pipeline, pipeline, 20, 2000)
	if err != nil || moved.StageID != 3 || moved.PendingApprovalTaskID != 0 || moved.Probability != 30 || moved.StageEnteredTime != 2000 {
		t.Fatalf("approved move: %+v %v", moved, err)
	}
	if stageLog.UserID != 20 || stageLog.FromStageID != 2 || stageLog.ToStageID != 3 || stageLog.Kind != StageLogKindMoved {
		t.Fatalf("the approver authors the log row: %+v", stageLog)
	}
	// Required data is checked again at approval time.
	pipeline.Stages[2].RequiredStandardFields = []int8{StandardFieldExpectedCloseDate}
	if _, _, err := PlanApprovedMove(opportunity, task, pipeline, pipeline, 20, 2000); err == nil {
		t.Fatal("a move released by an approval still needs the stage's required data")
	}

	cancelled, err := PlanApprovalCancel(task, opportunity, 7, 2000)
	if err != nil || cancelled.TaskStatus != TaskStatusCancelled {
		t.Fatalf("the requester cancels: %+v %v", cancelled, err)
	}
	if _, err := PlanApprovalCancel(task, opportunity, 20, 2000); err == nil {
		t.Fatal("only the requester cancels")
	}
}

func TestTemplateTaskDismissalAndCompletion(t *testing.T) {
	task := CrmTask{ID: 8, OpportunityID: 5, Kind: TaskKindActivity, AssignedUserID: 7, TaskStatus: TaskStatusPending}

	if _, err := PlanTaskDismissal(task, "", 9, 10); err == nil {
		t.Fatal("only the assignee dismisses")
	}
	dismissed, err := PlanTaskDismissal(task, " no aplica ", 7, 10)
	if err != nil || dismissed.TaskStatus != TaskStatusDismissed || dismissed.ResultNote != "no aplica" || dismissed.Status != TaskClosed {
		t.Fatalf("dismissal: %+v %v", dismissed, err)
	}
	if _, err := PlanTaskDismissal(CrmTask{Kind: TaskKindApproval, AssignedUserID: 7, TaskStatus: TaskStatusPending}, "", 7, 10); err == nil {
		t.Fatal("an approval is decided, not dismissed")
	}

	done, err := PlanActivityTaskDone(task, CrmActivity{ID: 31, OpportunityID: 5}, TaskKindActivity, 7, 10)
	if err != nil || done.TaskStatus != TaskStatusDone || done.ResultRefID != 31 {
		t.Fatalf("completion: %+v %v", done, err)
	}
	if _, err := PlanActivityTaskDone(task, CrmActivity{ID: 31, OpportunityID: 6}, TaskKindActivity, 7, 10); err == nil {
		t.Fatal("the activity must belong to the task's opportunity")
	}
	if _, err := PlanActivityTaskDone(task, CrmActivity{ID: 31, OpportunityID: 5}, TaskKindActivity, 9, 10); err == nil {
		t.Fatal("only the assignee completes")
	}
	if _, err := PlanActivityTaskDone(dismissed, CrmActivity{ID: 31, OpportunityID: 5}, TaskKindActivity, 7, 10); err == nil {
		t.Fatal("a closed task cannot be completed")
	}

	// A "send email" task closes only by sending the email, never by a hand-written activity.
	emailTask := task
	emailTask.Kind = TaskKindSendEmail
	if _, err := PlanActivityTaskDone(emailTask, CrmActivity{ID: 31, OpportunityID: 5}, TaskKindActivity, 7, 10); err == nil {
		t.Fatal("an activity must not close an email task")
	}
	if _, err := PlanActivityTaskDone(emailTask, CrmActivity{ID: 31, OpportunityID: 5}, TaskKindSendEmail, 7, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanActivityTaskDone(task, CrmActivity{ID: 31, OpportunityID: 5}, TaskKindSendEmail, 7, 10); err == nil {
		t.Fatal("an email must not close an activity task")
	}
}

func TestEditChangesTaskCard(t *testing.T) {
	stored := CrmOpportunity{ID: 1, Title: "A", ClientID: 5, Amount: 100, Probability: 20, OwnerUserID: 7}
	for _, edited := range []CrmOpportunity{
		{ID: 1, Title: "B", ClientID: 5, Amount: 100},
		{ID: 1, Title: "A", ClientID: 6, Amount: 100},
		{ID: 1, Title: "A", ClientID: 5, Amount: 200},
	} {
		if !EditChangesTaskCard(stored, edited) {
			t.Fatalf("a new title, client or amount must resend the tasks: %+v", edited)
		}
	}
	if EditChangesTaskCard(stored, CrmOpportunity{ID: 1, Title: "A", ClientID: 5, Amount: 100, Probability: 60, OwnerUserID: 9}) {
		t.Fatal("probability or owner are not on the task card")
	}
}

func TestCanSeeTasksOf(t *testing.T) {
	teams := []CrmTeam{
		{ID: 1, LeaderUserID: 3, MemberUserIDs: []int32{3, 4, 5}, Status: CrmStatusActive},
		{ID: 2, LeaderUserID: 8, MemberUserIDs: []int32{8, 9}, Status: CrmStatusDeleted},
	}
	if !CanSeeTasksOf(4, 4, teams) || !CanSeeTasksOf(3, 5, teams) {
		t.Fatal("a user sees their own tasks and a leader their members'")
	}
	if CanSeeTasksOf(4, 5, teams) || CanSeeTasksOf(3, 9, teams) || CanSeeTasksOf(8, 9, teams) {
		t.Fatal("a member does not see a colleague's; a deleted team gives no view")
	}
}

func TestPipelineTaskTemplatesAreNormalizedAndChecked(t *testing.T) {
	known := map[int16]bool{}
	none := func(int16) bool { return false }
	template := CrmStageTaskTemplate{Kind: TaskKindActivity, Title: " Llamar ", ActivityTypeID: ActivityTypeCall, AssignTo: TaskAssignOwner, AssignUserID: 99}
	payload := CrmPipeline{Name: "P", Stages: []CrmPipelineStage{
		{Name: "A", Status: CrmStatusActive, ApproverUserID: 20, TaskTemplates: []CrmStageTaskTemplate{template, template}},
	}}
	saved, err := ApplyPipelineEdit(CrmPipeline{}, payload, none, known)
	if err != nil {
		t.Fatal(err)
	}
	templates := saved.Stages[0].TaskTemplates
	if len(templates) != 2 || templates[0].ID != 1 || templates[1].ID != 2 || templates[0].Title != "Llamar" || templates[0].AssignUserID != 0 {
		t.Fatalf("new templates get IDs 1..n, trimmed titles and no user when assigned to the owner: %+v", templates)
	}

	// A stored template keeps its ID; a new one continues after the highest.
	saved.Stages[0].TaskTemplates = append(saved.Stages[0].TaskTemplates[:1], template)
	resaved, err := ApplyPipelineEdit(saved, saved, none, known)
	if err != nil || resaved.Stages[0].TaskTemplates[0].ID != 1 {
		t.Fatalf("stored template: %+v %v", resaved.Stages[0].TaskTemplates, err)
	}

	invalid := map[string]func(*CrmStageTaskTemplate){
		"empty title":        func(template *CrmStageTaskTemplate) { template.Title = " " },
		"unknown kind":       func(template *CrmStageTaskTemplate) { template.Kind = 2 },
		"email, no template": func(template *CrmStageTaskTemplate) { template.Kind = TaskKindSendEmail },
		"negative due":       func(template *CrmStageTaskTemplate) { template.DueDays = -1 },
		"no assignment":      func(template *CrmStageTaskTemplate) { template.AssignTo = 0 },
		"fixed user 0":       func(template *CrmStageTaskTemplate) { template.AssignTo, template.AssignUserID = TaskAssignUser, 0 },
		"foreign ID":         func(template *CrmStageTaskTemplate) { template.ID = 7 },
	}
	for caseName, mutate := range invalid {
		broken := template
		mutate(&broken)
		payload.Stages = []CrmPipelineStage{{Name: "A", Status: CrmStatusActive, TaskTemplates: []CrmStageTaskTemplate{broken}}}
		if _, err := ApplyPipelineEdit(CrmPipeline{}, payload, none, known); err == nil {
			t.Errorf("%s: expected an error", caseName)
		}
	}

	// An email template keeps its template and is logged as an Email activity; an activity one drops
	// a stale template ID.
	emailTemplate := template
	emailTemplate.Kind, emailTemplate.EmailTemplateID = TaskKindSendEmail, 3
	payload.Stages = []CrmPipelineStage{{Name: "A", Status: CrmStatusActive, TaskTemplates: []CrmStageTaskTemplate{emailTemplate}}}
	savedEmail, err := ApplyPipelineEdit(CrmPipeline{}, payload, none, known)
	if err != nil || savedEmail.Stages[0].TaskTemplates[0].ActivityTypeID != ActivityTypeEmail || savedEmail.Stages[0].TaskTemplates[0].EmailTemplateID != 3 {
		t.Fatalf("email template: %+v %v", savedEmail.Stages[0].TaskTemplates, err)
	}
	activityWithStaleID := template
	activityWithStaleID.EmailTemplateID = 3
	payload.Stages = []CrmPipelineStage{{Name: "A", Status: CrmStatusActive, TaskTemplates: []CrmStageTaskTemplate{activityWithStaleID}}}
	if savedActivity, err := ApplyPipelineEdit(CrmPipeline{}, payload, none, known); err != nil || savedActivity.Stages[0].TaskTemplates[0].EmailTemplateID != 0 {
		t.Fatalf("activity template keeps no email template: %v", err)
	}

	// References: the approver, the fixed assignee, a company activity type and an email template must be active.
	withReferences := CrmPipeline{Stages: []CrmPipelineStage{{Name: "A", Status: CrmStatusActive, ApproverUserID: 20, TaskTemplates: []CrmStageTaskTemplate{
		{Title: "T", Kind: TaskKindActivity, AssignTo: TaskAssignUser, AssignUserID: 9, ActivityTypeID: 41},
		{Title: "E", Kind: TaskKindSendEmail, AssignTo: TaskAssignOwner, ActivityTypeID: ActivityTypeEmail, EmailTemplateID: 3},
	}}}}
	userIDs, activityTypeIDs, emailTemplateIDs := PipelineReferenceIDs(withReferences)
	if len(userIDs) != 2 || len(activityTypeIDs) != 1 || activityTypeIDs[0] != 41 || len(emailTemplateIDs) != 1 || emailTemplateIDs[0] != 3 {
		t.Fatalf("reference IDs: %v %v %v", userIDs, activityTypeIDs, emailTemplateIDs)
	}
	activeTemplates := map[int16]bool{3: true}
	allActive := PipelineReferences{ActiveUserIDs: map[int32]bool{20: true, 9: true}, ActiveActivityTypeIDs: map[int16]bool{41: true}, ActiveEmailTemplateIDs: activeTemplates}
	if err := ValidatePipelineReferences(withReferences, allActive); err != nil {
		t.Fatal(err)
	}
	for caseName, references := range map[string]PipelineReferences{
		"approver inactive": {ActiveUserIDs: map[int32]bool{9: true}, ActiveActivityTypeIDs: map[int16]bool{41: true}, ActiveEmailTemplateIDs: activeTemplates},
		"assignee inactive": {ActiveUserIDs: map[int32]bool{20: true}, ActiveActivityTypeIDs: map[int16]bool{41: true}, ActiveEmailTemplateIDs: activeTemplates},
		"type inactive":     {ActiveUserIDs: map[int32]bool{20: true, 9: true}, ActiveEmailTemplateIDs: activeTemplates},
		"template inactive": {ActiveUserIDs: map[int32]bool{20: true, 9: true}, ActiveActivityTypeIDs: map[int16]bool{41: true}},
	} {
		if ValidatePipelineReferences(withReferences, references) == nil {
			t.Errorf("%s: expected an error", caseName)
		}
	}
}
