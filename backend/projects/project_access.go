package projects

import (
	"errors"
	"path"
	"slices"

	"app/cloud"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"app/projects/types"
)

// projectAccessError is a refusal of the caller's role in the project: it is answered 403, where
// every other error of a project write is a 400.
type projectAccessError struct{ message string }

func (accessError projectAccessError) Error() string { return accessError.message }

// makeProjectErr answers a project write's error, with the 403 of a role refusal.
func makeProjectErr(req *core.HandlerArgs, err error) core.HandlerResponse {
	if accessError, isAccessError := errors.AsType[projectAccessError](err); isAccessError {
		return req.MakeErrCode(accessError.message, 403)
	}
	return req.MakeErr(err)
}

// lockProjectWriteIn is the gate of every write inside a project: it locks the project's row in the
// caller's transaction, then checks the caller's role on it. ProjectRoleMember lets any team member
// in, ProjectRoleAdmin only its admins. Every read the write depends on happens after it, so the
// records the handler rewrites cannot change under it: the lock is held until the transaction ends.
func lockProjectWriteIn(transaction *db.Transaction, req *core.HandlerArgs, projectID int32, requiredRole int8) (types.Project, error) {
	if projectID <= 0 {
		return types.Project{}, core.Err("No se envió el proyecto.")
	}
	lockedProjects := []types.Project{{CompanyID: req.User.CompanyID, ID: projectID}}
	if err := db.LockByKeys(transaction, &lockedProjects); err != nil {
		return types.Project{}, core.Err("Error al obtener el proyecto:", err)
	}
	if len(lockedProjects) == 0 {
		return types.Project{}, core.Err("El proyecto no existe.")
	}
	return lockedProjects[0], checkProjectWrite(lockedProjects[0], req.User.ID, requiredRole)
}

// authorizeProjectWrite loads an active, non-archived project and checks the caller's role in it,
// without the lock: for what writes no project record (a file upload).
func authorizeProjectWrite(req *core.HandlerArgs, projectID int32, requiredRole int8) (types.Project, core.HandlerResponse, bool) {
	project, err := loadProject(req.User.CompanyID, projectID)
	if err != nil {
		return types.Project{}, req.MakeErr("Error al obtener el proyecto:", err), true
	}
	if project == nil {
		return types.Project{}, req.MakeErr("El proyecto no existe."), true
	}
	if err := checkProjectWrite(*project, req.User.ID, requiredRole); err != nil {
		return types.Project{}, makeProjectErr(req, err), true
	}
	return *project, core.HandlerResponse{}, false
}

// checkProjectWrite refuses a write to a deleted or archived project, or by a user below requiredRole.
func checkProjectWrite(project types.Project, userID int32, requiredRole int8) error {
	if project.Status != types.StatusActive {
		return core.Err("El proyecto no existe.")
	}
	if project.State == types.ProjectStateArchived {
		return core.Err("El proyecto está archivado: es de solo lectura.")
	}
	callerRole := projectRole(project, userID)
	if callerRole == 0 || callerRole < requiredRole {
		if requiredRole == types.ProjectRoleAdmin {
			return projectAccessError{"Solo los administradores del proyecto pueden hacer esto."}
		}
		return projectAccessError{"Solo el equipo del proyecto puede hacer esto."}
	}
	return nil
}

func loadProject(companyID int32, projectID int32) (*types.Project, error) {
	projects := []types.Project{}
	query := db.Query(&projects)
	query.CompanyID.Equals(companyID).ID.Equals(projectID)
	if err := query.Exec(); err != nil || len(projects) == 0 {
		return nil, err
	}
	return &projects[0], nil
}

// loadProjectEpicIDs are the IDs of the project's active epics: the ones a story may take.
func loadProjectEpicIDs(companyID int32, projectID int32) ([]int64, error) {
	epics := []types.Epic{}
	query := db.Query(&epics)
	query.CompanyID.Equals(companyID).ProjectID.Equals(projectID).Status.Equals(types.StatusActive)
	if err := query.Exec(); err != nil {
		return nil, err
	}
	epicIDs := make([]int64, 0, len(epics))
	for _, epic := range epics {
		epicIDs = append(epicIDs, epic.ID)
	}
	return epicIDs, nil
}

// loadProjectStories are the project's active stories.
func loadProjectStories(companyID int32, projectID int32) ([]types.Story, error) {
	stories := []types.Story{}
	query := db.Query(&stories)
	query.CompanyID.Equals(companyID).ProjectID.Equals(projectID).Status.Equals(types.StatusActive)
	return stories, query.Exec()
}

// loadProjectStory is an active story of the project, nil when it is gone or of another project.
func loadProjectStory(companyID int32, projectID int32, storyID int64) (*types.Story, error) {
	stories := []types.Story{}
	query := db.Query(&stories)
	query.CompanyID.Equals(companyID).ID.Equals(storyID)
	if err := query.Exec(); err != nil {
		return nil, err
	}
	if len(stories) == 0 || stories[0].ProjectID != projectID || stories[0].Status != types.StatusActive {
		return nil, nil
	}
	return &stories[0], nil
}

// loadProjectSprints are the project's active sprints, the closed ones included.
func loadProjectSprints(companyID int32, projectID int32) ([]types.Sprint, error) {
	sprints := []types.Sprint{}
	query := db.Query(&sprints)
	query.CompanyID.Equals(companyID).ProjectID.Equals(projectID).Status.Equals(types.StatusActive)
	return sprints, query.Exec()
}

// canWorkInProjects: the user holds the projects access at the write level, which every team
// member needs (the router asks it on every POST of the module). User 1 holds every access.
func canWorkInProjects(user coreTypes.User) bool {
	if user.ID == 1 {
		return true
	}
	if user.Status != 1 {
		return false
	}
	grants, err := core.DecodeAccesosGrants(user.AccesosComputed, user.AccesosSubComputed)
	if err != nil {
		core.Log("Error al leer los accesos del usuario", user.ID, err)
		return false
	}
	return slices.ContainsFunc(grants, func(grant core.AccesoGrant) bool {
		return grant.AccesoID == types.AccesoIDProjects && grant.Nivel >= 2
	})
}

// deleteProjectFiles removes the files dropped from a story. Best effort: the records no longer
// name them, so a failure only leaves an orphan file behind.
func deleteProjectFiles(objectKeys []string) {
	for _, objectKey := range objectKeys {
		if err := cloud.DeleteFile(cloud.SaveFileArgs{Path: path.Dir(objectKey), Name: path.Base(objectKey)}); err != nil {
			core.Log("Error al eliminar el archivo", objectKey, err)
		}
	}
}
