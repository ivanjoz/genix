package projects

import (
	"context"
	"path"
	"slices"

	"app/cloud"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"app/projects/types"
)

// projectLockWaiters is how many writes of one project may queue behind the running one: a burst of
// board moves by the team, one request each.
const projectLockWaiters = 8

// projectLockIdentifier keys the project lock. Project IDs are numbered per company, so the company
// is part of the key.
func projectLockIdentifier(companyID int32, projectID int32) int64 {
	return int64(companyID)<<32 | int64(uint32(projectID))
}

// openProjectWrite is the gate of every write inside a project: it takes the project's lock, then
// loads the project and checks the caller's role. ProjectRoleMember lets any team member in,
// ProjectRoleAdmin only its admins. Every read the write depends on happens after it, so the
// records the handler rewrites cannot change under it. The caller defers release.
func openProjectWrite(req *core.HandlerArgs, projectID int32, requiredRole int8) (types.Project, func(), core.HandlerResponse, bool) {
	noRelease := func() {}
	if projectID <= 0 {
		return types.Project{}, noRelease, req.MakeErr("No se envió el proyecto."), true
	}
	lock, lockErr := core.AcquireLock(context.Background(), core.ActionProjectWrite,
		projectLockIdentifier(req.User.CompanyID, projectID), projectLockWaiters)
	if lockErr != nil {
		return types.Project{}, noRelease, lockErr.Response(req), true
	}
	project, response, isDenied := authorizeProjectWrite(req, projectID, requiredRole)
	if isDenied {
		lock.Release()
		return types.Project{}, noRelease, response, true
	}
	return project, lock.Release, core.HandlerResponse{}, false
}

// authorizeProjectWrite loads an active, non-archived project and checks the caller's role in it,
// without the lock: for what writes no project record (a file upload).
func authorizeProjectWrite(req *core.HandlerArgs, projectID int32, requiredRole int8) (types.Project, core.HandlerResponse, bool) {
	project, err := loadProject(req.User.CompanyID, projectID)
	if err != nil {
		return types.Project{}, req.MakeErr("Error al obtener el proyecto:", err), true
	}
	if project == nil || project.Status != types.StatusActive {
		return types.Project{}, req.MakeErr("El proyecto no existe."), true
	}
	if project.State == types.ProjectStateArchived {
		return types.Project{}, req.MakeErr("El proyecto está archivado: es de solo lectura."), true
	}
	callerRole := projectRole(*project, req.User.ID)
	if callerRole == 0 || callerRole < requiredRole {
		if requiredRole == types.ProjectRoleAdmin {
			return types.Project{}, req.MakeErrCode("Solo los administradores del proyecto pueden hacer esto.", 403), true
		}
		return types.Project{}, req.MakeErrCode("Solo el equipo del proyecto puede hacer esto.", 403), true
	}
	return *project, core.HandlerResponse{}, false
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
