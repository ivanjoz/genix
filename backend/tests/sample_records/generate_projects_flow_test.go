package sample_records

import (
	"app/projects"
	projectsTypes "app/projects/types"
	"encoding/json"
	"math"
	"slices"
	"strings"
	"testing"
)

// The catalog feeds the handlers as is: anything they would reject fails the run half way through.
func TestProjectsFlowCatalogPassesTheHandlersRules(t *testing.T) {
	catalog := projectsFlowCatalog{}
	if err := json.Unmarshal(projectsFlowCatalogJSON, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.People) < 3 || len(catalog.Comments) == 0 || len(catalog.StopReasons) == 0 || len(catalog.SprintGoals) == 0 {
		t.Fatal("the catalog needs at least 3 people and some comments, stop reasons and sprint goals")
	}
	for _, theme := range catalog.Projects {
		if len(theme.Epics) < 2 || !slices.Contains(projects.ProjectColors, theme.Color) {
			t.Errorf("%s: needs 2+ epics and a palette color", theme.Name)
		}
		for _, epic := range theme.Epics {
			if len(epic.Stories) == 0 || len(epic.Name) > 80 || !slices.Contains(projects.ProjectColors, epic.Color) {
				t.Errorf("%s / %s: needs stories, a name up to 80 bytes and a palette color", theme.Name, epic.Name)
			}
			for _, story := range epic.Stories {
				isValid := strings.TrimSpace(story.Title) != "" && len(story.Title) <= 190 && len(story.Summary) <= 500 &&
					story.Type >= projectsTypes.StoryTypeStory && story.Type <= projectsTypes.StoryTypeTask &&
					story.Priority >= projectsTypes.StoryPriorityLow && story.Priority <= projectsTypes.StoryPriorityCritical &&
					story.Hours > 0 && math.Mod(float64(story.Hours)*2, 1) == 0
				if !isValid {
					t.Errorf("%s: invalid story %+v", theme.Name, story)
				}
			}
		}
	}
}

// makeConsistentSnapshot: project 7 with H1 finished in a closed sprint and carried nowhere, H2 in
// the active sprint, H3 never planned.
func makeConsistentSnapshot() projectSnapshot {
	projectRecordID := func(sequence int64) int64 { return 7<<projectsFlowSequenceBits | sequence }
	return projectSnapshot{
		project: projectsTypes.Project{ID: 7, Members: []projectsTypes.ProjectMember{{UserID: 1, Role: projectsTypes.ProjectRoleAdmin}}},
		epics:   []projectsTypes.Epic{{ID: projectRecordID(1), ProjectID: 7}},
		stories: []projectsTypes.Story{
			{ID: projectRecordID(1), ProjectID: 7, EpicID: projectRecordID(1), AssigneeID: 1, State: projectsTypes.StoryStateFinished,
				LastSprintID: projectRecordID(1), SprintIDs: []int64{projectRecordID(1)}, StateDates: []projectsTypes.StoryStateDate{{State: projectsTypes.StoryStateFinished, Date: 100}}},
			{ID: projectRecordID(2), ProjectID: 7, State: projectsTypes.StoryStateInDevelopment,
				LastSprintID: projectRecordID(2), SprintIDs: []int64{projectRecordID(2)}, StateDates: []projectsTypes.StoryStateDate{{State: projectsTypes.StoryStateInDevelopment, Date: 110}}},
			{ID: projectRecordID(3), ProjectID: 7},
		},
		sprints: []projectsTypes.Sprint{
			{ID: projectRecordID(1), ProjectID: 7, Name: "Sprint 1", State: projectsTypes.SprintStateClosed, ClosedAt: 50, ClosedBy: 1,
				Stories: []projectsTypes.SprintStory{{StoryID: projectRecordID(1), State: projectsTypes.StoryStateFinished, AddedAt: 10, DoneAt: 40}}},
			{ID: projectRecordID(2), ProjectID: 7, Name: "Sprint 2", State: projectsTypes.SprintStateActive,
				Stories: []projectsTypes.SprintStory{{StoryID: projectRecordID(2), State: projectsTypes.StoryStateInDevelopment, AddedAt: 60}}},
		},
		documents: []projectsTypes.Document{{ID: projectRecordID(1), ProjectID: 7, Key: "projects/c1/p7/abc/acta.txt"}},
	}
}

func TestCheckProjectInvariants(t *testing.T) {
	if err := checkProjectInvariants(makeConsistentSnapshot()); err != nil {
		t.Fatalf("a consistent project fails: %v", err)
	}
	brokenProjects := map[string]func(snapshot *projectSnapshot){
		"final state out of step with the sprints": func(snapshot *projectSnapshot) {
			snapshot.stories[1].State = projectsTypes.StoryStateInReview
		},
		"a story in two open sprints": func(snapshot *projectSnapshot) {
			snapshot.sprints[0].State = projectsTypes.SprintStatePlanned
			snapshot.sprints[0].Stories = append(snapshot.sprints[0].Stories, snapshot.sprints[1].Stories[0])
		},
		"two active sprints": func(snapshot *projectSnapshot) {
			snapshot.sprints[0].State = projectsTypes.SprintStateActive
		},
		"a finished card without its finish time": func(snapshot *projectSnapshot) {
			snapshot.sprints[0].Stories[0].DoneAt = 0
		},
		"a record numbered in another project": func(snapshot *projectSnapshot) {
			snapshot.epics[0].ID = 8<<projectsFlowSequenceBits | 1
		},
		"an assignee off the team": func(snapshot *projectSnapshot) {
			snapshot.stories[2].AssigneeID = 99
		},
		"a deleted epic on a story": func(snapshot *projectSnapshot) {
			snapshot.epics = nil
		},
		"a file outside the project's folder": func(snapshot *projectSnapshot) {
			snapshot.documents[0].Key = "projects/c1/p8/abc/acta.txt"
		},
		"a state without its date": func(snapshot *projectSnapshot) {
			snapshot.stories[1].StateDates = nil
		},
	}
	for name, breakProject := range brokenProjects {
		snapshot := makeConsistentSnapshot()
		breakProject(&snapshot)
		if err := checkProjectInvariants(snapshot); err == nil {
			t.Errorf("%s: not detected", name)
		}
	}
}
