package projects

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"app/projects/types"
)

// GetProjects is the delta list of every project of the company: all users with the access see all.
func GetProjects(req *core.HandlerArgs) core.HandlerResponse {
	projects := []types.Project{}
	query := db.Query(&projects)
	query.CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los proyectos:", err)
	}
	return core.MakeResponse(req, &projects)
}

// PostProject creates a project (any user with the access, who becomes its admin) or edits one (its
// admins): fields, team, archive. Users added to the team must be able to work in projects, or they
// could never write.
func PostProject(req *core.HandlerArgs) core.HandlerResponse {
	project := types.Project{}
	if err := json.Unmarshal([]byte(*req.Body), &project); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if project.ID == 0 {
		return createProject(req, project)
	}
	if err := normalizeProject(&project); err != nil {
		return req.MakeErr(err)
	}

	storedProject, release, response, isDenied := openProjectWrite(req, project.ID, types.ProjectRoleAdmin)
	if isDenied {
		return response
	}
	defer release()
	if response, isDenied := denyMembersWithoutAccess(req, newTeamMemberIDs(storedProject.Members, project.Members)); isDenied {
		return response
	}

	storedProject.Name, storedProject.Description, storedProject.Color = project.Name, project.Description, project.Color
	storedProject.StartDate, storedProject.EndDate = project.StartDate, project.EndDate
	storedProject.State, storedProject.Members, storedProject.UpdatedBy = project.State, project.Members, req.User.ID
	projectTable := db.TableOf[types.Project]()
	err := db.Update(&[]types.Project{storedProject}, projectTable.Status, projectTable.Name, projectTable.Description, projectTable.Color,
		projectTable.StartDate, projectTable.EndDate, projectTable.State, projectTable.Members, projectTable.UpdatedBy)
	if err != nil {
		return req.MakeErr("Error al guardar el proyecto:", err)
	}
	return req.MakeResponse(storedProject)
}

// createProject validates and stores a new project; its creator becomes an admin.
func createProject(req *core.HandlerArgs, project types.Project) core.HandlerResponse {
	if projectRole(project, req.User.ID) != types.ProjectRoleAdmin {
		project.Members = slices.DeleteFunc(project.Members, func(member types.ProjectMember) bool { return member.UserID == req.User.ID })
		project.Members = append(project.Members, types.ProjectMember{UserID: req.User.ID, Role: types.ProjectRoleAdmin})
	}
	if err := normalizeProject(&project); err != nil {
		return req.MakeErr(err)
	}
	if response, isDenied := denyMembersWithoutAccess(req, newTeamMemberIDs(nil, project.Members)); isDenied {
		return response
	}
	project.CompanyID, project.Status = req.User.CompanyID, types.StatusActive
	project.CreatedBy, project.Created, project.UpdatedBy = req.User.ID, core.SUnixTime(), req.User.ID
	projects := []types.Project{project}
	if err := db.Insert(&projects); err != nil {
		return req.MakeErr("Error al guardar el proyecto:", err)
	}
	return req.MakeResponse(projects[0])
}

// loadCompanyUsers reads users of the company by ID.
func loadCompanyUsers(companyID int32, userIDs []int32) ([]coreTypes.User, error) {
	users := []coreTypes.User{}
	if len(userIDs) == 0 {
		return users, nil
	}
	query := db.Query(&users)
	query.CompanyID.Equals(companyID).ID.In(userIDs...)
	return users, query.Exec()
}

// denyMembersWithoutAccess refuses a team change that adds a user who can't work in projects.
func denyMembersWithoutAccess(req *core.HandlerArgs, addedUserIDs []int32) (core.HandlerResponse, bool) {
	users, err := loadCompanyUsers(req.User.CompanyID, addedUserIDs)
	if err != nil {
		return req.MakeErr("Error al obtener los usuarios del equipo:", err), true
	}
	for _, userID := range addedUserIDs {
		userIndex := slices.IndexFunc(users, func(user coreTypes.User) bool { return user.ID == userID })
		if userIndex < 0 || !canWorkInProjects(users[userIndex]) {
			return req.MakeErr(fmt.Sprintf("El usuario %d no tiene el acceso de escritura a Proyectos.", userID)), true
		}
	}
	return core.HandlerResponse{}, false
}

type teamCandidate struct {
	ID   int32
	Name string
}

// GetTeamCandidates lists the users who can join a team: active users holding the projects access
// at the write level.
func GetTeamCandidates(req *core.HandlerArgs) core.HandlerResponse {
	users := []coreTypes.User{}
	query := db.Query(&users)
	query.CompanyID.Equals(req.User.CompanyID).Status.Equals(1)
	if err := query.AllowFilter().Exec(); err != nil {
		return req.MakeErr("Error al obtener los usuarios:", err)
	}

	teamCandidates := []teamCandidate{}
	for _, user := range users {
		if !canWorkInProjects(user) {
			continue
		}
		userName := strings.TrimSpace(user.FirstName + " " + user.LastName)
		if userName == "" {
			userName = user.User
		}
		teamCandidates = append(teamCandidates, teamCandidate{ID: user.ID, Name: userName})
	}
	return req.MakeResponse(teamCandidates)
}
