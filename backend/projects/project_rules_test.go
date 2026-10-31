package projects

import (
	"slices"
	"testing"

	"app/projects/types"
)

func TestNormalizeProject(t *testing.T) {
	admin := types.ProjectMember{UserID: 1, Role: types.ProjectRoleAdmin}
	member := types.ProjectMember{UserID: 2, Role: types.ProjectRoleMember}
	testCases := []struct {
		name    string
		project types.Project
		allowed bool
	}{
		{"valid", types.Project{Name: "Fertirriego", Members: []types.ProjectMember{admin, member}}, true},
		{"no name", types.Project{Members: []types.ProjectMember{admin}}, false},
		{"no admin", types.Project{Name: "F", Members: []types.ProjectMember{member}}, false},
		{"repeated member", types.Project{Name: "F", Members: []types.ProjectMember{admin, admin}}, false},
		{"invalid role", types.Project{Name: "F", Members: []types.ProjectMember{admin, {UserID: 3, Role: 9}}}, false},
		{"ends before it starts", types.Project{Name: "F", StartDate: 20, EndDate: 10, Members: []types.ProjectMember{admin}}, false},
	}
	for _, testCase := range testCases {
		project := testCase.project
		if err := normalizeProject(&project); (err == nil) != testCase.allowed {
			t.Errorf("%s: err = %v, allowed %v", testCase.name, err, testCase.allowed)
		}
	}

	project := types.Project{Name: " Fertirriego ", Color: "neon", Members: []types.ProjectMember{admin}}
	if err := normalizeProject(&project); err != nil {
		t.Fatal(err)
	}
	if project.Name != "Fertirriego" || project.Color != ProjectColors[0] || project.State != types.ProjectStateActive {
		t.Errorf("normalized project = %+v", project)
	}
}

func TestNormalizeEpics(t *testing.T) {
	named := types.Epic{ProjectID: 1, Name: "Catálogo", Status: types.StatusActive}
	deleted := types.Epic{ProjectID: 1, ID: 4, Status: types.StatusDeleted}
	testCases := []struct {
		name        string
		epics       []types.Epic
		allowed     bool
		hasDeletion bool
	}{
		{"one new epic", []types.Epic{named}, true, false},
		{"a deleted epic needs no name", []types.Epic{named, deleted}, true, true},
		{"nothing to save", []types.Epic{}, false, false},
		{"unnamed", []types.Epic{{ProjectID: 1, Name: " "}}, false, false},
		{"two projects", []types.Epic{named, {ProjectID: 2, Name: "Otra"}}, false, false},
		{"too many", slices.Repeat([]types.Epic{named}, maxSavedEpics+1), false, false},
	}
	for _, testCase := range testCases {
		projectID, hasDeletion, err := normalizeEpics(testCase.epics)
		if (err == nil) != testCase.allowed {
			t.Errorf("%s: err = %v, allowed %v", testCase.name, err, testCase.allowed)
		}
		if testCase.allowed && (projectID != 1 || hasDeletion != testCase.hasDeletion) {
			t.Errorf("%s: project %d, deletion %v", testCase.name, projectID, hasDeletion)
		}
	}
}

func TestProjectRoles(t *testing.T) {
	project := types.Project{Members: []types.ProjectMember{{UserID: 1, Role: types.ProjectRoleAdmin}, {UserID: 2, Role: types.ProjectRoleMember}}}
	if !isProjectAdmin(project, 1) || isProjectAdmin(project, 2) {
		t.Errorf("only user 1 is an admin")
	}
	if !isProjectMember(project, 2) || isProjectMember(project, 3) {
		t.Errorf("users 1 and 2 are on the team, 3 is not")
	}
}

func TestNewTeamMemberIDs(t *testing.T) {
	stored := []types.ProjectMember{{UserID: 1, Role: types.ProjectRoleAdmin}, {UserID: 2, Role: types.ProjectRoleMember}}
	edited := []types.ProjectMember{{UserID: 1, Role: types.ProjectRoleAdmin}, {UserID: 2, Role: types.ProjectRoleAdmin}, {UserID: 5, Role: types.ProjectRoleMember}}
	if addedUserIDs := newTeamMemberIDs(stored, edited); !slices.Equal(addedUserIDs, []int32{5}) {
		t.Errorf("only user 5 joins the team, got %v", addedUserIDs)
	}
}
