package crm

import (
	"encoding/json"
	"fmt"

	"app/core"
	"app/crm/types"
	"app/db"
)

// GetCrmTeams syncs every team of the company.
func GetCrmTeams(req *core.HandlerArgs) core.HandlerResponse {
	teams := []types.CrmTeam{}
	query := db.Query(&teams)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), types.CrmStatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los equipos:", err)
	}
	return req.MakeResponse(teams)
}

// PostCrmTeam creates or edits a team; Status 0 deletes it (soft).
func PostCrmTeam(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmTeam{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	activeUserIDs, err := loadActiveUserIDs(companyID, payload.MemberUserIDs)
	if err != nil {
		return req.MakeErr(err)
	}
	var savedTeam types.CrmTeam
	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		// Teams are few; reading them all inside the transaction keeps "one team per user" race free
		// enough for a settings screen.
		allTeams := []types.CrmTeam{}
		query := db.Query(&allTeams)
		query.CompanyID.Equals(companyID)
		if err := query.Exec(); err != nil {
			return err
		}
		if payload.ID != 0 && !teamExists(allTeams, payload.ID) {
			return fmt.Errorf("El equipo %d no existe.", payload.ID)
		}
		team, err := types.ApplyTeamEdit(payload, activeUserIDs, allTeams)
		if err != nil {
			return err
		}
		team.CompanyID, team.UpdatedBy = companyID, req.User.ID
		records := []types.CrmTeam{team}
		if team.ID == 0 {
			err = db.InsertIn(transaction, &records)
		} else {
			table := db.TableOf[types.CrmTeam]()
			err = db.UpdateIn(transaction, &records, table.Name, table.LeaderUserID, table.MemberUserIDs, table.Status, table.UpdatedBy)
		}
		savedTeam = records[0]
		return err
	})
	if saveError != nil {
		return req.MakeErr("Error al guardar el equipo:", saveError)
	}
	return req.MakeResponse(savedTeam)
}

func teamExists(teams []types.CrmTeam, teamID int16) bool {
	for _, team := range teams {
		if team.ID == teamID {
			return true
		}
	}
	return false
}

// loadActiveTeams returns the company's active teams.
func loadActiveTeams(companyID int32) ([]types.CrmTeam, error) {
	teams := []types.CrmTeam{}
	query := db.Query(&teams)
	query.CompanyID.Equals(companyID).Status.Equals(types.CrmStatusActive)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los equipos: %v", err)
	}
	return teams, nil
}
