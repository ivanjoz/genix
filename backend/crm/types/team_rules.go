package types

import (
	"fmt"
	"slices"
	"strings"
)

// ApplyTeamEdit validates a POST.crm-team payload and returns the record to save. otherTeams are
// the company's other active teams: a user may belong to only one, so team roll-ups never count a
// seller twice. Status 0 on an existing team deletes it (soft).
func ApplyTeamEdit(payload CrmTeam, activeUserIDs map[int32]bool, otherTeams []CrmTeam) (CrmTeam, error) {
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Status == CrmStatusDeleted && payload.ID != 0 {
		return payload, nil
	}
	payload.Status = CrmStatusActive
	if payload.Name == "" {
		return CrmTeam{}, fmt.Errorf("El nombre del equipo es obligatorio.")
	}
	if len(payload.MemberUserIDs) == 0 {
		return CrmTeam{}, fmt.Errorf("El equipo necesita al menos un miembro.")
	}
	if !slices.Contains(payload.MemberUserIDs, payload.LeaderUserID) {
		return CrmTeam{}, fmt.Errorf("El líder debe ser miembro del equipo.")
	}
	seenUserIDs := map[int32]bool{}
	for _, userID := range payload.MemberUserIDs {
		if !activeUserIDs[userID] {
			return CrmTeam{}, fmt.Errorf("El usuario %d no existe o está inactivo.", userID)
		}
		if seenUserIDs[userID] {
			return CrmTeam{}, fmt.Errorf("El usuario %d está repetido.", userID)
		}
		seenUserIDs[userID] = true
	}
	for _, other := range otherTeams {
		if other.ID == payload.ID || other.Status != CrmStatusActive {
			continue
		}
		if other.Name == payload.Name {
			return CrmTeam{}, fmt.Errorf("Ya existe un equipo llamado %q.", payload.Name)
		}
		for _, userID := range other.MemberUserIDs {
			if seenUserIDs[userID] {
				return CrmTeam{}, fmt.Errorf("El usuario %d ya pertenece al equipo %q.", userID, other.Name)
			}
		}
	}
	return payload, nil
}

// TeamOfUser returns the active team the user belongs to, or a zero team.
func TeamOfUser(teams []CrmTeam, userID int32) CrmTeam {
	for _, team := range teams {
		if team.Status == CrmStatusActive && slices.Contains(team.MemberUserIDs, userID) {
			return team
		}
	}
	return CrmTeam{}
}
