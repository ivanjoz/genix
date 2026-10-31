package projects

import (
	"slices"
	"testing"

	"app/projects/types"
)

func TestNormalizeStory(t *testing.T) {
	project := types.Project{Members: []types.ProjectMember{{UserID: 1, Role: types.ProjectRoleAdmin}}}
	validStory := types.Story{Title: "Load the corporate catalog", Type: types.StoryTypeStory, Priority: types.StoryPriorityMedium, EstimatedHours: 4.5}
	testCases := []struct {
		name    string
		change  func(story *types.Story)
		allowed bool
	}{
		{"valid", func(story *types.Story) {}, true},
		{"no title", func(story *types.Story) { story.Title = "  " }, false},
		{"bad type", func(story *types.Story) { story.Type = 7 }, false},
		{"bad priority", func(story *types.Story) { story.Priority = 0 }, false},
		{"estimate not in half hours", func(story *types.Story) { story.EstimatedHours = 1.25 }, false},
		{"negative estimate", func(story *types.Story) { story.EstimatedHours = -1 }, false},
		{"epic of the project", func(story *types.Story) { story.EpicID = 10 }, true},
		{"epic of another project", func(story *types.Story) { story.EpicID = 99 }, false},
		{"assignee on the team", func(story *types.Story) { story.AssigneeID = 1 }, true},
		{"assignee off the team", func(story *types.Story) { story.AssigneeID = 2 }, false},
	}
	for _, testCase := range testCases {
		story := validStory
		testCase.change(&story)
		if err := normalizeStory(&story, project, []int64{10}); (err == nil) != testCase.allowed {
			t.Errorf("%s: err = %v, allowed %v", testCase.name, err, testCase.allowed)
		}
	}
}

func TestStoryAssigneeRow(t *testing.T) {
	testCases := []struct {
		name       string
		story      types.Story
		wantStatus int8
	}{
		{"assigned", types.Story{ID: 4, ProjectID: 2, AssigneeID: 7, Status: types.StatusActive}, types.StatusActive},
		{"unassigned", types.Story{ID: 4, ProjectID: 2, Status: types.StatusActive}, types.StatusDeleted},
		{"deleted story", types.Story{ID: 4, ProjectID: 2, AssigneeID: 7, Status: types.StatusDeleted}, types.StatusDeleted},
	}
	for _, testCase := range testCases {
		assigneeRow := storyAssigneeRow(testCase.story)
		if assigneeRow.Status != testCase.wantStatus || assigneeRow.StoryID != 4 || assigneeRow.ProjectID != 2 || assigneeRow.AssigneeID != testCase.story.AssigneeID {
			t.Errorf("%s: row = %+v, want status %d", testCase.name, assigneeRow, testCase.wantStatus)
		}
	}
}

func TestStoryStateAfterCardMove(t *testing.T) {
	story := types.Story{LastSprintID: 3, State: types.StoryStateInDevelopment}
	if state := storyStateAfterCardMove(story, 3, types.StoryStateInReview); state != types.StoryStateInReview {
		t.Errorf("a move in the latest sprint: state %d, want %d", state, types.StoryStateInReview)
	}
	if state := storyStateAfterCardMove(story, 2, types.StoryStateInReview); state != types.StoryStateInDevelopment {
		t.Errorf("a move in an older sprint: state %d, want %d", state, types.StoryStateInDevelopment)
	}
}

func TestSetStoryState(t *testing.T) {
	story := types.Story{}
	setStoryState(&story, types.StoryStatePending, 100)
	setStoryState(&story, types.StoryStateInDevelopment, 101)
	setStoryState(&story, types.StoryStateInDevelopment, 102) // no change: keeps day 101
	setStoryState(&story, types.StoryStatePending, 103)
	setStoryState(&story, types.StoryStateNotPlanned, 104)
	want := []types.StoryStateDate{{State: types.StoryStatePending, Date: 103}, {State: types.StoryStateInDevelopment, Date: 101}}
	if story.State != types.StoryStateNotPlanned || !slices.Equal(story.StateDates, want) {
		t.Errorf("state %d, dates %+v: want 0 and %+v", story.State, story.StateDates, want)
	}
}

func TestNormalizeStoryDetail(t *testing.T) {
	detail := types.StoryDetail{Checklist: []types.ChecklistItem{{Text: " Sums match "}, {Text: "  "}}}
	if err := normalizeStoryDetail(&detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.Checklist) != 1 || detail.Checklist[0].Text != "Sums match" {
		t.Errorf("blank items are dropped and texts trimmed, got %+v", detail.Checklist)
	}
	pastedImage := types.StoryDetail{Description: `<p><img src="data:image/png;base64,AAAA"></p>`}
	if normalizeStoryDetail(&pastedImage) == nil {
		t.Errorf("a pasted image must go as an attachment")
	}
}

