package projects

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"app/projects/types"
)

// ProjectColors are the colors a project or an epic may take: genix-ui Calendar's palette.
var ProjectColors = []string{"blue", "green", "red", "amber", "purple", "teal", "pink", "gray"}

// projectRole is the user's role in the project, 0 when they are not on its team.
func projectRole(project types.Project, userID int32) int8 {
	for _, member := range project.Members {
		if member.UserID == userID {
			return member.Role
		}
	}
	return 0
}

func isProjectMember(project types.Project, userID int32) bool {
	return projectRole(project, userID) > 0
}

func isProjectAdmin(project types.Project, userID int32) bool {
	return projectRole(project, userID) == types.ProjectRoleAdmin
}

// normalizeProject validates and cleans the fields a project admin edits. The team must keep at
// least one admin, so a project can always be administered.
func normalizeProject(project *types.Project) error {
	project.Name = strings.TrimSpace(project.Name)
	project.Description = strings.TrimSpace(project.Description)
	if project.Name == "" || len(project.Name) > 120 {
		return errors.New("El proyecto necesita un nombre de hasta 120 caracteres.")
	}
	if len(project.Description) > 2000 {
		return errors.New("La descripción puede tener hasta 2000 caracteres.")
	}
	if project.StartDate > 0 && project.EndDate > 0 && project.StartDate > project.EndDate {
		return errors.New("El proyecto no puede terminar antes de empezar.")
	}
	if !slices.Contains(ProjectColors, project.Color) {
		project.Color = ProjectColors[0]
	}
	if project.State != types.ProjectStateArchived {
		project.State = types.ProjectStateActive
	}

	seenUserIDs := map[int32]bool{}
	hasAdmin := false
	for _, member := range project.Members {
		if member.UserID <= 0 || seenUserIDs[member.UserID] {
			return errors.New("El equipo tiene un usuario inválido o repetido.")
		}
		if member.Role != types.ProjectRoleMember && member.Role != types.ProjectRoleAdmin {
			return errors.New("Cada integrante del equipo debe ser Miembro o Administrador.")
		}
		seenUserIDs[member.UserID] = true
		hasAdmin = hasAdmin || member.Role == types.ProjectRoleAdmin
	}
	if !hasAdmin {
		return errors.New("El equipo necesita al menos un administrador.")
	}
	return nil
}

// normalizeEpic validates an epic's name and color.
func normalizeEpic(epic *types.Epic) error {
	epic.Name = strings.TrimSpace(epic.Name)
	if epic.Name == "" || len(epic.Name) > 80 {
		return errors.New("La épica necesita un nombre de hasta 80 caracteres.")
	}
	if !slices.Contains(ProjectColors, epic.Color) {
		epic.Color = ProjectColors[0]
	}
	return nil
}

// maxSavedEpics caps one save of the project panel's Epics tab: each epic is a write, and each
// deletion also rewrites its stories.
const maxSavedEpics = 50

// normalizeEpics validates the epics saved together from the project panel: all of one project,
// the kept ones named. It returns that project and whether any epic is deleted (admins only).
func normalizeEpics(epics []types.Epic) (projectID int32, hasDeletion bool, err error) {
	if len(epics) == 0 || len(epics) > maxSavedEpics {
		return 0, false, fmt.Errorf("Guarda entre 1 y %d épicas a la vez.", maxSavedEpics)
	}
	projectID = epics[0].ProjectID
	for epicIndex := range epics {
		epic := &epics[epicIndex]
		if epic.ProjectID != projectID {
			return 0, false, errors.New("Las épicas deben ser de un mismo proyecto.")
		}
		if epic.ID != 0 && epic.Status == types.StatusDeleted {
			hasDeletion = true
			continue
		}
		if err := normalizeEpic(epic); err != nil {
			return 0, false, err
		}
	}
	return projectID, hasDeletion, nil
}

// newTeamMemberIDs are the members the edit adds: only they are checked for the Edit level, so a
// team member who later lost it can still be removed or demoted.
func newTeamMemberIDs(storedMembers []types.ProjectMember, members []types.ProjectMember) []int32 {
	addedUserIDs := []int32{}
	for _, member := range members {
		isStored := slices.ContainsFunc(storedMembers, func(storedMember types.ProjectMember) bool {
			return storedMember.UserID == member.UserID
		})
		if !isStored {
			addedUserIDs = append(addedUserIDs, member.UserID)
		}
	}
	return addedUserIDs
}