func TestAttachmentObjectKey(t *testing.T) {
	objectKey := attachmentObjectKey(1, 4, "ab3de5gh9jk1mn0p", "Informe final (v2).pdf")
	if objectKey != "projects/c1/p4/ab3de5gh9jk1mn0p/Informe-final-v2.pdf" {
		t.Errorf("unexpected key %s", objectKey)
	}
	if !isProjectFileKey(1, 4, objectKey) {
		t.Errorf("the key %s must pass the attachment key check of its project", objectKey)
	}
	if isProjectFileKey(1, 14, objectKey) || isProjectFileKey(2, 4, objectKey) || isProjectFileKey(1, 0, objectKey) {
		t.Errorf("the key %s of company 1, project 4 must fail the check of another company or project", objectKey)
	}
	if !isProjectFileKey(1, 4, attachmentObjectKey(1, 4, "ab3de5gh9jk1mn0p", "***")) {
		t.Errorf("a name with no safe characters still makes a valid key")
	}
}

func TestMergeAttachments(t *testing.T) {
	stored := []types.Attachment{
		{Key: "projects/c1/p4/aaaaaaaaaaaaaaaa/a.pdf", Name: "a.pdf", Size: 10, UploadedBy: 7, UploadedAt: 100},
		{Key: "projects/c1/p4/bbbbbbbbbbbbbbbb/b.pdf", Name: "b.pdf", Size: 10, UploadedBy: 7, UploadedAt: 100},
	}
	incoming := []types.Attachment{
		{Key: "projects/c1/p4/aaaaaaaaaaaaaaaa/a.pdf", Name: "renamed by the client", UploadedBy: 99},
		{Key: "projects/c1/p4/cccccccccccccccc/c.png", Name: "c.png", Size: 2048, ContentType: "image/png", DocumentID: 77},
	}
	merged, removedKeys, err := mergeAttachments(1, 4, stored, incoming, 3, 500)
	if err != nil {
		t.Fatal(err)
	}
	if merged[0] != stored[0] {
		t.Errorf("a stored attachment keeps its stored metadata, got %+v", merged[0])
	}
	if merged[1].UploadedBy != 3 || merged[1].UploadedAt != 500 || merged[1].DocumentID != 0 {
		t.Errorf("a new attachment is stamped by the server, with no DocumentID of the client's, got %+v", merged[1])
	}
	if !slices.Equal(removedKeys, []string{"projects/c1/p4/bbbbbbbbbbbbbbbb/b.pdf"}) {
		t.Errorf("b.pdf was dropped, got %v", removedKeys)
	}

	invalidCases := []types.Attachment{
		{Key: "voluntariado/img/1.avif", Name: "x", Size: 10},
		{Key: "projects/c1/p5/cccccccccccccccc/c.png", Name: "c.png", Size: 2048},
		{Key: "projects/c1/p4/cccccccccccccccc/c.png", Name: "c.png", Size: MaxAttachmentBytes + 1},
		{Key: "projects/c1/p4/cccccccccccccccc/c.png", Name: "", Size: 10},
	}
	for _, invalidAttachment := range invalidCases {
		if _, _, err := mergeAttachments(1, 4, nil, []types.Attachment{invalidAttachment}, 3, 500); err == nil {
			t.Errorf("attachment %+v must be refused", invalidAttachment)
		}
	}
}

func TestStoryFinalState(t *testing.T) {
	sprints := []types.Sprint{
		{ID: 1, Status: types.StatusActive, State: types.SprintStateClosed, Stories: []types.SprintStory{
			{StoryID: 10, State: types.StoryStateInDevelopment, AddedAt: 100, CarriedToSprintID: 2},
			{StoryID: 11, State: types.StoryStateStopped, AddedAt: 100},
		}},
		{ID: 2, Status: types.StatusActive, State: types.SprintStateActive, Stories: []types.SprintStory{
			{StoryID: 10, State: types.StoryStateInReview, AddedAt: 200},
		}},
		{ID: 3, Status: types.StatusDeleted, State: types.SprintStatePlanned, Stories: []types.SprintStory{
			{StoryID: 11, State: types.StoryStatePending, AddedAt: 300},
		}},
	}
	finalState, lastSprintID, sprintIDs := storyFinalState(10, sprints)
	if finalState != types.StoryStateInReview || lastSprintID != 2 || !slices.Equal(sprintIDs, []int64{1, 2}) {
		t.Errorf("story 10 follows the sprint it was last added to: %d %d %v", finalState, lastSprintID, sprintIDs)
	}
	finalState, lastSprintID, _ = storyFinalState(11, sprints)
	if finalState != types.StoryStateStopped || lastSprintID != 1 {
		t.Errorf("story 11 came back from the closed sprint stopped (a deleted sprint does not count): %d %d", finalState, lastSprintID)
	}
	if finalState, lastSprintID, sprintIDs = storyFinalState(12, sprints); finalState != types.StoryStateNotPlanned || lastSprintID != 0 || sprintIDs != nil {
		t.Errorf("story 12 was never planned")
	}
}

func TestRankBetween(t *testing.T) {
	if rank := rankBetween(1, true, 2, true); rank != 1.5 {
		t.Errorf("between 1 and 2 → 1.5, got %v", rank)
	}
	if rank := rankBetween(4, true, 0, false); rank != 5 {
		t.Errorf("after the last (4) → 5, got %v", rank)
	}
	if rank := rankBetween(0, false, 3, true); rank != 2 {
		t.Errorf("before the first (3) → 2, got %v", rank)
	}
	if rank := rankBetween(0, false, 0, false); rank != 1 {
		t.Errorf("alone → 1, got %v", rank)
	}
}
